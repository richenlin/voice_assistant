package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"voice_assistant/pkg/protocol"
)

// TestProtocolBasicFunctionality 测试protocol包的基本功能
func TestProtocolBasicFunctionality(t *testing.T) {
	// 测试音频流消息创建
	audioMsg := protocol.NewAudioStreamMessage(
		"test_session",
		"pcm_16khz_16bit",
		1,
		true,
		[]byte("test audio data"),
	)

	assert.Equal(t, protocol.AudioStream, audioMsg.Type)
	assert.Equal(t, "test_session", audioMsg.SessionID)
	assert.NotZero(t, audioMsg.Timestamp)

	// 测试命令消息创建
	cmdMsg := protocol.NewCommandMessage(
		"test_session",
		protocol.CmdStartSession,
		protocol.ModeContinuous,
		nil,
	)

	assert.Equal(t, protocol.Command, cmdMsg.Type)
	assert.Equal(t, "test_session", cmdMsg.SessionID)

	// 测试响应消息创建
	respMsg := protocol.NewResponseMessage(
		"test_session",
		protocol.StageASR,
		"测试识别结果",
		0.95,
		true,
		nil,
	)

	assert.Equal(t, protocol.Response, respMsg.Type)
	assert.Equal(t, "test_session", respMsg.SessionID)

	// 测试状态消息创建
	statusMsg := protocol.NewStatusMessage(
		"test_session",
		protocol.StateConnected,
		protocol.ModeContinuous,
		1,
	)

	assert.Equal(t, protocol.Status, statusMsg.Type)
	assert.Equal(t, "test_session", statusMsg.SessionID)

	// 测试错误消息创建
	errorMsg := protocol.NewErrorMessage(
		"test_session",
		protocol.ErrConnectionFailed,
		"连接失败",
		true,
	)

	assert.Equal(t, protocol.Error, errorMsg.Type)
	assert.Equal(t, "test_session", errorMsg.SessionID)
}

// TestProtocolSerialization 测试消息序列化和反序列化
func TestProtocolSerialization(t *testing.T) {
	// 创建测试消息
	originalMsg := protocol.NewAudioStreamMessage(
		"test_session",
		"pcm_16khz_16bit",
		1,
		true,
		[]byte("test audio data"),
	)

	// 序列化
	data, err := originalMsg.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// 反序列化
	parsedMsg, err := protocol.FromJSON(data)
	require.NoError(t, err)
	assert.Equal(t, originalMsg.Type, parsedMsg.Type)
	assert.Equal(t, originalMsg.SessionID, parsedMsg.SessionID)
	assert.Equal(t, originalMsg.Timestamp, parsedMsg.Timestamp)
}

// TestProtocolConstants 测试所有协议常量
func TestProtocolConstants(t *testing.T) {
	// 测试消息类型
	assert.Equal(t, "audio_stream", string(protocol.AudioStream))
	assert.Equal(t, "command", string(protocol.Command))
	assert.Equal(t, "response", string(protocol.Response))
	assert.Equal(t, "status", string(protocol.Status))
	assert.Equal(t, "error", string(protocol.Error))

	// 测试命令类型
	assert.Equal(t, "start_session", protocol.CmdStartSession)
	assert.Equal(t, "end_session", protocol.CmdEndSession)
	assert.Equal(t, "stop_session", protocol.CmdStopSession)
	assert.Equal(t, "get_status", protocol.CmdGetStatus)
	assert.Equal(t, "pause", protocol.CmdPause)
	assert.Equal(t, "resume", protocol.CmdResume)

	// 测试模式
	assert.Equal(t, "continuous", protocol.ModeContinuous)
	assert.Equal(t, "wakeword", protocol.ModeWakeword)
	assert.Equal(t, "interrupt", protocol.ModeInterrupt)
	assert.Equal(t, "single", protocol.ModeSingle)

	// 测试处理阶段
	assert.Equal(t, "asr", protocol.StageASR)
	assert.Equal(t, "llm", protocol.StageLLM)
	assert.Equal(t, "tts", protocol.StageTTS)

	// 测试状态
	assert.Equal(t, "idle", protocol.StateIdle)
	assert.Equal(t, "listening", protocol.StateListening)
	assert.Equal(t, "processing", protocol.StateProcessing)
	assert.Equal(t, "speaking", protocol.StateSpeaking)
	assert.Equal(t, "connected", protocol.StateConnected)
	assert.Equal(t, "disconnected", protocol.StateDisconnected)

	// 测试错误代码
	assert.Equal(t, "CONNECTION_FAILED", protocol.ErrConnectionFailed)
	assert.Equal(t, "INTERNAL_ERROR", protocol.ErrInternalError)
	assert.Equal(t, "ASR_FAILED", protocol.ErrASRFailed)
	assert.Equal(t, "LLM_FAILED", protocol.ErrLLMFailed)
	assert.Equal(t, "TTS_FAILED", protocol.ErrTTSFailed)
}

