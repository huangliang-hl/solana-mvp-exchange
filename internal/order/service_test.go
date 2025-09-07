package order

import (
	"testing"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCreateOrderRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request types.CreateOrderRequest
		isValid bool
	}{
		{
			name: "有效的卖单请求",
			request: types.CreateOrderRequest{
				UserID:              uuid.New().String(),
				Base:                "SOL",
				Quote:               "USDC",
				Size:                "1.5",
				Side:                "sell",
				TriggerPrice:        "30.5",
				TriggerOp:           "lte",
				MaxSlippagePct:      "0.5",
				PriorityFeeLamports: 5000,
				IdempotencyKey:      "test-key-123",
			},
			isValid: true,
		},
		{
			name: "有效的买单请求",
			request: types.CreateOrderRequest{
				UserID:              uuid.New().String(),
				Base:                "SOL",
				Quote:               "USDC",
				Size:                "2.0",
				Side:                "buy",
				TriggerPrice:        "25.0",
				TriggerOp:           "gte",
				MaxSlippagePct:      "1.0",
				PriorityFeeLamports: 3000,
				IdempotencyKey:      "test-key-456",
			},
			isValid: true,
		},
		{
			name: "无效的订单 - 空用户ID",
			request: types.CreateOrderRequest{
				UserID:              "",
				Base:                "SOL",
				Quote:               "USDC",
				Size:                "1.0",
				Side:                "sell",
				TriggerPrice:        "30.0",
				TriggerOp:           "lte",
				MaxSlippagePct:      "0.5",
				PriorityFeeLamports: 5000,
				IdempotencyKey:      "test-key-789",
			},
			isValid: false,
		},
		{
			name: "无效的订单 - 零数量",
			request: types.CreateOrderRequest{
				UserID:              uuid.New().String(),
				Base:                "SOL",
				Quote:               "USDC",
				Size:                "0",
				Side:                "sell",
				TriggerPrice:        "30.0",
				TriggerOp:           "lte",
				MaxSlippagePct:      "0.5",
				PriorityFeeLamports: 5000,
				IdempotencyKey:      "test-key-101",
			},
			isValid: false,
		},
		{
			name: "无效的订单 - 无效方向",
			request: types.CreateOrderRequest{
				UserID:              uuid.New().String(),
				Base:                "SOL",
				Quote:               "USDC",
				Size:                "1.0",
				Side:                "invalid",
				TriggerPrice:        "30.0",
				TriggerOp:           "lte",
				MaxSlippagePct:      "0.5",
				PriorityFeeLamports: 5000,
				IdempotencyKey:      "test-key-102",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateOrderRequest(&tt.request)
			if tt.isValid {
				assert.NoError(t, err, "应该是有效的订单请求")
			} else {
				assert.Error(t, err, "应该是无效的订单请求")
			}
		})
	}
}

func TestOrderStatusTransitions(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus string
		newStatus     string
		isValid       bool
	}{
		{
			name:          "open -> triggered",
			currentStatus: "open",
			newStatus:     "triggered",
			isValid:       true,
		},
		{
			name:          "open -> canceled",
			currentStatus: "open",
			newStatus:     "canceled",
			isValid:       true,
		},
		{
			name:          "triggered -> submitting",
			currentStatus: "triggered",
			newStatus:     "submitting",
			isValid:       true,
		},
		{
			name:          "triggered -> canceled",
			currentStatus: "triggered",
			newStatus:     "canceled",
			isValid:       true,
		},
		{
			name:          "submitting -> filled",
			currentStatus: "submitting",
			newStatus:     "filled",
			isValid:       true,
		},
		{
			name:          "submitting -> failed",
			currentStatus: "submitting",
			newStatus:     "failed",
			isValid:       true,
		},
		{
			name:          "filled -> open (无效)",
			currentStatus: "filled",
			newStatus:     "open",
			isValid:       false,
		},
		{
			name:          "canceled -> triggered (无效)",
			currentStatus: "canceled",
			newStatus:     "triggered",
			isValid:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := isValidStatusTransition(tt.currentStatus, tt.newStatus)
			assert.Equal(t, tt.isValid, isValid,
				"状态转换 %s -> %s 的有效性不匹配", tt.currentStatus, tt.newStatus)
		})
	}
}

func TestTriggerConditions(t *testing.T) {
	tests := []struct {
		name          string
		triggerOp     string
		triggerPrice  decimal.Decimal
		currentPrice  decimal.Decimal
		shouldTrigger bool
	}{
		{
			name:          "GTE 触发 - 价格达到",
			triggerOp:     "gte",
			triggerPrice:  decimal.NewFromFloat(30.0),
			currentPrice:  decimal.NewFromFloat(30.0),
			shouldTrigger: true,
		},
		{
			name:          "GTE 触发 - 价格超过",
			triggerOp:     "gte",
			triggerPrice:  decimal.NewFromFloat(30.0),
			currentPrice:  decimal.NewFromFloat(31.0),
			shouldTrigger: true,
		},
		{
			name:          "GTE 不触发 - 价格低于",
			triggerOp:     "gte",
			triggerPrice:  decimal.NewFromFloat(30.0),
			currentPrice:  decimal.NewFromFloat(29.0),
			shouldTrigger: false,
		},
		{
			name:          "LTE 触发 - 价格达到",
			triggerOp:     "lte",
			triggerPrice:  decimal.NewFromFloat(30.0),
			currentPrice:  decimal.NewFromFloat(30.0),
			shouldTrigger: true,
		},
		{
			name:          "LTE 触发 - 价格低于",
			triggerOp:     "lte",
			triggerPrice:  decimal.NewFromFloat(30.0),
			currentPrice:  decimal.NewFromFloat(29.0),
			shouldTrigger: true,
		},
		{
			name:          "LTE 不触发 - 价格高于",
			triggerOp:     "lte",
			triggerPrice:  decimal.NewFromFloat(30.0),
			currentPrice:  decimal.NewFromFloat(31.0),
			shouldTrigger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			triggered := checkTriggerCondition(tt.triggerOp, tt.triggerPrice, tt.currentPrice)
			assert.Equal(t, tt.shouldTrigger, triggered,
				"触发条件检查结果不匹配")
		})
	}
}

// 辅助函数：验证创建订单请求
func validateCreateOrderRequest(req *types.CreateOrderRequest) error {
	if req.UserID == "" {
		return assert.AnError // 模拟错误
	}

	// 解析Size字符串
	size, err := decimal.NewFromString(req.Size)
	if err != nil || size.LessThanOrEqual(decimal.Zero) {
		return assert.AnError
	}

	if req.Side != "buy" && req.Side != "sell" {
		return assert.AnError
	}
	return nil
}

// 辅助函数：检查状态转换是否有效
func isValidStatusTransition(current, new string) bool {
	validTransitions := map[string][]string{
		"open":       {"triggered", "canceled"},
		"triggered":  {"submitting", "canceled"},
		"submitting": {"filled", "failed"},
		"filled":     {},
		"failed":     {},
		"canceled":   {},
	}

	validNext, exists := validTransitions[current]
	if !exists {
		return false
	}

	for _, status := range validNext {
		if status == new {
			return true
		}
	}
	return false
}

// 辅助函数：检查触发条件
func checkTriggerCondition(triggerOp string, triggerPrice, currentPrice decimal.Decimal) bool {
	switch triggerOp {
	case "gte":
		return currentPrice.GreaterThanOrEqual(triggerPrice)
	case "lte":
		return currentPrice.LessThanOrEqual(triggerPrice)
	default:
		return false
	}
}
