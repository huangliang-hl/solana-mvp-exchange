package repository

import (
	"context"
	"fmt"
	"time"

	"solana-limit-order-backend/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// DB 数据库连接池包装器
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB 创建新的数据库连接
func NewDB(databaseURL string, cfg *config.Config) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("解析数据库 URL 失败: %w", err)
	}

	// 使用配置文件中的连接池参数
	poolConfig.MaxConns = int32(cfg.DBMaxConns)
	poolConfig.MinConns = int32(cfg.DBMinConns)
	poolConfig.MaxConnLifetime = time.Duration(cfg.DBMaxConnLifetime) * time.Hour
	poolConfig.MaxConnIdleTime = time.Duration(cfg.DBMaxConnIdleTime) * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("创建数据库连接池失败: %w", err)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Info().Msg("数据库连接成功")
	return &DB{Pool: pool}, nil
}

// Close 关闭数据库连接
func (db *DB) Close() {
	db.Pool.Close()
	log.Info().Msg("数据库连接已关闭")
}

// BeginTx 开始事务
func (db *DB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.Pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, // 使用可重复读隔离级别
	})
}

// WithTx 在事务中执行函数
func (db *DB) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			// 事务可能已经提交或回滚，忽略错误
			_ = err
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// Exec 执行 SQL 语句
func (db *DB) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := db.Pool.Exec(ctx, sql, args...)
	return err
}

// QueryRow 查询单行
func (db *DB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.Pool.QueryRow(ctx, sql, args...)
}

// Query 查询多行
func (db *DB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.Pool.Query(ctx, sql, args...)
}

// Health 检查数据库健康状态
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Stats 获取连接池统计信息
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}
