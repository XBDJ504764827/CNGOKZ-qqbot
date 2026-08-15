// Package sdk 提供 LumiBot 事件上报 Go SDK（供 CNGOKZ 生态外部系统接入）。
//
// 面向调用方：
//   - LumiAdmin     （管理员操作 / 系统异常 / 用户管理事件）
//   - LumiForum     （举报 / 内容审核 / 用户异常行为）
//   - GameMonitor   （CS / KZ 服务器离线、玩家异常、性能告警）
//
// 当前阶段：仅定义接口与协议常量（与 internal/event 保持一致），
// HTTP 客户端实现将在后续版本提供，接口设计已稳定，不影响未来扩展。
package sdk

import (
	"context"
	"time"
)

// Client 事件上报客户端接口。
//
// 外部系统通过 HTTP POST /api/v1/events 上报事件，
// 本接口抽象该调用，后续提供 HTTP 实现（NewHTTPClient）。
type Client interface {
	// SendEvent 上报单个事件，返回服务端处理结果。
	SendEvent(ctx context.Context, req *SendEventRequest) (*SendEventResponse, error)
}

// SendEventRequest 事件上报请求（对应 API Body，字段语义见 docs/API.md）。
type SendEventRequest struct {
	// EventID 事件唯一标识，可省略（服务端生成）。
	EventID string
	// Source 事件来源系统（必填），使用本包 Source* 常量。
	Source string
	// EventType 事件类型（必填），使用本包 Event* 常量。
	EventType string
	// Level 事件级别：info / warning / error / critical，可省略（默认 info）。
	Level string
	// Timestamp 事件发生时间，可省略（默认服务端接收时间）。
	Timestamp time.Time
	// Title 事件标题。
	Title string
	// Message 事件描述。
	Message string
	// Data 业务附加数据（任意 JSON 对象）。
	Data map[string]interface{}
}

// SendEventResponse 事件上报响应。
type SendEventResponse struct {
	// Success 是否受理成功。
	Success bool
	// EventID 服务端生成/确认的事件 ID。
	EventID string
}

// 事件类型常量（协议值，与 internal/event/catalog.go 保持一致）。
const (
	EventSystemWarning           = "SYSTEM_WARNING"            // 系统警告
	EventServerOffline           = "SERVER_OFFLINE"            // 游戏服务器离线
	EventServerOnline            = "SERVER_ONLINE"             // 游戏服务器恢复
	EventForumReportCreated      = "FORUM_REPORT_CREATED"      // 论坛新举报
	EventAdminAction             = "ADMIN_ACTION"              // 管理操作
	EventWhitelistRequestCreated = "WHITELIST_REQUEST_CREATED" // 白名单新申请
)

// 事件来源常量。
const (
	SourceLumiForum   = "LumiForum"
	SourceLumiAdmin   = "LumiAdmin"
	SourceGameMonitor = "GameMonitor"
	SourceQQ          = "QQ"
)

// 事件级别常量。
const (
	LevelInfo     = "info"
	LevelWarning  = "warning"
	LevelError    = "error"
	LevelCritical = "critical"
)

// 说明：HTTP 客户端实现（基于 http.Client 调用 POST /api/v1/events）
// 将在下一阶段提供，接口设计已稳定（见 Client / SendEventRequest），
// 不影响外部系统按本包协议先行接入。
