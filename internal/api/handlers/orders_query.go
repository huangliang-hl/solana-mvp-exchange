package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// GetOrders 查询订单
func (h *OrdersHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetReqID(ctx)

	// 获取查询参数
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

	// 获取可选参数
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")

	limit := 50 // 默认限制
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 100 {
				limit = 100 // 最大限制
			}
		}
	}

	// 记录请求日志
	log.Info().
		Str("request_id", requestID).
		Str("user_id", userID.String()).
		Str("status", status).
		Int("limit", limit).
		Msg("收到查询订单请求")

	// 查询订单
	resp, err := h.orderService.GetOrders(ctx, userID, status, limit)
	if err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("user_id", userID.String()).
			Msg("查询订单失败")

		writeErrorResponse(w, http.StatusInternalServerError, "ORDER_QUERY_FAILED", err.Error(), requestID)
		return
	}

	// 记录成功日志
	log.Info().
		Str("request_id", requestID).
		Str("user_id", userID.String()).
		Int("order_count", len(resp.Orders)).
		Msg("订单查询成功")

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

// GetOrderByID 根据 ID 获取订单详情
func (h *OrdersHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
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

	// 记录请求日志
	log.Info().
		Str("request_id", requestID).
		Str("order_id", orderID.String()).
		Msg("收到获取订单详情请求")

	// 获取订单
	order, err := h.orderService.GetOrderByID(ctx, orderID)
	if err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("order_id", orderID.String()).
			Msg("获取订单详情失败")

		statusCode := http.StatusInternalServerError
		errorCode := "ORDER_FETCH_FAILED"

		if isNotFoundError(err) {
			statusCode = http.StatusNotFound
			errorCode = "ORDER_NOT_FOUND"
		}

		writeErrorResponse(w, statusCode, errorCode, err.Error(), requestID)
		return
	}

	// 记录成功日志
	log.Info().
		Str("request_id", requestID).
		Str("order_id", orderID.String()).
		Str("user_id", order.UserID.String()).
		Str("status", string(order.Status)).
		Msg("获取订单详情成功")

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Msg("编码响应失败")
	}
}
