package auth

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateLimiter 限流器接口。
//
// 当前实现：MemoryRateLimiter（进程内固定窗口）。
// 扩展能力：Redis 实现同一接口（如 RedisRateLimiter），
// 支持多实例共享限流状态，业务层无需改动。
type RateLimiter interface {
	// Allow 判断 key 是否允许通过；window 窗口内超过 limit 次则拒绝。
	Allow(key string) bool
}

// MemoryRateLimiter 固定窗口内存限流器（单实例适用）。
type MemoryRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*rateBucket
}

// rateBucket 单个 key 的计数窗口。
type rateBucket struct {
	windowStart time.Time
	count       int
}

// NewMemoryRateLimiter 构建限流器（limit 次 / window 时间窗口）。
func NewMemoryRateLimiter(limit int, window time.Duration) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		limit:   limit,
		window:  window,
		buckets: make(map[string]*rateBucket),
	}
}

// Allow 实现 RateLimiter 接口。
func (l *MemoryRateLimiter) Allow(key string) bool {
	if l.limit <= 0 {
		return true // limit <= 0 表示不限制
	}
	if key == "" {
		return true // 未知来源不限制（后续由认证层拒绝）
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok || now.Sub(b.windowStart) >= l.window {
		l.buckets[key] = &rateBucket{windowStart: now, count: 1}
		return true
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}

// RateLimitMiddleware 返回限流中间件：按认证通过的 API Key 限流，
// 超限返回 429 + 统一错误响应。
// 限流在认证之后执行（从 request context 读取 Key）。
func RateLimitMiddleware(limiter RateLimiter, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, _ := APIKeyFrom(r.Context())
			if !limiter.Allow(key) {
				logger.Warn("事件上报触发限流",
					zap.String("api_key", key),
					zap.String("remote_addr", r.RemoteAddr),
				)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   "rate limit exceeded",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
