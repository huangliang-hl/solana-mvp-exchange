package executor

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJupiterMockClient_GetQuote(t *testing.T) {
	tests := []struct {
		name        string
		config      *JupiterMockConfig
		request     *QuoteRequest
		expectError bool
	}{
		{
			name: "成功获取报价",
			config: &JupiterMockConfig{
				Enabled:     true,
				SuccessRate: 1.0, // 100% 成功率
				LatencyMs:   10 * time.Millisecond,
				MinSlippage: 0.1,
				MaxSlippage: 2.0,
			},
			request: &QuoteRequest{
				InputMint:   SOL_MINT,
				OutputMint:  DEVNET_USDC_MINT,
				Amount:      "1000000000", // 1 SOL in lamports
				SlippageBps: 50,           // 0.5%
			},
			expectError: false,
		},
		{
			name: "模拟失败",
			config: &JupiterMockConfig{
				Enabled:     true,
				SuccessRate: 0.0, // 0% 成功率
				LatencyMs:   10 * time.Millisecond,
				MinSlippage: 0.1,
				MaxSlippage: 2.0,
			},
			request: &QuoteRequest{
				InputMint:   SOL_MINT,
				OutputMint:  DEVNET_USDC_MINT,
				Amount:      "1000000000",
				SlippageBps: 50,
			},
			expectError: true,
		},
		{
			name: "无效金额",
			config: &JupiterMockConfig{
				Enabled:     true,
				SuccessRate: 1.0,
				LatencyMs:   10 * time.Millisecond,
				MinSlippage: 0.1,
				MaxSlippage: 2.0,
			},
			request: &QuoteRequest{
				InputMint:   SOL_MINT,
				OutputMint:  DEVNET_USDC_MINT,
				Amount:      "invalid_amount",
				SlippageBps: 50,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewJupiterMockClient(tt.config)
			ctx := context.Background()

			start := time.Now()
			quote, err := client.GetQuote(ctx, tt.request)
			elapsed := time.Since(start)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, quote)
			} else {
				require.NoError(t, err)
				require.NotNil(t, quote)

				// 验证响应字段
				assert.Equal(t, tt.request.InputMint, quote.InputMint)
				assert.Equal(t, tt.request.OutputMint, quote.OutputMint)
				assert.Equal(t, tt.request.Amount, quote.InAmount)
				assert.Equal(t, tt.request.SlippageBps, quote.SlippageBps)

				// 验证输出金额是有效的数字
				outAmount, err := decimal.NewFromString(quote.OutAmount)
				require.NoError(t, err)
				assert.True(t, outAmount.GreaterThan(decimal.Zero))

				// 验证价格影响是有效的数字
				priceImpact, err := decimal.NewFromString(quote.PriceImpactPct)
				require.NoError(t, err)
				assert.True(t, priceImpact.LessThanOrEqual(decimal.Zero)) // 价格影响应该是负数或零

				// 验证路由计划
				assert.Len(t, quote.RoutePlan, 1)
				assert.Equal(t, 100, quote.RoutePlan[0].Percent)

				// 验证延迟
				if tt.config.LatencyMs > 0 {
					assert.True(t, elapsed >= tt.config.LatencyMs)
				}
			}
		})
	}
}

func TestJupiterMockClient_GetSwapTransaction(t *testing.T) {
	tests := []struct {
		name        string
		config      *JupiterMockConfig
		request     *SwapRequest
		expectError bool
	}{
		{
			name: "成功获取交换交易",
			config: &JupiterMockConfig{
				Enabled:     true,
				SuccessRate: 1.0,
				LatencyMs:   10 * time.Millisecond,
				MinSlippage: 0.1,
				MaxSlippage: 2.0,
			},
			request: &SwapRequest{
				QuoteResponse: QuoteResponse{
					InputMint:  SOL_MINT,
					OutputMint: DEVNET_USDC_MINT,
					InAmount:   "1000000000",
					OutAmount:  "30000000", // 30 USDC in micro units
				},
				UserPublicKey:                 "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
				WrapAndUnwrapSol:              true,
				ComputeUnitPriceMicroLamports: 5000,
			},
			expectError: false,
		},
		{
			name: "模拟失败",
			config: &JupiterMockConfig{
				Enabled:     true,
				SuccessRate: 0.0,
				LatencyMs:   10 * time.Millisecond,
				MinSlippage: 0.1,
				MaxSlippage: 2.0,
			},
			request: &SwapRequest{
				QuoteResponse: QuoteResponse{
					InputMint:  SOL_MINT,
					OutputMint: DEVNET_USDC_MINT,
					InAmount:   "1000000000",
					OutAmount:  "30000000",
				},
				UserPublicKey:    "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
				WrapAndUnwrapSol: true,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewJupiterMockClient(tt.config)
			ctx := context.Background()

			start := time.Now()
			swap, err := client.GetSwapTransaction(ctx, tt.request)
			elapsed := time.Since(start)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, swap)
			} else {
				require.NoError(t, err)
				require.NotNil(t, swap)

				// 验证响应字段
				assert.NotEmpty(t, swap.SwapTransaction)
				assert.Greater(t, swap.LastValidBlockHeight, int64(0))

				// 验证延迟
				if tt.config.LatencyMs > 0 {
					assert.True(t, elapsed >= tt.config.LatencyMs)
				}
			}
		})
	}
}

