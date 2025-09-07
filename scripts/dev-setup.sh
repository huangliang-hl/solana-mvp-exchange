#!/bin/bash

# 开发环境设置脚本
# 在 devcontainer 创建后自动运行

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 重试函数
retry_command() {
    local cmd="$1"
    local max_attempts="$2"
    local delay="$3"
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        log_info "尝试执行: $cmd (第 $attempt/$max_attempts 次)"
        if eval "$cmd"; then
            log_success "命令执行成功"
            return 0
        else
            if [ $attempt -eq $max_attempts ]; then
                log_error "命令执行失败，已达到最大重试次数"
                return 1
            fi
            log_warning "命令执行失败，等待 ${delay}s 后重试..."
            sleep $delay
            ((attempt++))
        fi
    done
}

log_info "🚀 开始设置 Solana 限价订单后端开发环境..."

# 检查是否在 Codespaces 环境中
if [ -n "$CODESPACES" ]; then
    log_info "🌐 检测到 GitHub Codespaces 环境"
    export CODESPACES_ENV=true
else
    log_info "🖥️  本地开发环境"
    export CODESPACES_ENV=false
fi

# 等待服务启动（减少初始等待时间）
log_info "⏳ 等待服务启动..."
sleep 5

# 检查数据库连接（带重试）
log_info "🔍 检查数据库连接..."
if ! retry_command "pg_isready -h postgres -p 5432 -U postgres -q" 30 2; then
    log_error "无法连接到 PostgreSQL 数据库"
    exit 1
fi
log_success "PostgreSQL 数据库连接正常"

# 检查 Redis 连接（带重试）
log_info "🔍 检查 Redis 连接..."
if ! retry_command "redis-cli -h redis ping > /dev/null 2>&1" 15 2; then
    log_error "无法连接到 Redis"
    exit 1
fi
log_success "Redis 连接正常"

# 设置 Go 环境
log_info "🛠️  设置 Go 开发环境..."
go version

# 下载项目依赖（带重试）
log_info "📦 下载 Go 模块依赖..."
if ! retry_command "go mod download" 3 5; then
    log_warning "Go 模块下载失败，尝试清理缓存后重试..."
    go clean -modcache
    if ! retry_command "go mod download" 2 10; then
        log_error "Go 模块下载失败"
        exit 1
    fi
fi

if ! retry_command "go mod tidy" 3 2; then
    log_warning "go mod tidy 失败，但可以继续"
fi

log_success "Go 模块依赖下载完成"

# 检查是否需要安装额外工具（避免重复安装）
log_info "🔧 检查开发工具..."
if [ "$CODESPACES_ENV" = "true" ]; then
    log_info "Codespaces 环境，跳过额外工具安装（已在 Dockerfile 中安装）"
else
    log_info "安装额外的开发工具..."
    if ! command -v air &> /dev/null; then
        retry_command "go install github.com/cosmtrek/air@latest" 3 5
    fi
    if ! command -v migrate &> /dev/null; then
        retry_command "go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest" 3 5
    fi
fi

# 创建环境配置文件（如果不存在）
if [ ! -f ".env" ]; then
    log_info "📝 创建开发环境配置文件..."
    if [ -f ".env.example" ]; then
        cp .env.example .env
        # 更新配置以使用容器内的服务
        sed -i 's/localhost:5432/postgres:5432/g' .env 2>/dev/null || true
        sed -i 's/localhost:6379/redis:6379/g' .env 2>/dev/null || true
        sed -i 's/MODE=dev/MODE=dev/g' .env 2>/dev/null || true
        sed -i 's/LOG_LEVEL=info/LOG_LEVEL=debug/g' .env 2>/dev/null || true
        log_success "环境配置文件创建完成"
    else
        log_warning ".env.example 文件不存在，跳过环境配置文件创建"
    fi
else
    log_success "环境配置文件已存在"
fi

# 运行数据库迁移
log_info "🗄️  检查数据库迁移..."
if retry_command "psql -h postgres -U postgres -d solana_limit_order -c \"SELECT 1 FROM information_schema.tables WHERE table_name = 'users';\" | grep -q 1" 3 2; then
    log_success "数据库表已存在，跳过迁移"
else
    log_info "📊 数据库表不存在，但迁移文件已通过 docker-entrypoint-initdb.d 自动执行"
    # 等待数据库初始化完成
    sleep 10
    if retry_command "psql -h postgres -U postgres -d solana_limit_order -c \"SELECT 1 FROM information_schema.tables WHERE table_name = 'users';\" | grep -q 1" 5 3; then
        log_success "数据库迁移完成"
    else
        log_warning "数据库迁移可能未完成，请手动检查"
    fi
