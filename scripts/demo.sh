#!/bin/bash

# Solana 限价单后端 MVP 演示脚本
# 演示完整的交易流程：入金 → 下单 → 触发 → 成交 → 回执

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# API 配置
API_BASE="http://localhost:8080/api/v1"
PRICE_WS="ws://localhost:8081/price"
API_KEY="test-key-1"

# 生成唯一标识符
DEMO_USER_ID=""
DEMO_ORDER_IDS=()
DEMO_TIMESTAMP=$(date +%s)

# 工具函数
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

    # 检查价格服务
    if curl -s -f "http://localhost:8081/health" > /dev/null; then
        log_success "价格服务运行正常"
    else
        log_error "价格服务未运行，请先启动服务"
        exit 1
    fi
}

# 调用 API 的通用函数
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    local expect_status="${4:-200}"
    local idempotency_key="${5:-}"

    local headers=(-H "Content-Type: application/json" -H "X-Api-Key: $API_KEY")
    
    # 暂时禁用幂等性键以避免中间件问题
    # if [ -n "$idempotency_key" ]; then
    #     headers+=(-H "Idempotency-Key: $idempotency_key")
    # fi

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

# 1. 创建演示用户
create_demo_user() {
    log_step "1. 创建演示用户"

    local username="demo_user_$DEMO_TIMESTAMP"
    local idempotency_key="create_user_$DEMO_TIMESTAMP"

    local request_data='{
        "username": "'$username'",
        "idempotency_key": "'$idempotency_key'"
    }'

    log_info "创建用户: $username"
    local response=$(api_call "POST" "/users" "$request_data" "201" "$idempotency_key")

    if [ $? -eq 0 ]; then
        # 检查响应是否为JSON
        if echo "$response" | jq . >/dev/null 2>&1; then
            DEMO_USER_ID=$(echo "$response" | jq -r '.user_id')
            log_success "用户创建成功: $DEMO_USER_ID"
            echo "$response" | jq '.'
        else
            log_error "响应格式错误: $response"
            exit 1
        fi
    else
        log_error "用户创建失败"
        exit 1
    fi
}

# 2. 创建钱包并入金
setup_wallets_and_deposit() {
    log_step "2. 创建钱包并入金"

    # 创建 SOL 钱包
    log_info "创建 SOL 钱包"
    local sol_wallet_request='{
        "user_id": "'$DEMO_USER_ID'",
        "asset": "SOL",
        "idempotency_key": "create_sol_wallet_'$DEMO_TIMESTAMP'"
    }'

    local response=$(api_call "POST" "/wallets" "$sol_wallet_request" "201" "create_sol_wallet_$DEMO_TIMESTAMP")
    if [ $? -eq 0 ]; then
        log_success "SOL 钱包创建成功"
        echo "$response" | jq '.'
    fi

    # 创建 USDC 钱包
    log_info "创建 USDC 钱包"
    local usdc_wallet_request='{
        "user_id": "'$DEMO_USER_ID'",
        "asset": "USDC",
        "idempotency_key": "create_usdc_wallet_'$DEMO_TIMESTAMP'"
    }'

    response=$(api_call "POST" "/wallets" "$usdc_wallet_request" "201" "create_usdc_wallet_$DEMO_TIMESTAMP")
    if [ $? -eq 0 ]; then
        log_success "USDC 钱包创建成功"
        echo "$response" | jq '.'
    fi

    # SOL 入金
    log_info "SOL 入金 10 个"
    local sol_deposit_request='{
        "user_id": "'$DEMO_USER_ID'",
        "asset": "SOL",
        "amount": "10.0",
        "idempotency_key": "deposit_sol_'$DEMO_TIMESTAMP'"
    }'

    response=$(api_call "POST" "/deposits" "$sol_deposit_request" "201" "deposit_sol_$DEMO_TIMESTAMP")
    if [ $? -eq 0 ]; then
        log_success "SOL 入金成功"
        echo "$response" | jq '.'
    fi

    # USDC 入金
    log_info "USDC 入金 1000 个"
    local usdc_deposit_request='{
        "user_id": "'$DEMO_USER_ID'",
        "asset": "USDC",
        "amount": "1000.0",
        "idempotency_key": "deposit_usdc_'$DEMO_TIMESTAMP'"
    }'

    response=$(api_call "POST" "/deposits" "$usdc_deposit_request" "201" "deposit_usdc_$DEMO_TIMESTAMP")
    if [ $? -eq 0 ]; then
        log_success "USDC 入金成功"
        echo "$response" | jq '.'
    fi

    # 查看余额
    log_info "查看用户余额"
    response=$(api_call "GET" "/balances/$DEMO_USER_ID" "" "200")
    if [ $? -eq 0 ]; then
        log_success "余额查询成功"
        echo "$response" | jq '.'
    fi
}

