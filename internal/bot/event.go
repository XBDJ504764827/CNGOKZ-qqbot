package bot

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/websocket"
	"go.uber.org/zap"
)

// RegisterEvents 注册 QQ 网关事件回调，返回 websocket 鉴权所需的 intent 集合。
//
// 当前支持的事件（每个事件独立 Handler，统一分发到 *Handler）：
//   - READY                网关连接就绪
//   - MESSAGE_CREATE       频道消息
//   - AT_MESSAGE_CREATE    频道内 @机器人 消息
//   - C2C_MESSAGE_CREATE   私聊消息（用于获取用户 openid / 绑定）
//   - ERROR_NOTIFY         网关连接异常（SDK 内部回调）
//   - PLAIN                未注册事件兜底（透传日志）
//
// 后续阶段新增事件（群@消息、论坛、服务器状态等）在此集中注册。
func RegisterEvents(h *Handler, logger *zap.Logger) dto.Intent {
	return websocket.RegisterHandlers(
		readyHandler(h),
		errorNotifyHandler(h),
		messageCreateHandler(h, logger),
		atMessageHandler(h, logger),
		c2cMessageHandler(h, logger),
		interactionHandler(h, logger),
		plainHandler(h),
	)
}

// readyHandler READY 事件：网关鉴权成功（连接就绪）。
func readyHandler(h *Handler) event.ReadyHandler {
	return func(_ *dto.WSPayload, data *dto.WSReadyData) {
		h.OnReady(data)
	}
}

// errorNotifyHandler 网关连接异常回调（如 invalid session、重连失败）。
func errorNotifyHandler(h *Handler) event.ErrorNotifyHandler {
	return func(err error) {
		h.OnError(err)
	}
}

// messageCreateHandler MESSAGE_CREATE 事件：收到频道消息。
func messageCreateHandler(h *Handler, logger *zap.Logger) event.MessageEventHandler {
	return func(_ *dto.WSPayload, data *dto.WSMessageData) error {
		if err := h.OnMessage(context.Background(), (*dto.Message)(data)); err != nil {
			logger.Warn("处理 MESSAGE_CREATE 事件失败", zap.Error(err))
		}
		return nil
	}
}

// atMessageHandler AT_MESSAGE_CREATE 事件：频道内 @机器人 消息。
func atMessageHandler(h *Handler, logger *zap.Logger) event.ATMessageEventHandler {
	return func(_ *dto.WSPayload, data *dto.WSATMessageData) error {
		if err := h.OnATMessage(context.Background(), (*dto.Message)(data)); err != nil {
			logger.Warn("处理 AT_MESSAGE_CREATE 事件失败", zap.Error(err))
		}
		return nil
	}
}

// c2cMessageHandler C2C_MESSAGE_CREATE 事件：用户私聊机器人。
// 私聊事件中的 author.ID 即用户 openid（C2C 私聊推送的目标），
// 用于获取管理员 openid 配置与未来的用户绑定流程。
func c2cMessageHandler(h *Handler, logger *zap.Logger) event.C2CMessageEventHandler {
	return func(_ *dto.WSPayload, data *dto.WSC2CMessageData) error {
		if err := h.OnC2CMessage(context.Background(), (*dto.Message)(data)); err != nil {
			logger.Warn("处理 C2C_MESSAGE_CREATE 事件失败", zap.Error(err))
		}
		return nil
	}
}

// interactionHandler INTERACTION_CREATE 事件：消息按钮点击等互动。
// 用于 QQ 聊天审批白名单（通过/拒绝按钮）。
func interactionHandler(h *Handler, logger *zap.Logger) event.InteractionEventHandler {
	return func(_ *dto.WSPayload, data *dto.WSInteractionData) error {
		if err := h.OnInteraction(context.Background(), data); err != nil {
			logger.Warn("处理 INTERACTION_CREATE 事件失败", zap.Error(err))
		}
		return nil
	}
}

// plainHandler 未显式注册事件的透传回调，作为兜底排查手段。
func plainHandler(h *Handler) event.PlainEventHandler {
	return func(event *dto.WSPayload, payload []byte) error {
		h.OnPlain(event, payload)
		return nil
	}
}
