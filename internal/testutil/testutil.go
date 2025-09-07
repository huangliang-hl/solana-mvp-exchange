package testutil

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"solana-limit-order-backend/internal/config"
	"solana-limit-order-backend/internal/queue"
	"solana-limit-order-backend/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// TestContext 测试上下文，包含所有测试所需的依赖
type TestContext struct {
	DB         *repository.DB
	Redis      *redis.Client
	Queue      *queue.RedisQueue
	Config     *config.Config
	TestDB     *pgxpool.Pool
	Cleanup    func()
	TestServer *httptest.Server
	Router     chi.Router
}

// SetupTestEnv 设置测试环境
func SetupTestEnv(t *testing.T) *TestContext {
	// 设置日志级别为错误，减少测试输出
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// 加载测试配置
	cfg, err := loadTestConfig()
	require.NoError(t, err)

	// 创建测试数据库连接
	testDB, err := setupTestDB(cfg.DatabaseURL)
	require.NoError(t, err)

	// 创建 DB 包装器
	db := &repository.DB{Pool: testDB}

	// 创建 Redis 客户端
	redisClient, err := setupTestRedis(cfg.RedisURL)
	require.NoError(t, err)

	// 创建队列
	queueClient, err := queue.NewRedisClient(cfg.RedisURL)
	require.NoError(t, err)

	// 清理函数
	cleanup := func() {
		// 清理数据库
		cleanupTestDB(t, testDB)
		testDB.Close()

		// 清理 Redis
		cleanupTestRedis(t, redisClient)
		redisClient.Close()
	}

	return &TestContext{
		DB:      db,
		Redis:   redisClient,
		Queue:   queueClient,
		Config:  cfg,
		TestDB:  testDB,
		Cleanup: cleanup,
	}
}

// loadTestConfig 加载测试配置
func loadTestConfig() (*config.Config, error) {
	// 默认测试环境变量（如果未设置则使用这些值）
	defaultTestEnvs := map[string]string{
		"DB_URL":           "postgres://postgres:password@localhost:5432/postgres?sslmode=disable", // 使用默认postgres数据库
		"REDIS_URL":        "redis://localhost:6379/15",                                            // 使用数据库15进行测试
		"RPC_URL":          "https://api.devnet.solana.com",
		"JUPITER_ENDPOINT": "https://lite-api.jup.ag/swap/v1/",
		"API_BIND":         "0.0.0.0:0", // 随机端口
		"PRICE_WS_BIND":    "0.0.0.0:0", // 随机端口
		"MODE":             "test",
		"LOG_LEVEL":        "error",
		"API_KEYS":         "test-key-1,test-key-2",
		"PRIVATE_KEY":      "test_private_key_for_development_only",
	}

	// 只有在环境变量未设置时才使用默认值
	for key, defaultValue := range defaultTestEnvs {
		if os.Getenv(key) == "" {
			os.Setenv(key, defaultValue)
		}
	}

	return config.Load()
}

// setupTestDB 设置测试数据库
func setupTestDB(databaseURL string) (*pgxpool.Pool, error) {
	// 创建测试数据库连接
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 5
	config.MinConns = 1
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 5

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	// 确保测试数据库干净
	if err := resetTestDB(pool); err != nil {
		return nil, err
	}

	return pool, nil
}

// setupTestRedis 设置测试 Redis
func setupTestRedis(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	// 清空测试数据库
	if err := client.FlushDB(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

// resetTestDB 重置测试数据库
func resetTestDB(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// 清理所有表的数据
	tables := []string{
		"idempotency_keys",
		"price_ticks",
		"tx_attempts",
		"ledger_entries",
		"orders",
		"wallets",
		"users",
	}

	for _, table := range tables {
		if _, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			// 忽略表不存在的错误，这在测试环境中是正常的
			if !isTableNotExistError(err) {
				return fmt.Errorf("清理表 %s 失败: %w", table, err)
			}
		}
	}

	return nil
}

// cleanupTestDB 清理测试数据库
func cleanupTestDB(t *testing.T, pool *pgxpool.Pool) {
	if err := resetTestDB(pool); err != nil {
		t.Logf("清理测试数据库失败: %v", err)
	}
}

// cleanupTestRedis 清理测试 Redis
func cleanupTestRedis(t *testing.T, client *redis.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Logf("清理测试 Redis 失败: %v", err)
	}
}

