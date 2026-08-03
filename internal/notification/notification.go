// Package notification 实现通知系统：将内部事件转换为管理员可读的 QQ 通知。
//
// 流程：
//
//	Event → 规则判断（internal/rule）→ 模板渲染（template.go）
//	→ 冷却防刷（cooldown.go）→ QQ 发送（通过 message.Sender，依赖倒置接口）
//
// 与 bot 层完全解耦：不直接操作 QQ SDK，仅依赖 message 包的发送能力接口。
package notification

import (
	"time"

	"github.com/google/uuid"

	"github.com/XBDJ504764827/LumiBot/internal/event"
)

// Channel 通知发送渠道。
type Channel string

const (
	// ChannelQQPrivate 私聊通知（管理员 QQ，推荐用于告警）。
	ChannelQQPrivate Channel = "QQ_PRIVATE"
	// ChannelQQChannel 频道通知（QQ 频道内指定子频道）。
	ChannelQQChannel Channel = "QQ_CHANNEL"
	// ChannelEmail 邮件通知（预留）。
	ChannelEmail Channel = "EMAIL"
	// ChannelWebhook Webhook 通知（预留）。
	ChannelWebhook Channel = "WEBHOOK"
)

// Notification 通知模型。
type Notification struct {
	// ID 通知唯一标识。
	ID string
	// EventID 来源事件 ID（关联审计）。
	EventID string
	// Channel 发送渠道。
	Channel Channel
	// Level 通知等级（来自事件级别）。
	Level string
	// Title 通知标题。
	Title string
	// Content 渲染后的通知内容（发送给 QQ 的正文）。
	Content string
	// CreatedAt 通知创建时间。
	CreatedAt time.Time
}

// NewNotification 由事件构建通知（ID / 时间戳自动生成）。
//
// channel 默认私聊（管理员通知），后续可按规则/模板扩展渠道选择。
func NewNotification(ev event.Event, channel Channel, title, content string) Notification {
	return Notification{
		ID:        uuid.NewString(),
		EventID:   ev.ID,
		Channel:   channel,
		Level:     ev.Level,
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}
}
