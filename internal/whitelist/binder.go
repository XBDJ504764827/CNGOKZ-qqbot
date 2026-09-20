// Package whitelist 处理 QQ 私聊白名单验证码绑定。
//
// 玩家添加机器人 QQ 好友后，私聊直接发送验证码，本包解析并调用 LumiAdmin
// 完成 Steam↔QQ(openid) 绑定，随后在私聊中被动回复结果。
//
// 采用私聊（C2C）而非 QQ 群：群内主动消息受平台权限限制（40034105），
// 私聊对玩家刚发来的消息做被动回复不受该限制。
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

// Binder 处理私聊验证码绑定。
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

// HandleC2CMessage 处理玩家私聊消息：若包含验证码则完成绑定并回复。
// 返回 handled 表示该消息已被消费（已回复），err 为处理过程中的错误。
//
// 即使集成未启用（client 为 nil），只要消息包含验证码也会给出提示回复，
// 避免玩家发码后毫无反馈、一直等待。
func (b *Binder) HandleC2CMessage(ctx context.Context, openID, username, content string) (bool, error) {
	if b == nil || openID == "" {
		return false, nil
	}

	match := codePattern.FindStringSubmatch(content)
	if match == nil {
		return false, nil
	}

	if !b.Enabled() {
		b.logger.Warn("收到白名单验证码，但 LumiAdmin 集成未启用，无法绑定",
			zap.String("openid", openID),
		)
		b.reply(ctx, openID, "绑定服务暂不可用，请联系管理员检查机器人配置。")
		return true, nil
	}

	code := "WL-" + strings.ToUpper(match[1])

	outcome, err := b.client.VerifyAndBind(ctx, lumiadmin.BindVerifyRequest{
		Code:       code,
		QQOpenID:   openID,
		QQUsername: username,
	})
	if err != nil {
		b.logger.Warn("验证码绑定校验失败",
			zap.String("openid", openID),
			zap.String("code", code),
			zap.Error(err),
		)
		b.reply(ctx, openID, "验证失败，请稍后重试或联系管理员。")
		return true, err
	}

	b.reply(ctx, openID, replyText(outcome))
	b.logger.Info("白名单验证码绑定处理完成",
		zap.String("openid", openID),
		zap.String("result", outcome.Result),
		zap.String("steamid64", outcome.SteamID64),
	)
	return true, nil
}

// replyText 根据绑定结果生成私聊回复文案。
func replyText(outcome lumiadmin.BindOutcome) string {
	switch outcome.Result {
	case "bound":
		if outcome.Already {
			return "该账号已完成过绑定，可直接返回网站继续申请白名单。"
		}
		return "绑定成功！请返回网站继续下一步。"
	case "invalid_code":
		return "验证码无效或已过期（有效期 5 分钟），请回到网站重新生成。"
	case "consumed":
		return "该验证码已被使用，请回到网站重新生成。"
	case "limit_reached":
		return "该 QQ 绑定的 Steam 账号数量已达上限，请联系管理员处理。"
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

// reply 在私聊中发送文本回复（失败仅记录日志）。
func (b *Binder) reply(ctx context.Context, openID, content string) {
	if b.sender == nil {
		return
	}
	if _, err := b.sender.SendC2CMessage(ctx, openID, content); err != nil {
		b.logger.Warn("私聊回复发送失败",
			zap.String("openid", openID),
			zap.Error(err),
		)
	}
}
