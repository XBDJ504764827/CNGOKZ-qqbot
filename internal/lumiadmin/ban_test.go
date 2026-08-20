package lumiadmin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientQueryBanStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != banStatusPath {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("steam_input") != "76561198012345678" {
			t.Errorf("steam_input = %q", r.URL.Query().Get("steam_input"))
		}
		if r.Header.Get("x-qq-token") != "token" {
			t.Errorf("x-qq-token = %q", r.Header.Get("x-qq-token"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"steamid64":"76561198012345678","local_bans":[],"global_bans":[]}`)
	}))
	defer server.Close()

	result, err := NewClient(server.URL, "token", server.Client()).QueryBanStatus(context.Background(), "76561198012345678")
	if err != nil {
		t.Fatalf("QueryBanStatus() error = %v", err)
	}
	if result.SteamID64 != "76561198012345678" || len(result.LocalBans) != 0 || len(result.GlobalBans) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
