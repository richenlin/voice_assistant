package llm

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketLLM WebSocket LLM实现
type WebSocketLLM struct {
	config              LLMConfig
	conn                *websocket.Conn
	url                 string
	headers             map[string]string
	isInitialized       bool
	isConnected         bool
	mu                  sync.RWMutex
	modelInfo           ModelInfo
	conversationManager *ConversationManager
	reconnectTicker     *time.Ticker
	pingTicker          *time.Ticker
	stopChan            chan struct{}
	responseChan        chan WebSocketResponse
	requestID           int64
	pendingRequests     map[int64]chan LLMResponse
}

// WebSocketRequest WebSocket请求
type WebSocketRequest struct {
	ID             int64                  `json:"id"`
	Type           string                 `json:"type"`
	Model          string                 `json:"model,omitempty"`
	Messages       []Message              `json:"messages,omitempty"`
	Prompt         string                 `json:"prompt,omitempty"`
	Stream         bool                   `json:"stream,omitempty"`
	Temperature    float32                `json:"temperature,omitempty"`
	TopP           float32                `json:"top_p,omitempty"`
	MaxTokens      int                    `json:"max_tokens,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// WebSocketResponse WebSocket响应
type WebSocketResponse struct {
	ID           int64                  `json:"id"`
	Type         string                 `json:"type"`
	Content      string                 `json:"content,omitempty"`
	Role         string                 `json:"role,omitempty"`
	Model        string                 `json:"model,omitempty"`
	FinishReason string                 `json:"finish_reason,omitempty"`
	IsDelta      bool                   `json:"is_delta,omitempty"`
	IsComplete   bool                   `json:"is_complete,omitempty"`
	Error        string                 `json:"error,omitempty"`
	TokenUsage   TokenUsage             `json:"token_usage,omitempty"`
	Timestamp    int64                  `json:"timestamp,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NewWebSocketLLM 创建WebSocket LLM实例
func NewWebSocketLLM(config LLMConfig) (*WebSocketLLM, error) {
	w := &WebSocketLLM{
		config:              config,
		headers:             config.WebSocketConfig.Headers,
		stopChan:            make(chan struct{}),
		responseChan:        make(chan WebSocketResponse, 100),
		pendingRequests:     make(map[int64]chan LLMResponse),
		conversationManager: NewConversationManager(100),
	}

	return w, nil
}

// Initialize 初始化WebSocket LLM
func (w *WebSocketLLM) Initialize(config LLMConfig) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	log.Println("WebSocketLLM: 初始化中...")

	// 检查URL
	if config.WebSocketConfig.URL == "" {
		return fmt.Errorf("WebSocket URL不能为空")
	}

	w.url = config.WebSocketConfig.URL
	w.config = config

	// 设置模型信息
	w.modelInfo = ModelInfo{
		Name:          config.Model,
		Version:       "1.0.0",
		Type:          "text-generation",
		Provider:      "WebSocket",
		MaxTokens:     config.MaxTokens,
		ContextWindow: config.ContextWindow,
		Languages:     []string{"zh", "en"},
		Capabilities:  []string{"chat", "completion", "streaming"},
		LoadTime:      time.Now().UnixMilli(),
	}

	// 建立连接
	if err := w.connect(); err != nil {
		return fmt.Errorf("连接WebSocket失败: %w", err)
	}

	// 启动消息处理
	go w.handleMessages()

	// 启动心跳
	w.startPing()

	// 启动重连机制
	w.startReconnect()

	w.isInitialized = true

	log.Println("WebSocketLLM: 初始化成功")
	return nil
}

// GenerateResponse 生成回复
func (w *WebSocketLLM) GenerateResponse(ctx context.Context, messages []Message) (LLMResponse, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.isInitialized {
		return LLMResponse{}, ErrLLMNotInitialized
	}

	if !w.isConnected {
		return LLMResponse{}, ErrConnectionFailed
	}

	startTime := time.Now()

	// 生成请求ID
	w.requestID++
	requestID := w.requestID

	// 创建请求
	request := WebSocketRequest{
		ID:          requestID,
		Type:        "generate",
		Model:       w.config.Model,
		Messages:    messages,
		Stream:      false,
		Temperature: w.config.Temperature,
		TopP:        w.config.TopP,
		MaxTokens:   w.config.MaxTokens,
	}

	// 创建响应通道
	responseChan := make(chan LLMResponse, 1)
	w.pendingRequests[requestID] = responseChan

	// 发送请求
	if err := w.conn.WriteJSON(request); err != nil {
		delete(w.pendingRequests, requestID)
		return LLMResponse{}, err
	}

	// 等待响应
	select {
	case response := <-responseChan:
		delete(w.pendingRequests, requestID)
		response.ProcessTime = time.Since(startTime).Milliseconds()
		return response, nil
	case <-ctx.Done():
		delete(w.pendingRequests, requestID)
		return LLMResponse{}, ctx.Err()
	case <-time.After(time.Duration(w.config.Timeout) * time.Second):
		delete(w.pendingRequests, requestID)
		return LLMResponse{}, ErrTimeout
	}
}

