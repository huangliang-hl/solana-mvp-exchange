package handlers

import (
	"encoding/json"
	"net/http"

	"solana-limit-order-backend/internal/types"
	"solana-limit-order-backend/internal/wallet"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

// WalletsHandler 钱包处理器
type WalletsHandler struct {
	walletService wallet.Service
	validator     *validator.Validate
}

// NewWalletsHandler 创建钱包处理器
func NewWalletsHandler(walletService wallet.Service) *WalletsHandler {
	return &WalletsHandler{
		walletService: walletService,
		validator:     validator.New(),
	}
}

// CreateWallet 创建钱包
// POST /api/v1/wallets
func (h *WalletsHandler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 解析请求体
	var req types.CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "请求体格式错误", requestID)
		return
	}

	// 验证请求参数
	if err := h.validator.Struct(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "请求参数验证失败: "+err.Error(), requestID)
		return
	}

	// 调用服务创建钱包
	resp, err := h.walletService.CreateWallet(ctx, &req)
	if err != nil {
		if err.Error() == "钱包已存在" {
			writeErrorResponse(w, http.StatusConflict, "WALLET_EXISTS", err.Error(), requestID)
			return
		}
		if err.Error() == "用户不存在" {
			writeErrorResponse(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), requestID)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "创建钱包失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// GetWallet 获取钱包信息
// GET /api/v1/wallets/{user_id}/{asset}
func (h *WalletsHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 获取路径参数
	userID := chi.URLParam(r, "user_id")
	asset := chi.URLParam(r, "asset")

	if userID == "" || asset == "" {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "用户 ID 和资产类型不能为空", requestID)
		return
	}

	// 调用服务获取钱包
	resp, err := h.walletService.GetWallet(ctx, userID, asset)
	if err != nil {
		if err.Error() == "钱包不存在" {
			writeErrorResponse(w, http.StatusNotFound, "WALLET_NOT_FOUND", err.Error(), requestID)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "获取钱包失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// GetWallets 获取用户所有钱包
// GET /api/v1/wallets/{user_id}
func (h *WalletsHandler) GetWallets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 获取路径参数
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "用户 ID 不能为空", requestID)
		return
	}

	// 调用服务获取用户所有钱包
	resp, err := h.walletService.GetWallets(ctx, userID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "获取用户钱包失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// GetBalance 获取余额
// GET /api/v1/balances/{user_id}/{asset}
func (h *WalletsHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 获取路径参数
	userID := chi.URLParam(r, "user_id")
	asset := chi.URLParam(r, "asset")

	if userID == "" || asset == "" {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "用户 ID 和资产类型不能为空", requestID)
		return
	}

	// 调用服务获取余额
	resp, err := h.walletService.GetBalance(ctx, userID, asset)
	if err != nil {
		if err.Error() == "钱包不存在" {
			writeErrorResponse(w, http.StatusNotFound, "WALLET_NOT_FOUND", err.Error(), requestID)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "获取余额失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// GetBalances 获取所有余额
// GET /api/v1/balances/{user_id}
func (h *WalletsHandler) GetBalances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 获取路径参数
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "用户 ID 不能为空", requestID)
		return
	}

	// 调用服务获取所有余额
	resp, err := h.walletService.GetBalances(ctx, userID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "获取余额失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Deposit 充值
// POST /api/v1/deposits
func (h *WalletsHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 解析请求体
	var req types.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "请求体格式错误", requestID)
		return
	}

	// 验证请求参数
	if err := h.validator.Struct(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "请求参数验证失败: "+err.Error(), requestID)
		return
	}

	// 调用服务处理充值
	resp, err := h.walletService.Deposit(ctx, &req)
	if err != nil {
		if err.Error() == "用户不存在" {
			writeErrorResponse(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), requestID)
			return
		}
		if err.Error() == "无效的充值金额" || err.Error() == "充值金额必须大于零" {
			writeErrorResponse(w, http.StatusBadRequest, "INVALID_AMOUNT", err.Error(), requestID)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "充值失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Withdraw 提现
// POST /api/v1/withdrawals
func (h *WalletsHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 解析请求体
	var req types.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "请求体格式错误", requestID)
		return
	}

	// 验证请求参数
	if err := h.validator.Struct(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "请求参数验证失败: "+err.Error(), requestID)
		return
	}

	// 调用服务处理提现
	resp, err := h.walletService.Withdraw(ctx, &req)
	if err != nil {
		if err.Error() == "用户不存在" {
			writeErrorResponse(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), requestID)
			return
		}
		if err.Error() == "钱包不存在" {
			writeErrorResponse(w, http.StatusNotFound, "WALLET_NOT_FOUND", err.Error(), requestID)
			return
		}
		if err.Error() == "无效的提现金额" || err.Error() == "提现金额必须大于零" {
			writeErrorResponse(w, http.StatusBadRequest, "INVALID_AMOUNT", err.Error(), requestID)
			return
		}
		if err.Error() == "余额不足" {
			writeErrorResponse(w, http.StatusBadRequest, "INSUFFICIENT_BALANCE", err.Error(), requestID)
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "提现失败: "+err.Error(), requestID)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
