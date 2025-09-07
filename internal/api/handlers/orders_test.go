package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"solana-limit-order-backend/internal/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrderService 模拟订单服务
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) CreateOrder(ctx context.Context, req *types.CreateOrderRequest) (*types.CreateOrderResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.CreateOrderResponse), args.Error(1)
}

func (m *MockOrderService) CancelOrder(ctx context.Context, orderID, userID uuid.UUID) (*types.CancelOrderResponse, error) {
	args := m.Called(ctx, orderID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.CancelOrderResponse), args.Error(1)
}

func (m *MockOrderService) GetOrders(ctx context.Context, userID uuid.UUID, status string, limit int) (*types.OrderQueryResponse, error) {
	args := m.Called(ctx, userID, status, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrderQueryResponse), args.Error(1)
}

func (m *MockOrderService) GetOrderByID(ctx context.Context, orderID uuid.UUID) (*types.Order, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Order), args.Error(1)
}

func (m *MockOrderService) ValidateOrderRequest(req *types.CreateOrderRequest) error {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func TestCreateOrder_Success(t *testing.T) {
	// 创建模拟服务
	mockService := new(MockOrderService)
	handler := &OrdersHandler{
		orderService: mockService,
	}

	// 准备测试数据
	request := types.CreateOrderRequest{
		UserID:              "test-user-123",
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

	expectedResponse := &types.CreateOrderResponse{
		OrderID: "order-123",
		Status:  "open",
		AvailableBalances: map[string]string{
			"SOL":  "5.0",
			"USDC": "1000.0",
		},
	}

	// 设置模拟期望
	mockService.On("CreateOrder", mock.Anything, &request).Return(expectedResponse, nil)

	// 创建HTTP请求
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", request.IdempotencyKey)

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用处理器
	handler.CreateOrder(w, req)

	// 验证响应
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response types.CreateOrderResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse.OrderID, response.OrderID)
	assert.Equal(t, expectedResponse.Status, response.Status)

	// 验证模拟调用
	mockService.AssertExpectations(t)
}

func TestCreateOrder_InvalidJSON(t *testing.T) {
	mockService := new(MockOrderService)
	handler := &OrdersHandler{
		orderService: mockService,
	}

	// 创建无效JSON请求
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.CreateOrder(w, req)

	// 验证错误响应
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp types.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST_BODY", errorResp.Error.Code)
}

func TestCreateOrder_ServiceError(t *testing.T) {
	mockService := new(MockOrderService)
	handler := &OrdersHandler{
		orderService: mockService,
	}

	// 设置服务返回错误
	request := types.CreateOrderRequest{
		UserID:       "user-123",
		Base:         "SOL",
		Quote:        "USDC",
		Size:         "1.5",
		Side:         "sell",
		TriggerPrice: "30.5",
		TriggerOp:    "lte",
	}

	mockService.On("CreateOrder", mock.Anything, &request).Return(nil, assert.AnError)

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.CreateOrder(w, req)

	// 验证错误响应
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResp types.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Equal(t, "ORDER_CREATION_FAILED", errorResp.Error.Code)
}

func TestCreateOrderJSONParsing(t *testing.T) {
	// 测试CreateOrder handler是否能正确解析JSON
	mockService := new(MockOrderService)
	handler := &OrdersHandler{
		orderService: mockService,
	}

	// 设置成功的模拟响应
	mockService.On("ValidateOrderRequest", mock.Anything).Return(nil)
	mockService.On("CreateOrder", mock.Anything, mock.Anything).Return(&types.CreateOrderResponse{
		OrderID: "test-order-123",
		Status:  "open",
	}, nil)

	// 测试有效的JSON请求
	validRequest := types.CreateOrderRequest{
		UserID:       "user-123",
		Base:         "SOL",
		Quote:        "USDC",
		Size:         "1.5",
		Side:         "sell",
		TriggerPrice: "30.5",
		TriggerOp:    "lte",
	}

	body, _ := json.Marshal(validRequest)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.CreateOrder(w, req)

	// 验证响应
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// 验证响应内容
	var response types.CreateOrderResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-order-123", response.OrderID)
}
