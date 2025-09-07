package wallet

import (
	"context"
	"fmt"
	"time"

	"solana-limit-order-backend/internal/ledger"
	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// Service 钱包服务接口
type Service interface {
	// CreateWallet 创建钱包
	CreateWallet(ctx context.Context, req *types.CreateWalletRequest) (*types.CreateWalletResponse, error)
	// GetWallet 获取钱包信息
	GetWallet(ctx context.Context, userID, asset string) (*types.GetWalletResponse, error)
	// GetWallets 获取用户所有钱包
	GetWallets(ctx context.Context, userID string) (*types.GetWalletsResponse, error)
	// GetBalance 获取余额
	GetBalance(ctx context.Context, userID, asset string) (*types.GetBalanceResponse, error)
	// GetBalances 获取所有余额
	GetBalances(ctx context.Context, userID string) (*types.GetBalancesResponse, error)
	// Deposit 充值
	Deposit(ctx context.Context, req *types.DepositRequest) (*types.DepositResponse, error)
	// Withdraw 提现
	Withdraw(ctx context.Context, req *types.WithdrawRequest) (*types.WithdrawResponse, error)
	// WalletExists 检查钱包是否存在
	WalletExists(ctx context.Context, userID, asset string) (bool, error)
}

// service 钱包服务实现
type service struct {
	db         DBInterface
	walletRepo repository.WalletRepository
	userRepo   repository.UserRepository
	ledger     ledger.Service
}

// NewService 创建钱包服务
func NewService(
	db *repository.DB,
	walletRepo repository.WalletRepository,
	userRepo repository.UserRepository,
	ledgerService ledger.Service,
) Service {
	return &service{
		db:         db,
		walletRepo: walletRepo,
		userRepo:   userRepo,
		ledger:     ledgerService,
	}
}

// CreateWallet 创建钱包
func (s *service) CreateWallet(ctx context.Context, req *types.CreateWalletRequest) (*types.CreateWalletResponse, error) {
	// 解析用户 ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户 ID: %s", req.UserID)
	}

	// 检查用户是否存在
	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 检查钱包是否已存在
	existingWallet, err := s.walletRepo.GetByUserIDAndAsset(ctx, userID, req.Asset)
	if err == nil && existingWallet != nil {
		return nil, fmt.Errorf("钱包已存在: user_id=%s, asset=%s", req.UserID, req.Asset)
	}

	// 创建新钱包
	wallet := &types.Wallet{
		ID:               uuid.New(),
		UserID:           userID,
		Asset:            req.Asset,
		AvailableDecimal: decimal.Zero,
		LockedDecimal:    decimal.Zero,
		UpdatedAt:        time.Now().UTC(),
	}

	// 使用事务执行
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 创建钱包
		if err := s.walletRepo.CreateWithTx(ctx, tx, wallet); err != nil {
			return fmt.Errorf("创建钱包失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.CreateWalletResponse{
		WalletID:         wallet.ID.String(),
		UserID:           wallet.UserID.String(),
		Asset:            wallet.Asset,
		AvailableBalance: wallet.AvailableDecimal,
		LockedBalance:    wallet.LockedDecimal,
		CreatedAt:        wallet.UpdatedAt,
	}, nil
}

// GetWallet 获取钱包信息
func (s *service) GetWallet(ctx context.Context, userID, asset string) (*types.GetWalletResponse, error) {
	// 解析用户 ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户 ID: %s", userID)
	}

	// 查询钱包
	wallet, err := s.walletRepo.GetByUserIDAndAsset(ctx, uid, asset)
	if err != nil {
		return nil, fmt.Errorf("获取钱包失败: %w", err)
	}

	return &types.GetWalletResponse{
		WalletID:         wallet.ID.String(),
		UserID:           wallet.UserID.String(),
		Asset:            wallet.Asset,
		AvailableBalance: wallet.AvailableDecimal,
		LockedBalance:    wallet.LockedDecimal,
		UpdatedAt:        wallet.UpdatedAt,
	}, nil
}

