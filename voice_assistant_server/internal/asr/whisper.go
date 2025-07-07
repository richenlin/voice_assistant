package asr

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WhisperASR Whisper ASR实现
type WhisperASR struct {
	config         ASRConfig
	modelPath      string
	language       string
	tempDir        string
	isInitialized  bool
	mu             sync.RWMutex
	processTimeout time.Duration
	modelInfo      ModelInfo
	supportedLangs []string
}

// NewWhisperASR 创建Whisper ASR实例
func NewWhisperASR(config ASRConfig) (*WhisperASR, error) {
	w := &WhisperASR{
		config:         config,
		processTimeout: time.Duration(config.Timeout) * time.Second,
	}

	if w.processTimeout == 0 {
		w.processTimeout = 30 * time.Second
	}

	return w, nil
}

// Initialize 初始化Whisper ASR
func (w *WhisperASR) Initialize(config ASRConfig) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	log.Println("WhisperASR: 初始化中...")

	// 检查whisper-cpp命令是否可用
	if err := w.checkWhisperInstallation(); err != nil {
		return fmt.Errorf("whisper检查失败: %v", err)
	}

	// 设置模型路径
	modelPath := config.ModelPath
	if !filepath.IsAbs(modelPath) {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("获取当前工作目录失败: %w", err)
		}
		modelPath = filepath.Join(currentDir, modelPath)
	}

	// 检查模型文件是否存在
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("未找到whisper模型文件: %s", modelPath)
	}

	w.modelPath = modelPath

	// 设置语言
	w.language = config.Language
	if w.language == "" {
		w.language = "zh"
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "whisper-asr-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	w.tempDir = tempDir

	// 设置支持的语言
	w.supportedLangs = []string{
		"zh", "en", "ja", "ko", "fr", "de", "es", "it", "pt", "ru",
		"ar", "hi", "th", "vi", "tr", "pl", "nl", "sv", "da", "no",
	}

	// 设置模型信息
	w.modelInfo = ModelInfo{
		Name:       "Whisper",
		Version:    "1.0.0",
		Type:       "speech-to-text",
		Languages:  w.supportedLangs,
		SampleRate: config.SampleRate,
		Channels:   config.Channels,
		LoadTime:   time.Now().UnixMilli(),
	}

	w.config = config
	w.isInitialized = true

	log.Println("WhisperASR: 初始化成功")
	return nil
}

// ProcessAudio 处理音频数据
func (w *WhisperASR) ProcessAudio(ctx context.Context, audioData []byte) (ASRResult, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.isInitialized {
		return ASRResult{}, ErrASRNotInitialized
	}

	startTime := time.Now()

	// 将音频数据转换为float32
	audioFloat, err := w.bytesToFloat32(audioData)
	if err != nil {
		return ASRResult{}, fmt.Errorf("音频数据转换失败: %w", err)
	}

	// 创建临时WAV文件
	wavFile, err := w.createTempWavFile(audioFloat)
	if err != nil {
		return ASRResult{}, fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(wavFile)

	// 运行Whisper识别
	text, err := w.runWhisperCommand(ctx, wavFile)
	if err != nil {
		return ASRResult{}, fmt.Errorf("Whisper识别失败: %w", err)
	}

	processTime := time.Since(startTime)

	result := ASRResult{
		Text:        strings.TrimSpace(text),
		Confidence:  0.8, // Whisper不提供置信度，使用默认值
		Language:    w.language,
		IsFinal:     true,
		StartTime:   startTime.UnixMilli(),
		EndTime:     time.Now().UnixMilli(),
		ProcessTime: processTime.Milliseconds(),
		ModelInfo:   "Whisper",
	}

	return result, nil
}

// ProcessAudioStream 处理音频流
func (w *WhisperASR) ProcessAudioStream(ctx context.Context, audioStream io.Reader) (<-chan ASRResult, error) {
	return nil, fmt.Errorf("Whisper ASR不支持流式处理")
}

// ProcessAudioBytes 处理音频字节流
func (w *WhisperASR) ProcessAudioBytes(ctx context.Context, audioBytes []byte, isFinal bool) (ASRResult, error) {
	if !isFinal {
		return ASRResult{IsFinal: false}, nil
	}
	return w.ProcessAudio(ctx, audioBytes)
}

// GetSupportedLanguages 获取支持的语言列表
func (w *WhisperASR) GetSupportedLanguages() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.supportedLangs
}

// SetLanguage 设置识别语言
func (w *WhisperASR) SetLanguage(language string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查语言是否支持
	for _, lang := range w.supportedLangs {
		if lang == language {
			w.language = language
			return nil
		}
	}
	return ErrLanguageNotSupported
}

