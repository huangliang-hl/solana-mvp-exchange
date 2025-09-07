package repository

import (
	"context"
	"fmt"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// WalletRepository 钱包仓储接口
type WalletRepository interface {
	GetByUserIDAndAsset(ctx context.Context, userID uuid.UUID, asset string) (*types.Wallet, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*types.Wallet, error)
	Create(ctx context.Context, wallet *types.Wallet) error
	CreateWithTx(ctx context.Context, tx pgx.Tx, wallet *types.Wallet) error
	UpdateBalances(ctx context.Context, userID uuid.UUID, asset string, availableDelta, lockedDelta decimal.Decimal) error
	UpdateBalancesWithTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string, availableDelta, lockedDelta decimal.Decimal) (*types.Wallet, error)
	LockForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string) (*types.Wallet, error)
}

// walletRepository 钱包仓储实现
type walletRepository struct {
	db *DB
}

// NewWalletRepository 创建钱包仓储
func NewWalletRepository(db *DB) WalletRepository {
	return &walletRepository{db: db}
}

// GetByUserIDAndAsset 根据用户 ID 和资产类型获取钱包
func (r *walletRepository) GetByUserIDAndAsset(ctx context.Context, userID uuid.UUID, asset string) (*types.Wallet, error) {
	query := `
		SELECT id, user_id, asset, available_decimal, locked_decimal, updated_at
		FROM wallets 
		WHERE user_id = $1 AND asset = $2
	`

	var wallet types.Wallet
	err := r.db.QueryRow(ctx, query, userID, asset).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Asset,
		&wallet.AvailableDecimal,
		&wallet.LockedDecimal,
		&wallet.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("钱包不存在: user_id=%s, asset=%s", userID, asset)
		}
		return nil, fmt.Errorf("查询钱包失败: %w", err)
	}

	return &wallet, nil
}

// GetByUserID 根据用户 ID 获取所有钱包
func (r *walletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*types.Wallet, error) {
	query := `
		SELECT id, user_id, asset, available_decimal, locked_decimal, updated_at
		FROM wallets 
		WHERE user_id = $1
		ORDER BY asset
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户钱包失败: %w", err)
	}
	defer rows.Close()

	var wallets []*types.Wallet
	for rows.Next() {
		var wallet types.Wallet
		err := rows.Scan(
			&wallet.ID,
			&wallet.UserID,
			&wallet.Asset,
			&wallet.AvailableDecimal,
			&wallet.LockedDecimal,
			&wallet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描钱包数据失败: %w", err)
		}
		wallets = append(wallets, &wallet)
	}

	return wallets, nil
}

// Create 创建钱包
func (r *walletRepository) Create(ctx context.Context, wallet *types.Wallet) error {
	query := `
		INSERT INTO wallets (id, user_id, asset, available_decimal, locked_decimal, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		wallet.ID,
		wallet.UserID,
		wallet.Asset,
		wallet.AvailableDecimal,
		wallet.LockedDecimal,
		wallet.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建钱包失败: %w", err)
	}

	return nil
}

// CreateWithTx 在事务中创建钱包
func (r *walletRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, wallet *types.Wallet) error {
	query := `
		INSERT INTO wallets (id, user_id, asset, available_decimal, locked_decimal, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := tx.Exec(ctx, query,
		wallet.ID,
		wallet.UserID,
		wallet.Asset,
		wallet.AvailableDecimal,
		wallet.LockedDecimal,
		wallet.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建钱包失败: %w", err)
	}

	return nil
}

// UpdateBalances 更新余额
func (r *walletRepository) UpdateBalances(ctx context.Context, userID uuid.UUID, asset string, availableDelta, lockedDelta decimal.Decimal) error {
	query := `
		UPDATE wallets 
		SET available_decimal = available_decimal + $3,
		    locked_decimal = locked_decimal + $4,
		    updated_at = NOW()
		WHERE user_id = $1 AND asset = $2
	`

	result, err := r.db.Pool.Exec(ctx, query, userID, asset, availableDelta, lockedDelta)
	if err != nil {
		return fmt.Errorf("更新钱包余额失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("钱包不存在或更新失败: user_id=%s, asset=%s", userID, asset)
	}

	return nil
}

// UpdateBalancesWithTx 在事务中更新余额并返回更新后的钱包
func (r *walletRepository) UpdateBalancesWithTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string, availableDelta, lockedDelta decimal.Decimal) (*types.Wallet, error) {
	query := `
		UPDATE wallets 
		SET available_decimal = available_decimal + $3,
		    locked_decimal = locked_decimal + $4,
		    updated_at = NOW()
		WHERE user_id = $1 AND asset = $2
		RETURNING id, user_id, asset, available_decimal, locked_decimal, updated_at
	`

	var wallet types.Wallet
	err := tx.QueryRow(ctx, query, userID, asset, availableDelta, lockedDelta).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Asset,
		&wallet.AvailableDecimal,
		&wallet.LockedDecimal,
		&wallet.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("钱包不存在: user_id=%s, asset=%s", userID, asset)
		}
		return nil, fmt.Errorf("更新钱包余额失败: %w", err)
	}

	return &wallet, nil
}

// LockForUpdate 锁定钱包记录用于更新（SELECT FOR UPDATE）
func (r *walletRepository) LockForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string) (*types.Wallet, error) {
	query := `
		SELECT id, user_id, asset, available_decimal, locked_decimal, updated_at
		FROM wallets 
		WHERE user_id = $1 AND asset = $2
		FOR UPDATE
	`

	var wallet types.Wallet
	err := tx.QueryRow(ctx, query, userID, asset).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Asset,
		&wallet.AvailableDecimal,
		&wallet.LockedDecimal,
		&wallet.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("钱包不存在: user_id=%s, asset=%s", userID, asset)
		}
		return nil, fmt.Errorf("锁定钱包失败: %w", err)
	}

	return &wallet, nil
}
