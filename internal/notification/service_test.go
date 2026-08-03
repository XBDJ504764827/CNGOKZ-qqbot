package notification

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/config"
	"github.com/XBDJ504764827/LumiBot/internal/event"
)

// fakeSender 测试替身：记录发送调用并支持注入错误。
type fakeSender struct {
	mu       sync.Mutex
	c2cCalls []string // userID
	chCalls  []string // channelID
	fail     bool
}

func (f *fakeSender) SendC2CMessage(_ context.Context, userID, _ string) (*dto.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.c2cCalls = append(f.c2cCalls, userID)
	if f.fail {
		return nil, errors.New("qq api unavailable")
	}
	return &dto.Message{ID: "m-1"}, nil
}

func (f *fakeSender) SendChannelMessage(_ context.Context, channelID, _ string) (*dto.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chCalls = append(f.chCalls, channelID)
	if f.fail {
		return nil, errors.New("qq api unavailable")
	}
	return &dto.Message{ID: "m-2"}, nil
}

func (f *fakeSender) c2cCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.c2cCalls)
}

func newTestService(sender *fakeSender) *Service {
	cfg := config.NotificationConfig{
		Enable:        true,
		Cooldown:      5 * time.Minute,
		PrivateTarget: "admin-qq-openid",
		ChannelTarget: "channel-1",
	}
	if sender == nil {
		sender = &fakeSender{}
	}
	return NewService(cfg, sender, zap.NewNop())
}

func offlineEvent() event.Event {
	return event.Event{
		ID:        "evt-1",
		Source:    "GameMonitor",
		EventType: event.EventServerOffline,
		Level:     event.LevelCritical,
		Timestamp: time.Now(),
		Title:     "服务器离线",
		Message:   "KZ服务器01停止响应",
		Data:      map[string]interface{}{"server": "KZ-01", "status": "离线"},
	}
}

// TestHandleEvent_EventToNotification 验证事件 → 通知 → QQ 发送全链路。
func TestHandleEvent_EventToNotification(t *testing.T) {
	sender := &fakeSender{}
	s := newTestService(sender)

	if err := s.HandleEvent(context.Background(), offlineEvent()); err != nil {
		t.Fatalf("HandleEvent() error = %v", err)
	}
	if sender.c2cCount() != 1 {
		t.Fatalf("c2c send calls = %d, want 1", sender.c2cCount())
	}
	if sender.c2cCalls[0] != "admin-qq-openid" {
		t.Errorf("sent to %q, want admin-qq-openid", sender.c2cCalls[0])
	}
}

// TestHandleEvent_Cooldown 验证冷却机制：窗口内重复事件只发送一次。
func TestHandleEvent_Cooldown(t *testing.T) {
	sender := &fakeSender{}
	s := newTestService(sender)

	ev := offlineEvent()
	if err := s.HandleEvent(context.Background(), ev); err != nil {
		t.Fatalf("first HandleEvent() error = %v", err)
	}
	if err := s.HandleEvent(context.Background(), ev); err != nil {
		t.Fatalf("second HandleEvent() error = %v", err)
	}
	if err := s.HandleEvent(context.Background(), ev); err != nil {
		t.Fatalf("third HandleEvent() error = %v", err)
	}
	if sender.c2cCount() != 1 {
		t.Fatalf("c2c send calls = %d, want 1 (cooldown should dedupe)", sender.c2cCount())
	}
}

// TestHandleEvent_CooldownExpired 验证冷却过期后可再次发送。
func TestHandleEvent_CooldownExpired(t *testing.T) {
	sender := &fakeSender{}
	s := newTestService(sender)

	ev := offlineEvent()
	if err := s.HandleEvent(context.Background(), ev); err != nil {
		t.Fatalf("first HandleEvent() error = %v", err)
	}
	// 手动清空冷却状态模拟窗口过期
	s.cooldown.Reset()
	if err := s.HandleEvent(context.Background(), ev); err != nil {
		t.Fatalf("second HandleEvent() error = %v", err)
	}
	if sender.c2cCount() != 2 {
		t.Fatalf("c2c send calls = %d, want 2 (cooldown expired)", sender.c2cCount())
	}
}

// TestHandleEvent_Disabled 验证开关关闭时不发送。
func TestHandleEvent_Disabled(t *testing.T) {
	sender := &fakeSender{}
	cfg := config.NotificationConfig{Enable: false, Cooldown: 5 * time.Minute, PrivateTarget: "admin"}
	s := NewService(cfg, sender, zap.NewNop())

	if err := s.HandleEvent(context.Background(), offlineEvent()); err != nil {
		t.Fatalf("HandleEvent() error = %v", err)
	}
	if sender.c2cCount() != 0 {
		t.Fatalf("c2c send calls = %d, want 0 (disabled)", sender.c2cCount())
	}
}

// TestHandleEvent_RuleFiltered 验证不满足规则的事件（如未知类型）不发送。
func TestHandleEvent_RuleFiltered(t *testing.T) {
	sender := &fakeSender{}
	s := newTestService(sender)

	ev := offlineEvent()
	ev.EventType = "MESSAGE_RECEIVED"
	if err := s.HandleEvent(context.Background(), ev); err != nil {
		t.Fatalf("HandleEvent() error = %v", err)
	}
	if sender.c2cCount() != 0 {
		t.Fatalf("c2c send calls = %d, want 0 (rule filtered)", sender.c2cCount())
	}
}

// TestHandleEvent_SendFailure 验证发送失败时返回错误（上层记录日志）。
func TestHandleEvent_SendFailure(t *testing.T) {
	sender := &fakeSender{fail: true}
	s := newTestService(sender)

	if err := s.HandleEvent(context.Background(), offlineEvent()); err == nil {
		t.Fatal("HandleEvent() expected error on send failure, got nil")
	}
}

// TestHandleEvent_NoPrivateTarget 验证未配置私聊目标时发送被跳过并返回错误。
func TestHandleEvent_NoPrivateTarget(t *testing.T) {
	sender := &fakeSender{}
	cfg := config.NotificationConfig{Enable: true, Cooldown: 5 * time.Minute, PrivateTarget: ""}
	s := NewService(cfg, sender, zap.NewNop())

	if err := s.HandleEvent(context.Background(), offlineEvent()); err == nil {
		t.Fatal("HandleEvent() expected error when target not configured, got nil")
	}
	if sender.c2cCount() != 0 {
		t.Fatalf("c2c send calls = %d, want 0", sender.c2cCount())
	}
}
