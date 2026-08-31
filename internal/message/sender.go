package message

import (
	"context"
	"fmt"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi/options"
	"go.uber.org/zap"
)

// MessageAPI 最小消息发送接口。
//
// 只声明发送所需的三个方法，隔离 botgo 庞大的完整接口，
// 便于单元测试 mock 与未来替换实现（如接入多平台）。
// openapi.OpenAPI 天然满足该接口。
type MessageAPI interface {
	PostMessage(ctx context.Context, channelID string, msg *dto.MessageToCreate, opt ...options.Option) (*dto.Message, error)
	PostGroupMessage(ctx context.Context, groupID string, msg dto.APIMessage, opt ...options.Option) (*dto.Message, error)
	PostC2CMessage(ctx context.Context, userID string, msg dto.APIMessage, opt ...options.Option) (*dto.Message, error)
}

// Sender 封装 QQ 消息发送能力（纯通知，无交互按钮）。
//
// 未来 LumiAdmin 可通过 POST /api/message/send 调用本组件：
//   - 频道消息（SendChannelMessage）
//   - 群消息（SendGroupMessage）
//   - 私聊 / 管理员通知（SendC2CMessage）
type Sender struct {
	api    MessageAPI
	logger *zap.Logger
}

// NewSender 构建消息发送器（依赖注入：最小发送接口 + 日志）。
func NewSender(api MessageAPI, logger *zap.Logger) *Sender {
	return &Sender{api: api, logger: logger}
}

// SendChannelMessage 发送 QQ 频道消息。
func (s *Sender) SendChannelMessage(ctx context.Context, channelID, content string) (*dto.Message, error) {
	if err := validateTarget(channelID); err != nil {
		return nil, fmt.Errorf("发送频道消息: %w", err)
	}
	if err := validateContent(content); err != nil {
		return nil, fmt.Errorf("发送频道消息: %w", err)
	}

	msg, err := s.api.PostMessage(ctx, channelID, &dto.MessageToCreate{Content: content})
	if err != nil {
		s.logger.Error("发送频道消息失败", zap.String("channel_id", channelID), zap.Error(err))
		return nil, err
	}
	s.logger.Info("频道消息发送成功",
		zap.String("channel_id", channelID),
		zap.String("message_id", msg.ID),
	)
	return msg, nil
}

// SendGroupMessage 发送 QQ 群消息（groupID 为群 openid）。
func (s *Sender) SendGroupMessage(ctx context.Context, groupID, content string) (*dto.Message, error) {
	if err := validateTarget(groupID); err != nil {
		return nil, fmt.Errorf("发送群消息: %w", err)
	}
	if err := validateContent(content); err != nil {
		return nil, fmt.Errorf("发送群消息: %w", err)
	}

	msg, err := s.api.PostGroupMessage(ctx, groupID, &dto.MessageToCreate{Content: content})
	if err != nil {
		s.logger.Error("发送群消息失败", zap.String("group_id", groupID), zap.Error(err))
		return nil, err
	}
	s.logger.Info("群消息发送成功",
		zap.String("group_id", groupID),
		zap.String("message_id", msg.ID),
	)
	return msg, nil
}

// SendC2CMessage 发送私聊消息（userID 为 C2C openid，用于管理员通知）。
// 纯文本通知，不附带任何按钮/键盘，保证 QQ 内用户无需（且无法）进行交互操作。
func (s *Sender) SendC2CMessage(ctx context.Context, userID, content string) (*dto.Message, error) {
	if err := validateTarget(userID); err != nil {
		return nil, fmt.Errorf("发送私聊消息: %w", err)
	}
	if err := validateContent(content); err != nil {
		return nil, fmt.Errorf("发送私聊消息: %w", err)
	}

	msg := &dto.MessageToCreate{Content: content}
	sent, err := s.api.PostC2CMessage(ctx, userID, msg)
	if err != nil {
		s.logger.Error("发送私聊消息失败", zap.String("user_id", userID), zap.Error(err))
		return nil, err
	}
	s.logger.Info("私聊消息发送成功",
		zap.String("user_id", userID),
		zap.String("message_id", sent.ID),
	)
	return sent, nil
}

// validateTarget 校验接收方 ID 非空。
func validateTarget(target string) error {
	if target == "" {
		return fmt.Errorf("接收方 ID 不能为空")
	}
	return nil
}

// validateContent 校验消息内容非空。
func validateContent(content string) error {
	if content == "" {
		return fmt.Errorf("消息内容不能为空")
	}
	return nil
}
