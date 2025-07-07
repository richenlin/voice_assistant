package audio

import (
	"math"
	"time"
)

// VADDetector 语音活动检测器
type VADDetector struct {
	threshold          float64
	minSpeechDuration  int // 毫秒
	minSilenceDuration int // 毫秒

	// 状态跟踪
	isInSpeech       bool
	speechStartTime  time.Time
	silenceStartTime time.Time

	// 统计信息
	frameCount    int64
	speechFrames  int64
	silenceFrames int64

	// 能量计算
	energyHistory []float64
	historySize   int
	historyIndex  int
}

// NewVADDetector 创建新的VAD检测器
func NewVADDetector(threshold float64, minSpeechDuration, minSilenceDuration int) *VADDetector {
	return &VADDetector{
		threshold:          threshold,
		minSpeechDuration:  minSpeechDuration,
		minSilenceDuration: minSilenceDuration,
		historySize:        10,
		energyHistory:      make([]float64, 10),
	}
}

// Detect 检测语音活动
func (v *VADDetector) Detect(audioData []float32) bool {
	v.frameCount++

	// 计算音频能量
	energy := v.calculateEnergy(audioData)

	// 更新能量历史
	v.updateEnergyHistory(energy)

	// 获取自适应阈值
	adaptiveThreshold := v.getAdaptiveThreshold()

	// 检测语音活动
	hasVoice := energy > adaptiveThreshold

	now := time.Now()

	if hasVoice {
		if !v.isInSpeech {
			// 开始检测到语音
			v.speechStartTime = now
			v.isInSpeech = true
		}
		v.speechFrames++

		// 检查是否满足最小语音持续时间
		if now.Sub(v.speechStartTime) >= time.Duration(v.minSpeechDuration)*time.Millisecond {
			return true
		}
	} else {
		if v.isInSpeech {
			// 开始检测到静音
			v.silenceStartTime = now
			v.isInSpeech = false
		}
		v.silenceFrames++

		// 检查是否满足最小静音持续时间
		if now.Sub(v.silenceStartTime) >= time.Duration(v.minSilenceDuration)*time.Millisecond {
			return false
		}

		// 如果还在语音状态但检测到静音，继续返回true直到满足静音时间
		if v.isInSpeech {
			return true
		}
	}

	return v.isInSpeech
}

// calculateEnergy 计算音频能量
func (v *VADDetector) calculateEnergy(audioData []float32) float64 {
	var sum float64
	for _, sample := range audioData {
		sum += float64(sample) * float64(sample)
	}

	// 计算RMS（均方根）
	rms := math.Sqrt(sum / float64(len(audioData)))

	// 转换为dB
	if rms > 0 {
		return 20 * math.Log10(rms)
	}

	return -100.0 // 静音时的dB值
}

// updateEnergyHistory 更新能量历史
func (v *VADDetector) updateEnergyHistory(energy float64) {
	v.energyHistory[v.historyIndex] = energy
	v.historyIndex = (v.historyIndex + 1) % v.historySize
}

// getAdaptiveThreshold 获取自适应阈值
func (v *VADDetector) getAdaptiveThreshold() float64 {
	if v.frameCount < int64(v.historySize) {
		return v.threshold
	}

	// 计算能量历史的平均值和标准差
	var sum, sumSquares float64
	for _, energy := range v.energyHistory {
		sum += energy
		sumSquares += energy * energy
	}

	mean := sum / float64(v.historySize)
	variance := (sumSquares / float64(v.historySize)) - (mean * mean)
	stdDev := math.Sqrt(variance)

	// 自适应阈值 = 平均值 + 2*标准差
	adaptiveThreshold := mean + 2*stdDev

	// 确保不低于最小阈值
	if adaptiveThreshold < v.threshold {
		adaptiveThreshold = v.threshold
	}

	return adaptiveThreshold
}

// GetStats 获取VAD统计信息
func (v *VADDetector) GetStats() VADStats {
	return VADStats{
		FrameCount:    v.frameCount,
		SpeechFrames:  v.speechFrames,
		SilenceFrames: v.silenceFrames,
		SpeechRatio:   float64(v.speechFrames) / float64(v.frameCount),
		IsInSpeech:    v.isInSpeech,
	}
}

// VADStats VAD统计信息
type VADStats struct {
	FrameCount    int64
	SpeechFrames  int64
	SilenceFrames int64
	SpeechRatio   float64
	IsInSpeech    bool
}

// Reset 重置VAD检测器
func (v *VADDetector) Reset() {
	v.isInSpeech = false
	v.speechStartTime = time.Time{}
	v.silenceStartTime = time.Time{}
	v.frameCount = 0
	v.speechFrames = 0
	v.silenceFrames = 0
	v.historyIndex = 0
	for i := range v.energyHistory {
		v.energyHistory[i] = 0
	}
}

// SetThreshold 设置阈值
func (v *VADDetector) SetThreshold(threshold float64) {
	v.threshold = threshold
}

// SetMinDurations 设置最小持续时间
func (v *VADDetector) SetMinDurations(minSpeechDuration, minSilenceDuration int) {
	v.minSpeechDuration = minSpeechDuration
	v.minSilenceDuration = minSilenceDuration
}

// IsInSpeech 检查是否处于语音状态
func (v *VADDetector) IsInSpeech() bool {
	return v.isInSpeech
}
