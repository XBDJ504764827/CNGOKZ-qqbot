package message

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// Receiver 处理收到的用户消息。
//
// QQ 仅作为通知渠道（纯文本推送）：收到的用户消息只记录日志，不做任何业务处理。
// 不允许在 QQ 内通过指令/文本进行审批等操作。
type Receiver struct {
	logger *zap.Logger
}

// NewReceiver 构建消息接收处理器。
func NewReceiver(logger *zap.Logger) *Receiver {
	return &Receiver{logger: logger}
}

// ReceiveMessage 处理收到的用户消息：仅记录日志，不消费、不响应。
func (r *Receiver) ReceiveMessage(ctx context.Context, msg *dto.Message) error {
	openid := authorID(msg.Author)

	r.logger.Info("message received",
		zap.String("user_id", openid),
		zap.String("guild_id", msg.GuildID),
		zap.String("channel_id", msg.ChannelID),
		zap.String("message_id", msg.ID),
		zap.String("content", msg.Content),
	)
	return nil
}

// authorID 安全获取发送者 ID（Author 可能为空）。
func authorID(u *dto.User) string {
	if u == nil {
		return ""
	}
	return u.ID
}