func TestJupiterMockClient_CalculateMockOutput(t *testing.T) {
	config := &JupiterMockConfig{
		Enabled:     true,
		SuccessRate: 1.0,
		LatencyMs:   0,
		MinSlippage: 0.1,
		MaxSlippage: 2.0,
	}
	client := NewJupiterMockClient(config)

	tests := []struct {
		name       string
		inputMint  string
		outputMint string
		inAmount   decimal.Decimal
	}{
		{
			name:       "SOL to USDC",
			inputMint:  SOL_MINT,
			outputMint: DEVNET_USDC_MINT,
			inAmount:   decimal.NewFromInt(1000000000), // 1 SOL
		},
		{
			name:       "USDC to SOL",
			inputMint:  DEVNET_USDC_MINT,
			outputMint: SOL_MINT,
			inAmount:   decimal.NewFromInt(30000000), // 30 USDC
		},
		{
			name:       "未知代币对",
			inputMint:  "unknown_mint_1",
			outputMint: "unknown_mint_2",
			inAmount:   decimal.NewFromInt(1000000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outAmount, priceImpact := client.calculateMockOutput(tt.inputMint, tt.outputMint, tt.inAmount)

			// 验证输出金额为正数
			assert.True(t, outAmount.GreaterThan(decimal.Zero))

			// 验证价格影响在合理范围内
			assert.True(t, priceImpact.LessThanOrEqual(decimal.Zero))                   // 应该是负数或零
			assert.True(t, priceImpact.GreaterThanOrEqual(decimal.NewFromFloat(-10.0))) // 不应该超过 -10%
		})
	}
}

func TestJupiterMockClient_ShouldSucceed(t *testing.T) {
	tests := []struct {
		name        string
		successRate float64
		iterations  int
	}{
		{
			name:        "100% 成功率",
			successRate: 1.0,
			iterations:  100,
		},
		{
			name:        "0% 成功率",
			successRate: 0.0,
			iterations:  100,
		},
		{
			name:        "50% 成功率",
			successRate: 0.5,
			iterations:  1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &JupiterMockConfig{
				Enabled:     true,
				SuccessRate: tt.successRate,
				LatencyMs:   0,
				MinSlippage: 0.1,
				MaxSlippage: 2.0,
			}
			client := NewJupiterMockClient(config)

			successCount := 0
			for i := 0; i < tt.iterations; i++ {
				if client.shouldSucceed() {
					successCount++
				}
			}

			successRate := float64(successCount) / float64(tt.iterations)

			if tt.successRate == 1.0 {
				assert.Equal(t, 1.0, successRate)
			} else if tt.successRate == 0.0 {
				assert.Equal(t, 0.0, successRate)
			} else {
				// 对于随机成功率，允许一定的误差范围
				tolerance := 0.1 // 10% 误差
				assert.InDelta(t, tt.successRate, successRate, tolerance)
			}
		})
	}
}

func TestJupiterMockClient_ContextCancellation(t *testing.T) {
	config := &JupiterMockConfig{
		Enabled:     true,
		SuccessRate: 1.0,
		LatencyMs:   100 * time.Millisecond, // 较长的延迟
		MinSlippage: 0.1,
		MaxSlippage: 2.0,
	}
	client := NewJupiterMockClient(config)

	// 测试 GetQuote 的上下文取消
	t.Run("GetQuote context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		request := &QuoteRequest{
			InputMint:   SOL_MINT,
			OutputMint:  DEVNET_USDC_MINT,
			Amount:      "1000000000",
			SlippageBps: 50,
		}

		_, err := client.GetQuote(ctx, request)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	// 测试 GetSwapTransaction 的上下文取消
	t.Run("GetSwapTransaction context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		request := &SwapRequest{
			QuoteResponse: QuoteResponse{
				InputMint:  SOL_MINT,
				OutputMint: DEVNET_USDC_MINT,
				InAmount:   "1000000000",
				OutAmount:  "30000000",
			},
			UserPublicKey:    "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
			WrapAndUnwrapSol: true,
		}

		_, err := client.GetSwapTransaction(ctx, request)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestDefaultJupiterMockConfig(t *testing.T) {
	config := DefaultJupiterMockConfig()

	assert.False(t, config.Enabled)
	assert.Equal(t, 0.95, config.SuccessRate)
	assert.Equal(t, 200*time.Millisecond, config.LatencyMs)
	assert.Equal(t, 0.1, config.MinSlippage)
	assert.Equal(t, 2.0, config.MaxSlippage)
}

func TestJupiterClientInterface(t *testing.T) {
	// 测试真实客户端实现接口
	var realClient JupiterClientInterface = NewJupiterClient("https://example.com")
	assert.NotNil(t, realClient)

	// 测试模拟客户端实现接口
	var mockClient JupiterClientInterface = NewJupiterMockClient(DefaultJupiterMockConfig())
	assert.NotNil(t, mockClient)
}
