package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"voice_assistant/pkg/protocol"
	"voice_assistant/voice_assistant_server/internal/asr"
	"voice_assistant/voice_assistant_server/internal/config"
	"voice_assistant/voice_assistant_server/internal/llm"
	"voice_assistant/voice_assistant_server/internal/server"
	"voice_assistant/voice_assistant_server/internal/tts"

	"github.com/gin-gonic/gin"
)

func main() {
	// 解析命令行参数
	var configPath string
	flag.StringVar(&configPath, "config", "config/server.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	cfg, err := config.LoadConfig(configData)
	if err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	// 创建WebSocket配置
	wsConfig := server.WebSocketConfig{
		ReadBufferSize:  cfg.WebSocket.ReadBufferSize,
		WriteBufferSize: cfg.WebSocket.WriteBufferSize,
		MaxConnections:  cfg.WebSocket.MaxConnections,
		PingPeriod:      cfg.WebSocket.PingPeriod,
		PongWait:        cfg.WebSocket.PongWait,
		WriteWait:       cfg.WebSocket.WriteWait,
	}

	// 创建WebSocket服务器
	wsServer := server.NewWebSocketServer(wsConfig)

	// 转换配置类型
	asrConfig := asr.ASRConfig{
		Type:       cfg.ASR.Provider,
		ModelPath:  cfg.ASR.Whisper.ModelPath,
		Language:   cfg.ASR.Whisper.Language,
		SampleRate: 16000,
		Channels:   1,
		APIKey:     cfg.ASR.OpenAI.APIKey,
		Timeout:    30,
	}

	llmConfig := llm.LLMConfig{
		Type:        cfg.LLM.Provider,
		Model:       cfg.LLM.OpenAI.Model,
		APIKey:      cfg.LLM.OpenAI.APIKey,
		Temperature: float32(cfg.LLM.OpenAI.Temperature),
		MaxTokens:   cfg.LLM.OpenAI.MaxTokens,
		Timeout:     30,
		OpenAIConfig: llm.OpenAIConfig{
			BaseURL: "https://api.openai.com/v1",
			Stream:  true,
		},
		OllamaConfig: llm.OllamaConfig{
			Host: cfg.LLM.Ollama.BaseURL,
			Port: 11434,
		},
		WebSocketConfig: llm.WebSocketConfig{
			URL: cfg.LLM.WebSocket.URL,
		},
	}

	ttsConfig := tts.TTSConfig{
		Type:     cfg.TTS.Provider,
		Voice:    cfg.TTS.EdgeTTS.Voice,
		Language: "zh-CN",
		Speed:    1.0,
		Pitch:    1.0,
		Volume:   1.0,
		Format:   "wav",
		Timeout:  30,
		EdgeConfig: tts.EdgeConfig{
			UseWebSocket: true,
		},
	}

	// 创建处理器配置
	processorConfig := server.ProcessorConfig{
		ASRConfig:             asrConfig,
		LLMConfig:             llmConfig,
		TTSConfig:             ttsConfig,
		EnableContinuousMode:  true,
		MaxConcurrentSessions: 10,
		SessionTimeout:        300,
		AudioBufferSize:       4096,
	}

	// 创建消息处理器
	processor := server.NewMessageProcessor(processorConfig)
	if err := processor.Initialize(); err != nil {
		log.Fatalf("初始化消息处理器失败: %v", err)
	}

	// 设置处理器
	wsServer.SetProcessor(processor)

	// 注册消息处理器
	wsServer.RegisterHandler("audio_stream", func(client *server.Client, msg *protocol.Message) error {
		return processor.ProcessMessage(client, msg)
	})
	wsServer.RegisterHandler("command", func(client *server.Client, msg *protocol.Message) error {
		return processor.ProcessMessage(client, msg)
	})

	// 创建HTTP服务器
	router := gin.Default()

	// WebSocket端点
	router.GET("/ws", func(c *gin.Context) {
		wsServer.HandleConnection(c.Writer, c.Request)
	})

	// 健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"clients":   wsServer.GetClientCount(),
			"timestamp": fmt.Sprintf("%d", cfg.Server.Port),
		})
	})

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("服务器启动在 %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
