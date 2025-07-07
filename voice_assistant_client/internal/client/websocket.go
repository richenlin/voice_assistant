package client

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"voice_assistant/pkg/protocol"

	"github.com/gorilla/websocket"
)

// WebSocketClient WebSocket客户端
type WebSocketClient struct {
	// 连接配置
	serverURL            string
	sessionID            string
	reconnectInterval    time.Duration
	maxReconnectAttempts int
	connectionTimeout    time.Duration
	pingInterval         time.Duration
	pongTimeout          time.Duration

	// 连接状态
	conn        *websocket.Conn
	isConnected bool
	mu          sync.RWMutex

	// 消息处理
	messageHandlers map[protocol.MessageType]MessageHandler

	// 通道
	sendChan    chan *protocol.Message
	receiveChan chan *protocol.Message
	closeChan   chan struct{}

	// 重连控制
	reconnectCount  int
	lastConnectTime time.Time

	// 统计信息
	stats ConnectionStats
}

// MessageHandler 消息处理器函数类型
type MessageHandler func(msg *protocol.Message) error

// ConnectionStats 连接统计信息
type ConnectionStats struct {
	ConnectTime      time.Time
	LastMessageTime  time.Time
	MessagesSent     int64
	MessagesReceived int64
	ReconnectCount   int
	BytesSent        int64
	BytesReceived    int64
}

// ClientConfig 客户端配置
type ClientConfig struct {
	ServerURL            string        `yaml:"server_url"`
	SessionID            string        `yaml:"session_id"`
	ReconnectInterval    time.Duration `yaml:"reconnect_interval"`
	MaxReconnectAttempts int           `yaml:"max_reconnect_attempts"`
	ConnectionTimeout    time.Duration `yaml:"connection_timeout"`
	PingInterval         time.Duration `yaml:"ping_interval"`
	PongTimeout          time.Duration `yaml:"pong_timeout"`
}

// NewWebSocketClient 创建WebSocket客户端
func NewWebSocketClient(config ClientConfig) *WebSocketClient {
	if config.SessionID == "" {
		config.SessionID = generateSessionID()
	}

	return &WebSocketClient{
		serverURL:            config.ServerURL,
		sessionID:            config.SessionID,
		reconnectInterval:    config.ReconnectInterval,
		maxReconnectAttempts: config.MaxReconnectAttempts,
		connectionTimeout:    config.ConnectionTimeout,
		pingInterval:         config.PingInterval,
		pongTimeout:          config.PongTimeout,

		messageHandlers: make(map[protocol.MessageType]MessageHandler),
		sendChan:        make(chan *protocol.Message, 100),
		receiveChan:     make(chan *protocol.Message, 100),
		closeChan:       make(chan struct{}),
	}
}

// Connect 连接到服务器
func (c *WebSocketClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.isConnected {
		c.mu.Unlock()
		return fmt.Errorf("已经连接到服务器")
	}
	c.mu.Unlock()

	// 解析URL
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("解析服务器URL失败: %w", err)
	}

	// 添加会话ID参数
	q := u.Query()
	q.Set("session_id", c.sessionID)
	u.RawQuery = q.Encode()

	// 设置连接超时
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = c.connectionTimeout

	// 建立连接
	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		c.reconnectCount++
		return fmt.Errorf("连接服务器失败: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.isConnected = true
	c.lastConnectTime = time.Now()
	c.stats.ConnectTime = time.Now()
	if c.reconnectCount > 0 {
		c.stats.ReconnectCount = c.reconnectCount
	}
	c.mu.Unlock()

	// 设置连接参数
	c.setupConnection()

	// 启动消息处理协程
	go c.readLoop(ctx)
	go c.writeLoop(ctx)
	go c.messageProcessor(ctx)
	go c.pingLoop(ctx)

	log.Printf("WebSocket连接已建立: %s (会话ID: %s)", c.serverURL, c.sessionID)
	return nil
}

// Disconnect 断开连接
func (c *WebSocketClient) Disconnect() error {
	c.mu.Lock()
	if !c.isConnected {
		c.mu.Unlock()
		return nil
	}
	c.isConnected = false
	c.mu.Unlock()

	// 发送关闭信号
	close(c.closeChan)

	// 关闭WebSocket连接
	if c.conn != nil {
		c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
	}

	log.Printf("WebSocket连接已断开")
	return nil
}

// SendAudioStream 发送音频流
func (c *WebSocketClient) SendAudioStream(audioData []byte, chunkID int, isFinal bool) error {
	if !c.IsConnected() {
		return fmt.Errorf("未连接到服务器")
	}

	msg := protocol.NewAudioStreamMessage(c.sessionID, "pcm_16khz_16bit", chunkID, isFinal, audioData)

	select {
	case c.sendChan <- msg:
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("发送音频流超时")
	}
}

// SendCommand 发送命令
func (c *WebSocketClient) SendCommand(command, mode string, parameters map[string]interface{}) error {
	if !c.IsConnected() {
		return fmt.Errorf("未连接到服务器")
	}

	msg := protocol.NewCommandMessage(c.sessionID, command, mode, parameters)

	select {
	case c.sendChan <- msg:
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("发送命令超时")
	}
}

// RegisterHandler 注册消息处理器
func (c *WebSocketClient) RegisterHandler(msgType protocol.MessageType, handler MessageHandler) {
	c.messageHandlers[msgType] = handler
}

