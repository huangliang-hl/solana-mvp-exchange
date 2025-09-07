package handlers

import (
	"encoding/json"
	"net/http"

	"solana-limit-order-backend/internal/types"

	"github.com/rs/zerolog/log"
)

// writeErrorResponse 写入错误响应
func writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := types.ErrorResponse{
		Error: types.ErrorDetail{
			Code:      errorCode,
			Message:   message,
			RequestID: requestID,
		},
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Msg("编码错误响应失败")
	}
}
