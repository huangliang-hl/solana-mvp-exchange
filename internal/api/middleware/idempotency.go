package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"solana-limit-order-backend/internal/repository"
)

// IdempotencyKeyHeader 幂等性键头部名称
const IdempotencyKeyHeader = "Idempotency-Key"

// Idempotency 幂等性中间件
func Idempotency(db *repository.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 只对 POST 请求应用幂等性检查
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			// 获取幂等性键
			idempotencyKey := r.Header.Get(IdempotencyKeyHeader)
			if idempotencyKey == "" {
				// 如果没有幂等性键，继续处理
				next.ServeHTTP(w, r)
				return
			}

			// 获取用户 ID（从请求体或查询参数）
			userID := getUserIDFromRequest(r)
			// 对于用户创建请求，使用 idempotency key 作为用户标识
			if userID == "" {
				// 检查是否为用户创建请求
				if r.URL.Path == "/api/v1/users" {
					userID = idempotencyKey // 使用幂等性键作为临时用户标识
				} else {
					writeErrorResponse(w, http.StatusBadRequest, "MISSING_USER_ID", "缺少用户 ID", getRequestID(r))
					return
				}
			}

			// 读取请求体
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "无法读取请求体", getRequestID(r))
				return
			}

			// 计算请求哈希
			requestHash := calculateRequestHash(r.Method, r.URL.Path, body)

			// 检查幂等性
			ctx := r.Context()
			existingResponse, err := checkIdempotency(ctx, db, idempotencyKey, userID, requestHash)
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "IDEMPOTENCY_CHECK_FAILED", "幂等性检查失败", getRequestID(r))
				return
			}

			if existingResponse != "" {
				// 返回已存在的响应
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(existingResponse)); err != nil {
					fmt.Printf("写入响应失败: %v\n", err)
				}
				return
			}

			// 重新设置请求体
			r.Body = io.NopCloser(strings.NewReader(string(body)))

			// 创建响应记录器
			recorder := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           make([]byte, 0),
			}

			// 继续处理请求
			next.ServeHTTP(recorder, r)

			// 如果请求成功，保存响应
			if recorder.statusCode >= 200 && recorder.statusCode < 300 {
				err = saveIdempotencyResponse(ctx, db, idempotencyKey, userID, requestHash, string(recorder.body))
				if err != nil {
					// 记录错误但不影响响应
					fmt.Printf("保存幂等性响应失败: %v\n", err)
				}
			}
		})
	}
}

// responseRecorder 响应记录器
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	r.body = append(r.body, data...)
	return r.ResponseWriter.Write(data)
}

// getUserIDFromRequest 从请求中获取用户 ID
func getUserIDFromRequest(r *http.Request) string {
	// 尝试从查询参数获取
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		return userID
	}

	// 尝试从请求体获取（需要重新设置请求体）
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ""
	}

	// 重新设置请求体
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		return ""
	}

	if userID, ok := requestData["user_id"].(string); ok {
		return userID
	}

	return ""
}

// calculateRequestHash 计算请求哈希
func calculateRequestHash(method, path string, body []byte) string {
	hash := sha256.New()
	hash.Write([]byte(method))
	hash.Write([]byte(path))
	hash.Write(body)
	return hex.EncodeToString(hash.Sum(nil))
}

// checkIdempotency 检查幂等性
func checkIdempotency(ctx context.Context, db *repository.DB, key, userID, requestHash string) (string, error) {
	query := `
		SELECT response_data 
		FROM idempotency_keys 
		WHERE key = $1 AND user_id = $2 AND request_hash = $3 AND expires_at > NOW()
	`

	var responseData string
	err := db.QueryRow(ctx, query, key, userID, requestHash).Scan(&responseData)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return "", nil // 没有找到记录
		}
		return "", fmt.Errorf("查询幂等性记录失败: %w", err)
	}

	return responseData, nil
}

// saveIdempotencyResponse 保存幂等性响应
func saveIdempotencyResponse(ctx context.Context, db *repository.DB, key, userID, requestHash, responseData string) error {
	query := `
		INSERT INTO idempotency_keys (key, user_id, request_hash, response_data, created_at, expires_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW() + INTERVAL '24 hours')
		ON CONFLICT (key) DO NOTHING
	`

	_, err := db.Pool.Exec(ctx, query, key, userID, requestHash, responseData)
	if err != nil {
		return fmt.Errorf("保存幂等性记录失败: %w", err)
	}

	return nil
}
