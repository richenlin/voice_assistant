package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"voice_assistant/pkg/protocol"
	"voice_assistant/voice_assistant_client/internal/audio"
	"voice_assistant/voice_assistant_client/internal/client"
	"voice_assistant/voice_assistant_client/internal/config"
	"voice_assistant/voice_assistant_client/internal/ui"
)

// 版本信息
const (
	Version = "v1.0.0"
	Name    = "语音助手客户端"
)

// 命令行参数
var (
	configFile  = flag.String("config", "config/client.yaml", "配置文件路径")
	showVersion = flag.Bool("version", false, "显示版本信息")
	showDevices = flag.Bool("devices", false, "显示音频设备列表")
	debugMode   = flag.Bool("debug", false, "启用调试模式")
	serverURL   = flag.String("server", "", "服务器URL (覆盖配置文件)")
	sessionMode = flag.String("mode", "", "会话模式 (continuous/single/wakeword)")
)

// VoiceAssistantClient 语音助手客户端
type VoiceAssistantClient struct {
	config      *config.Config
	wsClient    *client.WebSocketClient
	audioInput  *audio.AudioInput
	audioOutput *audio.AudioOutput
	uiManager   *ui.Manager

	// 状态管理
	isRunning   bool
	isRecording bool
	isPlaying   bool

	// 音频处理
	chunkID     int
	audioBuffer [][]byte
}

func main() {
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Printf("%s %s\n", Name, Version)
		os.Exit(0)
	}

	// 显示音频设备列表
	if *showDevices {
		showAudioDevices()
		os.Exit(0)
	}

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建客户端
	client, err := NewVoiceAssistantClient(cfg)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	// 启动客户端
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := client.Start(ctx); err != nil {
		log.Fatalf("启动客户端失败: %v", err)
	}

	// 等待信号
	waitForSignal(cancel)

	// 停止客户端
	if err := client.Stop(); err != nil {
		log.Printf("停止客户端失败: %v", err)
	}

	log.Println("客户端已退出")
}

// NewVoiceAssistantClient 创建语音助手客户端
func NewVoiceAssistantClient(cfg *config.Config) (*VoiceAssistantClient, error) {
	// 创建WebSocket客户端
	wsClient := client.NewWebSocketClient(cfg.ToClientConfig())

	// 创建音频输入
	audioInput, err := audio.NewAudioInput(cfg.ToAudioInputConfig())
	if err != nil {
		return nil, fmt.Errorf("创建音频输入失败: %w", err)
	}

	// 创建音频输出
	audioOutput, err := audio.NewAudioOutput(cfg.ToAudioOutputConfig())
	if err != nil {
		return nil, fmt.Errorf("创建音频输出失败: %w", err)
	}

	// 创建UI管理器
	uiManager := ui.NewManager(cfg.UI)

	client := &VoiceAssistantClient{
		config:      cfg,
		wsClient:    wsClient,
		audioInput:  audioInput,
		audioOutput: audioOutput,
		uiManager:   uiManager,
		audioBuffer: make([][]byte, 0),
	}

	// 注册消息处理器
	client.registerMessageHandlers()

	return client, nil
}

// Start 启动客户端
func (c *VoiceAssistantClient) Start(ctx context.Context) error {
	log.Printf("启动%s %s", Name, Version)

	// 启动UI
	if err := c.uiManager.Start(ctx); err != nil {
		return fmt.Errorf("启动UI失败: %w", err)
	}

	// 连接到服务器
	if err := c.wsClient.Connect(ctx); err != nil {
		return fmt.Errorf("连接服务器失败: %w", err)
	}

	// 启动音频输入
	if err := c.audioInput.Start(ctx); err != nil {
		return fmt.Errorf("启动音频输入失败: %w", err)
	}

	// 启动音频输出
	if err := c.audioOutput.Start(ctx); err != nil {
		return fmt.Errorf("启动音频输出失败: %w", err)
	}

	// 启动音频处理协程
	go c.audioProcessingLoop(ctx)

	// 启动会话
	mode := c.config.Session.Mode
	if *sessionMode != "" {
		mode = *sessionMode
	}

	if err := c.wsClient.StartSession(mode); err != nil {
		return fmt.Errorf("启动会话失败: %w", err)
	}

	c.isRunning = true
	log.Printf("客户端启动成功，会话模式: %s", mode)

	return nil
}

// Stop 停止客户端
func (c *VoiceAssistantClient) Stop() error {
	if !c.isRunning {
		return nil
	}

	log.Println("正在停止客户端...")

	c.isRunning = false

	// 停止会话
	if c.wsClient.IsConnected() {
		c.wsClient.StopSession()
	}

	// 停止音频输入
	if c.audioInput != nil {
		c.audioInput.Stop()
	}

	// 停止音频输出
	if c.audioOutput != nil {
		c.audioOutput.Stop()
	}

	// 断开WebSocket连接
	if c.wsClient != nil {
		c.wsClient.Disconnect()
	}

	// 停止UI
	if c.uiManager != nil {
		c.uiManager.Stop()
	}

	return nil
}

// registerMessageHandlers 注册消息处理器
func (c *VoiceAssistantClient) registerMessageHandlers() {
	// 响应消息处理器
	c.wsClient.RegisterHandler(protocol.Response, c.handleResponseMessage)

	// 状态消息处理器
	c.wsClient.RegisterHandler(protocol.Status, c.handleStatusMessage)

	// 错误消息处理器
	c.wsClient.RegisterHandler(protocol.Error, c.handleErrorMessage)
}

