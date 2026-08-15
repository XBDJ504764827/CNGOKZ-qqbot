package notification

import (
	"context"
	"fmt"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/config"
	"github.com/XBDJ504764827/LumiBot/internal/event"
	"github.com/XBDJ504764827/LumiBot/internal/rule"
)

// QQMessageSender 通知系统依赖的最小 QQ 发送接口（依赖倒置）。
//
// *message.Sender 天然满足该接口；测试可用 fake 替身，
// 通知系统不直接依赖 botgo / message 具体实现。
type QQMessageSender interface {
	SendC2CMessage(ctx context.Context, userID, content string) (*dto.Message, error)
	SendChannelMessage(ctx context.Context, channelID, content string) (*dto.Message, error)
}

// Service 通知服务：事件 → 规则 → 模板 → 冷却 → QQ 发送。
//
// 实现 event.Handler 接口，通过 Bus.Subscribe 注册。
type Service struct {
	logger    *zap.Logger
	sender    QQMessageSender
	rules     *rule.Rules
	templates *Templates
	cooldown  Cooldown

	enable        bool
	privateTarget string // QQ_PRIVATE 目标（管理员 QQ openid）
	channelTarget string // QQ_CHANNEL 目标（子频道 ID）
}

// NewService 构建通知服务（依赖注入：配置、发送器、日志）。
func NewService(cfg config.NotificationConfig, sender QQMessageSender, logger *zap.Logger) *Service {
	return &Service{
		logger:        logger,
		sender:        sender,
		rules:         rule.New(cfg.Cooldown),
		templates:     NewTemplates(),
		cooldown:      NewMemoryCooldown(),
		enable:        cfg.Enable,
		privateTarget: cfg.PrivateTarget,
		channelTarget: cfg.ChannelTarget,
	}
}

// HandleEvent 处理事件（实现 event.Handler）：
//
//  1. 开关检查（NOTIFICATION_ENABLE）
//  2. 规则判断（internal/rule）
//  3. 冷却防刷（重复事件窗口内只发一次）
//  4. 模板渲染（template.go）
//  5. 发送（QQMessageSender）并记录通知日志
func (s *Service) HandleEvent(ctx context.Context, ev event.Event) error {
	// 1. 开关
	if !s.enable {
		s.logger.Debug("通知系统已关闭，跳过",
			zap.String("event_id", ev.ID),
			zap.String("event_type", ev.EventType),
		)
		return nil
	}

	// 2. 规则判断
	shouldNotify, r := s.rules.ShouldNotify(ev)
	if !shouldNotify {
		s.logger.Debug("事件不满足通知规则，跳过",
			zap.String("event_id", ev.ID),
			zap.String("event_type", ev.EventType),
		)
		return nil
	}

	// 3. 冷却防刷。
	// 白名单申请事件彼此独立（每人每申请各成一条，用户通过后不可重复提交），
	// 故以事件 ID 为去重键，保证每条申请都能被即时通知，不被同类型窗口吞掉；
	// 其余事件保持按事件类型窗口去重（同类型重复打扰防刷）。
	cooldownKey := ev.EventType
	if ev.EventType == event.EventWhitelistRequestCreated {
		cooldownKey = ev.ID
	}
	if !s.cooldown.Allow(cooldownKey, r.Cooldown) {
		s.logger.Debug("事件在冷却窗口内，跳过重复通知",
			zap.String("event_id", ev.ID),
			zap.String("event_type", ev.EventType),
			zap.Duration("cooldown", r.Cooldown),
		)
		return nil
	}

	// 4. 模板渲染 → 构建通知
	title, content := s.templates.Render(ev)
	n := NewNotification(ev, ChannelQQPrivate, title, content)

	// 5. 发送 + 通知日志（event_id / event_type / send_status / error）
	status, err := s.send(ctx, ev, n)
	if err != nil {
		s.logger.Error("notification send failed",
			zap.String("notification_id", n.ID),
			zap.String("event_id", ev.ID),
			zap.String("event_type", ev.EventType),
			zap.String("send_status", status),
			zap.Error(err),
		)
		return err
	}
	s.logger.Info("notification sent",
		zap.String("notification_id", n.ID),
		zap.String("event_id", ev.ID),
		zap.String("event_type", ev.EventType),
		zap.String("send_status", status),
		zap.String("channel", string(n.Channel)),
	)
	return nil
}

// resolvePrivateTargets 解析私聊通知目标：
//   - 优先取事件 data.openids（网站后台填了 openid 的管理员，见 LumiAdmin 白名单申请上报）；
//   - 未提供时回退到配置的默认管理员（NOTIFY_PRIVATE_TARGET）；
//   - 均未配置时返回 nil（跳过私聊）。
func (s *Service) resolvePrivateTargets(ev event.Event) []string {
	if raw, ok := ev.Data["openids"]; ok {
		var openids []string
		switch v := raw.(type) {
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok && str != "" {
					openids = append(openids, str)
				}
			}
		case []string:
			for _, str := range v {
				if str != "" {
					openids = append(openids, str)
				}
			}
		}
		if len(openids) > 0 {
			return openids
		}
	}
	if s.privateTarget != "" {
		return []string{s.privateTarget}
	}
	return nil
}

// send 按渠道发送通知，返回发送状态（sent / skipped）与错误。
// 私聊渠道支持多个目标：events data.openids 列表逐个发送；缺省时发默认管理员。
func (s *Service) send(ctx context.Context, ev event.Event, n Notification) (string, error) {
	switch n.Channel {
	case ChannelQQPrivate:
		targets := s.resolvePrivateTargets(ev)
		if len(targets) == 0 {
			return "skipped", fmt.Errorf("未配置 QQ_PRIVATE 通知目标（NOTIFY_PRIVATE_TARGET 或 data.openids）")
		}
		for _, target := range targets {
			if _, err := s.sender.SendC2CMessage(ctx, target, n.Content); err != nil {
				return "failed", err
			}
		}
	case ChannelQQChannel:
		if s.channelTarget == "" {
			return "skipped", fmt.Errorf("未配置 QQ_CHANNEL 通知目标（NOTIFY_CHANNEL_TARGET）")
		}
		if _, err := s.sender.SendChannelMessage(ctx, s.channelTarget, n.Content); err != nil {
			return "failed", err
		}
	default:
		return "skipped", fmt.Errorf("暂不支持的通知渠道: %s", n.Channel)
	}
	return "sent", nil
}
