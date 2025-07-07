package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"voice_assistant/pkg/protocol"
	"voice_assistant/voice_assistant_server/internal/asr"
	"voice_assistant/voice_assistant_server/internal/llm"
	"voice_assistant/voice_assistant_server/internal/tts"
)

// MessageProcessor 消息处理器
type MessageProcessor struct {
	// 服务实例
	asrService asr.ASRService
	llmService llm.LLMService
	ttsService tts.TTSService

	// 配置
	config ProcessorConfig

	// 会话管理
	sessions map[string]*Session
	mu       sync.RWMutex

	// 处理状态
	isInitialized bool
}

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	ASRConfig asr.ASRConfig `yaml:"asr"`
	LLMConfig llm.LLMConfig `yaml:"llm"`
	TTSConfig tts.TTSConfig `yaml:"tts"`

	// 处理选项
	EnableContinuousMode  bool `yaml:"enable_continuous_mode"`
	MaxConcurrentSessions int  `yaml:"max_concurrent_sessions"`
	SessionTimeout        int  `yaml:"session_timeout"` // 秒
	AudioBufferSize       int  `yaml:"audio_buffer_size"`
}

// Session 会话状态
type Session struct {
	ID             string
	State          SessionState
	ConversationID string
	AudioBuffer    []byte
	LastActivity   time.Time
	IsProcessing   bool
	ContinuousMode bool

	// 处理通道
	audioStreamChan chan []byte
	responseChan    chan *protocol.Message

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// SessionState 会话状态
type SessionState string

const (
	StateIdle       SessionState = "idle"
	StateListening  SessionState = "listening"
	StateProcessing SessionState = "processing"
	StateResponding SessionState = "responding"
	StateError      SessionState = "error"
)

// NewMessageProcessor 创建消息处理器
func NewMessageProcessor(config ProcessorConfig) *MessageProcessor {
	return &MessageProcessor{
		config:   config,
		sessions: make(map[string]*Session),
	}
}

// Initialize 初始化处理器
func (p *MessageProcessor) Initialize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Println("MessageProcessor: 初始化中...")

	// 初始化ASR服务
	asrService, err := asr.CreateASR(p.config.ASRConfig)
	if err != nil {
		return fmt.Errorf("创建ASR服务失败: %w", err)
	}
	if err := asrService.Initialize(p.config.ASRConfig); err != nil {
		return fmt.Errorf("初始化ASR服务失败: %w", err)
	}
	p.asrService = asrService

	// 初始化LLM服务
	llmService, err := llm.CreateLLM(p.config.LLMConfig)
	if err != nil {
		return fmt.Errorf("创建LLM服务失败: %w", err)
	}
	if err := llmService.Initialize(p.config.LLMConfig); err != nil {
		return fmt.Errorf("初始化LLM服务失败: %w", err)
	}
	p.llmService = llmService

	// 初始化TTS服务
	ttsService, err := tts.CreateTTS(p.config.TTSConfig)
	if err != nil {
		return fmt.Errorf("创建TTS服务失败: %w", err)
	}
	if err := ttsService.Initialize(p.config.TTSConfig); err != nil {
		return fmt.Errorf("初始化TTS服务失败: %w", err)
	}
	p.ttsService = ttsService

	p.isInitialized = true

	log.Println("MessageProcessor: 初始化成功")
	return nil
}

// ProcessMessage 处理消息
func (p *MessageProcessor) ProcessMessage(client *Client, msg *protocol.Message) error {
	if !p.isInitialized {
		return p.sendError(client, "PROCESSOR_NOT_INITIALIZED", "处理器未初始化", true)
	}

	// 获取或创建会话
	session := p.getOrCreateSession(msg.SessionID)

	switch msg.Type {
	case protocol.AudioStream:
		return p.handleAudioStream(client, session, msg)
	case protocol.Command:
		return p.handleCommand(client, session, msg)
	default:
		return p.sendError(client, "UNSUPPORTED_MESSAGE_TYPE", fmt.Sprintf("不支持的消息类型: %s", msg.Type), false)
	}
}

// handleAudioStream 处理音频流
func (p *MessageProcessor) handleAudioStream(client *Client, session *Session, msg *protocol.Message) error {
	var audioData protocol.AudioStreamData
	if err := p.parseMessageData(msg.Data, &audioData); err != nil {
		return p.sendError(client, "INVALID_AUDIO_DATA", "无效的音频数据", false)
	}

	session.mu.Lock()
	session.LastActivity = time.Now()

	// 添加音频数据到缓冲区
	session.AudioBuffer = append(session.AudioBuffer, audioData.AudioData...)

	// 如果是最终数据或缓冲区足够大，处理音频
	shouldProcess := audioData.IsFinal || len(session.AudioBuffer) >= p.config.AudioBufferSize
	session.mu.Unlock()

	if shouldProcess {
		go p.processAudioBuffer(client, session, audioData.IsFinal)
	}

	return nil
}

