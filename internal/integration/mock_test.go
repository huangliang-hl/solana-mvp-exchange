package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"solana-limit-order-backend/internal/testutil"
	"solana-limit-order-backend/internal/types"
)

// TestMockClients 测试模拟客户端
func TestMockClients(t *testing.T) {
	// 测试Jupiter模拟客户端
	t.Run("Jupiter模拟客户端", func(t *testing.T) {
		mockJupiter := &testutil.MockJupiterClient{
			QuoteResponse: &testutil.JupiterQuoteResponse{
				InputMint:   "SOL",
				InAmount:    "1000000000",
				OutputMint:  "USDC",
				OutAmount:   "30000000",
				SlippageBps: 50,
			},
		}

		req := &testutil.JupiterQuoteRequest{
			InputMint:   "SOL",
			OutputMint:  "USDC",
			Amount:      "1000000000",
			SlippageBps: 50,
		}

		quote, err := mockJupiter.GetQuote(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "SOL", quote.InputMint)
		assert.Equal(t, "USDC", quote.OutputMint)
		assert.Equal(t, "1000000000", quote.InAmount)

		swapReq := &testutil.JupiterSwapRequest{
			QuoteResponse: quote,
			UserPublicKey: "test_pubkey",
		}

		swap, err := mockJupiter.GetSwapTransaction(context.Background(), swapReq)
		assert.NoError(t, err)
		assert.NotEmpty(t, swap.SwapTransaction)
	})

	// 测试Solana模拟客户端
	t.Run("Solana模拟客户端", func(t *testing.T) {
		mockSolana := &testutil.MockSolanaClient{
			TransactionSignature: "test_signature",
			ConfirmationStatus:   "finalized",
		}

		sig, err := mockSolana.SendTransaction("test_transaction")
		assert.NoError(t, err)
		assert.Equal(t, "test_signature", sig)

		status, err := mockSolana.ConfirmTransaction(sig)
		assert.NoError(t, err)
		assert.Equal(t, "finalized", status)
	})

	// 测试失败场景
	t.Run("模拟失败场景", func(t *testing.T) {
		mockJupiter := &testutil.MockJupiterClient{
			ShouldFail:     true,
			FailureMessage: "Jupiter API 错误",
		}

		req := &testutil.JupiterQuoteRequest{
			InputMint:   "SOL",
			OutputMint:  "USDC",
			Amount:      "1000000000",
			SlippageBps: 50,
		}

		_, err := mockJupiter.GetQuote(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Jupiter API 错误")

		mockSolana := &testutil.MockSolanaClient{
			ShouldFail:     true,
			FailureMessage: "Solana RPC 错误",
		}

		_, err = mockSolana.SendTransaction("test_transaction")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Solana RPC 错误")
	})
}

// TestDataStructures 测试数据结构
func TestDataStructures(t *testing.T) {
	t.Run("订单数据结构", func(t *testing.T) {
		req := types.CreateOrderRequest{
			UserID:              "test-user-123",
			Base:                "SOL",
			Quote:               "USDC",
			Size:                "1.5",
			Side:                "sell",
			TriggerPrice:        "30.5",
			TriggerOp:           "lte",
			MaxSlippagePct:      "0.5",
			PriorityFeeLamports: 5000,
			IdempotencyKey:      "test-key-123",
		}

		assert.Equal(t, "test-user-123", req.UserID)
		assert.Equal(t, "SOL", req.Base)
		assert.Equal(t, "USDC", req.Quote)
		assert.Equal(t, "sell", req.Side)
		assert.Equal(t, "lte", req.TriggerOp)
	})

	t.Run("执行订单任务", func(t *testing.T) {
		task := types.ExecuteOrderMessage{
			OrderID:             "order-123",
			UserID:              "user-123",
			MaxSlippagePct:      "0.5",
			PriorityFeeLamports: 5000,
			IdempotencyKey:      "test-key-123",
			Timestamp:           1697362800000,
		}

		assert.Equal(t, "order-123", task.OrderID)
		assert.Equal(t, "user-123", task.UserID)
		assert.Equal(t, int64(5000), task.PriorityFeeLamports)
	})
}

// TestConfigAndConstants 测试配置和常量
func TestConfigAndConstants(t *testing.T) {
	t.Run("订单状态常量", func(t *testing.T) {
		assert.Equal(t, types.OrderStatus("open"), types.OrderStatusOpen)
		assert.Equal(t, types.OrderStatus("triggered"), types.OrderStatusTriggered)
		assert.Equal(t, types.OrderStatus("filled"), types.OrderStatusFilled)
		assert.Equal(t, types.OrderStatus("failed"), types.OrderStatusFailed)
		assert.Equal(t, types.OrderStatus("canceled"), types.OrderStatusCanceled)
	})

	t.Run("触发操作常量", func(t *testing.T) {
		assert.Equal(t, types.TriggerOp("gte"), types.TriggerOpGTE)
		assert.Equal(t, types.TriggerOp("lte"), types.TriggerOpLTE)
	})

	t.Run("订单方向常量", func(t *testing.T) {
		assert.Equal(t, types.OrderSide("buy"), types.OrderSideBuy)
		assert.Equal(t, types.OrderSide("sell"), types.OrderSideSell)
	})
}
