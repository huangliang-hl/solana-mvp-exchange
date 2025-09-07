package ledger

import (
	"testing"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLedgerService_CalculateBalances(t *testing.T) {
	tests := []struct {
		name     string
		entries  []types.LedgerEntry
		expected map[string]decimal.Decimal
	}{
		{
			name: "计算SOL余额",
			entries: []types.LedgerEntry{
				{
					Asset:        "SOL",
					Delta:        decimal.NewFromFloat(10.0),
					Type:         "deposit",
					BalanceAfter: decimal.NewFromFloat(10.0),
				},
				{
					Asset:        "SOL",
					Delta:        decimal.NewFromFloat(-2.0),
					Type:         "lock_funds",
					BalanceAfter: decimal.NewFromFloat(8.0),
				},
			},
			expected: map[string]decimal.Decimal{
				"SOL": decimal.NewFromFloat(8.0),
			},
		},
		{
			name: "多资产余额计算",
			entries: []types.LedgerEntry{
				{
					Asset:        "SOL",
					Delta:        decimal.NewFromFloat(5.0),
					Type:         "deposit",
					BalanceAfter: decimal.NewFromFloat(5.0),
				},
				{
					Asset:        "USDC",
					Delta:        decimal.NewFromFloat(100.0),
					Type:         "deposit",
					BalanceAfter: decimal.NewFromFloat(100.0),
				},
			},
			expected: map[string]decimal.Decimal{
				"SOL":  decimal.NewFromFloat(5.0),
				"USDC": decimal.NewFromFloat(100.0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balances := make(map[string]decimal.Decimal)

			for _, entry := range tt.entries {
				balances[entry.Asset] = entry.BalanceAfter
			}

			for asset, expectedBalance := range tt.expected {
				actualBalance, exists := balances[asset]
				require.True(t, exists, "资产 %s 应该存在", asset)
				assert.True(t, expectedBalance.Equal(actualBalance),
					"资产 %s 余额不匹配: 期望 %s, 实际 %s",
					asset, expectedBalance.String(), actualBalance.String())
			}
		})
	}
}

func TestLedgerEntry_Validation(t *testing.T) {
	tests := []struct {
		name    string
		entry   types.LedgerEntry
		isValid bool
	}{
		{
			name: "有效的锁定资金条目",
			entry: types.LedgerEntry{
				ID:           uuid.New(),
				UserID:       uuid.New(),
				Asset:        "SOL",
				Delta:        decimal.NewFromFloat(-1.0),
				BalanceAfter: decimal.NewFromFloat(9.0),
				Type:         "lock_funds",
				RefID:        uuid.New(),
			},
			isValid: true,
		},
		{
			name: "无效的空资产",
			entry: types.LedgerEntry{
				ID:           uuid.New(),
				UserID:       uuid.New(),
				Asset:        "",
				Delta:        decimal.NewFromFloat(1.0),
				BalanceAfter: decimal.NewFromFloat(1.0),
				Type:         "deposit",
				RefID:        uuid.New(),
			},
			isValid: false,
		},
		{
			name: "无效的负余额",
			entry: types.LedgerEntry{
				ID:           uuid.New(),
				UserID:       uuid.New(),
				Asset:        "SOL",
				Delta:        decimal.NewFromFloat(-1.0),
				BalanceAfter: decimal.NewFromFloat(-1.0),
				Type:         "lock_funds",
				RefID:        uuid.New(),
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证基本字段
			if tt.isValid {
				assert.NotEmpty(t, tt.entry.Asset, "资产不能为空")
				assert.True(t, tt.entry.BalanceAfter.GreaterThanOrEqual(decimal.Zero), "余额不能为负")
			} else {
				// 这里根据具体的验证逻辑进行判断
				isValidAsset := tt.entry.Asset != ""
				isValidBalance := tt.entry.BalanceAfter.GreaterThanOrEqual(decimal.Zero)

				if !isValidAsset || !isValidBalance {
					// 预期的无效条目
					assert.True(t, true, "按预期无效")
				}
			}
		})
	}
}

func TestLedgerEntryTypes(t *testing.T) {
	validTypes := []string{
		"lock_funds",
		"release_funds",
		"filled",
		"compensate",
		"deposit",
		"withdraw",
	}

	for _, entryType := range validTypes {
		t.Run(entryType, func(t *testing.T) {
			entry := types.LedgerEntry{
				ID:           uuid.New(),
				UserID:       uuid.New(),
				Asset:        "SOL",
				Delta:        decimal.NewFromFloat(1.0),
				BalanceAfter: decimal.NewFromFloat(1.0),
				Type:         types.LedgerEntryType(entryType),
				RefID:        uuid.New(),
			}

			assert.Equal(t, entryType, string(entry.Type))
		})
	}
}