// CreateTestUser 创建测试用户
func CreateTestUser(t *testing.T, db *repository.DB, username string) string {
	ctx := context.Background()

	var userID string
	err := db.QueryRow(ctx,
		"INSERT INTO users (username) VALUES ($1) RETURNING id",
		username,
	).Scan(&userID)
	require.NoError(t, err)

	return userID
}

// CreateTestWallet 创建测试钱包
func CreateTestWallet(t *testing.T, db *repository.DB, userID, asset string, available, locked float64) {
	ctx := context.Background()

	_, err := db.Pool.Exec(ctx,
		"INSERT INTO wallets (user_id, asset, available_decimal, locked_decimal) VALUES ($1, $2, $3, $4)",
		userID, asset, available, locked,
	)
	require.NoError(t, err)
}

// CreateTestOrder 创建测试订单
func CreateTestOrder(t *testing.T, db *repository.DB, userID, base, quote, side, triggerOp string,
	size, triggerPrice float64, status string) string {
	ctx := context.Background()

	var orderID string
	err := db.QueryRow(ctx,
		`INSERT INTO orders (user_id, base, quote, size, side, trigger_price, trigger_op, status, version) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0) RETURNING id`,
		userID, base, quote, size, side, triggerPrice, triggerOp, status,
	).Scan(&orderID)
	require.NoError(t, err)

	return orderID
}

// CreateTestOrderWithLock 创建测试订单并锁定资金
func CreateTestOrderWithLock(t *testing.T, db *repository.DB, userID, base, quote, side, triggerOp string,
	size, triggerPrice float64, status string) string {
	ctx := context.Background()

	// 创建订单
	orderID := CreateTestOrder(t, db, userID, base, quote, side, triggerOp, size, triggerPrice, status)

	// 如果是非取消状态的订单，需要锁定资金（模拟实际的订单创建逻辑）
	if status != "canceled" {
		var lockAsset string
		var lockAmount float64

		if side == "buy" {
			lockAsset = quote
			lockAmount = size * triggerPrice
		} else {
			lockAsset = base
			lockAmount = size
		}

		// 更新钱包余额来模拟资金锁定
		_, err := db.Pool.Exec(ctx,
			`UPDATE wallets SET 
				available_decimal = available_decimal - $1,
				locked_decimal = locked_decimal + $1
			WHERE user_id = $2 AND asset = $3`,
			lockAmount, userID, lockAsset,
		)
		require.NoError(t, err)
	}

	return orderID
}

// WaitForCondition 等待条件满足
func WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("等待条件超时")
}

// GetOrderStatus 获取订单状态
func GetOrderStatus(t *testing.T, db *repository.DB, orderID string) string {
	ctx := context.Background()

	var status string
	err := db.QueryRow(ctx, "SELECT status FROM orders WHERE id = $1", orderID).Scan(&status)
	require.NoError(t, err)

	return status
}

// Balance 余额结构
type Balance struct {
	Available decimal.Decimal
	Locked    decimal.Decimal
}

// GetUserBalances 获取用户余额
func GetUserBalances(t *testing.T, db *repository.DB, userID string) map[string]Balance {
	ctx := context.Background()

	rows, err := db.Pool.Query(ctx,
		"SELECT asset, available_decimal, locked_decimal FROM wallets WHERE user_id = $1",
		userID)
	require.NoError(t, err)
	defer rows.Close()

	balances := make(map[string]Balance)
	for rows.Next() {
		var asset string
		var available, locked decimal.Decimal
		err := rows.Scan(&asset, &available, &locked)
		require.NoError(t, err)

		balances[asset] = Balance{
			Available: available,
			Locked:    locked,
		}
	}

	return balances
}

// GetWalletBalance 获取钱包余额
func GetWalletBalance(t *testing.T, db *repository.DB, userID, asset string) (available, locked float64) {
	ctx := context.Background()

	err := db.QueryRow(ctx,
		"SELECT available_decimal, locked_decimal FROM wallets WHERE user_id = $1 AND asset = $2",
		userID, asset,
	).Scan(&available, &locked)
	require.NoError(t, err)

	return available, locked
}

// MockJupiterClient Jupiter API 模拟客户端
type MockJupiterClient struct {
	QuoteResponse  *JupiterQuoteResponse
	SwapResponse   *JupiterSwapResponse
	ShouldFail     bool
	FailureMessage string
}

