package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"solana-limit-order-backend/internal/api"
	"solana-limit-order-backend/internal/ledger"
	"solana-limit-order-backend/internal/order"
	"solana-limit-order-backend/internal/price"
	"solana-limit-order-backend/internal/queue"
	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/testutil"
	"solana-limit-order-backend/internal/trigger"
	"solana-limit-order-backend/internal/types"
	"solana-limit-order-backend/internal/user"
	"solana-limit-order-backend/internal/wallet"
)

// shouldSkipDatabaseTests 检查是否应该跳过数据库测试
func shouldSkipDatabaseTests() bool {
	// 在 Docker 环境中不跳过（Docker 会设置这些环境变量）
	if os.Getenv("DB_URL") != "" && os.Getenv("REDIS_URL") != "" {
		return false
	}

	// 在 short 模式下跳过
	return true
}

// ExecutorServiceInterface 执行器服务接口
type ExecutorServiceInterface interface {
	StartWorker(ctx context.Context, workerID int) error
	RecoverPendingOrders(ctx context.Context) error
	ExecuteOrder(ctx context.Context, msg *types.ExecuteOrderMessage) error
}

// IntegrationTestSuite 集成测试套件
type IntegrationTestSuite struct {
	TestCtx       *testutil.TestContext
	OrderService  order.Service
	LedgerService ledger.Service
	PriceService  price.ServiceInterface
	TriggerEngine *trigger.Engine
	ExecutorSvc   ExecutorServiceInterface
	APIRouter     *api.Router
	MockJupiter   *testutil.MockJupiterClient
	MockSolana    *testutil.MockSolanaClient
}

// setupIntegrationTest 设置集成测试
func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	// 设置测试环境
	testCtx := testutil.SetupTestEnv(t)

	// 初始化队列和消费者组
	queueManager := queue.NewQueueManager(testCtx.Queue)
	err := queueManager.InitializeStreams(context.Background())
	require.NoError(t, err)

	// 创建存储库
	userRepo := repository.NewUserRepository(testCtx.DB)
	walletRepo := repository.NewWalletRepository(testCtx.DB)
	orderRepo := repository.NewOrderRepository(testCtx.DB)
	ledgerRepo := repository.NewLedgerRepository(testCtx.DB)

	// 创建服务
	ledgerService := ledger.NewService(testCtx.DB, walletRepo, ledgerRepo)
	orderService := order.NewService(orderRepo, userRepo, ledgerService)
	priceService := price.NewService(testCtx.DB, testCtx.Queue)

	// 创建模拟客户端
	mockJupiter := &testutil.MockJupiterClient{}
	mockSolana := &testutil.MockSolanaClient{}

	// 创建模拟执行器服务（避免真实的外部API调用）
	executorSvc := testutil.NewMockExecutorService(
		testCtx.Queue,
		orderRepo,
		ledgerService,
		mockJupiter,
		mockSolana,
	)

	// 创建用户和钱包服务
	userService := user.NewService(testCtx.DB, userRepo)
	walletService := wallet.NewService(testCtx.DB, walletRepo, userRepo, ledgerService)

	// 创建触发引擎
	triggerEngine := trigger.NewEngine(testCtx.Queue, orderRepo)

	// 创建 API 路由
	apiRouter := api.NewRouter(testCtx.Config, testCtx.DB, orderService, userService, walletService)

	return &IntegrationTestSuite{
		TestCtx:       testCtx,
		OrderService:  orderService,
		LedgerService: ledgerService,
		PriceService:  priceService,
		TriggerEngine: triggerEngine,
		ExecutorSvc:   executorSvc,
		APIRouter:     apiRouter,
		MockJupiter:   mockJupiter,
		MockSolana:    mockSolana,
	}
}

