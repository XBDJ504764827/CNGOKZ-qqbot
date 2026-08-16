package bot

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/message"
)

// Handler 将 QQ 网关事件分发到业务处理层（message 包）。
//
// 事件注册（event.go）只做 botgo 回调适配，业务分发全部收敛于此，
// 未来新增事件类型只需在 event.go 注册并在此增加分发方法。
type Handler struct {
	receiver      *message.Receiver
	sender        *message.Sender
	logger        *zap.Logger
	onInteraction func(ctx context.Context, data *dto.WSInteractionData) error
}

// NewHandler 构建事件分发器（依赖注入：消息接收器、消息发送器、日志）。
func NewHandler(receiver *message.Receiver, sender *message.Sender, logger *zap.Logger) *Handler {
	return &Handler{receiver: receiver, sender: sender, logger: logger}
}

// SetInteractionHandler 注册互动事件处理器（QQ 审批按钮点击等）。
func (h *Handler) SetInteractionHandler(fn func(ctx context.Context, data *dto.WSInteractionData) error) {
	h.onInteraction = fn
}

// OnReady 分发 READY 事件：网关连接就绪。
func (h *Handler) OnReady(data *dto.WSReadyData) {
	h.logger.Info("QQ 网关连接就绪",
		zap.Int("version", data.Version),
		zap.String("session_id", data.SessionID),
		zap.String("bot_id", data.User.ID),
		zap.String("bot_name", data.User.Username),
	)
}

// OnError 分发网关连接错误。
func (h *Handler) OnError(err error) {
	h.logger.Error("QQ 网关连接异常", zap.Error(err))
}

// OnMessage 分发 MESSAGE_CREATE 事件：收到频道消息。
func (h *Handler) OnMessage(ctx context.Context, msg *dto.Message) error {
	return h.receiver.ReceiveMessage(ctx, msg)
}

// OnATMessage 分发 AT_MESSAGE_CREATE 事件：频道内 @机器人 消息。
func (h *Handler) OnATMessage(ctx context.Context, msg *dto.Message) error {
	return h.receiver.ReceiveMessage(ctx, msg)
}

// OnC2CMessage 分发 C2C_MESSAGE_CREATE 事件：用户私聊机器人。
// 日志中的 user_id 即用户 openid，可用于配置通知目标（NOTIFY_PRIVATE_TARGET）。
func (h *Handler) OnC2CMessage(ctx context.Context, msg *dto.Message) error {
	return h.receiver.ReceiveMessage(ctx, msg)
}

// OnInteraction 分发 INTERACTION_CREATE 事件（消息按钮点击等）。
func (h *Handler) OnInteraction(ctx context.Context, data *dto.WSInteractionData) error {
	if h.onInteraction == nil {
		h.logger.Debug("收到互动事件但未注册处理器", zap.String("interaction_id", data.ID))
		return nil
	}
	return h.onInteraction(ctx, data)
}

// OnPlain 分发未注册事件的兜底日志。
func (h *Handler) OnPlain(event *dto.WSPayload, payload []byte) {
	h.logger.Debug("收到未注册网关事件",
		zap.String("type", string(event.Type)),
		zap.ByteString("payload", payload),
	)
}
