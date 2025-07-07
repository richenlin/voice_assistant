# Voice Assistant - 企业级语音助手系统

一个基于 **客户端-服务端分离架构** 的现代化语音助手系统，采用 Go 语言开发，支持实时语音交互、多引擎切换和容器化部署。

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](docker-compose.yml)

## 🏗️ 系统架构

### 整体架构

```
┌─────────────────┐    WebSocket    ┌─────────────────┐    HTTP/API    ┌─────────────────┐
│   客户端 (轻量)   │ ◄──────────────► │   服务端 (重型)   │ ◄─────────────► │   外部Ollama     │
│                 │                 │                 │                │                 │
│ • 音频输入输出   │                 │ • ASR (FunASR)  │                │ • LLM推理       │
│ • VAD语音检测   │                 │ • TTS (ChatTTS) │                │ • 模型管理       │
│ • WebSocket连接 │                 │ • 会话管理       │                │ • API服务       │
│ • 实时音频流    │                 │ • 消息处理       │                │                 │
└─────────────────┘                 └─────────────────┘                └─────────────────┘
```

### 技术栈

#### 服务端 (Linux部署)
- **Web框架**: Gin + WebSocket
- **ASR引擎**: FunASR (默认), OpenAI Whisper
- **LLM引擎**: Ollama (默认), OpenAI GPT, WebSocket
- **TTS引擎**: ChatTTS (默认), Edge-TTS, Sherpa-ONNX
- **容器化**: Docker + Docker Compose
- **监控**: Prometheus + Grafana

#### 客户端 (跨平台)
- **音频处理**: PortAudio
- **WebSocket**: Gorilla WebSocket
- **VAD检测**: 自实现VAD算法
- **跨平台构建**: Go交叉编译
- **UI支持**: 控制台/图形界面

## 🚀 核心特性

### 基础功能
- **客户端-服务端分离**：轻量客户端 + 重型服务端，优化资源分配
- **实时WebSocket通信**：低延迟双向音频流传输
- **多引擎支持**：ASR、LLM、TTS引擎可灵活配置和切换
- **完全离线运行**：支持FunASR + ChatTTS + Ollama的完全离线配置
- **跨平台支持**：客户端支持Windows/Linux/macOS

### 🌟 高级功能 (开箱即用!)

#### 1. 连续对话模式 ✅ 默认启用
- **智能唤醒**: 唤醒词激活后，可连续接收语音命令无需重复唤醒
- **自动休眠**: 静音检测后智能休眠，节省资源
- **无缝切换**: 自动在唤醒词模式、连续对话模式、可打断模式之间智能切换

#### 2. 并发音频处理 ✅ 默认启用
- **同时播放和录音**: 系统播放回复时可同时接收新的语音输入
- **智能打断**: 支持语音打断当前播放，实现自然的对话体验
- **多音频流管理**: 自动管理并发音频流，优化性能

#### 3. 智能上下文管理 ✅ 默认启用
- **对话历史**: 自动维护对话历史和上下文
- **话题跟踪**: 智能识别和跟踪对话话题变化
- **用户偏好**: 学习和记录用户偏好，提供个性化体验

#### 4. 企业级特性 ✅ 生产就绪
- **容器化部署**: Docker + Docker Compose一键部署
- **监控告警**: Prometheus + Grafana监控体系
- **健康检查**: 自动健康检查和服务重启
- **资源管理**: 内存和CPU资源限制配置
- **日志管理**: 结构化日志和日志轮转

## 📁 项目结构

```
voice_assistant/
├── voice_assistant_server/          # 服务端 (Linux部署)
│   ├── cmd/server/                  # 服务端主程序
│   ├── internal/                    # 内部实现
│   │   ├── asr/                     # ASR模块 (FunASR, OpenAI)
│   │   ├── llm/                     # LLM模块 (Ollama, OpenAI, WebSocket)
│   │   ├── tts/                     # TTS模块 (ChatTTS, Edge-TTS)
│   │   ├── server/                  # WebSocket服务器
│   │   └── session/                 # 会话管理
│   ├── config/                      # 配置文件
│   ├── docker-compose.yml          # 容器编排
│   └── DEPLOYMENT.md               # 部署文档
├── voice_assistant_client/          # 客户端 (跨平台)
│   ├── cmd/client/                  # 客户端主程序
│   ├── internal/                    # 内部实现
│   │   ├── audio/                   # 音频处理
│   │   ├── client/                  # WebSocket客户端
│   │   └── ui/                      # 用户界面
│   ├── Makefile                     # 跨平台构建
│   └── BUILD.md                     # 构建文档
└── pkg/protocol/                    # 通信协议包
```

