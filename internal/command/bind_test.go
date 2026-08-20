package command

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

type bindFakeSender struct {
	calls   int
	target  string
	content string
}

func (f *bindFakeSender) SendC2CMessage(_ context.Context, target, content string) (*dto.Message, error) {
	f.calls++
	f.target = target
	f.content = content
	return &dto.Message{ID: "reply"}, nil
}

func (f *bindFakeSender) SendGroupMessage(_ context.Context, _, _ string) (*dto.Message, error) {
	return nil, nil
}

func (f *bindFakeSender) SendChannelMessage(_ context.Context, _, _ string) (*dto.Message, error) {
	return nil, nil
}

type bindFakeAuditor struct {
	events []AuditEvent
}

func (a *bindFakeAuditor) Record(event AuditEvent) error {
	a.events = append(a.events, event)
	return nil
}

func privateBindMessage(content, openid string) *dto.Message {
	return &dto.Message{
		DirectMessage: true,
		Content:       content,
		Author:        &dto.User{ID: openid},
	}
}

func TestBindHandlerRepliesOpenID(t *testing.T) {
	sender := &bindFakeSender{}
	auditor := &bindFakeAuditor{}
	h := NewBindHandler(sender, auditor, zap.NewNop())

	handled, err := h.Handle(context.Background(), privateBindMessage("/bind", "openid-123"))
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v)", handled, err)
	}
	want := "你的 QQ OpenID：\n\nopenid-123\n\n请复制此 OpenID 到网站的 QQ 绑定页面。"
	if sender.content != want {
		t.Errorf("content = %q, want %q", sender.content, want)
	}
	if sender.target != "openid-123" || sender.calls != 1 {
		t.Errorf("reply = (%q, %d)", sender.target, sender.calls)
	}
	if len(auditor.events) != 1 || auditor.events[0].Event != "command_bind" || auditor.events[0].Result != "success" {
		t.Fatalf("audit events = %+v", auditor.events)
	}
}

func TestBindHandlerOnlyExactCommand(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewBindHandler(sender, nil, zap.NewNop())

	for _, content := range []string{"/bind x", "bind", "/openid", "/bind\nextra"} {
		handled, err := h.Handle(context.Background(), privateBindMessage(content, "openid-123"))
		if err != nil || handled {
			t.Errorf("content %q: Handle() = (%v, %v), want unhandled", content, handled, err)
		}
	}
	if sender.calls != 0 {
		t.Errorf("sender calls = %d, want 0", sender.calls)
	}
}

func TestBindHandlerIgnoresNonPrivateMessage(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewBindHandler(sender, nil, zap.NewNop())

	message := privateBindMessage("/bind", "openid-123")
	message.DirectMessage = false
	message.GuildID = "guild-1"
	message.ChannelID = "channel-1"
	handled, err := h.Handle(context.Background(), message)
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v)", handled, err)
	}
	if sender.calls != 0 {
		t.Errorf("sender calls = %d, want 0", sender.calls)
	}
}

func TestBindHandlerRateLimit(t *testing.T) {
	sender := &bindFakeSender{}
	auditor := &bindFakeAuditor{}
	h := NewBindHandler(sender, auditor, zap.NewNop())
	current := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return current }

	for i := 0; i < 5; i++ {
		handled, err := h.Handle(context.Background(), privateBindMessage("/bind", "openid-123"))
		if err != nil || !handled {
			t.Fatalf("request %d: Handle() = (%v, %v)", i, handled, err)
		}
	}
	handled, err := h.Handle(context.Background(), privateBindMessage("/bind", "openid-123"))
	if err != nil || !handled {
		t.Fatalf("limited request: Handle() = (%v, %v)", handled, err)
	}
	if !strings.Contains(sender.content, "请求过于频繁") {
		t.Errorf("limited content = %q", sender.content)
	}
	if sender.calls != 6 {
		t.Errorf("sender calls = %d, want 6", sender.calls)
	}
	if auditor.events[len(auditor.events)-1].Result != "rate_limited" {
		t.Errorf("last audit = %+v", auditor.events[len(auditor.events)-1])
	}

	current = current.Add(time.Minute)
	handled, err = h.Handle(context.Background(), privateBindMessage("/bind", "openid-123"))
	if err != nil || !handled || !strings.Contains(sender.content, "openid-123") {
		t.Fatalf("after window: (%v, %v), content=%q", handled, err, sender.content)
	}
}
