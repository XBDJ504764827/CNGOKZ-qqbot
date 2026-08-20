package lumiadmin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientQueryWhitelistStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != whitelistStatusPath {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("steam_input"); got != "STEAM_0:1:12345" {
			t.Errorf("steam_input = %q", got)
		}
		if got := r.Header.Get("x-qq-token"); got != "secret-token" {
			t.Errorf("x-qq-token = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"steamid64":"76561197960290419","items":[{"status":"pending","applied_at":"2026-08-01T00:00:00Z"}]}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token", server.Client())
	result, err := client.QueryWhitelistStatus(context.Background(), "STEAM_0:1:12345")
	if err != nil {
		t.Fatalf("QueryWhitelistStatus() error = %v", err)
	}
	if result.SteamID64 != "76561197960290419" || len(result.Items) != 1 || result.Items[0].Status != "pending" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestClientQueryWhitelistStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid steam input"}`)
	}))
	defer server.Close()

	_, err := NewClient(server.URL, "secret-token", server.Client()).QueryWhitelistStatus(context.Background(), "bad")
	if err == nil || !strings.Contains(err.Error(), "invalid steam input") {
		t.Fatalf("error = %v, want API error", err)
	}
}
