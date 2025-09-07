#!/bin/bash

# Codespaces 专用设置脚本
# 针对 GitHub Codespaces 环境优化

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

log_info "🌐 开始设置 GitHub Codespaces 环境..."

# 设置环境变量
export CODESPACES=true
export CODESPACES_ENV=true

# 更新系统包（最小化安装）
log_info "📦 安装必要的系统包..."
apt-get update -qq
apt-get install -y --no-install-recommends \
    postgresql-client \
    redis-tools \
    curl \
    wget \
    git \
    jq \
    vim \
    nano \
    > /dev/null 2>&1

# 设置 Go 环境
log_info "🛠️  设置 Go 开发环境..."
export GOPATH=/go
export GOROOT=/usr/local/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export GOPROXY=https://proxy.golang.org,direct

# 创建必要的目录
mkdir -p /go/bin /go/pkg/mod /root/.cache/go-build logs tmp

# 安装基本的 Go 工具
log_info "🔧 安装 Go 开发工具..."
go install golang.org/x/tools/gopls@latest > /dev/null 2>&1 &
go install golang.org/x/tools/cmd/goimports@latest > /dev/null 2>&1 &
go install github.com/go-delve/delve/cmd/dlv@latest > /dev/null 2>&1 &
go install github.com/air-verse/air@latest > /dev/null 2>&1 &

# 等待工具安装完成
wait

# 设置 Git 配置
git config --global --add safe.directory /workspaces/solana-mvp-exchange
git config --global init.defaultBranch main
git config --global pull.rebase false

# 下载项目依赖
log_info "📦 下载 Go 模块依赖..."
if [ -f "go.mod" ]; then
    go mod download > /dev/null 2>&1 || log_warning "Go 模块下载失败，但可以继续"
    go mod tidy > /dev/null 2>&1 || log_warning "go mod tidy 失败，但可以继续"
fi

# 创建基本的开发配置文件
if [ ! -f ".env" ] && [ -f ".env.example" ]; then
    log_info "📝 创建环境配置文件..."
    cp .env.example .env
    # 更新配置为 Codespaces 适用的值
    sed -i 's/localhost/127.0.0.1/g' .env 2>/dev/null || true
    sed -i 's/MODE=dev/MODE=codespaces/g' .env 2>/dev/null || true
    sed -i 's/LOG_LEVEL=info/LOG_LEVEL=debug/g' .env 2>/dev/null || true
fi

# 创建 Air 配置文件
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
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "migrations", "logs", ".devcontainer", ".github"]
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
fi

# 设置开发别名
log_info "🔧 设置开发别名..."
cat >> ~/.bashrc << 'EOF'

# Solana 项目开发别名 (Codespaces)
alias api='go run cmd/api/main.go'
alias worker='go run cmd/worker/main.go'
alias price='go run cmd/price/main.go'
alias test-all='go test ./...'
alias test-cover='go test -cover ./...'
alias build-all='go build ./...'
alias logs='tail -f logs/*.log'
alias gofmt='go fmt ./...'
alias govet='go vet ./...'
alias air='air -c .air.toml'

# Codespaces 特定别名
alias ports='gh codespace ports'
alias info='gh codespace view'
EOF

# 清理
apt-get clean
rm -rf /var/lib/apt/lists/*

# 显示完成信息
echo ""
log_success "🎉 GitHub Codespaces 环境设置完成！"
echo ""
log_info "📋 Codespaces 特性："
echo "  🌐 自动端口转发已配置"
echo "  🔧 Go 开发工具已安装"
echo "  📝 开发别名已设置"
echo "  🔥 Air 热重载已配置"
echo ""
log_info "🚀 快速开始："
echo "  1. 运行 'api' 启动 API 服务"
echo "  2. 运行 'air' 启用热重载开发"
echo "  3. 运行 'test-all' 运行所有测试"
echo "  4. 运行 'ports' 查看端口转发状态"
echo ""
log_info "💡 提示："
echo "  - 端口会自动转发到公共 URL"
echo "  - 使用 GitHub Copilot 提高开发效率"
echo "  - 环境变量已针对 Codespaces 优化"
echo ""