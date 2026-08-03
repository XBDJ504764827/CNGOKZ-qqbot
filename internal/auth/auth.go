// Package auth 提供事件上报 API 的保护能力：
//
//   - API Key 认证（X-API-Key，支持多个 Key，分别分配给不同外部系统）
//   - 限流（防止异常系统大量发送事件，内存实现，预留 Redis）
//
// 认证与限流均为标准 HTTP 中间件（func(http.Handler) http.Handler），
// 与业务 Handler 解耦，可复用于未来新增的 API 路由。
package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// apiKeyCtxKey 用于在请求 context 中传递认证通过的 API Key。
type apiKeyCtxKey struct{}

// WithAPIKey 将认证通过的 API Key 写入 context。
func WithAPIKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, apiKeyCtxKey{}, key)
}

// APIKeyFrom 从 context 中读取认证通过的 API Key。
func APIKeyFrom(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(apiKeyCtxKey{}).(string)
	return key, ok
}

// Authenticator 基于 X-API-Key 的请求认证器。
//
// 支持多个 Key（如 LumiAdmin: key-admin、LumiForum: key-forum），
// 通过 EVENT_API_KEYS 逗号分隔配置。
type Authenticator struct {
	keys map[string]struct{}
}

// NewAuthenticator 构建认证器。
func NewAuthenticator(keys []string) *Authenticator {
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k != "" {
			set[k] = struct{}{}
		}
	}
	return &Authenticator{keys: set}
}

// Valid 判断 Key 是否合法。
func (a *Authenticator) Valid(key string) bool {
	_, ok := a.keys[key]
	return ok
}

// Middleware 返回认证中间件：
//   - 认证通过：Key 写入请求 context，放行
//   - 认证失败：401 + 统一错误响应
//   - 未配置任何 Key：拒绝所有请求（fail-closed）
func (a *Authenticator) Middleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(a.keys) == 0 {
				logger.Warn("未配置 EVENT_API_KEYS，拒绝事件上报请求（fail-closed）")
				writeJSON(w, http.StatusUnauthorized, "invalid api key")
				return
			}

			key := r.Header.Get("X-API-Key")
			if !a.Valid(key) {
				writeJSON(w, http.StatusUnauthorized, "invalid api key")
				return
			}

			next.ServeHTTP(w, r.WithContext(WithAPIKey(r.Context(), key)))
		})
	}
}

// writeJSON 输出统一错误响应格式：{"success":false,"error":"..."}。
func writeJSON(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   msg,
	})
}
