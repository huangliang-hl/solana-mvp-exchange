package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// CancelOrder 取消订单
func (h *OrdersHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 获取订单 ID
	orderIDStr := chi.URLParam(r, "id")
	if orderIDStr == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_ORDER_ID", "缺少订单 ID", requestID)
		return
	}

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("order_id", orderIDStr).
			Msg("无效的订单 ID")

		writeErrorResponse(w, http.StatusBadRequest, "INVALID_ORDER_ID", "无效的订单 ID", requestID)
		return
	}

	// 获取用户 ID（从查询参数或请求体）
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_USER_ID", "缺少用户 ID", requestID)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("user_id", userIDStr).
			Msg("无效的用户 ID")

		writeErrorResponse(w, http.StatusBadRequest, "INVALID_USER_ID", "无效的用户 ID", requestID)
		return
	}

	// 记录请求日志
	log.Info().
		Str("request_id", requestID).
		Str("order_id", orderID.String()).
		Str("user_id", userID.String()).
		Msg("收到取消订单请求")

	// 取消订单
	resp, err := h.orderService.CancelOrder(ctx, orderID, userID)
	if err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("order_id", orderID.String()).
			Str("user_id", userID.String()).
			Msg("取消订单失败")

		// 根据错误类型返回不同的状态码
		statusCode := http.StatusInternalServerError
		errorCode := "ORDER_CANCELLATION_FAILED"

		if isNotFoundError(err) {
			statusCode = http.StatusNotFound
			errorCode = "ORDER_NOT_FOUND"
		} else if isPermissionError(err) {
			statusCode = http.StatusForbidden
			errorCode = "PERMISSION_DENIED"
		} else if isInvalidStatusError(err) {
			statusCode = http.StatusConflict
			errorCode = "INVALID_ORDER_STATUS"
		}

		writeErrorResponse(w, statusCode, errorCode, err.Error(), requestID)
		return
	}

	// 记录成功日志
	log.Info().
		Str("request_id", requestID).
		Str("order_id", orderID.String()).
		Str("user_id", userID.String()).
		Str("status", resp.Status).
		Msg("订单取消成功")

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Msg("编码响应失败")
	}
}

// isNotFoundError 检查是否为未找到错误
func isNotFoundError(err error) bool {
	errorMsg := err.Error()
	return strings.Contains(errorMsg, "不存在") || strings.Contains(errorMsg, "未找到")
}

// isPermissionError 检查是否为权限错误
func isPermissionError(err error) bool {
	return strings.Contains(err.Error(), "无权限")
}

// isInvalidStatusError 检查是否为无效状态错误
func isInvalidStatusError(err error) bool {
	errorMsg := err.Error()
	return strings.Contains(errorMsg, "状态不允许") || strings.Contains(errorMsg, "不能取消")
}
