package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OpenAILLM OpenAI LLM实现
type OpenAILLM struct {
	config              LLMConfig
	apiKey              string
	apiURL              string
	client              *http.Client
	isInitialized       bool
	mu                  sync.RWMutex
	modelInfo           ModelInfo
	supportedModels     []string
	conversationManager *ConversationManager
}

// OpenAIRequest OpenAI API请求
type OpenAIRequest struct {
	Model       string           `json:"model"`
	Messages    []OpenAIMessage  `json:"messages"`
	Temperature float32          `json:"temperature,omitempty"`
	TopP        float32          `json:"top_p,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
	Functions   []OpenAIFunction `json:"functions,omitempty"`
	ToolChoice  interface{}      `json:"tool_choice,omitempty"`
}

// OpenAIMessage OpenAI消息格式
type OpenAIMessage struct {
	Role         string              `json:"role"`
	Content      string              `json:"content"`
	Name         string              `json:"name,omitempty"`
	FunctionCall *OpenAIFunctionCall `json:"function_call,omitempty"`
	ToolCalls    []OpenAIToolCall    `json:"tool_calls,omitempty"`
}

// OpenAIFunction OpenAI函数定义
type OpenAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// OpenAIFunctionCall OpenAI函数调用
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAIToolCall OpenAI工具调用
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIResponse OpenAI API响应
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

// OpenAIChoice OpenAI选择
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	Delta        OpenAIMessage `json:"delta"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage OpenAI使用情况
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ConversationManager 对话管理器
type ConversationManager struct {
	conversations map[string]*ConversationContext
	mu            sync.RWMutex
	maxContexts   int
}

// NewConversationManager 创建对话管理器
func NewConversationManager(maxContexts int) *ConversationManager {
	return &ConversationManager{
		conversations: make(map[string]*ConversationContext),
		maxContexts:   maxContexts,
	}
}

// GetOrCreateConversation 获取或创建对话
func (cm *ConversationManager) GetOrCreateConversation(id string, systemPrompt string, maxTokens int) *ConversationContext {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conv, exists := cm.conversations[id]; exists {
		conv.UpdatedAt = time.Now().UnixMilli()
		return conv
	}

	// 如果超过最大数量，删除最旧的对话
	if len(cm.conversations) >= cm.maxContexts {
		var oldestID string
		var oldestTime int64 = time.Now().UnixMilli()

		for cid, conv := range cm.conversations {
			if conv.UpdatedAt < oldestTime {
				oldestTime = conv.UpdatedAt
				oldestID = cid
			}
		}

		if oldestID != "" {
			delete(cm.conversations, oldestID)
		}
	}

	conv := &ConversationContext{
		ID:           id,
		Messages:     make([]Message, 0),
		SystemPrompt: systemPrompt,
		CreatedAt:    time.Now().UnixMilli(),
		UpdatedAt:    time.Now().UnixMilli(),
		TokenCount:   0,
		MaxTokens:    maxTokens,
		Metadata:     make(map[string]interface{}),
	}

	// 添加系统提示
	if systemPrompt != "" {
		conv.Messages = append(conv.Messages, Message{
			Role:      "system",
			Content:   systemPrompt,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	cm.conversations[id] = conv
	return conv
}

// NewOpenAILLM 创建OpenAI LLM实例
func NewOpenAILLM(config LLMConfig) (*OpenAILLM, error) {
	o := &OpenAILLM{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		conversationManager: NewConversationManager(100), // 最多100个对话上下文
	}

	if o.client.Timeout == 0 {
		o.client.Timeout = 60 * time.Second
	}

	return o, nil
}

// Initialize 初始化OpenAI LLM
func (o *OpenAILLM) Initialize(config LLMConfig) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	log.Println("OpenAILLM: 初始化中...")

	// 检查API密钥
	if config.APIKey == "" {
		return fmt.Errorf("OpenAI API密钥不能为空")
	}

	o.apiKey = config.APIKey
	o.apiURL = config.APIUrl
	if o.apiURL == "" {
		o.apiURL = "https://api.openai.com/v1/chat/completions"
	}

	// 设置支持的模型
	o.supportedModels = []string{
		"gpt-4", "gpt-4-turbo", "gpt-4-turbo-preview",
		"gpt-3.5-turbo", "gpt-3.5-turbo-16k",
		"gpt-4o", "gpt-4o-mini",
	}

	// 设置默认模型
	if config.Model == "" {
		config.Model = "gpt-3.5-turbo"
	}

	// 设置模型信息
	o.modelInfo = ModelInfo{
		Name:          config.Model,
		Version:       "1.0.0",
		Type:          "text-generation",
		Provider:      "OpenAI",
		MaxTokens:     config.MaxTokens,
		ContextWindow: o.getContextWindow(config.Model),
		Languages:     []string{"zh", "en", "ja", "ko", "fr", "de", "es", "it", "pt", "ru"},
		Capabilities:  []string{"chat", "completion", "function-calling"},
		LoadTime:      time.Now().UnixMilli(),
	}

	o.config = config
	o.isInitialized = true

	log.Println("OpenAILLM: 初始化成功")
	return nil
}

// GenerateResponse 生成回复
func (o *OpenAILLM) GenerateResponse(ctx context.Context, messages []Message) (LLMResponse, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.isInitialized {
		return LLMResponse{}, ErrLLMNotInitialized
	}

	startTime := time.Now()

	// 转换消息格式
	openaiMessages := o.convertMessages(messages)

	// 创建请求
	request := OpenAIRequest{
		Model:       o.config.Model,
		Messages:    openaiMessages,
		Temperature: o.config.Temperature,
		TopP:        o.config.TopP,
		MaxTokens:   o.config.MaxTokens,
		Stream:      false,
	}

	// 调用API
	response, err := o.callOpenAIAPI(ctx, request)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("OpenAI API调用失败: %w", err)
	}

	processTime := time.Since(startTime)

	if len(response.Choices) == 0 {
		return LLMResponse{}, fmt.Errorf("OpenAI API返回空响应")
	}

	choice := response.Choices[0]
	result := LLMResponse{
		Content:      choice.Message.Content,
		Role:         choice.Message.Role,
		Model:        response.Model,
		FinishReason: choice.FinishReason,
		TokenUsage: TokenUsage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
		IsComplete:  true,
		ProcessTime: processTime.Milliseconds(),
		Timestamp:   time.Now().UnixMilli(),
	}

	// 处理函数调用
	if choice.Message.FunctionCall != nil {
		result.FunctionCall = &FunctionCall{
			Name:      choice.Message.FunctionCall.Name,
			Arguments: choice.Message.FunctionCall.Arguments,
		}
	}

	// 处理工具调用
	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			result.ToolCalls[i] = ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return result, nil
}

// GenerateResponseStream 生成流式回复
func (o *OpenAILLM) GenerateResponseStream(ctx context.Context, messages []Message) (<-chan LLMResponse, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.isInitialized {
		return nil, ErrLLMNotInitialized
	}

	// 转换消息格式
	openaiMessages := o.convertMessages(messages)

	// 创建流式请求
	request := OpenAIRequest{
		Model:       o.config.Model,
		Messages:    openaiMessages,
		Temperature: o.config.Temperature,
		TopP:        o.config.TopP,
		MaxTokens:   o.config.MaxTokens,
		Stream:      true,
	}

	// 创建响应通道
	responseChan := make(chan LLMResponse, 10)

	go func() {
		defer close(responseChan)

		if err := o.callOpenAIStreamAPI(ctx, request, responseChan); err != nil {
			responseChan <- LLMResponse{
				Error: err,
			}
		}
	}()

	return responseChan, nil
}

// Chat 聊天对话
func (o *OpenAILLM) Chat(ctx context.Context, userInput string, conversationID string) (LLMResponse, error) {
	// 获取或创建对话上下文
	conv := o.conversationManager.GetOrCreateConversation(
		conversationID,
		o.config.SystemPrompt,
		o.config.MaxContextLength,
	)

	// 添加用户消息
	userMessage := Message{
		Role:      "user",
		Content:   userInput,
		Timestamp: time.Now().UnixMilli(),
	}
	conv.Messages = append(conv.Messages, userMessage)

	// 修剪上下文（如果需要）
	if o.config.EnableContextTrim {
		o.trimContext(conv)
	}

	// 生成响应
	response, err := o.GenerateResponse(ctx, conv.Messages)
	if err != nil {
		return response, err
	}

	// 添加助手消息到对话历史
	assistantMessage := Message{
		Role:      "assistant",
		Content:   response.Content,
		Timestamp: time.Now().UnixMilli(),
	}
	conv.Messages = append(conv.Messages, assistantMessage)
	conv.UpdatedAt = time.Now().UnixMilli()
	conv.TokenCount += response.TokenUsage.TotalTokens

	response.ConversationID = conversationID
	return response, nil
}

// ChatStream 流式聊天对话
func (o *OpenAILLM) ChatStream(ctx context.Context, userInput string, conversationID string) (<-chan LLMResponse, error) {
	// 获取或创建对话上下文
	conv := o.conversationManager.GetOrCreateConversation(
		conversationID,
		o.config.SystemPrompt,
		o.config.MaxContextLength,
	)

	// 添加用户消息
	userMessage := Message{
		Role:      "user",
		Content:   userInput,
		Timestamp: time.Now().UnixMilli(),
	}
	conv.Messages = append(conv.Messages, userMessage)

	// 修剪上下文（如果需要）
	if o.config.EnableContextTrim {
		o.trimContext(conv)
	}

	// 生成流式响应
	responseChan, err := o.GenerateResponseStream(ctx, conv.Messages)
	if err != nil {
		return nil, err
	}

	// 包装响应通道以添加对话ID
	wrappedChan := make(chan LLMResponse, 10)
	go func() {
		defer close(wrappedChan)
		var fullContent strings.Builder

		for response := range responseChan {
			response.ConversationID = conversationID
			wrappedChan <- response

			if response.IsDelta {
				fullContent.WriteString(response.Content)
			}

			if response.IsComplete {
				// 添加完整的助手消息到对话历史
				assistantMessage := Message{
					Role:      "assistant",
					Content:   fullContent.String(),
					Timestamp: time.Now().UnixMilli(),
				}
				conv.Messages = append(conv.Messages, assistantMessage)
				conv.UpdatedAt = time.Now().UnixMilli()
				conv.TokenCount += response.TokenUsage.TotalTokens
			}
		}
	}()

	return wrappedChan, nil
}

// GetSupportedModels 获取支持的模型列表
func (o *OpenAILLM) GetSupportedModels() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.supportedModels
}

// SetModel 设置使用的模型
func (o *OpenAILLM) SetModel(model string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 检查模型是否支持
	for _, m := range o.supportedModels {
		if m == model {
			o.config.Model = model
			o.modelInfo.Name = model
			o.modelInfo.ContextWindow = o.getContextWindow(model)
			return nil
		}
	}
	return ErrInvalidModel
}

// GetModelInfo 获取模型信息
func (o *OpenAILLM) GetModelInfo() ModelInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.modelInfo
}

