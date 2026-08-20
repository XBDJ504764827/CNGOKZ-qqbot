package command

import (
	"context"
	"strings"
	"testing"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/lumiadmin"
)

type fakeBanQuerier struct {
	result *lumiadmin.BanStatusResponse
	err    error
	input  string
}

func (f *fakeBanQuerier) QueryBanStatus(_ context.Context, input string) (*lumiadmin.BanStatusResponse, error) {
	f.input = input
	return f.result, f.err
}

func TestBanHandler_HandlePrivate(t *testing.T) {
	sender := &fakeSender{}
	querier := &fakeBanQuerier{result: &lumiadmin.BanStatusResponse{
		SteamID64: "76561198012345678",
		LocalBans: []lumiadmin.LocalBanRecord{{
			Status:    "active",
			BanType:   "steam",
			Reason:    "作弊",
			CreatedAt: "2026-08-20T04:00:00Z",
		}},
		GlobalBans: []lumiadmin.GlobalBanRecord{{
			BanType:   "cheat",
			Notes:     stringPtr("KZTimer Global 的 notes"),
			CreatedOn: stringPtr("2026-08-20T05:00:00Z"),
			ExpiresOn: stringPtr("9999-12-31T00:00:00Z"),
		}},
	}}
	h := NewBanHandler(sender, querier, zap.NewNop())

	handled, err := h.Handle(context.Background(), &dto.Message{
		Content: "/ban 76561198012345678",
		Author:  &dto.User{ID: "openid-1"},
	})
	if err != nil || !handled {
		t.Fatalf("Handle() = (%v, %v)", handled, err)
	}
	if querier.input != "76561198012345678" {
		t.Errorf("querier input = %q", querier.input)
	}
	for _, want := range []string{
		"网站封禁",
		"状态：🔒 封禁中",
		"原因：作弊",
		"全球封禁",
		"状态：🌍 生效中",
		"原因：KZTimer Global 的 notes",
		"到期时间：永久",
		"封禁时间：2026-08-20 13:00:00",
	} {
		if !strings.Contains(sender.content, want) {
			t.Errorf("reply = %q, missing %q", sender.content, want)
		}
	}
}

func TestRenderBanStatusHistoryAndManualUnban(t *testing.T) {
	result := &lumiadmin.BanStatusResponse{
		SteamID64: "76561198012345678",
		LocalBans: []lumiadmin.LocalBanRecord{
			{Status: "inactive", BanType: "ip", Reason: "辱骂", CreatedAt: "2026-08-01T00:00:00Z", RemovedAt: stringPtr("2026-08-02T00:00:00Z"), RemovedReason: stringPtr("误封")},
			{Status: "active", BanType: "steam", Reason: "作弊", CreatedAt: "2026-08-03T00:00:00Z"},
		},
		GlobalBans: []lumiadmin.GlobalBanRecord{{
			BanType:        "bhop_hack",
			IsExpired:      true,
			ManualUnbanned: true,
			Notes:          stringPtr("旧记录"),
			CreatedOn:      stringPtr("2026-08-01T00:00:00Z"),
		}},
	}
	text := RenderBanStatus(result)
	for _, want := range []string{
		"记录 1",
		"✅ 已解除",
		"解封原因：误封",
		"记录 2",
		"🔒 封禁中",
		"全球封禁状态：🌍 全球封禁中",
		"备注：本地服务器已解除该封禁",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("rendered text = %q, missing %q", text, want)
		}
	}
}

func TestRenderBanStatusNoBans(t *testing.T) {
	text := RenderBanStatus(&lumiadmin.BanStatusResponse{SteamID64: "76561198012345678"})
	if !strings.Contains(text, "网站封禁：✅ 未发现封禁") || !strings.Contains(text, "全球封禁：✅ 未发现封禁") {
		t.Errorf("rendered text = %q", text)
	}
}
