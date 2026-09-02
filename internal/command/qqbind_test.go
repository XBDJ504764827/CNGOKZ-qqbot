package command

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// groupBindMessage 构造群消息（GroupID 非空）。
func groupBindMessage(content, openid string) *dto.Message {
	return &dto.Message{
		GroupID: "group-1",
		Content: content,
		Author:  &dto.User{ID: openid},
	}
}

func TestExtractUUID(t *testing.T) {
	cases := []struct {
		content string
		want    string
	}{
		{"68890057-1234-5678-9abc-def012345678", "68890057-1234-5678-9abc-def012345678"},
		{"@bot 68890057-1234-5678-9abc-def012345678", "68890057-1234-5678-9abc-def012345678"},
		{" 68890057-1234-5678-9abc-def012345678 ", "68890057-1234-5678-9abc-def012345678"},
		{"6889", ""},
		{"hello world", ""},
		{"@bot 你好", ""},
	}
	for _, tc := range cases {
		if got := extractUUID(tc.content); got != tc.want {
			t.Errorf("extractUUID(%q) = %q, want %q", tc.content, got, tc.want)
		}
	}
}

func TestQQBindHandlerGroupMessageCallsAPI(t *testing.T) {
	const code = "68890057-1234-5678-9abc-def012345678"

	var gotCode, gotOpenid string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/qq/bind" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "token-xyz" {
			t.Errorf("missing api key header")
		}
		var payload struct {
			Code     string `json:"code"`
			QQOpenID string `json:"qq_openid"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		gotCode = payload.Code
		gotOpenid = payload.QQOpenID
		_, _ = w.Write([]byte(`{"success":true,"steamid64":"76561198000000001","message":"绑定成功"}`))
	}))
	defer server.Close()

	sender := &bindFakeSender{}
	h := NewQQBindHandler(QQBindHandlerOptions{
		Sender:   sender,
		Logger:   zap.NewNop(),
		APIURL:   server.URL,
		APIToken: "token-xyz",
	})

	handled, err := h.Handle(context.Background(), groupBindMessage("@bot "+code, "openid-888"))
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v)", handled, err)
	}
	if gotCode != code || gotOpenid != "openid-888" {
		t.Errorf("api payload = (%q, %q)", gotCode, gotOpenid)
	}
	if sender.calls != 1 {
		t.Errorf("sender calls = %d, want 1", sender.calls)
	}
	if !strings.Contains(sender.content, "绑定成功") {
		t.Errorf("reply content = %q", sender.content)
	}
}

func TestQQBindHandlerIgnoresNonUUID(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewQQBindHandler(QQBindHandlerOptions{
		Sender: sender,
		Logger: zap.NewNop(),
	})

	for _, content := range []string{"hello", "/bind", "@bot 你好世界", "123"} {
		handled, err := h.Handle(context.Background(), groupBindMessage(content, "openid-888"))
		if handled || err != nil {
			t.Errorf("content %q: handled=%v err=%v, want unhandled", content, handled, err)
		}
	}
	if sender.calls != 0 {
		t.Errorf("sender calls = %d, want 0", sender.calls)
	}
}

func TestQQBindHandlerIgnoresPrivateMessage(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewQQBindHandler(QQBindHandlerOptions{Sender: sender, Logger: zap.NewNop()})

	handled, err := h.Handle(context.Background(), privateBindMessage("68890057-1234-5678-9abc-def012345678", "openid-888"))
	if err != nil || handled {
		t.Fatalf("private msg Handle() = (%v, %v), want unhandled", handled, err)
	}
	if sender.calls != 0 {
		t.Errorf("sender calls = %d, want 0", sender.calls)
	}
}

func TestQQBindHandlerUnconfiguredRepliesDisabled(t *testing.T) {
	sender := &bindFakeSender{}
	h := NewQQBindHandler(QQBindHandlerOptions{Sender: sender, Logger: zap.NewNop()})

	handled, err := h.Handle(context.Background(), groupBindMessage("@bot 68890057-1234-5678-9abc-def012345678", "openid-888"))
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v)", handled, err)
	}
	if !strings.Contains(sender.content, "未启用") {
		t.Errorf("reply content = %q, want 未启用提示", sender.content)
	}
}
