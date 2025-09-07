# Solana 限价单交易后端 MVP

> 基于 Solana devnet 的中心化托管触发式市价单（Stop-Market）交易系统

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-ready-blue.svg)](https://docker.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> **📋 点击查看 [演示录屏.mp4](./docs/演示录屏.mp4)**

## 🔗 系统设计文档

> **📋 [完整系统设计文档](./docs/系统设计.md)** - 架构图、数据模型、API设计、状态机等详细设计  

## 🚀 快速开始

### 🌐 GitHub Codespaces 一键启动

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new)

**推荐使用 Codespaces 进行开发，零配置即可开始！**

#### 快速启动步骤：
1. 点击上方的 **Open in GitHub Codespaces** 按钮
2. 等待容器构建完成（约 3-5 分钟）

#### 注意：
下载依赖时可能会出现网络错误

### 本地开发环境

#### 前置要求

- **Go 1.23+**
- **Docker & Docker Compose**
- **PostgreSQL 15+** (可选，推荐使用 Docker)
- **Redis 7+** (可选，推荐使用 Docker)

#### 快速启动

```bash
# 1. 克隆项目
git clone https://github.com/huangliang-hl/solana-mvp-exchange.git
cd solana-mvp-exchange

# 2. 初始化环境
make init

# 3. 启动开发环境
make dev

# 4. 运行演示
make demo
```

服务地址：
- **API 服务**: http://localhost:8080
- **价格 WebSocket**: ws://localhost:8081/price
- **测试 API Key**: `test-key-1`

## 📁 项目结构

```
.
├── bin/                        # 编译后的可执行文件
│   ├── api                    # HTTP API 服务
│   ├── price                  # 价格推送服务
│   └── worker                 # 订单执行器
├── docs/                      # 系统设计文档
│   └── 系统设计.md            # 完整系统设计文档
├── cmd/                        # 可执行程序入口
│   ├── api/main.go            # HTTP API 服务入口
│   ├── price/main.go          # 价格推送服务入口
│   └── worker/main.go         # 订单执行器入口
├── internal/                   # 内部模块
│   ├── api/                   # HTTP 路由和处理器
│   │   ├── handlers/          # API 处理器 (用户、钱包、订单)
│   │   ├── middleware/        # 中间件 (鉴权、幂等性)
│   │   └── router.go          # 路由配置
│   ├── config/                # 配置管理
│   ├── executor/              # 订单执行器
│   │   ├── jupiter.go         # Jupiter 集成
│   │   ├── jupiter_factory.go # Jupiter 工厂模式
│   │   ├── jupiter_mock.go    # Jupiter Mock 实现
│   │   ├── service.go         # 执行器服务
│   │   └── solana.go          # Solana RPC 客户端
│   ├── integration/           # 集成测试
│   ├── ledger/                # 记账服务
│   │   ├── reconcile.go       # 对账逻辑
│   │   └── service.go         # 记账服务
│   ├── logger/                # 日志服务
│   ├── order/                 # 订单服务
│   ├── price/                 # 价格服务
│   │   ├── hub.go            # WebSocket 连接中心
│   │   ├── service.go        # 价格服务
│   │   └── simulator.go      # 价格模拟器
│   ├── queue/                 # 消息队列 (Redis)
│   │   ├── redis.go          # Redis 队列实现
│   │   └── topics.go         # 队列主题定义
│   ├── repository/            # 数据访问层
│   │   ├── db.go             # 数据库连接
│   │   ├── ledger_repository.go    # 记账数据访问
│   │   ├── order_repository.go     # 订单数据访问
│   │   ├── user_repository.go      # 用户数据访问
│   │   └── wallet_repository.go    # 钱包数据访问
│   ├── testutil/              # 测试工具
│   ├── trigger/               # 触发引擎
│   │   ├── engine.go         # 触发引擎核心
│   │   └── logic_test.go     # 触发逻辑测试
│   ├── types/                 # 数据模型和类型定义
│   ├── user/                  # 用户服务
│   └── wallet/                # 钱包服务
├── logs/                       # 日志文件目录
├── migrations/                 # 数据库迁移脚本
│   ├── 0001_init.sql          # 初始化数据库结构
├── scripts/                   # 脚本目录
│   ├── cleanup_demo.sh        # 清理演示数据
│   ├── demo.sh                # 完整演示脚本
│   ├── dev-setup.sh           # 开发环境设置
│   ├── price_pusher.py        # 价格推送脚本
│   └── run_integration_tests.sh  # 集成测试运行脚本
├── docker-compose.yml         # 开发环境容器编排
├── docker-compose.prod.yml    # 生产环境容器编排
├── docker-compose.test.yml    # 测试环境容器编排
├── Dockerfile                 # Docker 镜像构建文件
├── go.mod                     # Go 模块依赖
├── go.sum                     # Go 模块校验和
└── Makefile                   # 构建和开发命令
```

