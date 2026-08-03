package handler

import (
	"context"
	"testing"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

func TestDefaultHandler_HandleATMessage(t *testing.T) {
	h := NewDefaultHandler(zap.NewNop())
	data := &dto.WSATMessageData{
		ID:        "msg-1",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		Content:   "hello @bot",
		Author:    &dto.User{ID: "user-1"},
	}
	if err := h.HandleATMessage(context.Background(), data); err != nil {
		t.Fatalf("HandleATMessage() error = %v", err)
	}
}

func TestDefaultHandler_HandleGroupATMessage(t *testing.T) {
	h := NewDefaultHandler(zap.NewNop())
	data := &dto.WSGroupATMessageData{
		ID:      "msg-2",
		GroupID: "group-1",
		Content: "hi @bot",
		Author:  &dto.User{ID: "user-2"},
	}
	if err := h.HandleGroupATMessage(context.Background(), data); err != nil {
		t.Fatalf("HandleGroupATMessage() error = %v", err)
	}
}

func TestDefaultHandler_HandleC2CMessage(t *testing.T) {
	h := NewDefaultHandler(zap.NewNop())
	data := &dto.WSC2CMessageData{
		ID:      "msg-3",
		Content: "private message",
		Author:  &dto.User{ID: "user-3"},
	}
	if err := h.HandleC2CMessage(context.Background(), data); err != nil {
		t.Fatalf("HandleC2CMessage() error = %v", err)
	}
}

// TestDefaultHandler_NilAuthor 验证发送者为空时处理器不 panic。
func TestDefaultHandler_NilAuthor(t *testing.T) {
	h := NewDefaultHandler(zap.NewNop())
	data := &dto.WSATMessageData{}
	if err := h.HandleATMessage(context.Background(), data); err != nil {
		t.Fatalf("HandleATMessage() with nil author error = %v", err)
	}
}
