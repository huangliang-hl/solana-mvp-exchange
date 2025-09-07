package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"solana-limit-order-backend/internal/order"
	"solana-limit-order-backend/internal/types"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

// OrdersHandler 订单处理器
type OrdersHandler struct {
	orderService order.Service
}

// NewOrdersHandler 创建订单处理器
func NewOrdersHandler(orderService order.Service) *OrdersHandler {
	return &OrdersHandler{
		orderService: orderService,
	}
}

// CreateOrder 创建订单
func (h *OrdersHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 解析请求体
	var req types.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Msg("解析创建订单请求失败")

		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求体格式错误", requestID)
		return
	}

	// 记录请求日志
	log.Info().
		Str("request_id", requestID).
		Str("user_id", req.UserID).
		Str("base", req.Base).
		Str("quote", req.Quote).
		Str("size", req.Size).
		Str("side", req.Side).
		Str("trigger_price", req.TriggerPrice).
		Str("trigger_op", req.TriggerOp).
		Str("idempotency_key", req.IdempotencyKey).
		Msg("收到创建订单请求")

	// 创建订单
	resp, err := h.orderService.CreateOrder(ctx, &req)
	if err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("user_id", req.UserID).
			Msg("创建订单失败")

		// 根据错误类型返回不同的状态码
		statusCode := http.StatusInternalServerError
		errorCode := "ORDER_CREATION_FAILED"

		if isValidationError(err) {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		} else if isInsufficientBalanceError(err) {
			statusCode = http.StatusBadRequest
			errorCode = "INSUFFICIENT_BALANCE"
		}

		writeErrorResponse(w, statusCode, errorCode, err.Error(), requestID)
		return
	}

	// 记录成功日志
	log.Info().
		Str("request_id", requestID).
		Str("user_id", req.UserID).
		Str("order_id", resp.OrderID).
		Str("status", resp.Status).
		Msg("订单创建成功")

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Msg("编码响应失败")
	}
}

// isValidationError 检查是否为验证错误
func isValidationError(err error) bool {
	errorMsg := err.Error()
	return strings.Contains(errorMsg, "验证失败") ||
		strings.Contains(errorMsg, "无效") ||
		strings.Contains(errorMsg, "格式错误")
}

// isInsufficientBalanceError 检查是否为余额不足错误
func isInsufficientBalanceError(err error) bool {
	errorMsg := err.Error()
	return strings.Contains(errorMsg, "余额不足") ||
		strings.Contains(errorMsg, "insufficient")
}
