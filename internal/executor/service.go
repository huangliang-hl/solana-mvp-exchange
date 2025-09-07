package executor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"solana-limit-order-backend/internal/config"
	"solana-limit-order-backend/internal/ledger"
	"solana-limit-order-backend/internal/queue"
	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// Service 执行器服务接口
type Service interface {
	// StartWorker 启动工作器
	StartWorker(ctx context.Context, workerID int) error

	// RecoverPendingOrders 恢复待处理订单
	RecoverPendingOrders(ctx context.Context) error

	// ExecuteOrder 执行订单
	ExecuteOrder(ctx context.Context, msg *types.ExecuteOrderMessage) error
}

// service 执行器服务实现
type service struct {
	cfg           *config.Config
	queueClient   *queue.RedisQueue
	orderRepo     repository.OrderRepository
	ledgerService ledger.Service
	jupiterClient JupiterClientInterface
	solanaClient  *SolanaClient
}

// NewService 创建执行器服务
func NewService(
	cfg *config.Config,
	queueClient *queue.RedisQueue,
	orderRepo repository.OrderRepository,
	ledgerService ledger.Service,
	jupiterClient JupiterClientInterface,
	solanaClient *SolanaClient,
) Service {
	return &service{
		cfg:           cfg,
		queueClient:   queueClient,
		orderRepo:     orderRepo,
		ledgerService: ledgerService,
		jupiterClient: jupiterClient,
		solanaClient:  solanaClient,
	}
}

// StartWorker 启动工作器
func (s *service) StartWorker(ctx context.Context, workerID int) error {
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
		Msg("执行器工作器启动")

	for {
		select {
		case <-ctx.Done():
			log.Info().Int("worker_id", workerID).Msg("执行器工作器停止")
			return nil

		default:
			// 消费消息
			timeout := time.Duration(s.cfg.WorkerConsumerTimeout) * time.Second
			messages, err := s.queueClient.Consume(ctx, queue.StreamOrderExecute, consumerGroup, consumerName, 1, timeout)
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
func (s *service) RecoverPendingOrders(ctx context.Context) error {
	log.Info().Msg("开始恢复待处理订单")

	// 查找所有 triggered 和 submitting 状态的订单
	// 这需要在 OrderRepository 中实现相应的方法
	// 暂时返回 nil，实际实现需要根据具体需求完善

	log.Info().Msg("待处理订单恢复完成")
	return nil
}

// processMessage 处理消息
func (s *service) processMessage(ctx context.Context, msg *queue.Message) error {
	// 解析执行消息
	var executeMsg types.ExecuteOrderMessage
	if err := msg.GetJSONData("data", &executeMsg); err != nil {
		return fmt.Errorf("解析执行消息失败: %w", err)
	}

	log.Info().
		Str("order_id", executeMsg.OrderID).
		Str("user_id", executeMsg.UserID).
		Str("idempotency_key", executeMsg.IdempotencyKey).
		Msg("开始执行订单")

	// 执行订单
	return s.ExecuteOrder(ctx, &executeMsg)
}

// ExecuteOrder 执行订单
func (s *service) ExecuteOrder(ctx context.Context, msg *types.ExecuteOrderMessage) error {
	orderID, err := uuid.Parse(msg.OrderID)
	if err != nil {
		return fmt.Errorf("无效的订单ID: %w", err)
	}

	_, err = uuid.Parse(msg.UserID)
	if err != nil {
		return fmt.Errorf("无效的用户ID: %w", err)
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

	// 获取代币地址
	inputMint, err := GetTokenMint(order.Base, true) // 假设使用 devnet
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("获取输入代币地址失败: %w", err))
	}

	outputMint, err := GetTokenMint(order.Quote, true)
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("获取输出代币地址失败: %w", err))
	}

	// 如果是卖单，交换输入输出
	if order.Side == types.OrderSideSell {
		inputMint, outputMint = outputMint, inputMint
	}

	// 计算滑点
	maxSlippagePct, err := decimal.NewFromString(msg.MaxSlippagePct)
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("无效的滑点参数: %w", err))
	}

	slippageBps := CalculateSlippageBps(maxSlippagePct)

	// 计算实际的输入资产和金额
	var inputAsset string
	var inputAmount decimal.Decimal

	if order.Side == types.OrderSideBuy {
		// 买单：输入 USDC，输出 SOL
		inputAsset = order.Quote
		inputAmount = order.Size.Mul(order.TriggerPrice) // 金额 = 数量 * 价格
	} else {
		// 卖单：输入 SOL，输出 USDC
		inputAsset = order.Base
		inputAmount = order.Size
	}

	// 转换为正确的单位（最小单位）
	if inputAsset == "SOL" {
		// SOL 需要转换为 lamports (1 SOL = 1,000,000,000 lamports)
		inputAmount = inputAmount.Mul(decimal.NewFromInt(1000000000))
	} else if inputAsset == "USDC" {
		// USDC 需要转换为微单位 (1 USDC = 1,000,000 micro units)
		inputAmount = inputAmount.Mul(decimal.NewFromInt(1000000))
	}

	// 确保金额是整数
	amountInt := inputAmount.IntPart()

	// 获取 Jupiter 报价
	quoteReq := &QuoteRequest{
		InputMint:   inputMint,
		OutputMint:  outputMint,
		Amount:      fmt.Sprintf("%d", amountInt),
		SlippageBps: slippageBps,
	}

	quote, err := s.jupiterClient.GetQuote(ctx, quoteReq)
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("获取 Jupiter 报价失败: %w", err))
	}

	// 验证报价
	if err := ValidateQuote(quote, maxSlippagePct); err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("报价验证失败: %w", err))
	}

	// 获取用户公钥
	userPubkey, err := s.solanaClient.GetUserPublicKey()
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("获取用户公钥失败: %w", err))
	}

	// 构建交换请求
	swapReq := &SwapRequest{
		QuoteResponse:                 *quote,
		UserPublicKey:                 userPubkey,
		WrapAndUnwrapSol:              true,
		ComputeUnitPriceMicroLamports: int(msg.PriorityFeeLamports),
	}

	// 获取交换交易
	swapResp, err := s.jupiterClient.GetSwapTransaction(ctx, swapReq)
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("获取交换交易失败: %w", err))
	}

	// 提交交易到 Solana
	txSig, err := s.solanaClient.SubmitTransaction(ctx, swapResp.SwapTransaction, msg.PriorityFeeLamports)
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("提交交易失败: %w", err))
	}

	// 等待交易确认
	timeout := time.Duration(s.cfg.OrderExecutionTimeout) * time.Second
	confirmed, err := s.solanaClient.WaitForConfirmation(ctx, txSig, timeout)
	if err != nil {
		return s.handleExecutionError(ctx, order, fmt.Errorf("等待交易确认失败: %w", err))
	}

	if !confirmed {
		return s.handleExecutionError(ctx, order, fmt.Errorf("交易确认超时"))
	}

	// 处理成功的交易
	return s.handleExecutionSuccess(ctx, order, quote, txSig)
}

