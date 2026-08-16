package qqapproval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/keyboard"
	"go.uber.org/zap"
)

// fakeSender 记录私聊发送。
type fakeSender struct {
	mu  sync.Mutex
	c2c []string // contents
}

func (f *fakeSender) SendC2CMessage(_ context.Context, _, content string) (*dto.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.c2c = append(f.c2c, content)
	return &dto.Message{ID: "m"}, nil
}

func (f *fakeSender) SendC2CMessageWithKeyboard(_ context.Context, _, content string, _ *keyboard.CustomKeyboard) (*dto.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.c2c = append(f.c2c, content)
	return &dto.Message{ID: "m"}, nil
}

func lastMSG(f *fakeSender) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.c2c) == 0 {
		return ""
	}
	return f.c2c[len(f.c2c)-1]
}

func newTestSvc(f *fakeSender, srv *httptest.Server) *Service {
	s, _ := New(Options{
		Sender:  f,
		BaseURL: srv.URL,
		Token:   "test-token",
	}, zap.NewNop())
	return s
}

// 一个新申请对应的 LumiAdmin 审批服务，返回唯一注册地址。
func bootServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestBuildKeyboard(t *testing.T) {
	f := &fakeSender{}
	s, _ := New(Options{Sender: f}, zap.NewNop())
	kb := s.BuildKeyboard("wl-123", "张三", nil)
	if kb == nil || len(kb.Rows) == 0 || len(kb.Rows[0].Buttons) != 2 {
		t.Fatalf("expected 2 buttons, got %+v", kb)
	}
	b0, b1 := kb.Rows[0].Buttons[0], kb.Rows[0].Buttons[1]
	if !strings.HasPrefix(b0.Action.Data, "approve:wl-123") {
		t.Errorf("button0 data = %q, want approve:wl-123", b0.Action.Data)
	}
	if !strings.HasPrefix(b1.Action.Data, "reject:wl-123") {
		t.Errorf("button1 data = %q, want reject:wl-123", b1.Action.Data)
	}
}

func TestParseAction(t *testing.T) {
	cases := []struct {
		in     string
		action string
		wl     string
		wantOK bool
	}{
		{"approve:wl-1", "approve", "wl-1", true},
		{"reject:wl-2", "reject", "wl-2", true},
		{"approve:wl-1:3f2a-etc", "approve", "wl-1", true}, // Button.ID 带 uuid
		{"unknown:x", "", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		a, wl, ok := ParseAction(c.in)
		if ok != c.wantOK || a != c.action || wl != c.wl {
			t.Errorf("ParseAction(%q) = (%q,%q,%v), want (%q,%q,%v)",
				c.in, a, wl, ok, c.action, c.wl, c.wantOK)
		}
	}
}

func TestApproveReplyRestoresNicknameFromRegistration(t *testing.T) {
	srv := bootServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"item":{}}`))
	})
	f := &fakeSender{}
	s := newTestSvc(f, srv)

	// 发送通知时登记了玩家昵称（BuildKeyboard 内部按 whitelistID 保存）
	kb := s.BuildKeyboard("wl-123", "张三", []string{"openid-1"})
	if kb == nil {
		t.Fatal("expected keyboard")
	}
	// 按钮点击回调里拿不到昵称（parseInteraction 恒返回空），只传 whitelistID
	err := s.DoAction(context.Background(), "openid-1", "approve", "wl-123", "")
	if err != nil {
		t.Fatalf("DoAction error = %v", err)
	}
	if !strings.Contains(lastMSG(f), "玩家「张三」") {
		t.Errorf("回复应含玩家昵称，实际: %q", lastMSG(f))
	}

	// 拒绝侧：同样能还原昵称到反问话术与提交结果
	if err := s.DoAction(context.Background(), "openid-2", "reject", "wl-123", ""); err != nil {
		t.Fatalf("DoAction reject error = %v", err)
	}
	if !strings.Contains(lastMSG(f), "玩家「张三」") {
		t.Errorf("拒绝反问应含玩家昵称，实际: %q", lastMSG(f))
	}
	if handled, err := s.HandleUserText(context.Background(), "openid-2", "外挂作弊"); err != nil || !handled {
		t.Fatalf("HandleUserText handled=%v err=%v", handled, err)
	}
	if !strings.Contains(lastMSG(f), "玩家「张三」") {
		t.Errorf("拒绝结果应含玩家昵称，实际: %q", lastMSG(f))
	}
}

func TestApproveSendsReviewAndReplies(t *testing.T) {
	var mu sync.Mutex
	var gotPath string
	var gotAction, gotOpenid string
	srv := bootServer(t, func(w http.ResponseWriter, r *http.Request) {
		var payload reviewPayload
		_ = jsonDecode(r, &payload)
		mu.Lock()
		gotPath = r.URL.Path
		gotAction = payload.Action
		gotOpenid = payload.OpenID
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"item":{}}`))
	})

	f := &fakeSender{}
	s := newTestSvc(f, srv)
	err := s.DoAction(context.Background(), "openid-1", "approve", "wl-1", "张三")
	if err != nil {
		t.Fatalf("DoAction approve error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !strings.HasSuffix(gotPath, "/api/integration/qq/whitelist/wl-1/review") {
		t.Errorf("path = %q, want suffix /wl-1/review", gotPath)
	}
	if gotAction != "approve" || gotOpenid != "openid-1" {
		t.Errorf("body action=%q openid=%q", gotAction, gotOpenid)
	}
	if !strings.Contains(lastMSG(f), "已通过") {
		t.Errorf("reply missing 已通过: %q", lastMSG(f))
	}
}

