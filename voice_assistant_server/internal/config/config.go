package config

import (
	"time"

	"gopkg.in/yaml.v3"
)

// Config 服务器配置
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	WebSocket WebSocketConfig `yaml:"websocket"`
	ASR       ASRConfig       `yaml:"asr"`
	LLM       LLMConfig       `yaml:"llm"`
	TTS       TTSConfig       `yaml:"tts"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	ReadBufferSize  int           `yaml:"read_buffer_size"`
	WriteBufferSize int           `yaml:"write_buffer_size"`
	MaxConnections  int           `yaml:"max_connections"`
	PingPeriod      time.Duration `yaml:"ping_period"`
	PongWait        time.Duration `yaml:"pong_wait"`
	WriteWait       time.Duration `yaml:"write_wait"`
}

// ASRConfig ASR配置
type ASRConfig struct {
	Provider string          `yaml:"provider"` // whisper|openai|funasr
	Whisper  WhisperConfig   `yaml:"whisper"`
	OpenAI   OpenAIASRConfig `yaml:"openai"`
	FunASR   FunASRConfig    `yaml:"funasr"` // 新增FunASR配置
	Settings ASRSettings     `yaml:"settings"`
}

// WhisperConfig Whisper配置
type WhisperConfig struct {
	ModelPath string `yaml:"model_path"`
	Language  string `yaml:"language"`
}

// OpenAIASRConfig OpenAI ASR配置
type OpenAIASRConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

// FunASRConfig FunASR配置
type FunASRConfig struct {
	ModelDir          string `yaml:"model_dir"`            // 模型目录
	ModelRevision     string `yaml:"model_revision"`       // 模型版本
	DeviceID          string `yaml:"device_id"`            // 设备ID (cpu|cuda:0)
	IntraOpNumThreads int    `yaml:"intra_op_num_threads"` // 线程数
	BatchSize         int    `yaml:"batch_size"`           // 批处理大小
	MaxSentenceLength int    `yaml:"max_sentence_length"`  // 最大句子长度
}

// LLMConfig LLM配置
type LLMConfig struct {
	Provider  string                 `yaml:"provider"`
	OpenAI    OpenAILLMConfig        `yaml:"openai"`
	Ollama    OllamaConfig           `yaml:"ollama"`
	WebSocket WebSocketLLMConfig     `yaml:"websocket"`
	Settings  map[string]interface{} `yaml:"settings"`
}

// OpenAILLMConfig OpenAI LLM配置
type OpenAILLMConfig struct {
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

// OllamaConfig Ollama配置
type OllamaConfig struct {
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

// WebSocketLLMConfig WebSocket LLM配置
type WebSocketLLMConfig struct {
	URL string `yaml:"url"`
}

// TTSConfig TTS配置
type TTSConfig struct {
	Provider string        `yaml:"provider"` // edge_tts|sherpa|chattts
	EdgeTTS  EdgeTTSConfig `yaml:"edge_tts"`
	Sherpa   SherpaConfig  `yaml:"sherpa"`
	ChatTTS  ChatTTSConfig `yaml:"chattts"` // 新增ChatTTS配置
	Settings TTSSettings   `yaml:"settings"`
}

// EdgeTTSConfig Edge TTS配置
type EdgeTTSConfig struct {
	Voice string `yaml:"voice"`
	Rate  string `yaml:"rate"`
	Pitch string `yaml:"pitch"`
}

// SherpaConfig Sherpa配置
type SherpaConfig struct {
	ModelPath string `yaml:"model_path"`
}

// ChatTTSConfig ChatTTS配置
type ChatTTSConfig struct {
	ModelPath   string  `yaml:"model_path"`  // 模型路径（可选）
	Device      string  `yaml:"device"`      // cpu|cuda
	Temperature float32 `yaml:"temperature"` // 温度参数
	TopP        float32 `yaml:"top_p"`       // Top-p参数
	TopK        int     `yaml:"top_k"`       // Top-k参数
	SpeakerID   int     `yaml:"speaker_id"`  // 说话人ID
	NumThreads  int     `yaml:"num_threads"` // 线程数
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// ASRSettings ASR通用设置
type ASRSettings struct {
	SampleRate int `yaml:"sample_rate"`
	Channels   int `yaml:"channels"`
}

// TTSSettings TTS通用设置
type TTSSettings struct {
	SampleRate int    `yaml:"sample_rate"`
	Format     string `yaml:"format"`
	Quality    string `yaml:"quality"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		WebSocket: WebSocketConfig{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			MaxConnections:  100,
			PingPeriod:      54 * time.Second,
			PongWait:        60 * time.Second,
			WriteWait:       10 * time.Second,
		},
		ASR: ASRConfig{
			Provider: "whisper",
			Whisper: WhisperConfig{
				ModelPath: "./models/whisper/ggml-base.bin",
				Language:  "zh",
			},
			OpenAI: OpenAIASRConfig{
				APIKey: "",
				Model:  "whisper-1",
			},
		},
		LLM: LLMConfig{
			Provider: "openai",
			OpenAI: OpenAILLMConfig{
				APIKey:      "",
				Model:       "gpt-3.5-turbo",
				Temperature: 0.7,
				MaxTokens:   2000,
			},
			Ollama: OllamaConfig{
				BaseURL: "http://localhost:11434",
				Model:   "llama2",
			},
			WebSocket: WebSocketLLMConfig{
				URL: "ws://localhost:8081/llm",
			},
		},
		TTS: TTSConfig{
			Provider: "edge_tts",
			EdgeTTS: EdgeTTSConfig{
				Voice: "zh-CN-XiaoxiaoNeural",
				Rate:  "0%",
				Pitch: "0%",
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}

// LoadConfig 加载配置
func LoadConfig(data []byte) (*Config, error) {
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}
	return config, nil
}
