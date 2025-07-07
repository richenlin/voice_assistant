# 语音助手客户端跨平台构建 Makefile

# 项目信息
PROJECT_NAME := voice-assistant-client
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 构建参数
GO := go
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
CGO_ENABLED := 1

# 构建目录
BUILD_DIR := build
DIST_DIR := dist

# 源码目录
CMD_DIR := cmd/client
MAIN_FILE := $(CMD_DIR)/main.go

# 构建标志
LDFLAGS := -X 'main.Version=$(VERSION)' \
           -X 'main.BuildTime=$(BUILD_TIME)' \
           -X 'main.GitCommit=$(GIT_COMMIT)' \
           -s -w

# 平台配置
PLATFORMS := \
	windows/amd64 \
	windows/386 \
	windows/arm64 \
	linux/amd64 \
	linux/386 \
	linux/arm64 \
	linux/arm \
	darwin/amd64 \
	darwin/arm64

# 默认目标
.PHONY: all
all: clean deps build

# 安装依赖
.PHONY: deps
deps:
	@echo "📦 安装依赖..."
	$(GO) mod download
	$(GO) mod tidy

# 本地构建
.PHONY: build
build:
	@echo "🔨 构建本地版本 ($(GOOS)/$(GOARCH))..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(PROJECT_NAME)$(shell if [ "$(GOOS)" = "windows" ]; then echo ".exe"; fi) \
		$(MAIN_FILE)
	@echo "✅ 构建完成: $(BUILD_DIR)/$(PROJECT_NAME)"

# 跨平台构建
.PHONY: build-all
build-all: clean deps
	@echo "🌍 开始跨平台构建..."
	@mkdir -p $(DIST_DIR)
	@$(foreach platform,$(PLATFORMS), \
		$(call build_platform,$(platform)) \
	)
	@echo "✅ 所有平台构建完成!"
	@echo "📁 构建文件位于: $(DIST_DIR)/"

# 构建特定平台
.PHONY: build-windows
build-windows: clean deps
	@$(call build_platform,windows/amd64)
	@$(call build_platform,windows/386)
	@$(call build_platform,windows/arm64)

.PHONY: build-linux
build-linux: clean deps
	@$(call build_platform,linux/amd64)
	@$(call build_platform,linux/386)
	@$(call build_platform,linux/arm64)
	@$(call build_platform,linux/arm)

.PHONY: build-darwin
build-darwin: clean deps
	@$(call build_platform,darwin/amd64)
	@$(call build_platform,darwin/arm64)

# 构建函数
define build_platform
	$(eval GOOS_TARGET := $(word 1,$(subst /, ,$(1))))
	$(eval GOARCH_TARGET := $(word 2,$(subst /, ,$(1))))
	$(eval OUTPUT_NAME := $(PROJECT_NAME)-$(VERSION)-$(GOOS_TARGET)-$(GOARCH_TARGET))
	$(eval EXT := $(if $(filter windows,$(GOOS_TARGET)),.exe,))
	@echo "🔨 构建 $(GOOS_TARGET)/$(GOARCH_TARGET)..."
	@mkdir -p $(DIST_DIR)/$(OUTPUT_NAME)
	@CGO_ENABLED=$(if $(filter windows darwin,$(GOOS_TARGET)),1,$(if $(filter linux,$(GOOS_TARGET)),1,0)) \
		GOOS=$(GOOS_TARGET) GOARCH=$(GOARCH_TARGET) \
		$(GO) build \
		-ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(OUTPUT_NAME)/$(PROJECT_NAME)$(EXT) \
		$(MAIN_FILE) || (echo "❌ 构建失败: $(GOOS_TARGET)/$(GOARCH_TARGET)" && exit 1)
	@cp -r config $(DIST_DIR)/$(OUTPUT_NAME)/
	@cp README.md $(DIST_DIR)/$(OUTPUT_NAME)/ 2>/dev/null || true
	@echo "📦 打包 $(OUTPUT_NAME)..."
	@cd $(DIST_DIR) && \
		$(if $(filter windows,$(GOOS_TARGET)), \
			zip -r $(OUTPUT_NAME).zip $(OUTPUT_NAME)/, \
			tar -czf $(OUTPUT_NAME).tar.gz $(OUTPUT_NAME)/ \
		)
	@rm -rf $(DIST_DIR)/$(OUTPUT_NAME)
	@echo "✅ 完成: $(DIST_DIR)/$(OUTPUT_NAME).$(if $(filter windows,$(GOOS_TARGET)),zip,tar.gz)"