## 功能特点

- 🎯 管道架构：基于Pipeline模式的语音处理流程
- 🔄 事件驱动：基于事件总线（EventBus）的组件通信
- 🚦 状态管理：完善的状态机控制流程
- 🎨 模块化设计：各组件可独立配置和替换
- 🔒 **离线优先**：默认使用离线引擎，保护数据隐私，无网络依赖
- 🔊 离线语音处理：支持本地语音唤醒、识别和合成
- 🤖 智能对话：集成多种大语言模型（支持本地部署）
- 🎵 实时音频处理：支持实时音频输入输出和回声消除
- ⚡ 灵活切换：支持离线/在线引擎快速切换
- 📦 配置灵活：支持 YAML 配置文件
- 📊 状态监控：内置统计和错误处理机制

## 系统架构

系统采用管道（Pipeline）模式处理语音交互流程，通过事件总线（EventBus）实现组件间通信，并使用状态机管理系统状态。

### 核心组件

- **语音唤醒 (KWS)**：检测唤醒词，支持 sherpa-onnx、snowboy 等引擎
- **语音活动检测 (VAD)**：检测语音活动开始和结束，支持 webrtc 等引擎
- **语音识别 (ASR)**：将语音转换为文本，支持 sherpa-onnx、whisper 等引擎
- **大语言模型 (LLM)**：处理自然语言对话，支持 OpenAI、Ollama、WebSocket 等
- **语音合成 (TTS)**：将文本转换为语音，支持 sherpa-onnx、edge-tts 等引擎
- **音频处理**：处理音频输入输出，支持回声消除

### 工作流程

1. **唤醒阶段**：KWS 模块检测唤醒词
2. **录音阶段**：VAD 模块检测用户语音输入的开始和结束
3. **识别阶段**：ASR 模块将语音转换为文本
4. **对话阶段**：LLM 模块处理文本并生成回复
5. **合成阶段**：TTS 模块将回复文本转换为语音
6. **播放阶段**：音频输出模块播放合成的语音

### 事件驱动

系统基于事件总线实现组件间通信，主要事件包括：

- **唤醒事件**：触发录音开始
- **语音开始/结束事件**：控制录音过程
- **识别结果事件**：传递识别文本
- **对话结果事件**：传递 LLM 回复
- **合成完成事件**：触发语音播放

## 🛠️ 快速开始

### 系统要求

#### 服务端 (Linux)
- **操作系统**: Linux (Ubuntu 20.04+, CentOS 8+)
- **硬件**: 2核心以上 CPU, 4GB以上内存, 20GB可用存储
- **软件**: Docker 20.10+, Docker Compose 2.0+
- **外部依赖**: Ollama服务（独立部署）

#### 客户端 (跨平台)
- **操作系统**: Windows 10+, macOS 10.15+, Linux
- **硬件**: 支持音频输入输出设备
- **网络**: 能够连接到服务端的网络环境

### 部署步骤

#### 1. 服务端部署 (Linux)

```bash
# 1. 克隆项目
git clone <repository-url>
cd voice_assistant/voice_assistant_server

# 2. 安装并启动Ollama (必需的外部依赖)
curl -fsSL https://ollama.ai/install.sh | sh
ollama serve
ollama pull qwen:7b  # 下载中文模型

# 3. 配置服务端
cp env.example .env
# 编辑 .env 文件设置必要参数

# 4. 一键部署服务端
./scripts/deploy.sh
# 选择 1) 基础模式 或 2) 监控模式

# 5. 验证部署
curl http://localhost:8080/health
```

#### 2. 客户端部署 (任意平台)

##### Windows 用户
```bash
# 1. 下载预编译版本
# 从 Releases 页面下载 voice_assistant_client-windows-amd64.zip

# 2. 解压并配置
# 编辑 config/client.yaml 设置服务端地址

# 3. 运行客户端
voice_assistant_client.exe
```

##### Linux/macOS 用户
```bash
# 1. 克隆客户端代码
cd voice_assistant/voice_assistant_client

# 2. 安装依赖
sudo apt-get install portaudio19-dev  # Ubuntu
# 或
brew install portaudio  # macOS

# 3. 构建客户端
make build

# 4. 配置并运行
# 编辑 config/client.yaml 设置服务端地址
./bin/voice_assistant_client
```

