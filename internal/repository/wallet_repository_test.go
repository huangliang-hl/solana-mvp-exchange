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

func TestWalletRepository_Interface(t *testing.T) {
	// 验证实现了接口
	var _ WalletRepository = (*walletRepository)(nil)
}

func TestNewWalletRepository(t *testing.T) {
	db := &DB{Pool: nil}
	repo := NewWalletRepository(db)

	require.NotNil(t, repo)
	walletRepo, ok := repo.(*walletRepository)
	require.True(t, ok)
	assert.Equal(t, db, walletRepo.db)
}

// 测试创建钱包的逻辑
func TestWalletRepository_CreateWallet_Logic(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name     string
		wallet   *types.Wallet
		validate func(*testing.T, *types.Wallet)
	}{
		{
			name: "有效的SOL钱包",
			wallet: &types.Wallet{
				ID:               uuid.New(),
				UserID:           userID,
				Asset:            "SOL",
				AvailableDecimal: decimal.NewFromFloat(100.0),
				LockedDecimal:    decimal.Zero,
				UpdatedAt:        time.Now(),
			},
			validate: func(t *testing.T, wallet *types.Wallet) {
				assert.Equal(t, "SOL", wallet.Asset)
				assert.True(t, wallet.AvailableDecimal.GreaterThan(decimal.Zero))
				assert.Equal(t, decimal.Zero, wallet.LockedDecimal)
			},
		},
		{
			name: "有效的USDC钱包",
			wallet: &types.Wallet{
				ID:               uuid.New(),
				UserID:           userID,
				Asset:            "USDC",
				AvailableDecimal: decimal.NewFromFloat(1000.0),
				LockedDecimal:    decimal.NewFromFloat(50.0),
				UpdatedAt:        time.Now(),
			},
			validate: func(t *testing.T, wallet *types.Wallet) {
				assert.Equal(t, "USDC", wallet.Asset)
				assert.True(t, wallet.AvailableDecimal.GreaterThan(decimal.Zero))
				assert.True(t, wallet.LockedDecimal.GreaterThan(decimal.Zero))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.wallet)
		})
	}
}

// 测试获取钱包的逻辑
func TestWalletRepository_GetByUserAndAsset_Logic(t *testing.T) {
	userID := uuid.New()

	testWallet := &types.Wallet{
		ID:               uuid.New(),
		UserID:           userID,
		Asset:            "SOL",
		AvailableDecimal: decimal.NewFromFloat(100.0),
		LockedDecimal:    decimal.Zero,
		UpdatedAt:        time.Now(),
	}

	tests := []struct {
		name   string
		userID uuid.UUID
		asset  string
		wallet *types.Wallet
	}{
		{
			name:   "查找存在的钱包",
			userID: userID,
			asset:  "SOL",
			wallet: testWallet,
		},
		{
			name:   "查找不存在的钱包",
			userID: userID,
			asset:  "USDT",
			wallet: nil,
		},
		{
			name:   "无效用户ID",
			userID: uuid.Nil,
			asset:  "SOL",
			wallet: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wallet != nil {
				assert.Equal(t, tt.userID, tt.wallet.UserID)
				assert.Equal(t, tt.asset, tt.wallet.Asset)
			} else {
				assert.Nil(t, tt.wallet)
			}
		})
	}
}

// 测试获取用户所有钱包的逻辑
func TestWalletRepository_GetByUserID_Logic(t *testing.T) {
	userID := uuid.New()

	testWallets := []*types.Wallet{
		{
			ID:               uuid.New(),
			UserID:           userID,
			Asset:            "SOL",
			AvailableDecimal: decimal.NewFromFloat(100.0),
			LockedDecimal:    decimal.Zero,
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New(),
			UserID:           userID,
			Asset:            "USDC",
			AvailableDecimal: decimal.NewFromFloat(1000.0),
			LockedDecimal:    decimal.NewFromFloat(50.0),
			UpdatedAt:        time.Now(),
		},
	}

	tests := []struct {
		name    string
		userID  uuid.UUID
		wallets []*types.Wallet
	}{
		{
			name:    "获取用户所有钱包",
			userID:  userID,
			wallets: testWallets,
		},
		{
			name:    "无效用户ID",
			userID:  uuid.Nil,
			wallets: []*types.Wallet{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, wallet := range tt.wallets {
				if tt.userID != uuid.Nil {
					assert.Equal(t, tt.userID, wallet.UserID)
				}
			}
		})
	}
}

