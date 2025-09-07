package middleware

import (
	"encoding/json"
	"net/http"

	"solana-limit-order-backend/internal/types"

	"github.com/go-chi/chi/v5/middleware"
)

// writeErrorResponse 写入错误响应
func writeErrorResponse(w http.ResponseWriter, statusCode int, code, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := types.ErrorResponse{
		Error: types.ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		},
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		// 如果JSON编码失败，记录错误但不再尝试写入响应
		// 因为可能会导致进一步的写入问题
		// 这里有意留空，因为已经无法向客户端发送更多响应
		_ = err
	}
}

// getRequestID 获取请求 ID
func getRequestID(r *http.Request) string {
	requestID := middleware.GetReqID(r.Context())
	if requestID == "" {
		requestID = "unknown"
	}
	return requestID
}
