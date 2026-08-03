// Package notification 负责事件通知的发送（QQ 通知等）。
//
// 当前阶段：监听关键事件并记录日志（含事件全量字段），
// 为未来接入 QQ 通知 / Webhook 预留处理位置。
package notification

import (
	"context"

	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/event"
)

// Handler 通知处理器：订阅关键事件类型。
//
// 当前监听（注册方式见 cmd/bot/main.go）：
//   - SYSTEM_WARNING         系统警告
//   - SERVER_OFFLINE         游戏服务器离线
//   - FORUM_REPORT_CREATED   论坛新举报
//
// 后续阶段：注入 message.Sender，将事件渲染为 QQ 通知消息发送给管理员。
type Handler struct {
	logger *zap.Logger
}

// NewHandler 构建通知处理器。
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{logger: logger}
}

// HandleEvent 处理事件：记录结构化日志，标记待发送 QQ 通知。
func (h *Handler) HandleEvent(_ context.Context, ev event.Event) error {
	h.logger.Info("收到事件，准备发送QQ通知（通知发送待后续阶段实现）",
		zap.String("event_id", ev.ID),
		zap.String("source", ev.Source),
		zap.String("event_type", ev.EventType),
		zap.String("level", ev.Level),
		zap.String("title", ev.Title),
		zap.String("message", ev.Message),
	)
	return nil
}
