package protocol

import (
	"encoding/json"
	"time"
)

// MessageType 消息类型
type MessageType string

const (
	AudioStream MessageType = "audio_stream"
	Command     MessageType = "command"
	Response    MessageType = "response"
	Status      MessageType = "status"
	Error       MessageType = "error"
)

// Message 基础消息结构
type Message struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// AudioStreamData 音频流数据
type AudioStreamData struct {
	Format    string `json:"format"`     // pcm_16khz_16bit, mp3, wav
	ChunkID   int    `json:"chunk_id"`   // 音频块ID
	IsFinal   bool   `json:"is_final"`   // 是否为最后一块
	AudioData []byte `json:"audio_data"` // 音频数据（base64编码）
}

// CommandData 控制命令数据
type CommandData struct {
	Command    string                 `json:"command"`    // 命令类型
	Mode       string                 `json:"mode"`       // 模式
	Parameters map[string]interface{} `json:"parameters"` // 参数
}

// 命令类型常量
const (
	CmdStartSession = "start_session"
	CmdEndSession   = "end_session"
	CmdStopSession  = "stop_session"
	CmdPause        = "pause"
	CmdResume       = "resume"
	CmdSetMode      = "set_mode"
	CmdGetStatus    = "get_status"
	CmdInterrupt    = "interrupt"
	CmdClearContext = "clear_context"
	CmdSetParameter = "set_parameter"
)

// 模式常量
const (
	ModeContinuous = "continuous"
	ModeWakeword   = "wakeword"
	ModeInterrupt  = "interrupt"
	ModeSingle     = "single"
)

// ResponseData 服务端响应数据
type ResponseData struct {
	Stage      string                 `json:"stage"`                // 处理阶段: asr, llm, tts
	Content    string                 `json:"content"`              // 响应内容
	Confidence float64                `json:"confidence"`           // 置信度
	IsFinal    bool                   `json:"is_final"`             // 是否为最终结果
	AudioData  []byte                 `json:"audio_data,omitempty"` // 音频数据（TTS结果）
	Metadata   map[string]interface{} `json:"metadata,omitempty"`   // 元数据
}

// 处理阶段常量
const (
	StageASR = "asr"
	StageLLM = "llm"
	StageTTS = "tts"
)

// StatusData 状态数据
type StatusData struct {
	State             string       `json:"state"`                  // 当前状态
	Mode              string       `json:"mode"`                   // 当前模式
	ConcurrentStreams int          `json:"concurrent_streams"`     // 并发流数量
	SessionInfo       *SessionInfo `json:"session_info,omitempty"` // 会话信息
}

// 状态常量
const (
	StateIdle         = "idle"
	StateListening    = "listening"
	StateProcessing   = "processing"
	StateSpeaking     = "speaking"
	StateError        = "error"
	StateConnected    = "connected"
	StateDisconnected = "disconnected"
)

// SessionInfo 会话信息
type SessionInfo struct {
	ID           string    `json:"id"`
	StartTime    time.Time `json:"start_time"`
	LastActivity time.Time `json:"last_activity"`
	MessageCount int       `json:"message_count"`
	Duration     int64     `json:"duration"` // 秒
}

// ErrorData 错误数据
type ErrorData struct {
	Code        string                 `json:"code"`              // 错误代码
	Message     string                 `json:"message"`           // 错误消息
	Recoverable bool                   `json:"recoverable"`       // 是否可恢复
	Details     map[string]interface{} `json:"details,omitempty"` // 错误详情
}

// 错误代码常量
const (
	ErrProcessorNotInitialized = "PROCESSOR_NOT_INITIALIZED"
	ErrInvalidAudioData        = "INVALID_AUDIO_DATA"
	ErrInvalidCommandData      = "INVALID_COMMAND_DATA"
	ErrUnsupportedMessageType  = "UNSUPPORTED_MESSAGE_TYPE"
	ErrUnsupportedCommand      = "UNSUPPORTED_COMMAND"
	ErrASRFailed               = "ASR_FAILED"
	ErrLLMFailed               = "LLM_FAILED"
	ErrTTSFailed               = "TTS_FAILED"
	ErrSessionNotFound         = "SESSION_NOT_FOUND"
	ErrSessionLimitExceeded    = "SESSION_LIMIT_EXCEEDED"
	ErrConnectionFailed        = "CONNECTION_FAILED"
	ErrAuthenticationFailed    = "AUTHENTICATION_FAILED"
	ErrRateLimitExceeded       = "RATE_LIMIT_EXCEEDED"
	ErrInternalError           = "INTERNAL_ERROR"
)

