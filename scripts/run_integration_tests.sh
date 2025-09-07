#!/bin/bash

# 集成测试运行脚本
set -e

echo "运行集成测试..."

# 设置测试环境
export TEST_MODE=true
export LOG_LEVEL=error

# 检查是否有 PostgreSQL 和 Redis 客户端工具
HAS_PSQL=$(command -v psql >/dev/null 2>&1 && echo "yes" || echo "no")
HAS_REDIS=$(command -v redis-cli >/dev/null 2>&1 && echo "yes" || echo "no")

if [ "$HAS_PSQL" = "yes" ] && [ "$HAS_REDIS" = "yes" ]; then
    echo "检测到完整的数据库环境，运行完整集成测试..."
    
    # 确保测试数据库存在
    echo "设置测试数据库..."
    createdb solana_limit_order_test || echo "测试数据库已存在"
    
    # 运行数据库迁移
    echo "运行数据库迁移..."
    psql -d solana_limit_order_test -f migrations/0001_init.sql || echo "迁移已应用"
    
    # 检查 Redis 是否运行
    echo "检查 Redis 是否运行..."
    redis-cli ping || (echo "启动 Redis..." && redis-server --daemonize yes)
    
    # 运行完整集成测试
    echo "运行完整集成测试..."
    go test -v -tags=integration ./internal/integration/...
else
    echo "未检测到完整数据库环境，运行基础集成测试..."
    echo "要运行完整测试，请安装 PostgreSQL 和 Redis 客户端工具："
    echo "  sudo apt-get install postgresql-client redis-tools"
    echo ""
    
    # 运行不需要数据库的基础测试
    echo "运行基础集成测试（无外部依赖）..."
    go test -v ./internal/integration/ -run "TestMock|TestData|TestConfig"
fi

echo "集成测试完成！"
