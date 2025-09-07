package handlers

import (
	"encoding/json"
	"net/http"

	"solana-limit-order-backend/internal/types"
	"solana-limit-order-backend/internal/user"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

// UsersHandler 用户处理器
type UsersHandler struct {
	userService user.Service
	validator   *validator.Validate
}

// NewUsersHandler 创建用户处理器
func NewUsersHandler(userService user.Service) *UsersHandler {
	return &UsersHandler{
		userService: userService,
		validator:   validator.New(),
	}
}

// CreateUser 创建用户
// POST /api/v1/users
func (h *UsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 解析请求体
	var req types.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "请求体格式错误", requestID)
		return
	}

	// 验证请求参数
	if err := h.validator.Struct(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "请求参数验证失败: "+err.Error(), requestID)
		return
	}

	// 调用服务创建用户
	resp, err := h.userService.CreateUser(ctx, &req)
	if err != nil {
		if err.Error() == "用户名已存在" {
			writeErrorResponse(w, http.StatusConflict, "USER_EXISTS", err.Error(), requestID)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "创建用户失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// GetUser 获取用户信息
// GET /api/v1/users/{id}
func (h *UsersHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 获取路径参数
	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "用户 ID 不能为空", requestID)
		return
	}

	// 调用服务获取用户
	resp, err := h.userService.GetUser(ctx, userID)
	if err != nil {
		if err.Error() == "用户不存在" {
			writeErrorResponse(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), requestID)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "获取用户失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// GetUserByUsername 根据用户名获取用户信息
// GET /api/v1/users/username/{username}
func (h *UsersHandler) GetUserByUsername(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 获取路径参数
	username := chi.URLParam(r, "username")
	if username == "" {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "用户名不能为空", requestID)
		return
	}

	// 调用服务获取用户
	resp, err := h.userService.GetUserByUsername(ctx, username)
	if err != nil {
		if err.Error() == "用户不存在" {
			writeErrorResponse(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), requestID)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "获取用户失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// UpdateUser 更新用户信息
// PUT /api/v1/users/{id}
func (h *UsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 获取路径参数
	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "用户 ID 不能为空", requestID)
		return
	}

	// 解析请求体
	var req types.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "请求体格式错误", requestID)
		return
	}

	// 验证请求参数
	if err := h.validator.Struct(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "请求参数验证失败: "+err.Error(), requestID)
		return
	}

	// 调用服务更新用户
	resp, err := h.userService.UpdateUser(ctx, userID, &req)
	if err != nil {
		if err.Error() == "用户不存在" {
			writeErrorResponse(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), requestID)
			return
		}
		if err.Error() == "用户名已存在" {
			writeErrorResponse(w, http.StatusConflict, "USER_EXISTS", err.Error(), requestID)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "更新用户失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// DeleteUser 删除用户
// DELETE /api/v1/users/{id}
func (h *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 获取路径参数
	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "用户 ID 不能为空", requestID)
		return
	}

	// 调用服务删除用户
	err := h.userService.DeleteUser(ctx, userID)
	if err != nil {
		if err.Error() == "用户不存在" {
			writeErrorResponse(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), requestID)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "删除用户失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}
