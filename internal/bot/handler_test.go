package bot

import (
	"context"
	"testing"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/message"
)

func newTestHandler() *Handler {
	return NewHandler(
		message.NewReceiver(zap.NewNop()),
		message.NewSender(nil, zap.NewNop()),
		zap.NewNop(),
	)
}

func TestHandler_OnMessage(t *testing.T) {
	h := newTestHandler()
	msg := &dto.Message{ID: "msg-1", GuildID: "guild-1", ChannelID: "channel-1", Content: "hello", Author: &dto.User{ID: "user-1"}}
	if err := h.OnMessage(context.Background(), msg); err != nil {
		t.Fatalf("OnMessage() error = %v", err)
	}
}

func TestHandler_OnATMessage(t *testing.T) {
	h := newTestHandler()
	msg := &dto.Message{ID: "msg-2", ChannelID: "channel-1", Content: "@bot hi"}
	if err := h.OnATMessage(context.Background(), msg); err != nil {
		t.Fatalf("OnATMessage() error = %v", err)
	}
}

// TestHandler_OnC2CMessage 验证私聊消息分发（user_id 即用户 openid）。
func TestHandler_OnC2CMessage(t *testing.T) {
	h := newTestHandler()
	msg := &dto.Message{ID: "msg-3", Content: "你好", Author: &dto.User{ID: "openid-abc123"}}
	if err := h.OnC2CMessage(context.Background(), msg); err != nil {
		t.Fatalf("OnC2CMessage() error = %v", err)
	}
}

// TestHandler_OnReadyOnErrorOnPlain 验证连接类事件分发不 panic。
func TestHandler_OnReadyOnErrorOnPlain(t *testing.T) {
	h := newTestHandler()

	h.OnReady(&dto.WSReadyData{SessionID: "s1", User: struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Bot      bool   `json:"bot"`
	}{ID: "bot-1"}})
	h.OnError(context.Canceled)
	h.OnPlain(&dto.WSPayload{WSPayloadBase: dto.WSPayloadBase{Type: dto.EventMessageCreate}}, []byte("{}"))
}
