package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/lumiadmin"
)

// BanHandler 处理 /ban 封禁信息查询指令。
type BanHandler struct {
	sender  MessageSender
	querier lumiadmin.BanStatusQuerier
	logger  *zap.Logger
}

// NewBanHandler 构建 /ban 指令处理器。
func NewBanHandler(sender MessageSender, querier lumiadmin.BanStatusQuerier, logger *zap.Logger) *BanHandler {
	return &BanHandler{sender: sender, querier: querier, logger: logger}
}

// Handle 处理消息中的 /ban 指令。
func (h *BanHandler) Handle(ctx context.Context, msg *dto.Message) (bool, error) {
	if msg == nil {
		return false, nil
	}
	args, ok := parseBanCommand(msg.Content)
	if !ok {
		return false, nil
	}
	if len(args) != 1 {
		return true, h.reply(ctx, msg, "用法：/ban <steamid64/steamid2>\n支持 SteamID64 或 SteamID2。")
	}

	result, err := h.querier.QueryBanStatus(ctx, args[0])
	if err != nil {
		h.logger.Warn("封禁信息查询失败",
			zap.String("user_id", authorID(msg)),
			zap.String("steam_input", args[0]),
			zap.Error(err),
		)
		return true, h.reply(ctx, msg, "查询封禁信息失败，请稍后重试。")
	}
	return true, h.reply(ctx, msg, RenderBanStatus(result))
}

func parseBanCommand(content string) ([]string, bool) {
	fields := strings.Fields(normalizeCommandContent(content))
	if len(fields) == 0 || !strings.EqualFold(fields[0], "/ban") {
		return nil, false
	}
	return fields[1:], true
}

