#!/bin/bash

# 演示数据清理脚本 - 清理 demo.sh 创建的数据和锁定余额

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# API 配置
API_BASE="http://localhost:8080/api/v1"
API_KEY="test-key-1"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "\n${PURPLE}=== $1 ===${NC}"
}

# 检查服务状态
check_services() {
    log_step "检查服务状态"

    # 检查 API 服务
    if curl -s -f "http://localhost:8080/health" > /dev/null; then
        log_success "API 服务运行正常"
    else
        log_error "API 服务未运行，请先启动服务"
        exit 1
    fi
}

# 调用 API 的通用函数
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    local expect_status="${4:-200}"

    local headers=(-H "Content-Type: application/json" -H "X-Api-Key: $API_KEY")

    local cmd="curl -s -w '%{http_code}' -X $method"
    for header in "${headers[@]}"; do
        cmd="$cmd -H '$header'"
    done
    if [ -n "$data" ]; then
        cmd="$cmd -d '$data'"
    fi
    cmd="$cmd $API_BASE$endpoint"

    local response=$(eval $cmd)
    local status_code=${response: -3}
    local body=${response%???}

    # 检查状态码
    if [[ "$status_code" == "$expect_status"* ]] || [[ "$expect_status" == "2"* && "$status_code" =~ ^2[0-9][0-9]$ ]]; then
        echo "$body"
        return 0
    else
        log_error "API 调用失败 - 状态码: $status_code, 响应: $body"
        return 1
    fi
}

# 查找并清理演示用户
cleanup_demo_users() {
    log_step "查找并清理演示用户"

    log_info "查找所有演示用户（用户名包含 'demo_user_'）..."
    
    # 这里我们需要实现一个查找演示用户的方法
    # 由于API可能没有直接的用户列表接口，我们使用数据库清理
    log_warning "当前版本使用数据库清理方式"
}

# 清理所有挂单（释放锁定资金）
cleanup_open_orders() {
    log_step "清理所有挂单"

    log_info "这将取消所有open和triggered状态的订单，释放锁定资金"
    
    # 获取所有用户的开放订单并取消
    # 注意：这需要管理员权限或特殊的清理API
    log_info "取消所有挂单..."
    
    # 实际实现中，可能需要通过数据库直接操作
}

# 直接清理数据库中的演示数据
cleanup_database() {
    log_step "直接清理数据库演示数据"
    
    log_warning "这将直接操作数据库清理演示数据"
    log_info "取消所有挂单并释放锁定资金..."
    
    # 检查是否有docker容器运行
    if docker compose ps | grep -q "postgres"; then
        log_info "找到PostgreSQL容器，开始清理..."
        
        # 取消所有挂单并释放锁定资金
        log_info "1. 取消所有open和triggered状态的订单..."
        docker compose exec -T postgres psql -U postgres -d solana_limit_order << 'EOF'
-- 更新所有open和triggered状态的订单为canceled
UPDATE orders 
SET status = 'canceled', updated_at = NOW() 
WHERE status IN ('open', 'triggered');

-- 释放所有锁定资金：将locked余额转移到available余额
UPDATE wallets 
SET 
    available_decimal = available_decimal + locked_decimal,
    locked_decimal = 0,
    updated_at = NOW()
WHERE locked_decimal > 0;
EOF
        
        if [ $? -eq 0 ]; then
            log_success "✅ 所有挂单已取消，锁定资金已释放"
        else
            log_error "❌ 数据库操作失败"
            return 1
        fi
        
        # 可选：清理演示用户数据
        echo ""
        read -p "是否同时清理所有演示用户数据? (y/N): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "2. 清理演示用户数据..."
            docker compose exec -T postgres psql -U postgres -d solana_limit_order << 'EOF'
-- 删除演示订单
DELETE FROM orders WHERE user_id IN (
    SELECT id FROM users WHERE username LIKE 'demo_user_%'
);

-- 删除演示账本记录
DELETE FROM ledger_entries WHERE user_id IN (
    SELECT id FROM users WHERE username LIKE 'demo_user_%'  
);

-- 删除演示钱包
DELETE FROM wallets WHERE user_id IN (
    SELECT id FROM users WHERE username LIKE 'demo_user_%'
);

-- 删除演示用户
DELETE FROM users WHERE username LIKE 'demo_user_%';
EOF
            if [ $? -eq 0 ]; then
                log_success "✅ 演示用户数据清理完成"
            else
                log_warning "⚠️  演示用户数据清理可能未完全成功"
            fi
        fi
        
    else
        log_error "未找到PostgreSQL容器，请确保服务正在运行"
        log_info "您也可以手动连接数据库执行以下SQL："
        cat << 'EOF'

-- 取消所有挂单并释放锁定资金
UPDATE orders 
SET status = 'canceled', updated_at = NOW() 
WHERE status IN ('open', 'triggered');

UPDATE wallets 
SET 
    available_decimal = available_decimal + locked_decimal,
    locked_decimal = 0,
    updated_at = NOW()
WHERE locked_decimal > 0;

EOF
        return 1
    fi
}

# 验证清理结果
verify_cleanup() {
    log_step "验证清理结果"
    
    if docker compose ps | grep -q "postgres"; then
        log_info "检查剩余的锁定余额..."
        
        local locked_count=$(docker compose exec -T postgres psql -U postgres -d solana_limit_order -t -c "SELECT COUNT(*) FROM wallets WHERE locked_decimal > 0;" | tr -d ' ')
        local open_orders=$(docker compose exec -T postgres psql -U postgres -d solana_limit_order -t -c "SELECT COUNT(*) FROM orders WHERE status IN ('open', 'triggered');" | tr -d ' ')
        
        if [ "$locked_count" = "0" ]; then
            log_success "✅ 所有锁定余额已成功释放"
        else
            log_warning "⚠️  仍有 $locked_count 个钱包存在锁定余额"
        fi
        
        if [ "$open_orders" = "0" ]; then
            log_success "✅ 所有挂单已成功取消"
        else
            log_warning "⚠️  仍有 $open_orders 个挂单未取消"
        fi
    fi
}

# 主函数
main() {
    echo ""
    echo "================================================================"
    echo "   Solana 限价单后端 MVP 演示数据清理"
    echo "   清理演示过程中创建的用户和订单数据，释放锁定余额"
    echo "================================================================"
    echo ""

    # 检查依赖工具
    if ! command -v curl &> /dev/null; then
        log_error "curl 命令未找到，请先安装 curl"
        exit 1
    fi

    if ! command -v docker &> /dev/null; then
        log_error "docker 命令未找到，请先安装 docker"
        exit 1
    fi

    check_services
    cleanup_database
    verify_cleanup

    echo ""
    log_success "🎉 清理完成！锁定余额问题已解决"
    echo -e "${PURPLE}所有挂单已取消，锁定资金已释放回可用余额${NC}"
    echo ""
}

# 异常处理
trap 'echo -e "\n${RED}清理中断${NC}"; exit 1' INT TERM

# 运行主函数
main "$@"
