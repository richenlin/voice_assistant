# 语音助手服务端部署指南

## 概述

本文档介绍如何使用Docker容器化部署语音助手服务端。本项目作为Ollama客户端，需要外部Ollama服务支持。

## 系统要求

### 硬件要求
- **CPU**: 2核心以上（推荐4核心）
- **内存**: 4GB以上（推荐8GB）
- **存储**: 20GB以上可用空间
- **网络**: 稳定的互联网连接

### 软件要求
- **操作系统**: Linux (Ubuntu 20.04+, CentOS 8+)
- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **Git**: 2.0+
- **外部Ollama服务**: 需要独立部署

## 前置条件

### 1. 安装并启动Ollama服务

```bash
# 安装Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# 启动Ollama服务
ollama serve

# 下载模型（例如qwen:7b）
ollama pull qwen:7b
```

### 2. 验证Ollama服务

```bash
# 测试Ollama API
curl http://localhost:11434/api/version

# 列出已安装的模型
ollama list
```

## 快速开始

### 1. 克隆项目

```bash
git clone <repository-url>
cd voice_assistant_server
```

### 2. 配置环境变量

```bash
# 复制环境变量模板
cp env.example .env

# 编辑配置文件
nano .env
```

### 3. 配置Ollama连接

编辑 `config/server.yaml`：

```yaml
llm:
  provider: "ollama"
  ollama:
    base_url: "http://localhost:11434"  # Ollama服务地址
    model: "qwen:7b"                   # 使用的模型
    timeout: 30
```

### 4. 一键部署

```bash
# 运行部署脚本
./scripts/deploy.sh
```

## 部署模式

### 基础模式 (推荐)

包含核心服务：
- 语音助手服务端

```bash
docker-compose up -d voice-assistant-server
```

### 监控模式

包含监控服务：
- 语音助手服务端
- Prometheus (监控)
- Grafana (可视化)

```bash
docker-compose --profile monitoring up -d
```

## 手动部署步骤

### 1. 准备环境

```bash
# 创建数据目录
mkdir -p data/{models,cache}
mkdir -p logs

# 设置权限
sudo chown -R 1000:1000 data/ logs/
```

### 2. 配置文件

#### 环境变量 (.env)
```env
# OpenAI API密钥（如果使用在线服务）
OPENAI_API_KEY=your_api_key_here

# 日志级别
LOG_LEVEL=info

# Grafana管理员密码
GRAFANA_PASSWORD=admin123

# 时区设置
TZ=Asia/Shanghai
```

#### 服务端配置 (config/server.yaml)
```yaml
server:
  host: "0.0.0.0"
  port: 8080
  
asr:
  provider: "funasr"
  
llm:
  provider: "ollama"
  ollama:
    base_url: "http://localhost:11434"
    model: "qwen:7b"
    timeout: 30
  
tts:
  provider: "chattts"
```

### 3. 构建和启动

```bash
# 构建镜像
docker-compose build

# 启动服务
docker-compose up -d

# 查看状态
docker-compose ps
```

## 服务访问

### 服务端点

| 服务 | 地址 | 说明 |
|------|------|------|
| 语音助手API | http://localhost:8080 | 主服务API |
| WebSocket | ws://localhost:8080/ws | 客户端连接 |
| 健康检查 | http://localhost:8080/health | 服务状态 |
| Prometheus | http://localhost:9090 | 监控数据 (监控模式) |
| Grafana | http://localhost:3000 | 监控面板 (监控模式) |

### 外部依赖

| 服务 | 地址 | 说明 |
|------|------|------|
| Ollama API | http://localhost:11434 | 外部LLM服务 |

### 默认凭据

- **Grafana**: admin / admin123

## 常用命令

### 服务管理

```bash
# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f voice-assistant-server

# 重启服务
docker-compose restart voice-assistant-server

# 停止所有服务
docker-compose down

# 停止并删除数据
docker-compose down -v
```

### 更新服务

```bash
# 拉取最新镜像
docker-compose pull

# 重新构建并启动
docker-compose up -d --build

# 清理旧镜像
docker image prune -f
```

### 备份和恢复

```bash
# 备份数据
tar -czf backup-$(date +%Y%m%d).tar.gz data/ logs/

# 恢复数据
tar -xzf backup-20240101.tar.gz
```

## 性能优化