### 开发环境搭建

#### 服务端开发
```bash
cd voice_assistant_server

# 安装Go依赖
go mod download

# 本地运行 (需要先启动Ollama)
go run cmd/server/main.go
```

#### 客户端开发
```bash
cd voice_assistant_client

# 安装系统依赖
# Linux: sudo apt-get install portaudio19-dev
# macOS: brew install portaudio
# Windows: 参考 BUILD.md

# 安装Go依赖
go mod download

# 本地运行
go run cmd/client/main.go
```

## 🔧 配置说明

### 服务端配置 (`voice_assistant_server/config/server.yaml`)

```yaml
# 服务器配置
server:
  host: "0.0.0.0"
  port: 8080

# ASR配置 - 默认使用FunASR（离线）
asr:
  provider: "funasr"
  funasr:
    model_dir: "./models/funasr/paraformer-zh"
    device_id: "cpu"

# LLM配置 - 默认使用Ollama（离线）
llm:
  provider: "ollama"
  ollama:
    base_url: "http://localhost:11434"  # 外部Ollama服务
    model: "qwen:7b"

# TTS配置 - 默认使用ChatTTS（离线）
tts:
  provider: "chattts"
  chattts:
    device: "cpu"
    temperature: 0.3
```

### 客户端配置 (`voice_assistant_client/config/client.yaml`)

```yaml
# 服务器连接配置
server:
  host: "localhost"  # 服务端地址
  port: 8080
  websocket_path: "/ws"
  reconnect_interval: 5s
  max_reconnect_attempts: 10

# 音频配置
audio:
  input:
    device_id: -1  # -1表示默认设备
    sample_rate: 16000
    channels: 1
  output:
    device_id: -1
    sample_rate: 16000
    channels: 1
  vad:
    enabled: true
    threshold: 0.5

# 会话配置
session:
  mode: "continuous"  # continuous, single, wakeword
  timeout: 30m
  auto_reconnect: true
```

## 🔧 支持的引擎

### 服务端引擎支持

#### 语音识别 (ASR)
- **FunASR** (离线，默认) - 达摩院开源，高准确率95%+，中文优化
- **OpenAI Whisper** (在线) - OpenAI官方API，多语言支持
- **本地Whisper** (离线，开发中) - 本地部署Whisper模型

#### 大语言模型 (LLM)
- **Ollama** (离线，默认) - 本地部署开源模型，支持Qwen、LLaMA等
- **OpenAI GPT** (在线) - GPT-3.5/4系列模型
- **WebSocket LLM** (可离线/在线) - 自定义LLM服务接口

#### 语音合成 (TTS)
- **ChatTTS** (离线，默认) - 开源TTS模型，顶级音质，情感丰富
- **Edge TTS** (在线) - 微软Edge浏览器TTS服务，多语言支持
- **Sherpa-ONNX** (离线，开发中) - 轻量级TTS引擎

### 客户端功能

#### 语音活动检测 (VAD)
- **自实现VAD** (默认) - 基于音频能量和频谱分析
- **阈值可调节** - 支持敏感度调整

#### 音频处理
- **PortAudio** - 跨平台音频I/O，支持多种音频设备
- **实时处理** - 低延迟音频流处理
- **格式转换** - PCM/Float32自动转换

## 🚀 使用场景

### 基础使用场景

#### 场景1: 连续对话 ✅ 自动启用
```
用户: 启动客户端后自动连接服务端
系统: "语音助手已启动，连续对话模式已激活"
用户: "今天天气怎么样？"           # 直接说话，无需唤醒词
系统: "今天天气晴朗..."
用户: "那明天呢？"               # 继续对话，自动维护上下文
系统: "明天预计..."
[静音30秒后会话超时]
```

#### 场景2: 智能打断 ✅ 自动启用
```
用户: "请介绍一下人工智能"
系统: [开始长篇回复] "人工智能是..."
用户: "停止"                   # 直接说话打断
系统: [立即停止] "好的，我已停止。请问还有什么需要帮助的吗？"
```

### 企业级部署场景

#### 场景1: 客服系统
- **服务端**: 部署在云服务器，支持多客户并发
- **客户端**: 部署在客服工作站，实时语音交互
- **监控**: Grafana监控系统负载和响应时间

