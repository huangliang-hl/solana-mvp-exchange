package integration

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"solana-limit-order-backend/internal/testutil"
)

// shouldSkipDatabaseTests 检查是否应该跳过数据库测试
func shouldSkipDatabaseTestsSimple() bool {
	// 在 Docker 环境中不跳过（Docker 会设置这些环境变量）
	if os.Getenv("DB_URL") != "" && os.Getenv("REDIS_URL") != "" {
		return false
	}

	// 在 short 模式下跳过
	return true
}

// TestBasicSetup 测试基本设置
func TestBasicSetup(t *testing.T) {
	if testing.Short() && shouldSkipDatabaseTestsSimple() {
		t.Skip("跳过集成测试")
	}

	// 设置测试环境
	testCtx := testutil.SetupTestEnv(t)
	defer testCtx.Cleanup()

	// 验证数据库连接
	ctx := context.Background()
	err := testCtx.DB.Health(ctx)
	require.NoError(t, err, "数据库应该正常连接")

	// 验证Redis连接
	err = testCtx.Redis.Ping(ctx).Err()
	require.NoError(t, err, "Redis应该正常连接")

	// 创建测试用户
	userID := testutil.CreateTestUser(t, testCtx.DB, "test_user_basic")
	assert.NotEmpty(t, userID, "应该成功创建用户")

	// 创建测试钱包
	testutil.CreateTestWallet(t, testCtx.DB, userID, "SOL", 100.0, 0.0)
	testutil.CreateTestWallet(t, testCtx.DB, userID, "USDC", 10000.0, 0.0)

	// 验证钱包余额
	available, locked := testutil.GetWalletBalance(t, testCtx.DB, userID, "SOL")
	assert.Equal(t, 100.0, available)
	assert.Equal(t, 0.0, locked)

	available, locked = testutil.GetWalletBalance(t, testCtx.DB, userID, "USDC")
	assert.Equal(t, 10000.0, available)
	assert.Equal(t, 0.0, locked)
}
