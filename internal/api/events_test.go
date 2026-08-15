package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/config"
	"github.com/XBDJ504764827/LumiBot/internal/event"
)

const testAPIKey = "test-key"

// newTestEventsServer 构建带完整中间件链（认证 + 限流）的测试服务。
func newTestEventsServer(t *testing.T, eventCfg config.EventConfig) *Server {
	t.Helper()
	return NewServer(config.HTTPConfig{
		Addr:         "127.0.0.1:0",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}, eventCfg, zap.NewNop(), event.NewMemoryBus(zap.NewNop()))
}

func doEventsRequest(t *testing.T, s *Server, apiKey, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(body))
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	rec := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(rec, req)
	return rec
}

// decodeResponse 解析统一响应格式。
func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response %q: %v", rec.Body.String(), err)
	}
	return resp
}

func TestEventsAPI_OK(t *testing.T) {
	s := newTestEventsServer(t, config.EventConfig{APIKeys: []string{testAPIKey}})
	body := `{"source":"LumiForum","event_type":"FORUM_REPORT_CREATED","level":"warning","title":"新举报","message":"发现违规帖子"}`

	rec := doEventsRequest(t, s, testAPIKey, body)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	resp := decodeResponse(t, rec)
	if resp["success"] != true {
		t.Errorf("success = %v, want true", resp["success"])
	}
	if id, _ := resp["event_id"].(string); id == "" {
		t.Errorf("event_id missing in response: %v", resp)
	}
}

func TestEventsAPI_Unauthorized(t *testing.T) {
	s := newTestEventsServer(t, config.EventConfig{APIKeys: []string{testAPIKey}})
	body := `{"source":"LumiForum","event_type":"FORUM_REPORT_CREATED"}`

	// 错误 Key
	rec := doEventsRequest(t, s, "wrong-key", body)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong key status = %d, want 401", rec.Code)
	}
	if resp := decodeResponse(t, rec); resp["success"] != false || resp["error"] == "" {
		t.Errorf("unexpected error response: %v", resp)
	}

	// 缺失 Key
	rec = doEventsRequest(t, s, "", body)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing key status = %d, want 401", rec.Code)
	}
}

func TestEventsAPI_InvalidJSON(t *testing.T) {
	s := newTestEventsServer(t, config.EventConfig{APIKeys: []string{testAPIKey}})
	rec := doEventsRequest(t, s, testAPIKey, `{invalid json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestEventsAPI_ValidationFailed(t *testing.T) {
	s := newTestEventsServer(t, config.EventConfig{APIKeys: []string{testAPIKey}})

	cases := []struct {
		name string
		body string
	}{
		{"missing source", `{"event_type":"X"}`},
		{"missing event_type", `{"source":"LumiForum"}`},
		{"invalid level", `{"source":"LumiForum","event_type":"X","level":"fatal"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doEventsRequest(t, s, testAPIKey, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if resp := decodeResponse(t, rec); resp["success"] != false {
				t.Errorf("success = %v, want false", resp["success"])
			}
		})
	}
}

// TestEventsAPI_RateLimited 验证超过限流阈值返回 429。
func TestEventsAPI_RateLimited(t *testing.T) {
	s := newTestEventsServer(t, config.EventConfig{
		APIKeys:   []string{testAPIKey},
		RateLimit: 2, // 每个 Key 每分钟仅 2 次
	})
	body := `{"source":"LumiForum","event_type":"X"}`

	for i := 0; i < 2; i++ {
		rec := doEventsRequest(t, s, testAPIKey, body)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("request %d status = %d, want 202", i+1, rec.Code)
		}
	}
	rec := doEventsRequest(t, s, testAPIKey, body)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("over-limit status = %d, want 429", rec.Code)
	}
}

// TestEventsAPI_NoKeysConfigured 验证未配置 API Key 时拒绝所有请求（fail-closed）。
func TestEventsAPI_NoKeysConfigured(t *testing.T) {
	s := newTestEventsServer(t, config.EventConfig{})
	rec := doEventsRequest(t, s, testAPIKey, `{"source":"LumiForum","event_type":"X"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestEventsIntegration 端到端验证：HTTP 全链（认证+限流+处理）→ Event Bus → 订阅者。
func TestEventsIntegration(t *testing.T) {
	bus := event.NewMemoryBus(zap.NewNop())
	var received []event.Event
	bus.Subscribe(event.EventServerOffline, event.HandlerFunc(func(_ context.Context, e event.Event) error {
		received = append(received, e)
		return nil
	}))

	s := NewServer(config.HTTPConfig{
		Addr:         "127.0.0.1:0",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}, config.EventConfig{APIKeys: []string{testAPIKey}}, zap.NewNop(), bus)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events",
		strings.NewReader(`{"source":"GameMonitor","event_type":"SERVER_OFFLINE","level":"critical","title":"服务器离线","data":{"server":"KZ-01"}}`))
	req.Header.Set("X-API-Key", testAPIKey)
	rec := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if len(received) != 1 {
		t.Fatalf("subscriber received %d events, want 1", len(received))
	}
	got := received[0]
	if got.Source != "GameMonitor" || got.EventType != event.EventServerOffline {
		t.Errorf("subscriber got unexpected event: %+v", got)
	}
	if got.Data["server"] != "KZ-01" {
		t.Errorf("event data lost: %+v", got.Data)
	}
	if got.ID == "" {
		t.Error("event ID should be generated")
	}
}
