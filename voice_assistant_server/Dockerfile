# 多阶段构建 - 构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的系统依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# 运行阶段
FROM python:3.11-slim

# 设置环境变量
ENV PYTHONUNBUFFERED=1
ENV DEBIAN_FRONTEND=noninteractive

# 安装系统依赖
RUN apt-get update && apt-get install -y \
    wget \
    curl \
    git \
    build-essential \
    ffmpeg \
    libsndfile1 \
    && rm -rf /var/lib/apt/lists/*

# 创建应用用户
RUN useradd -m -u 1000 app

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/server /app/server

# 复制配置文件和其他资源
COPY --from=builder /app/config /app/config
COPY --from=builder /app/pkg /app/pkg

# 安装Python依赖
RUN pip install --no-cache-dir \
    funasr \
    torch \
    torchaudio \
    librosa \
    soundfile \
    requests

# 安装ChatTTS
RUN git clone https://github.com/2noise/ChatTTS.git /tmp/ChatTTS && \
    cd /tmp/ChatTTS && \
    pip install -e . && \
    rm -rf /tmp/ChatTTS

# 创建必要的目录
RUN mkdir -p /app/models/funasr /app/models/chattts /app/logs

# 设置权限
RUN chown -R app:app /app
USER app

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 启动命令
CMD ["./server", "-config", "config/server.yaml"] 