// IsConnected 检查是否已连接
func (c *WebSocketClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// GetStats 获取连接统计信息
func (c *WebSocketClient) GetStats() ConnectionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// GetSessionID 获取会话ID
func (c *WebSocketClient) GetSessionID() string {
	return c.sessionID
}

// setupConnection 设置连接参数
func (c *WebSocketClient) setupConnection() {
	// 设置读取超时
	c.conn.SetReadDeadline(time.Now().Add(c.pongTimeout))

	// 设置Pong处理器
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.pongTimeout))
		return nil
	})

	// 设置关闭处理器
	c.conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("WebSocket连接关闭: code=%d, text=%s", code, text)
		c.handleDisconnection()
		return nil
	})
}

// readLoop 读取消息循环
func (c *WebSocketClient) readLoop(ctx context.Context) {
	defer func() {
		c.handleDisconnection()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closeChan:
			return
		default:
			_, messageData, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket读取错误: %v", err)
				}
				return
			}

			// 更新统计信息
			c.mu.Lock()
			c.stats.MessagesReceived++
			c.stats.BytesReceived += int64(len(messageData))
			c.stats.LastMessageTime = time.Now()
			c.mu.Unlock()

			// 解析消息
			msg, err := protocol.FromJSON(messageData)
			if err != nil {
				log.Printf("解析消息失败: %v", err)
				continue
			}

			// 发送到处理通道
			select {
			case c.receiveChan <- msg:
			case <-time.After(time.Second):
				log.Printf("接收队列已满，丢弃消息")
			}
		}
	}
}

// writeLoop 写入消息循环
func (c *WebSocketClient) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closeChan:
			return
		case msg := <-c.sendChan:
			if !c.IsConnected() {
				continue
			}

			// 序列化消息
			data, err := msg.ToJSON()
			if err != nil {
				log.Printf("序列化消息失败: %v", err)
				continue
			}

			// 设置写入超时
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			// 发送消息
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("发送消息失败: %v", err)
				c.handleDisconnection()
				return
			}

			// 更新统计信息
			c.mu.Lock()
			c.stats.MessagesSent++
			c.stats.BytesSent += int64(len(data))
			c.mu.Unlock()
		}
	}
}

// messageProcessor 消息处理器
func (c *WebSocketClient) messageProcessor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closeChan:
			return
		case msg := <-c.receiveChan:
			if handler, exists := c.messageHandlers[msg.Type]; exists {
				if err := handler(msg); err != nil {
					log.Printf("处理消息失败: %v", err)
				}
			} else {
				log.Printf("未找到消息处理器: %s", msg.Type)
			}
		}
	}
}

// pingLoop Ping循环
func (c *WebSocketClient) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closeChan:
			return
		case <-ticker.C:
			if !c.IsConnected() {
				continue
			}

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("发送Ping失败: %v", err)
				c.handleDisconnection()
				return
			}
		}
	}
}

// handleDisconnection 处理断开连接
func (c *WebSocketClient) handleDisconnection() {
	c.mu.Lock()
	wasConnected := c.isConnected
	c.isConnected = false
	c.mu.Unlock()

	if !wasConnected {
		return
	}

	log.Printf("连接断开，准备重连...")

	// 尝试重连
	go c.attemptReconnect()
}

// attemptReconnect 尝试重连
func (c *WebSocketClient) attemptReconnect() {
	for c.reconnectCount < c.maxReconnectAttempts {
		// 等待重连间隔
		time.Sleep(c.reconnectInterval)

		log.Printf("尝试重连 (%d/%d)...", c.reconnectCount+1, c.maxReconnectAttempts)

		// 尝试连接
		ctx, cancel := context.WithTimeout(context.Background(), c.connectionTimeout)
		if err := c.Connect(ctx); err != nil {
			log.Printf("重连失败: %v", err)
			cancel()
			continue
		}
		cancel()

		log.Printf("重连成功")
		return
	}

	log.Printf("重连失败，已达到最大尝试次数")
}

// generateSessionID 生成会话ID
func generateSessionID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// StartSession 启动会话
func (c *WebSocketClient) StartSession(mode string) error {
	return c.SendCommand(protocol.CmdStartSession, mode, nil)
}

// StopSession 停止会话
func (c *WebSocketClient) StopSession() error {
	return c.SendCommand(protocol.CmdStopSession, "", nil)
}

// PauseSession 暂停会话
func (c *WebSocketClient) PauseSession() error {
	return c.SendCommand(protocol.CmdPause, "", nil)
}

// ResumeSession 恢复会话
func (c *WebSocketClient) ResumeSession() error {
	return c.SendCommand(protocol.CmdResume, "", nil)
}

// SetMode 设置模式
func (c *WebSocketClient) SetMode(mode string) error {
	params := map[string]interface{}{
		"mode": mode,
	}
	return c.SendCommand(protocol.CmdSetMode, mode, params)
}

// GetStatus 获取状态
func (c *WebSocketClient) GetStatus() error {
	return c.SendCommand(protocol.CmdGetStatus, "", nil)
}

// InterruptSession 中断会话
func (c *WebSocketClient) InterruptSession() error {
	return c.SendCommand(protocol.CmdInterrupt, "", nil)
}

// ClearContext 清除上下文
func (c *WebSocketClient) ClearContext() error {
	return c.SendCommand(protocol.CmdClearContext, "", nil)
}
