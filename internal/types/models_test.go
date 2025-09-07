package types

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestOrderSideValidation(t *testing.T) {
	tests := []struct {
		name  string
		side  OrderSide
		valid bool
	}{
		{"买单", OrderSideBuy, true},
		{"卖单", OrderSideSell, true},
		{"无效订单", OrderSide("invalid"), false},
	}

	validSides := map[OrderSide]bool{
		OrderSideBuy:  true,
		OrderSideSell: true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, exists := validSides[tt.side]
			assert.Equal(t, tt.valid, exists)
		})
	}
}

func TestOrderStatusValidation(t *testing.T) {
	validStatuses := []OrderStatus{
		OrderStatusOpen,
		OrderStatusTriggered,
		OrderStatusSubmitting,
		OrderStatusFilled,
		OrderStatusFailed,
		OrderStatusCanceled,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			assert.NotEmpty(t, string(status))
		})
	}
}

func TestTriggerOpValidation(t *testing.T) {
	tests := []struct {
		name string
		op   TriggerOp
		desc string
	}{
		{"大于等于", TriggerOpGTE, "gte"},
		{"小于等于", TriggerOpLTE, "lte"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.desc, string(tt.op))
		})
	}
}

func TestLedgerEntryTypeValidation(t *testing.T) {
	validTypes := []LedgerEntryType{
		LedgerEntryTypeLockFunds,
		LedgerEntryTypeReleaseFunds,
		LedgerEntryTypeFilled,
		LedgerEntryTypeCompensate,
	}

	for _, entryType := range validTypes {
		t.Run(string(entryType), func(t *testing.T) {
			assert.NotEmpty(t, string(entryType))
		})
	}
}

func TestTxAttemptStatusValidation(t *testing.T) {
	validStatuses := []TxAttemptStatus{
		TxAttemptStatusPending,
		TxAttemptStatusSubmitted,
		TxAttemptStatusConfirmed,
		TxAttemptStatusFailed,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			assert.NotEmpty(t, string(status))
		})
	}
}

func TestUserModel(t *testing.T) {
	userID := uuid.New()
	username := "test_user"
	createdAt := time.Now()

	user := User{
		ID:        userID,
		Username:  username,
		CreatedAt: createdAt,
	}

	assert.Equal(t, userID, user.ID)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, createdAt, user.CreatedAt)
}

func TestWalletModel(t *testing.T) {
	walletID := uuid.New()
	userID := uuid.New()
	asset := "SOL"
	available := decimal.NewFromFloat(100.5)
	locked := decimal.NewFromFloat(50.25)
	updatedAt := time.Now()

	wallet := Wallet{
		ID:               walletID,
		UserID:           userID,
		Asset:            asset,
		AvailableDecimal: available,
		LockedDecimal:    locked,
		UpdatedAt:        updatedAt,
	}

	assert.Equal(t, walletID, wallet.ID)
	assert.Equal(t, userID, wallet.UserID)
	assert.Equal(t, asset, wallet.Asset)
	assert.True(t, available.Equal(wallet.AvailableDecimal))
	assert.True(t, locked.Equal(wallet.LockedDecimal))
	assert.Equal(t, updatedAt, wallet.UpdatedAt)
}

func TestOrderModel(t *testing.T) {
	orderID := uuid.New()
	userID := uuid.New()
	base := "SOL"
	quote := "USDC"
	size := decimal.NewFromFloat(1.5)
	side := OrderSideSell
	triggerPrice := decimal.NewFromFloat(30.5)
	triggerOp := TriggerOpLTE
	status := OrderStatusOpen
	createdAt := time.Now()
	updatedAt := time.Now()
	version := 0

	order := Order{
		ID:           orderID,
		UserID:       userID,
		Base:         base,
		Quote:        quote,
		Size:         size,
		Side:         side,
		TriggerPrice: triggerPrice,
		TriggerOp:    triggerOp,
		Status:       status,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		Version:      version,
	}

	assert.Equal(t, orderID, order.ID)
	assert.Equal(t, userID, order.UserID)
	assert.Equal(t, base, order.Base)
	assert.Equal(t, quote, order.Quote)
	assert.True(t, size.Equal(order.Size))
	assert.Equal(t, side, order.Side)
	assert.True(t, triggerPrice.Equal(order.TriggerPrice))
	assert.Equal(t, triggerOp, order.TriggerOp)
	assert.Equal(t, status, order.Status)
	assert.Equal(t, createdAt, order.CreatedAt)
	assert.Equal(t, updatedAt, order.UpdatedAt)
	assert.Equal(t, version, order.Version)
	assert.Nil(t, order.TriggeredAt)
	assert.Nil(t, order.SubmittedAt)
	assert.Nil(t, order.TxSig)
}

