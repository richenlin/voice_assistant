package config

import (
	"fmt"
	"os"
	"time"

	"voice_assistant/voice_assistant_client/internal/audio"
	"voice_assistant/voice_assistant_client/internal/client"

	"gopkg.in/yaml.v3"
)

// Config 客户端完整配置
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Audio       AudioConfig       `yaml:"audio"`
	Session     SessionConfig     `yaml:"session"`
	UI          UIConfig          `yaml:"ui"`
	Logging     LoggingConfig     `yaml:"logging"`
	Performance PerformanceConfig `yaml:"performance"`
	Security    SecurityConfig    `yaml:"security"`
	Advanced    AdvancedConfig    `yaml:"advanced"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host                 string        `yaml:"host"`
	Port                 int           `yaml:"port"`
	UseTLS               bool          `yaml:"use_tls"`
	WebSocketPath        string        `yaml:"websocket_path"`
	ReconnectInterval    time.Duration `yaml:"reconnect_interval"`
	MaxReconnectAttempts int           `yaml:"max_reconnect_attempts"`
	ConnectionTimeout    time.Duration `yaml:"connection_timeout"`
	PingInterval         time.Duration `yaml:"ping_interval"`
	PongTimeout          time.Duration `yaml:"pong_timeout"`
}

// AudioConfig 音频配置
type AudioConfig struct {
	Input      AudioInputConfig  `yaml:"input"`
	Output     AudioOutputConfig `yaml:"output"`
	VAD        VADConfig         `yaml:"vad"`
	Processing ProcessingConfig  `yaml:"processing"`
}

// AudioInputConfig 音频输入配置
type AudioInputConfig struct {
	DeviceID      int    `yaml:"device_id"`
	SampleRate    int    `yaml:"sample_rate"`
	Channels      int    `yaml:"channels"`
	Format        string `yaml:"format"`
	BufferSize    int    `yaml:"buffer_size"`
	ChunkDuration int    `yaml:"chunk_duration"`
}

// AudioOutputConfig 音频输出配置
type AudioOutputConfig struct {
	DeviceID   int    `yaml:"device_id"`
	SampleRate int    `yaml:"sample_rate"`
	Channels   int    `yaml:"channels"`
	Format     string `yaml:"format"`
	BufferSize int    `yaml:"buffer_size"`
}

// VADConfig VAD配置
type VADConfig struct {
	Enabled            bool    `yaml:"enabled"`
	Threshold          float64 `yaml:"threshold"`
	MinSpeechDuration  int     `yaml:"min_speech_duration"`
	MinSilenceDuration int     `yaml:"min_silence_duration"`
	PreEmphasis        float64 `yaml:"pre_emphasis"`
}

// ProcessingConfig 音频处理配置
type ProcessingConfig struct {
	NoiseReduction      bool `yaml:"noise_reduction"`
	AutoGainControl     bool `yaml:"auto_gain_control"`
	EchoCancellation    bool `yaml:"echo_cancellation"`
	VolumeNormalization bool `yaml:"volume_normalization"`
}

// SessionConfig 会话配置
type SessionConfig struct {
	Mode              string         `yaml:"mode"`
	Timeout           time.Duration  `yaml:"timeout"`
	AutoReconnect     bool           `yaml:"auto_reconnect"`
	KeepAliveInterval time.Duration  `yaml:"keep_alive_interval"`
	MaxMessageSize    int            `yaml:"max_message_size"`
	Wakeword          WakewordConfig `yaml:"wakeword"`
}

// WakewordConfig 唤醒词配置
type WakewordConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Keywords    []string `yaml:"keywords"`
	Sensitivity float64  `yaml:"sensitivity"`
}

// UIConfig 用户界面配置
type UIConfig struct {
	Type                 string        `yaml:"type"`
	LogLevel             string        `yaml:"log_level"`
	ShowAudioLevel       bool          `yaml:"show_audio_level"`
	ShowConnectionStatus bool          `yaml:"show_connection_status"`
	Console              ConsoleConfig `yaml:"console"`
	GUI                  GUIConfig     `yaml:"gui"`
}

// ConsoleConfig 控制台配置
type ConsoleConfig struct {
	ColoredOutput  bool   `yaml:"colored_output"`
	ShowTimestamps bool   `yaml:"show_timestamps"`
	Prompt         string `yaml:"prompt"`
}

// GUIConfig GUI配置
type GUIConfig struct {
	WindowTitle  string `yaml:"window_title"`
	WindowWidth  int    `yaml:"window_width"`
	WindowHeight int    `yaml:"window_height"`
	Theme        string `yaml:"theme"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	AudioBufferSize      int `yaml:"audio_buffer_size"`
	MessageBufferSize    int `yaml:"message_buffer_size"`
	MaxConcurrentStreams int `yaml:"max_concurrent_streams"`
	WorkerThreads        int `yaml:"worker_threads"`
	MaxMemoryUsage       int `yaml:"max_memory_usage"`
	GCPercent            int `yaml:"gc_percent"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	TLS  TLSConfig  `yaml:"tls"`
	Auth AuthConfig `yaml:"auth"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled            bool   `yaml:"enabled"`
	CertFile           string `yaml:"cert_file"`
	KeyFile            string `yaml:"key_file"`
	CAFile             string `yaml:"ca_file"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Type     string `yaml:"type"`
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// AdvancedConfig 高级配置
type AdvancedConfig struct {
	Debug         DebugConfig         `yaml:"debug"`
	Experimental  ExperimentalConfig  `yaml:"experimental"`
	Compatibility CompatibilityConfig `yaml:"compatibility"`
}

// DebugConfig 调试配置
type DebugConfig struct {
	Enabled       bool `yaml:"enabled"`
	DumpAudio     bool `yaml:"dump_audio"`
	DumpMessages  bool `yaml:"dump_messages"`
	ProfileCPU    bool `yaml:"profile_cpu"`
	ProfileMemory bool `yaml:"profile_memory"`
}

// ExperimentalConfig 实验性配置
type ExperimentalConfig struct {
	UseBinaryProtocol bool `yaml:"use_binary_protocol"`
	EnableCompression bool `yaml:"enable_compression"`
	AdaptiveBitrate   bool `yaml:"adaptive_bitrate"`
}

// CompatibilityConfig 兼容性配置
type CompatibilityConfig struct {
	LegacyProtocol bool   `yaml:"legacy_protocol"`
	FallbackFormat string `yaml:"fallback_format"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	// 如果没有指定配置文件路径，使用默认路径
	if configPath == "" {
		configPath = "config/client.yaml"
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 替换环境变量
	configData := os.ExpandEnv(string(data))

	// 解析YAML
	var config Config
	if err := yaml.Unmarshal([]byte(configData), &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 设置默认值
	setDefaults(&config)

	return &config, nil
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证服务器配置
	if config.Server.Host == "" {
		return fmt.Errorf("服务器主机不能为空")
	}
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("服务器端口无效: %d", config.Server.Port)
	}

	// 验证音频配置
	if config.Audio.Input.SampleRate <= 0 {
		return fmt.Errorf("输入采样率无效: %d", config.Audio.Input.SampleRate)
	}
	if config.Audio.Output.SampleRate <= 0 {
		return fmt.Errorf("输出采样率无效: %d", config.Audio.Output.SampleRate)
	}

	// 验证UI配置
	validUITypes := map[string]bool{"console": true, "gui": true, "headless": true}
	if !validUITypes[config.UI.Type] {
		return fmt.Errorf("无效的UI类型: %s", config.UI.Type)
	}

	return nil
}