// Close 关闭LLM服务
func (o *OpenAILLM) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.isInitialized = false
	log.Println("OpenAILLM: 已关闭")
	return nil
}

// convertMessages 转换消息格式
func (o *OpenAILLM) convertMessages(messages []Message) []OpenAIMessage {
	openaiMessages := make([]OpenAIMessage, len(messages))
	for i, msg := range messages {
		openaiMsg := OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Name:    msg.Name,
		}

		if msg.FunctionCall != nil {
			openaiMsg.FunctionCall = &OpenAIFunctionCall{
				Name:      msg.FunctionCall.Name,
				Arguments: msg.FunctionCall.Arguments,
			}
		}

		if len(msg.ToolCalls) > 0 {
			openaiMsg.ToolCalls = make([]OpenAIToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				openaiMsg.ToolCalls[j] = OpenAIToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: OpenAIFunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		openaiMessages[i] = openaiMsg
	}
	return openaiMessages
}

// callOpenAIAPI 调用OpenAI API
func (o *OpenAILLM) callOpenAIAPI(ctx context.Context, request OpenAIRequest) (*OpenAIResponse, error) {
	// 序列化请求
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", o.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	if o.config.OpenAIConfig.Organization != "" {
		req.Header.Set("OpenAI-Organization", o.config.OpenAIConfig.Organization)
	}

	// 发送请求
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败: %d, %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析响应
	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// callOpenAIStreamAPI 调用OpenAI流式API
func (o *OpenAILLM) callOpenAIStreamAPI(ctx context.Context, request OpenAIRequest, responseChan chan<- LLMResponse) error {
	// 序列化请求
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", o.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	if o.config.OpenAIConfig.Organization != "" {
		req.Header.Set("OpenAI-Organization", o.config.OpenAIConfig.Organization)
	}

	// 发送请求
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API请求失败: %d, %s", resp.StatusCode, string(bodyBytes))
	}

	// 处理流式响应
	scanner := bufio.NewScanner(resp.Body)
	var sequenceNum int

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 处理data行
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// 检查结束标记
			if data == "[DONE]" {
				responseChan <- LLMResponse{
					IsComplete: true,
					Timestamp:  time.Now().UnixMilli(),
				}
				break
			}

			// 解析JSON数据
			var streamResponse OpenAIResponse
			if err := json.Unmarshal([]byte(data), &streamResponse); err != nil {
				continue // 跳过无效的JSON
			}

			// 处理响应
			if len(streamResponse.Choices) > 0 {
				choice := streamResponse.Choices[0]

				response := LLMResponse{
					Content:      choice.Delta.Content,
					Role:         choice.Delta.Role,
					Model:        streamResponse.Model,
					FinishReason: choice.FinishReason,
					IsDelta:      true,
					IsComplete:   choice.FinishReason != "",
					SequenceNum:  sequenceNum,
					Timestamp:    time.Now().UnixMilli(),
				}

				responseChan <- response
				sequenceNum++
			}
		}
	}

	return scanner.Err()
}

