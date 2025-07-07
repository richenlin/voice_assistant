package audio

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
)

// InputConfig 音频输入配置
type InputConfig struct {
	DeviceID           int     `yaml:"device_id"`
	SampleRate         int     `yaml:"sample_rate"`
	Channels           int     `yaml:"channels"`
	Format             string  `yaml:"format"`
	BufferSize         int     `yaml:"buffer_size"`
	ChunkDuration      int     `yaml:"chunk_duration"` // 毫秒
	VADEnabled         bool    `yaml:"vad_enabled"`
	VADThreshold       float64 `yaml:"vad_threshold"`
	MinSpeechDuration  int     `yaml:"min_speech_duration"`  // 毫秒
	MinSilenceDuration int     `yaml:"min_silence_duration"` // 毫秒
}

// AudioInput 音频输入管理器
type AudioInput struct {
	config InputConfig
	stream *portaudio.Stream
	device *portaudio.DeviceInfo

	// 状态管理
	isRunning   bool
	isRecording bool
	mu          sync.RWMutex

	// 音频数据通道
	audioChan   chan []float32
	controlChan chan controlSignal

	// VAD检测
	vadDetector *VADDetector

	// 统计信息
	stats AudioStats
}

// controlSignal 控制信号
type controlSignal int

const (
	signalStart controlSignal = iota
	signalStop
	signalPause
	signalResume
)

// AudioStats 音频统计信息
type AudioStats struct {
	TotalFrames  int64
	ActiveFrames int64
	SilentFrames int64
	LastActivity time.Time
	AverageLevel float64
	PeakLevel    float64
}

// NewAudioInput 创建音频输入管理器
func NewAudioInput(config InputConfig) (*AudioInput, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化PortAudio失败: %w", err)
	}

	ai := &AudioInput{
		config:      config,
		audioChan:   make(chan []float32, 100),
		controlChan: make(chan controlSignal, 10),
		vadDetector: NewVADDetector(config.VADThreshold, config.MinSpeechDuration, config.MinSilenceDuration),
	}

	// 获取音频设备信息
	if err := ai.setupDevice(); err != nil {
		return nil, fmt.Errorf("设置音频设备失败: %w", err)
	}

	return ai, nil
}

// setupDevice 设置音频设备
func (ai *AudioInput) setupDevice() error {
	var device *portaudio.DeviceInfo
	var err error

	if ai.config.DeviceID == -1 {
		// 使用默认输入设备
		device, err = portaudio.DefaultInputDevice()
		if err != nil {
			return fmt.Errorf("获取默认输入设备失败: %w", err)
		}
	} else {
		// 使用指定设备
		devices, err := portaudio.Devices()
		if err != nil {
			return fmt.Errorf("获取设备列表失败: %w", err)
		}

		if ai.config.DeviceID >= len(devices) {
			return fmt.Errorf("设备ID %d 超出范围", ai.config.DeviceID)
		}

		device = devices[ai.config.DeviceID]
	}

	ai.device = device
	log.Printf("使用音频输入设备: %s", device.Name)

	return nil
}

// Start 启动音频输入
func (ai *AudioInput) Start(ctx context.Context) error {
	ai.mu.Lock()
	if ai.isRunning {
		ai.mu.Unlock()
		return fmt.Errorf("音频输入已经在运行")
	}
	ai.isRunning = true
	ai.mu.Unlock()

	// 创建音频流
	inputParams := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   ai.device,
			Channels: ai.config.Channels,
			Latency:  ai.device.DefaultLowInputLatency,
		},
		SampleRate:      float64(ai.config.SampleRate),
		FramesPerBuffer: ai.config.BufferSize,
	}

	var err error
	ai.stream, err = portaudio.OpenStream(inputParams, ai.audioCallback)
	if err != nil {
		ai.mu.Lock()
		ai.isRunning = false
		ai.mu.Unlock()
		return fmt.Errorf("打开音频流失败: %w", err)
	}

	// 启动音频流
	if err := ai.stream.Start(); err != nil {
		ai.stream.Close()
		ai.mu.Lock()
		ai.isRunning = false
		ai.mu.Unlock()
		return fmt.Errorf("启动音频流失败: %w", err)
	}

	log.Printf("音频输入已启动: %dHz, %d通道, 缓冲区%d",
		ai.config.SampleRate, ai.config.Channels, ai.config.BufferSize)

	// 启动控制协程
	go ai.controlLoop(ctx)

	return nil
}

// Stop 停止音频输入
func (ai *AudioInput) Stop() error {
	ai.mu.Lock()
	if !ai.isRunning {
		ai.mu.Unlock()
		return nil
	}
	ai.isRunning = false
	ai.mu.Unlock()

	// 发送停止信号
	select {
	case ai.controlChan <- signalStop:
	default:
	}

	// 停止音频流
	if ai.stream != nil {
		if err := ai.stream.Stop(); err != nil {
			log.Printf("停止音频流失败: %v", err)
		}
		if err := ai.stream.Close(); err != nil {
			log.Printf("关闭音频流失败: %v", err)
		}
	}

	// 关闭通道
	close(ai.audioChan)
	close(ai.controlChan)

	// 清理PortAudio
	if err := portaudio.Terminate(); err != nil {
		log.Printf("清理PortAudio失败: %v", err)
	}

	log.Println("音频输入已停止")
	return nil
}

