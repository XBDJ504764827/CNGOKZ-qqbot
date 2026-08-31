package event

// 本文件集中管理事件协议常量（事件类型 / 来源），避免字符串散落各处。
// 对外协议与 pkg/sdk 保持一致，完整说明见 docs/API.md。

// 事件类型（EventType）常量 —— 外部系统约定的协议值。
const (
	EventSystemWarning           = "SYSTEM_WARNING"            // 系统警告
	EventServerOffline           = "SERVER_OFFLINE"            // 游戏服务器离线
	EventServerOnline            = "SERVER_ONLINE"             // 游戏服务器恢复
	EventForumReportCreated      = "FORUM_REPORT_CREATED"      // 论坛新举报
	EventAdminAction             = "ADMIN_ACTION"              // 管理操作
	EventWhitelistRequestCreated = "WHITELIST_REQUEST_CREATED" // LumiAdmin 白名单新申请
	EventWhitelistAutoApproved   = "WHITELIST_AUTO_APPROVED"   // 低风险白名单自动通过
)

// 事件来源（Source）常量 —— 记录事件来自哪个外部系统。
const (
	SourceLumiForum   = "LumiForum"   // 论坛系统
	SourceLumiAdmin   = "LumiAdmin"   // 管理后台
	SourceGameMonitor = "GameMonitor" // 游戏服务器监控（CS / KZ）
	SourceQQ          = "QQ"          // QQ 平台自身事件
)

// eventTitles 事件类型 → 中文标题（用于 SDK 提示 / 文档）。
var eventTitles = map[string]string{
	EventSystemWarning:           "系统警告",
	EventServerOffline:           "服务器离线",
	EventServerOnline:            "服务器恢复",
	EventForumReportCreated:      "论坛举报",
	EventAdminAction:             "管理操作",
	EventWhitelistRequestCreated: "新白名单申请",
	EventWhitelistAutoApproved:   "白名单自动通过",
}

// KnownEventTypes 返回全部已知事件类型。
func KnownEventTypes() []string {
	types := make([]string, 0, len(eventTitles))
	for eventType := range eventTitles {
		types = append(types, eventType)
	}
	return types
}

// IsKnownEventType 判断事件类型是否已注册。
func IsKnownEventType(eventType string) bool {
	_, ok := eventTitles[eventType]
	return ok
}

// EventTypeTitle 返回事件类型的中文标题，未知类型返回原值。
func EventTypeTitle(eventType string) string {
	if title, ok := eventTitles[eventType]; ok {
		return title
	}
	return eventType
}
