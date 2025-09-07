package ledger

import (
	"context"
	"fmt"
	"sort"
	"time"

	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// Service 记账服务接口
type Service interface {
	// LockFunds 锁定资金
	LockFunds(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error)

	// ReleaseFunds 释放资金
	ReleaseFunds(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error)

	// ProcessFill 处理成交
	ProcessFill(ctx context.Context, userID uuid.UUID, sellAsset, buyAsset string, sellAmount, buyAmount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error)

	// ProcessFillWithLockRelease 处理成交（释放锁定资金版本）
	ProcessFillWithLockRelease(ctx context.Context, userID uuid.UUID, order *types.Order, sellAsset, buyAsset string, sellAmount, buyAmount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error)

	// Compensate 补偿（失败回退）
	Compensate(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error)

	// GetBalances 获取用户余额
	GetBalances(ctx context.Context, userID uuid.UUID) (*types.BalanceSnapshot, error)

	// CheckSufficientBalance 检查余额是否充足
	CheckSufficientBalance(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal) (bool, error)

	// GetLedgerEntries 获取账本流水
	GetLedgerEntries(ctx context.Context, userID uuid.UUID, limit int) ([]*types.LedgerEntry, error)

	// CreateEntryWithTx 在事务中创建账本条目
	CreateEntryWithTx(ctx context.Context, tx pgx.Tx, entry *types.LedgerEntry) error
}

// service 记账服务实现
type service struct {
	db         *repository.DB
	walletRepo repository.WalletRepository
	ledgerRepo repository.LedgerRepository
}

// NewService 创建记账服务
func NewService(db *repository.DB, walletRepo repository.WalletRepository, ledgerRepo repository.LedgerRepository) Service {
	return &service{
		db:         db,
		walletRepo: walletRepo,
		ledgerRepo: ledgerRepo,
	}
}

// LockFunds 锁定资金
func (s *service) LockFunds(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error) {
	var snapshot *types.BalanceSnapshot

	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 锁定钱包记录
		wallet, err := s.walletRepo.LockForUpdate(ctx, tx, userID, asset)
		if err != nil {
			return fmt.Errorf("锁定钱包失败: %w", err)
		}

		// 检查余额是否充足
		if wallet.AvailableDecimal.LessThan(amount) {
			return fmt.Errorf("余额不足: 可用=%s, 需要=%s", wallet.AvailableDecimal.String(), amount.String())
		}

		// 更新余额：可用减少，锁定增加
		updatedWallet, err := s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, asset, amount.Neg(), amount)
		if err != nil {
			return fmt.Errorf("更新钱包余额失败: %w", err)
		}

		// 创建账本条目
		entry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        asset,
			Delta:        amount.Neg(), // 可用余额减少
			BalanceAfter: updatedWallet.AvailableDecimal,
			Type:         types.LedgerEntryTypeLockFunds,
			RefID:        refID,
			CreatedAt:    time.Now(),
		}

		if err := s.ledgerRepo.CreateWithTx(ctx, tx, entry); err != nil {
			return fmt.Errorf("创建账本条目失败: %w", err)
		}

		// 获取更新后的余额快照
		snapshot, err = s.getBalanceSnapshotWithTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("获取余额快照失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// ReleaseFunds 释放资金
func (s *service) ReleaseFunds(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error) {
	var snapshot *types.BalanceSnapshot

	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 锁定钱包记录
		wallet, err := s.walletRepo.LockForUpdate(ctx, tx, userID, asset)
		if err != nil {
			return fmt.Errorf("锁定钱包失败: %w", err)
		}

		// 检查锁定余额是否充足
		if wallet.LockedDecimal.LessThan(amount) {
			return fmt.Errorf("锁定余额不足: 锁定=%s, 需要释放=%s", wallet.LockedDecimal.String(), amount.String())
		}

		// 更新余额：锁定减少，可用增加
		updatedWallet, err := s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, asset, amount, amount.Neg())
		if err != nil {
			return fmt.Errorf("更新钱包余额失败: %w", err)
		}

		// 创建账本条目
		entry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        asset,
			Delta:        amount, // 可用余额增加
			BalanceAfter: updatedWallet.AvailableDecimal,
			Type:         types.LedgerEntryTypeReleaseFunds,
			RefID:        refID,
			CreatedAt:    time.Now(),
		}

		if err := s.ledgerRepo.CreateWithTx(ctx, tx, entry); err != nil {
			return fmt.Errorf("创建账本条目失败: %w", err)
		}

		// 获取更新后的余额快照
		snapshot, err = s.getBalanceSnapshotWithTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("获取余额快照失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// ProcessFillWithLockRelease 处理成交（释放锁定资金版本）
func (s *service) ProcessFillWithLockRelease(ctx context.Context, userID uuid.UUID, order *types.Order, sellAsset, buyAsset string, sellAmount, buyAmount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error) {
	var snapshot *types.BalanceSnapshot

	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 计算原本锁定的资金
		var originalLockAsset string
		var originalLockAmount decimal.Decimal

		if order.Side == types.OrderSideBuy {
			originalLockAsset = order.Quote
			originalLockAmount = order.Size.Mul(order.TriggerPrice)
		} else {
			originalLockAsset = order.Base
			originalLockAmount = order.Size
		}

		// 预锁定所有涉及的资产钱包，按字母顺序避免死锁
		assetsToLockSet := make(map[string]bool)
		assetsToLockSet[originalLockAsset] = true
		assetsToLockSet[sellAsset] = true
		assetsToLockSet[buyAsset] = true

		// 将资产按字母顺序排序，确保锁定顺序一致
		var assetsToLock []string
		for asset := range assetsToLockSet {
			assetsToLock = append(assetsToLock, asset)
		}

		// 按字母顺序排序以避免死锁
		sort.Strings(assetsToLock)

		wallets := make(map[string]*types.Wallet)
		for _, asset := range assetsToLock {
			wallet, err := s.walletRepo.LockForUpdate(ctx, tx, userID, asset)
			if err != nil {
				return fmt.Errorf("锁定%s钱包失败: %w", asset, err)
			}
			wallets[asset] = wallet
		}

		// 第一步：释放原本锁定的全部资金
		originalWallet := wallets[originalLockAsset]
		if originalWallet.LockedDecimal.LessThan(originalLockAmount) {
			return fmt.Errorf("原始锁定余额不足: 锁定=%s, 需要释放=%s", originalWallet.LockedDecimal.String(), originalLockAmount.String())
		}

		// 释放原本锁定的资金到可用余额
		_, err := s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, originalLockAsset, originalLockAmount, originalLockAmount.Neg())
		if err != nil {
			return fmt.Errorf("释放原始锁定资金失败: %w", err)
		}

		// 创建释放资金的账本条目
		releaseEntry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        originalLockAsset,
			Delta:        originalLockAmount,
			BalanceAfter: originalWallet.AvailableDecimal.Add(originalLockAmount),
			Type:         types.LedgerEntryTypeReleaseFunds,
			RefID:        refID,
			CreatedAt:    time.Now(),
		}

		if err := s.ledgerRepo.CreateWithTx(ctx, tx, releaseEntry); err != nil {
			return fmt.Errorf("创建释放资金账本条目失败: %w", err)
		}

		// 第二步：扣除实际消耗的卖出资产
		// 如果释放和扣除的是同一个资产，需要重新获取钱包状态
		var currentAvailable decimal.Decimal
		if originalLockAsset == sellAsset {
			currentAvailable = originalWallet.AvailableDecimal.Add(originalLockAmount)
		} else {
			currentAvailable = wallets[sellAsset].AvailableDecimal
		}

		if currentAvailable.LessThan(sellAmount) {
			return fmt.Errorf("可用余额不足: 可用=%s, 需要=%s", currentAvailable.String(), sellAmount.String())
		}

		// 从可用余额中扣除实际消耗的金额
		sellWalletAfter, err := s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, sellAsset, sellAmount.Neg(), decimal.Zero)
		if err != nil {
			return fmt.Errorf("扣除卖出资产失败: %w", err)
		}

		// 创建卖出资产账本条目
		sellEntry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        sellAsset,
			Delta:        sellAmount.Neg(),
			BalanceAfter: sellWalletAfter.AvailableDecimal,
			Type:         types.LedgerEntryTypeFilled,
			RefID:        refID,
			CreatedAt:    time.Now(),
		}

		if err := s.ledgerRepo.CreateWithTx(ctx, tx, sellEntry); err != nil {
			return fmt.Errorf("创建卖出资产账本条目失败: %w", err)
		}

		// 第三步：增加买入资产到可用余额
		updatedBuyWallet, err := s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, buyAsset, buyAmount, decimal.Zero)
		if err != nil {
			return fmt.Errorf("增加买入资产失败: %w", err)
		}

		// 创建买入资产账本条目
		buyEntry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        buyAsset,
			Delta:        buyAmount,
			BalanceAfter: updatedBuyWallet.AvailableDecimal,
			Type:         types.LedgerEntryTypeFilled,
			RefID:        refID,
			CreatedAt:    time.Now(),
		}

		if err := s.ledgerRepo.CreateWithTx(ctx, tx, buyEntry); err != nil {
			return fmt.Errorf("创建买入资产账本条目失败: %w", err)
		}

		// 获取更新后的余额快照
		snapshot, err = s.getBalanceSnapshotWithTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("获取余额快照失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// ProcessFill 处理成交（保持向后兼容）
func (s *service) ProcessFill(ctx context.Context, userID uuid.UUID, sellAsset, buyAsset string, sellAmount, buyAmount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error) {
	var snapshot *types.BalanceSnapshot

	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 处理卖出资产：从锁定余额中扣除实际消耗的金额
		sellWallet, err := s.walletRepo.LockForUpdate(ctx, tx, userID, sellAsset)
		if err != nil {
			return fmt.Errorf("锁定卖出资产钱包失败: %w", err)
		}

		if sellWallet.LockedDecimal.LessThan(sellAmount) {
			return fmt.Errorf("锁定余额不足: 锁定=%s, 需要=%s", sellWallet.LockedDecimal.String(), sellAmount.String())
		}

		// 扣除锁定的卖出资产（实际消耗的金额）
		sellWalletAfter, err := s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, sellAsset, decimal.Zero, sellAmount.Neg())
		if err != nil {
			return fmt.Errorf("扣除卖出资产失败: %w", err)
		}

		// 创建卖出资产账本条目
		sellEntry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        sellAsset,
			Delta:        sellAmount.Neg(),
			BalanceAfter: sellWalletAfter.LockedDecimal,
			Type:         types.LedgerEntryTypeFilled,
			RefID:        refID,
			CreatedAt:    time.Now(),
		}

		if err := s.ledgerRepo.CreateWithTx(ctx, tx, sellEntry); err != nil {
			return fmt.Errorf("创建卖出资产账本条目失败: %w", err)
		}

		// 处理买入资产：增加到可用余额
		_, err = s.walletRepo.LockForUpdate(ctx, tx, userID, buyAsset)
		if err != nil {
			return fmt.Errorf("锁定买入资产钱包失败: %w", err)
		}

		// 增加买入资产到可用余额
		updatedBuyWallet, err := s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, buyAsset, buyAmount, decimal.Zero)
		if err != nil {
			return fmt.Errorf("增加买入资产失败: %w", err)
		}

		// 创建买入资产账本条目
		buyEntry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        buyAsset,
			Delta:        buyAmount,
			BalanceAfter: updatedBuyWallet.AvailableDecimal,
			Type:         types.LedgerEntryTypeFilled,
			RefID:        refID,
			CreatedAt:    time.Now(),
		}

		if err := s.ledgerRepo.CreateWithTx(ctx, tx, buyEntry); err != nil {
			return fmt.Errorf("创建买入资产账本条目失败: %w", err)
		}

		// 获取更新后的余额快照
		snapshot, err = s.getBalanceSnapshotWithTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("获取余额快照失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// Compensate 补偿（失败回退）
func (s *service) Compensate(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal, refID uuid.UUID) (*types.BalanceSnapshot, error) {
	var snapshot *types.BalanceSnapshot

	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 锁定钱包记录
		_, err := s.walletRepo.LockForUpdate(ctx, tx, userID, asset)
		if err != nil {
			return fmt.Errorf("锁定钱包失败: %w", err)
		}

		// 增加可用余额作为补偿
		updatedWallet, err := s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, asset, amount, decimal.Zero)
		if err != nil {
			return fmt.Errorf("补偿余额失败: %w", err)
		}

		// 创建补偿账本条目
		entry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        asset,
			Delta:        amount,
			BalanceAfter: updatedWallet.AvailableDecimal,
			Type:         types.LedgerEntryTypeCompensate,
			RefID:        refID,
			CreatedAt:    time.Now(),
		}

		if err := s.ledgerRepo.CreateWithTx(ctx, tx, entry); err != nil {
			return fmt.Errorf("创建补偿账本条目失败: %w", err)
		}

		// 获取更新后的余额快照
		snapshot, err = s.getBalanceSnapshotWithTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("获取余额快照失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// GetBalances 获取用户余额
func (s *service) GetBalances(ctx context.Context, userID uuid.UUID) (*types.BalanceSnapshot, error) {
	wallets, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户钱包失败: %w", err)
	}

	snapshot := &types.BalanceSnapshot{
		Available: make(map[string]decimal.Decimal),
		Locked:    make(map[string]decimal.Decimal),
	}

	for _, wallet := range wallets {
		snapshot.Available[wallet.Asset] = wallet.AvailableDecimal
		snapshot.Locked[wallet.Asset] = wallet.LockedDecimal
	}

	return snapshot, nil
}

