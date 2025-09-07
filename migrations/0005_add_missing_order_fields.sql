-- 为 orders 表添加缺失的字段
-- 添加最大滑点百分比字段
ALTER TABLE orders ADD COLUMN max_slippage_pct NUMERIC(5,2) NOT NULL DEFAULT 1.0;

-- 添加优先费用lamports字段
ALTER TABLE orders ADD COLUMN priority_fee_lamports BIGINT NOT NULL DEFAULT 0;

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_orders_max_slippage ON orders(max_slippage_pct);
CREATE INDEX IF NOT EXISTS idx_orders_priority_fee ON orders(priority_fee_lamports);

-- 更新现有订单的默认值
UPDATE orders SET max_slippage_pct = 1.0 WHERE max_slippage_pct IS NULL;
UPDATE orders SET priority_fee_lamports = 0 WHERE priority_fee_lamports IS NULL;
