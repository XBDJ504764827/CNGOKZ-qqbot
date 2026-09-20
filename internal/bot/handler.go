package bot

import (
	"context"
	"strings"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/message"
	"github.com/XBDJ504764827/LumiBot/internal/whitelist"
)

// ChatRecorder 将玩家私聊消息回传 LumiAdmin 的最小接口。
// *lumiadmin.Client 天然满足；nil 时跳过记录。
type ChatRecorder interface {
	RecordInboundChat(ctx context.Context, openID, content string) error
}

// Handler 将 QQ 网关事件分发到业务处理层（message 包）。
//
// 事件注册（event.go）只做 botgo 回调适配，业务分发全部收敛于此，
// 未来新增事件类型只需在 event.go 注册并在此增加分发方法。
// 注意：不再提供互动事件处理器（无按钮交互，QQ 仅通知）。
type Handler struct {
	receiver *message.Receiver
	sender   *message.Sender
	binder   *whitelist.Binder
	chat     ChatRecorder
	logger   *zap.Logger
}

// NewHandler 构建事件分发器（依赖注入：消息接收器、消息发送器、日志）。
func NewHandler(receiver *message.Receiver, sender *message.Sender, logger *zap.Logger) *Handler {
	return &Handler{receiver: receiver, sender: sender, logger: logger}
}

// WithBinder 注入白名单绑定处理器（可选），用于私聊验证码绑定。
func (h *Handler) WithBinder(binder *whitelist.Binder) *Handler {
	h.binder = binder
	return h
}

// WithChatRecorder 注入玩家私聊消息回传器（可选）。
func (h *Handler) WithChatRecorder(rec ChatRecorder) *Handler {
	h.chat = rec
	return h
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
// 优先交给 Binder 处理验证码绑定；其余私聊消息回传 LumiAdmin
// （按 openid 归属到 Steam），供管理员聊天面板查看。
func (h *Handler) OnC2CMessage(ctx context.Context, msg *dto.Message) error {
	openID := ""
	username := ""
	if msg.Author != nil {
		openID = msg.Author.ID
		username = msg.Author.Username
	}
	if h.binder != nil {
		handled, err := h.binder.HandleC2CMessage(ctx, openID, username, msg.Content)
		if handled {
			return err
		}
	}
	// 非验证码私聊消息：回传 LumiAdmin 供管理员聊天面板查看
	if h.chat != nil && openID != "" && strings.TrimSpace(msg.Content) != "" {
		if err := h.chat.RecordInboundChat(ctx, openID, msg.Content); err != nil {
			h.logger.Warn("玩家私聊消息回传失败", zap.String("openid", openID), zap.Error(err))
		}
	}
	return h.receiver.ReceiveMessage(ctx, msg)
}

// OnGroupMessage 分发 GROUP_AT_MESSAGE_CREATE 事件：群消息（仅记录日志）。
// 白名单验证已改为私聊，群消息不再参与绑定。
func (h *Handler) OnGroupMessage(ctx context.Context, msg *dto.Message) error {
	return h.receiver.ReceiveMessage(ctx, msg)
}

// OnPlain 分发未注册事件的兜底日志。
func (h *Handler) OnPlain(event *dto.WSPayload, payload []byte) {
	h.logger.Debug("收到未注册网关事件",
		zap.String("type", string(event.Type)),
		zap.ByteString("payload", payload),
	)
}