#### 场景2: 智能家居
- **服务端**: 部署在家庭NAS或边缘计算设备
- **客户端**: 部署在各房间的智能设备
- **特点**: 完全离线运行，保护隐私

## 📊 性能特点

### 音频处理性能
- **延迟**: < 100ms 端到端延迟
- **并发**: 支持多音频流同时处理
- **格式**: 16kHz PCM，自动格式转换
- **VAD**: 实时语音活动检测，减少无效传输

### 网络通信性能
- **协议**: WebSocket长连接，低延迟
- **重连**: 自动重连机制，网络中断恢复
- **压缩**: 可选音频数据压缩传输
- **心跳**: 30秒心跳检测，保持连接活跃

### 系统资源使用
- **服务端**: 2-4GB内存，2-4核CPU（根据并发数）
- **客户端**: < 100MB内存，< 1核CPU
- **存储**: 服务端需要模型存储空间（5-20GB）

## 🔄 扩展和定制

### 添加新的ASR引擎

```go
// 1. 实现ASRService接口
type CustomASR struct {
    config ASRConfig
}

func (c *CustomASR) RecognizeStream(ctx context.Context, 
    audioStream <-chan []byte) (<-chan ASRResult, error) {
    // 实现流式识别逻辑
}

// 2. 注册引擎
func init() {
    RegisterASR("custom", func(config ASRConfig) (ASRService, error) {
        return NewCustomASR(config)
    })
}
```

### 添加新的LLM引擎

```go
// 1. 实现LLMService接口
type CustomLLM struct {
    config LLMConfig
}

func (c *CustomLLM) Chat(ctx context.Context, 
    userInput string, conversationID string) (LLMResponse, error) {
    // 实现对话逻辑
}

// 2. 注册引擎
func init() {
    RegisterLLM("custom", func(config LLMConfig) (LLMService, error) {
        return NewCustomLLM(config)
    })
}
```

### 自定义通信协议

```go
// 扩展消息类型
const (
    MessageTypeCustom MessageType = "custom"
)

// 自定义消息数据
type CustomData struct {
    Action string      `json:"action"`
    Data   interface{} `json:"data"`
}
```

## 状态流转

语音助手系统包含以下主要状态：

1. **空闲状态 (Idle)**：等待唤醒
2. **资源检查 (ResourceCheck)**：检查系统资源
3. **监听状态 (Listening)**：接收用户语音输入
4. **处理状态 (Processing)**：处理语音识别和对话
5. **说话状态 (Speaking)**：播放合成的语音
6. **错误状态 (Error)**：处理系统错误

## 贡献指南

欢迎提交 Issue 和 Pull Request。在提交 PR 前，请确保：

1. 代码符合 Go 语言规范
2. 添加了必要的测试
3. 更新了相关文档
4. 遵循现有的代码风格

## 许可证

[许可证类型]

## 相关文档

- [离线/在线引擎切换指南](docs/离线在线引擎切换指南.md) - **推荐阅读**
- [FunASR使用指南](docs/FunASR使用指南.md) - **默认ASR引擎**
- [TTS引擎推荐与对比](docs/TTS引擎推荐与对比.md) - **ChatTTS详细说明**
- [使用 Whisper 离线语音识别](docs/使用Whisper离线语音识别.md)
- [Sherpa-onnx TTS 指南](docs/sherpa_onnx_tts_guide.md)
- [ASR引擎升级方案](docs/ASR引擎升级方案.md)
- [语音助手性能优化总结](docs/语音助手性能优化总结.md)

## 🌟 核心特性

### 基础功能
- **完全离线运行**: 支持FunASR + ChatTTS + Ollama的完全离线配置
- **多引擎支持**: 灵活的语音识别、语音合成和大语言模型引擎配置
- **事件驱动架构**: 基于事件总线的松耦合设计
- **管道处理**: 流式音频处理管道，支持实时语音交互

### 🚀 自然语音交互功能 (默认启用!)

#### 1. 连续对话模式 ✅ 默认启用
- **智能唤醒**: 唤醒词激活后，可连续接收语音命令无需重复唤醒
- **自动休眠**: 静音检测后智能休眠，节省资源
- **无缝切换**: 自动在唤醒词模式、连续对话模式、可打断模式之间智能切换

