package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	upgrader websocket.Upgrader
}

// NewMockWebSocketServer 创建模拟WebSocket服务器
func NewMockWebSocketServer() *MockWebSocketServer {
	server := &MockWebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
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

	for {
		var msg protocol.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		// 回显消息
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

// TestWebSocketConnection 测试WebSocket连接
func TestWebSocketConnection(t *testing.T) {
	// 创建模拟服务器
	server := NewMockWebSocketServer()
	defer server.Close()

	// 连接WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	require.NoError(t, err)
	defer conn.Close()

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

// TestAudioStreamProcessing 测试音频流处理
func TestAudioStreamProcessing(t *testing.T) {
	server := NewMockWebSocketServer()
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	require.NoError(t, err)
	defer conn.Close()

	// 发送音频流消息
	audioData := []byte("test audio data")
	audioMsg := protocol.NewAudioStreamMessage(
		"test_session",
		"pcm_16khz_16bit",
		1,
		true,
		audioData,
	)

	err = conn.WriteJSON(audioMsg)
	require.NoError(t, err)

	// 接收响应
	var response protocol.Message
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, protocol.Response, response.Type)

	// 验证响应数据
	responseData, err := protocol.ParseResponseData(response.Data)
	require.NoError(t, err)
	assert.Equal(t, protocol.StageASR, responseData.Stage)
	assert.Equal(t, "测试识别结果", responseData.Content)
}

// TestCommandProcessing 测试命令处理
func TestCommandProcessing(t *testing.T) {
	server := NewMockWebSocketServer()
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	require.NoError(t, err)
	defer conn.Close()

	// 发送开始会话命令
	cmdMsg := protocol.NewCommandMessage(
		"test_session",
		protocol.CmdStartSession,
		protocol.ModeContinuous,
		nil,
	)

	err = conn.WriteJSON(cmdMsg)
	require.NoError(t, err)

	// 接收响应
	var response protocol.Message
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, protocol.Status, response.Type)

	// 验证状态数据
	statusData, err := protocol.ParseStatusData(response.Data)
	require.NoError(t, err)
	assert.Equal(t, protocol.StateConnected, statusData.State)
}

// TestMessageSerialization 测试消息序列化
func TestMessageSerialization(t *testing.T) {
	// 创建测试消息
	msg := protocol.NewAudioStreamMessage(
		"test_session",
		"pcm_16khz_16bit",
		1,
		true,
		[]byte("test audio"),
	)

	// 序列化
	data, err := msg.ToJSON()
	require.NoError(t, err)
	assert.True(t, json.Valid(data))

	// 反序列化
	parsedMsg, err := protocol.FromJSON(data)
	require.NoError(t, err)
	assert.Equal(t, msg.Type, parsedMsg.Type)
	assert.Equal(t, msg.SessionID, parsedMsg.SessionID)
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
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

	// 尝试读取响应（可能没有响应）
	var response protocol.Message
	err = conn.ReadJSON(&response)
	// 无效消息可能导致连接关闭或没有响应
}

// BenchmarkWebSocketConnection 基准测试WebSocket连接
func BenchmarkWebSocketConnection(b *testing.B) {
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

// BenchmarkMessageProcessing 基准测试消息处理
func BenchmarkMessageProcessing(b *testing.B) {
	server := NewMockWebSocketServer()
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(server.GetWebSocketURL(), nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	msg := protocol.NewAudioStreamMessage(
		"test_session",
		"pcm_16khz_16bit",
		1,
		true,
		[]byte("test audio data"),
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
