package tts

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// EdgeTTS Edge-TTS实现
type EdgeTTS struct {
	config          TTSConfig
	conn            *websocket.Conn
	isInitialized   bool
	mu              sync.RWMutex
	modelInfo       ModelInfo
	supportedVoices []Voice
	currentVoice    string
	requestID       int64
}

// EdgeTTSRequest Edge-TTS请求
type EdgeTTSRequest struct {
	Text     string `json:"text"`
	Voice    string `json:"voice"`
	Rate     string `json:"rate"`
	Volume   string `json:"volume"`
	Pitch    string `json:"pitch"`
	Format   string `json:"format"`
	Language string `json:"language"`
}

// EdgeVoice Edge声音信息
type EdgeVoice struct {
	Name           string   `json:"Name"`
	ShortName      string   `json:"ShortName"`
	Gender         string   `json:"Gender"`
	Locale         string   `json:"Locale"`
	SuggestedCodec string   `json:"SuggestedCodec"`
	FriendlyName   string   `json:"FriendlyName"`
	Status         string   `json:"Status"`
	VoiceTag       VoiceTag `json:"VoiceTag"`
}

// VoiceTag 声音标签
type VoiceTag struct {
	ContentCategories  []string `json:"ContentCategories"`
	VoicePersonalities []string `json:"VoicePersonalities"`
}

// NewEdgeTTS 创建Edge-TTS实例
func NewEdgeTTS(config TTSConfig) (*EdgeTTS, error) {
	e := &EdgeTTS{
		config: config,
	}
	return e, nil
}

// Initialize 初始化Edge-TTS
func (e *EdgeTTS) Initialize(config TTSConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	log.Println("EdgeTTS: 初始化中...")

	// 获取支持的声音列表
	voices := e.getSupportedVoices()
	e.supportedVoices = voices

	// 设置默认声音
	if config.Voice == "" {
		config.Voice = "zh-CN-XiaoxiaoNeural"
	}
	e.currentVoice = config.Voice

	// 设置模型信息
	e.modelInfo = ModelInfo{
		Name:          "Edge-TTS",
		Version:       "1.0.0",
		Type:          "text-to-speech",
		Provider:      "Microsoft",
		Languages:     []string{"zh-CN", "en-US", "ja-JP", "ko-KR"},
		Voices:        voices,
		SampleRates:   []int{16000, 24000, 48000},
		Formats:       []string{"wav", "mp3"},
		MaxTextLength: 1000,
		LoadTime:      time.Now().UnixMilli(),
	}

	e.config = config
	e.isInitialized = true

	log.Println("EdgeTTS: 初始化成功")
	return nil
}

// SynthesizeText 合成文本
func (e *EdgeTTS) SynthesizeText(ctx context.Context, text string) (TTSResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.isInitialized {
		return TTSResult{}, ErrTTSNotInitialized
	}

	startTime := time.Now()

	// 建立WebSocket连接
	if err := e.connect(); err != nil {
		return TTSResult{}, fmt.Errorf("连接Edge-TTS失败: %w", err)
	}
	defer e.disconnect()

	// 发送合成请求
	audioData, err := e.synthesize(ctx, text)
	if err != nil {
		return TTSResult{}, fmt.Errorf("合成失败: %w", err)
	}

	processTime := time.Since(startTime)

	result := TTSResult{
		AudioData:   audioData,
		Format:      "mp3",
		SampleRate:  24000,
		Channels:    1,
		Duration:    int64(len(audioData) * 1000 / (24000 * 1 * 2)),
		Text:        text,
		Voice:       e.currentVoice,
		Language:    e.config.Language,
		IsComplete:  true,
		ProcessTime: processTime.Milliseconds(),
		ModelInfo:   "Edge-TTS",
		Timestamp:   time.Now().UnixMilli(),
	}

	return result, nil
}

// SynthesizeTextStream 流式合成文本
func (e *EdgeTTS) SynthesizeTextStream(ctx context.Context, text string) (<-chan TTSResult, error) {
	resultChan := make(chan TTSResult, 1)

	go func() {
		defer close(resultChan)

		result, err := e.SynthesizeText(ctx, text)
		if err != nil {
			result.Error = err
		}

		resultChan <- result
	}()

	return resultChan, nil
}