#### 2. 并发音频处理 ✅ 默认启用
- **同时播放和录音**: 系统播放回复时可同时接收新的语音输入
- **智能打断**: 支持语音打断当前播放，实现自然的对话体验
- **多音频流管理**: 最多支持3个并发音频流，自动管理

#### 3. 智能上下文管理 ✅ 默认启用
- **对话历史**: 自动维护最近20轮对话历史
- **话题跟踪**: 智能识别和跟踪对话话题变化
- **用户偏好**: 学习和记录用户偏好，提供个性化体验
- **上下文压缩**: 为大语言模型自动提供精简的上下文信息

#### 4. 增强状态机 ✅ 默认启用
- **7种状态**: Idle、Listening、Processing、Speaking、Active、Concurrent、AwaitingInput
- **智能转换**: 基于事件和条件的智能状态转换
- **并发控制**: 支持多状态并发执行，自动优化性能

## 📁 项目结构

```
voice_assistant/
├── cmd/                           # 应用入口
│   ├── main.go                   # 主程序
│   └── app.go                    # 应用初始化
├── config/                       # 配置管理
│   ├── config.go                 # 配置结构和加载
│   └── config.yaml               # 配置文件
├── internal/
│   ├── common/                   # 公共组件
│   │   ├── states.go            # 增强状态定义
│   │   ├── conversation_manager.go  # 对话管理器
│   │   ├── events_enhanced.go   # 增强事件系统
│   │   ├── enhanced_test.go     # 功能测试
│   │   └── error_handler.go     # 错误处理
│   ├── pipeline/                 # 核心管道
│   │   ├── pipeline.go          # 主管道逻辑
│   │   └── state_machine.go  # 增强状态机
│   ├── eventbus/                 # 事件总线
│   ├── resource/                 # 资源管理
│   ├── asr/                      # 语音识别
│   ├── llm/                      # 大语言模型
│   ├── tts/                      # 语音合成
│   ├── kws/                      # 关键词检测
│   └── vad/                      # 语音活动检测
└── logs/                         # 日志目录
```

## 🛠️ 安装和使用

### 环境要求
- Go 1.19+
- Linux/Windows/macOS
- 音频设备支持

### 快速开始

1. **克隆项目**
```bash
git clone <repository-url>
cd voice_assistant
```

2. **安装依赖**
```bash
go mod tidy
```

3. **配置系统**
编辑 `config/config.yaml` 文件，配置各种引擎参数。

4. **编译运行**
```bash
go build -o voice_assistant cmd/main.go cmd/app.go
./voice_assistant
```

**注意**: 系统启动后会自动启用所有增强功能，无需手动配置。你将听到"语音助手已启动，连续对话模式已激活"的提示。

### 测试增强功能

运行功能测试：
```bash
go test ./internal/common -v
```

运行性能测试：
```bash
go test ./internal/common -bench=.
```

## 🎯 使用场景 (开箱即用!)

### 场景1: 连续对话 ✅ 自动启用
```
用户: "小助手"                    # 唤醒词
系统: "语音助手已启动，连续对话模式已激活。我在，请说"
用户: "今天天气怎么样？"           # 无需再次唤醒
系统: "今天天气晴朗..."
用户: "那明天呢？"               # 继续对话，自动维护上下文
系统: "明天预计..."
[静音2分钟后自动休眠]
```

### 场景2: 智能打断 ✅ 自动启用
```
用户: "小助手，播放一首音乐"
系统: [开始播放音乐]
用户: "停止播放"                 # 播放时直接说话，自动打断
系统: [立即停止播放] "好的，我停下来听您说。已停止播放"
```

### 场景3: 上下文理解 ✅ 自动启用
```
用户: "帮我查一下苹果的价格"
系统: "苹果手机的价格是..."
用户: "那安卓呢？"               # 系统自动理解指的是安卓手机
系统: "安卓手机的价格..."
[系统自动维护对话历史和话题上下文]
```

## 🔧 配置说明

### 默认增强功能配置

所有增强功能已在 `config/config.yaml` 中默认启用：