### 资源限制

在 `docker-compose.yml` 中调整资源限制：

```yaml
deploy:
  resources:
    limits:
      memory: 4G
      cpus: '2.0'
    reservations:
      memory: 2G
      cpus: '1.0'
```

### 模型优化

1. **ASR模型**: 使用本地FunASR提高响应速度
2. **LLM模型**: 在外部Ollama服务中根据硬件选择合适的模型大小
3. **TTS模型**: 使用ChatTTS获得最佳音质

### Ollama优化

```bash
# 设置Ollama环境变量
export OLLAMA_NUM_PARALLEL=4
export OLLAMA_MAX_LOADED_MODELS=2
export OLLAMA_FLASH_ATTENTION=1

# 重启Ollama服务
ollama serve
```

## 故障排查

### 常见问题

#### 1. 服务无法启动

```bash
# 检查端口占用
sudo netstat -tulpn | grep :8080

# 检查Docker状态
sudo systemctl status docker

# 查看详细日志
docker-compose logs voice-assistant-server
```

#### 2. 无法连接Ollama

```bash
# 检查Ollama服务状态
curl http://localhost:11434/api/version

# 检查Ollama进程
ps aux | grep ollama

# 重启Ollama服务
ollama serve
```

#### 3. 内存不足

```bash
# 检查内存使用
docker stats

# 调整内存限制
# 编辑 docker-compose.yml 中的 memory 配置
```

#### 4. 模型加载失败

```bash
# 检查模型文件
ls -la data/models/

# 检查Ollama模型
ollama list

# 重新下载模型
ollama pull qwen:7b
```

### 日志分析

```bash
# 查看实时日志
docker-compose logs -f --tail=100

# 查看特定服务日志
docker-compose logs voice-assistant-server

# 导出日志
docker-compose logs > debug.log 2>&1
```

## 监控和告警

### Prometheus监控

访问 http://localhost:9090 查看监控数据

常用查询：
- 内存使用率: `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100`
- CPU使用率: `100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`
- 请求延迟: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`

### Grafana仪表板

1. 访问 http://localhost:3000
2. 使用 admin/admin123 登录
3. 导入预配置的仪表板

## 安全配置

### 防火墙设置

```bash
# 开放必要端口
sudo ufw allow 8080/tcp
```

### 网络安全

```bash
# 限制Ollama访问（如果需要）
sudo ufw allow from 172.20.0.0/16 to any port 11434
```

## 集成测试

### 完整测试流程

```bash
# 1. 测试Ollama连接
curl -X POST http://localhost:11434/api/generate \
  -H "Content-Type: application/json" \
  -d '{"model": "qwen:7b", "prompt": "Hello", "stream": false}'

# 2. 测试语音助手健康检查
curl http://localhost:8080/health

# 3. 测试WebSocket连接
# 使用客户端连接 ws://localhost:8080/ws
```

## 扩展部署

### 多实例部署

```yaml
# 多实例部署
voice-assistant-server:
  deploy:
    replicas: 3
    update_config:
      parallelism: 1
      delay: 10s
```

### 负载均衡

如果需要负载均衡，可以在前端添加Nginx或HAProxy。

## 维护计划

### 定期维护

1. **每日**: 检查服务状态和日志
2. **每周**: 清理旧日志和临时文件
3. **每月**: 更新镜像和安全补丁
4. **每季度**: 备份数据和配置

### 自动化脚本

```bash
# 创建维护脚本
cat > maintenance.sh << 'EOF'
#!/bin/bash
# 清理日志
find logs/ -name "*.log" -mtime +30 -delete
# 清理Docker
docker system prune -f
# 备份数据
tar -czf backup-$(date +%Y%m%d).tar.gz data/
EOF

# 添加到定时任务
crontab -e
# 0 2 * * 0 /path/to/maintenance.sh
```

## 支持和帮助

### 获取帮助

1. 查看项目文档: [README.md](README.md)
2. 检查问题跟踪: GitHub Issues
3. 社区支持: Discord/QQ群

### 贡献指南

1. Fork项目
2. 创建功能分支
3. 提交Pull Request
4. 参与代码审查

---

**注意**: 
- 本项目仅作为Ollama客户端，需要外部Ollama服务支持
- 请确保Ollama服务正常运行后再启动本服务
- 可根据实际情况调整配置参数 