# 语音助手系统 (Voice Assistant)

一个基于Go语言的企业级智能语音助手系统，采用客户端-服务端分离架构，支持实时语音交互、多引擎切换和完全离线部署。

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](voice_assistant_server/docker-compose.yml)

## 介绍

语音助手系统是一个现代化的语音交互解决方案，通过WebSocket实现客户端与服务端的实时通信。系统集成了语音识别(ASR)、大语言模型(LLM)对话、语音合成(TTS)的完整语音处理流程，支持连续对话、并发音频处理等智能特性。

**核心设计理念：**
- **轻量客户端**：音频输入输出、实时音频流处理
- **重型服务端**：ASR/LLM/TTS模型推理、会话管理
- **实时通信**：WebSocket双向音频流传输
- **离线优先**：支持完全离线部署，保护数据隐私

## 特性

### 🎯 核心功能
- **智能语音交互**：完整的语音识别、理解、合成流程
- **连续对话模式**：唤醒词激活后持续对话，无需重复唤醒
- **并发音频处理**：播放回复时同时接收新的语音输入
- **多引擎支持**：ASR/LLM/TTS引擎可灵活配置切换

### 🔧 技术特性
- **客户端-服务端分离**：架构清晰，职责明确
- **实时WebSocket通信**：低延迟双向音频流传输
- **跨平台支持**：Windows/Linux/macOS客户端
- **容器化部署**：Docker + Docker Compose一键部署

### 🌟 高级特性
- **完全离线运行**：支持FunASR + ChatTTS + Ollama离线配置
- **智能上下文管理**：维护对话历史，提供个性化体验
- **企业级监控**：Prometheus + Grafana监控体系
- **自动重连机制**：网络断开自动重连，会话无缝恢复

## 架构

### 系统架构图

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

### 项目结构

```
voice_assistant/
├── voice_assistant_server/          # 服务端 (Linux部署)
│   ├── cmd/server/                  # 服务端主程序
│   ├── internal/                    # 内部实现
│   │   ├── asr/                     # ASR模块 (FunASR, OpenAI)
│   │   ├── llm/                     # LLM模块 (Ollama, OpenAI, WebSocket)
│   │   ├── tts/                     # TTS模块 (ChatTTS, Edge-TTS)
│   │   └── server/                  # WebSocket服务器
│   ├── config/                      # 配置文件
│   └── docker-compose.yml          # 容器编排
├── voice_assistant_client/          # 客户端 (跨平台)
│   ├── cmd/client/                  # 客户端主程序
│   ├── internal/                    # 内部实现
│   │   ├── audio/                   # 音频处理
│   │   ├── client/                  # WebSocket客户端
│   │   └── ui/                      # 用户界面
│   ├── config/                      # 配置文件
│   └── Makefile                     # 跨平台构建
└── pkg/protocol/                    # 通信协议包
```

### 技术栈

**服务端**
- **框架**: Gin + WebSocket
- **ASR**: FunASR (默认), OpenAI Whisper  
- **LLM**: Ollama (默认), OpenAI GPT, WebSocket
- **TTS**: ChatTTS (默认), Edge-TTS
- **部署**: Docker + Docker Compose

**客户端**
- **音频**: PortAudio
- **通信**: Gorilla WebSocket
- **VAD**: 自实现语音活动检测
- **构建**: Go交叉编译

## 用法

### 快速开始

#### 1. 服务端部署

```bash
# 克隆项目
git clone <repository-url>
cd voice_assistant/voice_assistant_server

# 安装Ollama (外部依赖)
curl -fsSL https://ollama.ai/install.sh | sh
ollama serve
ollama pull qwen:7b  # 下载中文模型

# 一键部署服务端
./scripts/deploy.sh
# 选择: 1) 基础模式 或 2) 监控模式

# 验证部署
curl http://localhost:8080/health
```

#### 2. 客户端部署

**Linux/macOS用户**
```bash
cd voice_assistant/voice_assistant_client

# 安装依赖
sudo apt-get install portaudio19-dev  # Ubuntu
# 或 brew install portaudio  # macOS

# 构建运行
make build
./bin/voice_assistant_client
```

**Windows用户**
```bash
# 下载预编译版本
# 从 Releases 页面下载对应版本

# 解压并运行
voice_assistant_client.exe
```

### 配置说明

#### 服务端配置 (`config/server.yaml`)

```yaml
# 服务器配置
server:
  host: "0.0.0.0"
  port: 8080

# ASR引擎配置
asr:
  provider: "funasr"  # funasr/openai/whisper
  
# LLM引擎配置  
llm:
  provider: "ollama"  # ollama/openai/websocket
  ollama:
    base_url: "http://localhost:11434"
    model: "qwen:7b"
    
# TTS引擎配置
tts:
  provider: "chattts"  # chattts/edge-tts
```

#### 客户端配置 (`config/client.yaml`)

```yaml
# 服务器连接
server:
  host: "localhost"
  port: 8080
  websocket_path: "/ws"

# 音频配置
audio:
  input:
    sample_rate: 16000
    channels: 1
    format: "int16"
  
# 会话模式
session:
  mode: "continuous"  # continuous/wakeword/single
```

### 使用方式

1. **启动服务端**：`./scripts/deploy.sh`
2. **启动客户端**：`./bin/voice_assistant_client`
3. **语音交互**：
   - 说出唤醒词激活系统
   - 开始连续对话
   - 系统自动识别、理解、回复

## API

### WebSocket 通信协议

**连接地址**: `ws://localhost:8080/ws`

#### 消息类型

```go
// 消息类型
const (
    AudioStream = "audio_stream"  // 音频流
    Command     = "command"       // 控制命令
    Response    = "response"      // 服务端响应
    Status      = "status"        // 状态信息
    Error       = "error"         // 错误信息
)
```

#### 音频流消息

```json
{
    "type": "audio_stream",
    "session_id": "session_123",
    "timestamp": 1700000000000,
    "data": {
        "format": "pcm_16khz_16bit",
        "chunk_id": 1,
        "is_final": false,
        "audio_data": "base64编码的音频数据"
    }
}
```

#### 控制命令

```json
{
    "type": "command",
    "session_id": "session_123",
    "timestamp": 1700000000000,
    "data": {
        "command": "start_session",
        "mode": "continuous",
        "parameters": {}
    }
}
```

#### 服务端响应

```json
{
    "type": "response",
    "session_id": "session_123",
    "timestamp": 1700000000000,
    "data": {
        "stage": "asr",  // asr/llm/tts
        "content": "识别的文本内容",
        "confidence": 0.95,
        "is_final": true,
        "audio_data": "base64编码的音频数据"
    }
}
```

### HTTP API

#### 健康检查

```bash
GET /health
```

响应：
```json
{
    "status": "ok",
    "clients": 5,
    "timestamp": "1700000000"
}
```

#### 获取系统状态

```bash
GET /status
```

响应：
```json
{
    "server_status": "running",
    "active_sessions": 3,
    "total_processed": 1234,
    "uptime": "2h30m"
}
```

### 客户端API

#### 发送音频流

```go
client.SendAudioStream(audioData, chunkID, isFinal)
```

#### 发送控制命令

```go
client.SendCommand(command, mode, parameters)
```

#### 注册消息处理器

```go
client.RegisterHandler(protocol.Response, func(msg *protocol.Message) error {
    // 处理响应消息
    return nil
})
```

---

更多详细信息请参考：
- [服务端部署文档](voice_assistant_server/DEPLOYMENT.md)
- [客户端构建文档](voice_assistant_client/BUILD.md)
- [架构设计文档](SOLUTION.md)