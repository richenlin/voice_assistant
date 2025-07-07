# 语音助手客户端 (Windows)

## 📝 项目简介

语音助手客户端是一个轻量化的Windows应用程序，通过WebSocket连接到语音助手服务端，提供语音输入和音频播放功能。支持实时语音识别、智能对话和语音合成。

## 🚀 核心功能

- **语音输入** - 实时麦克风音频采集和VAD检测
- **音频播放** - 高质量音频输出和播放控制
- **实时通信** - WebSocket连接，低延迟数据传输
- **自动重连** - 网络断开自动重连机制
- **简单易用** - 一键启动，无需复杂配置

## 🏗️ 系统要求

### 最低要求
- **操作系统**: Windows 10 或更高版本
- **内存**: 512MB RAM
- **存储**: 50MB 可用空间
- **网络**: 稳定的网络连接

### 推荐配置
- **操作系统**: Windows 11
- **内存**: 2GB RAM 或更高
- **音频设备**: 高质量麦克风和扬声器
- **网络**: 宽带连接

## 📦 安装部署

### 方式一：预编译版本

```bash
# 1. 下载最新版本
wget https://github.com/your-org/voice_assistant_client/releases/latest/voice_assistant_client.exe

# 2. 直接运行
voice_assistant_client.exe
```

### 方式二：源码编译

```bash
# 1. 安装Go环境
# 下载并安装Go 1.21+: https://golang.org/dl/

# 2. 安装PortAudio
# 下载并安装PortAudio: http://www.portaudio.com/

# 3. 克隆项目
git clone <repository-url>
cd voice_assistant_client

# 4. 构建项目
scripts\build_windows.bat

# 5. 运行程序
voice_assistant_client.exe
```

### 方式三：安装包

```bash
# 1. 下载MSI安装包
voice_assistant_client_setup.msi

# 2. 运行安装程序
# 按照向导完成安装

# 3. 从开始菜单启动
# 或桌面快捷方式
```

## ⚙️ 配置说明

### 配置文件位置

- **用户配置**: `%APPDATA%\VoiceAssistant\client.yaml`
- **系统配置**: `%PROGRAMFILES%\VoiceAssistant\config\client.yaml`

### 基本配置

```yaml
server:
  host: "localhost"  # 服务端地址
  port: 8080         # 服务端端口
  use_tls: false     # 是否使用HTTPS/WSS

audio:
  input_device: "default"   # 输入设备
  output_device: "default"  # 输出设备
  sample_rate: 16000        # 采样率
  channels: 1               # 声道数
```

### 高级配置

```yaml
session:
  mode: "continuous"        # 连续对话模式
  auto_reconnect: true      # 自动重连
  timeout: 30m              # 会话超时

ui:
  type: "console"           # 界面类型
  log_level: "info"         # 日志级别
  show_audio_level: true    # 显示音频电平

windows:
  audio_driver: "wasapi"    # 音频驱动
  system_tray: true         # 系统托盘
  auto_start: false         # 开机自启
```

## 🎯 使用指南

### 快速开始

1. **启动程序**
   ```bash
   voice_assistant_client.exe
   ```

2. **连接服务器**
   - 程序自动连接到默认服务器
   - 看到"连接成功"提示

3. **开始对话**
   - 直接开始说话
   - 系统自动检测语音
   - 实时显示识别结果

4. **退出程序**
   - 按 `Ctrl+C` 退出
   - 或关闭控制台窗口

### 命令行参数

```bash
# 指定服务器地址
voice_assistant_client.exe --server ws://192.168.1.100:8080/ws

# 指定配置文件
voice_assistant_client.exe --config custom_config.yaml

# 调试模式
voice_assistant_client.exe --debug

# 显示版本信息
voice_assistant_client.exe --version

# 显示帮助
voice_assistant_client.exe --help
```

### 快捷键

- `Ctrl+C` - 退出程序
- `Space` - 手动触发语音识别
- `M` - 静音/取消静音
- `R` - 重新连接服务器
- `S` - 显示状态信息

## 🔧 音频设备配置

### 查看可用设备

```bash
# 启动程序时会显示可用设备
voice_assistant_client.exe --list-devices
```

### 设备选择

```yaml
audio:
  # 按名称选择
  input_device: "Microphone (Realtek Audio)"
  output_device: "Speakers (Realtek Audio)"
  
  # 按索引选择
  input_device_index: 0
  output_device_index: 1
```

### 音频优化

