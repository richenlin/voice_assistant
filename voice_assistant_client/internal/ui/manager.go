package ui

import (
	"context"
	"fmt"
	"time"

	"voice_assistant/voice_assistant_client/internal/config"
)

// Manager UI管理器
type Manager struct {
	config config.UIConfig

	// 状态
	isRunning bool

	// 显示组件
	console *ConsoleUI
}

// NewManager 创建UI管理器
func NewManager(config config.UIConfig) *Manager {
	return &Manager{
		config: config,
	}
}

// Start 启动UI
func (m *Manager) Start(ctx context.Context) error {
	if m.config.Type == "console" {
		m.console = NewConsoleUI(m.config.Console)
		if err := m.console.Start(ctx); err != nil {
			return fmt.Errorf("启动控制台UI失败: %w", err)
		}
	}

	m.isRunning = true
	return nil
}

// Stop 停止UI
func (m *Manager) Stop() error {
	if !m.isRunning {
		return nil
	}

	if m.console != nil {
		m.console.Stop()
	}

	m.isRunning = false
	return nil
}

// ShowASRResult 显示ASR识别结果
func (m *Manager) ShowASRResult(content string, confidence float64, isFinal bool) {
	if m.console != nil {
		m.console.ShowASRResult(content, confidence, isFinal)
	}
}

// ShowLLMResponse 显示LLM回复
func (m *Manager) ShowLLMResponse(content string, isFinal bool) {
	if m.console != nil {
		m.console.ShowLLMResponse(content, isFinal)
	}
}

// UpdateStatus 更新状态
func (m *Manager) UpdateStatus(state, mode string) {
	if m.console != nil {
		m.console.UpdateStatus(state, mode)
	}
}

// ShowError 显示错误
func (m *Manager) ShowError(code, message string) {
	if m.console != nil {
		m.console.ShowError(code, message)
	}
}

// ShowMessage 显示消息
func (m *Manager) ShowMessage(message string) {
	if m.console != nil {
		m.console.ShowMessage(message)
	}
}

// UpdateAudioLevel 更新音频级别
func (m *Manager) UpdateAudioLevel(average, peak float64) {
	if m.console != nil && m.config.ShowAudioLevel {
		m.console.UpdateAudioLevel(average, peak)
	}
}

// ConsoleUI 控制台UI
type ConsoleUI struct {
	config config.ConsoleConfig

	// 状态
	isRunning bool

	// 显示状态
	currentState string
	currentMode  string
	lastUpdate   time.Time
}

// NewConsoleUI 创建控制台UI
func NewConsoleUI(config config.ConsoleConfig) *ConsoleUI {
	return &ConsoleUI{
		config: config,
	}
}

// Start 启动控制台UI
func (c *ConsoleUI) Start(ctx context.Context) error {
	c.isRunning = true

	// 显示欢迎信息
	c.printWelcome()

	return nil
}

// Stop 停止控制台UI
func (c *ConsoleUI) Stop() error {
	if !c.isRunning {
		return nil
	}

	c.isRunning = false
	fmt.Println("\n再见！👋")
	return nil
}

// ShowASRResult 显示ASR识别结果
func (c *ConsoleUI) ShowASRResult(content string, confidence float64, isFinal bool) {
	timestamp := c.getTimestamp()
	status := "🎤"
	if isFinal {
		status = "✅"
	}

	if c.config.ColoredOutput {
		fmt.Printf("%s %s \033[36m[ASR]\033[0m %s (置信度: %.2f)\n",
			timestamp, status, content, confidence)
	} else {
		fmt.Printf("%s %s [ASR] %s (置信度: %.2f)\n",
			timestamp, status, content, confidence)
	}
}

// ShowLLMResponse 显示LLM回复
func (c *ConsoleUI) ShowLLMResponse(content string, isFinal bool) {
	timestamp := c.getTimestamp()
	status := "💭"
	if isFinal {
		status = "🤖"
	}

	if c.config.ColoredOutput {
		fmt.Printf("%s %s \033[32m[LLM]\033[0m %s\n", timestamp, status, content)
	} else {
		fmt.Printf("%s %s [LLM] %s\n", timestamp, status, content)
	}
}

// UpdateStatus 更新状态
func (c *ConsoleUI) UpdateStatus(state, mode string) {
	if state != c.currentState || mode != c.currentMode {
		c.currentState = state
		c.currentMode = mode
		c.lastUpdate = time.Now()

		timestamp := c.getTimestamp()
		statusIcon := c.getStatusIcon(state)

		if c.config.ColoredOutput {
			fmt.Printf("%s %s \033[33m[状态]\033[0m %s (%s)\n",
				timestamp, statusIcon, state, mode)
		} else {
			fmt.Printf("%s %s [状态] %s (%s)\n",
				timestamp, statusIcon, state, mode)
		}
	}
}

// ShowError 显示错误
func (c *ConsoleUI) ShowError(code, message string) {
	timestamp := c.getTimestamp()

	if c.config.ColoredOutput {
		fmt.Printf("%s ❌ \033[31m[错误]\033[0m %s: %s\n",
			timestamp, code, message)
	} else {
		fmt.Printf("%s ❌ [错误] %s: %s\n",
			timestamp, code, message)
	}
}

// ShowMessage 显示消息
func (c *ConsoleUI) ShowMessage(message string) {
	timestamp := c.getTimestamp()

	if c.config.ColoredOutput {
		fmt.Printf("%s 💬 \033[37m%s\033[0m\n", timestamp, message)
	} else {
		fmt.Printf("%s 💬 %s\n", timestamp, message)
	}
}

// UpdateAudioLevel 更新音频级别
func (c *ConsoleUI) UpdateAudioLevel(average, peak float64) {
	// 简单的音频级别显示（可以优化为进度条）
	if peak > 0.1 {
		level := int(peak * 10)
		if level > 10 {
			level = 10
		}

		bar := ""
		for i := 0; i < level; i++ {
			bar += "█"
		}
		for i := level; i < 10; i++ {
			bar += "░"
		}

		// 使用回车符覆盖上一行
		fmt.Printf("\r🔊 音频级别: [%s] %.2f", bar, peak)
	}
}

// printWelcome 打印欢迎信息
func (c *ConsoleUI) printWelcome() {
	if c.config.ColoredOutput {
		fmt.Println("\033[36m" + `
╔══════════════════════════════════════╗
║           语音助手客户端             ║
║        Voice Assistant Client       ║
╚══════════════════════════════════════╝
` + "\033[0m")
	} else {
		fmt.Println(`
╔══════════════════════════════════════╗
║           语音助手客户端             ║
║        Voice Assistant Client       ║
╚══════════════════════════════════════╝
`)
	}

	fmt.Println("🎤 请开始说话，系统会自动检测语音...")
	fmt.Println("📝 按 Ctrl+C 退出程序")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// getTimestamp 获取时间戳
func (c *ConsoleUI) getTimestamp() string {
	if !c.config.ShowTimestamps {
		return ""
	}

	return fmt.Sprintf("[%s]", time.Now().Format("15:04:05"))
}

// getStatusIcon 获取状态图标
func (c *ConsoleUI) getStatusIcon(state string) string {
	switch state {
	case "idle":
		return "😴"
	case "listening":
		return "👂"
	case "processing":
		return "🧠"
	case "speaking":
		return "🗣️"
	case "error":
		return "❌"
	case "connected":
		return "🔗"
	case "disconnected":
		return "🔌"
	default:
		return "❓"
	}
}
