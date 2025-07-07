package asr

import (
	"context"
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

// FunASR FunASR实现
type FunASR struct {
	config ASRConfig

	// 状态
	isInitialized bool

	// 统计信息
	totalRequests  int64
	totalDuration  float64
	totalAudioTime float64
}

// NewFunASR 创建FunASR实例
func NewFunASR(config ASRConfig) *FunASR {
	return &FunASR{
		config: config,
	}
}

// Initialize 初始化ASR服务
func (f *FunASR) Initialize(config ASRConfig) error {
	log.Println("初始化FunASR服务...")

	f.config = config

	// 检查FunASR环境
	if err := f.checkEnvironment(); err != nil {
		return fmt.Errorf("FunASR环境检查失败: %w", err)
	}

	// 检查模型文件
	if err := f.validateModelFiles(); err != nil {
		return fmt.Errorf("模型文件验证失败: %w", err)
	}

	f.isInitialized = true
	log.Printf("FunASR服务初始化成功")

	return nil
}

// ProcessAudio 处理音频数据（批量处理）
func (f *FunASR) ProcessAudio(ctx context.Context, audioData []byte) (ASRResult, error) {
	if !f.isInitialized {
		return ASRResult{}, fmt.Errorf("FunASR服务未初始化")
	}

	if len(audioData) == 0 {
		return ASRResult{}, fmt.Errorf("音频数据为空")
	}

	startTime := time.Now()

	// 保存音频到临时文件
	tempFile, err := f.saveAudioToTemp(audioData)
	if err != nil {
		return ASRResult{}, fmt.Errorf("保存音频文件失败: %w", err)
	}
	defer os.Remove(tempFile)

	// 执行FunASR识别
	result, err := f.runFunASR(ctx, tempFile)
	if err != nil {
		return ASRResult{}, fmt.Errorf("FunASR识别失败: %w", err)
	}

	// 更新统计信息
	f.updateStats(time.Since(startTime), len(audioData))

	return result, nil
}

// ProcessAudioStream 处理音频流（流式处理）
func (f *FunASR) ProcessAudioStream(ctx context.Context, audioStream io.Reader) (<-chan ASRResult, error) {
	resultChan := make(chan ASRResult, 1)

	go func() {
		defer close(resultChan)

		// 读取完整音频流
		audioData, err := ioutil.ReadAll(audioStream)
		if err != nil {
			resultChan <- ASRResult{Error: fmt.Errorf("读取音频流失败: %w", err)}
			return
		}

		// 处理音频
		result, err := f.ProcessAudio(ctx, audioData)
		if err != nil {
			result.Error = err
		}

		resultChan <- result
	}()

	return resultChan, nil
}

// ProcessAudioBytes 处理音频字节流（实时处理）
func (f *FunASR) ProcessAudioBytes(ctx context.Context, audioBytes []byte, isFinal bool) (ASRResult, error) {
	// FunASR暂不支持真正的流式处理，当isFinal为true时才处理
	if !isFinal {
		return ASRResult{
			Text:      "",
			IsFinal:   false,
			StartTime: time.Now().UnixNano() / 1000000,
		}, nil
	}

	return f.ProcessAudio(ctx, audioBytes)
}

// GetSupportedLanguages 获取支持的语言列表
func (f *FunASR) GetSupportedLanguages() []string {
	return []string{"zh", "en", "zh-cn", "en-us"}
}

// SetLanguage 设置识别语言
func (f *FunASR) SetLanguage(language string) error {
	f.config.Language = language
	return nil
}

// Close 关闭ASR服务
func (f *FunASR) Close() error {
	f.isInitialized = false
	log.Println("FunASR服务已关闭")
	return nil
}

// GetModelInfo 获取模型信息
func (f *FunASR) GetModelInfo() ModelInfo {
	return ModelInfo{
		Name:      "FunASR",
		Version:   "1.0.0",
		Type:      "transformer",
		Languages: f.GetSupportedLanguages(),
		LoadTime:  0, // TODO: 实际加载时间
	}
}

// checkEnvironment 检查FunASR环境
func (f *FunASR) checkEnvironment() error {
	// 检查Python环境
	_, err := exec.LookPath("python")
	if err != nil {
		return fmt.Errorf("未找到Python环境")
	}

	// 检查FunASR是否安装
	cmd := exec.Command("python", "-c", "import funasr")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FunASR未安装，请运行: pip install funasr")
	}

	return nil
}

// validateModelFiles 验证模型文件
func (f *FunASR) validateModelFiles() error {
	if f.config.FunASRConfig.ModelDir == "" {
		return fmt.Errorf("模型目录未配置")
	}

	// 检查模型目录是否存在
	if _, err := os.Stat(f.config.FunASRConfig.ModelDir); os.IsNotExist(err) {
		return fmt.Errorf("模型目录不存在: %s", f.config.FunASRConfig.ModelDir)
	}

	return nil
}

