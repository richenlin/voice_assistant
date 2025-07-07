package tts

import (
	"context"
	"io"
)

// TTSService TTS服务接口
type TTSService interface {
	// Initialize 初始化TTS服务
	Initialize(config TTSConfig) error

	// SynthesizeText 合成文本（批量处理）
	SynthesizeText(ctx context.Context, text string) (TTSResult, error)

	// SynthesizeTextStream 合成文本（流式处理）
	SynthesizeTextStream(ctx context.Context, text string) (<-chan TTSResult, error)

	// SynthesizeToFile 合成到文件
	SynthesizeToFile(ctx context.Context, text string, filePath string) error

	// SynthesizeToStream 合成到流
	SynthesizeToStream(ctx context.Context, text string, stream io.Writer) error

	// GetSupportedVoices 获取支持的声音列表
	GetSupportedVoices() []Voice

	// SetVoice 设置声音
	SetVoice(voiceID string) error

	// GetSupportedLanguages 获取支持的语言列表
	GetSupportedLanguages() []string

	// SetLanguage 设置语言
	SetLanguage(language string) error

	// GetModelInfo 获取模型信息
	GetModelInfo() ModelInfo

	// Close 关闭TTS服务
	Close() error
}

// TTSConfig TTS配置
type TTSConfig struct {
	Type       string  `yaml:"type"`        // edge|sherpa|paddlespeech|azure|aws|google
	Voice      string  `yaml:"voice"`       // 声音ID
	Language   string  `yaml:"language"`    // 语言代码
	SampleRate int     `yaml:"sample_rate"` // 采样率
	Channels   int     `yaml:"channels"`    // 声道数
	Format     string  `yaml:"format"`      // 音频格式 wav|mp3|ogg|flac
	Quality    string  `yaml:"quality"`     // 音质 low|medium|high
	Speed      float32 `yaml:"speed"`       // 语速
	Pitch      float32 `yaml:"pitch"`       // 音调
	Volume     float32 `yaml:"volume"`      // 音量
	APIKey     string  `yaml:"api_key"`     // API密钥
	APIUrl     string  `yaml:"api_url"`     // API地址
	Timeout    int     `yaml:"timeout"`     // 超时时间（秒）

	// Edge-TTS特定配置
	EdgeConfig EdgeConfig `yaml:"edge"`

	// Sherpa-ONNX特定配置
	SherpaConfig SherpaConfig `yaml:"sherpa"`

	// PaddleSpeech特定配置
	PaddleConfig PaddleConfig `yaml:"paddle"`
}

// EdgeConfig Edge-TTS配置
type EdgeConfig struct {
	UseWebSocket       bool   `yaml:"use_websocket"`        // 使用WebSocket
	Proxy              string `yaml:"proxy"`                // 代理地址
	UserAgent          string `yaml:"user_agent"`           // User-Agent
	TrustedClientToken string `yaml:"trusted_client_token"` // 客户端令牌
}

// SherpaConfig Sherpa-ONNX TTS配置
type SherpaConfig struct {
	ModelPath   string `yaml:"model_path"`   // 模型路径
	LexiconPath string `yaml:"lexicon_path"` // 词典路径
	TokensPath  string `yaml:"tokens_path"`  // 词汇表路径
	DataDir     string `yaml:"data_dir"`     // 数据目录
	NumThreads  int    `yaml:"num_threads"`  // 线程数
	Provider    string `yaml:"provider"`     // cpu|cuda|coreml
	Debug       bool   `yaml:"debug"`        // 调试模式
}

// PaddleConfig PaddleSpeech配置
type PaddleConfig struct {
	ModelDir     string `yaml:"model_dir"`     // 模型目录
	Device       string `yaml:"device"`        // cpu|gpu
	CPUThreads   int    `yaml:"cpu_threads"`   // CPU线程数
	UseGPU       bool   `yaml:"use_gpu"`       // 使用GPU
	GPUMemory    int    `yaml:"gpu_memory"`    // GPU内存（MB）
	EnableMKLDNN bool   `yaml:"enable_mkldnn"` // 启用MKLDNN
}