func TestRejectRequiresReasonFlow(t *testing.T) {
	srv := bootServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"item":{}}`))
	})
	f := &fakeSender{}
	s := newTestSvc(f, srv)

	// 点击拒绝 → 进入挂起并反问原因
	if err := s.DoAction(context.Background(), "openid-1", "reject", "wl-1", "张三"); err != nil {
		t.Fatalf("DoAction reject error = %v", err)
	}
	if !s.PeekPending("openid-1") {
		t.Fatal("expected pending reject reason")
	}
	if !strings.Contains(lastMSG(f), "请回复") && !strings.Contains(lastMSG(f), "原因") {
		t.Errorf("reply should ask reason: %q", lastMSG(f))
	}

	// 管理员回复原因 → 提交
	handled, err := s.HandleUserText(context.Background(), "openid-1", "外挂作弊")
	if err != nil {
		t.Fatalf("HandleUserText error = %v", err)
	}
	if !handled {
		t.Fatal("expected text handled")
	}
	if s.PeekPending("openid-1") {
		t.Fatal("pending should be cleared after submit")
	}
	if !strings.Contains(lastMSG(f), "已拒绝玩家「张三」") {
		t.Errorf("reply missing reject result: %q", lastMSG(f))
	}
}

func TestConcurrentReviewReturnsAlreadyReviewed(t *testing.T) {
	srv := bootServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"该申请已被他人审批，无法重复操作"}`))
	})
	f := &fakeSender{}
	s := newTestSvc(f, srv)

	err := s.DoAction(context.Background(), "openid-2", "approve", "wl-1", "李四")
	if err != nil {
		t.Fatalf("DoAction should handle already-reviewed internally, got %v", err)
	}
	if !strings.Contains(lastMSG(f), "已被其他管理员审批") {
		t.Errorf("reply missing already-reviewed notice: %q", lastMSG(f))
	}
}

func TestDoActionRejectWhileSomeoneAlreadyApproved(t *testing.T) {
	// 第一次 approve 后，另一管理员再 reject：并发保护体现在拒绝侧已由 LumiAdmin 兜底。
	// 这里验证 reject 走接口且 openid 不同也能正常挂起。
	srv := bootServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"item":{}}`))
	})
	f := &fakeSender{}
	s := newTestSvc(f, srv)
	if err := s.DoAction(context.Background(), "openid-3", "reject", "wl-2", "王五"); err != nil {
		t.Fatalf("reject start error = %v", err)
	}
	if !s.PeekPending("openid-3") {
		t.Fatal("reject should be pending")
	}
}

// 辅助：从请求体解码（v 需为指针）
func jsonDecode(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
