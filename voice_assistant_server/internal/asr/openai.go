package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"sync"
	"time"
)

// OpenAIASR OpenAI Whisper API实现
type OpenAIASR struct {
	config         ASRConfig
	apiKey         string
	apiURL         string
	client         *http.Client
	isInitialized  bool
	mu             sync.RWMutex
	modelInfo      ModelInfo
	supportedLangs []string
}

// OpenAIResponse OpenAI API响应
type OpenAIResponse struct {
	Text string `json:"text"`
}

// NewOpenAIASR 创建OpenAI ASR实例
func NewOpenAIASR(config ASRConfig) (*OpenAIASR, error) {
	o := &OpenAIASR{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}

	if o.client.Timeout == 0 {
		o.client.Timeout = 30 * time.Second
	}

	return o, nil
}

// Initialize 初始化OpenAI ASR
func (o *OpenAIASR) Initialize(config ASRConfig) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	log.Println("OpenAIASR: 初始化中...")

	// 检查API密钥
	if config.APIKey == "" {
		return fmt.Errorf("OpenAI API密钥不能为空")
	}

	o.apiKey = config.APIKey
	o.apiURL = config.APIUrl
	if o.apiURL == "" {
		o.apiURL = "https://api.openai.com/v1/audio/transcriptions"
	}

	// 设置支持的语言
	o.supportedLangs = []string{
		"zh", "en", "ja", "ko", "fr", "de", "es", "it", "pt", "ru",
		"ar", "hi", "th", "vi", "tr", "pl", "nl", "sv", "da", "no",
		"fi", "hu", "cs", "sk", "bg", "hr", "sl", "et", "lv", "lt",
	}

	// 设置模型信息
	o.modelInfo = ModelInfo{
		Name:       "OpenAI Whisper",
		Version:    "1.0.0",
		Type:       "speech-to-text",
		Languages:  o.supportedLangs,
		SampleRate: config.SampleRate,
		Channels:   config.Channels,
		LoadTime:   time.Now().UnixMilli(),
	}

	o.config = config
	o.isInitialized = true

	log.Println("OpenAIASR: 初始化成功")
	return nil
}

// ProcessAudio 处理音频数据
func (o *OpenAIASR) ProcessAudio(ctx context.Context, audioData []byte) (ASRResult, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.isInitialized {
		return ASRResult{}, ErrASRNotInitialized
	}

	startTime := time.Now()

	// 调用OpenAI API
	text, err := o.callOpenAIAPI(ctx, audioData)
	if err != nil {
		return ASRResult{}, fmt.Errorf("OpenAI API调用失败: %w", err)
	}

	processTime := time.Since(startTime)

	result := ASRResult{
		Text:        text,
		Confidence:  0.9, // OpenAI API通常有较高的准确率
		Language:    o.config.Language,
		IsFinal:     true,
		StartTime:   startTime.UnixMilli(),
		EndTime:     time.Now().UnixMilli(),
		ProcessTime: processTime.Milliseconds(),
		ModelInfo:   "OpenAI Whisper",
	}

	return result, nil
}

// ProcessAudioStream 处理音频流
func (o *OpenAIASR) ProcessAudioStream(ctx context.Context, audioStream io.Reader) (<-chan ASRResult, error) {
	return nil, fmt.Errorf("OpenAI ASR不支持流式处理")
}

// ProcessAudioBytes 处理音频字节流
func (o *OpenAIASR) ProcessAudioBytes(ctx context.Context, audioBytes []byte, isFinal bool) (ASRResult, error) {
	if !isFinal {
		return ASRResult{IsFinal: false}, nil
	}
	return o.ProcessAudio(ctx, audioBytes)
}

// GetSupportedLanguages 获取支持的语言列表
func (o *OpenAIASR) GetSupportedLanguages() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.supportedLangs
}

// SetLanguage 设置识别语言
func (o *OpenAIASR) SetLanguage(language string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 检查语言是否支持
	for _, lang := range o.supportedLangs {
		if lang == language {
			o.config.Language = language
			return nil
		}
	}
	return ErrLanguageNotSupported
}

// GetModelInfo 获取模型信息
func (o *OpenAIASR) GetModelInfo() ModelInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.modelInfo
}

// Close 关闭ASR服务
func (o *OpenAIASR) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.isInitialized = false
	log.Println("OpenAIASR: 已关闭")
	return nil
}

// callOpenAIAPI 调用OpenAI API
func (o *OpenAIASR) callOpenAIAPI(ctx context.Context, audioData []byte) (string, error) {
	// 创建multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// 添加音频文件
	audioWriter, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", err
	}

	// 转换音频数据为WAV格式
	wavData, err := o.convertToWAV(audioData)
	if err != nil {
		return "", err
	}

	if _, err := audioWriter.Write(wavData); err != nil {
		return "", err
	}

	// 添加模型参数
	if err := writer.WriteField("model", "whisper-1"); err != nil {
		return "", err
	}

	// 添加语言参数
	if o.config.Language != "" {
		if err := writer.WriteField("language", o.config.Language); err != nil {
			return "", err
		}
	}

	// 添加响应格式
	if err := writer.WriteField("response_format", "json"); err != nil {
		return "", err
	}

	writer.Close()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", o.apiURL, &body)
	if err != nil {
		return "", err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	resp, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API请求失败: %d, %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析响应
	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	return response.Text, nil
}

// convertToWAV 将音频数据转换为WAV格式
func (o *OpenAIASR) convertToWAV(audioData []byte) ([]byte, error) {
	// 简单的WAV头生成
	sampleRate := o.config.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}

	channels := o.config.Channels
	if channels == 0 {
		channels = 1
	}

	// WAV文件头
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")

	dataSize := len(audioData)
	fileSize := dataSize + 36

	// 文件大小
	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)

	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")

	// fmt chunk size (16)
	header[16] = 16
	header[17] = 0
	header[18] = 0
	header[19] = 0

	// PCM format (1)
	header[20] = 1
	header[21] = 0

	// Channels
	header[22] = byte(channels)
	header[23] = byte(channels >> 8)

	// Sample rate
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)

	// Byte rate
	byteRate := sampleRate * channels * 2
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)

	// Block align
	blockAlign := channels * 2
	header[32] = byte(blockAlign)
	header[33] = byte(blockAlign >> 8)

	// Bits per sample (16)
	header[34] = 16
	header[35] = 0

	copy(header[36:40], "data")

	// Data size
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	// 组合头部和数据
	wavData := make([]byte, len(header)+len(audioData))
	copy(wavData, header)
	copy(wavData[len(header):], audioData)

	return wavData, nil
}

// 注册OpenAI ASR
func init() {
	RegisterASR("openai", func(config ASRConfig) (ASRService, error) {
		return NewOpenAIASR(config)
	})
}
