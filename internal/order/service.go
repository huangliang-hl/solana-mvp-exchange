package order

import (
	"context"
	"fmt"
	"time"

	"solana-limit-order-backend/internal/ledger"
	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Service 订单服务接口
type Service interface {
	// CreateOrder 创建订单
	CreateOrder(ctx context.Context, req *types.CreateOrderRequest) (*types.CreateOrderResponse, error)

	// CancelOrder 取消订单
	CancelOrder(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) (*types.CancelOrderResponse, error)

	// GetOrders 查询订单
	GetOrders(ctx context.Context, userID uuid.UUID, status string, limit int) (*types.OrderQueryResponse, error)

	// GetOrderByID 根据 ID 获取订单
	GetOrderByID(ctx context.Context, orderID uuid.UUID) (*types.Order, error)

	// ValidateOrderRequest 验证订单请求
	ValidateOrderRequest(req *types.CreateOrderRequest) error
}

// service 订单服务实现
type service struct {
	orderRepo     repository.OrderRepository
	userRepo      repository.UserRepository
	ledgerService ledger.Service
}

// NewService 创建订单服务
func NewService(orderRepo repository.OrderRepository, userRepo repository.UserRepository, ledgerService ledger.Service) Service {
	return &service{
		orderRepo:     orderRepo,
		userRepo:      userRepo,
		ledgerService: ledgerService,
	}
}

// CreateOrder 创建订单
func (s *service) CreateOrder(ctx context.Context, req *types.CreateOrderRequest) (*types.CreateOrderResponse, error) {
	// 验证请求参数
	if err := s.ValidateOrderRequest(req); err != nil {
		return nil, fmt.Errorf("订单参数验证失败: %w", err)
	}

	// 解析用户 ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户 ID: %w", err)
	}

	// 检查用户是否存在
	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 检查幂等性
	existingOrder, err := s.orderRepo.CheckIdempotency(ctx, userID, req.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("检查幂等性失败: %w", err)
	}
	if existingOrder != nil {
		// 返回已存在的订单
		balances, _ := s.ledgerService.GetBalances(ctx, userID)
		return &types.CreateOrderResponse{
			OrderID:           existingOrder.ID.String(),
			Status:            string(existingOrder.Status),
			AvailableBalances: s.formatBalances(balances.Available),
		}, nil
	}

	// 解析订单参数
	size, err := decimal.NewFromString(req.Size)
	if err != nil {
		return nil, fmt.Errorf("无效的订单数量: %w", err)
	}

	triggerPrice, err := decimal.NewFromString(req.TriggerPrice)
	if err != nil {
		return nil, fmt.Errorf("无效的触发价格: %w", err)
	}

	// 确定需要锁定的资产和数量
	var lockAsset string
	var lockAmount decimal.Decimal

	if req.Side == "buy" {
		// 买单锁定报价资产（如 USDC）
		lockAsset = req.Quote
		lockAmount = size.Mul(triggerPrice) // 数量 * 价格
	} else {
		// 卖单锁定基础资产（如 SOL）
		lockAsset = req.Base
		lockAmount = size
	}

	// 检查余额是否充足
	sufficient, err := s.ledgerService.CheckSufficientBalance(ctx, userID, lockAsset, lockAmount)
	if err != nil {
		return nil, fmt.Errorf("检查余额失败: %w", err)
	}
	if !sufficient {
		return nil, fmt.Errorf("余额不足: 需要 %s %s", lockAmount.String(), lockAsset)
	}

	// 创建订单
	orderID := uuid.New()
	now := time.Now()

	order := &types.Order{
		ID:           orderID,
		UserID:       userID,
		Base:         req.Base,
		Quote:        req.Quote,
		Size:         size,
		Side:         types.OrderSide(req.Side),
		TriggerPrice: triggerPrice,
		TriggerOp:    types.TriggerOp(req.TriggerOp),
		Status:       types.OrderStatusOpen,
		CreatedAt:    now,
		UpdatedAt:    now,
		Version:      0,
	}

	// 锁定资金并创建订单（原子操作）
	balanceSnapshot, err := s.ledgerService.LockFunds(ctx, userID, lockAsset, lockAmount, orderID)
	if err != nil {
		return nil, fmt.Errorf("锁定资金失败: %w", err)
	}

	// 创建订单记录
	if err := s.orderRepo.Create(ctx, order); err != nil {
		// 如果创建订单失败，需要释放已锁定的资金
		_, releaseErr := s.ledgerService.ReleaseFunds(ctx, userID, lockAsset, lockAmount, orderID)
		if releaseErr != nil {
			// 记录释放资金失败的错误，但仍然返回原始错误
			fmt.Printf("释放资金失败: %v\n", releaseErr)
		}
		return nil, fmt.Errorf("创建订单失败: %w", err)
	}

	return &types.CreateOrderResponse{
		OrderID:           orderID.String(),
		Status:            string(types.OrderStatusOpen),
		AvailableBalances: s.formatBalances(balanceSnapshot.Available),
	}, nil
}

