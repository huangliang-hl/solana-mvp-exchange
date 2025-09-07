package testutil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"solana-limit-order-backend/internal/types"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestUser 测试用户数据
type TestUser struct {
	ID       uuid.UUID
	Username string
}

// TestWallet 测试钱包数据
type TestWallet struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Asset     string
	Available decimal.Decimal
	Locked    decimal.Decimal
}

// NewTestUser 创建测试用户
func NewTestUser(username string) *TestUser {
	return &TestUser{
		ID:       uuid.New(),
		Username: username,
	}
}

// NewTestWallet 创建测试钱包
func NewTestWallet(userID uuid.UUID, asset string, available, locked decimal.Decimal) *TestWallet {
	return &TestWallet{
		ID:        uuid.New(),
		UserID:    userID,
		Asset:     asset,
		Available: available,
		Locked:    locked,
	}
}

// ToTypesUser 转换为 types.User
func (tu *TestUser) ToTypesUser() *types.User {
	return &types.User{
		ID:        tu.ID,
		Username:  tu.Username,
		CreatedAt: time.Now(),
	}
}

// ToTypesWallet 转换为 types.Wallet
func (tw *TestWallet) ToTypesWallet() *types.Wallet {
	return &types.Wallet{
		ID:               tw.ID,
		UserID:           tw.UserID,
		Asset:            tw.Asset,
		AvailableDecimal: tw.Available,
		LockedDecimal:    tw.Locked,
		UpdatedAt:        time.Now(),
	}
}

// CreateTestRequest 创建测试请求
func CreateTestRequest(method, url string, body interface{}) *http.Request {
	var req *http.Request

	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest(method, url, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Body = httptest.NewRequest("POST", "/", nil).Body
		req.ContentLength = int64(len(bodyBytes))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}

	// 添加请求 ID
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
	req = req.WithContext(ctx)

	return req
}

// AddRouteParams 添加路由参数
func AddRouteParams(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

// AssertJSONResponse 验证 JSON 响应
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	assert.Equal(t, expectedStatus, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	if target != nil {
		err := json.Unmarshal(w.Body.Bytes(), target)
		assert.NoError(t, err)
	}
}

// AssertErrorResponse 验证错误响应
func AssertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedCode string) {
	assert.Equal(t, expectedStatus, w.Code)

	var errorResp types.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Equal(t, expectedCode, errorResp.Error.Code)
	assert.NotEmpty(t, errorResp.Error.Message)
	assert.NotEmpty(t, errorResp.Error.RequestID)
}

// GenerateUUID 生成测试用的 UUID
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateIdempotencyKey 生成幂等性键
func GenerateIdempotencyKey() string {
	return "test-key-" + uuid.New().String()
}

// DecimalFromFloat 从浮点数创建 decimal
func DecimalFromFloat(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}

