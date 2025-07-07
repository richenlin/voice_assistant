package audio

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
)

// OutputConfig 音频输出配置
type OutputConfig struct {
	DeviceID   int    `yaml:"device_id"`
	SampleRate int    `yaml:"sample_rate"`
	Channels   int    `yaml:"channels"`
	Format     string `yaml:"format"`
	BufferSize int    `yaml:"buffer_size"`
}

// AudioOutput 音频输出管理器
type AudioOutput struct {
	config OutputConfig
	stream *portaudio.Stream
	device *portaudio.DeviceInfo

	// 状态管理
	isRunning bool
	isPlaying bool
	mu        sync.RWMutex

	// 音频数据通道
	audioChan   chan []float32
	controlChan chan outputControlSignal

	// 播放队列
	playQueue    [][]float32
	playQueueMu  sync.Mutex
	currentIndex int

	// 统计信息
	stats OutputStats
}

// outputControlSignal 输出控制信号
type outputControlSignal int

const (
	outputSignalStart outputControlSignal = iota
	outputSignalStop
	outputSignalPause
	outputSignalResume
	outputSignalClear
)

// OutputStats 音频输出统计信息
type OutputStats struct {
	TotalFrames   int64
	PlayedFrames  int64
	DroppedFrames int64
	QueueSize     int
	LastPlayTime  time.Time
	PlayDuration  time.Duration
}

// NewAudioOutput 创建音频输出管理器
func NewAudioOutput(config OutputConfig) (*AudioOutput, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化PortAudio失败: %w", err)
	}

	ao := &AudioOutput{
		config:      config,
		audioChan:   make(chan []float32, 100),
		controlChan: make(chan outputControlSignal, 10),
		playQueue:   make([][]float32, 0),
	}

	// 获取音频设备信息
	if err := ao.setupDevice(); err != nil {
		return nil, fmt.Errorf("设置音频设备失败: %w", err)
	}

	return ao, nil
}

// setupDevice 设置音频设备
func (ao *AudioOutput) setupDevice() error {
	var device *portaudio.DeviceInfo
	var err error

	if ao.config.DeviceID == -1 {
		// 使用默认输出设备
		device, err = portaudio.DefaultOutputDevice()
		if err != nil {
			return fmt.Errorf("获取默认输出设备失败: %w", err)
		}
	} else {
		// 使用指定设备
		devices, err := portaudio.Devices()
		if err != nil {
			return fmt.Errorf("获取设备列表失败: %w", err)
		}

		if ao.config.DeviceID >= len(devices) {
			return fmt.Errorf("设备ID %d 超出范围", ao.config.DeviceID)
		}

		device = devices[ao.config.DeviceID]
	}

	ao.device = device
	log.Printf("使用音频输出设备: %s", device.Name)

	return nil
}

// Start 启动音频输出
func (ao *AudioOutput) Start(ctx context.Context) error {
	ao.mu.Lock()
	if ao.isRunning {
		ao.mu.Unlock()
		return fmt.Errorf("音频输出已经在运行")
	}
	ao.isRunning = true
	ao.mu.Unlock()

	// 创建音频流
	outputParams := portaudio.StreamParameters{
		Output: portaudio.StreamDeviceParameters{
			Device:   ao.device,
			Channels: ao.config.Channels,
			Latency:  ao.device.DefaultLowOutputLatency,
		},
		SampleRate:      float64(ao.config.SampleRate),
		FramesPerBuffer: ao.config.BufferSize,
	}

	var err error
	ao.stream, err = portaudio.OpenStream(outputParams, ao.audioCallback)
	if err != nil {
		ao.mu.Lock()
		ao.isRunning = false
		ao.mu.Unlock()
		return fmt.Errorf("打开音频流失败: %w", err)
	}

	// 启动音频流
	if err := ao.stream.Start(); err != nil {
		ao.stream.Close()
		ao.mu.Lock()
		ao.isRunning = false
		ao.mu.Unlock()
		return fmt.Errorf("启动音频流失败: %w", err)
	}

	log.Printf("音频输出已启动: %dHz, %d通道, 缓冲区%d",
		ao.config.SampleRate, ao.config.Channels, ao.config.BufferSize)

	// 启动控制协程
	go ao.controlLoop(ctx)

	return nil
}

