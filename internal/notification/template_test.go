package notification

import (
	"strings"
	"testing"
	"time"

	"github.com/XBDJ504764827/LumiBot/internal/event"
)

func testEvent(eventType, title, message string, data map[string]interface{}) event.Event {
	return event.Event{
		ID:        "evt-test",
		Source:    "GameMonitor",
		EventType: eventType,
		Level:     event.LevelCritical,
		Timestamp: time.Date(2026, 8, 3, 16, 0, 0, 0, time.Local),
		Title:     title,
		Message:   message,
		Data:      data,
	}
}

func TestRender_ServerOffline(t *testing.T) {
	tmpl := NewTemplates()
	ev := testEvent(event.EventServerOffline, "服务器离线", "KZ服务器01停止响应", map[string]interface{}{
		"server": "KZ-01",
		"status": "离线",
	})

	title, content := tmpl.Render(ev)
	if title != "服务器异常" {
		t.Errorf("title = %q, want 服务器异常", title)
	}

	for _, want := range []string{"[服务器异常]", "服务器: KZ-01", "状态: 离线", "时间: 2026-08-03 16:00", "请管理员处理。"} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing %q:\n%s", want, content)
		}
	}
}

// TestRender_AllEventTypes 验证所有内置事件类型模板均可渲染。
func TestRender_AllEventTypes(t *testing.T) {
	tmpl := NewTemplates()
	cases := []struct {
		eventType string
		title     string
		wantTag   string
	}{
		{event.EventServerOffline, "服务器异常", "[服务器异常]"},
		{event.EventServerOnline, "服务器恢复", "[服务器恢复]"},
		{event.EventSystemWarning, "系统警告", "[系统警告]"},
		{event.EventForumReportCreated, "论坛举报", "[论坛举报]"},
		{event.EventAdminAction, "管理操作", "[管理操作]"},
		{event.EventWhitelistRequestCreated, "新白名单申请", "📩 白名单申请"},
		{event.EventWhitelistAutoApproved, "白名单自动通过", "[白名单自动通过]"},
	}
	for _, tc := range cases {
		ev := testEvent(tc.eventType, "t", "m", map[string]interface{}{"server": "KZ-01"})
		title, content := tmpl.Render(ev)
		if title != tc.title {
			t.Errorf("[%s] title = %q, want %q", tc.eventType, title, tc.title)
		}
		if !strings.Contains(content, tc.wantTag) {
			t.Errorf("[%s] content missing %q:\n%s", tc.eventType, tc.wantTag, content)
		}
	}
}

// TestRender_WhitelistRequest 验证白名单申请模板渲染（联调用例 - 用户指定格式）。
func TestRender_WhitelistRequest(t *testing.T) {
	tmpl := NewTemplates()
	ev := testEvent(event.EventWhitelistRequestCreated, "新白名单申请", "玩家 张三 提交了白名单申请，等待审核", map[string]interface{}{
		"nickname_show":     "张三",
		"steamid64":         "76561198000000001",
		"risk_display":      "🔴 高风险",
		"ban_flags":         "❌ 全球封禁",
		"ban_reason":        "bhop_hack",
		"auto_approve_text": "风险玩家等待管理员进行手动审核",
		"detail_url":        "https://admin.example.com/whitelist",
	})

	title, content := tmpl.Render(ev)
	if title != "新白名单申请" {
		t.Errorf("title = %q, want 新白名单申请", title)
	}
	for _, want := range []string{
		"📩 白名单申请",
		"👤 张三",
		"🆔 76561198000000001",
		"风险：🔴 高风险",
		"封禁：❌ 全球封禁",
		"违规：",
		"bhop_hack",
		"自动审核：风险玩家等待管理员进行手动审核",
		"时间：",
		"🔗 点击查看详情：https://admin.example.com/whitelist",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing %q:\n%s", want, content)
		}
	}
}

