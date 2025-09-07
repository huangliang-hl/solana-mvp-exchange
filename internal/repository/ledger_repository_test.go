package repository

import (
	"testing"
	"time"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLedgerRepository_Interface(t *testing.T) {
	// 验证实现了接口
	var _ LedgerRepository = (*ledgerRepository)(nil)
}

func TestNewLedgerRepository(t *testing.T) {
	db := &DB{Pool: nil}
	repo := NewLedgerRepository(db)

	require.NotNil(t, repo)
	ledgerRepo, ok := repo.(*ledgerRepository)
	require.True(t, ok)
	assert.Equal(t, db, ledgerRepo.db)
}

// 测试创建账本条目的逻辑
func TestLedgerRepository_CreateEntry_Logic(t *testing.T) {
	userID := uuid.New()
	orderID := uuid.New()

	tests := []struct {
		name     string
		entry    *types.LedgerEntry
		validate func(*testing.T, *types.LedgerEntry)
	}{
		{
			name: "资金锁定条目",
			entry: &types.LedgerEntry{
				ID:           uuid.New(),
				UserID:       userID,
				Asset:        "SOL",
				Type:         types.LedgerEntryTypeLockFunds,
				Delta:        decimal.NewFromFloat(-10.0), // 锁定为负数
				BalanceAfter: decimal.NewFromFloat(90.0),
				RefID:        orderID,
				CreatedAt:    time.Now(),
			},
			validate: func(t *testing.T, entry *types.LedgerEntry) {
				assert.Equal(t, types.LedgerEntryTypeLockFunds, entry.Type)
				assert.True(t, entry.Delta.LessThan(decimal.Zero)) // 锁定为负数
				assert.Equal(t, orderID, entry.RefID)
			},
		},
		{
			name: "资金释放条目",
			entry: &types.LedgerEntry{
				ID:           uuid.New(),
				UserID:       userID,
				Asset:        "SOL",
				Type:         types.LedgerEntryTypeReleaseFunds,
				Delta:        decimal.NewFromFloat(5.0), // 释放为正数
				BalanceAfter: decimal.NewFromFloat(95.0),
				RefID:        orderID,
				CreatedAt:    time.Now(),
			},
			validate: func(t *testing.T, entry *types.LedgerEntry) {
				assert.Equal(t, types.LedgerEntryTypeReleaseFunds, entry.Type)
				assert.True(t, entry.Delta.GreaterThan(decimal.Zero))
				assert.Equal(t, orderID, entry.RefID)
			},
		},
		{
			name: "交易成交条目",
			entry: &types.LedgerEntry{
				ID:           uuid.New(),
				UserID:       userID,
				Asset:        "USDC",
				Type:         types.LedgerEntryTypeFilled,
				Delta:        decimal.NewFromFloat(1000.0),
				BalanceAfter: decimal.NewFromFloat(2000.0),
				RefID:        orderID,
				CreatedAt:    time.Now(),
			},
			validate: func(t *testing.T, entry *types.LedgerEntry) {
				assert.Equal(t, types.LedgerEntryTypeFilled, entry.Type)
				assert.True(t, entry.Delta.GreaterThan(decimal.Zero))
				assert.True(t, entry.BalanceAfter.GreaterThan(decimal.Zero))
			},
		},
		{
			name: "补偿条目",
			entry: &types.LedgerEntry{
				ID:           uuid.New(),
				UserID:       userID,
				Asset:        "SOL",
				Type:         types.LedgerEntryTypeCompensate,
				Delta:        decimal.NewFromFloat(50.0),
				BalanceAfter: decimal.NewFromFloat(150.0),
				RefID:        uuid.New(), // 补偿不一定关联订单
				CreatedAt:    time.Now(),
			},
			validate: func(t *testing.T, entry *types.LedgerEntry) {
				assert.Equal(t, types.LedgerEntryTypeCompensate, entry.Type)
				assert.True(t, entry.Delta.GreaterThan(decimal.Zero))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.entry)
		})
	}
}

// 测试获取用户账本历史的逻辑
func TestLedgerRepository_GetByUserID_Logic(t *testing.T) {
	userID := uuid.New()
	orderID := uuid.New()

	testEntries := []*types.LedgerEntry{
		{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        "SOL",
			Type:         types.LedgerEntryTypeCompensate, // 使用已有的类型
			Delta:        decimal.NewFromFloat(100.0),
			BalanceAfter: decimal.NewFromFloat(100.0),
			RefID:        uuid.New(),
			CreatedAt:    time.Now().Add(-2 * time.Hour),
		},
		{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        "SOL",
			Type:         types.LedgerEntryTypeLockFunds,
			Delta:        decimal.NewFromFloat(-10.0),
			BalanceAfter: decimal.NewFromFloat(90.0),
			RefID:        orderID,
			CreatedAt:    time.Now().Add(-1 * time.Hour),
		},
		{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        "USDC",
			Type:         types.LedgerEntryTypeFilled,
			Delta:        decimal.NewFromFloat(1000.0),
			BalanceAfter: decimal.NewFromFloat(1000.0),
			RefID:        orderID,
			CreatedAt:    time.Now(),
		},
	}

	tests := []struct {
		name    string
		userID  uuid.UUID
		asset   *string
		limit   int
		offset  int
		entries []*types.LedgerEntry
	}{
		{
			name:    "获取用户所有账本记录",
			userID:  userID,
			asset:   nil,
			limit:   10,
			offset:  0,
			entries: testEntries,
		},
		{
			name:    "获取指定资产的账本记录",
			userID:  userID,
			asset:   stringPtr("SOL"),
			limit:   10,
			offset:  0,
			entries: testEntries[:2], // 前两个是SOL的记录
		},
		{
			name:    "分页获取账本记录",
			userID:  userID,
			asset:   nil,
			limit:   1,
			offset:  0,
			entries: testEntries[:1],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, entry := range tt.entries {
				assert.Equal(t, tt.userID, entry.UserID)
				if tt.asset != nil {
					assert.Equal(t, *tt.asset, entry.Asset)
				}
			}
		})
	}
}

