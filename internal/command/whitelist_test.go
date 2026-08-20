package command

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/lumiadmin"
)

type fakeSender struct {
	kind    string
	target  string
	content string
}

func (f *fakeSender) SendC2CMessage(_ context.Context, target, content string) (*dto.Message, error) {
	f.kind, f.target, f.content = "c2c", target, content
	return &dto.Message{ID: "sent"}, nil
}

func (f *fakeSender) SendGroupMessage(_ context.Context, target, content string) (*dto.Message, error) {
	f.kind, f.target, f.content = "group", target, content
	return &dto.Message{ID: "sent"}, nil
}

func (f *fakeSender) SendChannelMessage(_ context.Context, target, content string) (*dto.Message, error) {
	f.kind, f.target, f.content = "channel", target, content
	return &dto.Message{ID: "sent"}, nil
}

type fakeQuerier struct {
	result *lumiadmin.WhitelistStatusResponse
	err    error
	input  string
}

func (f *fakeQuerier) QueryWhitelistStatus(_ context.Context, input string) (*lumiadmin.WhitelistStatusResponse, error) {
	f.input = input
	return f.result, f.err
}

func TestWhitelistHandler_HandlePrivateApproved(t *testing.T) {
	sender := &fakeSender{}
	querier := &fakeQuerier{result: &lumiadmin.WhitelistStatusResponse{
		SteamID64: "76561198012345678",
		Items: []lumiadmin.WhitelistRecord{{
			Status:     "approved",
			AppliedAt:  "2026-08-01T10:00:00Z",
			ApprovedAt: stringPtr("2026-08-02T10:30:00Z"),
		}},
	}}
	h := NewWhitelistHandler(sender, querier, zap.NewNop())

	handled, err := h.Handle(context.Background(), &dto.Message{
		Content: "/wl 76561198012345678",
		Author:  &dto.User{ID: "openid-1"},
	})
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v), want handled without error", handled, err)
	}
	if querier.input != "76561198012345678" {
		t.Errorf("querier input = %q", querier.input)
	}
	if sender.kind != "c2c" || sender.target != "openid-1" {
		t.Errorf("reply target = (%s, %s), want c2c/openid-1", sender.kind, sender.target)
	}
	if !strings.Contains(sender.content, "✅ 已通过") {
		t.Errorf("reply = %q, missing status", sender.content)
	}
}

func TestWhitelistHandler_HandleGroup(t *testing.T) {
	sender := &fakeSender{}
	querier := &fakeQuerier{result: &lumiadmin.WhitelistStatusResponse{SteamID64: "76561198012345678"}}
	h := NewWhitelistHandler(sender, querier, zap.NewNop())

	handled, err := h.Handle(context.Background(), &dto.Message{
		GroupID: "group-1",
		Content: "/wl STEAM_0:1:12345",
		Author:  &dto.User{ID: "member-1"},
	})
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v), want handled without error", handled, err)
	}
	if sender.kind != "group" || sender.target != "group-1" {
		t.Errorf("reply target = (%s, %s), want group/group-1", sender.kind, sender.target)
	}
	if !strings.Contains(sender.content, "⚪ 未找到记录") {
		t.Errorf("reply = %q, missing not-found status", sender.content)
	}
}

func TestWhitelistHandler_RenderAllRecords(t *testing.T) {
	result := &lumiadmin.WhitelistStatusResponse{
		SteamID64: "76561198012345678",
		Items: []lumiadmin.WhitelistRecord{
			{Status: "pending", AppliedAt: "2026-08-03T12:00:00Z"},
			{Status: "rejected", RejectedAt: stringPtr("2026-08-04T12:00:00Z")},
			{Status: "revoked", RevokedAt: stringPtr("2026-08-05T12:00:00Z")},
		},
	}
	text := RenderWhitelistStatus(result)
	for _, want := range []string{"记录 1", "⏳ 待审核", "记录 2", "❌ 已拒绝", "原因：未填写拒绝原因", "记录 3", "⚠️ 已撤销", "时间：2026-08-05 20:00:00"} {
		if !strings.Contains(text, want) {
			t.Errorf("rendered text = %q, missing %q", text, want)
		}
	}
}

func TestWhitelistHandler_InvalidCommand(t *testing.T) {
	sender := &fakeSender{}
	h := NewWhitelistHandler(sender, &fakeQuerier{}, zap.NewNop())

	handled, err := h.Handle(context.Background(), &dto.Message{Content: "/wl", Author: &dto.User{ID: "openid-1"}})
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v), want usage response", handled, err)
	}
	if !strings.Contains(sender.content, "用法：/wl") {
		t.Errorf("usage reply = %q", sender.content)
	}
}

func TestWhitelistHandler_QueryErrorIsGeneric(t *testing.T) {
	sender := &fakeSender{}
	h := NewWhitelistHandler(sender, &fakeQuerier{err: errors.New("database password=secret")}, zap.NewNop())

	handled, err := h.Handle(context.Background(), &dto.Message{Content: "/wl 76561198012345678", Author: &dto.User{ID: "openid-1"}})
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v), want generic error reply", handled, err)
	}
	if sender.content != "查询白名单状态失败，请稍后重试。" {
		t.Errorf("error reply = %q", sender.content)
	}
}

func stringPtr(value string) *string { return &value }
