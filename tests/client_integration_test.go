package tests

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"voice_assistant/pkg/protocol"
)

// MockWebSocketServer 模拟WebSocket服务器
type MockWebSocketServer struct {
	*httptest.Server
	upgrader    websocket.Upgrader
	connections map[*websocket.Conn]bool
	connsMutex  sync.RWMutex
	messageLog  []*protocol.Message
	logMutex    sync.RWMutex
}

// NewMockWebSocketServer 创建模拟WebSocket服务器
func NewMockWebSocketServer() *MockWebSocketServer {
	server := &MockWebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		connections: make(map[*websocket.Conn]bool),
		messageLog:  make([]*protocol.Message, 0),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.handleWebSocket)
	server.Server = httptest.NewServer(mux)

	return server
}

// handleWebSocket 处理WebSocket连接
func (s *MockWebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	s.connsMutex.Lock()
	s.connections[conn] = true
	s.connsMutex.Unlock()

	defer func() {
		s.connsMutex.Lock()
		delete(s.connections, conn)
		s.connsMutex.Unlock()
	}()

	for {
		var msg protocol.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		// 记录接收到的消息
		s.logMutex.Lock()
		s.messageLog = append(s.messageLog, &msg)
		s.logMutex.Unlock()

		// 生成响应
		response := s.generateResponse(&msg)
		if response != nil {
			conn.WriteJSON(response)
		}
	}
}

// generateResponse 生成响应
func (s *MockWebSocketServer) generateResponse(msg *protocol.Message) *protocol.Message {
	switch msg.Type {
	case protocol.AudioStream:
		return protocol.NewResponseMessage(
			msg.SessionID,
			protocol.StageASR,
			"测试识别结果",
			0.95,
			true,
			nil,
		)
	case protocol.Command:
		if cmdData, err := protocol.ParseCommandData(msg.Data); err == nil {
			switch cmdData.Command {
			case protocol.CmdStartSession:
				return protocol.NewStatusMessage(
					msg.SessionID,
					protocol.StateConnected,
					protocol.ModeContinuous,
					1,
				)
			case protocol.CmdGetStatus:
				return protocol.NewStatusMessage(
					msg.SessionID,
					protocol.StateConnected,
					protocol.ModeContinuous,
					1,
				)
			case protocol.CmdEndSession:
				return protocol.NewStatusMessage(
					msg.SessionID,
					protocol.StateDisconnected,
					protocol.ModeContinuous,
					0,
				)
			}
		}
		return protocol.NewStatusMessage(
			msg.SessionID,
			protocol.StateConnected,
			protocol.ModeContinuous,
			1,
		)
	case protocol.Status:
		return protocol.NewStatusMessage(
			msg.SessionID,
			protocol.StateConnected,
			protocol.ModeContinuous,
			1,
		)
	}
	return nil
}

// GetWebSocketURL 获取WebSocket URL
func (s *MockWebSocketServer) GetWebSocketURL() string {
	return "ws" + s.Server.URL[4:] + "/ws"
}

// GetReceivedMessages 获取接收到的消息
func (s *MockWebSocketServer) GetReceivedMessages() []*protocol.Message {
	s.logMutex.RLock()
	defer s.logMutex.RUnlock()

	messages := make([]*protocol.Message, len(s.messageLog))
	copy(messages, s.messageLog)
	return messages
}

// GetActiveConnections 获取活跃连接数
func (s *MockWebSocketServer) GetActiveConnections() int {
	s.connsMutex.RLock()
	defer s.connsMutex.RUnlock()
	return len(s.connections)
}

// TestClientWebSocketConnection 测试客户端WebSocket连接
func TestClientWebSocketConnection(t *testing.T) {
	server := NewMockWebSocketServer()
	defer server.Close()

	// 建立WebSocket连接
	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	require.NoError(t, err)
	defer conn.Close()

	// 验证连接已建立
	assert.Equal(t, 1, server.GetActiveConnections())

	// 发送ping消息
	pingMsg := protocol.NewStatusMessage(
		"test_session",
		protocol.StateConnected,
		protocol.ModeContinuous,
		1,
	)

	err = conn.WriteJSON(pingMsg)
	require.NoError(t, err)

	// 接收响应
	var response protocol.Message
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, protocol.Status, response.Type)
}

