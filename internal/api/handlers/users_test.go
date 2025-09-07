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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserService 用户服务的 mock 实现
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, req *types.CreateUserRequest) (*types.CreateUserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.CreateUserResponse), args.Error(1)
}

func (m *MockUserService) GetUser(ctx context.Context, userID string) (*types.GetUserResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.GetUserResponse), args.Error(1)
}

func (m *MockUserService) GetUserByUsername(ctx context.Context, username string) (*types.GetUserResponse, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.GetUserResponse), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, userID string, req *types.UpdateUserRequest) (*types.UpdateUserResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.UpdateUserResponse), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) UserExists(ctx context.Context, userID string) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func TestUsersHandler_CreateUser(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "成功创建用户",
			requestBody: types.CreateUserRequest{
				Username:       "testuser",
				IdempotencyKey: "test-key-123",
			},
			setupMocks: func(mockService *MockUserService) {
				response := &types.CreateUserResponse{
					UserID:    uuid.New().String(),
					Username:  "testuser",
					CreatedAt: time.Now(),
				}
				mockService.On("CreateUser", mock.Anything, mock.AnythingOfType("*types.CreateUserRequest")).Return(response, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "用户名已存在",
			requestBody: types.CreateUserRequest{
				Username:       "existinguser",
				IdempotencyKey: "test-key-123",
			},
			setupMocks: func(mockService *MockUserService) {
				mockService.On("CreateUser", mock.Anything, mock.AnythingOfType("*types.CreateUserRequest")).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "无效的请求体",
			requestBody:    "invalid json",
			setupMocks:     func(mockService *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
		{
			name: "验证失败",
			requestBody: types.CreateUserRequest{
				Username:       "", // 空用户名
				IdempotencyKey: "test-key-123",
			},
			setupMocks:     func(mockService *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockUserService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewUsersHandler(mockService)

			// 准备请求
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.CreateUser(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if w.Code == http.StatusCreated {
				var response types.CreateUserResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.UserID)
				assert.NotEmpty(t, response.Username)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}

func TestUsersHandler_GetUser(t *testing.T) {
	userID := uuid.New().String()

	tests := []struct {
		name           string
		userID         string
		setupMocks     func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "成功获取用户",
			userID: userID,
			setupMocks: func(mockService *MockUserService) {
				response := &types.GetUserResponse{
					UserID:    userID,
					Username:  "testuser",
					CreatedAt: time.Now(),
				}
				mockService.On("GetUser", mock.Anything, userID).Return(response, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "用户不存在",
			userID: userID,
			setupMocks: func(mockService *MockUserService) {
				mockService.On("GetUser", mock.Anything, userID).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "用户 ID 为空",
			userID:         "",
			setupMocks:     func(mockService *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockUserService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewUsersHandler(mockService)

			// 准备请求
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+tt.userID, nil)

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 设置路由参数
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.GetUser(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if w.Code == http.StatusOK {
				var response types.GetUserResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, userID, response.UserID)
				assert.NotEmpty(t, response.Username)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}

func TestUsersHandler_GetUserByUsername(t *testing.T) {
	username := "testuser"

	tests := []struct {
		name           string
		username       string
		setupMocks     func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "成功根据用户名获取用户",
			username: username,
			setupMocks: func(mockService *MockUserService) {
				response := &types.GetUserResponse{
					UserID:    uuid.New().String(),
					Username:  username,
					CreatedAt: time.Now(),
				}
				mockService.On("GetUserByUsername", mock.Anything, username).Return(response, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "用户不存在",
			username: username,
			setupMocks: func(mockService *MockUserService) {
				mockService.On("GetUserByUsername", mock.Anything, username).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "用户名为空",
			username:       "",
			setupMocks:     func(mockService *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockUserService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewUsersHandler(mockService)

			// 准备请求
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/username/"+tt.username, nil)

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 设置路由参数
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("username", tt.username)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.GetUserByUsername(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if w.Code == http.StatusOK {
				var response types.GetUserResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, username, response.Username)
				assert.NotEmpty(t, response.UserID)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}

func TestUsersHandler_UpdateUser(t *testing.T) {
	userID := uuid.New().String()

	tests := []struct {
		name           string
		userID         string
		requestBody    interface{}
		setupMocks     func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "成功更新用户",
			userID: userID,
			requestBody: types.UpdateUserRequest{
				Username: "newusername",
			},
			setupMocks: func(mockService *MockUserService) {
				response := &types.UpdateUserResponse{
					UserID:    userID,
					Username:  "newusername",
					UpdatedAt: time.Now(),
				}
				mockService.On("UpdateUser", mock.Anything, userID, mock.AnythingOfType("*types.UpdateUserRequest")).Return(response, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "用户不存在",
			userID: userID,
			requestBody: types.UpdateUserRequest{
				Username: "newusername",
			},
			setupMocks: func(mockService *MockUserService) {
				mockService.On("UpdateUser", mock.Anything, userID, mock.AnythingOfType("*types.UpdateUserRequest")).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "用户 ID 为空",
			userID:         "",
			requestBody:    types.UpdateUserRequest{Username: "newusername"},
			setupMocks:     func(mockService *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
		{
			name:           "无效的请求体",
			userID:         userID,
			requestBody:    "invalid json",
			setupMocks:     func(mockService *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
		{
			name:   "验证失败",
			userID: userID,
			requestBody: types.UpdateUserRequest{
				Username: "", // 空用户名
			},
			setupMocks:     func(mockService *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockUserService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewUsersHandler(mockService)

			// 准备请求
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+tt.userID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 设置路由参数
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.UpdateUser(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if w.Code == http.StatusOK {
				var response types.UpdateUserResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, userID, response.UserID)
				assert.NotEmpty(t, response.Username)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}

func TestUsersHandler_DeleteUser(t *testing.T) {
	userID := uuid.New().String()

	tests := []struct {
		name           string
		userID         string
		setupMocks     func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "成功删除用户",
			userID: userID,
			setupMocks: func(mockService *MockUserService) {
				mockService.On("DeleteUser", mock.Anything, userID).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "用户不存在",
			userID: userID,
			setupMocks: func(mockService *MockUserService) {
				mockService.On("DeleteUser", mock.Anything, userID).Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "用户 ID 为空",
			userID:         "",
			setupMocks:     func(mockService *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 服务
			mockService := new(MockUserService)
			tt.setupMocks(mockService)

			// 创建处理器
			handler := NewUsersHandler(mockService)

			// 准备请求
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+tt.userID, nil)

			// 添加请求 ID 到上下文
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id")
			req = req.WithContext(ctx)

			// 设置路由参数
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			handler.DeleteUser(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			// 验证 mock 调用
			mockService.AssertExpectations(t)
		})
	}
}
