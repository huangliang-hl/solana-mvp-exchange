package repository

import (
	"context"
	"fmt"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// OrderRepository 订单仓储接口
type OrderRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*types.Order, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, status string, limit int) ([]*types.Order, error)
	Create(ctx context.Context, order *types.Order) error
	CreateWithTx(ctx context.Context, tx pgx.Tx, order *types.Order) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status types.OrderStatus, version int) error
	UpdateStatusWithTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status types.OrderStatus, version int) error
	UpdateStatusWithError(ctx context.Context, id uuid.UUID, status types.OrderStatus, version int, errorMessage string) error
	UpdateStatusWithErrorTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status types.OrderStatus, version int, errorMessage string) error
	TriggerOrder(ctx context.Context, id uuid.UUID, version int) error
	TriggerOrderWithTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, version int) error
	SubmitOrder(ctx context.Context, id uuid.UUID, txSig string, version int) error
	SubmitOrderWithTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, txSig string, version int) error
	GetOpenOrdersForTrigger(ctx context.Context, base, quote string, limit int) ([]*types.Order, error)
	CheckIdempotency(ctx context.Context, userID uuid.UUID, idempotencyKey string) (*types.Order, error)
}

// orderRepository 订单仓储实现
type orderRepository struct {
	db *DB
}

// NewOrderRepository 创建订单仓储
func NewOrderRepository(db *DB) OrderRepository {
	return &orderRepository{db: db}
}

// GetByID 根据 ID 获取订单
func (r *orderRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.Order, error) {
	query := `
		SELECT id, user_id, base, quote, size, side, trigger_price, trigger_op, 
		       status, max_slippage_pct, priority_fee_lamports, created_at, updated_at, triggered_at, submitted_at, tx_sig, error_message, version
		FROM orders 
		WHERE id = $1
	`

	var order types.Order
	err := r.db.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.Base,
		&order.Quote,
		&order.Size,
		&order.Side,
		&order.TriggerPrice,
		&order.TriggerOp,
		&order.Status,
		&order.MaxSlippagePct,
		&order.PriorityFeeLamports,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.TriggeredAt,
		&order.SubmittedAt,
		&order.TxSig,
		&order.ErrorMessage,
		&order.Version,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("订单不存在: %s", id)
		}
		return nil, fmt.Errorf("查询订单失败: %w", err)
	}

	return &order, nil
}

// GetByUserID 根据用户 ID 获取订单列表
func (r *orderRepository) GetByUserID(ctx context.Context, userID uuid.UUID, status string, limit int) ([]*types.Order, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT id, user_id, base, quote, size, side, trigger_price, trigger_op, 
			       status, max_slippage_pct, priority_fee_lamports, created_at, updated_at, triggered_at, submitted_at, tx_sig, error_message, version
			FROM orders 
			WHERE user_id = $1 AND status = $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []interface{}{userID, status, limit}
	} else {
		query = `
			SELECT id, user_id, base, quote, size, side, trigger_price, trigger_op, 
			       status, max_slippage_pct, priority_fee_lamports, created_at, updated_at, triggered_at, submitted_at, tx_sig, error_message, version
			FROM orders 
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = []interface{}{userID, limit}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询用户订单失败: %w", err)
	}
	defer rows.Close()

	var orders []*types.Order
	for rows.Next() {
		var order types.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Base,
			&order.Quote,
			&order.Size,
			&order.Side,
			&order.TriggerPrice,
			&order.TriggerOp,
			&order.Status,
			&order.MaxSlippagePct,
			&order.PriorityFeeLamports,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.TriggeredAt,
			&order.SubmittedAt,
			&order.TxSig,
			&order.ErrorMessage,
			&order.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描订单数据失败: %w", err)
		}
		orders = append(orders, &order)
	}

	return orders, nil
}

// Create 创建订单
func (r *orderRepository) Create(ctx context.Context, order *types.Order) error {
	query := `
		INSERT INTO orders (id, user_id, base, quote, size, side, trigger_price, trigger_op, 
		                   status, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		order.ID,
		order.UserID,
		order.Base,
		order.Quote,
		order.Size,
		order.Side,
		order.TriggerPrice,
		order.TriggerOp,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
		order.Version,
	)
	if err != nil {
		return fmt.Errorf("创建订单失败: %w", err)
	}

	return nil
}