# 3. 创建订单（买单和卖单）
create_orders() {
    log_step "3. 创建订单（买单和卖单）"

    # 创建 SOL 卖单（当价格 <= 32.0 时卖出）
    log_info "创建 SOL 卖单 - 触发价格: 32.0 USDC，滑点容忍: 3.0%"
    local sell_order_request='{
        "user_id": "'$DEMO_USER_ID'",
        "base": "SOL",
        "quote": "USDC",
        "size": "1.0",
        "side": "sell",
        "trigger_price": "32.0",
        "trigger_op": "lte",
        "max_slippage_pct": "5.0",
        "priority_fee_lamports": 5000,
        "idempotency_key": "sell_order_'$DEMO_TIMESTAMP'"
    }'

    local response=$(api_call "POST" "/orders" "$sell_order_request" "201" "sell_order_$DEMO_TIMESTAMP")
    if [ $? -eq 0 ]; then
        local sell_order_id=$(echo "$response" | jq -r '.order_id')
        DEMO_ORDER_IDS+=("$sell_order_id")
        log_success "卖单创建成功: $sell_order_id"
        echo "$response" | jq '.'
    fi

    # 创建 SOL 买单（当价格 >= 28.0 时买入）
    log_info "创建 SOL 买单 - 触发价格: 28.0 USDC，滑点容忍: 3.0%"
    local buy_order_request='{
        "user_id": "'$DEMO_USER_ID'",
        "base": "SOL",
        "quote": "USDC",
        "size": "0.5",
        "side": "buy",
        "trigger_price": "28.0",
        "trigger_op": "gte",
        "max_slippage_pct": "5.0",
        "priority_fee_lamports": 5000,
        "idempotency_key": "buy_order_'$DEMO_TIMESTAMP'"
    }'

    response=$(api_call "POST" "/orders" "$buy_order_request" "201" "buy_order_$DEMO_TIMESTAMP")
    if [ $? -eq 0 ]; then
        local buy_order_id=$(echo "$response" | jq -r '.order_id')
        DEMO_ORDER_IDS+=("$buy_order_id")
        log_success "买单创建成功: $buy_order_id"
        echo "$response" | jq '.'
    fi

    # 查看订单列表
    log_info "查看用户订单列表"
    response=$(api_call "GET" "/orders?user_id=$DEMO_USER_ID" "" "200")
    if [ $? -eq 0 ]; then
        log_success "订单列表查询成功"
        echo "$response" | jq '.'
    fi
}

# 4. 自动化价格推送并触发订单
trigger_orders_with_price() {
    log_step "4. 自动化价格推送并触发订单"

    log_info "正在启动自动化价格推送脚本..."
    log_info "WebSocket URL: $PRICE_WS"
    
    # 检查Python是否可用
    if ! command -v python3 &> /dev/null; then
        log_error "Python3 未安装，回退到手动模式"
        manual_price_trigger
        return
    fi
    
    # 检查价格推送脚本是否存在
    if [ ! -f "scripts/price_pusher.py" ]; then
        log_error "价格推送脚本不存在: scripts/price_pusher.py"
        manual_price_trigger
        return
    fi
    
    log_info "🚀 开始自动价格推送序列..."
    log_info "📊 这将推送一系列价格来触发创建的买单和卖单"
    echo ""
    
    # 运行价格推送脚本
    if python3 scripts/price_pusher.py; then
        log_success "自动价格推送完成"
    else
        log_warning "自动价格推送失败，请检查WebSocket连接"
        manual_price_trigger
        return
    fi
    
    echo ""
    log_info "等待3秒让订单执行完成..."
    sleep 3
}

# 手动价格推送备用方案
manual_price_trigger() {
    log_warning "切换到手动价格推送模式"
    log_info "您可以使用以下工具连接价格 WebSocket:"
    echo -e "${CYAN}WebSocket URL: $PRICE_WS${NC}"
    echo -e "${CYAN}示例消息格式:${NC}"
    cat << 'EOF'
{
  "sequence_id": 12345,
  "timestamp": 1697362800000,
  "symbol": "SOL/USDC",
  "bid_price": "27.95",
  "ask_price": "28.05",
  "mid_price": "28.0",
  "source": "manual"
}
EOF

    # 模拟价格推送步骤
    log_info "建议价格变化序列:"
    echo "1. 初始价格: 30.0 USDC (无触发)"
    echo "2. 价格下跌到: 28.0 USDC (触发买单 ≥28.0)"
    echo "3. 价格上涨到: 32.0 USDC (触发卖单 ≤32.0)"

    # 等待用户操作
    echo ""
    read -p "请手动连接到价格 WebSocket 并推送价格数据，完成后按 Enter 继续..." -r
    echo ""
}

