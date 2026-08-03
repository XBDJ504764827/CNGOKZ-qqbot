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

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return NewServer(config.HTTPConfig{
		Addr:         "127.0.0.1:0",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}, config.EventConfig{}, zap.NewNop(), event.NewMemoryBus(zap.NewNop()))
}

func TestHealth(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want json", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("body[status] = %q, want %q", body["status"], "ok")
	}
}

func TestHealthMethodNotAllowed(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /health status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestNotFound(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/message/send", nil)
	rec := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET unknown path status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestEventsIntegration 端到端验证：HTTP 路由 → Event Bus → 订阅者 全链路。
func TestEventsIntegration(t *testing.T) {
	bus := event.NewMemoryBus(zap.NewNop())

	var received []event.Event
	bus.Subscribe(event.EventServerOffline, event.HandlerFunc(func(_ context.Context, e event.Event) error {
		received = append(received, e)
		return nil
	}))

	srv := NewServer(config.HTTPConfig{
		Addr:         "127.0.0.1:0",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}, config.EventConfig{APIKeys: []string{testAPIKey}}, zap.NewNop(), bus)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events",
		strings.NewReader(`{"source":"GameServer","event_type":"SERVER_OFFLINE","level":"critical","title":"服务器离线"}`))
	req.Header.Set("X-API-Key", testAPIKey)
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if len(received) != 1 {
		t.Fatalf("subscriber received %d events, want 1", len(received))
	}
	got := received[0]
	if got.Source != "GameServer" || got.EventType != event.EventServerOffline || got.Level != "critical" {
		t.Errorf("subscriber got unexpected event: %+v", got)
	}
	if got.ID == "" {
		t.Error("event ID should be generated")
	}
}
