package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// JupiterClient Jupiter API 客户端
type JupiterClient struct {
	baseURL    string
	httpClient *http.Client
}

// QuoteRequest 报价请求
type QuoteRequest struct {
	InputMint           string `json:"inputMint"`
	OutputMint          string `json:"outputMint"`
	Amount              string `json:"amount"`
	SlippageBps         int    `json:"slippageBps"`
	OnlyDirectRoutes    bool   `json:"onlyDirectRoutes,omitempty"`
	AsLegacyTransaction bool   `json:"asLegacyTransaction,omitempty"`
}

// QuoteResponse 报价响应
type QuoteResponse struct {
	InputMint            string          `json:"inputMint"`
	InAmount             string          `json:"inAmount"`
	OutputMint           string          `json:"outputMint"`
	OutAmount            string          `json:"outAmount"`
	OtherAmountThreshold string          `json:"otherAmountThreshold"`
	SwapMode             string          `json:"swapMode"`
	SlippageBps          int             `json:"slippageBps"`
	PlatformFee          *PlatformFee    `json:"platformFee,omitempty"`
	PriceImpactPct       string          `json:"priceImpactPct"`
	RoutePlan            []RoutePlanStep `json:"routePlan"`
}

// PlatformFee 平台费用
type PlatformFee struct {
	Amount string `json:"amount"`
	FeeBps int    `json:"feeBps"`
}

// RoutePlanStep 路由计划步骤
type RoutePlanStep struct {
	SwapInfo SwapInfo `json:"swapInfo"`
	Percent  int      `json:"percent"`
}

// SwapInfo 交换信息
type SwapInfo struct {
	AmmKey     string `json:"ammKey"`
	Label      string `json:"label"`
	InputMint  string `json:"inputMint"`
	OutputMint string `json:"outputMint"`
	InAmount   string `json:"inAmount"`
	OutAmount  string `json:"outAmount"`
	FeeAmount  string `json:"feeAmount"`
	FeeMint    string `json:"feeMint"`
}

// SwapRequest 交换请求
type SwapRequest struct {
	QuoteResponse                 QuoteResponse `json:"quoteResponse"`
	UserPublicKey                 string        `json:"userPublicKey"`
	WrapAndUnwrapSol              bool          `json:"wrapAndUnwrapSol"`
	UseSharedAccounts             bool          `json:"useSharedAccounts,omitempty"`
	FeeAccount                    string        `json:"feeAccount,omitempty"`
	TrackingAccount               string        `json:"trackingAccount,omitempty"`
	ComputeUnitPriceMicroLamports int           `json:"computeUnitPriceMicroLamports,omitempty"`
}

// SwapResponse 交换响应
type SwapResponse struct {
	SwapTransaction      string `json:"swapTransaction"`
	LastValidBlockHeight int64  `json:"lastValidBlockHeight"`
}

// TokenInfo 代币信息
type TokenInfo struct {
	Symbol   string `json:"symbol"`
	Mint     string `json:"mint"`
	Decimals int    `json:"decimals"`
}

// 常用代币地址
var (
	// Solana 主网代币地址
	SOL_MINT  = "So11111111111111111111111111111111111111112"  // Wrapped SOL
	USDC_MINT = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v" // USDC

	// Devnet 代币地址（用于测试）
	DEVNET_SOL_MINT  = "So11111111111111111111111111111111111111112"
	DEVNET_USDC_MINT = "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU" // Devnet USDC
)

