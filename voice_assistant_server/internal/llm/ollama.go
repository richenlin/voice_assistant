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

// OllamaLLM Ollama LLM实现
type OllamaLLM struct {
	config              LLMConfig
	baseURL             string
	client              *http.Client
	isInitialized       bool
	mu                  sync.RWMutex
	modelInfo           ModelInfo
	supportedModels     []string
	conversationManager *ConversationManager
}

// OllamaRequest Ollama API请求
type OllamaRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages,omitempty"`
	Prompt   string          `json:"prompt,omitempty"`
	Stream   bool            `json:"stream,omitempty"`
	Options  OllamaOptions   `json:"options,omitempty"`
}

// OllamaMessage Ollama消息格式
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaOptions Ollama选项
type OllamaOptions struct {
	Temperature   float32  `json:"temperature,omitempty"`
	TopP          float32  `json:"top_p,omitempty"`
	TopK          int      `json:"top_k,omitempty"`
	RepeatPenalty float32  `json:"repeat_penalty,omitempty"`
	NumCtx        int      `json:"num_ctx,omitempty"`
	NumGPU        int      `json:"num_gpu,omitempty"`
	NumThread     int      `json:"num_thread,omitempty"`
	Stop          []string `json:"stop,omitempty"`
}

// OllamaResponse Ollama API响应
type OllamaResponse struct {
	Model     string        `json:"model"`
	Message   OllamaMessage `json:"message"`
	Response  string        `json:"response"`
	Done      bool          `json:"done"`
	CreatedAt time.Time     `json:"created_at"`
	Context   []int         `json:"context,omitempty"`
}

// OllamaModelInfo Ollama模型信息
type OllamaModelInfo struct {
	Name       string            `json:"name"`
	ModifiedAt time.Time         `json:"modified_at"`
	Size       int64             `json:"size"`
	Digest     string            `json:"digest"`
	Details    OllamaModelDetail `json:"details"`
}

// OllamaModelDetail Ollama模型详情
type OllamaModelDetail struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// OllamaModelsResponse Ollama模型列表响应
type OllamaModelsResponse struct {
	Models []OllamaModelInfo `json:"models"`
}

// NewOllamaLLM 创建Ollama LLM实例
func NewOllamaLLM(config LLMConfig) (*OllamaLLM, error) {
	o := &OllamaLLM{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		conversationManager: NewConversationManager(100),
	}

	if o.client.Timeout == 0 {
		o.client.Timeout = 120 * time.Second // Ollama可能需要更长时间
	}

	return o, nil
}

// Initialize 初始化Ollama LLM
func (o *OllamaLLM) Initialize(config LLMConfig) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	log.Println("OllamaLLM: 初始化中...")

	// 设置基础URL
	o.baseURL = config.APIUrl
	if o.baseURL == "" {
		host := config.OllamaConfig.Host
		if host == "" {
			host = "localhost"
		}
		port := config.OllamaConfig.Port
		if port == 0 {
			port = 11434
		}
		o.baseURL = fmt.Sprintf("http://%s:%d", host, port)
	}

	// 检查连接
	if err := o.checkConnection(); err != nil {
		return fmt.Errorf("连接Ollama服务失败: %w", err)
	}

	// 获取可用模型
	models, err := o.fetchAvailableModels()
	if err != nil {
		return fmt.Errorf("获取模型列表失败: %w", err)
	}
	o.supportedModels = models

	// 设置默认模型
	if config.Model == "" {
		if len(o.supportedModels) > 0 {
			config.Model = o.supportedModels[0]
		} else {
			config.Model = "llama2"
		}
	}

	// 设置模型信息
	o.modelInfo = ModelInfo{
		Name:          config.Model,
		Version:       "1.0.0",
		Type:          "text-generation",
		Provider:      "Ollama",
		MaxTokens:     config.MaxTokens,
		ContextWindow: config.OllamaConfig.NumCtx,
		Languages:     []string{"zh", "en"},
		Capabilities:  []string{"chat", "completion"},
		LoadTime:      time.Now().UnixMilli(),
	}

	if o.modelInfo.ContextWindow == 0 {
		o.modelInfo.ContextWindow = 4096
	}

	o.config = config
	o.isInitialized = true

	log.Printf("OllamaLLM: 初始化成功，模型: %s", config.Model)
	return nil
}

// GenerateResponse 生成回复
func (o *OllamaLLM) GenerateResponse(ctx context.Context, messages []Message) (LLMResponse, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.isInitialized {
		return LLMResponse{}, ErrLLMNotInitialized
	}

	startTime := time.Now()

	// 转换消息格式
	ollamaMessages := o.convertMessages(messages)

	// 创建请求
	request := OllamaRequest{
		Model:    o.config.Model,
		Messages: ollamaMessages,
		Stream:   false,
		Options:  o.buildOptions(),
	}

	// 调用API
	response, err := o.callOllamaAPI(ctx, request)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("Ollama API调用失败: %w", err)
	}

	processTime := time.Since(startTime)

	result := LLMResponse{
		Content:      response.Message.Content,
		Role:         response.Message.Role,
		Model:        response.Model,
		FinishReason: "stop",
		IsComplete:   true,
		ProcessTime:  processTime.Milliseconds(),
		Timestamp:    time.Now().UnixMilli(),
	}

	return result, nil
}

