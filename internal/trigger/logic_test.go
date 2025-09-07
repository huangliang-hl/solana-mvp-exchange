package trigger

import (
	"context"
	"testing"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTriggerLogic_ShouldTriggerOrder(t *testing.T) {
	// 创建一个简单的触发引擎实例用于测试
	engine := &Engine{}

	tests := []struct {
		name         string
		order        *types.Order
		currentPrice decimal.Decimal
		expected     bool
	}{
		{
			name: "GTE触发 - 价格达到触发点",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(30.0),
				TriggerOp:    types.TriggerOpGTE,
			},
			currentPrice: decimal.NewFromFloat(30.0),
			expected:     true,
		},
		{
			name: "GTE触发 - 价格超过触发点",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(30.0),
				TriggerOp:    types.TriggerOpGTE,
			},
			currentPrice: decimal.NewFromFloat(31.0),
			expected:     true,
		},
		{
			name: "GTE不触发 - 价格低于触发点",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(30.0),
				TriggerOp:    types.TriggerOpGTE,
			},
			currentPrice: decimal.NewFromFloat(29.0),
			expected:     false,
		},
		{
			name: "LTE触发 - 价格达到触发点",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(30.0),
				TriggerOp:    types.TriggerOpLTE,
			},
			currentPrice: decimal.NewFromFloat(30.0),
			expected:     true,
		},
		{
			name: "LTE触发 - 价格低于触发点",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(30.0),
				TriggerOp:    types.TriggerOpLTE,
			},
			currentPrice: decimal.NewFromFloat(29.0),
			expected:     true,
		},
		{
			name: "LTE不触发 - 价格高于触发点",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(30.0),
				TriggerOp:    types.TriggerOpLTE,
			},
			currentPrice: decimal.NewFromFloat(31.0),
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tick := &types.PriceTick{
				MidPrice: tt.currentPrice,
			}
			result := engine.shouldTriggerOrder(tt.order, tick)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSymbol(t *testing.T) {
	tests := []struct {
		name      string
		symbol    string
		wantBase  string
		wantQuote string
		wantErr   bool
	}{
		{
			name:      "正常格式",
			symbol:    "SOL/USDC",
			wantBase:  "SOL",
			wantQuote: "USDC",
			wantErr:   false,
		},
		{
			name:      "其他交易对",
			symbol:    "BTC/ETH",
			wantBase:  "BTC",
			wantQuote: "ETH",
			wantErr:   false,
		},
		{
			name:    "无分隔符",
			symbol:  "SOLUSDC",
			wantErr: true,
		},
		{
			name:    "空字符串",
			symbol:  "",
			wantErr: true,
		},
		{
			name:    "只有分隔符",
			symbol:  "/",
			wantErr: true,
		},
		{
			name:    "分隔符在开头",
			symbol:  "/USDC",
			wantErr: true,
		},
		{
			name:    "分隔符在结尾",
			symbol:  "SOL/",
			wantErr: true,
		},
		{
			name:    "太短",
			symbol:  "AB",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, quote, err := parseSymbol(tt.symbol)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantBase, base)
				assert.Equal(t, tt.wantQuote, quote)
			}
		})
	}
}

func TestGenerateIdempotencyKey(t *testing.T) {
	orderID := "order-123"
	sequenceID := int64(12345)

	key := generateIdempotencyKey(orderID, sequenceID)
	expected := "trigger-order-123-12345"

	assert.Equal(t, expected, key)
}