// StartRecording 开始录音
func (ai *AudioInput) StartRecording() error {
	ai.mu.Lock()
	if ai.isRecording {
		ai.mu.Unlock()
		return fmt.Errorf("已经在录音中")
	}
	ai.isRecording = true
	ai.mu.Unlock()

	select {
	case ai.controlChan <- signalStart:
		log.Println("开始录音")
		return nil
	default:
		ai.mu.Lock()
		ai.isRecording = false
		ai.mu.Unlock()
		return fmt.Errorf("发送开始录音信号失败")
	}
}

// StopRecording 停止录音
func (ai *AudioInput) StopRecording() error {
	ai.mu.Lock()
	if !ai.isRecording {
		ai.mu.Unlock()
		return fmt.Errorf("当前没有在录音")
	}
	ai.isRecording = false
	ai.mu.Unlock()

	select {
	case ai.controlChan <- signalStop:
		log.Println("停止录音")
		return nil
	default:
		return fmt.Errorf("发送停止录音信号失败")
	}
}

// PauseRecording 暂停录音
func (ai *AudioInput) PauseRecording() error {
	select {
	case ai.controlChan <- signalPause:
		log.Println("暂停录音")
		return nil
	default:
		return fmt.Errorf("发送暂停录音信号失败")
	}
}

// ResumeRecording 恢复录音
func (ai *AudioInput) ResumeRecording() error {
	select {
	case ai.controlChan <- signalResume:
		log.Println("恢复录音")
		return nil
	default:
		return fmt.Errorf("发送恢复录音信号失败")
	}
}

// GetAudioChannel 获取音频数据通道
func (ai *AudioInput) GetAudioChannel() <-chan []float32 {
	return ai.audioChan
}

// GetStats 获取统计信息
func (ai *AudioInput) GetStats() AudioStats {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.stats
}

// IsRunning 检查是否正在运行
func (ai *AudioInput) IsRunning() bool {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.isRunning
}

// IsRecording 检查是否正在录音
func (ai *AudioInput) IsRecording() bool {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.isRecording
}

// audioCallback 音频回调函数
func (ai *AudioInput) audioCallback(in []float32) {
	ai.mu.RLock()
	isRecording := ai.isRecording
	ai.mu.RUnlock()

	if !isRecording {
		return
	}

	// 更新统计信息
	ai.updateStats(in)

	// VAD检测
	if ai.config.VADEnabled {
		isVoice := ai.vadDetector.Detect(in)
		if !isVoice {
			return
		}
	}

	// 复制音频数据
	audioData := make([]float32, len(in))
	copy(audioData, in)

	// 发送音频数据
	select {
	case ai.audioChan <- audioData:
	default:
		log.Printf("音频缓冲区已满，丢弃数据")
	}
}

// controlLoop 控制循环
func (ai *AudioInput) controlLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case signal := <-ai.controlChan:
			switch signal {
			case signalStart:
				ai.mu.Lock()
				ai.isRecording = true
				ai.mu.Unlock()
			case signalStop:
				ai.mu.Lock()
				ai.isRecording = false
				ai.mu.Unlock()
			case signalPause:
				ai.mu.Lock()
				ai.isRecording = false
				ai.mu.Unlock()
			case signalResume:
				ai.mu.Lock()
				ai.isRecording = true
				ai.mu.Unlock()
			}
		}
	}
}

// updateStats 更新统计信息
func (ai *AudioInput) updateStats(data []float32) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	ai.stats.TotalFrames += int64(len(data))

	// 计算音频级别
	var sum float64
	var peak float64
	var activeFrames int64

	for _, sample := range data {
		abs := math.Abs(float64(sample))
		sum += abs
		if abs > peak {
			peak = abs
		}
		if abs > 0.01 { // 认为是活跃音频
			activeFrames++
		}
	}

	ai.stats.ActiveFrames += activeFrames
	ai.stats.SilentFrames += int64(len(data)) - activeFrames
	ai.stats.AverageLevel = sum / float64(len(data))
	ai.stats.PeakLevel = peak

	if activeFrames > 0 {
		ai.stats.LastActivity = time.Now()
	}
}

// GetDeviceList 获取可用的音频输入设备列表
func GetDeviceList() ([]*portaudio.DeviceInfo, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化PortAudio失败: %w", err)
	}
	defer portaudio.Terminate()

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("获取设备列表失败: %w", err)
	}

	var inputDevices []*portaudio.DeviceInfo
	for _, device := range devices {
		if device.MaxInputChannels > 0 {
			inputDevices = append(inputDevices, device)
		}
	}

	return inputDevices, nil
}

// PrintDeviceList 打印设备列表
func PrintDeviceList() error {
	devices, err := GetDeviceList()
	if err != nil {
		return err
	}

	log.Println("可用的音频输入设备:")
	for i, device := range devices {
		log.Printf("  %d: %s (输入通道: %d, 采样率: %.0f Hz)",
			i, device.Name, device.MaxInputChannels, device.DefaultSampleRate)
	}

	return nil
}