// TestRender_WhitelistRequest_NoBanReason 验证无封禁原因时隐藏原因区块。
func TestRender_WhitelistRequest_NoBanReason(t *testing.T) {
	tmpl := NewTemplates()
	ev := testEvent(event.EventWhitelistRequestCreated, "新白名单申请", "", map[string]interface{}{
		"nickname_show":     "李四",
		"steamid64":         "76561198000000002",
		"risk_display":      "🟢 低风险",
		"ban_flags":         "无",
		"ban_reason":        "-",
		"auto_approve_text": "3小时",
		"detail_url":        "",
	})

	_, content := tmpl.Render(ev)
	if strings.Contains(content, "违规：") {
		t.Errorf("无封禁原因时不应显示违规区块:\n%s", content)
	}
	if !strings.Contains(content, "封禁：无") {
		t.Errorf("content missing 封禁：无:\n%s", content)
	}
}

// TestRender_WhitelistAutoApproved 验证低风险自动通过事件模板渲染。
func TestRender_WhitelistAutoApproved(t *testing.T) {
	tmpl := NewTemplates()
	ev := testEvent(event.EventWhitelistAutoApproved, "白名单自动通过", "玩家 张三 的低风险白名单申请已自动通过", map[string]interface{}{
		"nickname":  "张三",
		"steamid64": "76561198000000001",
		"hours":     3,
	})

	title, content := tmpl.Render(ev)
	if title != "白名单自动通过" {
		t.Errorf("title = %q, want 白名单自动通过", title)
	}
	for _, want := range []string{
		"[白名单自动通过]", "玩家: 张三", "SteamID: 76561198000000001",
		"申请满 3 小时无人审核", "系统已自动通过",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing %q:\n%s", want, content)
		}
	}
}

// TestRender_WhitelistBanDetails 验证白名单模板在存在封禁详情时渲染违规原因。
func TestRender_WhitelistBanDetails(t *testing.T) {
	tmpl := NewTemplates()
	ev := testEvent(event.EventWhitelistRequestCreated, "新白名单申请", "m", map[string]interface{}{
		"nickname_show":     "张三",
		"steamid64":         "76561198000000001",
		"risk_display":      "🔴 高风险",
		"ban_flags":         "❌ 全球封禁 / 未解封",
		"ban_reason":        "bhop_hack",
		"auto_approve_text": "风险玩家等待管理员进行手动审核",
	})

	_, content := tmpl.Render(ev)
	for _, want := range []string{
		"封禁：❌ 全球封禁 / 未解封",
		"违规：",
		"bhop_hack",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing %q:\n%s", want, content)
		}
	}
	// 无原因时不应输出原因区块
	ev2 := testEvent(event.EventWhitelistRequestCreated, "新白名单申请", "m", map[string]interface{}{
		"nickname_show":     "李四",
		"steamid64":         "76561198000000002",
		"risk_display":      "🟢 低风险",
		"ban_flags":         "无",
		"ban_reason":        "-",
		"auto_approve_text": "3小时",
	})
	_, content2 := tmpl.Render(ev2)
	if strings.Contains(content2, "违规：") {
		t.Errorf("无封禁原因时不应显示原因区块:\n%s", content2)
	}
}

// TestRender_MissingData 验证 data 字段缺失时渲染不报错（输出 "-"）。
func TestRender_MissingData(t *testing.T) {
	tmpl := NewTemplates()
	ev := testEvent(event.EventServerOffline, "t", "m", nil)

	_, content := tmpl.Render(ev)
	if !strings.Contains(content, "服务器: -") {
		t.Errorf("missing data should render as '-':\n%s", content)
	}
}

// TestRender_UnknownEventType 验证未注册事件类型回退通用模板。
func TestRender_UnknownEventType(t *testing.T) {
	tmpl := NewTemplates()
	ev := testEvent("UNKNOWN_EVENT", "未知事件", "m", nil)

	title, content := tmpl.Render(ev)
	if title != "未知事件" {
		t.Errorf("fallback title = %q, want %q", title, "未知事件")
	}
	if !strings.Contains(content, "[未知事件]") {
		t.Errorf("fallback content missing title:\n%s", content)
	}
}
