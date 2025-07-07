#!/bin/bash

# 语音助手服务端Docker部署脚本

set -e

echo "🚀 开始部署语音助手服务端..."

# 检查Docker环境
if ! command -v docker &> /dev/null; then
    echo "❌ 错误: 未找到Docker，请先安装Docker"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ 错误: 未找到docker-compose，请先安装docker-compose"
    exit 1
fi

# 创建必要的目录
echo "📁 创建数据目录..."
mkdir -p data/{models,cache}
mkdir -p logs

# 设置权限
sudo chown -R 1000:1000 data/ logs/

# 检查环境变量文件
if [ ! -f ".env" ]; then
    if [ -f "env.example" ]; then
        echo "📋 复制环境变量配置文件..."
        cp env.example .env
        echo "⚠️  请编辑 .env 文件配置必要的环境变量"
    else
        echo "⚠️  未找到环境变量配置文件，使用默认配置"
    fi
fi

# 选择部署模式
echo "请选择部署模式："
echo "1) 基础模式 (仅语音助手服务端)"
echo "2) 监控模式 (包含Prometheus和Grafana)"
read -p "请输入选择 (1-2): " deploy_mode

case $deploy_mode in
    1)
        echo "🔧 部署基础模式..."
        docker-compose up -d voice-assistant-server
        ;;
    2)
        echo "🔧 部署监控模式..."
        docker-compose --profile monitoring up -d
        ;;
    *)
        echo "🔧 使用默认基础模式..."
        docker-compose up -d voice-assistant-server
        ;;
esac

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

# 检查服务状态
echo "🔍 检查服务状态..."
docker-compose ps

# 健康检查
echo "🏥 执行健康检查..."
if curl -f http://localhost:8080/health >/dev/null 2>&1; then
    echo "✅ 语音助手服务端启动成功！"
else
    echo "❌ 语音助手服务端健康检查失败"
    echo "📋 查看日志:"
    docker-compose logs voice-assistant-server
    exit 1
fi

echo ""
echo "🎉 部署完成！"
echo ""
echo "📊 服务信息："
echo "  - 语音助手服务端: http://localhost:8080"
echo "  - WebSocket连接: ws://localhost:8080/ws"

if [ "$deploy_mode" = "2" ]; then
    echo "  - Prometheus: http://localhost:9090"
    echo "  - Grafana: http://localhost:3000 (admin/admin123)"
fi

echo ""
echo "⚠️  注意事项："
echo "  - 请确保Ollama服务已在外部启动并可访问"
echo "  - 默认配置中Ollama地址为: http://localhost:11434"
echo "  - 可在config/server.yaml中修改Ollama配置"
echo ""
echo "🔧 常用命令："
echo "  - 查看日志: docker-compose logs -f"
echo "  - 重启服务: docker-compose restart"
echo "  - 停止服务: docker-compose down"
echo "  - 更新镜像: docker-compose pull && docker-compose up -d"
echo ""
echo "📖 更多信息请查看 README.md" 