// GetWallets 获取用户所有钱包
func (s *service) GetWallets(ctx context.Context, userID string) (*types.GetWalletsResponse, error) {
	// 解析用户 ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户 ID: %s", userID)
	}

	// 查询用户所有钱包
	wallets, err := s.walletRepo.GetByUserID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("获取用户钱包失败: %w", err)
	}

	// 转换为响应格式
	walletInfos := make([]types.WalletInfo, len(wallets))
	for i, wallet := range wallets {
		walletInfos[i] = types.WalletInfo{
			WalletID:         wallet.ID.String(),
			Asset:            wallet.Asset,
			AvailableBalance: wallet.AvailableDecimal.String(),
			LockedBalance:    wallet.LockedDecimal.String(),
			UpdatedAt:        wallet.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &types.GetWalletsResponse{
		Wallets: walletInfos,
	}, nil
}

// GetBalance 获取余额
func (s *service) GetBalance(ctx context.Context, userID, asset string) (*types.GetBalanceResponse, error) {
	// 解析用户 ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户 ID: %s", userID)
	}

	// 查询钱包
	wallet, err := s.walletRepo.GetByUserIDAndAsset(ctx, uid, asset)
	if err != nil {
		return nil, fmt.Errorf("获取钱包失败: %w", err)
	}

	totalBalance := wallet.AvailableDecimal.Add(wallet.LockedDecimal)

	return &types.GetBalanceResponse{
		UserID:           wallet.UserID.String(),
		Asset:            wallet.Asset,
		AvailableBalance: wallet.AvailableDecimal,
		LockedBalance:    wallet.LockedDecimal,
		TotalBalance:     totalBalance,
		UpdatedAt:        wallet.UpdatedAt,
	}, nil
}

// GetBalances 获取所有余额
func (s *service) GetBalances(ctx context.Context, userID string) (*types.GetBalancesResponse, error) {
	// 解析用户 ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户 ID: %s", userID)
	}

	// 查询用户所有钱包
	wallets, err := s.walletRepo.GetByUserID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("获取用户钱包失败: %w", err)
	}

	// 转换为余额信息
	balances := make([]types.BalanceInfo, len(wallets))
	for i, wallet := range wallets {
		totalBalance := wallet.AvailableDecimal.Add(wallet.LockedDecimal)
		balances[i] = types.BalanceInfo{
			Asset:            wallet.Asset,
			AvailableBalance: wallet.AvailableDecimal.String(),
			LockedBalance:    wallet.LockedDecimal.String(),
			TotalBalance:     totalBalance.String(),
			UpdatedAt:        wallet.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &types.GetBalancesResponse{
		UserID:   userID,
		Balances: balances,
	}, nil
}

// Deposit 充值
func (s *service) Deposit(ctx context.Context, req *types.DepositRequest) (*types.DepositResponse, error) {
	// 解析用户 ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户 ID: %s", req.UserID)
	}

	// 解析充值金额
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("无效的充值金额: %s", req.Amount)
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("充值金额必须大于零: %s", req.Amount)
	}

	// 检查用户是否存在
	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 生成交易 ID
	transactionID := uuid.New()
	var updatedWallet *types.Wallet

	// 使用事务执行
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 检查钱包是否存在，不存在则创建
		wallet, err := s.walletRepo.GetByUserIDAndAsset(ctx, userID, req.Asset)
		if err != nil {
			// 钱包不存在，创建新钱包
			wallet = &types.Wallet{
				ID:               uuid.New(),
				UserID:           userID,
				Asset:            req.Asset,
				AvailableDecimal: decimal.Zero,
				LockedDecimal:    decimal.Zero,
				UpdatedAt:        time.Now().UTC(),
			}
			if err := s.walletRepo.CreateWithTx(ctx, tx, wallet); err != nil {
				return fmt.Errorf("创建钱包失败: %w", err)
			}
		}

		// 更新钱包余额
		updatedWallet, err = s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, req.Asset, amount, decimal.Zero)
		if err != nil {
			return fmt.Errorf("更新钱包余额失败: %w", err)
		}

		// 记录账本条目
		ledgerEntry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        req.Asset,
			Delta:        amount,
			BalanceAfter: updatedWallet.AvailableDecimal,
			Type:         "deposit",
			RefID:        transactionID,
			CreatedAt:    time.Now().UTC(),
		}

		// 写入账本
		if err := s.ledger.CreateEntryWithTx(ctx, tx, ledgerEntry); err != nil {
			return fmt.Errorf("记录账本失败: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.DepositResponse{
		TransactionID:    transactionID.String(),
		UserID:           req.UserID,
		Asset:            req.Asset,
		Amount:           amount,
		AvailableBalance: updatedWallet.AvailableDecimal,
		LockedBalance:    updatedWallet.LockedDecimal,
		CreatedAt:        time.Now().UTC(),
	}, nil
}