// Stop 停止音频输出
func (ao *AudioOutput) Stop() error {
	ao.mu.Lock()
	if !ao.isRunning {
		ao.mu.Unlock()
		return nil
	}
	ao.isRunning = false
	ao.mu.Unlock()

	// 发送停止信号
	select {
	case ao.controlChan <- outputSignalStop:
	default:
	}

	// 停止音频流
	if ao.stream != nil {
		if err := ao.stream.Stop(); err != nil {
			log.Printf("停止音频流失败: %v", err)
		}
		if err := ao.stream.Close(); err != nil {
			log.Printf("关闭音频流失败: %v", err)
		}
	}

	// 关闭通道
	close(ao.audioChan)
	close(ao.controlChan)

	// 清理PortAudio
	if err := portaudio.Terminate(); err != nil {
		log.Printf("清理PortAudio失败: %v", err)
	}

	log.Println("音频输出已停止")
	return nil
}

// Play 播放音频数据
func (ao *AudioOutput) Play(audioData []float32) error {
	ao.mu.RLock()
	if !ao.isRunning {
		ao.mu.RUnlock()
		return fmt.Errorf("音频输出未运行")
	}
	ao.mu.RUnlock()

	// 添加到播放队列
	ao.playQueueMu.Lock()
	ao.playQueue = append(ao.playQueue, audioData)
	ao.playQueueMu.Unlock()

	// 发送播放信号
	select {
	case ao.controlChan <- outputSignalStart:
	default:
	}

	return nil
}

// PlayBytes 播放字节数据
func (ao *AudioOutput) PlayBytes(audioData []byte) error {
	// 转换字节数据为float32
	floatData := BytesToFloat32(audioData)
	return ao.Play(floatData)
}

// StartPlaying 开始播放
func (ao *AudioOutput) StartPlaying() error {
	ao.mu.Lock()
	if ao.isPlaying {
		ao.mu.Unlock()
		return fmt.Errorf("已经在播放中")
	}
	ao.isPlaying = true
	ao.mu.Unlock()

	select {
	case ao.controlChan <- outputSignalStart:
		log.Println("开始播放")
		return nil
	default:
		ao.mu.Lock()
		ao.isPlaying = false
		ao.mu.Unlock()
		return fmt.Errorf("发送开始播放信号失败")
	}
}

// StopPlaying 停止播放
func (ao *AudioOutput) StopPlaying() error {
	ao.mu.Lock()
	if !ao.isPlaying {
		ao.mu.Unlock()
		return fmt.Errorf("当前没有在播放")
	}
	ao.isPlaying = false
	ao.mu.Unlock()

	select {
	case ao.controlChan <- outputSignalStop:
		log.Println("停止播放")
		return nil
	default:
		return fmt.Errorf("发送停止播放信号失败")
	}
}

// PausePlaying 暂停播放
func (ao *AudioOutput) PausePlaying() error {
	select {
	case ao.controlChan <- outputSignalPause:
		log.Println("暂停播放")
		return nil
	default:
		return fmt.Errorf("发送暂停播放信号失败")
	}
}

// ResumePlaying 恢复播放
func (ao *AudioOutput) ResumePlaying() error {
	select {
	case ao.controlChan <- outputSignalResume:
		log.Println("恢复播放")
		return nil
	default:
		return fmt.Errorf("发送恢复播放信号失败")
	}
}

// ClearQueue 清空播放队列
func (ao *AudioOutput) ClearQueue() error {
	select {
	case ao.controlChan <- outputSignalClear:
		log.Println("清空播放队列")
		return nil
	default:
		return fmt.Errorf("发送清空队列信号失败")
	}
}

// GetStats 获取统计信息
func (ao *AudioOutput) GetStats() OutputStats {
	ao.mu.RLock()
	defer ao.mu.RUnlock()

	ao.playQueueMu.Lock()
	queueSize := len(ao.playQueue)
	ao.playQueueMu.Unlock()

	stats := ao.stats
	stats.QueueSize = queueSize
	return stats
}

// IsRunning 检查是否正在运行
func (ao *AudioOutput) IsRunning() bool {
	ao.mu.RLock()
	defer ao.mu.RUnlock()
	return ao.isRunning
}

// IsPlaying 检查是否正在播放
func (ao *AudioOutput) IsPlaying() bool {
	ao.mu.RLock()
	defer ao.mu.RUnlock()
	return ao.isPlaying
}

