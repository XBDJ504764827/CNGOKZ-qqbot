package event

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// 事件级别（Level）。
const (
	LevelInfo     = "info"     // 普通信息
	LevelWarning  = "warning"  // 警告，需要关注
	LevelError    = "error"    // 错误，需要处理
	LevelCritical = "critical" // 严重，需要立即处理
)

// 事件类型与来源常量集中管理于 catalog.go。

// Event 统一事件模型，是外部系统与 LumiBot 之间的数据协议。
//
// JSON 示例：
//
//	{
//	  "id": "uuid",
//	  "source": "LumiForum",
//	  "event_type": "FORUM_REPORT_CREATED",
//	  "level": "warning",
//	  "timestamp": "2026-08-03T12:00:00+08:00",
//	  "title": "新举报",
//	  "message": "发现违规帖子",
//	  "data": {}
//	}
type Event struct {
	// ID 事件唯一标识，可省略（服务端自动生成 UUID）。
	ID string `json:"id"`
	// Source 事件来源系统，必填。
	Source string `json:"source"`
	// EventType 事件类型，必填，取值见 docs/API.md。
	EventType string `json:"event_type"`
	// Level 事件级别：info / warning / error / critical，可省略（默认 info）。
	Level string `json:"level"`
	// Timestamp 事件发生时间，可省略（默认服务端接收时间）。
	Timestamp time.Time `json:"timestamp"`
	// Title 事件标题，用于通知展示。
	Title string `json:"title"`
	// Message 事件描述。
	Message string `json:"message"`
	// Data 附加业务数据（任意 JSON 对象）。
	Data map[string]interface{} `json:"data,omitempty"`
}

// Validate 校验事件必填字段与取值合法性。
func (e *Event) Validate() error {
	if strings.TrimSpace(e.Source) == "" {
		return fmt.Errorf("source 不能为空")
	}
	if strings.TrimSpace(e.EventType) == "" {
		return fmt.Errorf("event_type 不能为空")
	}
	if e.Level != "" && !validLevel(e.Level) {
		return fmt.Errorf("level 取值非法 %q，支持 %s", e.Level,
			strings.Join([]string{LevelInfo, LevelWarning, LevelError, LevelCritical}, " / "))
	}
	return nil
}

// Normalize 补全可省略字段：ID（生成 UUID）、Level（默认 info）、
// Timestamp（默认当前时间）、Data（初始化为空 map）。
// 应在 Validate 通过后调用。
func (e *Event) Normalize() {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.Level == "" {
		e.Level = LevelInfo
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	if e.Data == nil {
		e.Data = map[string]interface{}{}
	}
}

// validLevel 判断事件级别是否合法。
func validLevel(level string) bool {
	switch level {
	case LevelInfo, LevelWarning, LevelError, LevelCritical:
		return true
	default:
		return false
	}
}
