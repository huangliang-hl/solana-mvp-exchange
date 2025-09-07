package wallet

import (
	"context"
	"testing"
	"time"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock 实现用于测试

// MockDB 模拟数据库接口
type MockDB struct {
	mock.Mock
}

func (m *MockDB) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	args := m.Called(ctx, fn)
	if fn != nil {
		// 模拟事务执行
		return fn(nil)
	}
	return args.Error(0)
}

// MockWalletRepository 模拟钱包仓库
type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) GetByUserIDAndAsset(ctx context.Context, userID uuid.UUID, asset string) (*types.Wallet, error) {
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

func (m *MockWalletRepository) Create(ctx context.Context, wallet *types.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, wallet *types.Wallet) error {
	args := m.Called(ctx, tx, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) UpdateBalances(ctx context.Context, userID uuid.UUID, asset string, availableDelta, lockedDelta decimal.Decimal) error {
	args := m.Called(ctx, userID, asset, availableDelta, lockedDelta)
	return args.Error(0)
}

func (m *MockWalletRepository) UpdateBalancesWithTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string, availableDelta, lockedDelta decimal.Decimal) (*types.Wallet, error) {
	args := m.Called(ctx, tx, userID, asset, availableDelta, lockedDelta)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Wallet), args.Error(1)
}

func (m *MockWalletRepository) LockForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string) (*types.Wallet, error) {
	args := m.Called(ctx, tx, userID, asset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Wallet), args.Error(1)
}

// MockUserRepository 模拟用户仓库
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*types.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, user *types.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, user *types.User) error {
	args := m.Called(ctx, tx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *types.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockLedgerService 模拟账本服务
type MockLedgerService struct {
	mock.Mock
}

func (m *MockLedgerService) LockFunds(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error) {
	args := m.Called(ctx, userID, asset, amount, refID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.BalanceSnapshot), args.Error(1)
}

func (m *MockLedgerService) ReleaseFunds(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error) {
	args := m.Called(ctx, userID, asset, amount, refID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.BalanceSnapshot), args.Error(1)
}

func (m *MockLedgerService) ProcessFill(ctx context.Context, userID uuid.UUID, sellAsset, buyAsset string, sellAmount, buyAmount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error) {
	args := m.Called(ctx, userID, sellAsset, buyAsset, sellAmount, buyAmount, refID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.BalanceSnapshot), args.Error(1)
}

func (m *MockLedgerService) ProcessFillWithLockRelease(ctx context.Context, userID uuid.UUID, order *types.Order, sellAsset, buyAsset string, sellAmount, buyAmount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error) {
	args := m.Called(ctx, userID, order, sellAsset, buyAsset, sellAmount, buyAmount, refID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.BalanceSnapshot), args.Error(1)
}

func (m *MockLedgerService) Compensate(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error) {
	args := m.Called(ctx, userID, asset, amount, refID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.BalanceSnapshot), args.Error(1)
}

func (m *MockLedgerService) GetBalances(ctx context.Context, userID uuid.UUID) (*types.BalanceSnapshot, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.BalanceSnapshot), args.Error(1)
}

func (m *MockLedgerService) CheckSufficientBalance(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal) (bool, error) {
	args := m.Called(ctx, userID, asset, amount)
	return args.Bool(0), args.Error(1)
}

func (m *MockLedgerService) GetLedgerEntries(ctx context.Context, userID uuid.UUID, limit int) ([]*types.LedgerEntry, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.LedgerEntry), args.Error(1)
}

func (m *MockLedgerService) CreateEntryWithTx(ctx context.Context, tx pgx.Tx, entry *types.LedgerEntry) error {
	args := m.Called(ctx, tx, entry)
	return args.Error(0)
}

// 测试帮助函数
func setupWalletService(t *testing.T) (*service, *MockDB, *MockWalletRepository, *MockUserRepository, *MockLedgerService) {
	mockDB := &MockDB{}
	mockWalletRepo := &MockWalletRepository{}
	mockUserRepo := &MockUserRepository{}
	mockLedger := &MockLedgerService{}

	walletService := &service{
		db:         mockDB,
		walletRepo: mockWalletRepo,
		userRepo:   mockUserRepo,
		ledger:     mockLedger,
	}

	return walletService, mockDB, mockWalletRepo, mockUserRepo, mockLedger
}

