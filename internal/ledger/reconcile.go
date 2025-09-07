package ledger

import (
	"context"
	"fmt"
	"time"

	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// ReconcileService 对账服务接口
type ReconcileService interface {
	// ReconcileUser 对账单个用户
	ReconcileUser(ctx context.Context, userID uuid.UUID) (*ReconcileResult, error)

	// ReconcileAll 对账所有用户
	ReconcileAll(ctx context.Context) ([]*ReconcileResult, error)

	// FixDiscrepancy 修复差异
	FixDiscrepancy(ctx context.Context, userID uuid.UUID, asset string, expectedBalance decimal.Decimal) error
}

// ReconcileResult 对账结果
type ReconcileResult struct {
	UserID      uuid.UUID                  `json:"user_id"`
	Asset       string                     `json:"asset"`
	WalletTotal decimal.Decimal            `json:"wallet_total"`
	LedgerTotal decimal.Decimal            `json:"ledger_total"`
	Discrepancy decimal.Decimal            `json:"discrepancy"`
	IsBalanced  bool                       `json:"is_balanced"`
	Details     map[string]decimal.Decimal `json:"details"`
	CheckedAt   time.Time                  `json:"checked_at"`
}

// reconcileService 对账服务实现
type reconcileService struct {
	db         *repository.DB
	walletRepo repository.WalletRepository
	ledgerRepo repository.LedgerRepository
}

// NewReconcileService 创建对账服务
func NewReconcileService(db *repository.DB, walletRepo repository.WalletRepository, ledgerRepo repository.LedgerRepository) ReconcileService {
	return &reconcileService{
		db:         db,
		walletRepo: walletRepo,
		ledgerRepo: ledgerRepo,
	}
}

// ReconcileUser 对账单个用户
func (s *reconcileService) ReconcileUser(ctx context.Context, userID uuid.UUID) (*ReconcileResult, error) {
	// 获取用户所有钱包
	wallets, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户钱包失败: %w", err)
	}

	// 获取用户所有账本条目
	ledgerEntries, err := s.ledgerRepo.GetByUserID(ctx, userID, 10000) // 获取大量记录
	if err != nil {
		return nil, fmt.Errorf("获取用户账本条目失败: %w", err)
	}

	// 按资产分组计算
	assetResults := make(map[string]*ReconcileResult)

	// 计算钱包总额
	for _, wallet := range wallets {
		if _, exists := assetResults[wallet.Asset]; !exists {
			assetResults[wallet.Asset] = &ReconcileResult{
				UserID:    userID,
				Asset:     wallet.Asset,
				Details:   make(map[string]decimal.Decimal),
				CheckedAt: time.Now(),
			}
		}

		result := assetResults[wallet.Asset]
		result.WalletTotal = wallet.AvailableDecimal.Add(wallet.LockedDecimal)
		result.Details["wallet_available"] = wallet.AvailableDecimal
		result.Details["wallet_locked"] = wallet.LockedDecimal
	}

	// 计算账本流水总额
	ledgerTotals := make(map[string]decimal.Decimal)
	for _, entry := range ledgerEntries {
		if _, exists := ledgerTotals[entry.Asset]; !exists {
			ledgerTotals[entry.Asset] = decimal.Zero
		}
		ledgerTotals[entry.Asset] = ledgerTotals[entry.Asset].Add(entry.Delta)
	}

	// 设置账本总额并计算差异
	for asset, result := range assetResults {
		if ledgerTotal, exists := ledgerTotals[asset]; exists {
			result.LedgerTotal = ledgerTotal
		} else {
			result.LedgerTotal = decimal.Zero
		}

		result.Discrepancy = result.WalletTotal.Sub(result.LedgerTotal)
		result.IsBalanced = result.Discrepancy.IsZero()
		result.Details["ledger_total"] = result.LedgerTotal

		if !result.IsBalanced {
			log.Warn().
				Str("user_id", userID.String()).
				Str("asset", asset).
				Str("wallet_total", result.WalletTotal.String()).
				Str("ledger_total", result.LedgerTotal.String()).
				Str("discrepancy", result.Discrepancy.String()).
				Msg("发现余额不一致")
		}
	}

	// 返回第一个资产的结果（简化处理）
	for _, result := range assetResults {
		return result, nil
	}

	return nil, fmt.Errorf("用户没有钱包记录")
}

// ReconcileAll 对账所有用户
func (s *reconcileService) ReconcileAll(ctx context.Context) ([]*ReconcileResult, error) {
	// 获取所有用户 ID
	query := `SELECT DISTINCT user_id FROM wallets`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("获取用户列表失败: %w", err)
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("扫描用户 ID 失败: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	// 对每个用户进行对账
	var results []*ReconcileResult
	for _, userID := range userIDs {
		result, err := s.ReconcileUser(ctx, userID)
		if err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID.String()).
				Msg("用户对账失败")
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// FixDiscrepancy 修复差异
func (s *reconcileService) FixDiscrepancy(ctx context.Context, userID uuid.UUID, asset string, expectedBalance decimal.Decimal) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// 锁定钱包记录
		wallet, err := s.walletRepo.LockForUpdate(ctx, tx, userID, asset)
		if err != nil {
			return fmt.Errorf("锁定钱包失败: %w", err)
		}

		currentTotal := wallet.AvailableDecimal.Add(wallet.LockedDecimal)
		discrepancy := expectedBalance.Sub(currentTotal)

		if discrepancy.IsZero() {
			return nil // 无需修复
		}

		// 调整可用余额
		_, err = s.walletRepo.UpdateBalancesWithTx(ctx, tx, userID, asset, discrepancy, decimal.Zero)
		if err != nil {
			return fmt.Errorf("调整钱包余额失败: %w", err)
		}

		// 创建调整账本条目
		entry := &types.LedgerEntry{
			ID:           uuid.New(),
			UserID:       userID,
			Asset:        asset,
			Delta:        discrepancy,
			BalanceAfter: wallet.AvailableDecimal.Add(discrepancy),
			Type:         types.LedgerEntryTypeCompensate,
			RefID:        uuid.New(), // 使用新的 UUID 作为调整参考
			CreatedAt:    time.Now(),
		}

		if err := s.ledgerRepo.CreateWithTx(ctx, tx, entry); err != nil {
			return fmt.Errorf("创建调整账本条目失败: %w", err)
		}

		log.Info().
			Str("user_id", userID.String()).
			Str("asset", asset).
			Str("discrepancy", discrepancy.String()).
			Str("expected_balance", expectedBalance.String()).
			Msg("余额差异已修复")

		return nil
	})
}

// PeriodicReconcile 定期对账
func (s *reconcileService) PeriodicReconcile(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("定期对账服务停止")
			return
		case <-ticker.C:
			log.Info().Msg("开始定期对账")

			results, err := s.ReconcileAll(ctx)
			if err != nil {
				log.Error().Err(err).Msg("定期对账失败")
				continue
			}

			var unbalancedCount int
			for _, result := range results {
				if !result.IsBalanced {
					unbalancedCount++
				}
			}

			log.Info().
				Int("total_users", len(results)).
				Int("unbalanced_users", unbalancedCount).
				Msg("定期对账完成")
		}
	}
}
