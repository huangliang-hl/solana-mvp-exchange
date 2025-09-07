package testutil

import (
	"context"
	"fmt"
	"time"

	"solana-limit-order-backend/internal/executor"
	"solana-limit-order-backend/internal/ledger"
	"solana-limit-order-backend/internal/queue"
	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// MockExecutorService 模拟执行器服务，用于测试
type MockExecutorService struct {
	queueClient    *queue.RedisQueue
	orderRepo      repository.OrderRepository
	ledgerService  ledger.Service
	mockJupiter    *MockJupiterClient
	mockSolana     *MockSolanaClient
	realExecutor   *executor.Service
	useRealClients bool
}

// NewMockExecutorService 创建模拟执行器服务
func NewMockExecutorService(
	queueClient *queue.RedisQueue,
	orderRepo repository.OrderRepository,
	ledgerService ledger.Service,
	mockJupiter *MockJupiterClient,
	mockSolana *MockSolanaClient,
) *MockExecutorService {
	return &MockExecutorService{
		queueClient:    queueClient,
		orderRepo:      orderRepo,
		ledgerService:  ledgerService,
		mockJupiter:    mockJupiter,
		mockSolana:     mockSolana,
		useRealClients: false,
	}
}

// StartWorker 启动工作器（模拟实现）
func (s *MockExecutorService) StartWorker(ctx context.Context, workerID int) error {
	consumerGroup := "executor"
	consumerName := fmt.Sprintf("worker_%d", workerID)

	// 确保消费组存在
	if err := s.queueClient.CreateConsumerGroup(ctx, queue.StreamOrderExecute, consumerGroup); err != nil {
		return fmt.Errorf("创建消费组失败: %w", err)
	}

	log.Info().
		Int("worker_id", workerID).
		Str("consumer_group", consumerGroup).
		Str("consumer_name", consumerName).
		Msg("模拟执行器工作器启动")

	for {
		select {
		case <-ctx.Done():
			log.Info().Int("worker_id", workerID).Msg("模拟执行器工作器停止")
			return nil

		default:
			// 消费消息
			messages, err := s.queueClient.Consume(ctx, queue.StreamOrderExecute, consumerGroup, consumerName, 1, 5*time.Second)
			if err != nil {
				log.Error().
					Err(err).
					Int("worker_id", workerID).
					Msg("消费消息失败")
				continue
			}

			// 处理消息
			for _, msg := range messages {
				if err := s.processMessage(ctx, &msg); err != nil {
					log.Error().
						Err(err).
						Str("message_id", msg.ID).
						Int("worker_id", workerID).
						Msg("处理消息失败")
				}

				// 确认消息
				if err := s.queueClient.Ack(ctx, queue.StreamOrderExecute, consumerGroup, msg.ID); err != nil {
					log.Error().
						Err(err).
						Str("message_id", msg.ID).
						Msg("确认消息失败")
				}
			}
		}
	}
}

// RecoverPendingOrders 恢复待处理订单
func (s *MockExecutorService) RecoverPendingOrders(ctx context.Context) error {
	log.Info().Msg("开始恢复待处理订单（模拟）")
	return nil
}

// ExecuteOrder 执行订单（模拟实现）
func (s *MockExecutorService) ExecuteOrder(ctx context.Context, msg *types.ExecuteOrderMessage) error {
	orderID, err := uuid.Parse(msg.OrderID)
	if err != nil {
		return fmt.Errorf("无效的订单ID: %w", err)
	}

	// 获取订单信息
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("获取订单失败: %w", err)
	}

	// 检查订单状态
	if order.Status != types.OrderStatusTriggered {
		log.Warn().
			Str("order_id", orderID.String()).
			Str("status", string(order.Status)).
			Msg("订单状态不是 triggered，跳过执行")
		return nil
	}

	// 更新订单状态为 submitting
	if err := s.orderRepo.UpdateStatus(ctx, orderID, types.OrderStatusSubmitting, order.Version); err != nil {
		return fmt.Errorf("更新订单状态为 submitting 失败: %w", err)
	}

	// 更新order的version，因为状态已经改变
	order.Version++
	order.Status = types.OrderStatusSubmitting

	// 模拟 Jupiter 报价
	quoteReq := &JupiterQuoteRequest{
		InputMint:   "mock_input_mint",
		OutputMint:  "mock_output_mint",
		Amount:      order.Size.String(),
		SlippageBps: 50,
	}

	quote, err := s.mockJupiter.GetQuote(ctx, quoteReq)
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("获取 Jupiter 报价失败: %w", err))
	}

	// 模拟交换交易
	swapReq := &JupiterSwapRequest{
		QuoteResponse: quote,
		UserPublicKey: "mock_user_public_key",
	}

	swapResp, err := s.mockJupiter.GetSwapTransaction(ctx, swapReq)
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("获取交换交易失败: %w", err))
	}

	// 模拟提交交易
	txSig, err := s.mockSolana.SendTransaction(swapResp.SwapTransaction)
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("提交交易失败: %w", err))
	}

	// 模拟交易确认
	_, err = s.mockSolana.ConfirmTransaction(txSig)
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("等待交易确认失败: %w", err))
	}

	// 处理成功的交易
	return s.handleExecutionSuccess(ctx, order, quote, txSig)
}