// JupiterClientInterface Jupiter 客户端接口
type JupiterClientInterface interface {
	GetQuote(ctx context.Context, req *JupiterQuoteRequest) (*JupiterQuoteResponse, error)
	GetSwapTransaction(ctx context.Context, req *JupiterSwapRequest) (*JupiterSwapResponse, error)
}

// JupiterQuoteRequest Jupiter 报价请求
type JupiterQuoteRequest struct {
	InputMint   string `json:"inputMint"`
	OutputMint  string `json:"outputMint"`
	Amount      string `json:"amount"`
	SlippageBps int    `json:"slippageBps"`
}

// JupiterSwapRequest Jupiter 交换请求
type JupiterSwapRequest struct {
	QuoteResponse *JupiterQuoteResponse `json:"quoteResponse"`
	UserPublicKey string                `json:"userPublicKey"`
}

// JupiterQuoteResponse Jupiter 报价响应
type JupiterQuoteResponse struct {
	InputMint   string `json:"inputMint"`
	InAmount    string `json:"inAmount"`
	OutputMint  string `json:"outputMint"`
	OutAmount   string `json:"outAmount"`
	OtherAmount string `json:"otherAmount"`
	SwapMode    string `json:"swapMode"`
	SlippageBps int    `json:"slippageBps"`
}

// JupiterSwapResponse Jupiter 交换响应
type JupiterSwapResponse struct {
	SwapTransaction string `json:"swapTransaction"`
}

// GetQuote 获取报价（模拟）
func (m *MockJupiterClient) GetQuote(ctx context.Context, req *JupiterQuoteRequest) (*JupiterQuoteResponse, error) {
	if m.ShouldFail {
		return nil, fmt.Errorf(m.FailureMessage)
	}

	if m.QuoteResponse != nil {
		return m.QuoteResponse, nil
	}

	// 默认响应
	return &JupiterQuoteResponse{
		InputMint:   req.InputMint,
		InAmount:    req.Amount,
		OutputMint:  req.OutputMint,
		OutAmount:   req.Amount, // 简化，1:1 兑换
		OtherAmount: req.Amount,
		SwapMode:    "ExactIn",
		SlippageBps: req.SlippageBps,
	}, nil
}

// GetSwapTransaction 获取交换交易（模拟）
func (m *MockJupiterClient) GetSwapTransaction(ctx context.Context, req *JupiterSwapRequest) (*JupiterSwapResponse, error) {
	if m.ShouldFail {
		return nil, fmt.Errorf(m.FailureMessage)
	}

	if m.SwapResponse != nil {
		return m.SwapResponse, nil
	}

	// 默认响应
	return &JupiterSwapResponse{
		SwapTransaction: "mock_transaction_base64_encoded_data",
	}, nil
}

// MockSolanaClient Solana RPC 模拟客户端
type MockSolanaClient struct {
	TransactionSignature string
	ShouldFail           bool
	FailureMessage       string
	ConfirmationStatus   string
}

// SendTransaction 发送交易（模拟）
func (m *MockSolanaClient) SendTransaction(transaction string) (string, error) {
	if m.ShouldFail {
		return "", fmt.Errorf(m.FailureMessage)
	}

	if m.TransactionSignature != "" {
		return m.TransactionSignature, nil
	}

	// 默认响应
	return "mock_transaction_signature_" + fmt.Sprintf("%d", time.Now().Unix()), nil
}

// ConfirmTransaction 确认交易（模拟）
func (m *MockSolanaClient) ConfirmTransaction(signature string) (string, error) {
	if m.ShouldFail {
		return "", fmt.Errorf(m.FailureMessage)
	}

	if m.ConfirmationStatus != "" {
		return m.ConfirmationStatus, nil
	}

	// 默认响应
	return "finalized", nil
}

// AssertEventuallyEqual 断言最终相等（重试版本）
func AssertEventuallyEqual(t *testing.T, expected interface{}, actual func() interface{}, timeout time.Duration, msgAndArgs ...interface{}) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if expected == actual() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.Equal(t, expected, actual(), msgAndArgs...)
}

// AssertEventuallyTrue 断言最终为真
func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, condition(), msgAndArgs...)
}

// isTableNotExistError 检查是否为表不存在错误
func isTableNotExistError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "does not exist") && strings.Contains(errStr, "relation")
}
