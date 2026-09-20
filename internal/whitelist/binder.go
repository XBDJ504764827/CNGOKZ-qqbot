// Package whitelist 处理 QQ 群内白名单验证码绑定。
//
// 玩家在 LumiAdmin 网站生成验证码后，在 QQ 群内 @机器人 发送验证码，
// 本包解析出验证码并调用 LumiAdmin 完成 Steam↔QQ 绑定，随后在群内回复结果。
package whitelist

import (
	"context"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/lumiadmin"
	"github.com/XBDJ504764827/LumiBot/internal/message"
)

// codePattern 匹配验证码：WL-XXXXXX（6 位去混淆大写字符），允许玩家省略 WL- 前缀。
var codePattern = regexp.MustCompile(`(?i)\bWL[-\s]?([A-Z2-9]{6})\b`)

// Binder 处理群内验证码绑定。
type Binder struct {
	client *lumiadmin.Client
	sender *message.Sender
	logger *zap.Logger
}

// NewBinder 构建绑定处理器（client 为 nil 时禁用）。
func NewBinder(client *lumiadmin.Client, sender *message.Sender, logger *zap.Logger) *Binder {
	return &Binder{client: client, sender: sender, logger: logger}
}

// Enabled 是否启用绑定处理。
func (b *Binder) Enabled() bool {
	return b != nil && b.client != nil
}

// HandleGroupMessage 尝试从群消息中解析验证码并完成绑定。
// 返回 handled 表示该消息是绑定指令（已回复），err 为处理过程中的错误。
func (b *Binder) HandleGroupMessage(ctx context.Context, groupID, openID, username, content string) (bool, error) {
	if !b.Enabled() || groupID == "" {
		return false, nil
	}

	match := codePattern.FindStringSubmatch(content)
	if match == nil {
		return false, nil
	}
	code := "WL-" + strings.ToUpper(match[1])

	outcome, err := b.client.VerifyAndBind(ctx, lumiadmin.BindVerifyRequest{
		Code:       code,
		QQOpenID:   openID,
		QQGroupID:  groupID,
		QQUsername: username,
	})
	if err != nil {
		b.logger.Warn("验证码绑定校验失败",
			zap.String("group_id", groupID),
			zap.String("code", code),
			zap.Error(err),
		)
		b.reply(ctx, groupID, "验证失败，请稍后重试或联系管理员。")
		return true, err
	}

	b.reply(ctx, groupID, replyText(outcome))
	b.logger.Info("白名单验证码绑定处理完成",
		zap.String("group_id", groupID),
		zap.String("result", outcome.Result),
		zap.String("steamid64", outcome.SteamID64),
	)
	return true, nil
}

// replyText 根据绑定结果生成群内回复文案。
func replyText(outcome lumiadmin.BindOutcome) string {
	switch outcome.Result {
	case "bound":
		if outcome.Already {
			return "该账号已完成过绑定，可直接返回网站继续申请白名单。"
		}
		return "绑定成功！请返回网站继续填写申请理由并提交白名单申请。"
	case "invalid_code":
		return "验证码无效或已过期（有效期 5 分钟），请回到网站重新生成。"
	case "consumed":
		return "该验证码已被使用，请回到网站重新生成。"
	case "limit_reached":
		return "该 QQ 绑定的 Steam 账号数量已达上限，请联系管理员处理。"
	case "group_denied":
		return "当前 QQ 群不在允许绑定的群列表内，请联系管理员。"
	case "disabled":
		return "当前未开启 QQ 绑定，请稍后再试。"
	case "rejected":
		if outcome.Message != "" {
			return outcome.Message
		}
		return "绑定失败，请联系管理员。"
	default:
		if outcome.Message != "" {
			return outcome.Message
		}
		return "绑定失败，请联系管理员。"
	}
}

// reply 在群内发送文本回复（失败仅记录日志）。
func (b *Binder) reply(ctx context.Context, groupID, content string) {
	if b.sender == nil {
		return
	}
	if _, err := b.sender.SendGroupMessage(ctx, groupID, content); err != nil {
		b.logger.Warn("群内回复发送失败",
			zap.String("group_id", groupID),
			zap.Error(err),
		)
	}
}
