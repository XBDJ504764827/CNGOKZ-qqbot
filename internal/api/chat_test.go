package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi/options"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/config"
	"github.com/XBDJ504764827/LumiBot/internal/event"
	"github.com/XBDJ504764827/LumiBot/internal/message"
)

// fakeC2CAPI 实现 message.MessageAPI 的测试替身，只关心 C2C 发送。
type fakeC2CAPI struct {
	c2cSent []string
}

func (f *fakeC2CAPI) PostMessage(context.Context, string, *dto.MessageToCreate, ...options.Option) (*dto.Message, error) {
	return &dto.Message{}, nil
}

func (f *fakeC2CAPI) PostGroupMessage(context.Context, string, dto.APIMessage, ...options.Option) (*dto.Message, error) {
	return &dto.Message{}, nil
}

func (f *fakeC2CAPI) PostC2CMessage(_ context.Context, userID string, _ dto.APIMessage, _ ...options.Option) (*dto.Message, error) {
	f.c2cSent = append(f.c2cSent, userID)
	return &dto.Message{ID: "msg-1"}, nil
}

func TestChatHandlerSendsC2C(t *testing.T) {
	api := &fakeC2CAPI{}
	sender := message.NewSender(api, zap.NewNop())
	handler := NewChatHandler(sender, zap.NewNop())

	body := `{"openid":"user-1","content":"你好","operator":"管理员A"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("success = %v, want true", resp["success"])
	}
	if len(api.c2cSent) != 1 || api.c2cSent[0] != "user-1" {
		t.Errorf("c2c targets = %v, want [user-1]", api.c2cSent)
	}
}

func TestChatHandlerValidatesInput(t *testing.T) {
	sender := message.NewSender(&fakeC2CAPI{}, zap.NewNop())
	handler := NewChatHandler(sender, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/chat", strings.NewReader(`{"openid":"","content":""}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestServerRegistersChatRoute 验证 /api/v1/messages/chat 已注册且需要鉴权。
func TestServerRegistersChatRoute(t *testing.T) {
	s := NewServer(config.HTTPConfig{
		Addr: "127.0.0.1:0",
	}, config.EventConfig{APIKeys: []string{"test-key"}}, zap.NewNop(), event.NewMemoryBus(zap.NewNop()), nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/chat", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (missing X-API-Key)", rec.Code)
	}
}
