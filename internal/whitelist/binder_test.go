package whitelist

import (
	"testing"

	"github.com/XBDJ504764827/LumiBot/internal/lumiadmin"
)

func TestReplyText(t *testing.T) {
	cases := []struct {
		label   string
		outcome lumiadmin.BindOutcome
		want    string
	}{
		{"bound", lumiadmin.BindOutcome{Result: "bound"}, "绑定成功！请返回网站继续填写申请理由并提交白名单申请。"},
		{"bound_already", lumiadmin.BindOutcome{Result: "bound", Already: true}, "该账号已完成过绑定，可直接返回网站继续申请白名单。"},
		{"invalid", lumiadmin.BindOutcome{Result: "invalid_code"}, "验证码无效或已过期（有效期 5 分钟），请回到网站重新生成。"},
		{"consumed", lumiadmin.BindOutcome{Result: "consumed"}, "该验证码已被使用，请回到网站重新生成。"},
		{"limit", lumiadmin.BindOutcome{Result: "limit_reached"}, "该 QQ 绑定的 Steam 账号数量已达上限，请联系管理员处理。"},
		{"group", lumiadmin.BindOutcome{Result: "group_denied"}, "当前 QQ 群不在允许绑定的群列表内，请联系管理员。"},
		{"disabled", lumiadmin.BindOutcome{Result: "disabled"}, "当前未开启 QQ 绑定，请稍后再试。"},
		{"rejected_msg", lumiadmin.BindOutcome{Result: "rejected", Message: "该 Steam 账号已绑定其他 QQ，如需换绑请联系管理员解绑"}, "该 Steam 账号已绑定其他 QQ，如需换绑请联系管理员解绑"},
		{"unknown", lumiadmin.BindOutcome{Result: "something"}, "绑定失败，请联系管理员。"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			if got := replyText(tc.outcome); got != tc.want {
				t.Errorf("replyText(%q) = %q, want %q", tc.outcome.Result, got, tc.want)
			}
		})
	}
}

func TestCodePattern(t *testing.T) {
	valid := []string{
		"绑定 WL-7KQ2XA",
		"WL7KQ2XA",
		"@机器人 WL-ABCD23",
		"wl-abcdef",
	}
	for _, s := range valid {
		if m := codePattern.FindStringSubmatch(s); m == nil {
			t.Errorf("expected %q to match", s)
		}
	}

	invalid := []string{
		"你好",
		"WL-123",
		"WL-123456",
	}
	for _, s := range invalid {
		if m := codePattern.FindStringSubmatch(s); m != nil {
			t.Errorf("expected %q not to match, got %v", s, m)
		}
	}
}

func TestBinderDisabledWithoutClient(t *testing.T) {
	b := NewBinder(nil, nil, nil)
	if b.Enabled() {
		t.Fatal("binder with nil client should be disabled")
	}
}
