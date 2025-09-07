package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisQueue Redis 队列客户端
type RedisQueue struct {
	client *redis.Client
}

// NewRedisClient 创建 Redis 队列客户端
func NewRedisClient(redisURL string) (*RedisQueue, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("解析 Redis URL 失败: %w", err)
	}

	client := redis.NewClient(opts)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis 连接测试失败: %w", err)
	}

	log.Info().Msg("Redis 连接成功")
	return &RedisQueue{client: client}, nil
}

// Close 关闭 Redis 连接
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

// Publish 发布消息到 Stream
func (q *RedisQueue) Publish(ctx context.Context, stream string, data interface{}) error {
	// 将数据序列化为 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 发布到 Redis Stream
	args := &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"data":      string(jsonData),
			"timestamp": time.Now().UnixMilli(),
		},
	}

	_, err = q.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("发布消息到 Stream %s 失败: %w", stream, err)
	}

	log.Debug().
		Str("stream", stream).
		Str("data", string(jsonData)).
		Msg("消息发布成功")

	return nil
}

// CreateConsumerGroup 创建消费组
func (q *RedisQueue) CreateConsumerGroup(ctx context.Context, stream, group string) error {
	// 尝试创建消费组，如果stream不存在则先创建，然后再创建消费组
	err := q.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("创建消费组失败: %w", err)
	}

	log.Info().
		Str("stream", stream).
		Str("group", group).
		Msg("消费组创建成功")

	return nil
}

// Consume 消费消息
func (q *RedisQueue) Consume(ctx context.Context, stream, group, consumer string, count int64, block time.Duration) ([]Message, error) {
	args := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    block,
	}

	result, err := q.client.XReadGroup(ctx, args).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 没有消息
		}
		return nil, fmt.Errorf("消费消息失败: %w", err)
	}

	var messages []Message
	for _, stream := range result {
		for _, msg := range stream.Messages {
			message := Message{
				ID:     msg.ID,
				Stream: stream.Stream,
				Data:   msg.Values,
			}
			messages = append(messages, message)
		}
	}

	return messages, nil
}

// Ack 确认消息处理完成
func (q *RedisQueue) Ack(ctx context.Context, stream, group string, ids ...string) error {
	err := q.client.XAck(ctx, stream, group, ids...).Err()
	if err != nil {
		return fmt.Errorf("确认消息失败: %w", err)
	}

	log.Debug().
		Str("stream", stream).
		Str("group", group).
		Strs("ids", ids).
		Msg("消息确认成功")

	return nil
}

// GetPendingMessages 获取待处理消息
func (q *RedisQueue) GetPendingMessages(ctx context.Context, stream, group string) (*redis.XPending, error) {
	result, err := q.client.XPending(ctx, stream, group).Result()
	if err != nil {
		return nil, fmt.Errorf("获取待处理消息失败: %w", err)
	}

	return result, nil
}

// ClaimMessages 认领超时消息
func (q *RedisQueue) ClaimMessages(ctx context.Context, stream, group, consumer string, minIdleTime time.Duration, ids ...string) ([]redis.XMessage, error) {
	args := &redis.XClaimArgs{
		Stream:   stream,
		Group:    group,
		Consumer: consumer,
		MinIdle:  minIdleTime,
		Messages: ids,
	}

	result, err := q.client.XClaim(ctx, args).Result()
	if err != nil {
		return nil, fmt.Errorf("认领消息失败: %w", err)
	}

	return result, nil
}

// GetStreamInfo 获取 Stream 信息
func (q *RedisQueue) GetStreamInfo(ctx context.Context, stream string) (*redis.XInfoStream, error) {
	result, err := q.client.XInfoStream(ctx, stream).Result()
	if err != nil {
		return nil, fmt.Errorf("获取 Stream 信息失败: %w", err)
	}

	return result, nil
}

// TrimStream 修剪 Stream（保留最近的消息）
func (q *RedisQueue) TrimStream(ctx context.Context, stream string, maxLen int64) error {
	err := q.client.XTrimMaxLen(ctx, stream, maxLen).Err()
	if err != nil {
		return fmt.Errorf("修剪 Stream 失败: %w", err)
	}

	return nil
}

// Message 消息结构
type Message struct {
	ID     string
	Stream string
	Data   map[string]interface{}
}

// GetStringData 获取字符串数据
func (m *Message) GetStringData(key string) (string, bool) {
	if val, ok := m.Data[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

// GetJSONData 获取 JSON 数据并反序列化
func (m *Message) GetJSONData(key string, target interface{}) error {
	if val, ok := m.GetStringData(key); ok {
		return json.Unmarshal([]byte(val), target)
	}
	return fmt.Errorf("键 %s 不存在或不是字符串类型", key)
}

// Health 检查 Redis 健康状态
func (q *RedisQueue) Health(ctx context.Context) error {
	return q.client.Ping(ctx).Err()
}

// PublishExecuteOrder 发布执行订单任务
func (q *RedisQueue) PublishExecuteOrder(ctx context.Context, task ExecuteOrderTask) error {
	return q.Publish(ctx, StreamOrderExecute, task)
}

// ExecuteOrderTask 执行订单任务
type ExecuteOrderTask struct {
	OrderID             string `json:"order_id"`
	Timestamp           int64  `json:"timestamp"`
	Price               string `json:"price"`
	Side                string `json:"side"`
	MaxSlippagePct      string `json:"max_slippage_pct"`
	PriorityFeeLamports int64  `json:"priority_fee_lamports"`
	IdempotencyKey      string `json:"idempotency_key"`
}
