package qqapproval

import (
	"context"
	"encoding/json"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// HandleInteraction 处理 INTERACTION_CREATE 事件：解析按钮点击并分发审批动作。
// 供 bot.Handler 的互动事件回调调用。
func (s *Service) HandleInteraction(ctx context.Context, data *dto.WSInteractionData) error {
	openid, action, whitelistID, nickname := parseInteraction(data)
	if openid == "" || whitelistID == "" {
		s.logger.Debug("互动事件缺少必要字段，忽略",
			zap.String("openid", openid), zap.String("action", action), zap.String("whitelist_id", whitelistID))
		return nil
	}
	s.logger.Info("QQ 审批：收到按钮点击",
		zap.String("interaction_id", data.ID), zap.String("openid", openid),
		zap.String("action", action), zap.String("whitelist_id", whitelistID))

	// QQ 客户端要求第三方机器人确认互动已收到，否则即使后续业务成功，
	// 客户端仍会显示“请求第三方失败”。业务审批结果通过私聊消息另行反馈。
	if s.acker != nil && data.ID != "" {
		if err := s.acker.PutInteraction(ctx, data.ID, `{"code":0}`); err != nil {
			s.logger.Warn("QQ 审批：按钮互动回执失败",
				zap.String("interaction_id", data.ID), zap.Error(err))
		}
	}
	return s.DoAction(ctx, openid, action, whitelistID, nickname)
}

// parseInteraction 从互动事件提取 (openid, action, whitelistID, nickname)。
// 优先用 Interaction.UserOpenID；按钮数据取 Resolved 中的 button_id / button_data。
func parseInteraction(data *dto.WSInteractionData) (openid, action, whitelistID, nickname string) {
	if data == nil {
		return "", "", "", ""
	}
	openid = data.UserOpenID

	// 尝试从 Resolved 解析按钮数据
	if data.Data != nil && len(data.Data.Resolved) > 0 {
		var resolved struct {
			ButtonID   string `json:"button_id"`
			ButtonData string `json:"button_data"`
			UserID     string `json:"user_id"`
		}
		if err := json.Unmarshal(data.Data.Resolved, &resolved); err == nil {
			if openid == "" {
				openid = resolved.UserID
			}
			payload := resolved.ButtonData
			if payload == "" {
				payload = resolved.ButtonID
			}
			action, whitelistID, _ = ParseAction(payload)
		}
	}
	return openid, action, whitelistID, ""
}
