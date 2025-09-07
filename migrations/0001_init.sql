-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 钱包表
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    asset VARCHAR(10) NOT NULL,
    available_decimal NUMERIC(20,8) NOT NULL DEFAULT 0,
    locked_decimal NUMERIC(20,8) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, asset)
);

-- 订单表
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    base VARCHAR(10) NOT NULL,
    quote VARCHAR(10) NOT NULL,
    size NUMERIC(20,8) NOT NULL,
    side VARCHAR(4) NOT NULL CHECK (side IN ('buy', 'sell')),
    trigger_price NUMERIC(20,8) NOT NULL,
    trigger_op VARCHAR(3) NOT NULL CHECK (trigger_op IN ('gte', 'lte')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('open', 'triggered', 'submitting', 'filled', 'failed', 'canceled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    triggered_at TIMESTAMPTZ NULL,
    submitted_at TIMESTAMPTZ NULL,
    tx_sig VARCHAR(100) NULL,
    version INTEGER NOT NULL DEFAULT 0
);

-- 账本流水表
CREATE TABLE IF NOT EXISTS ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    asset VARCHAR(10) NOT NULL,
    delta NUMERIC(20,8) NOT NULL,
    balance_after NUMERIC(20,8) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('lock_funds', 'release_funds', 'filled', 'compensate', 'deposit', 'withdraw')),
    ref_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 交易尝试表（用于幂等与重试追踪）
CREATE TABLE IF NOT EXISTS tx_attempts (
    attempt_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    params JSONB NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'submitted', 'confirmed', 'failed')),
    tx_sig VARCHAR(100) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 价格 tick 表（用于持久化价格数据）
CREATE TABLE IF NOT EXISTS price_ticks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sequence_id BIGINT NOT NULL,
    timestamp BIGINT NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    bid_price NUMERIC(20,8) NOT NULL,
    ask_price NUMERIC(20,8) NOT NULL,
    mid_price NUMERIC(20,8) NOT NULL,
    source VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 幂等性表（用于 API 幂等性控制）
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key VARCHAR(255) PRIMARY KEY,
    user_id UUID NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    response_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_wallets_user_asset ON wallets(user_id, asset);
CREATE INDEX IF NOT EXISTS idx_orders_user_status ON orders(user_id, status);
CREATE INDEX IF NOT EXISTS idx_orders_trigger ON orders(status, base, quote, trigger_op, trigger_price) WHERE status = 'open';
CREATE INDEX IF NOT EXISTS idx_orders_status_version ON orders(status, version) WHERE status IN ('triggered', 'submitting');
CREATE INDEX IF NOT EXISTS idx_ledger_user_time ON ledger_entries(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tx_attempts_order ON tx_attempts(order_id);
CREATE INDEX IF NOT EXISTS idx_price_ticks_symbol_time ON price_ticks(symbol, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON idempotency_keys(expires_at);

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要自动更新 updated_at 的表创建触发器
CREATE TRIGGER update_wallets_updated_at BEFORE UPDATE ON wallets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 创建清理过期幂等性键的函数
CREATE OR REPLACE FUNCTION cleanup_expired_idempotency_keys()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM idempotency_keys WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- 插入测试用户和初始余额
INSERT INTO users (id, username) VALUES 
    ('550e8400-e29b-41d4-a716-446655440000', 'test_user_1'),
    ('550e8400-e29b-41d4-a716-446655440001', 'test_user_2')
ON CONFLICT (username) DO NOTHING;

-- 插入测试钱包余额
INSERT INTO wallets (user_id, asset, available_decimal, locked_decimal) VALUES 
    ('550e8400-e29b-41d4-a716-446655440000', 'SOL', 100.0, 0.0),
    ('550e8400-e29b-41d4-a716-446655440000', 'USDC', 10000.0, 0.0),
    ('550e8400-e29b-41d4-a716-446655440001', 'SOL', 50.0, 0.0),
    ('550e8400-e29b-41d4-a716-446655440001', 'USDC', 5000.0, 0.0)
ON CONFLICT (user_id, asset) DO NOTHING;