// GenerateResponseStream 生成流式回复
func (w *WebSocketLLM) GenerateResponseStream(ctx context.Context, messages []Message) (<-chan LLMResponse, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.isInitialized {
		return nil, ErrLLMNotInitialized
	}

	if !w.isConnected {
		return nil, ErrConnectionFailed
	}

	// 生成请求ID
	w.requestID++
	requestID := w.requestID

	// 创建请求
	request := WebSocketRequest{
		ID:          requestID,
		Type:        "generate_stream",
		Model:       w.config.Model,
		Messages:    messages,
		Stream:      true,
		Temperature: w.config.Temperature,
		TopP:        w.config.TopP,
		MaxTokens:   w.config.MaxTokens,
	}

	// 创建响应通道
	responseChan := make(chan LLMResponse, 10)
	w.pendingRequests[requestID] = responseChan

	// 发送请求
	if err := w.conn.WriteJSON(request); err != nil {
		delete(w.pendingRequests, requestID)
		return nil, err
	}

	// 返回响应通道
	return responseChan, nil
}

// Chat 聊天对话
func (w *WebSocketLLM) Chat(ctx context.Context, userInput string, conversationID string) (LLMResponse, error) {
	// 获取或创建对话上下文
	conv := w.conversationManager.GetOrCreateConversation(
		conversationID,
		w.config.SystemPrompt,
		w.config.MaxContextLength,
	)

	// 添加用户消息
	userMessage := Message{
		Role:      "user",
		Content:   userInput,
		Timestamp: time.Now().UnixMilli(),
	}
	conv.Messages = append(conv.Messages, userMessage)

	// 修剪上下文（如果需要）
	if w.config.EnableContextTrim {
		w.trimContext(conv)
	}

	// 生成响应
	response, err := w.GenerateResponse(ctx, conv.Messages)
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
func (w *WebSocketLLM) ChatStream(ctx context.Context, userInput string, conversationID string) (<-chan LLMResponse, error) {
	// 获取或创建对话上下文
	conv := w.conversationManager.GetOrCreateConversation(
		conversationID,
		w.config.SystemPrompt,
		w.config.MaxContextLength,
	)

	// 添加用户消息
	userMessage := Message{
		Role:      "user",
		Content:   userInput,
		Timestamp: time.Now().UnixMilli(),
	}
	conv.Messages = append(conv.Messages, userMessage)

	// 修剪上下文（如果需要）
	if w.config.EnableContextTrim {
		w.trimContext(conv)
	}

	// 生成流式响应
	responseChan, err := w.GenerateResponseStream(ctx, conv.Messages)
	if err != nil {
		return nil, err
	}

	// 包装响应通道
	wrappedChan := make(chan LLMResponse, 10)
	go func() {
		defer close(wrappedChan)
		var fullContent string

		for response := range responseChan {
			response.ConversationID = conversationID
			wrappedChan <- response

			if response.IsDelta {
				fullContent += response.Content
			}

			if response.IsComplete {
				// 添加完整的助手消息到对话历史
				assistantMessage := Message{
					Role:      "assistant",
					Content:   fullContent,
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
func (w *WebSocketLLM) GetSupportedModels() []string {
	return []string{w.config.Model}
}

// SetModel 设置使用的模型
func (w *WebSocketLLM) SetModel(model string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.config.Model = model
	w.modelInfo.Name = model
	return nil
}

// GetModelInfo 获取模型信息
func (w *WebSocketLLM) GetModelInfo() ModelInfo {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.modelInfo
}

// Close 关闭LLM服务
func (w *WebSocketLLM) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 停止定时器
	if w.pingTicker != nil {
		w.pingTicker.Stop()
	}
	if w.reconnectTicker != nil {
		w.reconnectTicker.Stop()
	}

	// 关闭停止通道
	close(w.stopChan)

	// 关闭连接
	if w.conn != nil {
		w.conn.Close()
	}

	w.isInitialized = false
	w.isConnected = false

	log.Println("WebSocketLLM: 已关闭")
	return nil
}

// connect 建立连接
func (w *WebSocketLLM) connect() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// 设置请求头
	header := http.Header{}
	for key, value := range w.headers {
		header.Set(key, value)
	}

	conn, _, err := dialer.Dial(w.url, header)
	if err != nil {
		return err
	}

	w.conn = conn
	w.isConnected = true

	// 设置读写超时
	if w.config.WebSocketConfig.ReadTimeout > 0 {
		w.conn.SetReadDeadline(time.Now().Add(time.Duration(w.config.WebSocketConfig.ReadTimeout) * time.Second))
	}
	if w.config.WebSocketConfig.WriteTimeout > 0 {
		w.conn.SetWriteDeadline(time.Now().Add(time.Duration(w.config.WebSocketConfig.WriteTimeout) * time.Second))
	}

	log.Println("WebSocketLLM: 连接成功")
	return nil
}

// handleMessages 处理消息
func (w *WebSocketLLM) handleMessages() {
	for {
		select {
		case <-w.stopChan:
			return
		default:
			var response WebSocketResponse
			if err := w.conn.ReadJSON(&response); err != nil {
				if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocketLLM: 读取消息失败: %v", err)
				}
				w.isConnected = false
				return
			}

			// 处理响应
			w.processResponse(response)
		}
	}
}

// processResponse 处理响应
func (w *WebSocketLLM) processResponse(wsResponse WebSocketResponse) {
	w.mu.RLock()
	responseChan, exists := w.pendingRequests[wsResponse.ID]
	w.mu.RUnlock()

	if !exists {
		return
	}

	// 转换响应格式
	response := LLMResponse{
		Content:      wsResponse.Content,
		Role:         wsResponse.Role,
		Model:        wsResponse.Model,
		FinishReason: wsResponse.FinishReason,
		IsDelta:      wsResponse.IsDelta,
		IsComplete:   wsResponse.IsComplete,
		TokenUsage:   wsResponse.TokenUsage,
		Timestamp:    wsResponse.Timestamp,
	}

	// 处理错误
	if wsResponse.Error != "" {
		response.Error = fmt.Errorf(wsResponse.Error)
	}

	// 发送响应
	select {
	case responseChan <- response:
	default:
		// 通道已满，跳过
	}

	// 如果响应完成，清理请求
	if wsResponse.IsComplete || wsResponse.Error != "" {
		w.mu.Lock()
		delete(w.pendingRequests, wsResponse.ID)
		close(responseChan)
		w.mu.Unlock()
	}
}

// startPing 启动心跳
func (w *WebSocketLLM) startPing() {
	if w.config.WebSocketConfig.PingInterval <= 0 {
		return
	}

	w.pingTicker = time.NewTicker(time.Duration(w.config.WebSocketConfig.PingInterval) * time.Second)
	go func() {
		for {
			select {
			case <-w.pingTicker.C:
				if w.isConnected {
					if err := w.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						log.Printf("WebSocketLLM: 心跳失败: %v", err)
						w.isConnected = false
					}
				}
			case <-w.stopChan:
				return
			}
		}
	}()
}

// startReconnect 启动重连机制
func (w *WebSocketLLM) startReconnect() {
	if w.config.WebSocketConfig.ReconnectInterval <= 0 {
		return
	}

	w.reconnectTicker = time.NewTicker(time.Duration(w.config.WebSocketConfig.ReconnectInterval) * time.Second)
	go func() {
		reconnectCount := 0
		for {
			select {
			case <-w.reconnectTicker.C:
				if !w.isConnected && reconnectCount < w.config.WebSocketConfig.MaxReconnects {
					log.Printf("WebSocketLLM: 尝试重连 (%d/%d)", reconnectCount+1, w.config.WebSocketConfig.MaxReconnects)
					if err := w.connect(); err != nil {
						log.Printf("WebSocketLLM: 重连失败: %v", err)
						reconnectCount++
					} else {
						log.Println("WebSocketLLM: 重连成功")
						reconnectCount = 0
						go w.handleMessages()
					}
				}
			case <-w.stopChan:
				return
			}
		}
	}()
}

// trimContext 修剪对话上下文
func (w *WebSocketLLM) trimContext(conv *ConversationContext) {
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

// 注册WebSocket LLM
func init() {
	RegisterLLM("websocket", func(config LLMConfig) (LLMService, error) {
		return NewWebSocketLLM(config)
	})
}
