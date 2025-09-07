.PHONY: help build run test clean dev docker-build docker-up docker-down logs

# 默认目标
help:
	@echo "Solana 限价单后端 MVP - 可用命令:"
	@echo ""
	@echo "构建和运行:"
	@echo "  build            - 构建所有二进制文件"
	@echo "  run-api          - 运行 API 服务"
	@echo "  run-price        - 运行价格服务"
	@echo "  run-worker       - 运行执行器服务"
	@echo ""
	@echo "测试:"
	@echo "  test                     - 运行单元测试"
	@echo "  test-integration         - 运行集成测试（智能检测环境）"
	@echo "  test-integration-docker  - 使用 Docker 运行完整集成测试"
	@echo "  test-all                - 运行所有测试"
	@echo "  install-test-deps       - 显示测试依赖安装指南"
	@echo ""
	@echo "开发环境:"
	@echo "  dev          - 启动开发环境（docker compose）"
	@echo "  dev-fresh    - 启动开发环境（强制重建镜像，确保使用最新代码）"
	@echo "  dev-hot      - 启动热重载开发环境（推荐）"
	@echo "  dev-down     - 停止开发环境"
	@echo "  dev-restart  - 重启开发环境服务"
	@echo "  logs-dev     - 查看开发环境日志"
	@echo "  clean        - 清理构建文件"
	@echo "  fmt          - 格式化代码"
	@echo "  lint         - 运行代码检查"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build       - 构建所有 Docker 镜像"
	@echo "  docker-build-fresh - 强制重新构建所有 Docker 镜像（无缓存）"
	@echo "  docker-up    - 启动所有服务"
	@echo "  docker-down  - 停止所有服务"
	@echo "  logs         - 查看服务日志"
	@echo ""
	@echo "生产环境:"
	@echo "  deploy-prod  - 部署生产环境"
	@echo "  stop-prod    - 停止生产环境"
	@echo "  logs-prod    - 查看生产环境日志"
	@echo "  health-prod  - 检查生产环境健康状态"
	@echo ""
	@echo "其他:"
	@echo "  demo         - 运行演示脚本"
	@echo "  cleanup-demo - 清理演示数据和锁定余额"
	@echo "  backup       - 备份数据库"
	@echo "  init         - 初始化开发环境"

# 构建所有二进制文件
build:
	@echo "构建所有服务..."
	@mkdir -p bin
	@go build -o bin/api ./cmd/api
	@go build -o bin/price ./cmd/price
	@go build -o bin/worker ./cmd/worker
	@echo "构建完成"

# 运行API服务
run-api:
	@echo "启动 API 服务..."
	@go run ./cmd/api/main.go

# 运行价格服务
run-price:
	@echo "启动价格服务..."
	@go run ./cmd/price/main.go

# 运行执行器服务
run-worker:
	@echo "启动执行器服务..."
	@go run ./cmd/worker/main.go

# 运行测试
test:
	@echo "运行单元测试..."
	@go test -v ./... -short

# 运行集成测试（智能检测环境）
test-integration:
	@echo "运行集成测试..."
	@./scripts/run_integration_tests.sh

# 使用 Docker 运行完整集成测试
test-integration-docker:
	@echo "使用 Docker 运行完整集成测试..."
	@docker compose -f docker-compose.test.yml up --build --abort-on-container-exit
	@docker compose -f docker-compose.test.yml down

# 运行所有测试
test-all: test test-integration

# 安装测试依赖
install-test-deps:
	@echo "安装测试依赖..."
	@echo "请根据您的系统运行以下命令之一："
	@echo ""
	@echo "Ubuntu/Debian:"
	@echo "  sudo apt-get update"
	@echo "  sudo apt-get install postgresql-client redis-tools"
	@echo ""
	@echo "macOS:"
	@echo "  brew install postgresql redis"
	@echo ""
	@echo "Arch Linux:"
	@echo "  sudo pacman -S postgresql-libs redis"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -rf bin/
	@docker system prune -f
	@echo "清理完成"

# 启动开发环境
dev: docker-build docker-up
	@echo "开发环境已启动"
	@echo "API 服务: http://localhost:8080"
	@echo "价格 WebSocket: ws://localhost:8081/price"
	@echo ""
	@echo "测试API密钥: test-key-1"
	@echo ""
	@echo "使用 'make logs' 查看日志"
	@echo "使用 'make docker-down' 停止服务"

# 构建Docker镜像
docker-build:
	@echo "构建 Docker 镜像..."
	@docker build --target api -t solana-limit-order/api:latest .
	@docker build --target price -t solana-limit-order/price:latest .
	@docker build --target worker -t solana-limit-order/worker:latest .
	@echo "Docker 镜像构建完成"

