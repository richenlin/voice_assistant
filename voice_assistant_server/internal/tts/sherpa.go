package tts

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SherpaTTS Sherpa-ONNX TTS实现
type SherpaTTS struct {
	config TTSConfig

	// 状态
	isInitialized bool

	// 统计信息
	totalRequests   int64
	totalCharacters int64
	totalDuration   float64
}

// NewSherpaTTS 创建Sherpa-ONNX TTS实例
func NewSherpaTTS(config TTSConfig) *SherpaTTS {
	return &SherpaTTS{
		config: config,
	}
}

// Initialize 初始化TTS引擎
func (s *SherpaTTS) Initialize(config TTSConfig) error {
	log.Println("初始化Sherpa-ONNX TTS引擎...")

	s.config = config

	// 检查模型文件
	if err := s.validateModelFiles(); err != nil {
		return fmt.Errorf("模型文件验证失败: %w", err)
	}

	s.isInitialized = true
	log.Printf("Sherpa-ONNX TTS引擎初始化成功")

	return nil
}

// SynthesizeText 合成语音
func (s *SherpaTTS) SynthesizeText(ctx context.Context, text string) (TTSResult, error) {
	if !s.isInitialized {
		return TTSResult{}, fmt.Errorf("TTS引擎未初始化")
	}

	if text == "" {
		return TTSResult{}, fmt.Errorf("文本不能为空")
	}

	log.Printf("Sherpa-ONNX合成语音: %s", text)

	// 构建命令行参数
	args := s.buildCommandArgs(text)

	// 执行TTS命令
	cmd := exec.CommandContext(ctx, "sherpa-onnx-offline-tts", args...)

	// 设置工作目录
	if s.config.SherpaConfig.DataDir != "" {
		cmd.Dir = s.config.SherpaConfig.DataDir
	}

	// 执行命令并获取输出
	output, err := cmd.Output()
	if err != nil {
		return TTSResult{}, fmt.Errorf("执行TTS命令失败: %w", err)
	}

	// 解析输出（假设输出是WAV文件路径）
	outputPath := strings.TrimSpace(string(output))

	// 读取音频文件
	audioData, err := s.readAudioFile(outputPath)
	if err != nil {
		return TTSResult{}, fmt.Errorf("读取音频文件失败: %w", err)
	}

	// 更新统计信息
	s.updateStats(text, len(audioData))

	result := TTSResult{
		AudioData:   audioData,
		SampleRate:  s.config.SampleRate,
		Format:      s.config.Format,
		Duration:    int64(len(audioData)) / int64(s.config.SampleRate) / 2 * 1000, // 毫秒
		Text:        text,
		Voice:       s.config.Voice,
		Language:    s.config.Language,
		IsComplete:  true,
		ProcessTime: time.Now().UnixNano() / 1000000, // 毫秒
		Timestamp:   time.Now().UnixNano() / 1000000,
	}

	return result, nil
}

// SynthesizeTextStream 流式合成语音
func (s *SherpaTTS) SynthesizeTextStream(ctx context.Context, text string) (<-chan TTSResult, error) {
	resultChan := make(chan TTSResult, 1)

	go func() {
		defer close(resultChan)

		result, err := s.SynthesizeText(ctx, text)
		if err != nil {
			result.Error = err
		}

		resultChan <- result
	}()

	return resultChan, nil
}

// SynthesizeToFile 合成到文件
func (s *SherpaTTS) SynthesizeToFile(ctx context.Context, text string, filePath string) error {
	result, err := s.SynthesizeText(ctx, text)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, result.AudioData, 0644)
}

// SynthesizeToStream 合成到流
func (s *SherpaTTS) SynthesizeToStream(ctx context.Context, text string, stream io.Writer) error {
	result, err := s.SynthesizeText(ctx, text)
	if err != nil {
		return err
	}

	_, err = stream.Write(result.AudioData)
	return err
}