// processMessage 处理消息
func (s *MockExecutorService) processMessage(ctx context.Context, msg *queue.Message) error {
	// 解析执行消息
	var executeMsg types.ExecuteOrderMessage
	if err := msg.GetJSONData("data", &executeMsg); err != nil {
		return fmt.Errorf("解析执行消息失败: %w", err)
	}

	log.Info().
		Str("order_id", executeMsg.OrderID).
		Str("user_id", executeMsg.UserID).
		Str("idempotency_key", executeMsg.IdempotencyKey).
		Msg("开始执行订单（模拟）")

	// 执行订单
	return s.ExecuteOrder(ctx, &executeMsg)
}

// handleExecutionError 处理执行错误
func (s *MockExecutorService) handleExecutionError(ctx context.Context, order *types.Order, execErr error) error {
	log.Error().
		Err(execErr).
		Str("order_id", order.ID.String()).
		Msg("订单执行失败（模拟）")

	// 更新订单状态为 failed
	if err := s.orderRepo.UpdateStatus(ctx, order.ID, types.OrderStatusFailed, order.Version); err != nil {
		log.Error().
			Err(err).
			Str("order_id", order.ID.String()).
			Msg("更新订单状态为 failed 失败")
	}

	// 释放锁定资金
	var releaseAsset string
	var releaseAmount decimal.Decimal

	if order.Side == types.OrderSideBuy {
		releaseAsset = order.Quote
		releaseAmount = order.Size.Mul(order.TriggerPrice)
	} else {
		releaseAsset = order.Base
		releaseAmount = order.Size
	}

	if _, err := s.ledgerService.ReleaseFunds(ctx, order.UserID, releaseAsset, releaseAmount, order.ID); err != nil {
		log.Error().
			Err(err).
			Str("order_id", order.ID.String()).
			Str("asset", releaseAsset).
			Str("amount", releaseAmount.String()).
			Msg("释放锁定资金失败")
	}

	return execErr
}

// handleExecutionSuccess 处理执行成功
func (s *MockExecutorService) handleExecutionSuccess(ctx context.Context, order *types.Order, quote *JupiterQuoteResponse, txSig string) error {
	log.Info().
		Str("order_id", order.ID.String()).
		Str("tx_sig", txSig).
		Str("input_amount", quote.InAmount).
		Str("output_amount", quote.OutAmount).
		Msg("订单执行成功（模拟）")

	// 更新订单状态为 filled
	now := time.Now()
	order.Status = types.OrderStatusFilled
	order.TxSig = &txSig
	order.SubmittedAt = &now

	if err := s.orderRepo.UpdateStatus(ctx, order.ID, types.OrderStatusFilled, order.Version); err != nil {
		return fmt.Errorf("更新订单状态为 filled 失败: %w", err)
	}

	// 处理资金变动（简化版）
	var sellAsset, buyAsset string
	var sellAmount, buyAmount decimal.Decimal

	if order.Side == types.OrderSideBuy {
		sellAsset = order.Quote
		buyAsset = order.Base
		sellAmount = order.Size.Mul(order.TriggerPrice)
		buyAmount = order.Size
	} else {
		sellAsset = order.Base
		buyAsset = order.Quote
		sellAmount = order.Size
		buyAmount = order.Size.Mul(order.TriggerPrice)
	}

	// 处理成交
	if _, err := s.ledgerService.ProcessFill(ctx, order.UserID, sellAsset, buyAsset, sellAmount, buyAmount, order.ID); err != nil {
		return fmt.Errorf("处理成交失败: %w", err)
	}

	return nil
}