// TestAPIIntegration API 集成测试
func TestAPIIntegration(t *testing.T) {
	if testing.Short() && shouldSkipDatabaseTests() {
		t.Skip("跳过集成测试（需要数据库）")
	}

	suite := setupIntegrationTest(t)
	defer suite.TestCtx.Cleanup()

	// 创建测试用户和钱包
	userID := testutil.CreateTestUser(t, suite.TestCtx.DB, "api_test_user")
	testutil.CreateTestWallet(t, suite.TestCtx.DB, userID, "SOL", 100.0, 0.0)
	testutil.CreateTestWallet(t, suite.TestCtx.DB, userID, "USDC", 10000.0, 0.0)

	t.Run("创建订单成功", func(t *testing.T) {
		orderReq := types.CreateOrderRequest{
			UserID:              userID,
			Base:                "SOL",
			Quote:               "USDC",
			Size:                "1.5",
			Side:                "sell",
			TriggerPrice:        "30.5",
			TriggerOp:           "lte",
			MaxSlippagePct:      "0.5",
			PriorityFeeLamports: 5000,
			IdempotencyKey:      uuid.New().String(),
		}

		reqBody, _ := json.Marshal(orderReq)
		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Api-Key", "test-key-1")
		req.Header.Set("Idempotency-Key", orderReq.IdempotencyKey)

		w := httptest.NewRecorder()
		suite.APIRouter.SetupRoutes().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp types.CreateOrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEmpty(t, resp.OrderID)
		assert.Equal(t, "open", resp.Status)
		assert.Contains(t, resp.AvailableBalances, "SOL")
		assert.Contains(t, resp.AvailableBalances, "USDC")

		// 验证余额已锁定
		available, locked := testutil.GetWalletBalance(t, suite.TestCtx.DB, userID, "SOL")
		assert.Equal(t, 98.5, available) // 100 - 1.5
		assert.Equal(t, 1.5, locked)
	})

	t.Run("查询订单", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/orders?user_id=%s", userID), nil)
		req.Header.Set("X-Api-Key", "test-key-1")

		w := httptest.NewRecorder()
		suite.APIRouter.SetupRoutes().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp types.QueryOrdersResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Len(t, resp.Orders, 1)
		assert.Equal(t, "open", resp.Orders[0].Status)
	})

	t.Run("撤销订单", func(t *testing.T) {
		// 先创建一个订单
		orderID := testutil.CreateTestOrder(t, suite.TestCtx.DB, userID, "SOL", "USDC", "sell", "lte", 1.0, 25.0, "open")

		// 撤销订单（添加user_id查询参数）
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/orders/%s?user_id=%s", orderID, userID), nil)
		req.Header.Set("X-Api-Key", "test-key-1")

		w := httptest.NewRecorder()
		suite.APIRouter.SetupRoutes().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp types.CancelOrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, orderID, resp.OrderID)
		assert.Equal(t, "canceled", resp.Status)

		// 验证订单状态
		status := testutil.GetOrderStatus(t, suite.TestCtx.DB, orderID)
		assert.Equal(t, "canceled", status)
	})
}

// TestTriggerEngineIntegration 触发引擎集成测试
func TestTriggerEngineIntegration(t *testing.T) {
	if testing.Short() && shouldSkipDatabaseTests() {
		t.Skip("跳过集成测试（需要数据库）")
	}

	suite := setupIntegrationTest(t)
	defer suite.TestCtx.Cleanup()

	// 创建测试用户和钱包
	userID := testutil.CreateTestUser(t, suite.TestCtx.DB, "trigger_test_user")
	testutil.CreateTestWallet(t, suite.TestCtx.DB, userID, "SOL", 100.0, 0.0)

	t.Run("价格触发订单", func(t *testing.T) {
		// 创建触发条件：SOL <= 30.0
		orderID := testutil.CreateTestOrder(t, suite.TestCtx.DB, userID, "SOL", "USDC", "sell", "lte", 1.0, 30.0, "open")

		// 模拟价格 tick
		priceTick := types.PriceTick{
			SequenceID: 1,
			Timestamp:  time.Now().UnixMilli(),
			Symbol:     "SOL/USDC",
			BidPrice:   decimal.RequireFromString("29.5"), // 低于触发价格
			AskPrice:   decimal.RequireFromString("29.7"),
			MidPrice:   decimal.RequireFromString("29.6"),
			Source:     "test",
		}

		// 启动触发引擎
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		go suite.TriggerEngine.Start(ctx)

		// 发送价格事件
		suite.PriceService.Broadcast(&priceTick)

		// 等待订单被触发
		testutil.AssertEventuallyEqual(t, "triggered",
			func() interface{} { return testutil.GetOrderStatus(t, suite.TestCtx.DB, orderID) },
			3*time.Second, "订单应该被触发")
	})

	t.Run("价格不满足触发条件", func(t *testing.T) {
		// 创建触发条件：SOL >= 35.0
		orderID := testutil.CreateTestOrder(t, suite.TestCtx.DB, userID, "SOL", "USDC", "buy", "gte", 1.0, 35.0, "open")

		// 模拟价格 tick（不满足条件）
		priceTick := types.PriceTick{
			SequenceID: 2,
			Timestamp:  time.Now().UnixMilli(),
			Symbol:     "SOL/USDC",
			BidPrice:   decimal.RequireFromString("30.0"), // 低于触发价格
			AskPrice:   decimal.RequireFromString("30.2"),
			MidPrice:   decimal.RequireFromString("30.1"),
			Source:     "test",
		}

		// 发送价格事件
		suite.PriceService.Broadcast(&priceTick)

		// 等待一段时间，确认订单没有被触发
		time.Sleep(1 * time.Second)
		status := testutil.GetOrderStatus(t, suite.TestCtx.DB, orderID)
		assert.Equal(t, "open", status, "订单不应该被触发")
	})
}

