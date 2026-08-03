// Package handler 定义机器人消息处理入口。
//
// 第一阶段仅提供默认实现（结构化日志占位）；
// 第二阶段接入指令系统时，通过实现 Handler 接口即可扩展，
// 无需改动 bot 层与 cmd 层。
package handler

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// Handler 机器人消息统一处理接口。
type Handler interface {
	// HandleATMessage 频道内 @机器人 消息。
	HandleATMessage(ctx context.Context, data *dto.WSATMessageData) error
	// HandleGroupATMessage 群聊内 @机器人 消息。
	HandleGroupATMessage(ctx context.Context, data *dto.WSGroupATMessageData) error
	// HandleC2CMessage 私聊（C2C）消息。
	HandleC2CMessage(ctx context.Context, data *dto.WSC2CMessageData) error
}

// DefaultHandler Handler 的默认实现：仅记录结构化日志，占位等待业务接入。
type DefaultHandler struct {
	logger *zap.Logger
}

// NewDefaultHandler 构建默认消息处理器。
func NewDefaultHandler(logger *zap.Logger) *DefaultHandler {
	return &DefaultHandler{logger: logger}
}

// HandleATMessage 记录频道 @消息。
func (h *DefaultHandler) HandleATMessage(_ context.Context, data *dto.WSATMessageData) error {
	h.logger.Info("收到频道@消息（指令系统待接入）",
		zap.String("guild_id", data.GuildID),
		zap.String("channel_id", data.ChannelID),
		zap.String("author_id", authorID(data.Author)),
		zap.String("content", data.Content),
	)
	return nil
}

// HandleGroupATMessage 记录群聊 @消息。
func (h *DefaultHandler) HandleGroupATMessage(_ context.Context, data *dto.WSGroupATMessageData) error {
	h.logger.Info("收到群聊@消息（指令系统待接入）",
		zap.String("group_id", data.GroupID),
		zap.String("author_id", authorID(data.Author)),
		zap.String("content", data.Content),
	)
	return nil
}

// HandleC2CMessage 记录私聊消息。
func (h *DefaultHandler) HandleC2CMessage(_ context.Context, data *dto.WSC2CMessageData) error {
	h.logger.Info("收到私聊消息（指令系统待接入）",
		zap.String("author_id", authorID(data.Author)),
		zap.String("content", data.Content),
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