// NewMessage 创建新消息
func NewMessage(msgType MessageType, sessionID string, data interface{}) *Message {
	return &Message{
		Type:      msgType,
		SessionID: sessionID,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Data:      data,
	}
}

// NewAudioStreamMessage 创建音频流消息
func NewAudioStreamMessage(sessionID string, format string, chunkID int, isFinal bool, audioData []byte) *Message {
	data := &AudioStreamData{
		Format:    format,
		ChunkID:   chunkID,
		IsFinal:   isFinal,
		AudioData: audioData,
	}
	return NewMessage(AudioStream, sessionID, data)
}

// NewCommandMessage 创建命令消息
func NewCommandMessage(sessionID string, command, mode string, parameters map[string]interface{}) *Message {
	data := &CommandData{
		Command:    command,
		Mode:       mode,
		Parameters: parameters,
	}
	return NewMessage(Command, sessionID, data)
}

// NewResponseMessage 创建响应消息
func NewResponseMessage(sessionID string, stage, content string, confidence float64, isFinal bool, audioData []byte) *Message {
	data := &ResponseData{
		Stage:      stage,
		Content:    content,
		Confidence: confidence,
		IsFinal:    isFinal,
		AudioData:  audioData,
	}
	return NewMessage(Response, sessionID, data)
}

// NewStatusMessage 创建状态消息
func NewStatusMessage(sessionID string, state, mode string, concurrentStreams int) *Message {
	data := &StatusData{
		State:             state,
		Mode:              mode,
		ConcurrentStreams: concurrentStreams,
	}
	return NewMessage(Status, sessionID, data)
}

// NewErrorMessage 创建错误消息
func NewErrorMessage(sessionID string, code, message string, recoverable bool) *Message {
	data := &ErrorData{
		Code:        code,
		Message:     message,
		Recoverable: recoverable,
	}
	return NewMessage(Error, sessionID, data)
}

// ToJSON 将消息转换为JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON 从JSON创建消息
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseAudioStreamData 解析音频流数据
func ParseAudioStreamData(data interface{}) (*AudioStreamData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var audioData AudioStreamData
	if err := json.Unmarshal(jsonData, &audioData); err != nil {
		return nil, err
	}

	return &audioData, nil
}

// ParseCommandData 解析命令数据
func ParseCommandData(data interface{}) (*CommandData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var cmdData CommandData
	if err := json.Unmarshal(jsonData, &cmdData); err != nil {
		return nil, err
	}

	return &cmdData, nil
}

// ParseResponseData 解析响应数据
func ParseResponseData(data interface{}) (*ResponseData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var respData ResponseData
	if err := json.Unmarshal(jsonData, &respData); err != nil {
		return nil, err
	}

	return &respData, nil
}

// ParseStatusData 解析状态数据
func ParseStatusData(data interface{}) (*StatusData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var statusData StatusData
	if err := json.Unmarshal(jsonData, &statusData); err != nil {
		return nil, err
	}

	return &statusData, nil
}

// ParseErrorData 解析错误数据
func ParseErrorData(data interface{}) (*ErrorData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var errorData ErrorData
	if err := json.Unmarshal(jsonData, &errorData); err != nil {
		return nil, err
	}

	return &errorData, nil
}

// IsRecoverable 检查错误是否可恢复
func (e *ErrorData) IsRecoverable() bool {
	return e.Recoverable
}

// GetErrorLevel 获取错误级别
func (e *ErrorData) GetErrorLevel() string {
	switch e.Code {
	case ErrProcessorNotInitialized, ErrInternalError:
		return "critical"
	case ErrASRFailed, ErrLLMFailed, ErrTTSFailed:
		return "error"
	case ErrInvalidAudioData, ErrInvalidCommandData, ErrUnsupportedCommand:
		return "warning"
	default:
		return "info"
	}
}

// ValidateMessage 验证消息格式
func ValidateMessage(msg *Message) error {
	if msg.Type == "" {
		return &ErrorData{
			Code:    ErrInvalidCommandData,
			Message: "消息类型不能为空",
		}
	}

	if msg.SessionID == "" {
		return &ErrorData{
			Code:    ErrInvalidCommandData,
			Message: "会话ID不能为空",
		}
	}

	if msg.Data == nil {
		return &ErrorData{
			Code:    ErrInvalidCommandData,
			Message: "消息数据不能为空",
		}
	}

	return nil
}

// Error 实现error接口
func (e *ErrorData) Error() string {
	return e.Message
}