// TestExecutorIntegration 执行器集成测试
func TestExecutorIntegration(t *testing.T) {
	if testing.Short() && shouldSkipDatabaseTests() {
		t.Skip("跳过集成测试（需要数据库）")
	}

	suite := setupIntegrationTest(t)
	defer suite.TestCtx.Cleanup()

	// 创建测试用户和钱包
	userID := testutil.CreateTestUser(t, suite.TestCtx.DB, "executor_test_user")
	testutil.CreateTestWallet(t, suite.TestCtx.DB, userID, "SOL", 100.0, 0.0)
	testutil.CreateTestWallet(t, suite.TestCtx.DB, userID, "USDC", 10000.0, 0.0)

	t.Run("执行器处理订单成功", func(t *testing.T) {
		// 设置模拟响应
		suite.MockJupiter.QuoteResponse = &testutil.JupiterQuoteResponse{
			InputMint:   "SOL",
			InAmount:    "1000000000", // 1 SOL in lamports
			OutputMint:  "USDC",
			OutAmount:   "30000000", // 30 USDC in micro units
			SlippageBps: 50,
		}
		suite.MockSolana.TransactionSignature = "mock_tx_signature_success"

		// 创建已触发的订单（模拟资金锁定）
		orderID := testutil.CreateTestOrderWithLock(t, suite.TestCtx.DB, userID, "SOL", "USDC", "sell", "lte", 1.0, 30.0, "triggered")

		// 将执行任务加入队列
		task := types.ExecuteOrderMessage{
			OrderID:             orderID,
			UserID:              userID,
			MaxSlippagePct:      "0.5",
			PriorityFeeLamports: 5000,
			IdempotencyKey:      uuid.New().String(),
			Timestamp:           time.Now().UnixMilli(),
		}

		err := suite.TestCtx.Queue.Publish(context.Background(), "order.execute", task)
		require.NoError(t, err)

		// 启动执行器
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go suite.ExecutorSvc.StartWorker(ctx, 1)

		// 等待订单被执行
		testutil.AssertEventuallyEqual(t, "filled",
			func() interface{} { return testutil.GetOrderStatus(t, suite.TestCtx.DB, orderID) },
			5*time.Second, "订单应该被执行成功")
	})
}