// Withdraw 提现
func (s *service) Withdraw(ctx context.Context, req *types.WithdrawRequest) (*types.WithdrawResponse, error) {
	// 解析用户 ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户 ID: %s", req.UserID)
	}

	// 解析提现金额
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("无效的提现金额: %s", req.Amount)
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("提现金额必须大于零: %s", req.Amount)
	}

	// 检查用户是否存在
	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 生成交易 ID
	transactionID := uuid.New()
	var updatedWallet *types.Wallet

	// 使用事务执行
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {

		// 锁定钱包记录
		wallet, err := s.walletRepo.LockForUpdate(ctx, tx, userID, req.Asset)
		if err != nil {
			return fmt.Errorf("锁定钱包失败: %w", err)
		}

		// 检查余额是否足够
		if wallet.AvailableDecimal.LessThan(amount) {
			return fmt.Errorf("余额不足: 可用余额 %s, 提现金额 %s",
				wallet.AvailableDecimal.String(), amount.String())
		}

		// 更新钱包余额（减少可用余额）
		updatedWallet, err = s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, req.Asset, amount.Neg(), decimal.Zero)
		if err != nil {
			return fmt.Errorf("更新钱包余额失败: %w", err)
		}

		// 记录账本条目
		ledgerEntry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        req.Asset,
			Delta:        amount.Neg(),
			BalanceAfter: updatedWallet.AvailableDecimal,
			Type:         "withdraw",
			RefID:        transactionID,
			CreatedAt:    time.Now().UTC(),
		}

		// 写入账本
		if err := s.ledger.CreateEntryWithTx(ctx, tx, ledgerEntry); err != nil {
			return fmt.Errorf("记录账本失败: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.WithdrawResponse{
		TransactionID:    transactionID.String(),
		UserID:           req.UserID,
		Asset:            req.Asset,
		Amount:           amount,
		ToAddress:        req.ToAddress,
		AvailableBalance: updatedWallet.AvailableDecimal,
		LockedBalance:    updatedWallet.LockedDecimal,
		Status:           "pending", // 提现状态：pending, processing, completed, failed
		CreatedAt:        time.Now().UTC(),
	}, nil
}

// WalletExists 检查钱包是否存在
func (s *service) WalletExists(ctx context.Context, userID, asset string) (bool, error) {
	// 解析用户 ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("无效的用户 ID: %s", userID)
	}

	// 查询钱包
	_, err = s.walletRepo.GetByUserIDAndAsset(ctx, uid, asset)
	if err != nil {
		// 如果是记录不存在的错误，返回 false
		if err.Error() == "钱包不存在" {
			return false, nil
		}
		return false, fmt.Errorf("检查钱包存在性失败: %w", err)
	}

	return true, nil
}