// GetSupportedVoices 获取可用声音列表
func (s *SherpaTTS) GetSupportedVoices() []Voice {
	// Sherpa-ONNX通常通过speaker_id来区分不同的声音
	voices := []Voice{
		{
			ID:       "0",
			Name:     "Default Voice",
			Language: "zh-CN",
			Gender:   "unknown",
		},
	}

	// 如果配置了多个speaker，可以添加更多声音
	for i := 1; i <= 10; i++ { // 假设最多支持10个speaker
		voices = append(voices, Voice{
			ID:       strconv.Itoa(i),
			Name:     fmt.Sprintf("Speaker %d", i),
			Language: "zh-CN",
			Gender:   "unknown",
		})
	}

	return voices
}

// SetVoice 设置声音
func (s *SherpaTTS) SetVoice(voiceID string) error {
	s.config.Voice = voiceID
	return nil
}

// GetSupportedLanguages 获取支持的语言列表
func (s *SherpaTTS) GetSupportedLanguages() []string {
	return []string{"zh-CN", "en-US"}
}

// SetLanguage 设置语言
func (s *SherpaTTS) SetLanguage(language string) error {
	s.config.Language = language
	return nil
}

// GetModelInfo 获取模型信息
func (s *SherpaTTS) GetModelInfo() ModelInfo {
	return ModelInfo{
		Name:      "Sherpa-ONNX",
		Version:   "1.0.0",
		Type:      "neural",
		Provider:  "sherpa",
		Languages: []string{"zh-CN", "en-US"},
		Voices:    s.GetSupportedVoices(),
	}
}

// Close 关闭TTS引擎
func (s *SherpaTTS) Close() error {
	s.isInitialized = false
	log.Println("Sherpa-ONNX TTS引擎已关闭")
	return nil
}

// validateModelFiles 验证模型文件
func (s *SherpaTTS) validateModelFiles() error {
	requiredFiles := []string{
		s.config.SherpaConfig.ModelPath,
		s.config.SherpaConfig.LexiconPath,
		s.config.SherpaConfig.TokensPath,
	}

	for _, file := range requiredFiles {
		if file == "" {
			continue
		}

		// 如果是相对路径，与DataDir组合
		if s.config.SherpaConfig.DataDir != "" && !filepath.IsAbs(file) {
			file = filepath.Join(s.config.SherpaConfig.DataDir, file)
		}

		if !fileExists(file) {
			return fmt.Errorf("模型文件不存在: %s", file)
		}
	}

	return nil
}

// buildCommandArgs 构建命令行参数
func (s *SherpaTTS) buildCommandArgs(text string) []string {
	args := []string{
		"--model", s.config.SherpaConfig.ModelPath,
		"--lexicon", s.config.SherpaConfig.LexiconPath,
		"--tokens", s.config.SherpaConfig.TokensPath,
		"--text", text,
		"--speed", fmt.Sprintf("%.2f", s.config.Speed),
		"--num-threads", strconv.Itoa(s.config.SherpaConfig.NumThreads),
	}

	if s.config.SherpaConfig.DataDir != "" {
		args = append(args, "--data-dir", s.config.SherpaConfig.DataDir)
	}

	return args
}

// readAudioFile 读取音频文件
func (s *SherpaTTS) readAudioFile(filePath string) ([]byte, error) {
	return ioutil.ReadFile(filePath)
}

// updateStats 更新统计信息
func (s *SherpaTTS) updateStats(text string, audioSize int) {
	s.totalRequests++
	s.totalCharacters += int64(len(text))
	s.totalDuration += float64(audioSize) / float64(s.config.SampleRate) / 2
}

// fileExists 检查文件是否存在
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// 注册Sherpa-ONNX TTS
func init() {
	RegisterTTS("sherpa", func(config TTSConfig) (TTSService, error) {
		return NewSherpaTTS(config), nil
	})
}
