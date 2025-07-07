package llm

import (
	"context"
)

// LLMService LLM服务接口
type LLMService interface {
	// Initialize 初始化LLM服务
	Initialize(config LLMConfig) error

	// GenerateResponse 生成回复（批量处理）
	GenerateResponse(ctx context.Context, messages []Message) (LLMResponse, error)

	// GenerateResponseStream 生成回复（流式处理）
	GenerateResponseStream(ctx context.Context, messages []Message) (<-chan LLMResponse, error)

	// Chat 聊天对话
	Chat(ctx context.Context, userInput string, conversationID string) (LLMResponse, error)

	// ChatStream 流式聊天对话
	ChatStream(ctx context.Context, userInput string, conversationID string) (<-chan LLMResponse, error)

	// GetSupportedModels 获取支持的模型列表
	GetSupportedModels() []string

	// SetModel 设置使用的模型
	SetModel(model string) error

	// GetModelInfo 获取模型信息
	GetModelInfo() ModelInfo

	// Close 关闭LLM服务
	Close() error
}

// LLMConfig LLM配置
type LLMConfig struct {
	Type      string `yaml:"type"`       // openai|ollama|websocket|anthropic|gemini
	Model     string `yaml:"model"`      // 模型名称
	APIKey    string `yaml:"api_key"`    // API密钥
	APIUrl    string `yaml:"api_url"`    // API地址
	Timeout   int    `yaml:"timeout"`    // 超时时间（秒）
	MaxTokens int    `yaml:"max_tokens"` // 最大token数

	// 通用参数
	Temperature       float32 `yaml:"temperature"`        // 温度参数
	TopP              float32 `yaml:"top_p"`              // Top-p参数
	TopK              int     `yaml:"top_k"`              // Top-k参数
	RepetitionPenalty float32 `yaml:"repetition_penalty"` // 重复惩罚
	PresencePenalty   float32 `yaml:"presence_penalty"`   // 存在惩罚
	FrequencyPenalty  float32 `yaml:"frequency_penalty"`  // 频率惩罚

	// 系统提示
	SystemPrompt string `yaml:"system_prompt"` // 系统提示

	// 上下文管理
	MaxContextLength  int  `yaml:"max_context_length"`  // 最大上下文长度
	ContextWindow     int  `yaml:"context_window"`      // 上下文窗口
	EnableContextTrim bool `yaml:"enable_context_trim"` // 启用上下文修剪
	KeepSystemPrompt  bool `yaml:"keep_system_prompt"`  // 保留系统提示

	// OpenAI特定配置
	OpenAIConfig OpenAIConfig `yaml:"openai"`

	// Ollama特定配置
	OllamaConfig OllamaConfig `yaml:"ollama"`

	// WebSocket特定配置
	WebSocketConfig WebSocketConfig `yaml:"websocket"`
}

// OpenAIConfig OpenAI配置
type OpenAIConfig struct {
	Organization string     `yaml:"organization"` // 组织ID
	BaseURL      string     `yaml:"base_url"`     // 基础URL
	Models       []string   `yaml:"models"`       // 可用模型列表
	Stream       bool       `yaml:"stream"`       // 流式响应
	Functions    []Function `yaml:"functions"`    // 函数定义
}

// OllamaConfig Ollama配置
type OllamaConfig struct {
	Host      string `yaml:"host"`       // 主机地址
	Port      int    `yaml:"port"`       // 端口
	KeepAlive string `yaml:"keep_alive"` // 保持连接时间
	NumCtx    int    `yaml:"num_ctx"`    // 上下文长度
	NumGPU    int    `yaml:"num_gpu"`    // GPU数量
	NumThread int    `yaml:"num_thread"` // 线程数
}

// WebSocketConfig WebSocket LLM配置
type WebSocketConfig struct {
	URL               string            `yaml:"url"`                // WebSocket地址
	Headers           map[string]string `yaml:"headers"`            // 请求头
	ReconnectInterval int               `yaml:"reconnect_interval"` // 重连间隔（秒）
	MaxReconnects     int               `yaml:"max_reconnects"`     // 最大重连次数
	PingInterval      int               `yaml:"ping_interval"`      // 心跳间隔（秒）
	WriteTimeout      int               `yaml:"write_timeout"`      // 写超时（秒）
	ReadTimeout       int               `yaml:"read_timeout"`       // 读超时（秒）
}