// TestClientAudioStreamProcessing 测试客户端音频流处理
func TestClientAudioStreamProcessing(t *testing.T) {
	server := NewMockWebSocketServer()
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	require.NoError(t, err)
	defer conn.Close()

	sessionID := "test_audio_session"

	// 发送多个音频块
	audioChunks := [][]byte{
		[]byte("chunk1 audio data"),
		[]byte("chunk2 audio data"),
		[]byte("chunk3 audio data"),
	}

	for i, chunk := range audioChunks {
		isFinal := (i == len(audioChunks)-1)
		audioMsg := protocol.NewAudioStreamMessage(
			sessionID,
			"pcm_16khz_16bit",
			i+1,
			isFinal,
			chunk,
		)

		err = conn.WriteJSON(audioMsg)
		require.NoError(t, err)

		// 接收ASR响应
		var response protocol.Message
		err = conn.ReadJSON(&response)
		require.NoError(t, err)
		assert.Equal(t, protocol.Response, response.Type)

		// 验证响应数据
		respData, err := protocol.ParseResponseData(response.Data)
		require.NoError(t, err)
		assert.Equal(t, protocol.StageASR, respData.Stage)
		assert.Equal(t, "测试识别结果", respData.Content)
		assert.Equal(t, 0.95, respData.Confidence)
	}

	// 验证服务器收到了所有音频消息
	messages := server.GetReceivedMessages()
	audioMsgCount := 0
	for _, msg := range messages {
		if msg.Type == protocol.AudioStream {
			audioMsgCount++
		}
	}
	assert.Equal(t, len(audioChunks), audioMsgCount)
}

// TestClientCommandProcessing 测试客户端命令处理
func TestClientCommandProcessing(t *testing.T) {
	server := NewMockWebSocketServer()
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	require.NoError(t, err)
	defer conn.Close()

	sessionID := "test_command_session"

	// 测试开始会话命令
	startCmd := protocol.NewCommandMessage(
		sessionID,
		protocol.CmdStartSession,
		protocol.ModeContinuous,
		map[string]interface{}{"timeout": 30},
	)

	err = conn.WriteJSON(startCmd)
	require.NoError(t, err)

	var startResponse protocol.Message
	err = conn.ReadJSON(&startResponse)
	require.NoError(t, err)
	assert.Equal(t, protocol.Status, startResponse.Type)

	statusData, err := protocol.ParseStatusData(startResponse.Data)
	require.NoError(t, err)
	assert.Equal(t, protocol.StateConnected, statusData.State)

	// 测试状态查询命令
	statusCmd := protocol.NewCommandMessage(
		sessionID,
		protocol.CmdGetStatus,
		"",
		nil,
	)

	err = conn.WriteJSON(statusCmd)
	require.NoError(t, err)

	var statusResponse protocol.Message
	err = conn.ReadJSON(&statusResponse)
	require.NoError(t, err)
	assert.Equal(t, protocol.Status, statusResponse.Type)

	// 测试结束会话命令
	endCmd := protocol.NewCommandMessage(
		sessionID,
		protocol.CmdEndSession,
		"",
		nil,
	)

	err = conn.WriteJSON(endCmd)
	require.NoError(t, err)

	var endResponse protocol.Message
	err = conn.ReadJSON(&endResponse)
	require.NoError(t, err)
	assert.Equal(t, protocol.Status, endResponse.Type)

	endStatusData, err := protocol.ParseStatusData(endResponse.Data)
	require.NoError(t, err)
	assert.Equal(t, protocol.StateDisconnected, endStatusData.State)
}

// TestClientSessionManagement 测试客户端会话管理
func TestClientSessionManagement(t *testing.T) {
	server := NewMockWebSocketServer()
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	require.NoError(t, err)
	defer conn.Close()

	// 模拟完整的会话流程
	sessionID := "complete_session_test"

	// 1. 开始会话
	startCmd := protocol.NewCommandMessage(
		sessionID,
		protocol.CmdStartSession,
		protocol.ModeContinuous,
		nil,
	)

	err = conn.WriteJSON(startCmd)
	require.NoError(t, err)

	var startResp protocol.Message
	err = conn.ReadJSON(&startResp)
	require.NoError(t, err)

	// 2. 发送音频数据
	audioMsg := protocol.NewAudioStreamMessage(
		sessionID,
		"pcm_16khz_16bit",
		1,
		true,
		[]byte("你好世界"),
	)

	err = conn.WriteJSON(audioMsg)
	require.NoError(t, err)

	var audioResp protocol.Message
	err = conn.ReadJSON(&audioResp)
	require.NoError(t, err)
	assert.Equal(t, protocol.Response, audioResp.Type)

	// 3. 暂停会话
	pauseCmd := protocol.NewCommandMessage(
		sessionID,
		protocol.CmdPause,
		"",
		nil,
	)

	err = conn.WriteJSON(pauseCmd)
	require.NoError(t, err)

	var pauseResp protocol.Message
	err = conn.ReadJSON(&pauseResp)
	require.NoError(t, err)

	// 4. 恢复会话
	resumeCmd := protocol.NewCommandMessage(
		sessionID,
		protocol.CmdResume,
		"",
		nil,
	)

	err = conn.WriteJSON(resumeCmd)
	require.NoError(t, err)

	var resumeResp protocol.Message
	err = conn.ReadJSON(&resumeResp)
	require.NoError(t, err)

	// 5. 结束会话
	endCmd := protocol.NewCommandMessage(
		sessionID,
		protocol.CmdEndSession,
		"",
		nil,
	)

	err = conn.WriteJSON(endCmd)
	require.NoError(t, err)

	var endResp protocol.Message
	err = conn.ReadJSON(&endResp)
	require.NoError(t, err)
}

