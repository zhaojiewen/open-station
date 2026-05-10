.PHONY: all install start stop restart test clean docker-up docker-down docker-build migrate deps lint fmt help build run logs status mcp-config release release-local version download install-binary uninstall upgrade

# 默认目标
all: help

# 帮助
help:
	@echo "=========================================="
	@echo "   Open Station - 企业级AI网关"
	@echo "=========================================="
	@echo ""
	@echo "使用方法: make [目标]"
	@echo ""
	@echo "快速开始:"
	@echo "  install      - 交互式安装（本地开发）"
	@echo "  start        - 一键启动（Docker，推荐）"
	@echo "  stop         - 停止服务"
	@echo "  restart      - 重启服务"
	@echo ""
	@echo "发布与安装:"
	@echo "  release      - 构建发布版本（指定 VERSION=x.x.x）"
	@echo "  release-local - 本地构建多平台二进制"
	@echo "  download      - 下载预编译二进制"
	@echo "  install-binary - 安装二进制到系统"
	@echo "  uninstall     - 从系统卸载"
	@echo "  upgrade       - 升级到最新版本"
	@echo "  version       - 显示版本信息"
	@echo ""
	@echo "MCP 配置:"
	@echo "  mcp-config      - 配置 Claude Code CLI MCP"
	@echo "  mcp-config-cursor - 配置 Cursor IDE MCP"
	@echo "  mcp-config-vscode - 配置 VS Code MCP"
	@echo ""
	@echo "开发:"
	@echo "  build        - 编译服务"
	@echo "  run          - 运行服务（本地）"
	@echo "  test         - 运行测试"
	@echo "  lint         - 代码检查"
	@echo "  fmt          - 格式化代码"
	@echo ""
	@echo "Docker:"
	@echo "  install-docker - 自动安装 Docker"
	@echo "  docker-up    - 启动 Docker 服务"
	@echo "  docker-down  - 停止 Docker 服务"
	@echo "  docker-build - 构建 Docker 镜像"
	@echo ""
	@echo "其他:"
	@echo "  logs         - 查看日志"
	@echo "  status       - 查看服务状态"
	@echo "  clean        - 清理构建文件"
	@echo "  deps         - 安装依赖"
	@echo "  migrate      - 运行数据库迁移"
	@echo ""

# 快速安装
install:
	@chmod +x scripts/*.sh
	@./scripts/quick-install.sh

# Docker 启动
start:
	@chmod +x scripts/*.sh
	@./scripts/start-docker.sh

# 停止服务
stop:
	@if [ -f scripts/stop.sh ]; then ./scripts/stop.sh; else docker-compose -f deployments/docker/docker-compose.yml down; fi

# 重启
restart: stop start

# 构建
build:
	@go build -o bin/server ./cmd/server

# 运行（本地）
run:
	@go run ./cmd/server -config configs/config.yaml

# 测试
test:
	@go test -v ./...

# 测试覆盖率
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	@go test ./pkg/... ./internal/domain/... ./internal/infrastructure/auth/... ./internal/application/service/... ./internal/interfaces/http/middleware/... -coverprofile=coverage.out
	@go tool cover -func=coverage.out | tail -1
	@echo "生成HTML覆盖率报告: coverage.html"
	@go tool cover -html=coverage.out -o coverage.html

# 详细测试覆盖率
test-coverage-detail:
	@go test ./... -coverprofile=full_coverage.out 2>&1 | grep -E "(ok|FAIL|coverage)"
	@go tool cover -func=full_coverage.out

# 代码检查
lint:
	@golangci-lint run ./...

# 格式化
fmt:
	@go fmt ./...

# Docker 服务
install-docker:
	@chmod +x scripts/*.sh
	@./scripts/install-docker.sh

docker-up:
	@docker-compose -f deployments/docker/docker-compose.yml up -d

docker-down:
	@docker-compose -f deployments/docker/docker-compose.yml down

docker-build:
	@docker-compose -f deployments/docker/docker-compose.yml build

# 数据库迁移
migrate:
	@go run ./cmd/migrate -config configs/config.yaml

# 清理
clean:
	@rm -rf bin/ server.log server.pid
	@docker-compose -f deployments/docker/docker-compose.yml down -v

# 依赖
deps:
	@go mod download
	@go mod tidy

# MCP 配置
mcp-config:
	@chmod +x scripts/*.sh
	@./scripts/setup-mcp.sh --claude

mcp-config-cursor:
	@chmod +x scripts/*.sh
	@./scripts/setup-mcp.sh --cursor

mcp-config-vscode:
	@chmod +x scripts/*.sh
	@./scripts/setup-mcp.sh --vscode

# 日志
logs:
	@if [ -f server.log ]; then tail -f server.log; else docker-compose -f deployments/docker/docker-compose.yml logs -f app; fi

# 状态
status:
	@echo "服务状态:"
	@docker-compose -f deployments/docker/docker-compose.yml ps 2>/dev/null || echo "Docker 未运行"
	@echo ""
	@curl -s http://localhost:8080/health && echo " - API Gateway 运行中" || echo "API Gateway 未运行"
	@curl -s http://localhost:8080/mcp -X POST -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","id":1,"method":"ping"}' > /dev/null 2>&1 && echo "MCP Endpoint 运行中" || echo "MCP Endpoint 未运行"
	@echo ""

# 发布构建
release:
	@echo "构建发布版本..."
	@chmod +x scripts/build-release.sh
	@if [ -z "$(VERSION)" ]; then \
		echo "请指定版本: make release VERSION=x.x.x"; \
		exit 1; \
	fi
	@./scripts/build-release.sh $(VERSION)

# 本地多平台构建
release-local:
	@echo "本地多平台构建..."
	@chmod +x scripts/build-release.sh
	@./scripts/build-release.sh dev

# 下载预编译二进制
download:
	@chmod +x scripts/download.sh
	@./scripts/download.sh $(VERSION)

# 安装二进制
install-binary:
	@chmod +x scripts/install-binary.sh
	@./scripts/install-binary.sh

# 卸载
uninstall:
	@chmod +x scripts/uninstall.sh
	@./scripts/uninstall.sh

# 升级
upgrade:
	@chmod +x scripts/upgrade.sh
	@./scripts/upgrade.sh

# 显示版本
version:
	@if [ -f bin/server ]; then \
		./bin/server --version; \
	elif [ -f /usr/local/bin/open-station ]; then \
		/usr/local/bin/open-station --version; \
	else \
		echo "Open Station 未安装"; \
	fi