// TestProtocolDataParsing 测试数据解析功能
func TestProtocolDataParsing(t *testing.T) {
	// 测试音频流数据解析
	audioMsg := protocol.NewAudioStreamMessage(
		"test_session",
		"pcm_16khz_16bit",
		1,
		true,
		[]byte("test audio"),
	)

	audioData, err := protocol.ParseAudioStreamData(audioMsg.Data)
	require.NoError(t, err)
	assert.Equal(t, "pcm_16khz_16bit", audioData.Format)
	assert.Equal(t, 1, audioData.ChunkID)
	assert.True(t, audioData.IsFinal)
	assert.Equal(t, []byte("test audio"), audioData.AudioData)

	// 测试命令数据解析
	cmdMsg := protocol.NewCommandMessage(
		"test_session",
		protocol.CmdStartSession,
		protocol.ModeContinuous,
		map[string]interface{}{"timeout": 30},
	)

	cmdData, err := protocol.ParseCommandData(cmdMsg.Data)
	require.NoError(t, err)
	assert.Equal(t, protocol.CmdStartSession, cmdData.Command)
	assert.Equal(t, protocol.ModeContinuous, cmdData.Mode)
	assert.Equal(t, float64(30), cmdData.Parameters["timeout"])

	// 测试响应数据解析
	respMsg := protocol.NewResponseMessage(
		"test_session",
		protocol.StageASR,
		"测试内容",
		0.95,
		true,
		[]byte("audio data"),
	)

	respData, err := protocol.ParseResponseData(respMsg.Data)
	require.NoError(t, err)
	assert.Equal(t, protocol.StageASR, respData.Stage)
	assert.Equal(t, "测试内容", respData.Content)
	assert.Equal(t, 0.95, respData.Confidence)
	assert.True(t, respData.IsFinal)
	assert.Equal(t, []byte("audio data"), respData.AudioData)

	// 测试状态数据解析
	statusMsg := protocol.NewStatusMessage(
		"test_session",
		protocol.StateConnected,
		protocol.ModeContinuous,
		1,
	)

	statusData, err := protocol.ParseStatusData(statusMsg.Data)
	require.NoError(t, err)
	assert.Equal(t, protocol.StateConnected, statusData.State)
	assert.Equal(t, protocol.ModeContinuous, statusData.Mode)
	assert.Equal(t, 1, statusData.ConcurrentStreams)

	// 测试错误数据解析
	errorMsg := protocol.NewErrorMessage(
		"test_session",
		protocol.ErrConnectionFailed,
		"连接失败",
		true,
	)

	errorData, err := protocol.ParseErrorData(errorMsg.Data)
	require.NoError(t, err)
	assert.Equal(t, protocol.ErrConnectionFailed, errorData.Code)
	assert.Equal(t, "连接失败", errorData.Message)
	assert.True(t, errorData.Recoverable)
}

// TestProtocolValidation 测试消息验证
func TestProtocolValidation(t *testing.T) {
	// 测试有效消息
	validMsg := protocol.NewAudioStreamMessage(
		"test_session",
		"pcm_16khz_16bit",
		1,
		true,
		[]byte("test audio"),
	)

	err := protocol.ValidateMessage(validMsg)
	assert.NoError(t, err)

	// 测试无效消息（空会话ID）
	invalidMsg := protocol.NewAudioStreamMessage(
		"",
		"pcm_16khz_16bit",
		1,
		true,
		[]byte("test audio"),
	)

	err = protocol.ValidateMessage(invalidMsg)
	assert.Error(t, err)
}

// BenchmarkProtocolSerialization 基准测试序列化性能
func BenchmarkProtocolSerialization(b *testing.B) {
	msg := protocol.NewAudioStreamMessage(
		"test_session",
		"pcm_16khz_16bit",
		1,
		true,
		make([]byte, 1024),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := msg.ToJSON()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProtocolDeserialization 基准测试反序列化性能
func BenchmarkProtocolDeserialization(b *testing.B) {
	msg := protocol.NewAudioStreamMessage(
		"test_session",
		"pcm_16khz_16bit",
		1,
		true,
		make([]byte, 1024),
	)

	data, err := msg.ToJSON()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := protocol.FromJSON(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