func TestLedgerEntryModel(t *testing.T) {
	entryID := uuid.New()
	userID := uuid.New()
	asset := "SOL"
	delta := decimal.NewFromFloat(-1.5)
	balanceAfter := decimal.NewFromFloat(98.5)
	entryType := LedgerEntryTypeLockFunds
	refID := uuid.New()
	createdAt := time.Now()

	entry := LedgerEntry{
		ID:           entryID,
		UserID:       userID,
		Asset:        asset,
		Delta:        delta,
		BalanceAfter: balanceAfter,
		Type:         entryType,
		RefID:        refID,
		CreatedAt:    createdAt,
	}

	assert.Equal(t, entryID, entry.ID)
	assert.Equal(t, userID, entry.UserID)
	assert.Equal(t, asset, entry.Asset)
	assert.True(t, delta.Equal(entry.Delta))
	assert.True(t, balanceAfter.Equal(entry.BalanceAfter))
	assert.Equal(t, entryType, entry.Type)
	assert.Equal(t, refID, entry.RefID)
	assert.Equal(t, createdAt, entry.CreatedAt)
}

func TestTxAttemptModel(t *testing.T) {
	attemptID := uuid.New()
	orderID := uuid.New()
	params := `{"slippage": "0.5"}`
	status := TxAttemptStatusPending
	createdAt := time.Now()

	attempt := TxAttempt{
		AttemptID: attemptID,
		OrderID:   orderID,
		Params:    params,
		Status:    status,
		CreatedAt: createdAt,
	}

	assert.Equal(t, attemptID, attempt.AttemptID)
	assert.Equal(t, orderID, attempt.OrderID)
	assert.Equal(t, params, attempt.Params)
	assert.Equal(t, status, attempt.Status)
	assert.Equal(t, createdAt, attempt.CreatedAt)
	assert.Nil(t, attempt.TxSig)
}

func TestPriceTickModel(t *testing.T) {
	sequenceID := int64(12345)
	timestamp := int64(1697362800000)
	symbol := "SOL/USDC"
	bidPrice := decimal.NewFromFloat(30.10)
	askPrice := decimal.NewFromFloat(30.30)
	midPrice := decimal.NewFromFloat(30.20)
	source := "simulator"

	tick := PriceTick{
		SequenceID: sequenceID,
		Timestamp:  timestamp,
		Symbol:     symbol,
		BidPrice:   bidPrice,
		AskPrice:   askPrice,
		MidPrice:   midPrice,
		Source:     source,
	}

	assert.Equal(t, sequenceID, tick.SequenceID)
	assert.Equal(t, timestamp, tick.Timestamp)
	assert.Equal(t, symbol, tick.Symbol)
	assert.True(t, bidPrice.Equal(tick.BidPrice))
	assert.True(t, askPrice.Equal(tick.AskPrice))
	assert.True(t, midPrice.Equal(tick.MidPrice))
	assert.Equal(t, source, tick.Source)
}

func TestCreateOrderRequest(t *testing.T) {
	req := CreateOrderRequest{
		UserID:              "user-123",
		Base:                "SOL",
		Quote:               "USDC",
		Size:                "1.5",
		Side:                "sell",
		TriggerPrice:        "30.5",
		TriggerOp:           "lte",
		MaxSlippagePct:      "0.5",
		PriorityFeeLamports: 5000,
		IdempotencyKey:      "uuid-abc123",
	}

	assert.Equal(t, "user-123", req.UserID)
	assert.Equal(t, "SOL", req.Base)
	assert.Equal(t, "USDC", req.Quote)
	assert.Equal(t, "1.5", req.Size)
	assert.Equal(t, "sell", req.Side)
	assert.Equal(t, "30.5", req.TriggerPrice)
	assert.Equal(t, "lte", req.TriggerOp)
	assert.Equal(t, "0.5", req.MaxSlippagePct)
	assert.Equal(t, int64(5000), req.PriorityFeeLamports)
	assert.Equal(t, "uuid-abc123", req.IdempotencyKey)
}

func TestExecuteOrderMessage(t *testing.T) {
	msg := ExecuteOrderMessage{
		OrderID:             "order-123",
		UserID:              "user-123",
		MaxSlippagePct:      "0.5",
		PriorityFeeLamports: 5000,
		IdempotencyKey:      "trigger-order-123-12345",
		Timestamp:           1697362800000,
	}

	assert.Equal(t, "order-123", msg.OrderID)
	assert.Equal(t, "user-123", msg.UserID)
	assert.Equal(t, "0.5", msg.MaxSlippagePct)
	assert.Equal(t, int64(5000), msg.PriorityFeeLamports)
	assert.Equal(t, "trigger-order-123-12345", msg.IdempotencyKey)
	assert.Equal(t, int64(1697362800000), msg.Timestamp)
}

