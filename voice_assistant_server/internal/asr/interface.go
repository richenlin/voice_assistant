package asr

import (
	"context"
	"io"
)

// ASRService ASR服务接口
type ASRService interface {
	// Initialize 初始化ASR服务
	Initialize(config ASRConfig) error

	// ProcessAudio 处理音频数据（批量处理）
	ProcessAudio(ctx context.Context, audioData []byte) (ASRResult, error)

	// ProcessAudioStream 处理音频流（流式处理）
	ProcessAudioStream(ctx context.Context, audioStream io.Reader) (<-chan ASRResult, error)

	// ProcessAudioBytes 处理音频字节流（实时处理）
	ProcessAudioBytes(ctx context.Context, audioBytes []byte, isFinal bool) (ASRResult, error)

	// GetSupportedLanguages 获取支持的语言列表
	GetSupportedLanguages() []string

	// SetLanguage 设置识别语言
	SetLanguage(language string) error

	// Close 关闭ASR服务
	Close() error

	// GetModelInfo 获取模型信息
	GetModelInfo() ModelInfo
}

// ASRConfig ASR配置
type ASRConfig struct {
	Type       string `yaml:"type"`        // whisper|sherpa|funasr|openai
	ModelPath  string `yaml:"model_path"`  // 模型路径
	Language   string `yaml:"language"`    // 语言代码
	SampleRate int    `yaml:"sample_rate"` // 采样率
	Channels   int    `yaml:"channels"`    // 声道数
	APIKey     string `yaml:"api_key"`     // API密钥（在线服务）
	APIUrl     string `yaml:"api_url"`     // API地址
	Timeout    int    `yaml:"timeout"`     // 超时时间（秒）

	// Whisper特定配置
	WhisperConfig WhisperConfig `yaml:"whisper"`

	// Sherpa-ONNX特定配置
	SherpaConfig SherpaConfig `yaml:"sherpa"`

	// FunASR特定配置
	FunASRConfig FunASRConfig `yaml:"funasr"`
}

// WhisperConfig Whisper配置
type WhisperConfig struct {
	ModelSize   string  `yaml:"model_size"`   // tiny|base|small|medium|large
	Device      string  `yaml:"device"`       // cpu|cuda
	ComputeType string  `yaml:"compute_type"` // int8|int16|float16|float32
	BeamSize    int     `yaml:"beam_size"`    // 束搜索大小
	Temperature float32 `yaml:"temperature"`  // 温度参数
	Patience    float32 `yaml:"patience"`     // 耐心参数
	VADFilter   bool    `yaml:"vad_filter"`   // VAD过滤
}

// SherpaConfig Sherpa-ONNX配置
type SherpaConfig struct {
	EncoderPath    string  `yaml:"encoder_path"`     // 编码器模型路径
	DecoderPath    string  `yaml:"decoder_path"`     // 解码器模型路径
	JoinerPath     string  `yaml:"joiner_path"`      // 连接器模型路径
	TokensPath     string  `yaml:"tokens_path"`      // 词汇表路径
	NumThreads     int     `yaml:"num_threads"`      // 线程数
	Provider       string  `yaml:"provider"`         // cpu|cuda|coreml
	MaxActivePaths int     `yaml:"max_active_paths"` // 最大活跃路径
	HotWordsFile   string  `yaml:"hot_words_file"`   // 热词文件
	HotWordsScore  float32 `yaml:"hot_words_score"`  // 热词分数
}

// FunASRConfig FunASR配置
type FunASRConfig struct {
	ModelDir          string `yaml:"model_dir"`            // 模型目录
	ModelRevision     string `yaml:"model_revision"`       // 模型版本
	DeviceID          string `yaml:"device_id"`            // 设备ID
	QuantType         string `yaml:"quant_type"`           // 量化类型
	IntraOpNumThreads int    `yaml:"intra_op_num_threads"` // 线程数
	CacheSize         int    `yaml:"cache_size"`           // 缓存大小
}

// ASRResult ASR识别结果
type ASRResult struct {
	Text       string  `json:"text"`       // 识别文本
	Confidence float64 `json:"confidence"` // 置信度
	Language   string  `json:"language"`   // 语言
	IsFinal    bool    `json:"is_final"`   // 是否为最终结果
	StartTime  int64   `json:"start_time"` // 开始时间（毫秒）
	EndTime    int64   `json:"end_time"`   // 结束时间（毫秒）
	Words      []Word  `json:"words"`      // 词级别信息

	// 元数据
	ProcessTime int64  `json:"process_time"` // 处理耗时（毫秒）
	ModelInfo   string `json:"model_info"`   // 模型信息
	Error       error  `json:"error"`        // 错误信息
}

// Word 词级别信息
type Word struct {
	Text       string  `json:"text"`       // 词文本
	StartTime  int64   `json:"start_time"` // 开始时间（毫秒）
	EndTime    int64   `json:"end_time"`   // 结束时间（毫秒）
	Confidence float64 `json:"confidence"` // 置信度
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name        string   `json:"name"`         // 模型名称
	Version     string   `json:"version"`      // 模型版本
	Type        string   `json:"type"`         // 模型类型
	Languages   []string `json:"languages"`    // 支持的语言
	SampleRate  int      `json:"sample_rate"`  // 采样率
	Channels    int      `json:"channels"`     // 声道数
	ModelSize   int64    `json:"model_size"`   // 模型大小（字节）
	LoadTime    int64    `json:"load_time"`    // 加载时间（毫秒）
	MemoryUsage int64    `json:"memory_usage"` // 内存使用（字节）
}

// ASRFactory ASR工厂函数类型
type ASRFactory func(config ASRConfig) (ASRService, error)

// 注册的ASR实现
var asrFactories = make(map[string]ASRFactory)

// RegisterASR 注册ASR实现
func RegisterASR(name string, factory ASRFactory) {
	asrFactories[name] = factory
}

// CreateASR 创建ASR服务
func CreateASR(config ASRConfig) (ASRService, error) {
	factory, exists := asrFactories[config.Type]
	if !exists {
		return nil, ErrUnsupportedASRType
	}
	return factory(config)
}

// GetAvailableASRTypes 获取可用的ASR类型
func GetAvailableASRTypes() []string {
	types := make([]string, 0, len(asrFactories))
	for t := range asrFactories {
		types = append(types, t)
	}
	return types
}
