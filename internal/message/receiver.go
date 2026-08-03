package message

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// Receiver 处理收到的用户消息。
//
// 第一阶段：结构化日志输出（用户ID、频道ID、消息内容），
// 为排查机器人事件提供依据；后续指令系统在此扩展。
type Receiver struct {
	logger *zap.Logger
}

// NewReceiver 构建消息接收处理器。
func NewReceiver(logger *zap.Logger) *Receiver {
	return &Receiver{logger: logger}
}

// ReceiveMessage 处理收到的用户消息。
//
// 当前实现记录结构化日志：
//
//	INFO  message received  {"user_id": "xxx", "channel_id": "xxx", "content": "hello"}
func (r *Receiver) ReceiveMessage(_ context.Context, msg *dto.Message) error {
	r.logger.Info("message received",
		zap.String("user_id", authorID(msg.Author)),
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