func (h *BanHandler) reply(ctx context.Context, msg *dto.Message, content string) error {
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

// RenderBanStatus 将两类封禁查询结果转换为 QQ 消息。
func RenderBanStatus(result *lumiadmin.BanStatusResponse) string {
	if result == nil {
		return "查询封禁信息失败，请稍后重试。"
	}
	player := result.SteamID64
	if player == "" {
		player = "未知"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "玩家：%s\n\n", player)

	if len(result.LocalBans) == 0 {
		b.WriteString("网站封禁：✅ 未发现封禁\n")
	} else {
		b.WriteString("网站封禁\n")
		writeLocalBanRecords(&b, result.LocalBans)
	}

	b.WriteString("\n")
	if len(result.GlobalBans) == 0 {
		b.WriteString("全球封禁：✅ 未发现封禁")
	} else {
		b.WriteString("全球封禁\n")
		writeGlobalBanRecords(&b, result.GlobalBans)
	}
	return strings.TrimSpace(b.String())
}

func writeLocalBanRecords(b *strings.Builder, records []lumiadmin.LocalBanRecord) {
	limit := min(len(records), maxBanRecords)
	for i := 0; i < limit; i++ {
		if i > 0 {
			b.WriteString("\n\n")
		}
		if len(records) > 1 {
			fmt.Fprintf(b, "记录 %d\n", i+1)
		}
		fmt.Fprintf(b, "状态：%s\n", localBanStatus(records[i]))
		fmt.Fprintf(b, "封禁类型：%s\n", localBanType(records[i].BanType))
		fmt.Fprintf(b, "原因：%s\n", nonEmpty(records[i].Reason, "未填写封禁原因"))
		fmt.Fprintf(b, "封禁时间：%s", formatBeijingTimestamp(records[i].CreatedAt))
		if records[i].ExpiresAt == nil || strings.TrimSpace(*records[i].ExpiresAt) == "" {
			b.WriteString("\n到期时间：永久")
		} else {
			fmt.Fprintf(b, "\n到期时间：%s", formatBeijingTimestamp(*records[i].ExpiresAt))
		}
		if records[i].Status == "inactive" {
			if records[i].RemovedAt != nil && strings.TrimSpace(*records[i].RemovedAt) != "" {
				fmt.Fprintf(b, "\n解封时间：%s", formatBeijingTimestamp(*records[i].RemovedAt))
			}
			if records[i].RemovedReason != nil && strings.TrimSpace(*records[i].RemovedReason) != "" {
				fmt.Fprintf(b, "\n解封原因：%s", strings.TrimSpace(*records[i].RemovedReason))
			}
		}
	}
	writeOverflow(b, len(records), limit, "网站封禁")
}

func writeGlobalBanRecords(b *strings.Builder, records []lumiadmin.GlobalBanRecord) {
	limit := min(len(records), maxBanRecords)
	for i := 0; i < limit; i++ {
		if i > 0 {
			b.WriteString("\n\n")
		}
		if len(records) > 1 {
			fmt.Fprintf(b, "记录 %d\n", i+1)
		}
		ban := records[i]
		if ban.ManualUnbanned {
			b.WriteString("全球封禁状态：🌍 全球封禁中\n")
		} else if ban.IsExpired {
			b.WriteString("状态：⌛ 已过期\n")
		} else {
			b.WriteString("状态：🌍 生效中\n")
		}
		fmt.Fprintf(b, "封禁类型：%s\n", globalBanType(ban.BanType))
		fmt.Fprintf(b, "原因：%s", globalBanReason(ban))
		if stats := shortGlobalStats(ban.Stats); stats != "" {
			fmt.Fprintf(b, "\n附加信息：%s", stats)
		}
		fmt.Fprintf(b, "\n封禁时间：%s", optionalTimestamp(ban.CreatedOn))
		fmt.Fprintf(b, "\n到期时间：%s", globalExpiry(ban.ExpiresOn))
		if ban.ManualUnbanned {
			b.WriteString("\n备注：本地服务器已解除该封禁")
		}
	}
	writeOverflow(b, len(records), limit, "全球封禁")
}

const maxBanRecords = 10

func writeOverflow(b *strings.Builder, total, shown int, label string) {
	if total > shown {
		fmt.Fprintf(b, "\n\n%s：显示最近 %d 条，另有 %d 条历史记录未显示", label, shown, total-shown)
	}
}

func localBanStatus(ban lumiadmin.LocalBanRecord) string {
	if strings.EqualFold(ban.Status, "active") {
		if ban.ExpiresAt != nil && isPastTimestamp(*ban.ExpiresAt) {
			return "⌛ 已过期"
		}
		return "🔒 封禁中"
	}
	if strings.EqualFold(ban.Status, "inactive") {
		return "✅ 已解除"
	}
	return "❔ " + nonEmpty(ban.Status, "未知")
}

func localBanType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "steam":
		return "Steam 账号封禁"
	case "ip":
		return "IP 封禁"
	default:
		return nonEmpty(value, "未知")
	}
}

func globalBanType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bhop_hack":
		return "连跳作弊"
	case "cheat", "hack":
		return "作弊"
	case "tool_assist":
		return "辅助工具"
	case "":
		return "未知"
	default:
		return value
	}
}

func globalBanReason(ban lumiadmin.GlobalBanRecord) string {
	if ban.Notes != nil && strings.TrimSpace(*ban.Notes) != "" {
		return strings.TrimSpace(*ban.Notes)
	}
	return "未填写封禁原因"
}

// shortGlobalStats 保留全球封禁 stats 的概要，避免 KZTimer 原始信息过长刷屏。
func shortGlobalStats(raw *string) string {
	if raw == nil {
		return ""
	}
	value := strings.Join(strings.Fields(*raw), " ")
	if value == "" {
		return ""
	}
	const maxRunes = 120
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}

func globalExpiry(raw *string) string {
	if raw == nil || strings.TrimSpace(*raw) == "" || strings.HasPrefix(strings.TrimSpace(*raw), "9999") {
		return "永久"
	}
	return formatBeijingTimestamp(*raw)
}

func optionalTimestamp(raw *string) string {
	if raw == nil {
		return "未记录"
	}
	return nonEmpty(formatBeijingTimestamp(*raw), "未记录")
}

func isPastTimestamp(raw string) bool {
	parsed, err := parseTimestamp(raw)
	return err == nil && parsed.Before(time.Now())
}

func parseTimestamp(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid timestamp")
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
