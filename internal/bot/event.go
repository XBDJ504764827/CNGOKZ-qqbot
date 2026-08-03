package bot

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/websocket"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/handler"
)

// RegisterEvents 注册 QQ 网关事件回调，返回 websocket 鉴权所需的 intent 集合。
//
// 新事件类型（如论坛事件 ForumThread / Post，管理事件等）在此集中注册，
// 并路由到 handler.Handler 接口，业务层无需感知 botgo。
func RegisterEvents(h handler.Handler, logger *zap.Logger) dto.Intent {
	return websocket.RegisterHandlers(
		readyHandler(logger),
		errorNotifyHandler(logger),
		plainHandler(logger),
		atMessageHandler(h, logger),
		groupATMessageHandler(h, logger),
		c2cMessageHandler(h, logger),
	)
}

// readyHandler 网关鉴权成功（连接就绪）事件。
func readyHandler(logger *zap.Logger) event.ReadyHandler {
	return func(_ *dto.WSPayload, data *dto.WSReadyData) {
		logger.Info("QQ 网关连接就绪",
			zap.Int("version", data.Version),
			zap.String("session_id", data.SessionID),
			zap.String("bot_id", data.User.ID),
			zap.String("bot_name", data.User.Username),
		)
	}
}

// errorNotifyHandler 网关连接异常回调（如 invalid session、重连失败）。
func errorNotifyHandler(logger *zap.Logger) event.ErrorNotifyHandler {
	return func(err error) {
		logger.Error("QQ 网关连接异常", zap.Error(err))
	}
}

// plainHandler 未显式注册事件的透传回调，作为兜底排查手段。
func plainHandler(logger *zap.Logger) event.PlainEventHandler {
	return func(event *dto.WSPayload, payload []byte) error {
		logger.Debug("收到未注册网关事件",
			zap.String("type", string(event.Type)),
			zap.ByteString("payload", payload),
		)
		return nil
	}
}

// atMessageHandler 频道内 @机器人 消息。
func atMessageHandler(h handler.Handler, logger *zap.Logger) event.ATMessageEventHandler {
	return func(_ *dto.WSPayload, data *dto.WSATMessageData) error {
		if err := h.HandleATMessage(context.Background(), data); err != nil {
			logger.Warn("处理频道@消息失败", zap.Error(err))
		}
		return nil
	}
}

// groupATMessageHandler 群聊内 @机器人 消息。
func groupATMessageHandler(h handler.Handler, logger *zap.Logger) event.GroupATMessageEventHandler {
	return func(_ *dto.WSPayload, data *dto.WSGroupATMessageData) error {
		if err := h.HandleGroupATMessage(context.Background(), data); err != nil {
			logger.Warn("处理群聊@消息失败", zap.Error(err))
		}
		return nil
	}
}

// c2cMessageHandler 私聊（C2C）消息。
func c2cMessageHandler(h handler.Handler, logger *zap.Logger) event.C2CMessageEventHandler {
	return func(_ *dto.WSPayload, data *dto.WSC2CMessageData) error {
		if err := h.HandleC2CMessage(context.Background(), data); err != nil {
			logger.Warn("处理私聊消息失败", zap.Error(err))
		}
		return nil
	}
}
