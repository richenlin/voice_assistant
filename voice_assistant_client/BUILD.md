# 语音助手客户端构建指南

## 概述

本文档介绍如何使用Make进行语音助手客户端的跨平台交叉编译。

## 系统要求

### 开发环境
- **Go**: 1.21+
- **Make**: 3.81+
- **Git**: 2.0+

### 支持平台
- **Windows**: amd64, 386, arm64
- **Linux**: amd64, 386, arm64, arm
- **macOS**: amd64, arm64

## 快速开始

### 1. 克隆项目

```bash
git clone <repository-url>
cd voice_assistant_client
```

### 2. 安装依赖

```bash
make deps
```

### 3. 构建本地版本

```bash
make build
```

## 构建命令

### 基础构建

```bash
# 构建本地版本
make build

# 开发模式构建（包含race检测）
make dev

# 构建并运行
make run
```

### 跨平台构建

```bash
# 构建所有平台
make build-all

# 构建Windows平台
make build-windows

# 构建Linux平台
make build-linux

# 构建macOS平台
make build-darwin
```

### 发布构建

```bash
# 生成发布包
make release
```

## 构建输出

### 本地构建

构建文件位于 `build/` 目录：
- `build/voice-assistant-client` (Linux/macOS)
- `build/voice-assistant-client.exe` (Windows)

### 跨平台构建

构建文件位于 `dist/` 目录：
- `dist/voice-assistant-client-v1.0.0-windows-amd64.zip`
- `dist/voice-assistant-client-v1.0.0-linux-amd64.tar.gz`
- `dist/voice-assistant-client-v1.0.0-darwin-amd64.tar.gz`
- ...

### 发布包

发布包位于 `dist/release/` 目录，包含所有平台的构建文件。

## 开发工具

### 代码质量

```bash
# 代码格式化
make fmt

# 代码检查
make lint

# 运行测试
make test

# 基准测试
make bench
```

### 清理

```bash
# 清理构建文件
make clean
```

## 版本信息

构建时会自动嵌入版本信息：
- **版本号**: 从Git标签获取
- **构建时间**: 构建时的UTC时间
- **Git提交**: 当前提交的短哈希

## 构建选项

### 环境变量

可以通过环境变量自定义构建：

```bash
# 自定义版本号
VERSION=v2.0.0 make build

# 自定义构建标志
LDFLAGS="-X main.CustomFlag=value" make build

# 禁用CGO
CGO_ENABLED=0 make build
```

### 构建标志

默认构建标志：
- `-s -w`: 压缩二进制文件
- `-X main.Version`: 嵌入版本信息
- `-X main.BuildTime`: 嵌入构建时间
- `-X main.GitCommit`: 嵌入Git提交

## 故障排查

### 常见问题

#### 1. Go环境问题

```bash
# 检查Go版本
go version

# 检查Go环境
go env

# 更新Go模块
go mod tidy
```

#### 2. 交叉编译问题

```bash
# 检查支持的平台
go tool dist list

# 清理模块缓存
go clean -modcache

# 重新下载依赖
go mod download
```

#### 3. CGO依赖问题

某些平台可能需要CGO支持（如音频处理）：

```bash
# 安装交叉编译工具链
# Ubuntu/Debian
sudo apt-get install gcc-mingw-w64

# macOS
brew install mingw-w64
```

### 构建日志

查看详细构建日志：

```bash
# 详细模式
make build V=1

# 查看构建信息
make info
```

## 部署

### 本地安装

```bash
# 安装到系统
make install

# 卸载
make uninstall
```

### 分发

1. 使用 `make release` 生成发布包
2. 上传到发布平台（GitHub Releases等）
3. 用户下载对应平台的包

## 持续集成

### GitHub Actions

```yaml
name: Build
on: [push, pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: 1.21
    - name: Build
      run: make build-all
    - name: Test
      run: make test
```

### 本地CI

```bash
# 完整的CI流程
make clean deps fmt lint test build-all
```

## 自定义构建

### 修改Makefile

根据项目需求修改 `Makefile`：

1. 添加新的平台支持
2. 自定义构建标志
3. 添加构建后处理

### 示例：添加新平台

```makefile
# 添加FreeBSD支持
PLATFORMS += freebsd/amd64
```

## 帮助

### 查看所有命令

```bash
make help
```

### 查看构建信息

```bash
make info
```

## 最佳实践

1. **版本管理**: 使用Git标签管理版本
2. **自动化**: 使用CI/CD自动化构建和发布
3. **测试**: 构建前运行测试确保代码质量
4. **文档**: 保持构建文档的更新

---

**注意**: 确保在构建前安装所有必要的依赖和工具链。 