// GetModelInfo 获取模型信息
func (w *WhisperASR) GetModelInfo() ModelInfo {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.modelInfo
}

// Close 关闭ASR服务
func (w *WhisperASR) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.tempDir != "" {
		os.RemoveAll(w.tempDir)
	}

	w.isInitialized = false
	log.Println("WhisperASR: 已关闭")
	return nil
}

// checkWhisperInstallation 检查whisper-cpp是否安装
func (w *WhisperASR) checkWhisperInstallation() error {
	cmd := exec.Command("whisper-cli", "--help")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("whisper-cpp未安装或不在PATH中: %v", err)
	}
	return nil
}

// bytesToFloat32 将字节数组转换为float32数组
func (w *WhisperASR) bytesToFloat32(data []byte) ([]float32, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("音频数据长度必须是偶数")
	}

	samples := make([]float32, len(data)/2)
	for i := 0; i < len(samples); i++ {
		// 16位PCM转float32
		sample := int16(data[i*2]) | int16(data[i*2+1])<<8
		samples[i] = float32(sample) / 32768.0
	}
	return samples, nil
}

// createTempWavFile 创建临时WAV文件
func (w *WhisperASR) createTempWavFile(audioData []float32) (string, error) {
	wavFile := filepath.Join(w.tempDir, fmt.Sprintf("audio_%d.wav", time.Now().UnixNano()))

	file, err := os.Create(wavFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 写入WAV头
	sampleRate := w.config.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}

	channels := w.config.Channels
	if channels == 0 {
		channels = 1
	}

	if err := w.writeWAVHeader(file, len(audioData), sampleRate, channels); err != nil {
		return "", err
	}

	// 写入音频数据
	for _, sample := range audioData {
		// 转换为16位PCM
		pcmSample := int16(sample * 32767)
		if err := w.writeInt16(file, pcmSample); err != nil {
			return "", err
		}
	}

	return wavFile, nil
}

// writeWAVHeader 写入WAV文件头
func (w *WhisperASR) writeWAVHeader(file *os.File, numSamples, sampleRate, channels int) error {
	bitsPerSample := 16
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := numSamples * bitsPerSample / 8

	// RIFF头
	file.WriteString("RIFF")
	w.writeInt32(file, int32(36+dataSize))
	file.WriteString("WAVE")

	// fmt子块
	file.WriteString("fmt ")
	w.writeInt32(file, 16) // PCM格式大小
	w.writeInt16(file, 1)  // PCM格式
	w.writeInt16(file, int16(channels))
	w.writeInt32(file, int32(sampleRate))
	w.writeInt32(file, int32(byteRate))
	w.writeInt16(file, int16(blockAlign))
	w.writeInt16(file, int16(bitsPerSample))

	// data子块
	file.WriteString("data")
	w.writeInt32(file, int32(dataSize))

	return nil
}

// writeInt16 写入16位整数
func (w *WhisperASR) writeInt16(file *os.File, value int16) error {
	return w.writeBytes(file, []byte{byte(value), byte(value >> 8)})
}

// writeInt32 写入32位整数
func (w *WhisperASR) writeInt32(file *os.File, value int32) error {
	return w.writeBytes(file, []byte{
		byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24),
	})
}

// writeBytes 写入字节
func (w *WhisperASR) writeBytes(file *os.File, data []byte) error {
	_, err := file.Write(data)
	return err
}

// runWhisperCommand 运行Whisper命令
func (w *WhisperASR) runWhisperCommand(ctx context.Context, wavFile string) (string, error) {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, w.processTimeout)
	defer cancel()

	args := []string{
		"-m", w.modelPath,
		"-f", wavFile,
		"-l", w.language,
		"--output-txt",
		"--no-timestamps",
	}

	// 应用Whisper特定配置
	if w.config.WhisperConfig.BeamSize > 0 {
		args = append(args, "--beam-size", fmt.Sprintf("%d", w.config.WhisperConfig.BeamSize))
	}

	if w.config.WhisperConfig.Temperature > 0 {
		args = append(args, "--temperature", fmt.Sprintf("%.2f", w.config.WhisperConfig.Temperature))
	}

	cmd := exec.CommandContext(ctx, "whisper-cli", args...)
	cmd.Dir = w.tempDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("whisper命令执行失败: %v, 输出: %s", err, string(output))
	}

	// 读取输出文件
	outputFile := strings.TrimSuffix(wavFile, ".wav") + ".txt"
	textBytes, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("读取输出文件失败: %v", err)
	}

	// 清理输出文件
	os.Remove(outputFile)

	return string(textBytes), nil
}

// 注册Whisper ASR
func init() {
	RegisterASR("whisper", func(config ASRConfig) (ASRService, error) {
		return NewWhisperASR(config)
	})
}
