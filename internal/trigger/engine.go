package trigger

import (
	"context"
	"fmt"
	"time"

	"solana-limit-order-backend/internal/queue"
	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/types"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// Engine 触发引擎
type Engine struct {
	// 依赖服务
	orderRepo repository.OrderRepository
	queue     *queue.RedisQueue

	// 运行状态
	ctx    context.Context
	cancel context.CancelFunc

	// 配置
	batchSize int
	symbols   []string
}

// Config 引擎配置
type Config struct {
	BatchSize int      `json:"batch_size"`
	Symbols   []string `json:"symbols"`
}

// NewEngine 创建触发引擎
func NewEngine(queueClient *queue.RedisQueue, orderRepo repository.OrderRepository) *Engine {
	return &Engine{
		orderRepo: orderRepo,
		queue:     queueClient,
		batchSize: 100,
		symbols:   []string{"SOL/USDC"},
	}
}

// Start 启动触发引擎
func (e *Engine) Start(ctx context.Context) error {
	e.ctx = ctx
	var cancel context.CancelFunc
	e.ctx, cancel = context.WithCancel(ctx)
	e.cancel = cancel

	log.Info().
		Int("batch_size", e.batchSize).
		Strs("symbols", e.symbols).
		Msg("触发引擎启动")

	// 创建消费组
	if err := e.queue.CreateConsumerGroup(ctx, "price.ticks", "trigger_engine"); err != nil {
		return fmt.Errorf("创建价格tick消费组失败: %w", err)
	}

	go e.run()
	return nil
}

// Stop 停止触发引擎
func (e *Engine) Stop() {
	log.Info().Msg("触发引擎停止")
	e.cancel()
}

// run 运行主循环
func (e *Engine) run() {
	for {
		select {
		case <-e.ctx.Done():
			return

		default:
			// 从Redis队列消费价格tick
			messages, err := e.queue.Consume(e.ctx, "price.ticks", "trigger_engine", "consumer_1", 1, 5*time.Second)
			if err != nil {
				log.Error().Err(err).Msg("消费价格tick失败")
				time.Sleep(time.Second)
				continue
			}

			for _, msg := range messages {
				var priceTick types.PriceTick
				if err := msg.GetJSONData("data", &priceTick); err != nil {
					log.Error().Err(err).Msg("解析价格tick失败")
					continue
				}

				e.processPriceTick(&priceTick)

				// 确认消息
				if err := e.queue.Ack(e.ctx, "price.ticks", "trigger_engine", msg.ID); err != nil {
					log.Error().Err(err).Msg("确认价格tick消息失败")
				}
			}
		}
	}
}

// processPriceTick 处理价格 tick
func (e *Engine) processPriceTick(tick *types.PriceTick) {
	// 解析交易对
	base, quote, err := parseSymbol(tick.Symbol)
	if err != nil {
		log.Error().
			Err(err).
			Str("symbol", tick.Symbol).
			Msg("解析交易对失败")
		return
	}

	// 获取可触发的订单
	orders, err := e.orderRepo.GetOpenOrdersForTrigger(e.ctx, base, quote, e.batchSize)
	if err != nil {
		log.Error().
			Err(err).
			Str("symbol", tick.Symbol).
			Msg("获取可触发订单失败")
		return
	}

	if len(orders) == 0 {
		return // 没有订单需要检查
	}

	log.Debug().
		Str("symbol", tick.Symbol).
		Int("order_count", len(orders)).
		Str("mid_price", tick.MidPrice.String()).
		Msg("检查订单触发条件")

	// 检查每个订单的触发条件
	triggeredCount := 0
	for _, order := range orders {
		if e.shouldTriggerOrder(order, tick) {
			if err := e.triggerOrder(order, tick); err != nil {
				log.Error().
					Err(err).
					Str("order_id", order.ID.String()).
					Msg("触发订单失败")
			} else {
				triggeredCount++
			}
		}
	}

	if triggeredCount > 0 {
		log.Info().
			Str("symbol", tick.Symbol).
			Int("triggered_count", triggeredCount).
			Int("total_checked", len(orders)).
			Str("mid_price", tick.MidPrice.String()).
			Msg("订单触发完成")
	}
}

// shouldTriggerOrder 检查订单是否应该触发
func (e *Engine) shouldTriggerOrder(order *types.Order, tick *types.PriceTick) bool {
	// 使用中间价进行触发判断
	currentPrice := tick.MidPrice
	triggerPrice := order.TriggerPrice

	switch order.TriggerOp {
	case types.TriggerOpGTE:
		// 大于等于触发价
		return currentPrice.GreaterThanOrEqual(triggerPrice)
	case types.TriggerOpLTE:
		// 小于等于触发价
		return currentPrice.LessThanOrEqual(triggerPrice)
	default:
		log.Error().
			Str("order_id", order.ID.String()).
			Str("trigger_op", string(order.TriggerOp)).
			Msg("未知的触发操作")
		return false
	}
}

// triggerOrder 触发订单
func (e *Engine) triggerOrder(order *types.Order, tick *types.PriceTick) error {
	// 使用乐观锁更新订单状态
	err := e.orderRepo.TriggerOrder(e.ctx, order.ID, order.Version)
	if err != nil {
		// 可能是并发冲突，记录警告但不返回错误
		log.Warn().
			Err(err).
			Str("order_id", order.ID.String()).
			Int("version", order.Version).
			Msg("触发订单时发生冲突，可能已被其他进程处理")
		return nil
	}

	// 创建执行消息
	executeMsg := &types.ExecuteOrderMessage{
		OrderID:             order.ID.String(),
		UserID:              order.UserID.String(),
		MaxSlippagePct:      "5.0", // 默认滑点5%，实际应该从订单中获取
		PriorityFeeLamports: 5000,  // 默认优先费，实际应该从订单中获取
		IdempotencyKey:      generateIdempotencyKey(order.ID.String(), tick.SequenceID),
		Timestamp:           tick.Timestamp,
	}

	// 发送到执行队列
	err = e.queue.Publish(e.ctx, queue.StreamOrderExecute, executeMsg)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", order.ID.String()).
			Msg("发送执行消息到队列失败")
		return fmt.Errorf("发送执行消息失败: %w", err)
	}

	log.Info().
		Str("order_id", order.ID.String()).
		Str("user_id", order.UserID.String()).
		Str("symbol", tick.Symbol).
		Str("trigger_price", order.TriggerPrice.String()).
		Str("current_price", tick.MidPrice.String()).
		Str("trigger_op", string(order.TriggerOp)).
		Int64("sequence_id", tick.SequenceID).
		Msg("订单触发成功")

	return nil
}