// handleCommand 处理命令
func (p *MessageProcessor) handleCommand(client *Client, session *Session, msg *protocol.Message) error {
	var cmdData protocol.CommandData
	if err := p.parseMessageData(msg.Data, &cmdData); err != nil {
		return p.sendError(client, "INVALID_COMMAND_DATA", "无效的命令数据", false)
	}

	switch cmdData.Command {
	case "start_session":
		return p.handleStartSession(client, session, cmdData)
	case "stop_session":
		return p.handleStopSession(client, session, cmdData)
	case "set_mode":
		return p.handleSetMode(client, session, cmdData)
	case "get_status":
		return p.handleGetStatus(client, session, cmdData)
	default:
		return p.sendError(client, "UNSUPPORTED_COMMAND", fmt.Sprintf("不支持的命令: %s", cmdData.Command), false)
	}
}

// processAudioBuffer 处理音频缓冲区
func (p *MessageProcessor) processAudioBuffer(client *Client, session *Session, isFinal bool) {
	session.mu.Lock()
	if session.IsProcessing {
		session.mu.Unlock()
		return
	}
	session.IsProcessing = true
	session.State = StateProcessing
	audioBuffer := make([]byte, len(session.AudioBuffer))
	copy(audioBuffer, session.AudioBuffer)
	if isFinal {
		session.AudioBuffer = session.AudioBuffer[:0] // 清空缓冲区
	}
	session.mu.Unlock()

	// 发送状态更新
	p.sendStatus(client, session)

	// ASR处理
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	asrResult, err := p.asrService.ProcessAudio(ctx, audioBuffer)
	if err != nil {
		log.Printf("ASR处理失败: %v", err)
		p.sendError(client, "ASR_FAILED", "语音识别失败", true)
		session.mu.Lock()
		session.IsProcessing = false
		session.State = StateError
		session.mu.Unlock()
		return
	}

	// 发送ASR结果
	p.sendResponse(client, "asr", asrResult.Text, asrResult.Confidence, asrResult.IsFinal, nil)

	if asrResult.Text == "" || !asrResult.IsFinal {
		session.mu.Lock()
		session.IsProcessing = false
		session.State = StateListening
		session.mu.Unlock()
		return
	}

	// LLM处理
	session.mu.Lock()
	session.State = StateProcessing
	conversationID := session.ConversationID
	session.mu.Unlock()

	llmResponse, err := p.llmService.Chat(ctx, asrResult.Text, conversationID)
	if err != nil {
		log.Printf("LLM处理失败: %v", err)
		p.sendError(client, "LLM_FAILED", "文本生成失败", true)
		session.mu.Lock()
		session.IsProcessing = false
		session.State = StateError
		session.mu.Unlock()
		return
	}

	// 发送LLM结果
	p.sendResponse(client, "llm", llmResponse.Content, 0.9, true, nil)

	// TTS处理
	session.mu.Lock()
	session.State = StateResponding
	session.mu.Unlock()

	ttsResult, err := p.ttsService.SynthesizeText(ctx, llmResponse.Content)
	if err != nil {
		log.Printf("TTS处理失败: %v", err)
		p.sendError(client, "TTS_FAILED", "语音合成失败", true)
		session.mu.Lock()
		session.IsProcessing = false
		session.State = StateError
		session.mu.Unlock()
		return
	}

	// 发送TTS结果
	p.sendResponse(client, "tts", "", 1.0, true, ttsResult.AudioData)

	// 重置会话状态
	session.mu.Lock()
	session.IsProcessing = false
	if session.ContinuousMode {
		session.State = StateListening
	} else {
		session.State = StateIdle
	}
	session.mu.Unlock()

	// 发送状态更新
	p.sendStatus(client, session)
}

// handleStartSession 处理开始会话
func (p *MessageProcessor) handleStartSession(client *Client, session *Session, cmdData protocol.CommandData) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	session.State = StateListening
	session.ContinuousMode = cmdData.Mode == "continuous"
	session.LastActivity = time.Now()

	// 创建新的对话ID
	session.ConversationID = fmt.Sprintf("conv_%s_%d", session.ID, time.Now().UnixNano())

	log.Printf("会话已启动: %s, 连续模式: %t", session.ID, session.ContinuousMode)

	return p.sendStatus(client, session)
}

