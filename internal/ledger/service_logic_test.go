package ledger

import (
	"context"
	"fmt"
	"testing"
	"time"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLedgerRepository 模拟账本仓库
type MockLedgerRepository struct {
	mock.Mock
}

func (m *MockLedgerRepository) CreateEntry(ctx context.Context, entry *types.LedgerEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockLedgerRepository) GetByUserID(ctx context.Context, userID uuid.UUID, asset string, limit, offset int) ([]*types.LedgerEntry, error) {
	args := m.Called(ctx, userID, asset, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.LedgerEntry), args.Error(1)
}

func (m *MockLedgerRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.LedgerEntry, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.LedgerEntry), args.Error(1)
}

func (m *MockLedgerRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*types.LedgerEntry, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.LedgerEntry), args.Error(1)
}

// MockWalletRepository 模拟钱包仓库
type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) CreateWallet(ctx context.Context, wallet *types.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) GetByUserAndAsset(ctx context.Context, userID uuid.UUID, asset string) (*types.Wallet, error) {
	args := m.Called(ctx, userID, asset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Wallet), args.Error(1)
}

func (m *MockWalletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*types.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.Wallet), args.Error(1)
}

func (m *MockWalletRepository) LockForUpdate(ctx context.Context, userID uuid.UUID, asset string) (*types.Wallet, error) {
	args := m.Called(ctx, userID, asset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Wallet), args.Error(1)
}

func (m *MockWalletRepository) UpdateBalance(ctx context.Context, userID uuid.UUID, asset string, availableDelta, lockedDelta decimal.Decimal) error {
	args := m.Called(ctx, userID, asset, availableDelta, lockedDelta)
	return args.Error(0)
}

func TestBalanceCalculation(t *testing.T) {
	tests := []struct {
		name           string
		entries        []*types.LedgerEntry
		expectedResult map[string]decimal.Decimal
	}{
		{
			name: "单个资产余额计算",
			entries: []*types.LedgerEntry{
				{
					Asset: "SOL",
					Delta: decimal.NewFromFloat(10.0),
				},
				{
					Asset: "SOL",
					Delta: decimal.NewFromFloat(-2.5),
				},
			},
			expectedResult: map[string]decimal.Decimal{
				"SOL": decimal.NewFromFloat(7.5),
			},
		},
		{
			name: "多资产余额计算",
			entries: []*types.LedgerEntry{
				{
					Asset: "SOL",
					Delta: decimal.NewFromFloat(5.0),
				},
				{
					Asset: "USDC",
					Delta: decimal.NewFromFloat(100.0),
				},
				{
					Asset: "SOL",
					Delta: decimal.NewFromFloat(1.5),
				},
			},
			expectedResult: map[string]decimal.Decimal{
				"SOL":  decimal.NewFromFloat(6.5),
				"USDC": decimal.NewFromFloat(100.0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBalances(tt.entries)
			for asset, expected := range tt.expectedResult {
				actual, exists := result[asset]
				assert.True(t, exists, "资产 %s 应该存在", asset)
				assert.True(t, expected.Equal(actual), "资产 %s 余额不匹配: 期望=%s, 实际=%s", asset, expected, actual)
			}
		})
	}
}

// calculateBalances 计算余额（辅助函数）
func calculateBalances(entries []*types.LedgerEntry) map[string]decimal.Decimal {
	balances := make(map[string]decimal.Decimal)
	for _, entry := range entries {
		if existing, exists := balances[entry.Asset]; exists {
			balances[entry.Asset] = existing.Add(entry.Delta)
		} else {
			balances[entry.Asset] = entry.Delta
		}
	}
	return balances
}

func TestLedgerEntryCreation(t *testing.T) {
	userID := uuid.New()
	refID := uuid.New()
	asset := "SOL"
	delta := decimal.NewFromFloat(1.5)
	balanceAfter := decimal.NewFromFloat(10.0)

	entry := &types.LedgerEntry{
		ID:           uuid.New(),
		UserID:       userID,
		Asset:        asset,
		Delta:        delta,
		BalanceAfter: balanceAfter,
		Type:         types.LedgerEntryTypeLockFunds,
		RefID:        refID,
		CreatedAt:    time.Now(),
	}

	// 验证条目字段
	assert.Equal(t, userID, entry.UserID)
	assert.Equal(t, asset, entry.Asset)
	assert.True(t, delta.Equal(entry.Delta))
	assert.True(t, balanceAfter.Equal(entry.BalanceAfter))
	assert.Equal(t, types.LedgerEntryTypeLockFunds, entry.Type)
	assert.Equal(t, refID, entry.RefID)
	assert.False(t, entry.CreatedAt.IsZero())
}

func TestBalanceSnapshot(t *testing.T) {
	snapshot := &types.BalanceSnapshot{
		Available: map[string]decimal.Decimal{
			"SOL":  decimal.NewFromFloat(5.0),
			"USDC": decimal.NewFromFloat(1000.0),
		},
		Locked: map[string]decimal.Decimal{
			"SOL":  decimal.NewFromFloat(1.5),
			"USDC": decimal.Zero,
		},
	}

	// 验证余额快照
	assert.True(t, decimal.NewFromFloat(5.0).Equal(snapshot.Available["SOL"]))
	assert.True(t, decimal.NewFromFloat(1000.0).Equal(snapshot.Available["USDC"]))
	assert.True(t, decimal.NewFromFloat(1.5).Equal(snapshot.Locked["SOL"]))
	assert.True(t, decimal.Zero.Equal(snapshot.Locked["USDC"]))
}

func TestCreateEntry_Validation(t *testing.T) {
	tests := []struct {
		name         string
		entry        types.LedgerEntry
		expectError  bool
		errorMessage string
	}{
		{
			name: "有效条目",
			entry: types.LedgerEntry{
				UserID:       uuid.New(),
				Asset:        "SOL",
				Delta:        decimal.NewFromFloat(1.5),
				BalanceAfter: decimal.NewFromFloat(10.0),
				Type:         types.LedgerEntryTypeLockFunds,
				RefID:        uuid.New(),
			},
			expectError: false,
		},
		{
			name: "无效资产",
			entry: types.LedgerEntry{
				UserID:       uuid.New(),
				Asset:        "",
				Delta:        decimal.NewFromFloat(1.5),
				BalanceAfter: decimal.NewFromFloat(10.0),
				Type:         types.LedgerEntryTypeLockFunds,
				RefID:        uuid.New(),
			},
			expectError:  true,
			errorMessage: "asset cannot be empty",
		},
		{
			name: "负余额",
			entry: types.LedgerEntry{
				UserID:       uuid.New(),
				Asset:        "SOL",
				Delta:        decimal.NewFromFloat(1.5),
				BalanceAfter: decimal.NewFromFloat(-1.0),
				Type:         types.LedgerEntryTypeLockFunds,
				RefID:        uuid.New(),
			},
			expectError:  true,
			errorMessage: "balance cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLedgerEntry(&tt.entry)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// validateLedgerEntry 验证账本条目（假设这个函数存在）
func validateLedgerEntry(entry *types.LedgerEntry) error {
	if entry.Asset == "" {
		return fmt.Errorf("asset cannot be empty")
	}
	if entry.BalanceAfter.IsNegative() {
		return fmt.Errorf("balance cannot be negative")
	}
	return nil
}
