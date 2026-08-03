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

	// 3. 冷却防刷
	if !s.cooldown.Allow(ev.EventType, r.Cooldown) {
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
	status, err := s.send(ctx, n)
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

// send 按渠道发送通知，返回发送状态（sent / skipped）与错误。
func (s *Service) send(ctx context.Context, n Notification) (string, error) {
	switch n.Channel {
	case ChannelQQPrivate:
		if s.privateTarget == "" {
			return "skipped", fmt.Errorf("未配置 QQ_PRIVATE 通知目标（NOTIFY_PRIVATE_TARGET）")
		}
		if _, err := s.sender.SendC2CMessage(ctx, s.privateTarget, n.Content); err != nil {
			return "failed", err
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
