package command

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
)

// MessageSender 是指令回复所需的最小 QQ 消息发送接口。
type MessageSender interface {
	SendC2CMessage(ctx context.Context, userID, content string) (*dto.Message, error)
	SendGroupMessage(ctx context.Context, groupID, content string) (*dto.Message, error)
	SendChannelMessage(ctx context.Context, channelID, content string) (*dto.Message, error)
}

func authorID(msg *dto.Message) string {
	if msg == nil || msg.Author == nil {
		return ""
	}
	return msg.Author.ID
}
