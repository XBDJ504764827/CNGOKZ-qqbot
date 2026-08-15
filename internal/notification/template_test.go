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
		{event.EventWhitelistRequestCreated, "新白名单申请", "[新白名单申请]"},
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

// TestRender_WhitelistRequest 验证白名单申请模板渲染（联调用例）。
func TestRender_WhitelistRequest(t *testing.T) {
	tmpl := NewTemplates()
	ev := testEvent(event.EventWhitelistRequestCreated, "新白名单申请", "玩家 张三 提交了白名单申请，等待审核", map[string]interface{}{
		"nickname":    "张三",
		"steamid64":   "76561198000000001",
		"contact":     "QQ 12345",
		"steam_level": 42,
		"ratings": map[string]interface{}{
			"kzt": 1560.5,
			"skz": 1300.0,
			"vnl": nil,
			"ovr": 1400.25,
		},
		"has_local_ban":  false,
		"has_global_ban": true,
		"has_active_ban": false,
		"profile_url":    "https://steamcommunity.com/profiles/76561198000000001",
	})

	title, content := tmpl.Render(ev)
	if title != "新白名单申请" {
		t.Errorf("title = %q, want 新白名单申请", title)
	}
	for _, want := range []string{
		"[新白名单申请]", "玩家: 张三", "SteamID: 76561198000000001", "联系方式: QQ 12345",
		"Steam等级: 42",
		"KZT rating: 1560.5", "SKZ rating: 1300", "VNL rating: -", "OVR rating: 1400.25",
		"本地封禁: 否", "全球封禁: 是", "未解封: 否",
		"Steam地址: https://steamcommunity.com/profiles/76561198000000001",
		"请管理员审核。",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing %q:\n%s", want, content)
		}
	}
}

// TestRender_WhitelistBanDetails 验证白名单模板在存在封禁详情时渲染原因。
func TestRender_WhitelistBanDetails(t *testing.T) {
	tmpl := NewTemplates()
	ev := testEvent(event.EventWhitelistRequestCreated, "新白名单申请", "m", map[string]interface{}{
		"steamid64":         "76561198000000001",
		"has_local_ban":     true,
		"local_ban_reason":  "外挂作弊",
		"has_global_ban":    false,
		"has_active_ban":    true,
		"active_ban_reason": "辱骂他人",
	})

	_, content := tmpl.Render(ev)
	for _, want := range []string{"本地封禁: 是", "本地封禁原因: 外挂作弊", "未解封: 是", "当前封禁原因: 辱骂他人"} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing %q:\n%s", want, content)
		}
	}
	// 无原因时不应输出“原因”占位行
	ev2 := testEvent(event.EventWhitelistRequestCreated, "新白名单申请", "m", map[string]interface{}{
		"steamid64":      "76561198000000002",
		"has_local_ban":  false,
		"has_global_ban": false,
	})
	_, content2 := tmpl.Render(ev2)
	if strings.Contains(content2, "本地封禁: -\n\nSteam") {
		// 空行处理允许；仅确保不输出“原因”占位行
	}
	for _, notWant := range []string{"本地封禁原因: -", "当前封禁原因: -"} {
		if strings.Contains(content2, notWant) {
			t.Errorf("content should not contain %q:\n%s", notWant, content2)
		}
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