// getContextWindow 获取模型的上下文窗口大小
func (o *OpenAILLM) getContextWindow(model string) int {
	switch model {
	case "gpt-4":
		return 8192
	case "gpt-4-turbo", "gpt-4-turbo-preview":
		return 128000
	case "gpt-4o":
		return 128000
	case "gpt-4o-mini":
		return 128000
	case "gpt-3.5-turbo":
		return 4096
	case "gpt-3.5-turbo-16k":
		return 16384
	default:
		return 4096
	}
}

// trimContext 修剪对话上下文
func (o *OpenAILLM) trimContext(conv *ConversationContext) {
	if len(conv.Messages) <= 2 {
		return // 保留系统提示和至少一条消息
	}

	// 计算大概的token数（简单估算）
	totalTokens := 0
	for _, msg := range conv.Messages {
		totalTokens += len(msg.Content) / 4 // 粗略估算：4个字符≈1个token
	}

	// 如果超过最大token数，删除中间的消息
	if totalTokens > conv.MaxTokens {
		// 保留系统提示（如果有）和最近的几条消息
		systemMessages := make([]Message, 0)
		recentMessages := make([]Message, 0)

		for i, msg := range conv.Messages {
			if msg.Role == "system" {
				systemMessages = append(systemMessages, msg)
			} else if i >= len(conv.Messages)-4 { // 保留最近4条消息
				recentMessages = append(recentMessages, msg)
			}
		}

		conv.Messages = append(systemMessages, recentMessages...)
	}
}

// 注册OpenAI LLM
func init() {
	RegisterLLM("openai", func(config LLMConfig) (LLMService, error) {
		return NewOpenAILLM(config)
	})
}