// CancelOrder 取消订单
func (s *service) CancelOrder(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) (*types.CancelOrderResponse, error) {
	// 获取订单
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}

	// 检查订单所有权
	if order.UserID != userID {
		return nil, fmt.Errorf("无权限取消此订单")
	}

	// 检查订单状态
	if order.Status != types.OrderStatusOpen && order.Status != types.OrderStatusTriggered {
		return nil, fmt.Errorf("订单状态不允许取消: %s", order.Status)
	}

	// 计算需要释放的资金
	var releaseAsset string
	var releaseAmount decimal.Decimal

	if order.Side == types.OrderSideBuy {
		releaseAsset = order.Quote
		releaseAmount = order.Size.Mul(order.TriggerPrice)
	} else {
		releaseAsset = order.Base
		releaseAmount = order.Size
	}

	// 释放资金并更新订单状态
	balanceSnapshot, err := s.ledgerService.ReleaseFunds(ctx, userID, releaseAsset, releaseAmount, orderID)
	if err != nil {
		return nil, fmt.Errorf("释放资金失败: %w", err)
	}

	// 更新订单状态为已取消
	if err := s.orderRepo.UpdateStatus(ctx, orderID, types.OrderStatusCanceled, order.Version); err != nil {
		return nil, fmt.Errorf("更新订单状态失败: %w", err)
	}

	return &types.CancelOrderResponse{
		OrderID:           orderID.String(),
		Status:            string(types.OrderStatusCanceled),
		AvailableBalances: s.formatBalances(balanceSnapshot.Available),
		LockedBalances:    s.formatBalances(balanceSnapshot.Locked),
	}, nil
}

// GetOrders 查询订单
func (s *service) GetOrders(ctx context.Context, userID uuid.UUID, status string, limit int) (*types.OrderQueryResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50 // 默认限制
	}

	orders, err := s.orderRepo.GetByUserID(ctx, userID, status, limit)
	if err != nil {
		return nil, fmt.Errorf("查询订单失败: %w", err)
	}

	var orderInfos []types.OrderInfo
	for _, order := range orders {
		orderInfo := types.OrderInfo{
			OrderID:          order.ID.String(),
			BaseAsset:        order.Base,
			QuoteAsset:       order.Quote,
			Size:             order.Size.String(),
			Side:             string(order.Side),
			TriggerPrice:     order.TriggerPrice.String(),
			TriggerCondition: string(order.TriggerOp),
			Status:           string(order.Status),
			CreatedAt:        order.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        order.UpdatedAt.Format(time.RFC3339),
			ErrorMessage:     order.ErrorMessage,
		}
		if order.TriggeredAt != nil {
			triggeredAtStr := order.TriggeredAt.Format(time.RFC3339)
			orderInfo.TriggeredAt = &triggeredAtStr
		}
		orderInfos = append(orderInfos, orderInfo)
	}

	return &types.OrderQueryResponse{
		Orders: orderInfos,
	}, nil
}

// GetOrderByID 根据 ID 获取订单
func (s *service) GetOrderByID(ctx context.Context, orderID uuid.UUID) (*types.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}

// ValidateOrderRequest 验证订单请求
func (s *service) ValidateOrderRequest(req *types.CreateOrderRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("用户 ID 不能为空")
	}

	if req.Base == "" || req.Quote == "" {
		return fmt.Errorf("交易对不能为空")
	}

	if req.Base == req.Quote {
		return fmt.Errorf("基础资产和报价资产不能相同")
	}

	if req.Side != "buy" && req.Side != "sell" {
		return fmt.Errorf("无效的订单方向: %s", req.Side)
	}

	if req.TriggerOp != "gte" && req.TriggerOp != "lte" {
		return fmt.Errorf("无效的触发操作: %s", req.TriggerOp)
	}

	// 验证数量
	size, err := decimal.NewFromString(req.Size)
	if err != nil {
		return fmt.Errorf("无效的订单数量: %s", req.Size)
	}
	if size.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("订单数量必须大于 0")
	}

	// 验证触发价格
	triggerPrice, err := decimal.NewFromString(req.TriggerPrice)
	if err != nil {
		return fmt.Errorf("无效的触发价格: %s", req.TriggerPrice)
	}
	if triggerPrice.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("触发价格必须大于 0")
	}

	// 验证滑点
	slippage, err := decimal.NewFromString(req.MaxSlippagePct)
	if err != nil {
		return fmt.Errorf("无效的滑点设置: %s", req.MaxSlippagePct)
	}
	if slippage.LessThan(decimal.Zero) || slippage.GreaterThan(decimal.NewFromFloat(50)) {
		return fmt.Errorf("滑点必须在 0-50%% 之间")
	}

	if req.IdempotencyKey == "" {
		return fmt.Errorf("幂等性键不能为空")
	}

	return nil
}

// formatBalances 格式化余额为字符串映射
func (s *service) formatBalances(balances map[string]decimal.Decimal) map[string]string {
	result := make(map[string]string)
	for asset, balance := range balances {
		result[asset] = balance.String()
	}
	return result
}
