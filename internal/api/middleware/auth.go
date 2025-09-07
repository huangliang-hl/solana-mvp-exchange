package middleware

import (
	"net/http"

	"solana-limit-order-backend/internal/config"
)

// Auth 认证中间件
func Auth(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 获取 API Key
			apiKey := r.Header.Get("X-Api-Key")
			if apiKey == "" {
				// 尝试从查询参数获取（用于 WebSocket）
				apiKey = r.URL.Query().Get("api_key")
			}

			if apiKey == "" {
				writeErrorResponse(w, http.StatusUnauthorized, "MISSING_API_KEY", "缺少 API 密钥", getRequestID(r))
				return
			}

			// 验证 API Key
			if !cfg.IsValidAPIKey(apiKey) {
				writeErrorResponse(w, http.StatusUnauthorized, "INVALID_API_KEY", "无效的 API 密钥", getRequestID(r))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth 可选认证中间件（用于健康检查等公开接口）
func OptionalAuth(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 检查路径是否需要认证
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// 对于非公开路径，执行认证
			Auth(cfg)(next).ServeHTTP(w, r)
		})
	}
}

// isPublicPath 检查是否为公开路径
func isPublicPath(path string) bool {
	publicPaths := []string{
		"/health",
		"/metrics",
		"/",
	}

	for _, publicPath := range publicPaths {
		if path == publicPath {
			return true
		}
	}

	return false
}
