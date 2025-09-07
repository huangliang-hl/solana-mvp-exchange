-- 为 orders 表添加 error_message 字段以记录失败原因
ALTER TABLE orders ADD COLUMN error_message TEXT NULL;

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_orders_error_message ON orders(error_message) WHERE error_message IS NOT NULL;

-- 为 tx_attempts 表也添加错误信息字段
ALTER TABLE tx_attempts ADD COLUMN error_message TEXT NULL;