## 🛠️ 开发指南

### 环境配置

```bash
# 复制并编辑环境配置
cp .env.example .env

# 关键配置项
RPC_URL=https://api.devnet.solana.com
JUPITER_ENDPOINT=https://quote-api.jup.ag/v6
PRIVATE_KEY=your_solana_private_key
DB_URL=postgres://postgres:password@localhost:5432/solana_limit_order
REDIS_URL=redis://localhost:6379
API_KEYS=test-key-1,test-key-2
```

### 开发命令

```bash
# 构建和运行
make build              # 构建所有服务
make run-api            # 单独运行 API 服务
make run-price          # 单独运行价格服务
make run-worker         # 单独运行执行器

# 测试
make test               # 单元测试
make test-integration   # 集成测试
make test-integration-docker # 使用 Docker 完整测试环境

# 代码质量
make fmt               # 格式化代码
make lint              # 代码检查

# Docker
make docker-build      # 构建镜像
make docker-up         # 启动所有服务
make docker-down       # 停止服务
make logs              # 查看日志
```

### API 使用示例

```bash
# 创建用户
curl -X POST http://localhost:8080/api/v1/users \
  -H "X-Api-Key: test-key-1" \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","idempotency_key":"user-001"}' | jq .

# 创建 SOL 钱包
curl -X POST http://localhost:8080/api/v1/wallets \
  -H "X-Api-Key: test-key-1" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER_ID\",\"asset\":\"SOL\",\"idempotency_key\":\"wallet-sol-001\"}" | jq .

# 充值 SOL
curl -X POST http://localhost:8080/api/v1/deposits \
  -H "X-Api-Key: test-key-1" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER_ID\",\"asset\":\"SOL\",\"amount\":\"10.0\",\"idempotency_key\":\"deposit-001\"}"  | jq .

# 创建限价单
curl -X POST http://localhost:8080/api/v1/orders \
  -H "X-Api-Key: test-key-1" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\":\"$USER_ID\",
    \"base\":\"SOL\",\"quote\":\"USDC\",
    \"size\":\"1.0\",\"side\":\"sell\",
    \"trigger_price\":\"30.0\",\"trigger_op\":\"lte\",
    \"max_slippage_pct\":\"0.5\",
    \"idempotency_key\":\"order-001\"
  }" | jq .

# WebSocket 价格订阅
wscat -c ws://localhost:8081/price?api_key=test-key-1
```

### 演示脚本

```bash
# 完整交易流程演示
make demo

# 手动执行步骤
./scripts/demo.sh
```

演示包含：
1. 创建用户和钱包
2. 充值
3. 创建 Stop-Market 订单
4. 价格推送触发
5. 自动执行和成交确认

## 🚀 部署

### Docker Compose (推荐)

```bash
# 生产环境部署
# 首先复制并配置生产环境变量
cp .env.prod.example .env.prod
# 编辑 .env.prod 填写实际的生产环境配置
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

# 开发环境
make dev
```

### 手动部署

```bash
# 构建
make build

# 启动数据库服务
docker run -d --name postgres -e POSTGRES_DB=solana_limit_order postgres:15
docker run -d --name redis redis:7-alpine

# 运行迁移
psql -f migrations/0001_init.sql

# 启动服务
./bin/api &
./bin/price &
./bin/worker &
```

## 📊 监控与日志

### 服务日志

```bash
# 所有服务日志
make logs

# 特定服务
make logs-api
make logs-worker
make logs-price
```

### 关键指标

- 订单触发率和执行成功率
- API 响应时间 (P50/P95)
- 队列积压长度
- 数据库连接池状态

## 🔒 安全注意事项

- **私钥管理**: 生产环境使用 KMS 或 Vault
- **API 鉴权**: 使用强随机 API Keys
- **网络安全**: 所有外部通信启用 TLS
- **资金安全**: 定期对账和监控异常

## 🆘 故障排除

### 常见问题

**Q: 服务启动失败？**
```bash
# 检查端口占用
lsof -i :8080
lsof -i :8081

# 查看详细日志
make logs
```

**Q: 数据库连接失败？**
```bash
# 重置数据库
make reset-db

# 检查连接
docker compose exec postgres psql -U postgres -d solana_limit_order -c "SELECT 1;"
```

**Q: Jupiter API 调用失败？**
- 确认网络连接
- 检查 devnet 环境配置
- 查看执行器日志

### 获取帮助

- 📖 [系统设计文档](./docs/系统设计.md)
- 🐛 [Issues](https://github.com/huangliang-hl/solana-mvp-exchange/issues)

---
