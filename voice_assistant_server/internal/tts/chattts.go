package tts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ChatTTSConfig ChatTTS特定配置
type ChatTTSConfig struct {
	ModelPath   string  `yaml:"model_path"`  // 模型路径
	Device      string  `yaml:"device"`      // cpu|cuda
	Temperature float32 `yaml:"temperature"` // 温度参数
	TopP        float32 `yaml:"top_p"`       // Top-p参数
	TopK        int     `yaml:"top_k"`       // Top-k参数
	SpeakerID   int     `yaml:"speaker_id"`  // 说话人ID
	NumThreads  int     `yaml:"num_threads"` // 线程数
}

// ChatTTS ChatTTS实现
type ChatTTS struct {
	config TTSConfig

	// 状态
	isInitialized bool

	// 统计信息
	totalRequests   int64
	totalCharacters int64
	totalDuration   float64
}

// NewChatTTS 创建ChatTTS实例
func NewChatTTS(config TTSConfig) *ChatTTS {
	return &ChatTTS{
		config: config,
	}
}

// Initialize 初始化TTS引擎
func (c *ChatTTS) Initialize(config TTSConfig) error {
	log.Println("初始化ChatTTS引擎...")

	c.config = config

	// 检查ChatTTS环境
	if err := c.checkEnvironment(); err != nil {
		return fmt.Errorf("ChatTTS环境检查失败: %w", err)
	}

	// 检查模型文件
	if err := c.validateModelFiles(); err != nil {
		return fmt.Errorf("模型文件验证失败: %w", err)
	}

	c.isInitialized = true
	log.Printf("ChatTTS引擎初始化成功")

	return nil
}

// SynthesizeText 合成语音
func (c *ChatTTS) SynthesizeText(ctx context.Context, text string) (TTSResult, error) {
	if !c.isInitialized {
		return TTSResult{}, fmt.Errorf("ChatTTS引擎未初始化")
	}

	if text == "" {
		return TTSResult{}, fmt.Errorf("文本不能为空")
	}

	log.Printf("ChatTTS合成语音: %s", text)

	startTime := time.Now()

	// 执行ChatTTS合成
	audioData, err := c.runChatTTS(ctx, text)
	if err != nil {
		return TTSResult{}, fmt.Errorf("ChatTTS合成失败: %w", err)
	}

	// 更新统计信息
	c.updateStats(text, len(audioData))

	result := TTSResult{
		AudioData:   audioData,
		SampleRate:  c.config.SampleRate,
		Format:      c.config.Format,
		Duration:    int64(len(audioData)) / int64(c.config.SampleRate) / 2 * 1000, // 毫秒
		Text:        text,
		Voice:       c.config.Voice,
		Language:    c.config.Language,
		IsComplete:  true,
		ProcessTime: time.Since(startTime).Nanoseconds() / 1000000, // 毫秒
		Timestamp:   time.Now().UnixNano() / 1000000,
	}

	return result, nil
}

// SynthesizeTextStream 流式合成语音
func (c *ChatTTS) SynthesizeTextStream(ctx context.Context, text string) (<-chan TTSResult, error) {
	resultChan := make(chan TTSResult, 1)

	go func() {
		defer close(resultChan)

		result, err := c.SynthesizeText(ctx, text)
		if err != nil {
			result.Error = err
		}

		resultChan <- result
	}()

	return resultChan, nil
}

// SynthesizeToFile 合成到文件
func (c *ChatTTS) SynthesizeToFile(ctx context.Context, text string, filePath string) error {
	result, err := c.SynthesizeText(ctx, text)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, result.AudioData, 0644)
}

// SynthesizeToStream 合成到流
func (c *ChatTTS) SynthesizeToStream(ctx context.Context, text string, stream io.Writer) error {
	result, err := c.SynthesizeText(ctx, text)
	if err != nil {
		return err
	}

	_, err = stream.Write(result.AudioData)
	return err
}

// GetSupportedVoices 获取可用声音列表
func (c *ChatTTS) GetSupportedVoices() []Voice {
	voices := []Voice{
		{
			ID:       "0",
			Name:     "ChatTTS Speaker 0",
			Language: "zh-CN",
			Gender:   "unknown",
		},
	}

	// ChatTTS支持多个说话人
	for i := 1; i <= 10; i++ {
		voices = append(voices, Voice{
			ID:       fmt.Sprintf("%d", i),
			Name:     fmt.Sprintf("ChatTTS Speaker %d", i),
			Language: "zh-CN",
			Gender:   "unknown",
		})
	}

	return voices
}

// SetVoice 设置声音
func (c *ChatTTS) SetVoice(voiceID string) error {
	c.config.Voice = voiceID
	return nil
}

// GetSupportedLanguages 获取支持的语言列表
func (c *ChatTTS) GetSupportedLanguages() []string {
	return []string{"zh-CN", "en-US"}
}

// SetLanguage 设置语言
func (c *ChatTTS) SetLanguage(language string) error {
	c.config.Language = language
	return nil
}