// TestCreateWallet 测试钱包创建
func TestCreateWallet(t *testing.T) {
	walletService, mockDB, mockWalletRepo, mockUserRepo, _ := setupWalletService(t)

	t.Run("成功创建钱包", func(t *testing.T) {
		// 准备测试数据
		userID := uuid.New()
		req := &types.CreateWalletRequest{
			UserID: userID.String(),
			Asset:  "SOL",
		}

		testUser := &types.User{
			ID:       userID,
			Username: "testuser",
		}

		// 模拟用户存在
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(testUser, nil).Once()

		// 模拟钱包不存在
		mockWalletRepo.On("GetByUserIDAndAsset", mock.Anything, userID, "SOL").Return(nil, assert.AnError).Once()

		// 模拟事务执行
		mockDB.On("WithTx", mock.Anything, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil).Once()

		// 模拟创建钱包
		mockWalletRepo.On("CreateWithTx", mock.Anything, mock.Anything, mock.AnythingOfType("*types.Wallet")).Return(nil).Once()

		// 执行测试
		resp, err := walletService.CreateWallet(context.Background(), req)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, userID.String(), resp.UserID)
		assert.Equal(t, "SOL", resp.Asset)
		assert.True(t, resp.AvailableBalance.IsZero())
		assert.True(t, resp.LockedBalance.IsZero())

		// 验证所有 mock 调用
		mockUserRepo.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockDB.AssertExpectations(t)
	})

	t.Run("无效的用户ID", func(t *testing.T) {
		req := &types.CreateWalletRequest{
			UserID: "invalid-uuid",
			Asset:  "SOL",
		}

		resp, err := walletService.CreateWallet(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "无效的用户 ID")
	})

	t.Run("用户不存在", func(t *testing.T) {
		userID := uuid.New()
		req := &types.CreateWalletRequest{
			UserID: userID.String(),
			Asset:  "SOL",
		}

		// 模拟用户不存在
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(nil, assert.AnError).Once()

		resp, err := walletService.CreateWallet(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "用户不存在")

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("钱包已存在", func(t *testing.T) {
		userID := uuid.New()
		req := &types.CreateWalletRequest{
			UserID: userID.String(),
			Asset:  "SOL",
		}

		testUser := &types.User{
			ID:       userID,
			Username: "testuser",
		}

		existingWallet := &types.Wallet{
			ID:     uuid.New(),
			UserID: userID,
			Asset:  "SOL",
		}

		// 模拟用户存在
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(testUser, nil).Once()

		// 模拟钱包已存在
		mockWalletRepo.On("GetByUserIDAndAsset", mock.Anything, userID, "SOL").Return(existingWallet, nil).Once()

		resp, err := walletService.CreateWallet(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "钱包已存在")

		mockUserRepo.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
	})
}

// TestGetWallet 测试获取钱包信息
func TestGetWallet(t *testing.T) {
	walletService, _, mockWalletRepo, _, _ := setupWalletService(t)

	t.Run("成功获取钱包", func(t *testing.T) {
		userID := uuid.New()
		walletID := uuid.New()
		asset := "SOL"

		testWallet := &types.Wallet{
			ID:               walletID,
			UserID:           userID,
			Asset:            asset,
			AvailableDecimal: decimal.NewFromFloat(100.5),
			LockedDecimal:    decimal.NewFromFloat(50.0),
			UpdatedAt:        time.Now(),
		}

		mockWalletRepo.On("GetByUserIDAndAsset", mock.Anything, userID, asset).Return(testWallet, nil).Once()

		resp, err := walletService.GetWallet(context.Background(), userID.String(), asset)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, walletID.String(), resp.WalletID)
		assert.Equal(t, userID.String(), resp.UserID)
		assert.Equal(t, asset, resp.Asset)
		assert.Equal(t, decimal.NewFromFloat(100.5), resp.AvailableBalance)
		assert.Equal(t, decimal.NewFromFloat(50.0), resp.LockedBalance)

		mockWalletRepo.AssertExpectations(t)
	})

	t.Run("无效的用户ID", func(t *testing.T) {
		resp, err := walletService.GetWallet(context.Background(), "invalid-uuid", "SOL")

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "无效的用户 ID")
	})

	t.Run("钱包不存在", func(t *testing.T) {
		userID := uuid.New()
		asset := "SOL"

		mockWalletRepo.On("GetByUserIDAndAsset", mock.Anything, userID, asset).Return(nil, assert.AnError).Once()

		resp, err := walletService.GetWallet(context.Background(), userID.String(), asset)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "获取钱包失败")

		mockWalletRepo.AssertExpectations(t)
	})
}