// parseSymbol 解析交易对符号
func parseSymbol(symbol string) (base, quote string, err error) {
	// 简单的解析逻辑，假设格式为 "BASE/QUOTE"
	if len(symbol) < 3 {
		return "", "", fmt.Errorf("无效的交易对符号: %s", symbol)
	}

	// 查找分隔符
	for i, char := range symbol {
		if char == '/' {
			if i == 0 || i == len(symbol)-1 {
				return "", "", fmt.Errorf("无效的交易对符号: %s", symbol)
			}
			return symbol[:i], symbol[i+1:], nil
		}
	}

	return "", "", fmt.Errorf("交易对符号缺少分隔符: %s", symbol)
}

// generateIdempotencyKey 生成幂等性键
func generateIdempotencyKey(orderID string, sequenceID int64) string {
	return fmt.Sprintf("trigger-%s-%d", orderID, sequenceID)
}

// GetStats 获取引擎统计信息
func (e *Engine) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"batch_size": e.batchSize,
		"symbols":    e.symbols,
		"running":    e.ctx.Err() == nil,
	}
}

// Matcher 订单匹配器（用于更复杂的触发逻辑）
type Matcher struct {
	engine *Engine
}

// NewMatcher 创建匹配器
func NewMatcher(engine *Engine) *Matcher {
	return &Matcher{engine: engine}
}

// MatchOrders 匹配订单（预留接口，用于未来扩展）
func (m *Matcher) MatchOrders(orders []*types.Order, tick *types.PriceTick) []*types.Order {
	var matchedOrders []*types.Order

	for _, order := range orders {
		if m.engine.shouldTriggerOrder(order, tick) {
			matchedOrders = append(matchedOrders, order)
		}
	}

	return matchedOrders
}

// ValidateTriggerCondition 验证触发条件
func ValidateTriggerCondition(order *types.Order, currentPrice decimal.Decimal) error {
	if order.TriggerPrice.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("触发价格必须大于 0")
	}

	if order.TriggerOp != types.TriggerOpGTE && order.TriggerOp != types.TriggerOpLTE {
		return fmt.Errorf("无效的触发操作: %s", order.TriggerOp)
	}

	// 检查价格是否合理（不能偏离当前价格太远）
	maxDeviation := currentPrice.Mul(decimal.NewFromFloat(0.5)) // 50% 偏差
	if order.TriggerPrice.Sub(currentPrice).Abs().GreaterThan(maxDeviation) {
		return fmt.Errorf("触发价格偏离当前价格过远")
	}

	return nil
}
