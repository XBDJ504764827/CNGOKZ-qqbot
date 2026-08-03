package message

import (
	"context"
	"testing"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

func TestReceiver_ReceiveMessage(t *testing.T) {
	r := NewReceiver(zap.NewNop())
	msg := &dto.Message{
		ID:        "msg-1",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		Content:   "hello",
		Author:    &dto.User{ID: "user-1"},
	}
	if err := r.ReceiveMessage(context.Background(), msg); err != nil {
		t.Fatalf("ReceiveMessage() error = %v", err)
	}
}

// TestReceiver_NilAuthor 验证发送者为空时不 panic（部分事件不含 Author）。
func TestReceiver_NilAuthor(t *testing.T) {
	r := NewReceiver(zap.NewNop())
	if err := r.ReceiveMessage(context.Background(), &dto.Message{}); err != nil {
		t.Fatalf("ReceiveMessage() with nil author error = %v", err)
	}
}