// Voice 声音信息
type Voice struct {
	ID          string   `json:"id"`           // 声音ID
	Name        string   `json:"name"`         // 声音名称
	DisplayName string   `json:"display_name"` // 显示名称
	Language    string   `json:"language"`     // 语言代码
	Locale      string   `json:"locale"`       // 地区代码
	Gender      string   `json:"gender"`       // 性别 male|female|neutral
	Age         string   `json:"age"`          // 年龄 child|adult|senior
	Style       []string `json:"style"`        // 风格列表
	SampleRate  int      `json:"sample_rate"`  // 采样率
	Quality     string   `json:"quality"`      // 音质
	Provider    string   `json:"provider"`     // 提供商
	Description string   `json:"description"`  // 描述
	Preview     string   `json:"preview"`      // 预览音频URL
}

// TTSResult TTS合成结果
type TTSResult struct {
	AudioData  []byte `json:"audio_data"`  // 音频数据
	Format     string `json:"format"`      // 音频格式
	SampleRate int    `json:"sample_rate"` // 采样率
	Channels   int    `json:"channels"`    // 声道数
	Duration   int64  `json:"duration"`    // 时长（毫秒）
	Text       string `json:"text"`        // 原始文本
	Voice      string `json:"voice"`       // 使用的声音
	Language   string `json:"language"`    // 语言

	// 流式合成相关
	IsChunk    bool   `json:"is_chunk"`    // 是否为分块数据
	IsComplete bool   `json:"is_complete"` // 是否完成
	ChunkIndex int    `json:"chunk_index"` // 分块索引
	StreamID   string `json:"stream_id"`   // 流ID

	// 元数据
	ProcessTime int64  `json:"process_time"` // 处理耗时（毫秒）
	ModelInfo   string `json:"model_info"`   // 模型信息
	Timestamp   int64  `json:"timestamp"`    // 时间戳
	Error       error  `json:"error"`        // 错误信息
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name          string   `json:"name"`            // 模型名称
	Version       string   `json:"version"`         // 模型版本
	Type          string   `json:"type"`            // 模型类型
	Provider      string   `json:"provider"`        // 提供商
	Languages     []string `json:"languages"`       // 支持的语言
	Voices        []Voice  `json:"voices"`          // 支持的声音
	SampleRates   []int    `json:"sample_rates"`    // 支持的采样率
	Formats       []string `json:"formats"`         // 支持的格式
	MaxTextLength int      `json:"max_text_length"` // 最大文本长度
	ModelSize     int64    `json:"model_size"`      // 模型大小（字节）
	LoadTime      int64    `json:"load_time"`       // 加载时间（毫秒）
	MemoryUsage   int64    `json:"memory_usage"`    // 内存使用（字节）
}

// AudioFormat 音频格式定义
type AudioFormat struct {
	Name       string `json:"name"`        // 格式名称
	Extension  string `json:"extension"`   // 文件扩展名
	MimeType   string `json:"mime_type"`   // MIME类型
	SampleRate int    `json:"sample_rate"` // 采样率
	Channels   int    `json:"channels"`    // 声道数
	BitRate    int    `json:"bit_rate"`    // 比特率
	Quality    string `json:"quality"`     // 音质
}

// TTSFactory TTS工厂函数类型
type TTSFactory func(config TTSConfig) (TTSService, error)

// 注册的TTS实现
var ttsFactories = make(map[string]TTSFactory)

// RegisterTTS 注册TTS实现
func RegisterTTS(name string, factory TTSFactory) {
	ttsFactories[name] = factory
}

// CreateTTS 创建TTS服务
func CreateTTS(config TTSConfig) (TTSService, error) {
	factory, exists := ttsFactories[config.Type]
	if !exists {
		return nil, ErrUnsupportedTTSType
	}
	return factory(config)
}

// GetAvailableTTSTypes 获取可用的TTS类型
func GetAvailableTTSTypes() []string {
	types := make([]string, 0, len(ttsFactories))
	for t := range ttsFactories {
		types = append(types, t)
	}
	return types
}