// NewJupiterClient 创建 Jupiter 客户端
func NewJupiterClient(baseURL string) *JupiterClient {
	return &JupiterClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetQuote 获取交换报价
func (c *JupiterClient) GetQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error) {
	// 构建查询参数
	params := url.Values{}
	params.Set("inputMint", req.InputMint)
	params.Set("outputMint", req.OutputMint)
	params.Set("amount", req.Amount)
	params.Set("slippageBps", fmt.Sprintf("%d", req.SlippageBps))

	if req.OnlyDirectRoutes {
		params.Set("onlyDirectRoutes", "true")
	}
	if req.AsLegacyTransaction {
		params.Set("asLegacyTransaction", "true")
	}

	// 构建请求 URL
	requestURL := fmt.Sprintf("%s/quote?%s", c.baseURL, params.Encode())

	log.Debug().
		Str("url", requestURL).
		Str("input_mint", req.InputMint).
		Str("output_mint", req.OutputMint).
		Str("amount", req.Amount).
		Int("slippage_bps", req.SlippageBps).
		Msg("获取 Jupiter 报价")

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jupiter API 错误 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var quoteResp QuoteResponse
	if err := json.Unmarshal(body, &quoteResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	log.Debug().
		Str("input_amount", quoteResp.InAmount).
		Str("output_amount", quoteResp.OutAmount).
		Str("price_impact", quoteResp.PriceImpactPct).
		Msg("获取 Jupiter 报价成功")

	return &quoteResp, nil
}

// GetSwapTransaction 获取交换交易
func (c *JupiterClient) GetSwapTransaction(ctx context.Context, req *SwapRequest) (*SwapResponse, error) {
	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	log.Debug().
		Str("user_public_key", req.UserPublicKey).
		Bool("wrap_unwrap_sol", req.WrapAndUnwrapSol).
		Int("compute_unit_price", req.ComputeUnitPriceMicroLamports).
		Msg("获取 Jupiter 交换交易")

	// 构建请求 URL
	requestURL := fmt.Sprintf("%s/swap", c.baseURL)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jupiter API 错误 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var swapResp SwapResponse
	if err := json.Unmarshal(body, &swapResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	log.Debug().
		Int64("last_valid_block_height", swapResp.LastValidBlockHeight).
		Msg("获取 Jupiter 交换交易成功")

	return &swapResp, nil
}

// GetTokenMint 根据资产符号获取代币地址
func GetTokenMint(symbol string, isDevnet bool) (string, error) {
	switch symbol {
	case "SOL":
		if isDevnet {
			return DEVNET_SOL_MINT, nil
		}
		return SOL_MINT, nil
	case "USDC":
		if isDevnet {
			return DEVNET_USDC_MINT, nil
		}
		return USDC_MINT, nil
	default:
		return "", fmt.Errorf("不支持的代币符号: %s", symbol)
	}
}

// CalculateSlippageBps 计算滑点基点
func CalculateSlippageBps(slippagePct decimal.Decimal) int {
	// 将百分比转换为基点 (1% = 100 bps)
	bps := slippagePct.Mul(decimal.NewFromInt(100))
	bpsInt, _ := bps.Float64()
	return int(bpsInt)
}

// ValidateQuote 验证报价
func ValidateQuote(quote *QuoteResponse, maxSlippagePct decimal.Decimal) error {
	// 检查价格影响
	priceImpact, err := decimal.NewFromString(quote.PriceImpactPct)
	if err != nil {
		return fmt.Errorf("无效的价格影响: %s", quote.PriceImpactPct)
	}

	// 价格影响通常是负数，取绝对值比较
	if priceImpact.Abs().GreaterThan(maxSlippagePct) {
		return fmt.Errorf("价格影响过大: %s%% > %s%%", priceImpact.String(), maxSlippagePct.String())
	}

	// 检查输出金额
	outAmount, err := decimal.NewFromString(quote.OutAmount)
	if err != nil {
		return fmt.Errorf("无效的输出金额: %s", quote.OutAmount)
	}

	if outAmount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("输出金额必须大于 0")
	}

	return nil
}

// EstimateGasAndFees 估算 Gas 和费用
func EstimateGasAndFees(quote *QuoteResponse) (computeUnits int, priorityFee int64, platformFee decimal.Decimal) {
	// 基础计算单元（根据路由复杂度调整）
	computeUnits = 200000 // 基础值

	// 根据路由步骤数量调整
	routeSteps := len(quote.RoutePlan)
	if routeSteps > 1 {
		computeUnits += (routeSteps - 1) * 50000
	}

	// 建议的优先费（微 lamports）
	priorityFee = 5000 // 0.005 SOL

	// 平台费用
	if quote.PlatformFee != nil {
		platformFee, _ = decimal.NewFromString(quote.PlatformFee.Amount)
	}

	return computeUnits, priorityFee, platformFee
}