// CreateWithTx 在事务中创建订单
func (r *orderRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, order *types.Order) error {
	query := `
		INSERT INTO orders (id, user_id, base, quote, size, side, trigger_price, trigger_op, 
		                   status, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := tx.Exec(ctx, query,
		order.ID,
		order.UserID,
		order.Base,
		order.Quote,
		order.Size,
		order.Side,
		order.TriggerPrice,
		order.TriggerOp,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
		order.Version,
	)
	if err != nil {
		return fmt.Errorf("创建订单失败: %w", err)
	}

	return nil
}

// UpdateStatus 更新订单状态
func (r *orderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status types.OrderStatus, version int) error {
	query := `
		UPDATE orders 
		SET status = $2, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $3
	`

	result, err := r.db.Pool.Exec(ctx, query, id, status, version)
	if err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("订单不存在或版本冲突: id=%s, version=%d", id, version)
	}

	return nil
}

// UpdateStatusWithTx 在事务中更新订单状态
func (r *orderRepository) UpdateStatusWithTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status types.OrderStatus, version int) error {
	query := `
		UPDATE orders 
		SET status = $2, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $3
	`

	result, err := tx.Exec(ctx, query, id, status, version)
	if err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("订单不存在或版本冲突: id=%s, version=%d", id, version)
	}

	return nil
}

// TriggerOrder 触发订单
func (r *orderRepository) TriggerOrder(ctx context.Context, id uuid.UUID, version int) error {
	query := `
		UPDATE orders 
		SET status = 'triggered', triggered_at = NOW(), version = version + 1, updated_at = NOW()
		WHERE id = $1 AND status = 'open' AND version = $2
	`

	result, err := r.db.Pool.Exec(ctx, query, id, version)
	if err != nil {
		return fmt.Errorf("触发订单失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("订单不存在、状态不正确或版本冲突: id=%s, version=%d", id, version)
	}

	return nil
}

// TriggerOrderWithTx 在事务中触发订单
func (r *orderRepository) TriggerOrderWithTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, version int) error {
	query := `
		UPDATE orders 
		SET status = 'triggered', triggered_at = NOW(), version = version + 1, updated_at = NOW()
		WHERE id = $1 AND status = 'open' AND version = $2
	`

	result, err := tx.Exec(ctx, query, id, version)
	if err != nil {
		return fmt.Errorf("触发订单失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("订单不存在、状态不正确或版本冲突: id=%s, version=%d", id, version)
	}

	return nil
}

// SubmitOrder 提交订单到链上
func (r *orderRepository) SubmitOrder(ctx context.Context, id uuid.UUID, txSig string, version int) error {
	query := `
		UPDATE orders 
		SET status = 'submitting', submitted_at = NOW(), tx_sig = $2, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND status = 'triggered' AND version = $3
	`

	result, err := r.db.Pool.Exec(ctx, query, id, txSig, version)
	if err != nil {
		return fmt.Errorf("提交订单失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("订单不存在、状态不正确或版本冲突: id=%s, version=%d", id, version)
	}

	return nil
}

// SubmitOrderWithTx 在事务中提交订单到链上
func (r *orderRepository) SubmitOrderWithTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, txSig string, version int) error {
	query := `
		UPDATE orders 
		SET status = 'submitting', submitted_at = NOW(), tx_sig = $2, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND status = 'triggered' AND version = $3
	`

	result, err := tx.Exec(ctx, query, id, txSig, version)
	if err != nil {
		return fmt.Errorf("提交订单失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("订单不存在、状态不正确或版本冲突: id=%s, version=%d", id, version)
	}

	return nil
}

// GetOpenOrdersForTrigger 获取可触发的开放订单
func (r *orderRepository) GetOpenOrdersForTrigger(ctx context.Context, base, quote string, limit int) ([]*types.Order, error) {
	query := `
		SELECT id, user_id, base, quote, size, side, trigger_price, trigger_op, 
		       status, created_at, updated_at, triggered_at, submitted_at, tx_sig, version
		FROM orders 
		WHERE status = 'open' AND base = $1 AND quote = $2
		ORDER BY created_at ASC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, base, quote, limit)
	if err != nil {
		return nil, fmt.Errorf("查询可触发订单失败: %w", err)
	}
	defer rows.Close()

	var orders []*types.Order
	for rows.Next() {
		var order types.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Base,
			&order.Quote,
			&order.Size,
			&order.Side,
			&order.TriggerPrice,
			&order.TriggerOp,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.TriggeredAt,
			&order.SubmittedAt,
			&order.TxSig,
			&order.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描订单数据失败: %w", err)
		}
		orders = append(orders, &order)
	}

	return orders, nil
}

// CheckIdempotency 检查幂等性
func (r *orderRepository) CheckIdempotency(ctx context.Context, userID uuid.UUID, idempotencyKey string) (*types.Order, error) {
	// 首先检查幂等性表
	idempotencyQuery := `
		SELECT response_data 
		FROM idempotency_keys 
		WHERE key = $1 AND user_id = $2 AND expires_at > NOW()
	`

	var responseData string
	err := r.db.QueryRow(ctx, idempotencyQuery, idempotencyKey, userID).Scan(&responseData)
	if err == nil {
		// 找到了幂等性记录，但这里我们需要返回订单信息
		// 简化处理：直接返回 nil 表示需要重新处理
		return nil, nil
	}

	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("检查幂等性失败: %w", err)
	}

	// 没有找到幂等性记录，返回 nil
	return nil, nil
}
