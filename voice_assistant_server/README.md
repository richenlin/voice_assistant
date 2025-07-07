# 语音助手服务端

这是语音助手的服务端组件，负责处理ASR（语音识别）、LLM（大语言模型）、TTS（语音合成）等重型计算任务。

## 功能特性

- **WebSocket通信**：支持多客户端并发连接
- **ASR支持**：集成Whisper、OpenAI Whisper API
- **LLM支持**：集成OpenAI GPT、Ollama本地模型、WebSocket LLM
- **TTS支持**：集成Edge-TTS、Sherpa-ONNX
- **会话管理**：支持连续对话和上下文管理
- **实时处理**：支持音频流实时处理
- **配置灵活**：支持YAML配置文件和环境变量

## 系统要求

- Go 1.19+
- Linux系统（推荐Ubuntu 20.04+）
- 内存：建议2GB+
- 存储：建议10GB+（用于模型文件）

## 快速开始

### 1. 构建应用

```bash
# 使用构建脚本
./scripts/build.sh

# 或手动构建
go build -o bin/server cmd/server/main.go
```

### 2. 配置服务

复制并编辑配置文件：

```bash
cp config/server.yaml config/server.yaml.local
```

主要配置项：

```yaml
# 服务器配置
server:
  host: "0.0.0.0"
  port: 8080

# ASR配置
asr:
  provider: "whisper"  # whisper|openai
  whisper:
    model_path: "./models/whisper/ggml-base.bin"
    language: "zh"
  openai:
    api_key: "${OPENAI_API_KEY}"

# LLM配置
llm:
  provider: "openai"  # openai|ollama|websocket
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"

# TTS配置
tts:
  provider: "edge_tts"  # edge_tts|sherpa
  edge_tts:
    voice: "zh-CN-XiaoxiaoNeural"
```

### 3. 运行服务

```bash
# 使用默认配置
./bin/server

# 使用自定义配置
./bin/server -config config/server.yaml.local

# 后台运行
nohup ./bin/server -config config/server.yaml.local > server.log 2>&1 &
```

## 模型准备

### Whisper模型

下载Whisper模型文件：

```bash
# 创建模型目录
mkdir -p models/whisper

# 下载base模型（推荐）
wget -O models/whisper/ggml-base.bin \
  https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin

# 或下载其他大小的模型
# tiny: https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin
# small: https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin
# medium: https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin
# large: https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin
```

### Ollama模型

如果使用Ollama，需要先安装并下载模型：

```bash
# 安装Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# 下载模型
ollama pull llama2
ollama pull qwen:7b
```

## API接口

### WebSocket连接

```
ws://localhost:8080/ws?session_id=your_session_id
```

### 健康检查

```
GET http://localhost:8080/health
```

响应：
```json
{
  "status": "ok",
  "clients": 2,
  "timestamp": "8080"
}
```

## 消息协议

### 音频流消息

```json
{
  "type": "audio_stream",
  "session_id": "session_123",
  "timestamp": 1234567890,
  "data": {
    "audio_data": "base64_encoded_audio",
    "sample_rate": 16000,
    "channels": 1,
    "format": "pcm_s16le",
    "is_final": false
  }
}
```

### 命令消息

```json
{
  "type": "command",
  "session_id": "session_123",
  "timestamp": 1234567890,
  "data": {
    "command": "start_session",
    "parameters": {
      "continuous_mode": true
    }
  }
}
```

### 响应消息

```json
{
  "type": "response",
  "session_id": "session_123",
  "timestamp": 1234567890,
  "data": {
    "stage": "llm_response",
    "content": "你好，我是语音助手",
    "confidence": 0.95,
    "is_final": true,
    "audio_data": "base64_encoded_audio"
  }
}
```

## 部署指南

### Docker部署

```bash
# 构建镜像
docker build -t voice-assistant-server .

# 运行容器
docker run -d \
  --name voice-assistant-server \
  -p 8080:8080 \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/models:/app/models \
  -e OPENAI_API_KEY=your_api_key \
  voice-assistant-server
```

### 系统服务

创建systemd服务文件：

```bash
sudo tee /etc/systemd/system/voice-assistant-server.service > /dev/null <<EOF
[Unit]
Description=Voice Assistant Server
After=network.target

[Service]
Type=simple
User=voice-assistant
WorkingDirectory=/opt/voice-assistant-server
ExecStart=/opt/voice-assistant-server/bin/server -config /opt/voice-assistant-server/config/server.yaml
Restart=always
RestartSec=5
Environment=OPENAI_API_KEY=your_api_key

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable voice-assistant-server
sudo systemctl start voice-assistant-server
```

## 性能优化

### 系统级优化

```bash
# 增加文件描述符限制
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# 优化网络参数
echo "net.core.somaxconn = 1024" >> /etc/sysctl.conf
echo "net.core.netdev_max_backlog = 5000" >> /etc/sysctl.conf
sysctl -p
```

### 应用级优化

在配置文件中调整：

```yaml
websocket:
  max_connections: 100
  read_buffer_size: 4096
  write_buffer_size: 4096

# 根据硬件调整模型配置
asr:
  whisper:
    model_size: "base"  # tiny|base|small|medium|large
    
llm:
  openai:
    max_tokens: 1000  # 减少token数量提高响应速度
```

## 故障排查

### 常见问题

1. **WebSocket连接失败**
   - 检查防火墙设置
   - 确认端口是否被占用
   - 检查服务器日志

2. **ASR识别错误**
   - 确认模型文件路径正确
   - 检查音频格式（需要16kHz PCM）
   - 验证API密钥配置

3. **LLM响应慢**
   - 调整max_tokens参数
   - 考虑使用本地模型
   - 检查网络连接

4. **TTS合成失败**
   - 检查网络连接（Edge-TTS需要联网）
   - 验证声音ID是否正确
   - 检查模型文件（Sherpa-ONNX）

### 日志查看

```bash
# 查看实时日志
tail -f server.log

# 查看系统服务日志
sudo journalctl -u voice-assistant-server -f

# 查看错误日志
grep ERROR server.log
```

## 开发指南

### 项目结构

```
voice_assistant_server/
├── cmd/server/          # 主程序入口
├── internal/
│   ├── asr/            # ASR模块
│   ├── llm/            # LLM模块
│   ├── tts/            # TTS模块
│   ├── server/         # 服务器模块
│   └── config/         # 配置模块
├── pkg/protocol/       # 通信协议
├── config/             # 配置文件
├── scripts/            # 构建脚本
└── README.md
```

### 添加新的ASR/LLM/TTS提供商

1. 在对应模块实现接口
2. 在init函数中注册工厂函数
3. 更新配置结构
4. 添加相应的测试

### 环境变量

支持的环境变量：

- `OPENAI_API_KEY`: OpenAI API密钥
- `CONFIG_PATH`: 配置文件路径
- `LOG_LEVEL`: 日志级别

## 许可证

MIT License 