// TestGetWallets 测试获取用户所有钱包
func TestGetWallets(t *testing.T) {
	walletService, _, mockWalletRepo, _, _ := setupWalletService(t)

	t.Run("成功获取用户所有钱包", func(t *testing.T) {
		userID := uuid.New()

		testWallets := []*types.Wallet{
			{
				ID:               uuid.New(),
				UserID:           userID,
				Asset:            "SOL",
				AvailableDecimal: decimal.NewFromFloat(100.0),
				LockedDecimal:    decimal.NewFromFloat(10.0),
				UpdatedAt:        time.Now(),
			},
			{
				ID:               uuid.New(),
				UserID:           userID,
				Asset:            "USDC",
				AvailableDecimal: decimal.NewFromFloat(500.0),
				LockedDecimal:    decimal.NewFromFloat(50.0),
				UpdatedAt:        time.Now(),
			},
		}

		mockWalletRepo.On("GetByUserID", mock.Anything, userID).Return(testWallets, nil).Once()

		resp, err := walletService.GetWallets(context.Background(), userID.String())

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Wallets, 2)

		// 验证第一个钱包
		wallet1 := resp.Wallets[0]
		assert.Equal(t, "SOL", wallet1.Asset)
		assert.Equal(t, "100", wallet1.AvailableBalance)
		assert.Equal(t, "10", wallet1.LockedBalance)

		// 验证第二个钱包
		wallet2 := resp.Wallets[1]
		assert.Equal(t, "USDC", wallet2.Asset)
		assert.Equal(t, "500", wallet2.AvailableBalance)
		assert.Equal(t, "50", wallet2.LockedBalance)

		mockWalletRepo.AssertExpectations(t)
	})

	t.Run("无效的用户ID", func(t *testing.T) {
		resp, err := walletService.GetWallets(context.Background(), "invalid-uuid")

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "无效的用户 ID")
	})
}

// TestGetBalance 测试获取余额
func TestGetBalance(t *testing.T) {
	walletService, _, mockWalletRepo, _, _ := setupWalletService(t)

	t.Run("成功获取余额", func(t *testing.T) {
		userID := uuid.New()
		asset := "SOL"

		testWallet := &types.Wallet{
			ID:               uuid.New(),
			UserID:           userID,
			Asset:            asset,
			AvailableDecimal: decimal.NewFromFloat(100.0),
			LockedDecimal:    decimal.NewFromFloat(25.0),
			UpdatedAt:        time.Now(),
		}

		mockWalletRepo.On("GetByUserIDAndAsset", mock.Anything, userID, asset).Return(testWallet, nil).Once()

		resp, err := walletService.GetBalance(context.Background(), userID.String(), asset)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, userID.String(), resp.UserID)
		assert.Equal(t, asset, resp.Asset)
		assert.Equal(t, decimal.NewFromFloat(100.0), resp.AvailableBalance)
		assert.Equal(t, decimal.NewFromFloat(25.0), resp.LockedBalance)
		assert.Equal(t, decimal.NewFromFloat(125.0), resp.TotalBalance) // 100 + 25

		mockWalletRepo.AssertExpectations(t)
	})
}