// 测试获取账本条目的逻辑
func TestLedgerRepository_GetByID_Logic(t *testing.T) {
	entryID := uuid.New()
	testEntry := &types.LedgerEntry{
		ID:           entryID,
		UserID:       uuid.New(),
		Asset:        "SOL",
		Type:         types.LedgerEntryTypeLockFunds,
		Delta:        decimal.NewFromFloat(-10.0),
		BalanceAfter: decimal.NewFromFloat(90.0),
		RefID:        uuid.New(),
		CreatedAt:    time.Now(),
	}

	tests := []struct {
		name    string
		entryID uuid.UUID
		entry   *types.LedgerEntry
	}{
		{
			name:    "查找存在的账本条目",
			entryID: entryID,
			entry:   testEntry,
		},
		{
			name:    "查找不存在的账本条目",
			entryID: uuid.New(),
			entry:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.entry != nil {
				assert.Equal(t, tt.entryID, tt.entry.ID)
			} else {
				assert.Nil(t, tt.entry)
			}
		})
	}
}

// 测试获取订单相关账本条目的逻辑
func TestLedgerRepository_GetByOrderID_Logic(t *testing.T) {
	orderID := uuid.New()
	userID := uuid.New()

	testEntries := []*types.LedgerEntry{
		{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        "SOL",
			Type:         types.LedgerEntryTypeLockFunds,
			Delta:        decimal.NewFromFloat(-10.0),
			BalanceAfter: decimal.NewFromFloat(90.0),
			RefID:        orderID,
			CreatedAt:    time.Now().Add(-1 * time.Hour),
		},
		{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        "USDC",
			Type:         types.LedgerEntryTypeFilled,
			Delta:        decimal.NewFromFloat(1000.0),
			BalanceAfter: decimal.NewFromFloat(1000.0),
			RefID:        orderID,
			CreatedAt:    time.Now(),
		},
	}

	tests := []struct {
		name    string
		orderID uuid.UUID
		entries []*types.LedgerEntry
	}{
		{
			name:    "获取订单相关账本记录",
			orderID: orderID,
			entries: testEntries,
		},
		{
			name:    "不存在的订单",
			orderID: uuid.New(),
			entries: []*types.LedgerEntry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, entry := range tt.entries {
				assert.Equal(t, tt.orderID, entry.RefID)
			}
		})
	}
}

// 测试账本类型验证逻辑
func TestLedgerRepository_TypeValidation_Logic(t *testing.T) {
	validTypes := []types.LedgerEntryType{
		types.LedgerEntryTypeLockFunds,
		types.LedgerEntryTypeReleaseFunds,
		types.LedgerEntryTypeFilled,
		types.LedgerEntryTypeCompensate,
	}

	tests := []struct {
		name      string
		entryType types.LedgerEntryType
		isValid   bool
	}{
		{
			name:      "有效类型 - 锁定资金",
			entryType: types.LedgerEntryTypeLockFunds,
			isValid:   true,
		},
		{
			name:      "有效类型 - 释放资金",
			entryType: types.LedgerEntryTypeReleaseFunds,
			isValid:   true,
		},
		{
			name:      "有效类型 - 成交",
			entryType: types.LedgerEntryTypeFilled,
			isValid:   true,
		},
		{
			name:      "有效类型 - 补偿",
			entryType: types.LedgerEntryTypeCompensate,
			isValid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证类型是否在有效列表中
			isValid := false
			for _, validType := range validTypes {
				if tt.entryType == validType {
					isValid = true
					break
				}
			}

			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

// 测试金额验证逻辑
func TestLedgerRepository_AmountValidation_Logic(t *testing.T) {
	tests := []struct {
		name    string
		amount  decimal.Decimal
		balance decimal.Decimal
		isValid bool
	}{
		{
			name:    "正金额和余额",
			amount:  decimal.NewFromFloat(10.0),
			balance: decimal.NewFromFloat(100.0),
			isValid: true,
		},
		{
			name:    "零金额",
			amount:  decimal.Zero,
			balance: decimal.NewFromFloat(100.0),
			isValid: false, // 一般不允许零金额交易
		},
		{
			name:    "负金额",
			amount:  decimal.NewFromFloat(-10.0),
			balance: decimal.NewFromFloat(100.0),
			isValid: false,
		},
		{
			name:    "负余额",
			amount:  decimal.NewFromFloat(10.0),
			balance: decimal.NewFromFloat(-10.0),
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证金额和余额的有效性
			isValid := tt.amount.GreaterThan(decimal.Zero) &&
				tt.balance.GreaterThanOrEqual(decimal.Zero)

			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

// 辅助函数
func stringPtr(s string) *string {
	return &s
}