// saveAudioToTemp 保存音频到临时文件
func (f *FunASR) saveAudioToTemp(audioData []byte) (string, error) {
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("funasr_audio_%d.wav", time.Now().UnixNano()))

	err := ioutil.WriteFile(tempFile, audioData, 0644)
	if err != nil {
		return "", err
	}

	return tempFile, nil
}

// runFunASR 执行FunASR识别
func (f *FunASR) runFunASR(ctx context.Context, audioFile string) (ASRResult, error) {
	// 构建Python脚本
	script := f.buildPythonScript(audioFile)

	// 创建临时脚本文件
	scriptFile, err := f.createTempScript(script)
	if err != nil {
		return ASRResult{}, fmt.Errorf("创建脚本文件失败: %w", err)
	}
	defer os.Remove(scriptFile)

	// 执行Python脚本
	cmd := exec.CommandContext(ctx, "python", scriptFile)
	output, err := cmd.Output()
	if err != nil {
		return ASRResult{}, fmt.Errorf("执行FunASR脚本失败: %w", err)
	}

	// 解析结果
	result, err := f.parseResult(string(output))
	if err != nil {
		return ASRResult{}, fmt.Errorf("解析识别结果失败: %w", err)
	}

	return result, nil
}

// buildPythonScript 构建Python脚本
func (f *FunASR) buildPythonScript(audioFile string) string {
	return fmt.Sprintf(`
import json
import sys
from funasr import AutoModel

try:
    # 初始化模型
    model = AutoModel(
        model="%s",
        model_revision="%s", 
        device_id="%s",
        ncpu=%d
    )
    
    # 识别音频
    result = model.generate(input="%s")
    
    # 输出结果
    if result and len(result) > 0:
        text = result[0].get("text", "")
        confidence = 1.0  # FunASR暂不提供置信度
        output = {
            "text": text,
            "confidence": confidence,
            "language": "%s",
            "is_final": True
        }
        print(json.dumps(output, ensure_ascii=False))
    else:
        print(json.dumps({"text": "", "confidence": 0.0, "language": "%s", "is_final": True}, ensure_ascii=False))

except Exception as e:
    error_output = {
        "text": "",
        "confidence": 0.0,
        "language": "%s",
        "is_final": True,
        "error": str(e)
    }
    print(json.dumps(error_output, ensure_ascii=False))
`,
		f.config.FunASRConfig.ModelDir,
		f.config.FunASRConfig.ModelRevision,
		f.config.FunASRConfig.DeviceID,
		f.config.FunASRConfig.IntraOpNumThreads,
		audioFile,
		f.config.Language,
		f.config.Language,
		f.config.Language,
	)
}

// createTempScript 创建临时脚本文件
func (f *FunASR) createTempScript(script string) (string, error) {
	tempDir := os.TempDir()
	scriptFile := filepath.Join(tempDir, fmt.Sprintf("funasr_script_%d.py", time.Now().UnixNano()))

	err := ioutil.WriteFile(scriptFile, []byte(script), 0644)
	if err != nil {
		return "", err
	}

	return scriptFile, nil
}

// parseResult 解析识别结果
func (f *FunASR) parseResult(output string) (ASRResult, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return ASRResult{
			Text:       "",
			Confidence: 0.0,
			Language:   f.config.Language,
			IsFinal:    true,
			StartTime:  time.Now().UnixNano() / 1000000,
			EndTime:    time.Now().UnixNano() / 1000000,
		}, nil
	}

	var result struct {
		Text       string  `json:"text"`
		Confidence float64 `json:"confidence"`
		Language   string  `json:"language"`
		IsFinal    bool    `json:"is_final"`
		Error      string  `json:"error,omitempty"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return ASRResult{}, fmt.Errorf("JSON解析失败: %w", err)
	}

	if result.Error != "" {
		return ASRResult{}, fmt.Errorf("FunASR错误: %s", result.Error)
	}

	return ASRResult{
		Text:        result.Text,
		Confidence:  result.Confidence,
		Language:    result.Language,
		IsFinal:     result.IsFinal,
		StartTime:   time.Now().UnixNano() / 1000000,
		EndTime:     time.Now().UnixNano() / 1000000,
		ProcessTime: time.Now().UnixNano() / 1000000,
		ModelInfo:   "FunASR",
	}, nil
}

// updateStats 更新统计信息
func (f *FunASR) updateStats(duration time.Duration, audioSize int) {
	f.totalRequests++
	f.totalDuration += duration.Seconds()

	// 估算音频时长（假设16kHz 16bit单声道）
	audioTime := float64(audioSize) / float64(f.config.SampleRate) / 2
	f.totalAudioTime += audioTime
}

// 注册FunASR
func init() {
	RegisterASR("funasr", func(config ASRConfig) (ASRService, error) {
		return NewFunASR(config), nil
	})
}