```yaml
# 连续对话配置 (默认启用)
conversation:
  enabled: true                   # ✅ 默认启用连续对话
  max_history_turns: 20           # 最大历史轮次
  session_timeout: "30m"          # 会话超时
  silence_timeout: "2m"           # 静音超时
  processing_timeout: "30s"       # 处理超时
  enable_smart_wakeup: true       # ✅ 智能唤醒
  enable_interrupt_mode: true     # ✅ 打断模式
  default_mode: "continuous"      # ✅ 默认连续对话模式

# 并发音频处理配置 (默认启用)
concurrent:
  enabled: true                   # ✅ 默认启用并发处理
  max_streams: 3                  # 最大并发流数
  buffer_size: 1024              # 音频缓冲区大小
  enable_interrupt: true          # ✅ 音频打断

# 上下文管理配置 (默认启用)
context:
  enabled: true                   # ✅ 默认启用上下文管理
  max_context_length: 2000       # 最大上下文长度
  topic_tracking: true            # ✅ 话题跟踪
  user_preference: true           # ✅ 用户偏好学习
```

**无需手动配置** - 系统启动后自动启用所有增强功能！

## 📊 性能指标

基于测试结果：
- **事件创建**: ~44ns/op (27M ops/sec)
- **事件优先级**: ~324ns/op (3.6M ops/sec)
- **状态转换**: <1ms
- **上下文查询**: <10ms
- **内存占用**: <50MB (空闲状态)

## 🔍 技术架构

### 增强状态机
```
StateIdle ──唤醒词──> StateListening ──语音检测──> StateProcessing
    ↑                      ↓                           ↓
    └──静音超时──── StateActive ←──处理完成──── StateSpeaking
                       ↓                           ↓
                 StateConcurrent ←──启用并发─── StateAwaitingInput
```

### 事件优先级系统
- **Critical (10)**: 系统错误、紧急事件
- **High (8)**: 唤醒词、打断请求、会话超时
- **Normal (5)**: ASR结果、LLM输出、TTS音频
- **Low (1)**: 上下文更新、偏好变更、统计指标

### 对话上下文结构
```go
type ConversationContext struct {
    SessionID     string
    History       []ConversationTurn  // 最近20轮对话
    CurrentTopic  string             // 当前话题
    UserProfile   UserProfile        // 用户偏好
    MaxHistory    int               // 最大历史数
}
```

## 🧪 测试覆盖

- ✅ 状态定义和转换测试
- ✅ 对话模式切换测试  
- ✅ 事件系统和优先级测试
- ✅ 对话管理器功能测试
- ✅ 性能基准测试
- ✅ 错误处理测试

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📜 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- FunASR: 语音识别引擎
- ChatTTS: 语音合成引擎  
- Ollama: 本地大语言模型
- Go社区: 优秀的开源库支持

---

**注意**: 这是一个持续开发的项目，新功能会不断添加。欢迎提出建议和贡献代码！

## 📈 监控和运维

### 监控体系

#### 服务端监控
```bash
# 启动监控模式
./scripts/deploy.sh
# 选择 2) 监控模式

# 访问监控界面
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3000 (admin/admin123)
```

#### 核心监控指标
- **系统资源**: CPU、内存、磁盘使用率
- **网络连接**: WebSocket连接数、消息吞吐量
- **业务指标**: 识别准确率、响应时间、错误率
- **引擎性能**: ASR/LLM/TTS处理延迟

#### 告警规则
```yaml
# Prometheus告警规则示例
groups:
  - name: voice_assistant_alerts
    rules:
      - alert: HighCPUUsage
        expr: cpu_usage > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "CPU使用率过高"
          
      - alert: WebSocketConnectionsHigh
        expr: websocket_connections > 50
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "WebSocket连接数异常"
```

### 日志管理

#### 日志级别
```yaml
# 服务端日志配置
logging:
  level: "info"          # debug, info, warn, error
  format: "json"         # text, json
  output: "stdout"       # stdout, file
  file_path: "logs/server.log"
  max_size: 10          # MB
  max_backups: 5
  max_age: 30           # 天
```

#### 日志查看
```bash
# 实时查看日志
docker-compose logs -f voice-assistant-server

# 查看错误日志
docker-compose logs voice-assistant-server | grep ERROR

# 导出日志
docker-compose logs > debug.log 2>&1
```

## 🔧 故障排查

### 常见问题

#### 1. 服务端启动失败

**症状**: 容器无法启动或健康检查失败
```bash
# 检查服务状态
docker-compose ps

# 查看详细日志
docker-compose logs voice-assistant-server

# 检查端口占用
sudo netstat -tulpn | grep :8080
```

**解决方案**:
- 确保Ollama服务已启动: `ollama serve`
- 检查配置文件语法: `./scripts/validate_config.sh`
- 清理旧容器: `docker-compose down && docker-compose up -d`