# 5. 检查订单执行状态
check_order_execution() {
    log_step "5. 检查订单执行状态"

    for order_id in "${DEMO_ORDER_IDS[@]}"; do
        log_info "检查订单状态: $order_id"
        local response=$(api_call "GET" "/orders/$order_id" "" "200")
        if [ $? -eq 0 ]; then
            local status=$(echo "$response" | jq -r '.status')
            local error_message=$(echo "$response" | jq -r '.error_message // "无"')
            log_info "订单 $order_id 状态: $status"
            if [ "$status" = "failed" ] && [ "$error_message" != "无" ] && [ "$error_message" != "null" ]; then
                log_error "订单失败原因: $error_message"
            fi
            echo "$response" | jq '.'
        fi
        echo ""
    done

    # 检查用户最新余额
    log_info "检查用户最新余额"
    local response=$(api_call "GET" "/balances/$DEMO_USER_ID" "" "200")
    if [ $? -eq 0 ]; then
        log_success "余额查询成功"
        echo "$response" | jq '.'
    fi
}

# 6. 清理演示数据
cleanup_demo_data() {
    log_step "6. 清理演示数据"

    # 取消所有未完成的订单
    for order_id in "${DEMO_ORDER_IDS[@]}"; do
        log_info "尝试取消订单: $order_id"
        
        # 先查询订单状态
        local status_response=$(api_call "GET" "/orders?user_id=$DEMO_USER_ID" "" "200")
        if [ $? -eq 0 ]; then
            local order_status=$(echo "$status_response" | jq -r --arg order_id "$order_id" '.orders[] | select(.order_id == $order_id) | .status')
            
            if [ "$order_status" = "open" ] || [ "$order_status" = "triggered" ]; then
                # 尝试取消订单
                local response=$(api_call "DELETE" "/orders/$order_id?user_id=$DEMO_USER_ID" "" "200")
                if [ $? -eq 0 ]; then
                    log_success "订单取消成功: $order_id"
                    # 检查响应是否为JSON格式
                    if echo "$response" | jq . >/dev/null 2>&1; then
                        echo "$response" | jq '.'
                    else
                        echo "响应: $response"
                    fi
                else
                    log_warning "订单取消失败: $order_id"
                fi
            else
                log_info "订单 $order_id 状态为 $order_status，无需取消"
            fi
        else
            log_warning "无法查询订单状态: $order_id"
        fi
    done

    # 删除演示用户（如果支持）
    log_info "删除演示用户: $DEMO_USER_ID"
    local response=$(api_call "DELETE" "/users/$DEMO_USER_ID" "" "204")
    if [ $? -eq 0 ]; then
        log_success "演示用户删除成功"
        # 检查响应是否为JSON格式
        if [ -n "$response" ] && echo "$response" | jq . >/dev/null 2>&1; then
            echo "$response" | jq '.'
        elif [ -n "$response" ]; then
            echo "响应: $response"
        fi
    else
        log_warning "演示用户删除失败或不支持"
    fi

    log_success "演示数据清理完成"
}

# 主要演示流程
main() {
    echo -e "${PURPLE}"
    echo "================================================================"
    echo "   Solana 限价单后端 MVP 演示脚本"
    echo "   演示流程: 入金 → 下单 → 推价触发 → 成交 → 回执"
    echo "================================================================"
    echo -e "${NC}"

    # 检查依赖工具
    if ! command -v curl &> /dev/null; then
        log_error "curl 命令未找到，请先安装 curl"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        log_error "jq 命令未找到，请先安装 jq"
        echo "Ubuntu/Debian: sudo apt-get install jq"
        echo "macOS: brew install jq"
        exit 1
    fi

    # 执行演示步骤
    check_services
    create_demo_user
    setup_wallets_and_deposit
    create_orders
    trigger_orders_with_price
    check_order_execution

    # 询问是否清理数据
    echo ""
    read -p "是否清理演示数据? (y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup_demo_data
    else
        log_info "保留演示数据，用户ID: $DEMO_USER_ID"
        log_info "订单IDs: ${DEMO_ORDER_IDS[*]}"
    fi

    echo ""
    log_success "演示完成！"
    echo -e "${PURPLE}感谢使用 Solana 限价单后端 MVP 演示脚本${NC}"
}

# 异常处理
trap 'echo -e "\n${RED}演示中断${NC}"; exit 1' INT TERM

# 运行主函数
main "$@"

