package bot

import (
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// Gateway 封装 QQ websocket 网关连接的生命周期。
//
// 连接建立、断线自动重连由 botgo 本地 session manager 内部完成，
// 本层只负责组装参数与启动。
type Gateway struct {
	manager     botgo.SessionManager
	wsInfo      *dto.WebsocketAP
	tokenSource oauth2.TokenSource
	intents     *dto.Intent
	logger      *zap.Logger
}

// NewGateway 构建网关连接。
func NewGateway(wsInfo *dto.WebsocketAP, tokenSource oauth2.TokenSource, intents *dto.Intent, logger *zap.Logger) *Gateway {
	return &Gateway{
		manager:     botgo.NewSessionManager(),
		wsInfo:      wsInfo,
		tokenSource: tokenSource,
		intents:     intents,
		logger:      logger,
	}
}

// Start 建立网关长连接并阻塞监听（断线后自动重连）。
//
// 注意：botgo v0.2.x 的本地 session manager 未提供显式停止接口，
// 进程退出即可终止连接；ctx 取消时的退出流程由上层（Client）负责。
func (g *Gateway) Start() error {
	g.logger.Info("QQ 网关连接启动",
		zap.String("url", g.wsInfo.URL),
		zap.Uint32("shards", g.wsInfo.Shards),
		zap.Int("intents", int(*g.intents)),
	)
	return g.manager.Start(g.wsInfo, g.tokenSource, g.intents)
}
