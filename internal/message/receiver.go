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
	// OnUserText 可选的用户文本回调（openid, content），返回 handled=true 表示已消费。
	// 审批服务使用该回调消费等待中的拒绝原因。
	OnUserText func(ctx context.Context, openid, content string) (handled bool, err error)
	// OnCommand 可选的指令回调。它在 OnUserText 未消费消息后执行，
	// 使审批文本与普通指令可以共存。
	OnCommand func(ctx context.Context, msg *dto.Message) (handled bool, err error)
}

// NewReceiver 构建消息接收处理器。
func NewReceiver(logger *zap.Logger) *Receiver {
	return &Receiver{logger: logger}
}

// ReceiveMessage 处理收到的用户消息。
func (r *Receiver) ReceiveMessage(ctx context.Context, msg *dto.Message) error {
	openid := authorID(msg.Author)

	// 先处理显式指令，避免管理员等待拒绝原因时发送 /wl 被当作拒绝理由。
	if r.OnCommand != nil {
		handled, err := r.OnCommand(ctx, msg)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}
	}

	// 未匹配指令时，再交给上层文本回调（例如 QQ 审批拒绝原因）。
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