// CheckSufficientBalance 检查余额是否充足
func (s *service) CheckSufficientBalance(ctx context.Context, userID uuid.UUID, asset string, amount decimal.Decimal) (bool, error) {
	wallet, err := s.walletRepo.GetByUserIDAndAsset(ctx, userID, asset)
	if err != nil {
		return false, fmt.Errorf("获取钱包失败: %w", err)
	}

	return wallet.AvailableDecimal.GreaterThanOrEqual(amount), nil
}

// GetLedgerEntries 获取账本流水
func (s *service) GetLedgerEntries(ctx context.Context, userID uuid.UUID, limit int) ([]*types.LedgerEntry, error) {
	return s.ledgerRepo.GetByUserID(ctx, userID, limit)
}

// getBalanceSnapshotWithTx 在事务中获取余额快照
func (s *service) getBalanceSnapshotWithTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*types.BalanceSnapshot, error) {
	// 在事务中查询所有钱包
	query := `
		SELECT asset, available_decimal, locked_decimal
		FROM wallets 
		WHERE user_id = $1
	`

	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("查询钱包失败: %w", err)
	}
	defer rows.Close()

	snapshot := &types.BalanceSnapshot{
		Available: make(map[string]decimal.Decimal),
		Locked:    make(map[string]decimal.Decimal),
	}

	for rows.Next() {
		var asset string
		var available, locked decimal.Decimal

		if err := rows.Scan(&asset, &available, &locked); err != nil {
			return nil, fmt.Errorf("扫描钱包数据失败: %w", err)
		}

		snapshot.Available[asset] = available
		snapshot.Locked[asset] = locked
	}

	return snapshot, nil
}

// CreateEntryWithTx 在事务中创建账本条目
func (s *service) CreateEntryWithTx(ctx context.Context, tx pgx.Tx, entry *types.LedgerEntry) error {
	return s.ledgerRepo.CreateWithTx(ctx, tx, entry)
}
