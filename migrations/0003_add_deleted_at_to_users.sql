-- 为 users 表添加 deleted_at 字段以支持软删除
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ NULL;

-- 创建索引以提高软删除查询性能
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- 更新现有的查询函数，以排除已删除的用户