// handleExecutionError 处理执行错误
func (s *service) handleExecutionError(ctx context.Context, order *types.Order, execErr error) error {
	log.Error().
		Err(execErr).
		Str("order_id", order.ID.String()).
		Msg("订单执行失败")

	// 更新订单状态为 failed 并记录错误信息
	if err := s.orderRepo.UpdateStatusWithError(ctx, order.ID, types.OrderStatusFailed, order.Version+1, execErr.Error()); err != nil {
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
func (s *service) handleExecutionSuccess(ctx context.Context, order *types.Order, quote *QuoteResponse, txSig string) error {
	log.Info().
		Str("order_id", order.ID.String()).
		Str("tx_sig", txSig).
		Str("input_amount", quote.InAmount).
		Str("output_amount", quote.OutAmount).
		Msg("订单执行成功")

	// 更新订单状态为 filled
	now := time.Now()
	order.Status = types.OrderStatusFilled
	order.TxSig = &txSig
	order.SubmittedAt = &now

	if err := s.orderRepo.UpdateStatus(ctx, order.ID, types.OrderStatusFilled, order.Version+1); err != nil {
		return fmt.Errorf("更新订单状态为 filled 失败: %w", err)
	}

	// 处理资金变动
	var sellAsset, buyAsset string
	var sellAmount, buyAmount decimal.Decimal

	if order.Side == types.OrderSideBuy {
		sellAsset = order.Quote
		buyAsset = order.Base
		sellAmount, _ = decimal.NewFromString(quote.InAmount)
		buyAmount, _ = decimal.NewFromString(quote.OutAmount)
	} else {
		sellAsset = order.Base
		buyAsset = order.Quote
		sellAmount, _ = decimal.NewFromString(quote.InAmount)
		buyAmount, _ = decimal.NewFromString(quote.OutAmount)
	}

	// 将最小单位转换回常规单位
	if sellAsset == "SOL" {
		sellAmount = sellAmount.Div(decimal.NewFromInt(1000000000)) // lamports -> SOL
	} else if sellAsset == "USDC" {
		sellAmount = sellAmount.Div(decimal.NewFromInt(1000000)) // micro USDC -> USDC
	}

	if buyAsset == "SOL" {
		buyAmount = buyAmount.Div(decimal.NewFromInt(1000000000)) // lamports -> SOL
	} else if buyAsset == "USDC" {
		buyAmount = buyAmount.Div(decimal.NewFromInt(1000000)) // micro USDC -> USDC
	}

	// 处理成交（使用新的锁定资金释放方法）
	if _, err := s.ledgerService.ProcessFillWithLockRelease(ctx, order.UserID, order, sellAsset, buyAsset, sellAmount, buyAmount, order.ID); err != nil {
		// 如果是并发冲突，记录详细日志
		if isConcurrencyError(err) {
			log.Warn().
				Err(err).
				Str("order_id", order.ID.String()).
				Str("user_id", order.UserID.String()).
				Msg("处理成交时发生并发冲突，ledger service 将自动重试")
		}
		return fmt.Errorf("处理成交失败: %w", err)
	}

	return nil
}

// isConcurrencyError 检查是否为PostgreSQL并发冲突错误
func isConcurrencyError(err error) bool {
	// 递归解包错误，查找底层的 pgconn.PgError
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 检查PostgreSQL并发相关的SQLSTATE
		switch pgErr.Code {
		case "40001": // serialization_failure
			return true
		case "40P01": // deadlock_detected
			return true
		case "23505": // unique_violation（在某些场景下也可能表示并发问题）
			return false // 这个通常不算并发错误，而是业务逻辑错误
		}
	}
	return false
}