// GenerateResponseStream 生成流式回复
func (o *OllamaLLM) GenerateResponseStream(ctx context.Context, messages []Message) (<-chan LLMResponse, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.isInitialized {
		return nil, ErrLLMNotInitialized
	}

	// 转换消息格式
	ollamaMessages := o.convertMessages(messages)

	// 创建流式请求
	request := OllamaRequest{
		Model:    o.config.Model,
		Messages: ollamaMessages,
		Stream:   true,
		Options:  o.buildOptions(),
	}

	// 创建响应通道
	responseChan := make(chan LLMResponse, 10)

	go func() {
		defer close(responseChan)

		if err := o.callOllamaStreamAPI(ctx, request, responseChan); err != nil {
			responseChan <- LLMResponse{
				Error: err,
			}
		}
	}()

	return responseChan, nil
}

// Chat 聊天对话
func (o *OllamaLLM) Chat(ctx context.Context, userInput string, conversationID string) (LLMResponse, error) {
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

	response.ConversationID = conversationID
	return response, nil
}

// ChatStream 流式聊天对话
func (o *OllamaLLM) ChatStream(ctx context.Context, userInput string, conversationID string) (<-chan LLMResponse, error) {
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
			}
		}
	}()

	return wrappedChan, nil
}

// GetSupportedModels 获取支持的模型列表
func (o *OllamaLLM) GetSupportedModels() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.supportedModels
}

// SetModel 设置使用的模型
func (o *OllamaLLM) SetModel(model string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 检查模型是否支持
	for _, m := range o.supportedModels {
		if m == model {
			o.config.Model = model
			o.modelInfo.Name = model
			return nil
		}
	}
	return ErrInvalidModel
}

// GetModelInfo 获取模型信息
func (o *OllamaLLM) GetModelInfo() ModelInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.modelInfo
}

// Close 关闭LLM服务
func (o *OllamaLLM) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.isInitialized = false
	log.Println("OllamaLLM: 已关闭")
	return nil
}

// checkConnection 检查连接
func (o *OllamaLLM) checkConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器响应状态码: %d", resp.StatusCode)
	}

	return nil
}

// fetchAvailableModels 获取可用模型
func (o *OllamaLLM) fetchAvailableModels() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取模型列表失败: %d", resp.StatusCode)
	}

	var modelsResp OllamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	models := make([]string, len(modelsResp.Models))
	for i, model := range modelsResp.Models {
		models[i] = model.Name
	}

	return models, nil
}

// convertMessages 转换消息格式
func (o *OllamaLLM) convertMessages(messages []Message) []OllamaMessage {
	ollamaMessages := make([]OllamaMessage, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = OllamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return ollamaMessages
}

// buildOptions 构建选项
func (o *OllamaLLM) buildOptions() OllamaOptions {
	options := OllamaOptions{
		Temperature:   o.config.Temperature,
		TopP:          o.config.TopP,
		TopK:          o.config.TopK,
		RepeatPenalty: o.config.RepetitionPenalty,
		NumCtx:        o.config.OllamaConfig.NumCtx,
		NumGPU:        o.config.OllamaConfig.NumGPU,
		NumThread:     o.config.OllamaConfig.NumThread,
	}

	if options.NumCtx == 0 {
		options.NumCtx = 4096
	}

	return options
}

// callOllamaAPI 调用Ollama API
func (o *OllamaLLM) callOllamaAPI(ctx context.Context, request OllamaRequest) (*OllamaResponse, error) {
	// 序列化请求
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

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
	var response OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// callOllamaStreamAPI 调用Ollama流式API
func (o *OllamaLLM) callOllamaStreamAPI(ctx context.Context, request OllamaRequest, responseChan chan<- LLMResponse) error {
	// 序列化请求
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

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

		if line == "" {
			continue
		}

		// 解析JSON数据
		var streamResponse OllamaResponse
		if err := json.Unmarshal([]byte(line), &streamResponse); err != nil {
			continue // 跳过无效的JSON
		}

		// 处理响应
		response := LLMResponse{
			Content:     streamResponse.Message.Content,
			Role:        streamResponse.Message.Role,
			Model:       streamResponse.Model,
			IsDelta:     !streamResponse.Done,
			IsComplete:  streamResponse.Done,
			SequenceNum: sequenceNum,
			Timestamp:   time.Now().UnixMilli(),
		}

		if streamResponse.Done {
			response.FinishReason = "stop"
		}

		responseChan <- response
		sequenceNum++

		if streamResponse.Done {
			break
		}
	}

	return scanner.Err()
}

// trimContext 修剪对话上下文
func (o *OllamaLLM) trimContext(conv *ConversationContext) {
	if len(conv.Messages) <= 2 {
		return
	}

	// 简单的修剪策略：保留系统提示和最近的消息
	maxMessages := 10
	if len(conv.Messages) > maxMessages {
		systemMessages := make([]Message, 0)
		recentMessages := make([]Message, 0)

		for i, msg := range conv.Messages {
			if msg.Role == "system" {
				systemMessages = append(systemMessages, msg)
			} else if i >= len(conv.Messages)-maxMessages+len(systemMessages) {
				recentMessages = append(recentMessages, msg)
			}
		}

		conv.Messages = append(systemMessages, recentMessages...)
	}
}

// 注册Ollama LLM
func init() {
	RegisterLLM("ollama", func(config LLMConfig) (LLMService, error) {
		return NewOllamaLLM(config)
	})
}
