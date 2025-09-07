package repository

import (
	"context"
	"fmt"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UpdateStatusWithError 更新订单状态并记录错误信息
func (r *orderRepository) UpdateStatusWithError(ctx context.Context, id uuid.UUID, status types.OrderStatus, version int, errorMessage string) error {
	query := `
		UPDATE orders 
		SET status = $2, error_message = $3, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $4
	`

	result, err := r.db.Pool.Exec(ctx, query, id, status, errorMessage, version)
	if err != nil {
		return fmt.Errorf("更新订单状态和错误信息失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("订单不存在或版本冲突: id=%s, version=%d", id, version)
	}

	return nil
}

// UpdateStatusWithErrorTx 在事务中更新订单状态并记录错误信息
func (r *orderRepository) UpdateStatusWithErrorTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status types.OrderStatus, version int, errorMessage string) error {
	query := `
		UPDATE orders 
		SET status = $2, error_message = $3, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $4
	`

	result, err := tx.Exec(ctx, query, id, status, errorMessage, version)
	if err != nil {
		return fmt.Errorf("更新订单状态和错误信息失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("订单不存在或版本冲突: id=%s, version=%d", id, version)
	}

	return nil
}