# 强制重新构建Docker镜像（无缓存）
docker-build-fresh:
	@echo "强制重新构建 Docker 镜像（无缓存）..."
	@docker build --no-cache --target api -t solana-limit-order/api:latest .
	@docker build --no-cache --target price -t solana-limit-order/price:latest .
	@docker build --no-cache --target worker -t solana-limit-order/worker:latest .
	@echo "Docker 镜像重新构建完成"

# 开发环境（强制重建）
dev-fresh: docker-build-fresh docker-up
	@echo "开发环境已启动（使用最新代码）"
	@echo "API 服务: http://localhost:8080"
	@echo "价格 WebSocket: ws://localhost:8081/price"
	@echo ""
	@echo "测试API密钥: test-key-1"
	@echo ""
	@echo "使用 'make logs' 查看日志"
	@echo "使用 'make docker-down' 停止服务"

# 真正的开发模式（热重载）
dev-hot: docker-down
	@echo "启动开发环境（热重载模式）..."
	@docker compose -f docker-compose.dev.yml up --build -d
	@echo "等待服务启动..."
	@sleep 10
	@echo "开发环境已启动（热重载模式）"
	@echo "API 服务: http://localhost:8080"
	@echo "价格 WebSocket: ws://localhost:8081/price"
	@echo ""
	@echo "测试API密钥: test-key-1"
	@echo ""
	@echo "✅ 代码更新将立即生效，无需重启容器！"
	@echo ""
	@echo "使用 'make logs-dev' 查看开发环境日志"
	@echo "使用 'make dev-down' 停止开发环境"

# 停止开发环境
dev-down:
	@echo "停止开发环境..."
	@docker compose -f docker-compose.dev.yml down

# 查看开发环境日志
logs-dev:
	@docker compose -f docker-compose.dev.yml logs -f

# 重启开发环境服务
dev-restart:
	@echo "重启开发环境服务..."
	@docker compose -f docker-compose.dev.yml restart api price worker
	@echo "开发环境服务已重启"

# 启动所有服务
docker-up: 
	@echo "启动所有服务..."
	@docker compose up -d
	@echo "等待服务启动..."
	@sleep 10
	@echo "检查服务状态..."
	@docker compose ps

# 停止所有服务
docker-down:
	@echo "停止所有服务..."
	@docker compose down

# 查看服务日志
logs:
	@docker compose logs -f

# 查看特定服务日志
logs-api:
	@docker compose logs -f api

logs-price:
	@docker compose logs -f price

logs-worker:
	@docker compose logs -f worker

logs-postgres:
	@docker compose logs -f postgres

logs-redis:
	@docker compose logs -f redis

# 格式化代码
fmt:
	@echo "格式化代码..."
	@go fmt ./...

# 代码检查
lint:
	@echo "运行代码检查..."
	@go vet ./...

# 下载依赖
deps:
	@echo "下载依赖..."
	@go mod download
	@go mod tidy

# 初始化开发环境
init:
	@echo "初始化开发环境..."
	@cp .env.example .env
	@echo "已创建 .env 文件，请根据需要修改配置"

# 重置数据库
reset-db:
	@echo "重置数据库..."
	@docker compose exec postgres psql -U postgres -d solana_limit_order -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@docker compose exec postgres psql -U postgres -d solana_limit_order -f /docker-entrypoint-initdb.d/0001_init.sql

# 检查服务健康状态
health:
	@echo "检查服务健康状态..."
	@curl -s http://localhost:8080/health | jq .
	@curl -s http://localhost:8081/health | jq .

# 演示脚本
demo:
	@echo "运行演示脚本..."
	@./scripts/demo.sh

# 清理演示数据
cleanup-demo:
	@echo "清理演示数据和锁定余额..."
	@./scripts/cleanup_demo.sh

# 备份数据库
backup:
	@echo "备份数据库..."
	@mkdir -p backups
	@docker compose exec postgres pg_dump -U postgres solana_limit_order > backups/backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "备份完成"

# 生产环境部署
deploy-prod:
	@echo "部署生产环境..."
	@if [ ! -f ".env.prod" ]; then \
		echo "❌ 未找到 .env.prod 文件"; \
		echo "请复制 env.prod.example 为 .env.prod 并填写配置"; \
		exit 1; \
	fi
	@docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
	@echo "✅ 生产环境部署完成"

# 停止生产环境
stop-prod:
	@echo "停止生产环境..."
	@docker compose -f docker-compose.prod.yml down
	@echo "✅ 生产环境已停止"

# 查看生产环境日志
logs-prod:
	@docker compose -f docker-compose.prod.yml logs -f

# 生产环境健康检查
health-prod:
	@echo "检查生产环境服务健康状态..."
	@docker compose -f docker-compose.prod.yml ps