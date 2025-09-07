# 基于 Go 1.23 官方镜像构建开发环境
FROM golang:1.23-bullseye

# 设置环境变量
ENV DEBIAN_FRONTEND=noninteractive
ENV GOPATH=/go
ENV GOROOT=/usr/local/go
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

# 设置 Go 环境优化
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org
ENV GONOPROXY=""
ENV GONOSUMDB=""
ENV GOPRIVATE=""

# 更新系统包并安装必要的工具（减少网络请求）
RUN apt-get update && apt-get install -y --no-install-recommends \
	# 基础工具
	curl \
	wget \
	git \
	unzip \
	ca-certificates \
	# 网络工具
	iputils-ping \
	netcat-openbsd \
	telnet \
	# 数据库客户端
	postgresql-client \
	redis-tools \
	# 编辑器和开发工具
	vim \
	nano \
	jq \
	# 进程管理
	htop \
	procps \
	# 构建工具
	build-essential \
	# GitHub CLI 依赖
	software-properties-common \
	# 清理缓存
	&& apt-get clean \
	&& rm -rf /var/lib/apt/lists/*

# 安装 GitHub CLI (内置，避免网络下载问题)
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
	&& chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
	&& echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
	&& apt-get update \
	&& apt-get install -y gh \
	&& apt-get clean \
	&& rm -rf /var/lib/apt/lists/*

# 批量安装 Go 开发工具（减少网络请求次数）
RUN go install golang.org/x/tools/gopls@latest \
	golang.org/x/tools/cmd/goimports@latest \
	github.com/go-delve/delve/cmd/dlv@latest \
	honnef.co/go/tools/cmd/staticcheck@latest \
	github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
	github.com/swaggo/swag/cmd/swag@latest \
	github.com/air-verse/air@latest \
	github.com/cosmtrek/air@latest \
	github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 创建工作目录
WORKDIR /workspaces/solana-mvp-exchange

# 设置 Git 安全目录（避免权限问题）
RUN git config --global --add safe.directory /workspaces/solana-mvp-exchange \
	&& git config --global init.defaultBranch main \
	&& git config --global pull.rebase false

# 创建 Go 项目的常用目录结构
RUN mkdir -p /go/pkg/mod \
	/go/bin \
	/root/.cache/go-build \
	/workspaces/solana-mvp-exchange/logs \
	/workspaces/solana-mvp-exchange/tmp

# 创建一些有用的别名和环境设置
RUN echo 'alias ll="ls -la"' >> ~/.bashrc \
	&& echo 'alias la="ls -A"' >> ~/.bashrc \
	&& echo 'alias l="ls -CF"' >> ~/.bashrc \
	&& echo 'alias grep="grep --color=auto"' >> ~/.bashrc \
	&& echo 'alias fgrep="fgrep --color=auto"' >> ~/.bashrc \
	&& echo 'alias egrep="egrep --color=auto"' >> ~/.bashrc \
	&& echo 'export PS1="\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "' >> ~/.bashrc \
	&& echo '# Solana 项目开发别名' >> ~/.bashrc \
	&& echo 'alias api="go run cmd/api/main.go"' >> ~/.bashrc \
	&& echo 'alias worker="go run cmd/worker/main.go"' >> ~/.bashrc \
	&& echo 'alias price="go run cmd/price/main.go"' >> ~/.bashrc \
	&& echo 'alias test-all="go test ./..."' >> ~/.bashrc \
	&& echo 'alias db="psql -h postgres -U postgres -d solana_limit_order"' >> ~/.bashrc \
	&& echo 'alias redis-cli="redis-cli -h redis"' >> ~/.bashrc

# 暴露常用端口
EXPOSE 8080 8081 40000

# 设置默认命令
CMD ["bash"]