// audioCallback 音频回调函数
func (ao *AudioOutput) audioCallback(out []float32) {
	ao.mu.RLock()
	isPlaying := ao.isPlaying
	ao.mu.RUnlock()

	if !isPlaying {
		// 输出静音
		for i := range out {
			out[i] = 0
		}
		return
	}

	// 从播放队列获取数据
	ao.playQueueMu.Lock()
	if len(ao.playQueue) == 0 {
		ao.playQueueMu.Unlock()
		// 没有数据，输出静音
		for i := range out {
			out[i] = 0
		}
		return
	}

	// 获取当前音频数据
	currentData := ao.playQueue[0]

	// 复制数据到输出缓冲区
	copyLen := len(out)
	if len(currentData) < copyLen {
		copyLen = len(currentData)
	}

	copy(out[:copyLen], currentData[:copyLen])

	// 如果输出缓冲区更大，用静音填充
	for i := copyLen; i < len(out); i++ {
		out[i] = 0
	}

	// 更新播放队列
	if len(currentData) <= len(out) {
		// 当前数据已播放完，移除
		ao.playQueue = ao.playQueue[1:]
	} else {
		// 当前数据还有剩余
		ao.playQueue[0] = currentData[len(out):]
	}

	ao.playQueueMu.Unlock()

	// 更新统计信息
	ao.updateStats(len(out))
}

// controlLoop 控制循环
func (ao *AudioOutput) controlLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case signal := <-ao.controlChan:
			switch signal {
			case outputSignalStart:
				ao.mu.Lock()
				ao.isPlaying = true
				ao.stats.LastPlayTime = time.Now()
				ao.mu.Unlock()
			case outputSignalStop:
				ao.mu.Lock()
				ao.isPlaying = false
				ao.mu.Unlock()
			case outputSignalPause:
				ao.mu.Lock()
				ao.isPlaying = false
				ao.mu.Unlock()
			case outputSignalResume:
				ao.mu.Lock()
				ao.isPlaying = true
				ao.stats.LastPlayTime = time.Now()
				ao.mu.Unlock()
			case outputSignalClear:
				ao.playQueueMu.Lock()
				ao.playQueue = ao.playQueue[:0]
				ao.currentIndex = 0
				ao.playQueueMu.Unlock()
			}
		}
	}
}

// updateStats 更新统计信息
func (ao *AudioOutput) updateStats(framesPlayed int) {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	ao.stats.TotalFrames += int64(framesPlayed)
	ao.stats.PlayedFrames += int64(framesPlayed)

	if !ao.stats.LastPlayTime.IsZero() {
		ao.stats.PlayDuration += time.Since(ao.stats.LastPlayTime)
		ao.stats.LastPlayTime = time.Now()
	}
}

// BytesToFloat32 将字节数据转换为float32
func BytesToFloat32(data []byte) []float32 {
	if len(data)%2 != 0 {
		// 确保字节数为偶数
		data = data[:len(data)-1]
	}

	result := make([]float32, len(data)/2)
	for i := 0; i < len(result); i++ {
		// 16位PCM转float32
		sample := int16(data[i*2]) | int16(data[i*2+1])<<8
		result[i] = float32(sample) / 32768.0
	}
	return result
}

// Float32ToBytes 将float32数据转换为字节
func Float32ToBytes(data []float32) []byte {
	result := make([]byte, len(data)*2)
	for i, sample := range data {
		// float32转16位PCM
		pcm := int16(sample * 32767)
		result[i*2] = byte(pcm)
		result[i*2+1] = byte(pcm >> 8)
	}
	return result
}

// GetOutputDeviceList 获取可用的音频输出设备列表
func GetOutputDeviceList() ([]*portaudio.DeviceInfo, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化PortAudio失败: %w", err)
	}
	defer portaudio.Terminate()

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("获取设备列表失败: %w", err)
	}

	var outputDevices []*portaudio.DeviceInfo
	for _, device := range devices {
		if device.MaxOutputChannels > 0 {
			outputDevices = append(outputDevices, device)
		}
	}

	return outputDevices, nil
}

// PrintOutputDeviceList 打印输出设备列表
func PrintOutputDeviceList() error {
	devices, err := GetOutputDeviceList()
	if err != nil {
		return err
	}

	log.Println("可用的音频输出设备:")
	for i, device := range devices {
		log.Printf("  %d: %s (输出通道: %d, 采样率: %.0f Hz)",
			i, device.Name, device.MaxOutputChannels, device.DefaultSampleRate)
	}

	return nil
}