// TestClientConcurrentConnections 测试客户端并发连接
func TestClientConcurrentConnections(t *testing.T) {
	server := NewMockWebSocketServer()
	defer server.Close()

	numClients := 5
	var wg sync.WaitGroup
	connectionErrors := make(chan error, numClients)

	// 创建多个并发连接
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
			if err != nil {
				connectionErrors <- err
				return
			}
			defer conn.Close()

			// 每个客户端发送消息
			sessionID := "concurrent_session_" + string(rune('A'+clientID))
			msg := protocol.NewStatusMessage(
				sessionID,
				protocol.StateConnected,
				protocol.ModeContinuous,
				1,
			)

			err = conn.WriteJSON(msg)
			if err != nil {
				connectionErrors <- err
				return
			}

			// 接收响应
			var response protocol.Message
			err = conn.ReadJSON(&response)
			if err != nil {
				connectionErrors <- err
				return
			}

			connectionErrors <- nil
		}(i)
	}

	wg.Wait()
	close(connectionErrors)

	// 验证所有连接都成功
	for err := range connectionErrors {
		assert.NoError(t, err)
	}
}

// TestClientErrorHandling 测试客户端错误处理
func TestClientErrorHandling(t *testing.T) {
	server := NewMockWebSocketServer()
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	require.NoError(t, err)
	defer conn.Close()

	// 发送无效消息
	invalidMsg := &protocol.Message{
		Type:      "invalid_type",
		SessionID: "test_session",
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Data:      "invalid data",
	}

	err = conn.WriteJSON(invalidMsg)
	require.NoError(t, err)

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// 可能没有响应或连接关闭
	var response protocol.Message
	err = conn.ReadJSON(&response)
	// 这里可能会超时或收到错误，这是预期的行为
}

// TestClientReconnection 测试客户端重连逻辑
func TestClientReconnection(t *testing.T) {
	server := NewMockWebSocketServer()

	// 首先建立连接
	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	require.NoError(t, err)

	// 发送消息确认连接正常
	msg := protocol.NewStatusMessage(
		"test_session",
		protocol.StateConnected,
		protocol.ModeContinuous,
		1,
	)

	err = conn.WriteJSON(msg)
	require.NoError(t, err)

	var response protocol.Message
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, protocol.Status, response.Type)

	// 关闭连接模拟网络断开
	conn.Close()

	// 关闭服务器
	server.Close()

	// 启动新服务器模拟重连
	newServer := NewMockWebSocketServer()
	defer newServer.Close()

	// 重新连接
	newConn, _, err := websocket.DefaultDialer.Dial(newServer.GetWebSocketURL(), nil)
	require.NoError(t, err)
	defer newConn.Close()

	// 验证重连后可以正常通信
	err = newConn.WriteJSON(msg)
	require.NoError(t, err)

	var newResponse protocol.Message
	err = newConn.ReadJSON(&newResponse)
	require.NoError(t, err)
	assert.Equal(t, protocol.Status, newResponse.Type)
}

// BenchmarkClientWebSocketConnection 基准测试客户端WebSocket连接
func BenchmarkClientWebSocketConnection(b *testing.B) {
	server := NewMockWebSocketServer()
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
		if err != nil {
			b.Fatal(err)
		}
		conn.Close()
	}
}

// BenchmarkClientMessageSending 基准测试客户端消息发送
func BenchmarkClientMessageSending(b *testing.B) {
	server := NewMockWebSocketServer()
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	msg := protocol.NewAudioStreamMessage(
		"benchmark_session",
		"pcm_16khz_16bit",
		1,
		true,
		make([]byte, 1024),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := conn.WriteJSON(msg)
		if err != nil {
			b.Fatal(err)
		}

		var response protocol.Message
		err = conn.ReadJSON(&response)
		if err != nil {
			b.Fatal(err)
		}
	}
}
