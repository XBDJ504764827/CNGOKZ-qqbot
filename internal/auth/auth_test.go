package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestAuthenticator_Valid(t *testing.T) {
	a := NewAuthenticator([]string{"key-admin", "key-forum"})

	if !a.Valid("key-admin") {
		t.Error("key-admin should be valid")
	}
	if !a.Valid("key-forum") {
		t.Error("key-forum should be valid")
	}
	if a.Valid("key-wrong") {
		t.Error("key-wrong should be invalid")
	}
	if a.Valid("") {
		t.Error("empty key should be invalid")
	}
}

func TestAuthenticator_NoKeys(t *testing.T) {
	a := NewAuthenticator(nil)
	if a.Valid("anything") {
		t.Error("authenticator without keys should reject all (fail-closed)")
	}
}

func TestAuthenticator_Middleware(t *testing.T) {
	a := NewAuthenticator([]string{"key-admin"})
	handler := a.Middleware(zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证认证通过的 Key 已写入 context
		key, ok := APIKeyFrom(r.Context())
		if !ok || key != "key-admin" {
			t.Errorf("context api key = %q, ok=%v, want key-admin", key, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))

	// 正确 Key：放行
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	req.Header.Set("X-API-Key", "key-admin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("valid key status = %d, want 200", rec.Code)
	}

	// 错误 Key：401 + 统一响应
	req = httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	req.Header.Set("X-API-Key", "wrong")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("invalid key status = %d, want 401", rec.Code)
	}

	// 缺失 Key：401
	req = httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing key status = %d, want 401", rec.Code)
	}
}