// setDefaults 设置默认值
func setDefaults(config *Config) {
	// 服务器默认值
	if config.Server.WebSocketPath == "" {
		config.Server.WebSocketPath = "/ws"
	}
	if config.Server.ReconnectInterval == 0 {
		config.Server.ReconnectInterval = 5 * time.Second
	}
	if config.Server.MaxReconnectAttempts == 0 {
		config.Server.MaxReconnectAttempts = 10
	}
	if config.Server.ConnectionTimeout == 0 {
		config.Server.ConnectionTimeout = 10 * time.Second
	}
	if config.Server.PingInterval == 0 {
		config.Server.PingInterval = 30 * time.Second
	}
	if config.Server.PongTimeout == 0 {
		config.Server.PongTimeout = 10 * time.Second
	}

	// 音频默认值
	if config.Audio.Input.SampleRate == 0 {
		config.Audio.Input.SampleRate = 16000
	}
	if config.Audio.Input.Channels == 0 {
		config.Audio.Input.Channels = 1
	}
	if config.Audio.Input.BufferSize == 0 {
		config.Audio.Input.BufferSize = 1024
	}
	if config.Audio.Output.SampleRate == 0 {
		config.Audio.Output.SampleRate = 16000
	}
	if config.Audio.Output.Channels == 0 {
		config.Audio.Output.Channels = 1
	}
	if config.Audio.Output.BufferSize == 0 {
		config.Audio.Output.BufferSize = 1024
	}

	// VAD默认值
	if config.Audio.VAD.Threshold == 0 {
		config.Audio.VAD.Threshold = 0.5
	}
	if config.Audio.VAD.MinSpeechDuration == 0 {
		config.Audio.VAD.MinSpeechDuration = 300
	}
	if config.Audio.VAD.MinSilenceDuration == 0 {
		config.Audio.VAD.MinSilenceDuration = 500
	}

	// 会话默认值
	if config.Session.Mode == "" {
		config.Session.Mode = "continuous"
	}
	if config.Session.Timeout == 0 {
		config.Session.Timeout = 30 * time.Minute
	}

	// UI默认值
	if config.UI.Type == "" {
		config.UI.Type = "console"
	}
	if config.UI.LogLevel == "" {
		config.UI.LogLevel = "info"
	}
	if config.UI.Console.Prompt == "" {
		config.UI.Console.Prompt = "语音助手> "
	}

	// 性能默认值
	if config.Performance.AudioBufferSize == 0 {
		config.Performance.AudioBufferSize = 8192
	}
	if config.Performance.MessageBufferSize == 0 {
		config.Performance.MessageBufferSize = 100
	}
	if config.Performance.WorkerThreads == 0 {
		config.Performance.WorkerThreads = 2
	}
}