// handleStopSession 处理停止会话
func (p *MessageProcessor) handleStopSession(client *Client, session *Session, cmdData protocol.CommandData) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	session.State = StateIdle
	session.ContinuousMode = false
	session.AudioBuffer = session.AudioBuffer[:0]

	log.Printf("会话已停止: %s", session.ID)

	return p.sendStatus(client, session)
}

// handleSetMode 处理设置模式
func (p *MessageProcessor) handleSetMode(client *Client, session *Session, cmdData protocol.CommandData) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	if mode, exists := cmdData.Parameters["mode"]; exists {
		if modeStr, ok := mode.(string); ok {
			session.ContinuousMode = modeStr == "continuous"
			log.Printf("会话模式已更新: %s, 连续模式: %t", session.ID, session.ContinuousMode)
		}
	}

	return p.sendStatus(client, session)
}

// handleGetStatus 处理获取状态
func (p *MessageProcessor) handleGetStatus(client *Client, session *Session, cmdData protocol.CommandData) error {
	return p.sendStatus(client, session)
}

// getOrCreateSession 获取或创建会话
func (p *MessageProcessor) getOrCreateSession(sessionID string) *Session {
	p.mu.Lock()
	defer p.mu.Unlock()

	if session, exists := p.sessions[sessionID]; exists {
		return session
	}

	// 检查会话数量限制
	if len(p.sessions) >= p.config.MaxConcurrentSessions {
		// 清理最旧的会话
		p.cleanupOldestSession()
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &Session{
		ID:              sessionID,
		State:           StateIdle,
		ConversationID:  fmt.Sprintf("conv_%s_%d", sessionID, time.Now().UnixNano()),
		AudioBuffer:     make([]byte, 0, p.config.AudioBufferSize),
		LastActivity:    time.Now(),
		IsProcessing:    false,
		ContinuousMode:  false,
		audioStreamChan: make(chan []byte, 100),
		responseChan:    make(chan *protocol.Message, 100),
		ctx:             ctx,
		cancel:          cancel,
	}

	p.sessions[sessionID] = session

	log.Printf("新会话已创建: %s", sessionID)
	return session
}

// cleanupOldestSession 清理最旧的会话
func (p *MessageProcessor) cleanupOldestSession() {
	var oldestID string
	var oldestTime time.Time

	for id, session := range p.sessions {
		if oldestID == "" || session.LastActivity.Before(oldestTime) {
			oldestID = id
			oldestTime = session.LastActivity
		}
	}

	if oldestID != "" {
		if session, exists := p.sessions[oldestID]; exists {
			session.cancel()
			delete(p.sessions, oldestID)
			log.Printf("已清理旧会话: %s", oldestID)
		}
	}
}

// sendResponse 发送响应
func (p *MessageProcessor) sendResponse(client *Client, stage, content string, confidence float64, isFinal bool, audioData []byte) error {
	responseData := &protocol.ResponseData{
		Stage:      stage,
		Content:    content,
		Confidence: confidence,
		IsFinal:    isFinal,
		AudioData:  audioData,
	}

	msg := protocol.NewMessage(protocol.Response, client.ID, responseData)
	return client.SendMessage(msg)
}

// sendStatus 发送状态
func (p *MessageProcessor) sendStatus(client *Client, session *Session) error {
	session.mu.RLock()
	statusData := &protocol.StatusData{
		State: string(session.State),
		Mode: func() string {
			if session.ContinuousMode {
				return "continuous"
			}
			return "single"
		}(),
		ConcurrentStreams: len(p.sessions),
	}
	session.mu.RUnlock()

	msg := protocol.NewMessage(protocol.Status, client.ID, statusData)
	return client.SendMessage(msg)
}

// sendError 发送错误
func (p *MessageProcessor) sendError(client *Client, code, message string, recoverable bool) error {
	errorData := &protocol.ErrorData{
		Code:        code,
		Message:     message,
		Recoverable: recoverable,
	}

	msg := protocol.NewMessage(protocol.Error, client.ID, errorData)
	return client.SendMessage(msg)
}

// parseMessageData 解析消息数据
func (p *MessageProcessor) parseMessageData(data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}

// Close 关闭处理器
func (p *MessageProcessor) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 关闭所有会话
	for _, session := range p.sessions {
		session.cancel()
	}
	p.sessions = make(map[string]*Session)

	// 关闭服务
	if p.asrService != nil {
		p.asrService.Close()
	}
	if p.llmService != nil {
		p.llmService.Close()
	}
	if p.ttsService != nil {
		p.ttsService.Close()
	}

	p.isInitialized = false

	log.Println("MessageProcessor: 已关闭")
	return nil
}