// Message 消息结构
type Message struct {
	Role         string        `json:"role"`                    // system|user|assistant|function
	Content      string        `json:"content"`                 // 消息内容
	Name         string        `json:"name,omitempty"`          // 消息名称
	FunctionCall *FunctionCall `json:"function_call,omitempty"` // 函数调用
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`    // 工具调用
	Timestamp    int64         `json:"timestamp"`               // 时间戳
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string `json:"name"`      // 函数名称
	Arguments string `json:"arguments"` // 函数参数（JSON字符串）
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string       `json:"id"`       // 调用ID
	Type     string       `json:"type"`     // 调用类型
	Function FunctionCall `json:"function"` // 函数信息
}

// Function 函数定义
type Function struct {
	Name        string                 `json:"name"`        // 函数名称
	Description string                 `json:"description"` // 函数描述
	Parameters  map[string]interface{} `json:"parameters"`  // 参数定义
}

// LLMResponse LLM响应
type LLMResponse struct {
	Content      string        `json:"content"`                 // 响应内容
	Role         string        `json:"role"`                    // 角色
	Model        string        `json:"model"`                   // 使用的模型
	FinishReason string        `json:"finish_reason"`           // 结束原因
	TokenUsage   TokenUsage    `json:"token_usage"`             // Token使用情况
	FunctionCall *FunctionCall `json:"function_call,omitempty"` // 函数调用
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`    // 工具调用

	// 流式响应相关
	IsDelta     bool   `json:"is_delta"`     // 是否为增量响应
	IsComplete  bool   `json:"is_complete"`  // 是否完成
	StreamID    string `json:"stream_id"`    // 流ID
	SequenceNum int    `json:"sequence_num"` // 序列号

	// 元数据
	ProcessTime    int64  `json:"process_time"`    // 处理耗时（毫秒）
	ConversationID string `json:"conversation_id"` // 对话ID
	MessageID      string `json:"message_id"`      // 消息ID
	Timestamp      int64  `json:"timestamp"`       // 时间戳
	Error          error  `json:"error"`           // 错误信息
}

// TokenUsage Token使用情况
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 提示Token数
	CompletionTokens int `json:"completion_tokens"` // 完成Token数
	TotalTokens      int `json:"total_tokens"`      // 总Token数
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name          string   `json:"name"`           // 模型名称
	Version       string   `json:"version"`        // 模型版本
	Type          string   `json:"type"`           // 模型类型
	Provider      string   `json:"provider"`       // 提供商
	MaxTokens     int      `json:"max_tokens"`     // 最大Token数
	ContextWindow int      `json:"context_window"` // 上下文窗口
	Languages     []string `json:"languages"`      // 支持的语言
	Capabilities  []string `json:"capabilities"`   // 能力列表
	ModelSize     int64    `json:"model_size"`     // 模型大小（字节）
	LoadTime      int64    `json:"load_time"`      // 加载时间（毫秒）
	MemoryUsage   int64    `json:"memory_usage"`   // 内存使用（字节）
}

// ConversationContext 对话上下文
type ConversationContext struct {
	ID           string                 `json:"id"`            // 对话ID
	Messages     []Message              `json:"messages"`      // 消息历史
	SystemPrompt string                 `json:"system_prompt"` // 系统提示
	CreatedAt    int64                  `json:"created_at"`    // 创建时间
	UpdatedAt    int64                  `json:"updated_at"`    // 更新时间
	TokenCount   int                    `json:"token_count"`   // Token计数
	MaxTokens    int                    `json:"max_tokens"`    // 最大Token数
	Metadata     map[string]interface{} `json:"metadata"`      // 元数据
}

// LLMFactory LLM工厂函数类型
type LLMFactory func(config LLMConfig) (LLMService, error)

// 注册的LLM实现
var llmFactories = make(map[string]LLMFactory)

// RegisterLLM 注册LLM实现
func RegisterLLM(name string, factory LLMFactory) {
	llmFactories[name] = factory
}

// CreateLLM 创建LLM服务
func CreateLLM(config LLMConfig) (LLMService, error) {
	factory, exists := llmFactories[config.Type]
	if !exists {
		return nil, ErrUnsupportedLLMType
	}
	return factory(config)
}

// GetAvailableLLMTypes 获取可用的LLM类型
func GetAvailableLLMTypes() []string {
	types := make([]string, 0, len(llmFactories))
	for t := range llmFactories {
		types = append(types, t)
	}
	return types
}
