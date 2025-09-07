package executor

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// JupiterMockClient Jupiter API 模拟客户端
type JupiterMockClient struct {
	config *JupiterMockConfig
}

// JupiterMockConfig Jupiter 模拟器配置
type JupiterMockConfig struct {
	Enabled     bool          // 是否启用模拟器
	SuccessRate float64       // 成功率 (0.0-1.0)
	LatencyMs   time.Duration // 模拟延迟
	MinSlippage float64       // 最小滑点 (%)
	MaxSlippage float64       // 最大滑点 (%)
}

// NewJupiterMockClient 创建 Jupiter 模拟客户端
func NewJupiterMockClient(config *JupiterMockConfig) *JupiterMockClient {
	return &JupiterMockClient{
		config: config,
	}
}

// GetQuote 模拟获取交换报价
func (m *JupiterMockClient) GetQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error) {
	// 模拟网络延迟
	if m.config.LatencyMs > 0 {
		select {
		case <-time.After(m.config.LatencyMs):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// 根据配置的成功率决定是否返回错误
	if !m.shouldSucceed() {
		return nil, fmt.Errorf("模拟的 Jupiter API 错误")
	}

	log.Debug().
		Str("input_mint", req.InputMint).
		Str("output_mint", req.OutputMint).
		Str("amount", req.Amount).
		Int("slippage_bps", req.SlippageBps).
		Msg("模拟 Jupiter 报价请求")

	// 解析输入金额
	inAmount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("无效的输入金额: %s", req.Amount)
	}

	// 计算模拟输出金额
	outAmount, priceImpact := m.calculateMockOutput(req.InputMint, req.OutputMint, inAmount)

	// 生成模拟报价响应
	response := &QuoteResponse{
		InputMint:            req.InputMint,
		InAmount:             req.Amount,
		OutputMint:           req.OutputMint,
		OutAmount:            outAmount.String(),
		OtherAmountThreshold: outAmount.Mul(decimal.NewFromFloat(0.99)).String(), // 1% 阈值
		SwapMode:             "ExactIn",
		SlippageBps:          req.SlippageBps,
		PriceImpactPct:       priceImpact.String(),
		RoutePlan: []RoutePlanStep{
			{
				SwapInfo: SwapInfo{
					AmmKey:     m.generateMockAmmKey(),
					Label:      "Mock DEX",
					InputMint:  req.InputMint,
					OutputMint: req.OutputMint,
					InAmount:   req.Amount,
					OutAmount:  outAmount.String(),
					FeeAmount:  "0",
					FeeMint:    req.InputMint,
				},
				Percent: 100,
			},
		},
	}

	log.Debug().
		Str("input_amount", response.InAmount).
		Str("output_amount", response.OutAmount).
		Str("price_impact", response.PriceImpactPct).
		Msg("模拟 Jupiter 报价成功")

	return response, nil
}

// GetSwapTransaction 模拟获取交换交易
func (m *JupiterMockClient) GetSwapTransaction(ctx context.Context, req *SwapRequest) (*SwapResponse, error) {
	// 模拟网络延迟
	if m.config.LatencyMs > 0 {
		select {
		case <-time.After(m.config.LatencyMs):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// 根据配置的成功率决定是否返回错误
	if !m.shouldSucceed() {
		return nil, fmt.Errorf("模拟的 Jupiter 交换交易错误")
	}

	log.Debug().
		Str("user_public_key", req.UserPublicKey).
		Bool("wrap_unwrap_sol", req.WrapAndUnwrapSol).
		Int("compute_unit_price", req.ComputeUnitPriceMicroLamports).
		Msg("模拟 Jupiter 交换交易请求")

	// 生成模拟交易数据
	mockTransaction := m.generateMockTransaction()
	mockBlockHeight := m.generateMockBlockHeight()

	response := &SwapResponse{
		SwapTransaction:      mockTransaction,
		LastValidBlockHeight: mockBlockHeight,
	}

	log.Debug().
		Int64("last_valid_block_height", response.LastValidBlockHeight).
		Msg("模拟 Jupiter 交换交易成功")

	return response, nil
}

// shouldSucceed 根据成功率决定是否成功
func (m *JupiterMockClient) shouldSucceed() bool {
	if m.config.SuccessRate >= 1.0 {
		return true
	}
	if m.config.SuccessRate <= 0.0 {
		return false
	}

	// 生成随机数判断是否成功
	randomInt, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		// 如果随机数生成失败，默认成功
		return true
	}

	threshold := int64(m.config.SuccessRate * 10000)
	return randomInt.Int64() < threshold
}

// calculateMockOutput 计算模拟输出金额
func (m *JupiterMockClient) calculateMockOutput(inputMint, outputMint string, inAmount decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	// 获取模拟汇率
	exchangeRate := m.getMockExchangeRate(inputMint, outputMint)

	// 基础输出金额
	baseOutput := inAmount.Mul(exchangeRate)

	// 生成随机滑点
	slippage := m.generateRandomSlippage()

	// 应用滑点（负数表示价格影响）
	priceImpact := decimal.NewFromFloat(-slippage)
	slippageMultiplier := decimal.NewFromInt(1).Add(priceImpact.Div(decimal.NewFromInt(100)))

	// 计算最终输出金额
	finalOutput := baseOutput.Mul(slippageMultiplier)

	// 确保输出金额为正数
	if finalOutput.LessThanOrEqual(decimal.Zero) {
		finalOutput = baseOutput.Mul(decimal.NewFromFloat(0.95)) // 最少 5% 滑点
		priceImpact = decimal.NewFromFloat(-5.0)
	}

	return finalOutput, priceImpact
}

// getMockExchangeRate 获取模拟汇率（最小单位之间的换算）
func (m *JupiterMockClient) getMockExchangeRate(inputMint, outputMint string) decimal.Decimal {
	// 定义各代币的小数位数
	const (
		SOL_DECIMALS  = 9 // SOL 有 9 位小数
		USDC_DECIMALS = 6 // USDC 有 6 位小数
	)

	// 模拟价格：1 SOL = 30 USDC
	baseRate := decimal.NewFromFloat(30.0)

	if inputMint == SOL_MINT && outputMint == DEVNET_USDC_MINT {
		// SOL -> USDC: 1 lamport (10^-9 SOL) 换多少 micro USDC (10^-6 USDC)
		// 1 lamport = 10^-9 SOL = 10^-9 * 30 USDC = 30 * 10^-9 USDC = 30 * 10^-9 * 10^6 micro USDC = 30 * 10^-3 micro USDC
		return baseRate.Mul(decimal.NewFromFloat(0.001)) // 30 * 10^-3
	} else if inputMint == DEVNET_USDC_MINT && outputMint == SOL_MINT {
		// USDC -> SOL: 1 micro USDC (10^-6 USDC) 换多少 lamports (10^-9 SOL)
		// 1 micro USDC = 10^-6 USDC = 10^-6 / 30 SOL = 10^-6 / 30 * 10^9 lamports = 10^3 / 30 lamports
		return decimal.NewFromFloat(1000.0 / 30.0) // 10^3 / 30
	}

	// 默认汇率 1:1
	return decimal.NewFromInt(1)
}

// generateRandomSlippage 生成随机滑点
func (m *JupiterMockClient) generateRandomSlippage() float64 {
	// 在配置的滑点范围内生成随机值
	minSlippage := m.config.MinSlippage
	maxSlippage := m.config.MaxSlippage

	if minSlippage >= maxSlippage {
		return minSlippage
	}

	// 生成 0-1 之间的随机数
	randomInt, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return minSlippage
	}

	randomFloat := float64(randomInt.Int64()) / 10000.0
	return minSlippage + randomFloat*(maxSlippage-minSlippage)
}

// generateMockAmmKey 生成模拟 AMM 密钥
func (m *JupiterMockClient) generateMockAmmKey() string {
	// 生成一个看起来像 Solana 地址的随机字符串
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)[:44] // Solana 地址长度
}