// SynthesizeToFile 合成到文件
func (e *EdgeTTS) SynthesizeToFile(ctx context.Context, text string, filePath string) error {
	result, err := e.SynthesizeText(ctx, text)
	if err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return ErrFileWriteFailed
	}
	defer file.Close()

	_, err = file.Write(result.AudioData)
	if err != nil {
		return ErrFileWriteFailed
	}

	return nil
}

// SynthesizeToStream 合成到流
func (e *EdgeTTS) SynthesizeToStream(ctx context.Context, text string, stream io.Writer) error {
	result, err := e.SynthesizeText(ctx, text)
	if err != nil {
		return err
	}

	_, err = stream.Write(result.AudioData)
	if err != nil {
		return ErrStreamWriteFailed
	}

	return nil
}

// GetSupportedVoices 获取支持的声音列表
func (e *EdgeTTS) GetSupportedVoices() []Voice {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.supportedVoices
}

// SetVoice 设置声音
func (e *EdgeTTS) SetVoice(voiceID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, voice := range e.supportedVoices {
		if voice.ID == voiceID {
			e.currentVoice = voiceID
			return nil
		}
	}
	return ErrVoiceNotFound
}

// GetSupportedLanguages 获取支持的语言列表
func (e *EdgeTTS) GetSupportedLanguages() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.modelInfo.Languages
}

// SetLanguage 设置语言
func (e *EdgeTTS) SetLanguage(language string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, lang := range e.modelInfo.Languages {
		if lang == language {
			e.config.Language = language
			return nil
		}
	}
	return ErrLanguageNotSupported
}

// GetModelInfo 获取模型信息
func (e *EdgeTTS) GetModelInfo() ModelInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.modelInfo
}

// Close 关闭TTS服务
func (e *EdgeTTS) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.disconnect()
	e.isInitialized = false

	log.Println("EdgeTTS: 已关闭")
	return nil
}

// connect 建立连接
func (e *EdgeTTS) connect() error {
	wsURL := "wss://speech.platform.bing.com/consumer/speech/synthesize/realtimestreaming/edge/v1"

	params := url.Values{}
	params.Set("TrustedClientToken", "6A5AA1D4EAFF4E9FB37E23D68491D6F4")
	params.Set("ConnectionId", e.generateConnectionID())

	fullURL := wsURL + "?" + params.Encode()

	header := http.Header{}
	header.Set("Origin", "chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold")
	header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	conn, _, err := websocket.DefaultDialer.Dial(fullURL, header)
	if err != nil {
		return err
	}

	e.conn = conn
	return nil
}

// disconnect 断开连接
func (e *EdgeTTS) disconnect() {
	if e.conn != nil {
		e.conn.Close()
		e.conn = nil
	}
}

// synthesize 执行合成
func (e *EdgeTTS) synthesize(ctx context.Context, text string) ([]byte, error) {
	// 发送配置消息
	configMsg := e.buildConfigMessage()
	if err := e.conn.WriteMessage(websocket.TextMessage, []byte(configMsg)); err != nil {
		return nil, err
	}

	// 发送SSML消息
	ssmlMsg := e.buildSSMLMessage(text)
	if err := e.conn.WriteMessage(websocket.TextMessage, []byte(ssmlMsg)); err != nil {
		return nil, err
	}

	// 接收音频数据
	var audioData []byte
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			messageType, data, err := e.conn.ReadMessage()
			if err != nil {
				return nil, err
			}

			if messageType == websocket.BinaryMessage {
				if len(data) > 2 {
					headerLength := int(data[0])<<8 | int(data[1])
					if len(data) > headerLength+2 {
						audioData = append(audioData, data[headerLength+2:]...)
					}
				}
			} else if messageType == websocket.TextMessage {
				if strings.Contains(string(data), "turn.end") {
					break
				}
			}
		}
	}

	return audioData, nil
}