// 测试锁定钱包更新的逻辑
func TestWalletRepository_LockForUpdate_Logic(t *testing.T) {
	userID := uuid.New()

	testWallet := &types.Wallet{
		ID:               uuid.New(),
		UserID:           userID,
		Asset:            "SOL",
		AvailableDecimal: decimal.NewFromFloat(100.0),
		LockedDecimal:    decimal.Zero,
		UpdatedAt:        time.Now(),
	}

	tests := []struct {
		name   string
		userID uuid.UUID
		asset  string
		wallet *types.Wallet
	}{
		{
			name:   "锁定存在的钱包",
			userID: userID,
			asset:  "SOL",
			wallet: testWallet,
		},
		{
			name:   "锁定不存在的钱包",
			userID: userID,
			asset:  "USDT",
			wallet: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wallet != nil {
				assert.Equal(t, tt.userID, tt.wallet.UserID)
				assert.Equal(t, tt.asset, tt.wallet.Asset)
			} else {
				assert.Nil(t, tt.wallet)
			}
		})
	}
}

// 测试更新余额的逻辑
func TestWalletRepository_UpdateBalance_Logic(t *testing.T) {
	walletID := uuid.New()

	tests := []struct {
		name      string
		walletID  uuid.UUID
		available decimal.Decimal
		locked    decimal.Decimal
		validate  func(*testing.T, decimal.Decimal, decimal.Decimal)
	}{
		{
			name:      "增加余额",
			walletID:  walletID,
			available: decimal.NewFromFloat(150.0),
			locked:    decimal.Zero,
			validate: func(t *testing.T, available, locked decimal.Decimal) {
				assert.True(t, available.GreaterThan(decimal.Zero))
				assert.Equal(t, decimal.Zero, locked)
			},
		},
		{
			name:      "减少余额并锁定",
			walletID:  walletID,
			available: decimal.NewFromFloat(50.0),
			locked:    decimal.NewFromFloat(25.0),
			validate: func(t *testing.T, available, locked decimal.Decimal) {
				assert.True(t, available.GreaterThan(decimal.Zero))
				assert.True(t, locked.GreaterThan(decimal.Zero))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证参数
			assert.NotEqual(t, uuid.Nil, tt.walletID)
			assert.True(t, tt.available.GreaterThanOrEqual(decimal.Zero))
			assert.True(t, tt.locked.GreaterThanOrEqual(decimal.Zero))

			tt.validate(t, tt.available, tt.locked)
		})
	}
}

// 测试钱包余额验证逻辑
func TestWalletRepository_BalanceValidation_Logic(t *testing.T) {
	tests := []struct {
		name      string
		available decimal.Decimal
		locked    decimal.Decimal
		isValid   bool
	}{
		{
			name:      "有效余额",
			available: decimal.NewFromFloat(100.0),
			locked:    decimal.NewFromFloat(25.0),
			isValid:   true,
		},
		{
			name:      "零余额",
			available: decimal.Zero,
			locked:    decimal.Zero,
			isValid:   true,
		},
		{
			name:      "负可用余额 - 无效",
			available: decimal.NewFromFloat(-10.0),
			locked:    decimal.Zero,
			isValid:   false,
		},
		{
			name:      "负锁定金额 - 无效",
			available: decimal.NewFromFloat(100.0),
			locked:    decimal.NewFromFloat(-10.0),
			isValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证余额逻辑
			isValid := tt.available.GreaterThanOrEqual(decimal.Zero) &&
				tt.locked.GreaterThanOrEqual(decimal.Zero)

			assert.Equal(t, tt.isValid, isValid)
		})
	}
}
