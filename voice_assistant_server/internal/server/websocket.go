package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"voice_assistant/pkg/protocol"

	"github.com/gorilla/websocket"
)

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	ReadBufferSize  int           `yaml:"read_buffer_size"`
	WriteBufferSize int           `yaml:"write_buffer_size"`
	MaxConnections  int           `yaml:"max_connections"`
	PingPeriod      time.Duration `yaml:"ping_period"`
	PongWait        time.Duration `yaml:"pong_wait"`
	WriteWait       time.Duration `yaml:"write_wait"`
}

// WebSocketServer WebSocket服务器
type WebSocketServer struct {
	config   WebSocketConfig
	upgrader websocket.Upgrader
	clients  map[string]*Client
	mu       sync.RWMutex

	// 消息处理器
	messageHandlers map[protocol.MessageType]MessageHandler

	// 处理器
	processor *MessageProcessor
}

// Client 客户端连接
type Client struct {
	ID       string
	Conn     *websocket.Conn
	SendChan chan *protocol.Message
	Server   *WebSocketServer
}

// MessageHandler 消息处理器函数类型
type MessageHandler func(client *Client, msg *protocol.Message) error

// NewWebSocketServer 创建新的WebSocket服务器
func NewWebSocketServer(config WebSocketConfig) *WebSocketServer {
	return &WebSocketServer{
		config: config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 生产环境需要严格检查
			},
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
		},
		clients:         make(map[string]*Client),
		messageHandlers: make(map[protocol.MessageType]MessageHandler),
	}
}

// SetProcessor 设置消息处理器
func (s *WebSocketServer) SetProcessor(processor *MessageProcessor) {
	s.processor = processor
}

// HandleConnection 处理WebSocket连接
func (s *WebSocketServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// 检查连接数限制
	s.mu.RLock()
	currentConnections := len(s.clients)
	s.mu.RUnlock()

	if currentConnections >= s.config.MaxConnections {
		http.Error(w, "连接数已达上限", http.StatusServiceUnavailable)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = s.generateSessionID()
	}

	client := &Client{
		ID:       sessionID,
		Conn:     conn,
		SendChan: make(chan *protocol.Message, 100),
		Server:   s,
	}

	s.mu.Lock()
	s.clients[sessionID] = client
	s.mu.Unlock()

	log.Printf("客户端连接: %s", sessionID)

	// 发送连接确认
	statusData := &protocol.StatusData{
		State:             "connected",
		Mode:              "idle",
		ConcurrentStreams: 0,
	}
	statusMsg := protocol.NewMessage(protocol.Status, sessionID, statusData)
	client.SendMessage(statusMsg)

	// 启动客户端处理协程
	go client.readLoop()
	go client.writeLoop()
}

// RegisterHandler 注册消息处理器
func (s *WebSocketServer) RegisterHandler(msgType protocol.MessageType, handler MessageHandler) {
	s.messageHandlers[msgType] = handler
}

// BroadcastToClient 向指定客户端发送消息
func (s *WebSocketServer) BroadcastToClient(clientID string, msg *protocol.Message) error {
	s.mu.RLock()
	client, exists := s.clients[clientID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("客户端不存在: %s", clientID)
	}

	return client.SendMessage(msg)
}

// GetClientCount 获取当前连接的客户端数量
func (s *WebSocketServer) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// SendMessage 发送消息给客户端
func (c *Client) SendMessage(msg *protocol.Message) error {
	select {
	case c.SendChan <- msg:
		return nil
	default:
		return fmt.Errorf("客户端发送队列已满")
	}
}

// readLoop 读取消息循环
func (c *Client) readLoop() {
	defer func() {
		c.Server.mu.Lock()
		delete(c.Server.clients, c.ID)
		c.Server.mu.Unlock()
		c.Conn.Close()
		log.Printf("客户端断开: %s", c.ID)
	}()

	// 设置读取超时
	c.Conn.SetReadDeadline(time.Now().Add(c.Server.config.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Server.config.PongWait))
		return nil
	})

	for {
		_, messageData, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}

		var msg protocol.Message
		if err := json.Unmarshal(messageData, &msg); err != nil {
			log.Printf("解析消息失败: %v", err)
			continue
		}

		// 处理消息
		if handler, exists := c.Server.messageHandlers[msg.Type]; exists {
			if err := handler(c, &msg); err != nil {
				log.Printf("处理消息失败: %v", err)
				// 发送错误响应
				errorData := &protocol.ErrorData{
					Code:        "PROCESSING_ERROR",
					Message:     err.Error(),
					Recoverable: true,
				}
				errorMsg := protocol.NewMessage(protocol.Error, c.ID, errorData)
				c.SendMessage(errorMsg)
			}
		} else {
			log.Printf("未找到消息处理器: %s", msg.Type)
		}
	}
}

// writeLoop 写入消息循环
func (c *Client) writeLoop() {
	ticker := time.NewTicker(c.Server.config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg := <-c.SendChan:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Server.config.WriteWait))

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("序列化消息失败: %v", err)
				continue
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("发送消息失败: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Server.config.WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("发送Ping失败: %v", err)
				return
			}
		}
	}
}

// generateSessionID 生成会话ID
func (s *WebSocketServer) generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
