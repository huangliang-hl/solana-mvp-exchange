-- 测试数据初始化脚本
-- 此脚本用于在测试环境中创建初始数据

-- 清理所有测试数据（仅当表存在时）
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'idempotency_keys') THEN
        DELETE FROM idempotency_keys;
    END IF;
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'price_ticks') THEN
        DELETE FROM price_ticks;
    END IF;
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'tx_attempts') THEN
        DELETE FROM tx_attempts;
    END IF;
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'ledger_entries') THEN
        DELETE FROM ledger_entries;
    END IF;
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'orders') THEN
        DELETE FROM orders;
    END IF;
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'wallets') THEN
        DELETE FROM wallets;
    END IF;
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users') THEN
        DELETE FROM users;
    END IF;
END
$$;

-- 插入测试用户
INSERT INTO users (id, username) VALUES 
    ('550e8400-e29b-41d4-a716-446655440000', 'test_user_1'),
    ('550e8400-e29b-41d4-a716-446655440001', 'test_user_2'),
    ('550e8400-e29b-41d4-a716-446655440002', 'integration_test_user')
ON CONFLICT (username) DO UPDATE SET username = EXCLUDED.username;

-- 插入测试钱包余额
INSERT INTO wallets (user_id, asset, available_decimal, locked_decimal) VALUES 
    ('550e8400-e29b-41d4-a716-446655440000', 'SOL', 100.0, 0.0),
    ('550e8400-e29b-41d4-a716-446655440000', 'USDC', 10000.0, 0.0),
    ('550e8400-e29b-41d4-a716-446655440001', 'SOL', 50.0, 0.0),
    ('550e8400-e29b-41d4-a716-446655440001', 'USDC', 5000.0, 0.0),
    ('550e8400-e29b-41d4-a716-446655440002', 'SOL', 200.0, 0.0),
    ('550e8400-e29b-41d4-a716-446655440002', 'USDC', 20000.0, 0.0)
ON CONFLICT (user_id, asset) DO UPDATE SET 
    available_decimal = EXCLUDED.available_decimal,
    locked_decimal = EXCLUDED.locked_decimal;