func TestBalanceSnapshot(t *testing.T) {
	available := map[string]decimal.Decimal{
		"SOL":  decimal.NewFromFloat(100.5),
		"USDC": decimal.NewFromFloat(1000.0),
	}
	locked := map[string]decimal.Decimal{
		"SOL":  decimal.NewFromFloat(10.5),
		"USDC": decimal.NewFromFloat(100.0),
	}

	snapshot := BalanceSnapshot{
		Available: available,
		Locked:    locked,
	}

	assert.Equal(t, available, snapshot.Available)
	assert.Equal(t, locked, snapshot.Locked)

	// 测试余额查询
	solAvailable, exists := snapshot.Available["SOL"]
	assert.True(t, exists)
	assert.True(t, decimal.NewFromFloat(100.5).Equal(solAvailable))

	solLocked, exists := snapshot.Locked["SOL"]
	assert.True(t, exists)
	assert.True(t, decimal.NewFromFloat(10.5).Equal(solLocked))
}

func TestErrorResponse(t *testing.T) {
	errorDetail := ErrorDetail{
		Code:      "ORDER_CONFLICT",
		Message:   "order already submitting",
		RequestID: "req-20231015-abcdef",
	}

	errorResp := ErrorResponse{
		Error: errorDetail,
	}

	assert.Equal(t, "ORDER_CONFLICT", errorResp.Error.Code)
	assert.Equal(t, "order already submitting", errorResp.Error.Message)
	assert.Equal(t, "req-20231015-abcdef", errorResp.Error.RequestID)
}

func TestTriggerCheckMessage(t *testing.T) {
	tick := PriceTick{
		SequenceID: 12345,
		Timestamp:  1697362800000,
		Symbol:     "SOL/USDC",
		BidPrice:   decimal.NewFromFloat(30.10),
		AskPrice:   decimal.NewFromFloat(30.30),
		MidPrice:   decimal.NewFromFloat(30.20),
		Source:     "simulator",
	}

	msg := TriggerCheckMessage{
		PriceTick: tick,
	}

	assert.Equal(t, tick, msg.PriceTick)
	assert.Equal(t, int64(12345), msg.PriceTick.SequenceID)
	assert.Equal(t, "SOL/USDC", msg.PriceTick.Symbol)
}

func TestCreateOrderResponse(t *testing.T) {
	balances := map[string]string{
		"SOL":  "100.5",
		"USDC": "1000.0",
	}

	resp := CreateOrderResponse{
		OrderID:           "order-123",
		Status:            "open",
		AvailableBalances: balances,
	}

	assert.Equal(t, "order-123", resp.OrderID)
	assert.Equal(t, "open", resp.Status)
	assert.Equal(t, balances, resp.AvailableBalances)
	assert.Equal(t, "100.5", resp.AvailableBalances["SOL"])
	assert.Equal(t, "1000.0", resp.AvailableBalances["USDC"])
}

func TestCancelOrderResponse(t *testing.T) {
	availableBalances := map[string]string{
		"SOL":  "110.0",
		"USDC": "1100.0",
	}
	lockedBalances := map[string]string{
		"SOL":  "0.0",
		"USDC": "0.0",
	}

	resp := CancelOrderResponse{
		OrderID:           "order-123",
		Status:            "canceled",
		AvailableBalances: availableBalances,
		LockedBalances:    lockedBalances,
	}

	assert.Equal(t, "order-123", resp.OrderID)
	assert.Equal(t, "canceled", resp.Status)
	assert.Equal(t, availableBalances, resp.AvailableBalances)
	assert.Equal(t, lockedBalances, resp.LockedBalances)
}

func TestOrderQueryResponse(t *testing.T) {
	orders := []OrderInfo{
		{
			OrderID:          "order-123",
			BaseAsset:        "SOL",
			QuoteAsset:       "USDC",
			Size:             "1.5",
			Side:             "sell",
			TriggerPrice:     "30.5",
			TriggerCondition: "lte",
			Status:           "open",
			CreatedAt:        "2023-10-15T08:30:00Z",
			UpdatedAt:        "2023-10-15T08:30:00Z",
		},
	}

	resp := OrderQueryResponse{
		Orders: orders,
	}

	assert.Equal(t, orders, resp.Orders)
	assert.Len(t, resp.Orders, 1)
	assert.Equal(t, "order-123", resp.Orders[0].OrderID)
	assert.Equal(t, "SOL", resp.Orders[0].BaseAsset)
}
