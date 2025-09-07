package handlers

import (
	"bytes"
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
	"github.com/stretchr/testify/mock"
)

// MockWalletService 钱包服务的 mock 实现
type MockWalletService struct {
	mock.Mock
}

func (m *MockWalletService) CreateWallet(ctx context.Context, req *types.CreateWalletRequest) (*types.CreateWalletResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.CreateWalletResponse), args.Error(1)
}

func (m *MockWalletService) GetWallet(ctx context.Context, userID, asset string) (*types.GetWalletResponse, error) {
	args := m.Called(ctx, userID, asset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.GetWalletResponse), args.Error(1)
}

func (m *MockWalletService) GetWallets(ctx context.Context, userID string) (*types.GetWalletsResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.GetWalletsResponse), args.Error(1)
}

func (m *MockWalletService) GetBalance(ctx context.Context, userID, asset string) (*types.GetBalanceResponse, error) {
	args := m.Called(ctx, userID, asset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.GetBalanceResponse), args.Error(1)
}

func (m *MockWalletService) GetBalances(ctx context.Context, userID string) (*types.GetBalancesResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.GetBalancesResponse), args.Error(1)
}

func (m *MockWalletService) Deposit(ctx context.Context, req *types.DepositRequest) (*types.DepositResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DepositResponse), args.Error(1)
}

func (m *MockWalletService) Withdraw(ctx context.Context, req *types.WithdrawRequest) (*types.WithdrawResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.WithdrawResponse), args.Error(1)
}

func (m *MockWalletService) WalletExists(ctx context.Context, userID, asset string) (bool, error) {
	args := m.Called(ctx, userID, asset)
	return args.Bool(0), args.Error(1)
}