// TestDeposit 测试充值
func TestDeposit(t *testing.T) {
	walletService, mockDB, mockWalletRepo, mockUserRepo, mockLedger := setupWalletService(t)

	t.Run("成功充值到现有钱包", func(t *testing.T) {
		userID := uuid.New()
		req := &types.DepositRequest{
			UserID: userID.String(),
			Asset:  "SOL",
			Amount: "100.5",
		}

		testUser := &types.User{
			ID:       userID,
			Username: "testuser",
		}

		existingWallet := &types.Wallet{
			ID:               uuid.New(),
			UserID:           userID,
			Asset:            "SOL",
			AvailableDecimal: decimal.NewFromFloat(50.0),
			LockedDecimal:    decimal.NewFromFloat(10.0),
		}

		updatedWallet := &types.Wallet{
			ID:               existingWallet.ID,
			UserID:           userID,
			Asset:            "SOL",
			AvailableDecimal: decimal.NewFromFloat(150.5), // 50 + 100.5
			LockedDecimal:    decimal.NewFromFloat(10.0),
		}

		// 模拟用户存在
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(testUser, nil).Once()

		// 模拟事务执行
		mockDB.On("WithTx", mock.Anything, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil).Once()

		// 模拟钱包存在
		mockWalletRepo.On("GetByUserIDAndAsset", mock.Anything, userID, "SOL").Return(existingWallet, nil).Once()

		// 模拟更新余额
		mockWalletRepo.On("UpdateBalancesWithTx", mock.Anything, mock.Anything, userID, "SOL",
			mock.AnythingOfType("decimal.Decimal"), mock.AnythingOfType("decimal.Decimal")).Return(updatedWallet, nil).Once()

		// 模拟账本记录
		mockLedger.On("CreateEntryWithTx", mock.Anything, mock.Anything,
			mock.AnythingOfType("*types.LedgerEntry")).Return(nil).Once()

		resp, err := walletService.Deposit(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, userID.String(), resp.UserID)
		assert.Equal(t, "SOL", resp.Asset)
		assert.Equal(t, decimal.NewFromFloat(100.5), resp.Amount)
		assert.Equal(t, decimal.NewFromFloat(150.5), resp.AvailableBalance)

		mockUserRepo.AssertExpectations(t)
		mockDB.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockLedger.AssertExpectations(t)
	})

	t.Run("成功充值到新钱包", func(t *testing.T) {
		userID := uuid.New()
		req := &types.DepositRequest{
			UserID: userID.String(),
			Asset:  "USDC",
			Amount: "200.0",
		}

		testUser := &types.User{
			ID:       userID,
			Username: "testuser",
		}

		updatedWallet := &types.Wallet{
			ID:               uuid.New(),
			UserID:           userID,
			Asset:            "USDC",
			AvailableDecimal: decimal.NewFromFloat(200.0),
			LockedDecimal:    decimal.Zero,
		}

		// 模拟用户存在
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(testUser, nil).Once()

		// 模拟事务执行
		mockDB.On("WithTx", mock.Anything, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil).Once()

		// 模拟钱包不存在
		mockWalletRepo.On("GetByUserIDAndAsset", mock.Anything, userID, "USDC").Return(nil, assert.AnError).Once()

		// 模拟创建新钱包
		mockWalletRepo.On("CreateWithTx", mock.Anything, mock.Anything,
			mock.AnythingOfType("*types.Wallet")).Return(nil).Once()

		// 模拟更新余额
		mockWalletRepo.On("UpdateBalancesWithTx", mock.Anything, mock.Anything, userID, "USDC",
			mock.AnythingOfType("decimal.Decimal"), mock.AnythingOfType("decimal.Decimal")).Return(updatedWallet, nil).Once()

		// 模拟账本记录
		mockLedger.On("CreateEntryWithTx", mock.Anything, mock.Anything,
			mock.AnythingOfType("*types.LedgerEntry")).Return(nil).Once()

		resp, err := walletService.Deposit(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, userID.String(), resp.UserID)
		assert.Equal(t, "USDC", resp.Asset)
		assert.True(t, resp.Amount.Equal(decimal.NewFromFloat(200.0)))

		mockUserRepo.AssertExpectations(t)
		mockDB.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockLedger.AssertExpectations(t)
	})

	t.Run("无效充值金额", func(t *testing.T) {
		req := &types.DepositRequest{
			UserID: uuid.New().String(),
			Asset:  "SOL",
			Amount: "invalid",
		}

		resp, err := walletService.Deposit(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "无效的充值金额")
	})

	t.Run("零充值金额", func(t *testing.T) {
		req := &types.DepositRequest{
			UserID: uuid.New().String(),
			Asset:  "SOL",
			Amount: "0",
		}

		resp, err := walletService.Deposit(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "充值金额必须大于零")
	})
}