// GetServerURL 获取服务器URL
func (c *Config) GetServerURL() string {
	scheme := "ws"
	if c.Server.UseTLS {
		scheme = "wss"
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, c.Server.Host, c.Server.Port, c.Server.WebSocketPath)
}

// ToClientConfig 转换为客户端配置
func (c *Config) ToClientConfig() client.ClientConfig {
	return client.ClientConfig{
		ServerURL:            c.GetServerURL(),
		SessionID:            "", // 将由客户端生成
		ReconnectInterval:    c.Server.ReconnectInterval,
		MaxReconnectAttempts: c.Server.MaxReconnectAttempts,
		ConnectionTimeout:    c.Server.ConnectionTimeout,
		PingInterval:         c.Server.PingInterval,
		PongTimeout:          c.Server.PongTimeout,
	}
}

// ToAudioInputConfig 转换为音频输入配置
func (c *Config) ToAudioInputConfig() audio.InputConfig {
	return audio.InputConfig{
		DeviceID:           c.Audio.Input.DeviceID,
		SampleRate:         c.Audio.Input.SampleRate,
		Channels:           c.Audio.Input.Channels,
		Format:             c.Audio.Input.Format,
		BufferSize:         c.Audio.Input.BufferSize,
		ChunkDuration:      c.Audio.Input.ChunkDuration,
		VADEnabled:         c.Audio.VAD.Enabled,
		VADThreshold:       c.Audio.VAD.Threshold,
		MinSpeechDuration:  c.Audio.VAD.MinSpeechDuration,
		MinSilenceDuration: c.Audio.VAD.MinSilenceDuration,
	}
}

// ToAudioOutputConfig 转换为音频输出配置
func (c *Config) ToAudioOutputConfig() audio.OutputConfig {
	return audio.OutputConfig{
		DeviceID:   c.Audio.Output.DeviceID,
		SampleRate: c.Audio.Output.SampleRate,
		Channels:   c.Audio.Output.Channels,
		Format:     c.Audio.Output.Format,
		BufferSize: c.Audio.Output.BufferSize,
	}
}

// SaveConfig 保存配置文件
func SaveConfig(config *Config, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *Config {
	config := &Config{
		Server: ServerConfig{
			Host:                 "localhost",
			Port:                 8080,
			UseTLS:               false,
			WebSocketPath:        "/ws",
			ReconnectInterval:    5 * time.Second,
			MaxReconnectAttempts: 10,
			ConnectionTimeout:    10 * time.Second,
			PingInterval:         30 * time.Second,
			PongTimeout:          10 * time.Second,
		},
		Audio: AudioConfig{
			Input: AudioInputConfig{
				DeviceID:      -1,
				SampleRate:    16000,
				Channels:      1,
				Format:        "pcm_16bit",
				BufferSize:    1024,
				ChunkDuration: 100,
			},
			Output: AudioOutputConfig{
				DeviceID:   -1,
				SampleRate: 16000,
				Channels:   1,
				Format:     "pcm_16bit",
				BufferSize: 1024,
			},
			VAD: VADConfig{
				Enabled:            true,
				Threshold:          0.5,
				MinSpeechDuration:  300,
				MinSilenceDuration: 500,
				PreEmphasis:        0.97,
			},
			Processing: ProcessingConfig{
				NoiseReduction:      true,
				AutoGainControl:     true,
				EchoCancellation:    false,
				VolumeNormalization: true,
			},
		},
		Session: SessionConfig{
			Mode:              "continuous",
			Timeout:           30 * time.Minute,
			AutoReconnect:     true,
			KeepAliveInterval: 30 * time.Second,
			MaxMessageSize:    1048576,
		},
		UI: UIConfig{
			Type:                 "console",
			LogLevel:             "info",
			ShowAudioLevel:       true,
			ShowConnectionStatus: true,
			Console: ConsoleConfig{
				ColoredOutput:  true,
				ShowTimestamps: true,
				Prompt:         "语音助手> ",
			},
		},
		Performance: PerformanceConfig{
			AudioBufferSize:      8192,
			MessageBufferSize:    100,
			MaxConcurrentStreams: 1,
			WorkerThreads:        2,
			MaxMemoryUsage:       128,
			GCPercent:            100,
		},
	}

	return config
}
