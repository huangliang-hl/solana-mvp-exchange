package repository

import (
	"context"
	"fmt"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// LedgerRepository 账本仓储接口
type LedgerRepository interface {
	Create(ctx context.Context, entry *types.LedgerEntry) error
	CreateWithTx(ctx context.Context, tx pgx.Tx, entry *types.LedgerEntry) error
	GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*types.LedgerEntry, error)
	GetByRefID(ctx context.Context, refID uuid.UUID) ([]*types.LedgerEntry, error)
}

// ledgerRepository 账本仓储实现
type ledgerRepository struct {
	db *DB
}

// NewLedgerRepository 创建账本仓储
func NewLedgerRepository(db *DB) LedgerRepository {
	return &ledgerRepository{db: db}
}

// Create 创建账本条目
func (r *ledgerRepository) Create(ctx context.Context, entry *types.LedgerEntry) error {
	query := `
		INSERT INTO ledger_entries (id, user_id, asset, delta, balance_after, type, ref_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		entry.ID,
		entry.UserID,
		entry.Asset,
		entry.Delta,
		entry.BalanceAfter,
		entry.Type,
		entry.RefID,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建账本条目失败: %w", err)
	}

	return nil
}

// CreateWithTx 在事务中创建账本条目
func (r *ledgerRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, entry *types.LedgerEntry) error {
	query := `
		INSERT INTO ledger_entries (id, user_id, asset, delta, balance_after, type, ref_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := tx.Exec(ctx, query,
		entry.ID,
		entry.UserID,
		entry.Asset,
		entry.Delta,
		entry.BalanceAfter,
		entry.Type,
		entry.RefID,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建账本条目失败: %w", err)
	}

	return nil
}

// GetByUserID 根据用户 ID 获取账本条目
func (r *ledgerRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*types.LedgerEntry, error) {
	query := `
		SELECT id, user_id, asset, delta, balance_after, type, ref_id, created_at
		FROM ledger_entries 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("查询用户账本条目失败: %w", err)
	}
	defer rows.Close()

	var entries []*types.LedgerEntry
	for rows.Next() {
		var entry types.LedgerEntry
		err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.Asset,
			&entry.Delta,
			&entry.BalanceAfter,
			&entry.Type,
			&entry.RefID,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描账本条目数据失败: %w", err)
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}

// GetByRefID 根据关联 ID 获取账本条目
func (r *ledgerRepository) GetByRefID(ctx context.Context, refID uuid.UUID) ([]*types.LedgerEntry, error) {
	query := `
		SELECT id, user_id, asset, delta, balance_after, type, ref_id, created_at
		FROM ledger_entries 
		WHERE ref_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, refID)
	if err != nil {
		return nil, fmt.Errorf("查询关联账本条目失败: %w", err)
	}
	defer rows.Close()

	var entries []*types.LedgerEntry
	for rows.Next() {
		var entry types.LedgerEntry
		err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.Asset,
			&entry.Delta,
			&entry.BalanceAfter,
			&entry.Type,
			&entry.RefID,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描账本条目数据失败: %w", err)
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}