// TestWithdraw 测试提现
func TestWithdraw(t *testing.T) {
	walletService, mockDB, mockWalletRepo, mockUserRepo, mockLedger := setupWalletService(t)

	t.Run("成功提现", func(t *testing.T) {
		userID := uuid.New()
		req := &types.WithdrawRequest{
			UserID:    userID.String(),
			Asset:     "SOL",
			Amount:    "50.0",
			ToAddress: "test_address",
		}

		testUser := &types.User{
			ID:       userID,
			Username: "testuser",
		}

		lockedWallet := &types.Wallet{
			ID:               uuid.New(),
			UserID:           userID,
			Asset:            "SOL",
			AvailableDecimal: decimal.NewFromFloat(100.0),
			LockedDecimal:    decimal.NewFromFloat(10.0),
		}

		updatedWallet := &types.Wallet{
			ID:               lockedWallet.ID,
			UserID:           userID,
			Asset:            "SOL",
			AvailableDecimal: decimal.NewFromFloat(50.0), // 100 - 50
			LockedDecimal:    decimal.NewFromFloat(10.0),
		}

		// 模拟用户存在
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(testUser, nil).Once()

		// 模拟事务执行
		mockDB.On("WithTx", mock.Anything, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil).Once()

		// 模拟锁定钱包
		mockWalletRepo.On("LockForUpdate", mock.Anything, mock.Anything, userID, "SOL").Return(lockedWallet, nil).Once()

		// 模拟更新余额
		mockWalletRepo.On("UpdateBalancesWithTx", mock.Anything, mock.Anything, userID, "SOL",
			mock.AnythingOfType("decimal.Decimal"), mock.AnythingOfType("decimal.Decimal")).Return(updatedWallet, nil).Once()

		// 模拟账本记录
		mockLedger.On("CreateEntryWithTx", mock.Anything, mock.Anything,
			mock.AnythingOfType("*types.LedgerEntry")).Return(nil).Once()

		resp, err := walletService.Withdraw(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, userID.String(), resp.UserID)
		assert.Equal(t, "SOL", resp.Asset)
		assert.True(t, resp.Amount.Equal(decimal.NewFromFloat(50.0)))
		assert.Equal(t, "test_address", resp.ToAddress)
		assert.Equal(t, "pending", resp.Status)
		assert.True(t, resp.AvailableBalance.Equal(decimal.NewFromFloat(50.0)))

		mockUserRepo.AssertExpectations(t)
		mockDB.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockLedger.AssertExpectations(t)
	})

	t.Run("余额不足", func(t *testing.T) {
		userID := uuid.New()
		req := &types.WithdrawRequest{
			UserID:    userID.String(),
			Asset:     "SOL",
			Amount:    "150.0", // 超过可用余额
			ToAddress: "test_address",
		}

		testUser := &types.User{
			ID:       userID,
			Username: "testuser",
		}

		lockedWallet := &types.Wallet{
			ID:               uuid.New(),
			UserID:           userID,
			Asset:            "SOL",
			AvailableDecimal: decimal.NewFromFloat(100.0), // 可用余额 100
			LockedDecimal:    decimal.NewFromFloat(10.0),
		}

		// 模拟用户存在
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(testUser, nil).Once()

		// 模拟事务执行但余额不足
		mockDB.On("WithTx", mock.Anything, mock.AnythingOfType("func(pgx.Tx) error")).
			Return(func(ctx context.Context, fn func(pgx.Tx) error) error {
				return fn(nil) // 执行传入的函数，让它返回余额不足错误
			}).Once()

		// 模拟锁定钱包
		mockWalletRepo.On("LockForUpdate", mock.Anything, mock.Anything, userID, "SOL").Return(lockedWallet, nil).Once()

		resp, err := walletService.Withdraw(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "余额不足")

		mockUserRepo.AssertExpectations(t)
		mockDB.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
	})

	t.Run("无效提现金额", func(t *testing.T) {
		req := &types.WithdrawRequest{
			UserID:    uuid.New().String(),
			Asset:     "SOL",
			Amount:    "invalid",
			ToAddress: "test_address",
		}

		resp, err := walletService.Withdraw(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "无效的提现金额")
	})
}

// TestWalletExists 测试钱包存在性检查
func TestWalletExists(t *testing.T) {
	walletService, _, mockWalletRepo, _, _ := setupWalletService(t)

	t.Run("钱包存在", func(t *testing.T) {
		userID := uuid.New()
		asset := "SOL"

		testWallet := &types.Wallet{
			ID:     uuid.New(),
			UserID: userID,
			Asset:  asset,
		}

		mockWalletRepo.On("GetByUserIDAndAsset", mock.Anything, userID, asset).Return(testWallet, nil).Once()

		exists, err := walletService.WalletExists(context.Background(), userID.String(), asset)

		require.NoError(t, err)
		assert.True(t, exists)

		mockWalletRepo.AssertExpectations(t)
	})

	t.Run("钱包不存在", func(t *testing.T) {
		userID := uuid.New()
		asset := "SOL"

		// 模拟特定的"钱包不存在"错误
		mockWalletRepo.On("GetByUserIDAndAsset", mock.Anything, userID, asset).
			Return(nil, func() error {
				return func() error { return assert.AnError }()
			}()).Once()

		exists, err := walletService.WalletExists(context.Background(), userID.String(), asset)

		assert.Error(t, err) // 因为模拟返回的是通用错误，不是"钱包不存在"
		assert.False(t, exists)

		mockWalletRepo.AssertExpectations(t)
	})

	t.Run("无效的用户ID", func(t *testing.T) {
		exists, err := walletService.WalletExists(context.Background(), "invalid-uuid", "SOL")

		assert.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "无效的用户 ID")
	})
}
