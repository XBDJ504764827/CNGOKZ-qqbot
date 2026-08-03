package notification

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/event"
)

// TestHandler_HandleEvent 验证通知处理器可处理关键事件。
func TestHandler_HandleEvent(t *testing.T) {
	h := NewHandler(zap.NewNop())

	cases := []event.Event{
		{ID: "e1", Source: event.SourceGameServer, EventType: event.EventServerOffline, Level: event.LevelCritical, Title: "服务器离线", Message: "mc-01 心跳超时"},
		{ID: "e2", Source: event.SourceLumiAdmin, EventType: event.EventSystemWarning, Level: event.LevelWarning, Title: "磁盘告警"},
		{ID: "e3", Source: event.SourceLumiForum, EventType: event.EventForumReportCreated, Level: event.LevelWarning, Title: "新举报"},
	}
	for _, ev := range cases {
		if err := h.HandleEvent(context.Background(), ev); err != nil {
			t.Fatalf("HandleEvent(%s) error = %v", ev.EventType, err)
		}
	}
}