// TestEndToEndFlow 端到端测试
func TestEndToEndFlow(t *testing.T) {
	if testing.Short() && shouldSkipDatabaseTests() {
		t.Skip("跳过集成测试（需要数据库）")
	}

	suite := setupIntegrationTest(t)
	defer suite.TestCtx.Cleanup()

	// 创建测试用户和钱包
	userID := testutil.CreateTestUser(t, suite.TestCtx.DB, "e2e_test_user")
	testutil.CreateTestWallet(t, suite.TestCtx.DB, userID, "SOL", 100.0, 0.0)
	testutil.CreateTestWallet(t, suite.TestCtx.DB, userID, "USDC", 10000.0, 0.0)

	// 设置模拟响应
	suite.MockJupiter.QuoteResponse = &testutil.JupiterQuoteResponse{
		InputMint:   "SOL",
		InAmount:    "1500000000", // 1.5 SOL in lamports
		OutputMint:  "USDC",
		OutAmount:   "45000000", // 45 USDC in micro units
		SlippageBps: 50,
	}
	suite.MockSolana.TransactionSignature = "e2e_test_signature"

	t.Run("完整交易流程", func(t *testing.T) {
		// 1. 创建订单
		orderReq := types.CreateOrderRequest{
			UserID:              userID,
			Base:                "SOL",
			Quote:               "USDC",
			Size:                "1.5",
			Side:                "sell",
			TriggerPrice:        "30.0",
			TriggerOp:           "lte",
			MaxSlippagePct:      "0.5",
			PriorityFeeLamports: 5000,
			IdempotencyKey:      uuid.New().String(),
		}

		reqBody, _ := json.Marshal(orderReq)
		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Api-Key", "test-key-1")
		req.Header.Set("Idempotency-Key", orderReq.IdempotencyKey)

		w := httptest.NewRecorder()
		suite.APIRouter.SetupRoutes().ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var createResp types.CreateOrderResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResp)
		require.NoError(t, err)

		orderID := createResp.OrderID

		// 2. 验证订单创建和资金锁定
		assert.Equal(t, "open", createResp.Status)
		available, locked := testutil.GetWalletBalance(t, suite.TestCtx.DB, userID, "SOL")
		assert.Equal(t, 98.5, available) // 100 - 1.5
		assert.Equal(t, 1.5, locked)

		// 3. 启动所有服务
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		go suite.TriggerEngine.Start(ctx)
		go suite.ExecutorSvc.StartWorker(ctx, 1)

		// 4. 发送触发价格
		priceTick := types.PriceTick{
			SequenceID: 100,
			Timestamp:  time.Now().UnixMilli(),
			Symbol:     "SOL/USDC",
			BidPrice:   decimal.RequireFromString("29.5"), // 低于触发价格
			AskPrice:   decimal.RequireFromString("29.7"),
			MidPrice:   decimal.RequireFromString("29.6"),
			Source:     "e2e_test",
		}

		suite.PriceService.Broadcast(&priceTick)

		// 5. 等待订单被触发和执行（由于触发引擎和执行器都在运行，订单可能直接变为filled）
		var finalStatus string
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			status := testutil.GetOrderStatus(t, suite.TestCtx.DB, orderID)
			if status == "triggered" || status == "filled" {
				finalStatus = status
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		// 验证订单被触发或已执行完成
		assert.Contains(t, []string{"triggered", "filled"}, finalStatus, "订单应该被触发或执行")

		// 6. 如果还是triggered状态，等待执行完成
		if finalStatus == "triggered" {
			testutil.AssertEventuallyEqual(t, "filled",
				func() interface{} { return testutil.GetOrderStatus(t, suite.TestCtx.DB, orderID) },
				10*time.Second, "订单应该被执行成功")
		}

		// 7. 验证最终状态
		// 验证订单状态
		finalOrderStatus := testutil.GetOrderStatus(t, suite.TestCtx.DB, orderID)
		assert.Equal(t, "filled", finalOrderStatus)

		// 验证资金状态（这里简化验证，实际应该根据 Jupiter 响应计算）
		finalAvailable, finalLocked := testutil.GetWalletBalance(t, suite.TestCtx.DB, userID, "SOL")
		assert.Equal(t, 98.5, finalAvailable) // 资金已释放或转换
		assert.Equal(t, 0.0, finalLocked)     // 锁定资金释放

		// 8. 查询最终订单状态
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/orders?user_id=%s", userID), nil)
		req.Header.Set("X-Api-Key", "test-key-1")

		w = httptest.NewRecorder()
		suite.APIRouter.SetupRoutes().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var queryResp types.QueryOrdersResponse
		err = json.Unmarshal(w.Body.Bytes(), &queryResp)
		require.NoError(t, err)

		// 找到我们的订单
		var finalOrder *types.OrderInfo
		for i := range queryResp.Orders {
			if queryResp.Orders[i].OrderID == orderID {
				finalOrder = &queryResp.Orders[i]
				break
			}
		}

		require.NotNil(t, finalOrder, "应该能找到订单")
		assert.Equal(t, "filled", finalOrder.Status)
		assert.NotNil(t, finalOrder.TriggeredAt)
	})
}

// TestMain 控制整个测试包的执行，并提供有用的日志说明
func TestMain(m *testing.M) {
	fmt.Println()
	fmt.Println("=== 开始运行Solana限价单后端集成测试 ===")
	fmt.Println("注意：测试过程中可能会看到以下预期的错误日志：")
	fmt.Println("1. 'context canceled' - 测试结束时上下文被取消导致的正常清理日志")
	fmt.Println("2. 'NOGROUP No such key' - Redis流和消费者组清理过程中的正常日志")
	fmt.Println("3. '消费消息失败' - 后台服务关闭时的正常错误日志")
	fmt.Println("这些错误不影响测试结果，是系统正常清理过程的一部分。")
	fmt.Println("=================================================")
	fmt.Println()

	// 运行测试
	code := m.Run()

	fmt.Println()
	fmt.Println("=== 集成测试执行完成 ===")
	if code == 0 {
		fmt.Println("✅ 所有测试通过！")
		fmt.Println("🎉 Solana限价单后端系统集成测试成功完成")
		fmt.Println()
		fmt.Println("测试验证了以下核心功能：")
		fmt.Println("• API接口：订单创建、查询、取消")
		fmt.Println("• 触发引擎：价格监控和订单触发")
		fmt.Println("• 执行器：订单执行和状态管理")
		fmt.Println("• 端到端流程：完整的交易生命周期")
		fmt.Println("• 模拟客户端：Jupiter和Solana API模拟")
		fmt.Println("• 数据结构和配置验证")
		fmt.Println()
		fmt.Println("💡 关于ERROR日志的说明：")
		fmt.Println("上述看到的ERROR日志（如'context canceled'、'NOGROUP'等）")
		fmt.Println("是测试框架正常清理资源时产生的预期日志，不代表测试失败。")
		fmt.Println("这是分布式系统测试中的正常现象。")
	} else {
		fmt.Println("❌ 部分测试失败，请检查错误信息")
	}
	fmt.Println("========================")
	fmt.Println()

	os.Exit(code)
}
