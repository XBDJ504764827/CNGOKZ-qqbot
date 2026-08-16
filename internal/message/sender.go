package message

import (
	"context"
	"fmt"
	"strings"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/keyboard"
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

// Sender 封装 QQ 消息发送能力。
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
func (s *Sender) SendC2CMessage(ctx context.Context, userID, content string) (*dto.Message, error) {
	return s.SendC2CMessageWithKeyboard(ctx, userID, content, nil)
}

// SendC2CMessageWithKeyboard 发送私聊消息，可附带按钮键盘组件。
// buttons 为 nil 时退化为纯文本私聊。
//
// QQ 官方平台约束：群聊/单聊中按钮组件仅在 msg_type=2（markdown 消息）下渲染；
// 纯文本消息（msg_type=0）携带 keyboard 时服务端接受请求（HTTP 200、日志显示成功）
// 但静默丢弃按钮，用户端看不到。故附带按钮时正文改以原生 markdown 承载，
// content 保留为不支持 markdown 渲染场景的降级文案。
func (s *Sender) SendC2CMessageWithKeyboard(ctx context.Context, userID, content string, buttons *keyboard.CustomKeyboard) (*dto.Message, error) {
	if err := validateTarget(userID); err != nil {
		return nil, fmt.Errorf("发送私聊消息: %w", err)
	}
	if err := validateContent(content); err != nil {
		return nil, fmt.Errorf("发送私聊消息: %w", err)
	}

	msg := &dto.MessageToCreate{Content: content}
	if buttons != nil && len(buttons.Rows) > 0 {
		msg.MsgType = dto.MarkdownMsg
		msg.Markdown = &dto.Markdown{Content: toMarkdown(content)}
		msg.Keyboard = &keyboard.MessageKeyboard{Content: buttons}
	}

	sent, err := s.api.PostC2CMessage(ctx, userID, msg)
	if err != nil {
		// markdown+keyboard 发送失败（如机器人未开通 markdown 权限）时，
		// 降级为纯文本重发一次，保证通知本身不丢失；同时记录原始错误便于排查权限问题。
		if msg.MsgType == dto.MarkdownMsg {
			s.logger.Error("发送带按钮私聊消息失败，降级为纯文本重发",
				zap.String("user_id", userID), zap.Error(err))
			return s.SendC2CMessage(ctx, userID, content)
		}
		s.logger.Error("发送私聊消息失败", zap.String("user_id", userID), zap.Error(err))
		return nil, err
	}
	s.logger.Info("私聊消息发送成功",
		zap.String("user_id", userID),
		zap.String("message_id", sent.ID),
		zap.Bool("with_buttons", msg.MsgType == dto.MarkdownMsg),
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

// toMarkdown 把纯文本通知内容转换为 Markdown 消息兼容的紧凑逐行排版。
//
// QQ Markdown（msg_type=2）引擎把空行（\n\n）渲染为段落分隔（产生大幅间距），
// 而单个 \n 在段落内按行显示。模板正文里字段用 \n、区块间用空行，
// 这里把空行压缩掉、只保留单 \n，让所有行在同一个段落内逐行紧凑展示，
// 观感与纯文本一致。若个别 QQ 版本将单 \n 也并成一段，可再切换为列表方案。
func toMarkdown(content string) string {
	if content == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	var b strings.Builder
	for _, ln := range lines {
		trimmed := strings.Trim(ln, " \r")
		if trimmed == "" {
			continue // 丢弃空行，避免段落间距
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(trimmed)
	}
	return b.String()
}