#### 2. 客户端连接失败

**症状**: 客户端无法连接到服务端
```bash
# 测试网络连通性
curl http://server_host:8080/health

# 测试WebSocket连接
wscat -c ws://server_host:8080/ws
```

**解决方案**:
- 检查防火墙设置: `sudo ufw status`
- 验证服务端地址配置
- 检查网络路由和DNS解析

#### 3. 音频处理问题

**症状**: 音频录制或播放异常
```bash
# 列出音频设备
./voice_assistant_client --devices

# 测试音频设备
arecord -l  # Linux
```

**解决方案**:
- 检查音频设备权限
- 调整VAD阈值设置
- 验证音频格式配置

#### 4. 引擎服务异常

**症状**: ASR/LLM/TTS引擎响应慢或失败

**FunASR问题**:
```bash
# 检查模型文件
ls -la models/funasr/

# 测试FunASR服务
python3 -c "import funasr; print('FunASR available')"
```

**Ollama问题**:
```bash
# 检查Ollama状态
ollama list
curl http://localhost:11434/api/version

# 重启Ollama
pkill ollama && ollama serve
```

**ChatTTS问题**:
```bash
# 检查ChatTTS安装
python3 -c "import ChatTTS; print('ChatTTS available')"

# 清理缓存
rm -rf ~/.cache/ChatTTS/
```

### 性能调优

#### 服务端优化
```yaml
# docker-compose.yml 资源限制调整
deploy:
  resources:
    limits:
      memory: 8G        # 根据实际需求调整
      cpus: '4.0'
    reservations:
      memory: 4G
      cpus: '2.0'
```

#### 客户端优化
```yaml
# client.yaml 性能配置
performance:
  audio_buffer_size: 8192      # 增大缓冲区
  max_concurrent_streams: 1    # 客户端限制并发
  worker_threads: 2           # 工作线程数
```

#### 网络优化
```yaml
# 服务端 WebSocket 配置
websocket:
  read_buffer_size: 4096
  write_buffer_size: 4096
  max_connections: 100
  ping_period: 30s
```

## 🔒 安全配置

### 网络安全
```bash
# 防火墙配置
sudo ufw allow 8080/tcp      # 服务端端口
sudo ufw deny 11434/tcp      # 保护Ollama端口

# 使用反向代理 (可选)
# Nginx/Traefik配置SSL/TLS
```

### 认证配置 (可选扩展)
```yaml
# 未来版本支持
security:
  auth:
    enabled: true
    type: "token"
    token: "your_secure_token"
```

## 📚 相关文档

### 部署文档
- [服务端部署指南](voice_assistant_server/DEPLOYMENT.md) - **详细部署说明**
- [客户端构建指南](voice_assistant_client/BUILD.md) - **跨平台构建**

### 技术文档
- [架构设计文档](SOLUTION.md) - **系统架构详解**
- [API接口文档](docs/API文档.md) - WebSocket协议说明
- [引擎配置指南](docs/引擎配置指南.md) - ASR/LLM/TTS配置

### 最佳实践
- [性能优化指南](docs/性能优化指南.md) - 系统调优建议
- [生产部署清单](docs/生产部署清单.md) - 上线前检查

## 🤝 贡献指南

### 开发环境
```bash
# 1. Fork并克隆项目
git clone your-fork-url
cd voice_assistant

# 2. 设置开发环境
cd voice_assistant_server && go mod download
cd ../voice_assistant_client && go mod download

# 3. 运行测试
make test

# 4. 提交代码
git checkout -b feature/your-feature
git commit -m "Add your feature"
git push origin feature/your-feature
```

### 代码规范
- Go代码遵循gofmt标准
- 提交信息使用conventional commits格式
- 添加必要的单元测试
- 更新相关文档

## 📜 许可证

本项目采用 [MIT 许可证](LICENSE)。

## 🙏 致谢

- **FunASR**: 阿里达摩院语音识别技术
- **ChatTTS**: 开源语音合成项目
- **Ollama**: 本地LLM部署方案
- **Go社区**: 优秀的开源库支持

---

**📧 联系方式**: 
- 项目主页: [GitHub Repository]
- 问题报告: [GitHub Issues]
- 讨论交流: [GitHub Discussions]

**⭐ 如果这个项目对您有帮助，请考虑给我们一个Star！**