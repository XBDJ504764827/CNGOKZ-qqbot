// Package command 处理 QQ 消息中的用户指令。
//
// 当前仅支持 /bind：用户私聊机器人获取自己的 QQ OpenID，
// 供管理员（或用户本人）复制后在 LumiAdmin 网站的 QQ 绑定页面自助绑定。
package command

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// BindCommand /bind 指令。
const BindCommand = "/bind"

const (
	// bindRateLimit 单个用户每分钟允许执行 /bind 的最大次数，防止刷屏。
	bindRateLimit = 5
	// bindWindow 限流窗口。
	bindWindow = time.Minute
)

// MessageSender 是指令回复所需的最小 QQ 消息发送接口。
// message.Sender 满足该接口（仅 /bind 使用 C2C 私聊回复）。
type MessageSender interface {
	SendC2CMessage(ctx context.Context, userID, content string) (*dto.Message, error)
}

// BindHandler 处理 QQ 私聊中的 /bind 指令：回复用户的 QQ OpenID。
type BindHandler struct {
	sender MessageSender
	logger *zap.Logger

	// requests 记录每个 OpenID 最近的请求时间，用于限流。
	mu       sync.Mutex
	requests map[string][]time.Time
	now      func() time.Time
}

// NewBindHandler 构建 /bind 指令处理器。
func NewBindHandler(sender MessageSender, logger *zap.Logger) *BindHandler {
	return &BindHandler{
		sender:   sender,
		logger:   logger,
		requests: make(map[string][]time.Time),
		now:      time.Now,
	}
}

// Handle 处理 /bind：
//   - 仅匹配私聊（C2C）消息；群聊/频道中静默消费，不回复、不暴露 OpenID；
//   - 仅精确匹配指令（忽略首尾空白，大小写不敏感）；
//   - 单用户限流（每分钟最多 5 次）。
func (h *BindHandler) Handle(ctx context.Context, msg *dto.Message) (bool, error) {
	if msg == nil || !isBindCommand(msg.Content) {
		return false, nil
	}

	openid := authorID(msg)
	if !isPrivateMessage(msg) {
		// 群聊/频道：消费指令但不回复，避免公开场景干扰或暴露绑定信息。
		h.logger.Info("QQ /bind 指令被忽略（仅支持私聊）")
		return true, nil
	}

	if !h.allow(openid) {
		h.logger.Warn("QQ /bind 请求过于频繁", zap.String("openid", openid))
		return true, h.reply(ctx, msg, "请求过于频繁，请稍后再试。")
	}
	if openid == "" {
		// 无法获取 OpenID，且没有目标可发送回复，仅记录日志。
		h.logger.Warn("QQ /bind 无法获取用户 OpenID")
		return true, nil
	}

	content := fmt.Sprintf("你的 QQ OpenID：\n\n%s\n\n请复制此 OpenID 到 LumiAdmin 网站的 QQ 绑定页面完成绑定。", openid)
	if err := h.reply(ctx, msg, content); err != nil {
		h.logger.Error("QQ /bind 回复失败", zap.String("openid", openid), zap.Error(err))
		return true, err
	}
	h.logger.Info("QQ /bind 指令处理成功", zap.String("openid", openid))
	return true, nil
}

// isBindCommand 判断消息是否精确匹配 /bind（忽略首尾空白，大小写不敏感）。
func isBindCommand(content string) bool {
	return strings.EqualFold(strings.TrimSpace(content), BindCommand)
}

// isPrivateMessage 判断消息是否来自 C2C 私聊。
func isPrivateMessage(msg *dto.Message) bool {
	if msg == nil {
		return false
	}
	// C2C_MESSAGE_CREATE 的消息通常没有 GroupID/GuildID/ChannelID，
	// 且 bot 层已显式标记 DirectMessage；带群/频道上下文的消息一律排除。
	return msg.GroupID == "" && msg.GuildID == "" && msg.ChannelID == "" &&
		(msg.DirectMessage || msg.Author != nil)
}

// allow 限流：单个 OpenID 在窗口内最多 bindRateLimit 次。
func (h *BindHandler) allow(openid string) bool {
	if openid == "" {
		return true
	}
	now := h.now()
	cutoff := now.Add(-bindWindow)

	h.mu.Lock()
	defer h.mu.Unlock()

	requests := h.requests[openid]
	kept := requests[:0]
	for _, timestamp := range requests {
		if timestamp.After(cutoff) {
			kept = append(kept, timestamp)
		}
	}
	if len(kept) >= bindRateLimit {
		h.requests[openid] = kept
		return false
	}
	h.requests[openid] = append(kept, now)
	return true
}

// reply 通过 C2C 私聊向消息发送者回复内容。
func (h *BindHandler) reply(ctx context.Context, msg *dto.Message, content string) error {
	if h.sender == nil {
		return fmt.Errorf("QQ /bind 回复失败：sender 未配置")
	}
	openid := authorID(msg)
	if openid == "" {
		return fmt.Errorf("QQ /bind 回复失败：消息缺少用户 OpenID")
	}
	_, err := h.sender.SendC2CMessage(ctx, openid, content)
	return err
}

// authorID 安全获取消息发送者 ID（Author 可能为空）。
func authorID(msg *dto.Message) string {
	if msg == nil || msg.Author == nil {
		return ""
	}
	return msg.Author.ID
}
