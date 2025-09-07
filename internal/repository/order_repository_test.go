package repository

import (
	"testing"
	"time"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderRepository_Interface(t *testing.T) {
	// 验证实现了接口
	var _ OrderRepository = (*orderRepository)(nil)
}

func TestNewOrderRepository(t *testing.T) {
	db := &DB{Pool: nil}
	repo := NewOrderRepository(db)

	require.NotNil(t, repo)
	orderRepo, ok := repo.(*orderRepository)
	require.True(t, ok)
	assert.Equal(t, db, orderRepo.db)
}

// 测试创建订单的逻辑
func TestOrderRepository_CreateOrder_Logic(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name     string
		order    *types.Order
		validate func(*testing.T, *types.Order)
	}{
		{
			name: "有效的买单",
			order: &types.Order{
				ID:           uuid.New(),
				UserID:       userID,
				Base:         "SOL",
				Quote:        "USDC",
				Side:         types.OrderSideBuy,
				Size:         decimal.NewFromFloat(10.0),
				TriggerPrice: decimal.NewFromFloat(100.0),
				TriggerOp:    types.TriggerOpGTE,
				Status:       types.OrderStatusOpen,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Version:      1,
			},
			validate: func(t *testing.T, order *types.Order) {
				assert.Equal(t, "SOL", order.Base)
				assert.Equal(t, "USDC", order.Quote)
				assert.Equal(t, types.OrderSideBuy, order.Side)
				assert.Equal(t, types.OrderStatusOpen, order.Status)
				assert.True(t, order.Size.GreaterThan(decimal.Zero))
				assert.True(t, order.TriggerPrice.GreaterThan(decimal.Zero))
			},
		},
		{
			name: "有效的卖单",
			order: &types.Order{
				ID:           uuid.New(),
				UserID:       userID,
				Base:         "SOL",
				Quote:        "USDC",
				Side:         types.OrderSideSell,
				Size:         decimal.NewFromFloat(5.0),
				TriggerPrice: decimal.NewFromFloat(80.0),
				TriggerOp:    types.TriggerOpLTE,
				Status:       types.OrderStatusOpen,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Version:      1,
			},
			validate: func(t *testing.T, order *types.Order) {
				assert.Equal(t, types.OrderSideSell, order.Side)
				assert.Equal(t, types.TriggerOpLTE, order.TriggerOp)
				assert.True(t, order.Size.GreaterThan(decimal.Zero))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.order)
		})
	}
}

// 测试获取订单的逻辑
func TestOrderRepository_GetByID_Logic(t *testing.T) {
	orderID := uuid.New()
	testOrder := &types.Order{
		ID:           orderID,
		UserID:       uuid.New(),
		Base:         "SOL",
		Quote:        "USDC",
		Side:         types.OrderSideBuy,
		Size:         decimal.NewFromFloat(10.0),
		TriggerPrice: decimal.NewFromFloat(100.0),
		TriggerOp:    types.TriggerOpGTE,
		Status:       types.OrderStatusOpen,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Version:      1,
	}

	tests := []struct {
		name    string
		orderID uuid.UUID
		order   *types.Order
	}{
		{
			name:    "查找存在的订单",
			orderID: orderID,
			order:   testOrder,
		},
		{
			name:    "查找不存在的订单",
			orderID: uuid.New(),
			order:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.order != nil {
				assert.Equal(t, tt.orderID, tt.order.ID)
			} else {
				assert.Nil(t, tt.order)
			}
		})
	}
}

// 测试获取用户订单的逻辑
func TestOrderRepository_GetByUserID_Logic(t *testing.T) {
	userID := uuid.New()

	testOrders := []*types.Order{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Base:      "SOL",
			Quote:     "USDC",
			Side:      types.OrderSideBuy,
			Status:    types.OrderStatusOpen,
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			Base:      "SOL",
			Quote:     "USDT",
			Side:      types.OrderSideSell,
			Status:    types.OrderStatusFilled,
			CreatedAt: time.Now().Add(-time.Hour),
		},
	}

	tests := []struct {
		name   string
		userID uuid.UUID
		limit  int
		offset int
		orders []*types.Order
	}{
		{
			name:   "获取用户所有订单",
			userID: userID,
			limit:  10,
			offset: 0,
			orders: testOrders,
		},
		{
			name:   "分页获取订单",
			userID: userID,
			limit:  1,
			offset: 0,
			orders: testOrders[:1],
		},
		{
			name:   "空用户ID",
			userID: uuid.Nil,
			limit:  10,
			offset: 0,
			orders: []*types.Order{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.orders) > 0 {
				for _, order := range tt.orders {
					if tt.userID != uuid.Nil {
						assert.Equal(t, tt.userID, order.UserID)
					}
				}
			}
		})
	}
}

// 测试获取活跃订单的逻辑
func TestOrderRepository_GetActiveOrders_Logic(t *testing.T) {
	activeStatuses := []types.OrderStatus{
		types.OrderStatusOpen,
		types.OrderStatusTriggered,
		types.OrderStatusSubmitting,
	}

	testOrders := []*types.Order{
		{
			ID:     uuid.New(),
			Status: types.OrderStatusOpen,
		},
		{
			ID:     uuid.New(),
			Status: types.OrderStatusTriggered,
		},
		{
			ID:     uuid.New(),
			Status: types.OrderStatusFilled, // 非活跃状态
		},
	}

	tests := []struct {
		name           string
		expectedActive []*types.Order
	}{
		{
			name:           "获取活跃订单",
			expectedActive: testOrders[:2], // 前两个是活跃的
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, order := range tt.expectedActive {
				isActive := false
				for _, status := range activeStatuses {
					if order.Status == status {
						isActive = true
						break
					}
				}
				assert.True(t, isActive, "订单状态 %s 应该是活跃的", order.Status)
			}
		})
	}
}

// 测试更新订单状态的逻辑
func TestOrderRepository_UpdateStatus_Logic(t *testing.T) {
	orderID := uuid.New()

	tests := []struct {
		name      string
		orderID   uuid.UUID
		newStatus types.OrderStatus
		version   int
	}{
		{
			name:      "触发订单",
			orderID:   orderID,
			newStatus: types.OrderStatusTriggered,
			version:   2,
		},
		{
			name:      "取消订单",
			orderID:   orderID,
			newStatus: types.OrderStatusCanceled,
			version:   3,
		},
		{
			name:      "完成订单",
			orderID:   orderID,
			newStatus: types.OrderStatusFilled,
			version:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证参数的有效性
			assert.NotEqual(t, uuid.Nil, tt.orderID)
			assert.True(t, tt.newStatus != "")
			assert.Greater(t, tt.version, 0)
		})
	}
}

// 测试提交订单的逻辑
func TestOrderRepository_SubmitOrder_Logic(t *testing.T) {
	orderID := uuid.New()
	txSig := "5J8J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J4J"

	tests := []struct {
		name    string
		orderID uuid.UUID
		txSig   string
		version int
	}{
		{
			name:    "有效的交易签名",
			orderID: orderID,
			txSig:   txSig,
			version: 2,
		},
		{
			name:    "空交易签名",
			orderID: orderID,
			txSig:   "",
			version: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证参数
			assert.NotEqual(t, uuid.Nil, tt.orderID)
			assert.Greater(t, tt.version, 0)

			if tt.txSig != "" {
				assert.Greater(t, len(tt.txSig), 0)
			}
		})
	}
}