// handleResponseMessage 处理响应消息
func (c *VoiceAssistantClient) handleResponseMessage(msg *protocol.Message) error {
	respData, err := protocol.ParseResponseData(msg.Data)
	if err != nil {
		return fmt.Errorf("解析响应数据失败: %w", err)
	}

	switch respData.Stage {
	case protocol.StageASR:
		// ASR识别结果
		c.uiManager.ShowASRResult(respData.Content, respData.Confidence, respData.IsFinal)

	case protocol.StageLLM:
		// LLM回复结果
		c.uiManager.ShowLLMResponse(respData.Content, respData.IsFinal)

	case protocol.StageTTS:
		// TTS音频数据
		if len(respData.AudioData) > 0 {
			if err := c.audioOutput.PlayBytes(respData.AudioData); err != nil {
				log.Printf("播放音频失败: %v", err)
			}
		}
	}

	return nil
}

// handleStatusMessage 处理状态消息
func (c *VoiceAssistantClient) handleStatusMessage(msg *protocol.Message) error {
	statusData, err := protocol.ParseStatusData(msg.Data)
	if err != nil {
		return fmt.Errorf("解析状态数据失败: %w", err)
	}

	// 更新UI状态显示
	c.uiManager.UpdateStatus(statusData.State, statusData.Mode)

	// 根据状态调整录音状态
	switch statusData.State {
	case protocol.StateListening:
		if !c.isRecording {
			c.startRecording()
		}
	case protocol.StateProcessing, protocol.StateSpeaking:
		if c.isRecording {
			c.stopRecording()
		}
	}

	return nil
}

// handleErrorMessage 处理错误消息
func (c *VoiceAssistantClient) handleErrorMessage(msg *protocol.Message) error {
	errorData, err := protocol.ParseErrorData(msg.Data)
	if err != nil {
		return fmt.Errorf("解析错误数据失败: %w", err)
	}

	// 显示错误信息
	c.uiManager.ShowError(errorData.Code, errorData.Message)

	// 如果是不可恢复的错误，停止客户端
	if !errorData.Recoverable {
		log.Printf("收到不可恢复错误，停止客户端: %s", errorData.Message)
		go func() {
			time.Sleep(time.Second)
			c.Stop()
		}()
	}

	return nil
}

// audioProcessingLoop 音频处理循环
func (c *VoiceAssistantClient) audioProcessingLoop(ctx context.Context) {
	audioChan := c.audioInput.GetAudioChannel()

	for {
		select {
		case <-ctx.Done():
			return
		case audioData := <-audioChan:
			if !c.isRunning || !c.isRecording {
				continue
			}

			// 转换音频数据为字节
			audioBytes := audio.Float32ToBytes(audioData)

			// 发送音频流
			c.chunkID++
			if err := c.wsClient.SendAudioStream(audioBytes, c.chunkID, false); err != nil {
				log.Printf("发送音频流失败: %v", err)
			}

			// 更新UI音频级别显示
			if c.config.UI.ShowAudioLevel {
				stats := c.audioInput.GetStats()
				c.uiManager.UpdateAudioLevel(stats.AverageLevel, stats.PeakLevel)
			}
		}
	}
}

// startRecording 开始录音
func (c *VoiceAssistantClient) startRecording() {
	if c.isRecording {
		return
	}

	if err := c.audioInput.StartRecording(); err != nil {
		log.Printf("开始录音失败: %v", err)
		return
	}

	c.isRecording = true
	c.chunkID = 0
	c.uiManager.ShowMessage("🎤 开始录音...")
}

// stopRecording 停止录音
func (c *VoiceAssistantClient) stopRecording() {
	if !c.isRecording {
		return
	}

	if err := c.audioInput.StopRecording(); err != nil {
		log.Printf("停止录音失败: %v", err)
		return
	}

	// 发送最终音频块
	if err := c.wsClient.SendAudioStream([]byte{}, c.chunkID+1, true); err != nil {
		log.Printf("发送最终音频块失败: %v", err)
	}

	c.isRecording = false
	c.uiManager.ShowMessage("⏹️ 停止录音")
}

// loadConfig 加载配置
func loadConfig() (*config.Config, error) {
	var cfg *config.Config
	var err error

	if _, statErr := os.Stat(*configFile); os.IsNotExist(statErr) {
		log.Printf("配置文件不存在，使用默认配置: %s", *configFile)
		cfg = config.GetDefaultConfig()
	} else {
		cfg, err = config.LoadConfig(*configFile)
		if err != nil {
			return nil, err
		}
	}

	// 命令行参数覆盖
	if *serverURL != "" {
		// 解析服务器URL并更新配置
		// 这里简化处理，实际应该解析URL的各个部分
		cfg.Server.Host = *serverURL
	}

	if *debugMode {
		cfg.UI.LogLevel = "debug"
		cfg.Advanced.Debug.Enabled = true
	}

	return cfg, nil
}

// showAudioDevices 显示音频设备列表
func showAudioDevices() {
	fmt.Println("=== 音频输入设备 ===")
	if err := audio.PrintDeviceList(); err != nil {
		log.Printf("获取输入设备列表失败: %v", err)
	}

	fmt.Println("\n=== 音频输出设备 ===")
	if err := audio.PrintOutputDeviceList(); err != nil {
		log.Printf("获取输出设备列表失败: %v", err)
	}
}

// waitForSignal 等待信号
func waitForSignal(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("收到信号: %v", sig)
	cancel()
}