func TestWalletsHandler_CreateWallet(t *testing.T) {
	userID := uuid.New().String()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockWalletService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "成功创建钱包",
			requestBody: types.CreateWalletRequest{
				UserID:         userID,
				Asset:          "SOL",
				IdempotencyKey: "test-key-123",
			},
			setupMocks: func(mockService *MockWalletService) {
				response := &types.CreateWalletResponse{
					WalletID:         uuid.New().String(),
					UserID:           userID,
					Asset:            "SOL",
					AvailableBalance: decimal.Zero,
					LockedBalance:    decimal.Zero,
					CreatedAt:        time.Now(),
				}
				mockService.On("CreateWallet", mock.Anything, mock.AnythingOfType("*types.CreateWalletRequest")).Return(response, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "用户不存在",
			requestBody: types.CreateWalletRequest{
				UserID:         userID,
				Asset:          "SOL",
				IdempotencyKey: "test-key-123",
			},
			setupMocks: func(mockService *MockWalletService) {
				mockService.On("CreateWallet", mock.Anything, mock.AnythingOfType("*types.CreateWalletRequest")).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "无效的请求体",
			requestBody:    "invalid json",
			setupMocks:     func(mockService *MockWalletService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
		{
			name: "验证失败",
			requestBody: types.CreateWalletRequest{
				UserID:         "", // 空用户 ID
				Asset:          "SOL",
				IdempotencyKey: "test-key-123",
			},
			setupMocks:     func(mockService *MockWalletService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockWalletService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewWalletsHandler(mockService)

			// 准备请求
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/wallets", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.CreateWallet(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if w.Code == http.StatusCreated {
				var response types.CreateWalletResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.WalletID)
				assert.NotEmpty(t, response.UserID)
				assert.NotEmpty(t, response.Asset)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}

func TestWalletsHandler_GetWallet(t *testing.T) {
	userID := uuid.New().String()
	walletID := uuid.New().String()

	tests := []struct {
		name           string
		userID         string
		asset          string
		setupMocks     func(*MockWalletService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "成功获取钱包",
			userID: userID,
			asset:  "SOL",
			setupMocks: func(mockService *MockWalletService) {
				response := &types.GetWalletResponse{
					WalletID:         walletID,
					UserID:           userID,
					Asset:            "SOL",
					AvailableBalance: decimal.NewFromFloat(10.5),
					LockedBalance:    decimal.NewFromFloat(1.5),
					UpdatedAt:        time.Now(),
				}
				mockService.On("GetWallet", mock.Anything, userID, "SOL").Return(response, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "钱包不存在",
			userID: userID,
			asset:  "SOL",
			setupMocks: func(mockService *MockWalletService) {
				mockService.On("GetWallet", mock.Anything, userID, "SOL").Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "用户 ID 为空",
			userID:         "",
			asset:          "SOL",
			setupMocks:     func(mockService *MockWalletService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
		{
			name:           "资产为空",
			userID:         userID,
			asset:          "",
			setupMocks:     func(mockService *MockWalletService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockWalletService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewWalletsHandler(mockService)

			// 准备请求
			req := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/"+tt.userID+"/"+tt.asset, nil)

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 设置路由参数
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("user_id", tt.userID)
			rctx.URLParams.Add("asset", tt.asset)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.GetWallet(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if w.Code == http.StatusOK {
				var response types.GetWalletResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, userID, response.UserID)
				assert.Equal(t, "SOL", response.Asset)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}

func TestWalletsHandler_GetWallets(t *testing.T) {
	userID := uuid.New().String()

	tests := []struct {
		name           string
		userID         string
		setupMocks     func(*MockWalletService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "成功获取用户所有钱包",
			userID: userID,
			setupMocks: func(mockService *MockWalletService) {
				response := &types.GetWalletsResponse{
					Wallets: []types.WalletInfo{
						{
							WalletID:         uuid.New().String(),
							Asset:            "SOL",
							AvailableBalance: "10.5",
							LockedBalance:    "1.5",
							UpdatedAt:        time.Now().Format(time.RFC3339),
						},
						{
							WalletID:         uuid.New().String(),
							Asset:            "USDC",
							AvailableBalance: "100.0",
							LockedBalance:    "0.0",
							UpdatedAt:        time.Now().Format(time.RFC3339),
						},
					},
				}
				mockService.On("GetWallets", mock.Anything, userID).Return(response, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "获取钱包失败",
			userID: userID,
			setupMocks: func(mockService *MockWalletService) {
				mockService.On("GetWallets", mock.Anything, userID).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "用户 ID 为空",
			userID:         "",
			setupMocks:     func(mockService *MockWalletService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockWalletService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewWalletsHandler(mockService)

			// 准备请求
			req := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/"+tt.userID, nil)

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 设置路由参数
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("user_id", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.GetWallets(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if w.Code == http.StatusOK {
				var response types.GetWalletsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Wallets)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}

func TestWalletsHandler_Deposit(t *testing.T) {
	userID := uuid.New().String()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockWalletService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "成功充值",
			requestBody: types.DepositRequest{
				UserID:         userID,
				Asset:          "SOL",
				Amount:         "5.0",
				IdempotencyKey: "test-key-123",
			},
			setupMocks: func(mockService *MockWalletService) {
				response := &types.DepositResponse{
					TransactionID:    uuid.New().String(),
					UserID:           userID,
					Asset:            "SOL",
					Amount:           decimal.NewFromFloat(5.0),
					AvailableBalance: decimal.NewFromFloat(15.0),
					LockedBalance:    decimal.Zero,
					CreatedAt:        time.Now(),
				}
				mockService.On("Deposit", mock.Anything, mock.AnythingOfType("*types.DepositRequest")).Return(response, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "用户不存在",
			requestBody: types.DepositRequest{
				UserID:         userID,
				Asset:          "SOL",
				Amount:         "5.0",
				IdempotencyKey: "test-key-123",
			},
			setupMocks: func(mockService *MockWalletService) {
				mockService.On("Deposit", mock.Anything, mock.AnythingOfType("*types.DepositRequest")).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "无效的请求体",
			requestBody:    "invalid json",
			setupMocks:     func(mockService *MockWalletService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
		{
			name: "验证失败",
			requestBody: types.DepositRequest{
				UserID:         "", // 空用户 ID
				Asset:          "SOL",
				Amount:         "5.0",
				IdempotencyKey: "test-key-123",
			},
			setupMocks:     func(mockService *MockWalletService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockWalletService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewWalletsHandler(mockService)

			// 准备请求
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/deposits", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.Deposit(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if w.Code == http.StatusCreated {
				var response types.DepositResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.TransactionID)
				assert.Equal(t, userID, response.UserID)
				assert.Equal(t, "SOL", response.Asset)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}

func TestWalletsHandler_Withdraw(t *testing.T) {
	userID := uuid.New().String()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockWalletService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "成功提现",
			requestBody: types.WithdrawRequest{
				UserID:         userID,
				Asset:          "SOL",
				Amount:         "2.0",
				ToAddress:      "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
				IdempotencyKey: "test-key-123",
			},
			setupMocks: func(mockService *MockWalletService) {
				response := &types.WithdrawResponse{
					TransactionID:    uuid.New().String(),
					UserID:           userID,
					Asset:            "SOL",
					Amount:           decimal.NewFromFloat(2.0),
					ToAddress:        "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
					AvailableBalance: decimal.NewFromFloat(8.0),
					LockedBalance:    decimal.Zero,
					Status:           "pending",
					CreatedAt:        time.Now(),
				}
				mockService.On("Withdraw", mock.Anything, mock.AnythingOfType("*types.WithdrawRequest")).Return(response, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "余额不足",
			requestBody: types.WithdrawRequest{
				UserID:         userID,
				Asset:          "SOL",
				Amount:         "20.0",
				ToAddress:      "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
				IdempotencyKey: "test-key-123",
			},
			setupMocks: func(mockService *MockWalletService) {
				mockService.On("Withdraw", mock.Anything, mock.AnythingOfType("*types.WithdrawRequest")).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "无效的请求体",
			requestBody:    "invalid json",
			setupMocks:     func(mockService *MockWalletService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
		{
			name: "验证失败",
			requestBody: types.WithdrawRequest{
				UserID:         "", // 空用户 ID
				Asset:          "SOL",
				Amount:         "2.0",
				ToAddress:      "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
				IdempotencyKey: "test-key-123",
			},
			setupMocks:     func(mockService *MockWalletService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockWalletService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewWalletsHandler(mockService)

			// 准备请求
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/withdrawals", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.Withdraw(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if w.Code == http.StatusCreated {
				var response types.WithdrawResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.TransactionID)
				assert.Equal(t, userID, response.UserID)
				assert.Equal(t, "SOL", response.Asset)
				assert.Equal(t, "pending", response.Status)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}

func TestWalletsHandler_GetBalance(t *testing.T) {
	userID := uuid.New().String()

	tests := []struct {
		name           string
		userID         string
		asset          string
		setupMocks     func(*MockWalletService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "成功获取余额",
			userID: userID,
			asset:  "SOL",
			setupMocks: func(mockService *MockWalletService) {
				response := &types.GetBalanceResponse{
					UserID:           userID,
					Asset:            "SOL",
					AvailableBalance: decimal.NewFromFloat(10.5),
					LockedBalance:    decimal.NewFromFloat(1.5),
					TotalBalance:     decimal.NewFromFloat(12.0),
					UpdatedAt:        time.Now(),
				}
				mockService.On("GetBalance", mock.Anything, userID, "SOL").Return(response, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "钱包不存在",
			userID: userID,
			asset:  "SOL",
			setupMocks: func(mockService *MockWalletService) {
				mockService.On("GetBalance", mock.Anything, userID, "SOL").Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "用户 ID 为空",
			userID:         "",
			asset:          "SOL",
			setupMocks:     func(mockService *MockWalletService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockWalletService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewWalletsHandler(mockService)

			// 准备请求
			req := httptest.NewRequest(http.MethodGet, "/api/v1/balances/"+tt.userID+"/"+tt.asset, nil)

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 设置路由参数
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("user_id", tt.userID)
			rctx.URLParams.Add("asset", tt.asset)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.GetBalance(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if w.Code == http.StatusOK {
				var response types.GetBalanceResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, userID, response.UserID)
				assert.Equal(t, "SOL", response.Asset)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}
