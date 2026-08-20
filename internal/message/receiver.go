package message

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// Receiver 处理收到的用户消息。
//
// 第一阶段：结构化日志输出；后续指令系统在此扩展。
// 可注入 OnUserText 回调，用于拦截私聊/消息并交给上层业务（如 QQ 审批的拒绝原因）。
type Receiver struct {
	logger *zap.Logger
	// OnCommand 可选的指令处理回调，返回 handled=true 表示已消费。
	// 指令优先于 OnUserText，避免显式指令被审批文本流程误消费。
	OnCommand func(ctx context.Context, msg *dto.Message) (handled bool, err error)
	// OnUserText 可选的用户文本回调（openid, content），返回 handled=true 表示已消费。
	OnUserText func(ctx context.Context, openid, content string) (handled bool, err error)
}

// NewReceiver 构建消息接收处理器。
func NewReceiver(logger *zap.Logger) *Receiver {
	return &Receiver{logger: logger}
}

// ReceiveMessage 处理收到的用户消息。
func (r *Receiver) ReceiveMessage(ctx context.Context, msg *dto.Message) error {
	openid := authorID(msg.Author)

	// 显式指令优先处理，例如 /bind 不应被审批拒绝原因流程消费。
	if r.OnCommand != nil {
		handled, err := r.OnCommand(ctx, msg)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}
	}

	// 若注册了用户文本回调，再交给上层业务判断是否消费（例如 QQ 审批拒绝原因）。
	if r.OnUserText != nil {
		handled, err := r.OnUserText(ctx, openid, msg.Content)
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
