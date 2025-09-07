package queue

import "context"

// 定义队列主题常量
const (
	// StreamOrderTriggered 订单触发流
	StreamOrderTriggered = "order.triggered"

	// StreamOrderExecute 订单执行流
	StreamOrderExecute = "order.execute"

	// StreamOrderRetry 订单重试流
	StreamOrderRetry = "order.retry"

	// StreamOrderDeadletter 死信队列
	StreamOrderDeadletter = "order.deadletter"

	// StreamPriceTick 价格 tick 流（可选）
	StreamPriceTick = "price.ticks"
)

// 定义消费组常量
const (
	// GroupExecutor 执行器消费组
	GroupExecutor = "executor"

	// GroupTrigger 触发器消费组
	GroupTrigger = "trigger"

	// GroupRetry 重试处理消费组
	GroupRetry = "retry"
)

// QueueManager 队列管理器
type QueueManager struct {
	redis *RedisQueue
}

// NewQueueManager 创建队列管理器
func NewQueueManager(redis *RedisQueue) *QueueManager {
	return &QueueManager{redis: redis}
}

// InitializeStreams 初始化所有流和消费组
func (qm *QueueManager) InitializeStreams(ctx context.Context) error {

	groups := map[string][]string{
		StreamOrderTriggered:  {GroupTrigger},
		StreamOrderExecute:    {GroupExecutor},
		StreamOrderRetry:      {GroupRetry},
		StreamOrderDeadletter: {GroupRetry},
		StreamPriceTick:       {GroupTrigger},
	}

	// 创建消费组
	for stream, groupList := range groups {
		for _, group := range groupList {
			if err := qm.redis.CreateConsumerGroup(ctx, stream, group); err != nil {
				return err
			}
		}
	}

	return nil
}