fi

# 运行基础测试（非关键，失败不退出）
log_info "🧪 运行基础测试..."
if [ -d "./internal/config" ]; then
    if go test -v ./internal/config -timeout=30s; then
        log_success "配置测试通过"
    else
        log_warning "配置测试失败，但可以继续"
    fi
else
    log_info "配置测试目录不存在，跳过测试"
fi

# 构建项目（非关键，失败不退出）
log_info "🔨 尝试构建项目..."
if [ -f "Makefile" ]; then
    if make build 2>/dev/null; then
        log_success "项目构建成功"
    else
        log_warning "项目构建失败，请检查 Makefile 和依赖"
    fi
else
    log_info "Makefile 不存在，跳过构建"
fi

# 创建开发用的日志目录
log_info "📁 创建项目目录..."
mkdir -p logs tmp

# 设置 Git hooks（如果存在且不在 Codespaces 中）
if [ -d ".git" ] && [ "$CODESPACES_ENV" != "true" ]; then
    log_info "🔗 设置 Git hooks..."
    # 设置 pre-commit hook
    cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# 运行 go fmt
go fmt ./...

# 运行 go vet
go vet ./...

# 运行简单测试（如果存在）
if [ -d "./internal/config" ]; then
    go test -timeout=10s ./internal/config
fi
EOF
    chmod +x .git/hooks/pre-commit
    log_success "Git hooks 设置完成"
elif [ "$CODESPACES_ENV" = "true" ]; then
    log_info "Codespaces 环境，跳过 Git hooks 设置"
else
    log_info "Git 仓库不存在，跳过 Git hooks 设置"
fi

# 创建有用的开发别名（仅在 Codespaces 环境中或别名不存在时）
if [ "$CODESPACES_ENV" = "true" ] || ! grep -q "# Solana 项目开发别名" ~/.bashrc; then
    log_info "🔧 设置开发别名..."
    cat >> ~/.bashrc << 'EOF'

# Solana 项目开发别名
alias api='go run cmd/api/main.go'
alias worker='go run cmd/worker/main.go'
alias price='go run cmd/price/main.go'
alias test-all='go test ./...'
alias test-cover='go test -cover ./...'
alias build-all='make build'
alias logs='tail -f logs/*.log'
alias db='psql -h postgres -U postgres -d solana_limit_order'
alias redis-cli='redis-cli -h redis'

# Go 开发工具别名
alias gofmt='go fmt ./...'
alias govet='go vet ./...'
alias golint='golangci-lint run'
alias air='air -c .air.toml'
EOF
    log_success "开发别名设置完成"
else
    log_info "开发别名已存在，跳过设置"
fi

# 创建 Air 配置文件（热重载）
if [ ! -f ".air.toml" ]; then
    log_info "🔥 创建 Air 热重载配置..."
    cat > .air.toml << 'EOF'
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/api"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "migrations", "logs", ".devcontainer"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
EOF
    log_success "Air 配置文件创建完成"
else
    log_info "Air 配置文件已存在，跳过创建"
fi

# 显示有用的开发信息
echo ""
log_success "🎉 开发环境设置完成！"
echo ""
log_info "📋 可用服务："
echo "  🌐 API 服务: http://localhost:8080"
echo "  📡 WebSocket 价格服务: http://localhost:8081"
echo "  🗄️  PostgreSQL: localhost:5432 (用户: postgres, 密码: password)"
echo "  🔗 Redis: localhost:6379"
echo "  🛠️  Redis Commander: http://localhost:8082"
echo "  📊 Adminer (数据库管理): http://localhost:8083"
echo ""
log_info "🚀 常用命令："
echo "  启动 API: make run-api 或 api"
echo "  启动 Worker: make run-worker 或 worker"
echo "  启动价格服务: make run-price 或 price"
echo "  运行测试: make test 或 test-all"
echo "  热重载开发: air"
echo "  查看日志: logs"
echo "  连接数据库: db"
echo "  连接 Redis: redis-cli"
echo ""
if [ "$CODESPACES_ENV" = "true" ]; then
    log_info "🌐 Codespaces 特定信息："
    echo "  - 端口会自动转发到公共 URL"
    echo "  - 使用 'gh codespace ports' 查看端口状态"
    echo "  - 环境变量已针对 Codespaces 优化"
    echo ""
fi
log_info "📚 更多信息请查看 README.md 和 docs/ 目录"
echo ""