// DecimalFromString 从字符串创建 decimal
func DecimalFromString(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

// TimeNow 获取当前时间
func TimeNow() time.Time {
	return time.Now().UTC()
}

// TimeFromString 从字符串解析时间
func TimeFromString(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// AssertDecimalEqual 验证 decimal 相等
func AssertDecimalEqual(t *testing.T, expected, actual decimal.Decimal, msgAndArgs ...interface{}) {
	assert.True(t, expected.Equal(actual), msgAndArgs...)
}

// AssertDecimalGreaterThan 验证 decimal 大于
func AssertDecimalGreaterThan(t *testing.T, expected, actual decimal.Decimal, msgAndArgs ...interface{}) {
	assert.True(t, actual.GreaterThan(expected), msgAndArgs...)
}

// AssertDecimalLessThan 验证 decimal 小于
func AssertDecimalLessThan(t *testing.T, expected, actual decimal.Decimal, msgAndArgs ...interface{}) {
	assert.True(t, actual.LessThan(expected), msgAndArgs...)
}

// CreateTestCreateUserRequest 创建测试用的创建用户请求
func CreateTestCreateUserRequest(username string) *types.CreateUserRequest {
	return &types.CreateUserRequest{
		Username:       username,
		IdempotencyKey: GenerateIdempotencyKey(),
	}
}

// CreateTestCreateWalletRequest 创建测试用的创建钱包请求
func CreateTestCreateWalletRequest(userID, asset string) *types.CreateWalletRequest {
	return &types.CreateWalletRequest{
		UserID:         userID,
		Asset:          asset,
		IdempotencyKey: GenerateIdempotencyKey(),
	}
}

// CreateTestDepositRequest 创建测试用的充值请求
func CreateTestDepositRequest(userID, asset, amount string) *types.DepositRequest {
	return &types.DepositRequest{
		UserID:         userID,
		Asset:          asset,
		Amount:         amount,
		IdempotencyKey: GenerateIdempotencyKey(),
	}
}

// CreateTestWithdrawRequest 创建测试用的提现请求
func CreateTestWithdrawRequest(userID, asset, amount, toAddress string) *types.WithdrawRequest {
	return &types.WithdrawRequest{
		UserID:         userID,
		Asset:          asset,
		Amount:         amount,
		ToAddress:      toAddress,
		IdempotencyKey: GenerateIdempotencyKey(),
	}
}

// CreateTestUpdateUserRequest 创建测试用的更新用户请求
func CreateTestUpdateUserRequest(username string) *types.UpdateUserRequest {
	return &types.UpdateUserRequest{
		Username: username,
	}
}

// TestConstants 测试常量
var TestConstants = struct {
	TestUserID      string
	TestWalletID    string
	TestUsername    string
	TestAssetSOL    string
	TestAssetUSDC   string
	TestAddress     string
	TestAmount      string
	TestBigAmount   string
	TestSmallAmount string
}{
	TestUserID:      "550e8400-e29b-41d4-a716-446655440000",
	TestWalletID:    "660e8400-e29b-41d4-a716-446655440000",
	TestUsername:    "testuser",
	TestAssetSOL:    "SOL",
	TestAssetUSDC:   "USDC",
	TestAddress:     "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
	TestAmount:      "10.5",
	TestBigAmount:   "1000.0",
	TestSmallAmount: "0.1",
}

// MockResponseBuilder 响应构建器
type MockResponseBuilder struct{}

// NewMockResponseBuilder 创建响应构建器
func NewMockResponseBuilder() *MockResponseBuilder {
	return &MockResponseBuilder{}
}

// CreateUserResponse 创建用户响应
func (b *MockResponseBuilder) CreateUserResponse(userID, username string) *types.CreateUserResponse {
	return &types.CreateUserResponse{
		UserID:    userID,
		Username:  username,
		CreatedAt: TimeNow(),
	}
}

// GetUserResponse 获取用户响应
func (b *MockResponseBuilder) GetUserResponse(userID, username string) *types.GetUserResponse {
	return &types.GetUserResponse{
		UserID:    userID,
		Username:  username,
		CreatedAt: TimeNow(),
	}
}

// CreateWalletResponse 创建钱包响应
func (b *MockResponseBuilder) CreateWalletResponse(walletID, userID, asset string) *types.CreateWalletResponse {
	return &types.CreateWalletResponse{
		WalletID:         walletID,
		UserID:           userID,
		Asset:            asset,
		AvailableBalance: decimal.Zero,
		LockedBalance:    decimal.Zero,
		CreatedAt:        TimeNow(),
	}
}

// GetWalletResponse 获取钱包响应
func (b *MockResponseBuilder) GetWalletResponse(walletID, userID, asset string, available, locked decimal.Decimal) *types.GetWalletResponse {
	return &types.GetWalletResponse{
		WalletID:         walletID,
		UserID:           userID,
		Asset:            asset,
		AvailableBalance: available,
		LockedBalance:    locked,
		UpdatedAt:        TimeNow(),
	}
}

// DepositResponse 充值响应
func (b *MockResponseBuilder) DepositResponse(userID, asset string, amount, available, locked decimal.Decimal) *types.DepositResponse {
	return &types.DepositResponse{
		TransactionID:    GenerateUUID(),
		UserID:           userID,
		Asset:            asset,
		Amount:           amount,
		AvailableBalance: available,
		LockedBalance:    locked,
		CreatedAt:        TimeNow(),
	}
}

// WithdrawResponse 提现响应
func (b *MockResponseBuilder) WithdrawResponse(userID, asset, toAddress string, amount, available, locked decimal.Decimal) *types.WithdrawResponse {
	return &types.WithdrawResponse{
		TransactionID:    GenerateUUID(),
		UserID:           userID,
		Asset:            asset,
		Amount:           amount,
		ToAddress:        toAddress,
		AvailableBalance: available,
		LockedBalance:    locked,
		Status:           "pending",
		CreatedAt:        TimeNow(),
	}
}
