package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMemoryRateLimiter_Allow(t *testing.T) {
	limiter := NewMemoryRateLimiter(2, time.Minute)

	if !limiter.Allow("key-admin") {
		t.Fatal("first request should be allowed")
	}
	if !limiter.Allow("key-admin") {
		t.Fatal("second request should be allowed")
	}
	if limiter.Allow("key-admin") {
		t.Fatal("third request within window should be rejected (limit 2)")
	}
	// 不同 key 独立计数
	if !limiter.Allow("key-forum") {
		t.Fatal("different key should be allowed independently")
	}
}

func TestMemoryRateLimiter_WindowExpired(t *testing.T) {
	limiter := NewMemoryRateLimiter(1, time.Millisecond)

	if !limiter.Allow("key-admin") {
		t.Fatal("first request should be allowed")
	}
	if limiter.Allow("key-admin") {
		t.Fatal("second request within window should be rejected")
	}
	time.Sleep(2 * time.Millisecond)
	if !limiter.Allow("key-admin") {
		t.Fatal("request after window expiry should be allowed")
	}
}

func TestMemoryRateLimiter_EmptyKey(t *testing.T) {
	limiter := NewMemoryRateLimiter(1, time.Minute)
	if !limiter.Allow("") {
		t.Fatal("empty key should always be allowed (auth layer rejects it later)")
	}
}

// TestRateLimitMiddleware 验证限流中间件：超限返回 429 + 统一响应。
func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewMemoryRateLimiter(1, time.Minute)
	mw := RateLimitMiddleware(limiter, zap.NewNop())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	req = req.WithContext(WithAPIKey(req.Context(), "key-admin"))

	// 第一次放行
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", rec.Code)
	}

	// 第二次触发限流
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", rec.Code)
	}
}