// GetModelInfo 获取模型信息
func (c *ChatTTS) GetModelInfo() ModelInfo {
	return ModelInfo{
		Name:      "ChatTTS",
		Version:   "1.0.0",
		Type:      "neural",
		Provider:  "chattts",
		Languages: []string{"zh-CN", "en-US"},
		Voices:    c.GetSupportedVoices(),
	}
}

// Close 关闭TTS引擎
func (c *ChatTTS) Close() error {
	c.isInitialized = false
	log.Println("ChatTTS引擎已关闭")
	return nil
}

// checkEnvironment 检查ChatTTS环境
func (c *ChatTTS) checkEnvironment() error {
	// 检查Python环境
	_, err := exec.LookPath("python")
	if err != nil {
		return fmt.Errorf("未找到Python环境")
	}

	// 检查ChatTTS是否安装
	cmd := exec.Command("python", "-c", "import ChatTTS")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ChatTTS未安装，请参考官方文档安装")
	}

	return nil
}

// validateModelFiles 验证模型文件
func (c *ChatTTS) validateModelFiles() error {
	// ChatTTS会自动下载模型，这里主要检查配置
	return nil
}

// runChatTTS 执行ChatTTS合成
func (c *ChatTTS) runChatTTS(ctx context.Context, text string) ([]byte, error) {
	// 构建Python脚本
	script := c.buildPythonScript(text)

	// 创建临时脚本文件
	scriptFile, err := c.createTempScript(script)
	if err != nil {
		return nil, fmt.Errorf("创建脚本文件失败: %w", err)
	}
	defer os.Remove(scriptFile)

	// 执行Python脚本
	cmd := exec.CommandContext(ctx, "python", scriptFile)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行ChatTTS脚本失败: %w", err)
	}

	// 解析结果
	audioData, err := c.parseResult(string(output))
	if err != nil {
		return nil, fmt.Errorf("解析合成结果失败: %w", err)
	}

	return audioData, nil
}

// buildPythonScript 构建Python脚本
func (c *ChatTTS) buildPythonScript(text string) string {
	return fmt.Sprintf(`
import json
import sys
import torch
import torchaudio
import ChatTTS
import tempfile
import base64
import os

try:
    # 初始化ChatTTS
    chat = ChatTTS.Chat()
    chat.load_models(compile=False)
    
    # 设置说话人
    spk = chat.sample_random_speaker()
    
    # 合成语音
    texts = ["%s"]
    wavs = chat.infer(texts, spk_emb=spk, temperature=0.3, top_P=0.7, top_K=20)
    
    # 保存到临时文件
    temp_file = tempfile.NamedTemporaryFile(suffix='.wav', delete=False)
    torchaudio.save(temp_file.name, torch.from_numpy(wavs[0]), %d)
    
    # 读取音频数据并编码为base64
    with open(temp_file.name, 'rb') as f:
        audio_data = f.read()
        audio_base64 = base64.b64encode(audio_data).decode('utf-8')
    
    # 清理临时文件
    os.unlink(temp_file.name)
    
    # 输出结果
    result = {
        "success": True,
        "audio_data": audio_base64,
        "sample_rate": %d,
        "format": "wav"
    }
    print(json.dumps(result))

except Exception as e:
    error_result = {
        "success": False,
        "error": str(e)
    }
    print(json.dumps(error_result))
`,
		strings.ReplaceAll(text, `"`, `\"`), // 转义引号
		c.config.SampleRate,
		c.config.SampleRate,
	)
}

// createTempScript 创建临时脚本文件
func (c *ChatTTS) createTempScript(script string) (string, error) {
	tempDir := os.TempDir()
	scriptFile := filepath.Join(tempDir, fmt.Sprintf("chattts_script_%d.py", time.Now().UnixNano()))

	err := ioutil.WriteFile(scriptFile, []byte(script), 0644)
	if err != nil {
		return "", err
	}

	return scriptFile, nil
}

// parseResult 解析合成结果
func (c *ChatTTS) parseResult(output string) ([]byte, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("ChatTTS输出为空")
	}

	var result struct {
		Success   bool   `json:"success"`
		AudioData string `json:"audio_data"`
		Error     string `json:"error"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("ChatTTS错误: %s", result.Error)
	}

	// 解码base64音频数据
	audioData, err := base64.StdEncoding.DecodeString(result.AudioData)
	if err != nil {
		return nil, fmt.Errorf("音频数据解码失败: %w", err)
	}

	return audioData, nil
}

// updateStats 更新统计信息
func (c *ChatTTS) updateStats(text string, audioSize int) {
	c.totalRequests++
	c.totalCharacters += int64(len(text))
	c.totalDuration += float64(audioSize) / float64(c.config.SampleRate) / 2
}

// 注册ChatTTS
func init() {
	RegisterTTS("chattts", func(config TTSConfig) (TTSService, error) {
		return NewChatTTS(config), nil
	})
}