// buildConfigMessage 构建配置消息
func (e *EdgeTTS) buildConfigMessage() string {
	timestamp := time.Now().Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)")

	return fmt.Sprintf(`X-Timestamp:%s
Content-Type:application/json; charset=utf-8
Path:speech.config

{"context":{"synthesis":{"audio":{"metadataoptions":{"sentenceBoundaryEnabled":"false","wordBoundaryEnabled":"true"},"outputFormat":"audio-24khz-48kbitrate-mono-mp3"}}}}`, timestamp)
}

// buildSSMLMessage 构建SSML消息
func (e *EdgeTTS) buildSSMLMessage(text string) string {
	timestamp := time.Now().Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)")
	requestId := e.generateRequestID()

	ssml := fmt.Sprintf(`<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='%s'>
<voice name='%s'>
<prosody rate='%s' pitch='%s' volume='%s'>
%s
</prosody>
</voice>
</speak>`,
		e.getLanguageFromVoice(),
		e.currentVoice,
		e.formatRate(),
		e.formatPitch(),
		e.formatVolume(),
		text)

	return fmt.Sprintf(`X-RequestId:%s
X-Timestamp:%s
Content-Type:application/ssml+xml
Path:ssml

%s`, requestId, timestamp, ssml)
}

// generateConnectionID 生成连接ID
func (e *EdgeTTS) generateConnectionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// generateRequestID 生成请求ID
func (e *EdgeTTS) generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// formatRate 格式化语速
func (e *EdgeTTS) formatRate() string {
	if e.config.Speed == 0 {
		return "+0%"
	}
	return fmt.Sprintf("%+.0f%%", (e.config.Speed-1)*100)
}

// formatPitch 格式化音调
func (e *EdgeTTS) formatPitch() string {
	if e.config.Pitch == 0 {
		return "+0Hz"
	}
	return fmt.Sprintf("%+.0fHz", e.config.Pitch*100)
}

// formatVolume 格式化音量
func (e *EdgeTTS) formatVolume() string {
	if e.config.Volume == 0 {
		return "+0%"
	}
	return fmt.Sprintf("%+.0f%%", (e.config.Volume-1)*100)
}

// getLanguageFromVoice 从声音获取语言
func (e *EdgeTTS) getLanguageFromVoice() string {
	if strings.HasPrefix(e.currentVoice, "zh-CN") {
		return "zh-CN"
	} else if strings.HasPrefix(e.currentVoice, "en-US") {
		return "en-US"
	} else if strings.HasPrefix(e.currentVoice, "ja-JP") {
		return "ja-JP"
	}
	return "zh-CN"
}

// getSupportedVoices 获取支持的声音列表
func (e *EdgeTTS) getSupportedVoices() []Voice {
	return []Voice{
		{
			ID:          "zh-CN-XiaoxiaoNeural",
			Name:        "Xiaoxiao",
			DisplayName: "晓晓",
			Language:    "zh-CN",
			Locale:      "zh-CN",
			Gender:      "female",
			Age:         "adult",
			Quality:     "high",
			Provider:    "Microsoft",
			Description: "中文女声",
		},
		{
			ID:          "zh-CN-YunxiNeural",
			Name:        "Yunxi",
			DisplayName: "云希",
			Language:    "zh-CN",
			Locale:      "zh-CN",
			Gender:      "male",
			Age:         "adult",
			Quality:     "high",
			Provider:    "Microsoft",
			Description: "中文男声",
		},
		{
			ID:          "en-US-AriaNeural",
			Name:        "Aria",
			DisplayName: "Aria",
			Language:    "en-US",
			Locale:      "en-US",
			Gender:      "female",
			Age:         "adult",
			Quality:     "high",
			Provider:    "Microsoft",
			Description: "英文女声",
		},
		{
			ID:          "en-US-GuyNeural",
			Name:        "Guy",
			DisplayName: "Guy",
			Language:    "en-US",
			Locale:      "en-US",
			Gender:      "male",
			Age:         "adult",
			Quality:     "high",
			Provider:    "Microsoft",
			Description: "英文男声",
		},
	}
}

// 注册Edge-TTS
func init() {
	RegisterTTS("edge", func(config TTSConfig) (TTSService, error) {
		return NewEdgeTTS(config)
	})
}
