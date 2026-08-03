// Package rule 实现通知规则系统：根据事件类型与级别判断是否需要通知。
//
// 规则为数据驱动（内置默认规则表），事件是否通知由规则决定，
// 业务层（notification.Service）不硬编码判断逻辑。
package rule

import (
	"time"

	"github.com/XBDJ504764827/LumiBot/internal/event"
)

// Rule 单条通知规则。
type Rule struct {
	// EventType 事件类型。
	EventType string
	// MinLevel 通知所需的最低事件级别（info < warning < error < critical）。
	// 事件级别达到或高于该级别才通知。
	MinLevel string
	// Enabled 是否启用该规则。
	Enabled bool
	// Cooldown 冷却时间：同一事件类型在窗口内的重复事件只通知一次。
	// 0 表示不限制。
	Cooldown time.Duration
}

// Rules 通知规则集合。
type Rules struct {
	byType map[string]Rule
}

// New 构建规则集合（内置默认规则表）。
//
// defaultCooldown 为全局默认冷却时间（来自 NOTICE_COOLDOWN 配置），
// 应用到内置的关键事件规则。
func New(defaultCooldown time.Duration) *Rules {
	rules := []Rule{
		{
			EventType: event.EventServerOffline,
			MinLevel:  event.LevelInfo,
			Enabled:   true,
			Cooldown:  defaultCooldown,
		},
		{
			EventType: event.EventServerOnline,
			MinLevel:  event.LevelInfo,
			Enabled:   true,
			Cooldown:  defaultCooldown,
		},
		{
			EventType: event.EventSystemWarning,
			MinLevel:  event.LevelWarning,
			Enabled:   true,
			Cooldown:  defaultCooldown,
		},
		{
			EventType: event.EventForumReportCreated,
			MinLevel:  event.LevelWarning,
			Enabled:   true,
			Cooldown:  defaultCooldown,
		},
		{
			EventType: event.EventAdminAction,
			MinLevel:  event.LevelError,
			Enabled:   false, // 管理操作默认不通知（审计用途），未来按需开启
		},
	}

	r := &Rules{byType: make(map[string]Rule, len(rules))}
	for _, rule := range rules {
		r.byType[rule.EventType] = rule
	}
	return r
}

// ShouldNotify 判断事件是否应发送通知，返回命中规则。
// 未配置规则 / 规则禁用 / 级别不足 均不通知。
func (r *Rules) ShouldNotify(ev event.Event) (bool, Rule) {
	rule, ok := r.byType[ev.EventType]
	if !ok {
		return false, Rule{}
	}
	if !rule.Enabled {
		return false, rule
	}
	if levelRank(ev.Level) < levelRank(rule.MinLevel) {
		return false, rule
	}
	return true, rule
}

// levelRank 事件级别排序（未知级别视为 info）。
func levelRank(level string) int {
	switch level {
	case event.LevelInfo:
		return 1
	case event.LevelWarning:
		return 2
	case event.LevelError:
		return 3
	case event.LevelCritical:
		return 4
	default:
		return 1
	}
}