func TestValidateTriggerCondition(t *testing.T) {
	tests := []struct {
		name         string
		order        *types.Order
		currentPrice decimal.Decimal
		wantErr      bool
		errMsg       string
	}{
		{
			name: "有效的GTE条件",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(30.0),
				TriggerOp:    types.TriggerOpGTE,
			},
			currentPrice: decimal.NewFromFloat(31.0),
			wantErr:      false,
		},
		{
			name: "有效的LTE条件",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(30.0),
				TriggerOp:    types.TriggerOpLTE,
			},
			currentPrice: decimal.NewFromFloat(29.0),
			wantErr:      false,
		},
		{
			name: "无效的触发价格 - 零",
			order: &types.Order{
				TriggerPrice: decimal.Zero,
				TriggerOp:    types.TriggerOpGTE,
			},
			currentPrice: decimal.NewFromFloat(30.0),
			wantErr:      true,
			errMsg:       "触发价格必须大于 0",
		},
		{
			name: "无效的触发价格 - 负数",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(-10.0),
				TriggerOp:    types.TriggerOpGTE,
			},
			currentPrice: decimal.NewFromFloat(30.0),
			wantErr:      true,
			errMsg:       "触发价格必须大于 0",
		},
		{
			name: "无效的触发操作",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(30.0),
				TriggerOp:    "invalid",
			},
			currentPrice: decimal.NewFromFloat(30.0),
			wantErr:      true,
			errMsg:       "无效的触发操作",
		},
		{
			name: "触发价格偏离过远 - 高于50%",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(50.0),
				TriggerOp:    types.TriggerOpGTE,
			},
			currentPrice: decimal.NewFromFloat(30.0), // 触发价格比当前价格高67%
			wantErr:      true,
			errMsg:       "触发价格偏离当前价格过远",
		},
		{
			name: "触发价格偏离过远 - 低于50%",
			order: &types.Order{
				TriggerPrice: decimal.NewFromFloat(10.0),
				TriggerOp:    types.TriggerOpLTE,
			},
			currentPrice: decimal.NewFromFloat(30.0), // 触发价格比当前价格低67%
			wantErr:      true,
			errMsg:       "触发价格偏离当前价格过远",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTriggerCondition(tt.order, tt.currentPrice)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTriggerLogic_PrecisionEdgeCases(t *testing.T) {
	engine := &Engine{}

	// 测试边界情况
	tests := []struct {
		name         string
		triggerPrice string
		triggerOp    types.TriggerOp
		currentPrice string
		expected     bool
	}{
		{
			name:         "GTE - 精确匹配",
			triggerPrice: "30.123456789",
			triggerOp:    types.TriggerOpGTE,
			currentPrice: "30.123456789",
			expected:     true,
		},
		{
			name:         "LTE - 精确匹配",
			triggerPrice: "30.123456789",
			triggerOp:    types.TriggerOpLTE,
			currentPrice: "30.123456789",
			expected:     true,
		},
		{
			name:         "GTE - 微小差异不触发",
			triggerPrice: "30.123456789",
			triggerOp:    types.TriggerOpGTE,
			currentPrice: "30.123456788",
			expected:     false,
		},
		{
			name:         "LTE - 微小差异不触发",
			triggerPrice: "30.123456789",
			triggerOp:    types.TriggerOpLTE,
			currentPrice: "30.123456790",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			triggerPrice, _ := decimal.NewFromString(tt.triggerPrice)
			currentPrice, _ := decimal.NewFromString(tt.currentPrice)

			order := &types.Order{
				TriggerPrice: triggerPrice,
				TriggerOp:    tt.triggerOp,
			}

			tick := &types.PriceTick{
				MidPrice: currentPrice,
			}

			result := engine.shouldTriggerOrder(order, tick)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatcherLogic(t *testing.T) {
	// 创建一个简单的matcher用于测试
	engine := &Engine{}
	matcher := &Matcher{engine: engine}

	// 创建测试订单
	order1 := &types.Order{
		ID:           uuid.New(),
		TriggerPrice: decimal.NewFromFloat(30.0),
		TriggerOp:    types.TriggerOpLTE,
	}

	order2 := &types.Order{
		ID:           uuid.New(),
		TriggerPrice: decimal.NewFromFloat(32.0),
		TriggerOp:    types.TriggerOpGTE,
	}

	order3 := &types.Order{
		ID:           uuid.New(),
		TriggerPrice: decimal.NewFromFloat(35.0),
		TriggerOp:    types.TriggerOpLTE,
	}

	orders := []*types.Order{order1, order2, order3}

	tick := &types.PriceTick{
		MidPrice: decimal.NewFromFloat(31.0),
	}

	matchedOrders := matcher.MatchOrders(orders, tick)

	// 应该匹配 order1 (31.0 <= 30.0 为 false，不匹配)
	// order2 (31.0 >= 32.0 为 false，不匹配)
	// order3 (31.0 <= 35.0 为 true，匹配)
	assert.Len(t, matchedOrders, 1)
	assert.Equal(t, order3.ID, matchedOrders[0].ID)
}

func TestEngineGetStats(t *testing.T) {
	engine := &Engine{
		batchSize: 100,
		symbols:   []string{"SOL/USDC"},
		ctx:       context.Background(),
	}

	stats := engine.GetStats()

	assert.Equal(t, 100, stats["batch_size"])
	assert.Equal(t, []string{"SOL/USDC"}, stats["symbols"])
	// running 状态取决于 ctx.Err() 的值，context.Background()的Err()为nil，所以为true
	assert.Equal(t, true, stats["running"])
}