```yaml
audio:
  # 缓冲区大小 (影响延迟)
  buffer_size: 1024
  
  # VAD敏感度
  vad:
    threshold: 0.01      # 越小越敏感
    min_speech_frames: 10
    max_silence_frames: 50
```

## 🐛 故障排查

### 常见问题

1. **无法连接服务器**
   ```
   错误: 连接服务器失败
   解决: 检查服务器地址和端口是否正确
        检查网络连接是否正常
        确认服务端是否正在运行
   ```

2. **音频设备问题**
   ```
   错误: 初始化音频失败
   解决: 检查音频设备是否被其他程序占用
        尝试更换音频设备
        重新安装音频驱动
   ```

3. **权限问题**
   ```
   错误: 访问被拒绝
   解决: 以管理员身份运行程序
        检查防火墙设置
        确认程序有麦克风权限
   ```

### 日志分析

```bash
# 查看日志文件
type %APPDATA%\VoiceAssistant\logs\client.log

# 实时查看日志
tail -f %APPDATA%\VoiceAssistant\logs\client.log
```

### 调试模式

```bash
# 启用详细日志
voice_assistant_client.exe --log-level debug

# 保存调试信息
voice_assistant_client.exe --debug > debug.log 2>&1
```

## 🔒 安全配置

### 网络安全

```yaml
server:
  use_tls: true                    # 启用HTTPS/WSS
  verify_certificate: true        # 验证服务器证书
  
security:
  auth_token: "your-auth-token"    # 认证令牌
  encrypt_audio: true              # 音频数据加密
```

### 隐私保护

```yaml
privacy:
  local_processing: false          # 本地处理模式
  data_retention: "none"           # 数据保留策略
  anonymous_mode: true             # 匿名模式
```

## 📊 性能监控

### 系统监控

```yaml
monitoring:
  enable_metrics: true
  metrics_port: 9091
  
  # 性能指标
  track_audio_latency: true
  track_network_latency: true
  track_memory_usage: true
```

### 性能优化

```yaml
performance:
  # 音频优化
  audio_buffer_count: 3
  audio_thread_priority: "high"
  
  # 网络优化
  websocket_buffer_size: 8192
  compression: true
  
  # 内存优化
  gc_percent: 100
  max_memory: "500MB"
```

## 🎨 界面定制

### 控制台界面

```yaml
ui:
  console:
    color_scheme: "dark"         # dark|light
    show_timestamps: true
    show_session_id: false
    animation: true
```

### 图形界面 (可选)

```yaml
ui:
  gui:
    theme: "modern"              # modern|classic
    window_size: "800x600"
    minimize_to_tray: true
    notifications: true
```

## 📦 打包分发

### 创建安装包

```bash
# 1. 构建可执行文件
scripts\build_windows.bat

# 2. 创建安装程序
scripts\create_installer.bat

# 3. 生成MSI包
scripts\build_msi.bat
```

### 便携版

```bash
# 1. 创建便携版目录
mkdir voice_assistant_portable

# 2. 复制必要文件
copy voice_assistant_client.exe voice_assistant_portable\
copy config\client.yaml voice_assistant_portable\config\

# 3. 创建启动脚本
echo @echo off > voice_assistant_portable\start.bat
echo voice_assistant_client.exe --config config\client.yaml >> voice_assistant_portable\start.bat
```

## 🔄 更新升级

### 自动更新

```yaml
update:
  auto_check: true               # 自动检查更新
  check_interval: "24h"          # 检查间隔
  auto_download: false           # 自动下载
  update_channel: "stable"       # stable|beta|dev
```

### 手动更新

```bash
# 1. 下载最新版本
wget https://github.com/your-org/voice_assistant_client/releases/latest

# 2. 备份配置
copy %APPDATA%\VoiceAssistant\client.yaml client_backup.yaml

# 3. 替换可执行文件
# 4. 恢复配置文件
```

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🤝 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📞 技术支持

- 📧 邮箱: support@example.com
- 💬 QQ群: 123456789
- 📖 文档: https://docs.example.com
- 🐛 问题反馈: https://github.com/example/issues

## 📋 更新日志

### v1.0.0 (2024-01-01)
- ✨ 首次发布
- 🎤 支持实时语音输入
- 🔊 支持音频播放
- 🌐 WebSocket通信
- 🔄 自动重连机制

### v1.1.0 (计划中)
- 🎨 图形用户界面
- 🔧 更多音频设备支持
- �� 性能监控面板
- 🔒 增强安全功能 