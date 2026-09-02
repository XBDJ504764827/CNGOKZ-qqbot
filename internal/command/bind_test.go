package command

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// bindFakeSender 记录 C2C 发送调用的测试桩。
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

// privateBindMessage 构造 C2C 私聊消息（与 bot 层处理一致：标记 DirectMessage）。
func privateBindMessage(content, openid string) *dto.Message {
	return &dto.Message{
		DirectMessage: true,
		Content:       content,
		Author:        &dto.User{ID: openid},
	}
}

func TestBindHandlerRepliesOpenID(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewBindHandler(sender, zap.NewNop())

	handled, err := h.Handle(context.Background(), privateBindMessage("/bind", "openid-123"))
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v)", handled, err)
	}
	want := "你的 QQ OpenID：\n\nopenid-123\n\n请复制此 OpenID 到 LumiAdmin 网站的 QQ 绑定页面完成绑定。"
	if sender.content != want {
		t.Errorf("content = %q, want %q", sender.content, want)
	}
	if sender.target != "openid-123" || sender.calls != 1 {
		t.Errorf("reply = (%q, %d)", sender.target, sender.calls)
	}
}

func TestBindHandlerExactCommandOnly(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewBindHandler(sender, zap.NewNop())

	for _, content := range []string{"/bind x", "bind", "/openid", "/bind\nextra"} {
		handled, err := h.Handle(context.Background(), privateBindMessage(content, "openid-123"))
		if handled {
			t.Errorf("content %q: handled = true, want unhandled", content)
		}
		if err != nil {
			t.Errorf("content %q: error = %v", content, err)
		}
	}
	if sender.calls != 0 {
		t.Errorf("sender calls = %d, want 0", sender.calls)
	}
}

// TestBindHandlerTrimWhitespace 验证忽略首尾空白后仍精确匹配。
func TestBindHandlerTrimWhitespace(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewBindHandler(sender, zap.NewNop())

	handled, err := h.Handle(context.Background(), privateBindMessage(" /bind ", "openid-123"))
	if err != nil || !handled || sender.calls != 1 {
		t.Fatalf("Handle() = (%v, %v), calls = %d", handled, err, sender.calls)
	}
}

func TestBindHandlerCaseInsensitive(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewBindHandler(sender, zap.NewNop())

	handled, err := h.Handle(context.Background(), privateBindMessage("/BIND", "openid-1"))
	if err != nil || !handled || sender.calls != 1 {
		t.Fatalf("Handle() = (%v, %v), calls = %d", handled, err, sender.calls)
	}
}

func TestBindHandlerIgnoresNonPrivateMessage(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewBindHandler(sender, zap.NewNop())

	message := privateBindMessage("/bind", "openid-123")
	message.DirectMessage = false
	message.GroupID = "group-1"
	handled, err := h.Handle(context.Background(), message)
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v)", handled, err)
	}
	if sender.calls != 0 {
		t.Errorf("sender calls = %d, want 0（群聊/频道不回复）", sender.calls)
	}
}

func TestBindHandlerNilMessage(t *testing.T) {
	h := NewBindHandler(&bindFakeSender{}, zap.NewNop())
	handled, err := h.Handle(context.Background(), nil)
	if err != nil || handled {
		t.Fatalf("Handle(nil) = (%v, %v), want (false, nil)", handled, err)
	}
}

func TestBindHandlerRateLimit(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewBindHandler(sender, zap.NewNop())
	current := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return current }

	for i := 0; i < bindRateLimit; i++ {
		handled, err := h.Handle(context.Background(), privateBindMessage("/bind", "openid-123"))
		if err != nil || !handled {
			t.Fatalf("request %d: Handle() = (%v, %v)", i, handled, err)
		}
	}
	// 第 6 次：命中限流，回复“请求过于频繁”。
	handled, err := h.Handle(context.Background(), privateBindMessage("/bind", "openid-123"))
	if err != nil || !handled {
		t.Fatalf("limited request: Handle() = (%v, %v)", handled, err)
	}
	if !strings.Contains(sender.content, "请求过于频繁") {
		t.Errorf("limited content = %q", sender.content)
	}
	if sender.calls != bindRateLimit+1 {
		t.Errorf("sender calls = %d, want %d", sender.calls, bindRateLimit+1)
	}

	// 窗口过后恢复。
	current = current.Add(bindWindow)
	handled, err = h.Handle(context.Background(), privateBindMessage("/bind", "openid-123"))
	if err != nil || !handled || !strings.Contains(sender.content, "openid-123") {
		t.Fatalf("after window: (%v, %v), content=%q", handled, err, sender.content)
	}
}

func TestBindHandlerDifferentOpenIDsIndependent(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewBindHandler(sender, zap.NewNop())
	current := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return current }

	for i := 0; i < bindRateLimit; i++ {
		// 用户 A 刷满限流。
		if _, err := h.Handle(context.Background(), privateBindMessage("/bind", "openid-A")); err != nil {
			t.Fatalf("user A request %d: %v", i, err)
		}
	}
	// 用户 B 不受影响。
	handled, err := h.Handle(context.Background(), privateBindMessage("/bind", "openid-B"))
	if err != nil || !handled || !strings.Contains(sender.content, "openid-B") {
		t.Fatalf("user B: (%v, %v), content=%q", handled, err, sender.content)
	}
}

func TestBindHandlerNilSender(t *testing.T) {
	h := NewBindHandler(nil, zap.NewNop())
	handled, err := h.Handle(context.Background(), privateBindMessage("/bind", "openid-123"))
	if err == nil || !handled {
		t.Fatalf("Handle() = (%v, %v), want handled with error", handled, err)
	}
}

func TestRouterDispatch(t *testing.T) {
	sender := &bindFakeSender{}
	bind := NewBindHandler(sender, zap.NewNop())
	router := NewRouter(bind)

	// 非匹配消息 → 未消费。
	handled, err := router.Handle(context.Background(), privateBindMessage("hello", "openid-1"))
	if err != nil || handled {
		t.Fatalf("non-command: (%v, %v)", handled, err)
	}
	// /bind → 消费并回复。
	handled, err = router.Handle(context.Background(), privateBindMessage("/bind", "openid-1"))
	if err != nil || !handled || sender.calls != 1 {
		t.Fatalf("/bind: (%v, %v), calls = %d", handled, err, sender.calls)
	}
}
