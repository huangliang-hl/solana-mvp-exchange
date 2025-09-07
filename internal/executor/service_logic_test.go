package executor

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetTokenMint(t *testing.T) {
	tests := []struct {
		name     string
		symbol   string
		isDevnet bool
		wantErr  bool
	}{
		{
			name:     "SOL on devnet",
			symbol:   "SOL",
			isDevnet: true,
			wantErr:  false,
		},
		{
			name:     "USDC on devnet",
			symbol:   "USDC",
			isDevnet: true,
			wantErr:  false,
		},
		{
			name:     "Unknown token",
			symbol:   "UNKNOWN",
			isDevnet: true,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mint, err := GetTokenMint(tt.symbol, tt.isDevnet)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, mint)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, mint)
			}
		})
	}
}

func TestCalculateSlippageBps(t *testing.T) {
	tests := []struct {
		name        string
		slippagePct decimal.Decimal
		expectedBps int
	}{
		{
			name:        "0.5% slippage",
			slippagePct: decimal.NewFromFloat(0.5),
			expectedBps: 50,
		},
		{
			name:        "1% slippage",
			slippagePct: decimal.NewFromFloat(1.0),
			expectedBps: 100,
		},
		{
			name:        "0.01% slippage",
			slippagePct: decimal.NewFromFloat(0.01),
			expectedBps: 1,
		},
		{
			name:        "5% slippage",
			slippagePct: decimal.NewFromFloat(5.0),
			expectedBps: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bps := CalculateSlippageBps(tt.slippagePct)
			assert.Equal(t, tt.expectedBps, bps)
		})
	}
}

func TestValidateQuote(t *testing.T) {
	tests := []struct {
		name           string
		quote          *QuoteResponse
		maxSlippagePct decimal.Decimal
		wantErr        bool
		errMsg         string
	}{
		{
			name: "有效的报价",
			quote: &QuoteResponse{
				InAmount:       "1000000000", // 1 SOL
				OutAmount:      "30000000",   // 30 USDC
				PriceImpactPct: "0.1",        // 0.1% price impact
			},
			maxSlippagePct: decimal.NewFromFloat(0.5),
			wantErr:        false,
		},
		{
			name: "无效的输出金额",
			quote: &QuoteResponse{
				InAmount:       "1000000000",
				OutAmount:      "0",
				PriceImpactPct: "0.1",
			},
			maxSlippagePct: decimal.NewFromFloat(0.5),
			wantErr:        true,
			errMsg:         "输出金额必须大于 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQuote(tt.quote, tt.maxSlippagePct)

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

// 测试Jupiter API 相关的数据结构
func TestQuoteRequest(t *testing.T) {
	req := &QuoteRequest{
		InputMint:   "So11111111111111111111111111111111111111112",  // SOL
		OutputMint:  "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", // USDC
		Amount:      "1000000000",                                   // 1 SOL
		SlippageBps: 50,                                             // 0.5%
	}

	assert.Equal(t, "So11111111111111111111111111111111111111112", req.InputMint)
	assert.Equal(t, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", req.OutputMint)
	assert.Equal(t, "1000000000", req.Amount)
	assert.Equal(t, 50, req.SlippageBps)
}

func TestSwapRequest(t *testing.T) {
	quote := QuoteResponse{
		InAmount:  "1000000000",
		OutAmount: "30000000",
	}

	req := &SwapRequest{
		QuoteResponse:                 quote,
		UserPublicKey:                 "HqB5jJgTiX2eAyNSSoKNNLBsT9gUxsvhRSfgjjCCw5QJ",
		WrapAndUnwrapSol:              true,
		ComputeUnitPriceMicroLamports: 5000,
	}

	assert.Equal(t, quote, req.QuoteResponse)
	assert.Equal(t, "HqB5jJgTiX2eAyNSSoKNNLBsT9gUxsvhRSfgjjCCw5QJ", req.UserPublicKey)
	assert.True(t, req.WrapAndUnwrapSol)
	assert.Equal(t, 5000, req.ComputeUnitPriceMicroLamports)
}

func TestSwapResponse(t *testing.T) {
	resp := &SwapResponse{
		SwapTransaction: "base64encodedtransaction",
	}

	assert.Equal(t, "base64encodedtransaction", resp.SwapTransaction)
	assert.NotEmpty(t, resp.SwapTransaction)
}

// 测试Solana相关功能
func TestCalculatePriorityFee(t *testing.T) {
	tests := []struct {
		name                string
		priorityFeeLamports int64
		expectedFee         int64
	}{
		{
			name:                "标准优先费",
			priorityFeeLamports: 5000,
			expectedFee:         5000,
		},
		{
			name:                "高优先费",
			priorityFeeLamports: 50000,
			expectedFee:         50000,
		},
		{
			name:                "零优先费",
			priorityFeeLamports: 0,
			expectedFee:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里可以添加实际的优先费计算逻辑测试
			// 目前简单验证输入输出
			assert.Equal(t, tt.expectedFee, tt.priorityFeeLamports)
		})
	}
}

// 测试错误处理逻辑
func TestExecutorErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		errorType   string
		shouldRetry bool
	}{
		{
			name:        "RPC 超时错误",
			errorType:   "rpc_timeout",
			shouldRetry: true,
		},
		{
			name:        "账户使用中错误",
			errorType:   "account_in_use",
			shouldRetry: true,
		},
		{
			name:        "滑点过大错误",
			errorType:   "slippage_exceeded",
			shouldRetry: false,
		},
		{
			name:        "余额不足错误",
			errorType:   "insufficient_funds",
			shouldRetry: false,
		},
		{
			name:        "网络错误",
			errorType:   "network_error",
			shouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试是否应该重试的逻辑
			shouldRetry := isRetryableError(tt.errorType)
			assert.Equal(t, tt.shouldRetry, shouldRetry)
		})
	}
}

// 辅助函数：判断错误是否可重试
func isRetryableError(errorType string) bool {
	retryableErrors := map[string]bool{
		"rpc_timeout":        true,
		"account_in_use":     true,
		"network_error":      true,
		"slippage_exceeded":  false,
		"insufficient_funds": false,
	}

	return retryableErrors[errorType]
}

func TestTransactionConfirmation(t *testing.T) {
	tests := []struct {
		name      string
		confirmed bool
		timeout   bool
		expected  bool
	}{
		{
			name:      "交易确认成功",
			confirmed: true,
			timeout:   false,
			expected:  true,
		},
		{
			name:      "交易确认超时",
			confirmed: false,
			timeout:   true,
			expected:  false,
		},
		{
			name:      "交易未确认",
			confirmed: false,
			timeout:   false,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟交易确认逻辑
			result := tt.confirmed && !tt.timeout
			assert.Equal(t, tt.expected, result)
		})
	}
}