endef

# 开发模式构建
.PHONY: dev
dev:
	@echo "🔧 开发模式构建..."
	CGO_ENABLED=1 $(GO) build -race -o $(BUILD_DIR)/$(PROJECT_NAME)-dev $(MAIN_FILE)

# 运行
.PHONY: run
run: build
	@echo "🚀 运行客户端..."
	@./$(BUILD_DIR)/$(PROJECT_NAME)

# 测试
.PHONY: test
test:
	@echo "🧪 运行测试..."
	$(GO) test -v ./...

# 基准测试
.PHONY: bench
bench:
	@echo "📊 运行基准测试..."
	$(GO) test -bench=. -benchmem ./...

# 代码检查
.PHONY: lint
lint:
	@echo "🔍 代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint 未安装，跳过代码检查"; \
		echo "安装命令: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# 格式化代码
.PHONY: fmt
fmt:
	@echo "📝 格式化代码..."
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

# 清理
.PHONY: clean
clean:
	@echo "🧹 清理构建文件..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)

# 安装到系统
.PHONY: install
install: build
	@echo "📥 安装到系统..."
	@sudo cp $(BUILD_DIR)/$(PROJECT_NAME) /usr/local/bin/
	@echo "✅ 安装完成: /usr/local/bin/$(PROJECT_NAME)"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "🗑️  从系统卸载..."
	@sudo rm -f /usr/local/bin/$(PROJECT_NAME)
	@echo "✅ 卸载完成"

# 生成发布包
.PHONY: release
release: clean build-all
	@echo "🎁 生成发布包..."
	@mkdir -p $(DIST_DIR)/release
	@cp -r $(DIST_DIR)/*.zip $(DIST_DIR)/*.tar.gz $(DIST_DIR)/release/ 2>/dev/null || true
	@echo "✅ 发布包生成完成: $(DIST_DIR)/release/"

# 显示帮助
.PHONY: help
help:
	@echo "语音助手客户端构建工具"
	@echo ""
	@echo "可用命令:"
	@echo "  all          - 清理、安装依赖、构建本地版本"
	@echo "  build        - 构建本地版本"
	@echo "  build-all    - 构建所有平台版本"
	@echo "  build-windows- 构建Windows版本"
	@echo "  build-linux  - 构建Linux版本"
	@echo "  build-darwin - 构建macOS版本"
	@echo "  dev          - 开发模式构建（包含race检测）"
	@echo "  run          - 构建并运行"
	@echo "  test         - 运行测试"
	@echo "  bench        - 运行基准测试"
	@echo "  lint         - 代码检查"
	@echo "  fmt          - 格式化代码"
	@echo "  clean        - 清理构建文件"
	@echo "  install      - 安装到系统"
	@echo "  uninstall    - 从系统卸载"
	@echo "  release      - 生成发布包"
	@echo "  deps         - 安装依赖"
	@echo "  help         - 显示此帮助"
	@echo ""
	@echo "示例:"
	@echo "  make build          # 构建本地版本"
	@echo "  make build-all      # 构建所有平台"
	@echo "  make build-windows  # 只构建Windows版本"
	@echo "  make release        # 生成发布包"

# 显示构建信息
.PHONY: info
info:
	@echo "📋 构建信息:"
	@echo "  项目名称: $(PROJECT_NAME)"
	@echo "  版本: $(VERSION)"
	@echo "  构建时间: $(BUILD_TIME)"
	@echo "  Git提交: $(GIT_COMMIT)"
	@echo "  Go版本: $(shell $(GO) version)"
	@echo "  目标平台: $(GOOS)/$(GOARCH)"
	@echo "  CGO: $(CGO_ENABLED)" 