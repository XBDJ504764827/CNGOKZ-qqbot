package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/event"
)

const testAPIKey = "test-key"

func newTestEventsHandler() *EventsHandler {
	return NewEventsHandler(event.NewMemoryBus(zap.NewNop()), []string{testAPIKey}, zap.NewNop())
}

func doEventsRequest(t *testing.T, h http.Handler, method, path, apiKey, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestEventsAPI_OK(t *testing.T) {
	h := newTestEventsHandler()
	body := `{"source":"LumiForum","event_type":"FORUM_REPORT_CREATED","level":"warning","title":"新举报","message":"发现违规帖子"}`

	rec := doEventsRequest(t, h, http.MethodPost, "/api/v1/events", testAPIKey, body)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var resp struct {
		Status  string `json:"status"`
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp.Status != "accepted" || resp.EventID == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestEventsAPI_Unauthorized(t *testing.T) {
	h := newTestEventsHandler()
	body := `{"source":"LumiForum","event_type":"FORUM_REPORT_CREATED"}`

	rec := doEventsRequest(t, h, http.MethodPost, "/api/v1/events", "wrong-key", body)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestEventsAPI_MissingAPIKey(t *testing.T) {
	h := newTestEventsHandler()
	rec := doEventsRequest(t, h, http.MethodPost, "/api/v1/events", "", `{"source":"LumiForum","event_type":"X"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestEventsAPI_InvalidJSON(t *testing.T) {
	h := newTestEventsHandler()
	rec := doEventsRequest(t, h, http.MethodPost, "/api/v1/events", testAPIKey, `{invalid json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestEventsAPI_ValidationFailed(t *testing.T) {
	h := newTestEventsHandler()

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
			rec := doEventsRequest(t, h, http.MethodPost, "/api/v1/events", testAPIKey, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

// TestEventsAPI_NoKeysConfigured 验证未配置 API Key 时拒绝所有请求（fail-closed）。
func TestEventsAPI_NoKeysConfigured(t *testing.T) {
	h := NewEventsHandler(event.NewMemoryBus(zap.NewNop()), nil, zap.NewNop())
	rec := doEventsRequest(t, h, http.MethodPost, "/api/v1/events", testAPIKey, `{"source":"LumiForum","event_type":"X"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
