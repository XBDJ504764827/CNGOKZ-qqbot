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

const bindCommand = "/bind"

// BindHandler 处理 QQ 私聊中的 /bind 指令。
type BindHandler struct {
	sender  MessageSender
	auditor Auditor
	logger  *zap.Logger

	mu       sync.Mutex
	requests map[string][]time.Time
	now      func() time.Time
}

// NewBindHandler 构建 /bind 指令处理器。
func NewBindHandler(sender MessageSender, auditor Auditor, logger *zap.Logger) *BindHandler {
	return &BindHandler{
		sender:   sender,
		auditor:  auditor,
		logger:   logger,
		requests: make(map[string][]time.Time),
		now:      time.Now,
	}
}

// Handle 处理 /bind。该处理器只接受 C2C 私聊消息：
// msg.DirectMessage 为 true，且消息不能带群/频道上下文。
func (h *BindHandler) Handle(ctx context.Context, msg *dto.Message) (bool, error) {
	if msg == nil || !isBindCommand(msg.Content) {
		return false, nil
	}

	openid := authorID(msg)
	if !isPrivateMessage(msg) {
		// 群聊/频道中不回复，避免公开场景产生干扰或暴露绑定信息。
		return true, nil
	}

	if !h.allow(openid) {
		h.audit(openid, "rate_limited")
		return true, h.reply(ctx, msg, "请求过于频繁，请稍后再试。")
	}
	if openid == "" {
		h.audit(openid, "failed")
		return true, h.reply(ctx, msg, "暂时无法获取你的 QQ OpenID，请稍后重试。")
	}

	content := fmt.Sprintf("你的 QQ OpenID：\n\n%s\n\n请复制此 OpenID 到网站的 QQ 绑定页面。", openid)
	h.audit(openid, "success")
	return true, h.reply(ctx, msg, content)
}

func isBindCommand(content string) bool {
	return strings.TrimSpace(content) == bindCommand
}

func isPrivateMessage(msg *dto.Message) bool {
	if msg == nil {
		return false
	}
	// C2C_MESSAGE_CREATE 的 Message 通常没有 GroupID/ChannelID；
	// DirectMessage 也视为私聊。显式群/频道上下文优先排除。
	return msg.GroupID == "" && msg.GuildID == "" && msg.ChannelID == "" && (msg.DirectMessage || msg.Author != nil)
}

func (h *BindHandler) allow(openid string) bool {
	if openid == "" {
		return true
	}
	now := h.now()
	cutoff := now.Add(-time.Minute)

	h.mu.Lock()
	defer h.mu.Unlock()

	requests := h.requests[openid]
	kept := requests[:0]
	for _, timestamp := range requests {
		if timestamp.After(cutoff) {
			kept = append(kept, timestamp)
		}
	}
	if len(kept) >= 5 {
		h.requests[openid] = kept
		return false
	}
	h.requests[openid] = append(kept, now)
	return true
}

func (h *BindHandler) reply(ctx context.Context, msg *dto.Message, content string) error {
	if h.sender == nil {
		return fmt.Errorf("发送 /bind 回复失败：sender 未配置")
	}
	openid := authorID(msg)
	if openid == "" {
		return fmt.Errorf("发送 /bind 回复失败：消息缺少用户 OpenID")
	}
	_, err := h.sender.SendC2CMessage(ctx, openid, content)
	return err
}

func (h *BindHandler) audit(openid, result string) {
	if h.auditor == nil {
		return
	}
	if err := h.auditor.Record(AuditEvent{
		Event:   "command_bind",
		Command: bindCommand,
		OpenID:  openid,
		Source:  "c2c",
		Result:  result,
	}); err != nil {
		h.logger.Error("QQ /bind 指令审计写入失败", zap.Error(err))
	}
}
