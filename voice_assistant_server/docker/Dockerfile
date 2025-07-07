# 使用官方Go镜像作为构建环境
FROM golang:1.21-alpine AS builder

# 安装必要的系统依赖
RUN apk add --no-cache \
    gcc \
    musl-dev \
    pkgconfig \
    alsa-lib-dev \
    portaudio-dev

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o voice_assistant_server ./cmd/server

# 使用Alpine作为运行环境
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    alsa-lib \
    portaudio

# 创建非root用户
RUN adduser -D -s /bin/sh appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/voice_assistant_server .

# 复制配置文件
COPY config/ ./config/

# 创建必要的目录
RUN mkdir -p logs models && \
    chown -R appuser:appuser /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
CMD ["./voice_assistant_server"] 