// generateMockTransaction 生成模拟交易数据
func (m *JupiterMockClient) generateMockTransaction() string {
	// 生成一个看起来像序列化交易的 base64 字符串
	bytes := make([]byte, 256) // 典型交易大小
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

// generateMockBlockHeight 生成模拟区块高度
func (m *JupiterMockClient) generateMockBlockHeight() int64 {
	// 生成一个合理的区块高度（当前时间戳的简化版本）
	baseHeight := time.Now().Unix() / 400 // 大约每 400 秒一个区块
	return baseHeight + 150               // 加上 150 个区块的有效期
}

// DefaultJupiterMockConfig 返回默认的模拟器配置
func DefaultJupiterMockConfig() *JupiterMockConfig {
	return &JupiterMockConfig{
		Enabled:     false,
		SuccessRate: 0.95,                   // 95% 成功率
		LatencyMs:   200 * time.Millisecond, // 200ms 延迟
		MinSlippage: 0.1,                    // 0.1% 最小滑点
		MaxSlippage: 2.0,                    // 2.0% 最大滑点
	}
}

// JupiterClientInterface 定义 Jupiter 客户端接口
type JupiterClientInterface interface {
	GetQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error)
	GetSwapTransaction(ctx context.Context, req *SwapRequest) (*SwapResponse, error)
}

// 确保两个客户端都实现了接口
var _ JupiterClientInterface = (*JupiterClient)(nil)
var _ JupiterClientInterface = (*JupiterMockClient)(nil)
