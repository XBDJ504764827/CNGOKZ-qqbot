package message

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// Receiver 处理收到的用户消息。
//
// QQ 主要作为通知渠道（纯文本推送），设计上不允许通过 QQ 文本进行审批等业务操作；
// 唯一保留的指令是 /bind（获取用户 QQ OpenID，供自助绑定），由 OnCommand 回调分发。
type Receiver struct {
	logger *zap.Logger
	// OnCommand 可选的指令处理回调，返回 handled=true 表示指令已被消费。
	OnCommand func(ctx context.Context, msg *dto.Message) (handled bool, err error)
}

// NewReceiver 构建消息接收处理器。
func NewReceiver(logger *zap.Logger) *Receiver {
	return &Receiver{logger: logger}
}

// ReceiveMessage 处理收到的用户消息：指令优先，未消费的消息仅记录日志。
func (r *Receiver) ReceiveMessage(ctx context.Context, msg *dto.Message) error {
	openid := authorID(msg.Author)

	// 显式指令优先处理，例如 /bind 不应落入普通文本日志流程。
	if r.OnCommand != nil {
		handled, err := r.OnCommand(ctx, msg)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}
	}

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
