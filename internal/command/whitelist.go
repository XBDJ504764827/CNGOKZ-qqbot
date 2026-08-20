// Package command 实现 LumiBot 的 QQ 文本指令。
package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/tencent-connect/botgo/dto"
	qqmessage "github.com/tencent-connect/botgo/dto/message"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/lumiadmin"
)

// MessageSender 是指令回复所需的最小消息发送接口。
type MessageSender interface {
	SendC2CMessage(ctx context.Context, userID, content string) (*dto.Message, error)
	SendGroupMessage(ctx context.Context, groupID, content string) (*dto.Message, error)
	SendChannelMessage(ctx context.Context, channelID, content string) (*dto.Message, error)
}

// WhitelistHandler 处理 /wl 白名单状态查询指令。
type WhitelistHandler struct {
	sender  MessageSender
	querier lumiadmin.WhitelistStatusQuerier
	logger  *zap.Logger
}

// NewWhitelistHandler 构建白名单状态查询指令处理器。
func NewWhitelistHandler(sender MessageSender, querier lumiadmin.WhitelistStatusQuerier, logger *zap.Logger) *WhitelistHandler {
	return &WhitelistHandler{sender: sender, querier: querier, logger: logger}
}

// Handle 处理消息中的 /wl 指令。
// 返回 handled=true 表示消息是 /wl 指令，无论查询成功还是失败。
func (h *WhitelistHandler) Handle(ctx context.Context, msg *dto.Message) (bool, error) {
	if msg == nil {
		return false, nil
	}
	args, ok := parseWhitelistCommand(msg.Content)
	if !ok {
		return false, nil
	}

	if len(args) != 1 {
		return true, h.reply(ctx, msg, usage())
	}

	result, err := h.querier.QueryWhitelistStatus(ctx, args[0])
	if err != nil {
		// 不向用户暴露 LumiAdmin 地址、HTTP 状态或数据库错误。
		h.logger.Warn("白名单状态查询失败",
			zap.String("user_id", authorID(msg)),
			zap.String("steam_input", args[0]),
			zap.Error(err),
		)
		return true, h.reply(ctx, msg, "查询白名单状态失败，请稍后重试。")
	}
	return true, h.reply(ctx, msg, RenderWhitelistStatus(result))
}

func parseWhitelistCommand(content string) ([]string, bool) {
	// GROUP_AT_MESSAGE_CREATE 可能带有 `<@!bot_id>` 前缀；使用 botgo
	// 提供的 ETLInput 清理 @ 标记，同时不影响私聊中的普通 /wl 指令。
	fields := strings.Fields(qqmessage.ETLInput(content))
	if len(fields) == 0 || !strings.EqualFold(fields[0], "/wl") {
		return nil, false
	}
	return fields[1:], true
}

func usage() string {
	return "用法：/wl <steamid64/steamid2>\n支持 SteamID64 或 SteamID2。"
}

// RenderWhitelistStatus 将 LumiAdmin 查询结果转换为 QQ 用户可读的中文消息。
func RenderWhitelistStatus(result *lumiadmin.WhitelistStatusResponse) string {
	if result == nil {
		return "查询白名单状态失败，请稍后重试。"
	}
	player := result.SteamID64
	if player == "" {
		player = "未知"
	}
	if len(result.Items) == 0 {
		return fmt.Sprintf("玩家：%s\n\n白名单状态：⚪ 未找到记录，该玩家可能未申请白名单", player)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "玩家：%s\n\n", player)
	multiple := len(result.Items) > 1
	for i, item := range result.Items {
		if multiple {
			fmt.Fprintf(&b, "记录 %d\n", i+1)
		}
		fmt.Fprintf(&b, "白名单状态：%s", statusText(item.Status))
		if item.Status == "rejected" {
			reason := "未填写拒绝原因"
			if item.RejectionReason != nil && strings.TrimSpace(*item.RejectionReason) != "" {
				reason = strings.TrimSpace(*item.RejectionReason)
			}
			fmt.Fprintf(&b, "\n原因：%s", reason)
		}
		// 单条记录保持简洁；历史记录超过一条时，为每条记录标注对应时间。
		if multiple {
			if timestamp := itemTimestamp(item); timestamp != "" {
				fmt.Fprintf(&b, "\n时间：%s", formatTimestamp(timestamp))
			}
		}
		if i < len(result.Items)-1 {
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func statusText(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved":
		return "✅ 已通过"
	case "pending":
		return "⏳ 待审核"
	case "rejected":
		return "❌ 已拒绝"
	case "revoked":
		return "⚠️ 已撤销"
	default:
		return "❔ " + status
	}
}

func itemTimestamp(item lumiadmin.WhitelistRecord) string {
	switch item.Status {
	case "approved":
		if item.ApprovedAt != nil && *item.ApprovedAt != "" {
			return *item.ApprovedAt
		}
	case "rejected":
		if item.RejectedAt != nil && *item.RejectedAt != "" {
			return *item.RejectedAt
		}
	case "revoked":
		if item.RevokedAt != nil && *item.RevokedAt != "" {
			return *item.RevokedAt
		}
	}
	return item.AppliedAt
}

func formatTimestamp(raw string) string {
	return formatBeijingTimestamp(raw)
}

func (h *WhitelistHandler) reply(ctx context.Context, msg *dto.Message, content string) error {
	if h.sender == nil {
		return fmt.Errorf("发送指令回复失败：sender 未配置")
	}
	if msg.GroupID != "" {
		_, err := h.sender.SendGroupMessage(ctx, msg.GroupID, content)
		return err
	}
	if msg.ChannelID != "" {
		_, err := h.sender.SendChannelMessage(ctx, msg.ChannelID, content)
		return err
	}
	if id := authorID(msg); id != "" {
		_, err := h.sender.SendC2CMessage(ctx, id, content)
		return err
	}
	return fmt.Errorf("发送指令回复失败：消息缺少群、用户或频道目标")
}

func authorID(msg *dto.Message) string {
	if msg == nil || msg.Author == nil {
		return ""
	}
	return msg.Author.ID
}
