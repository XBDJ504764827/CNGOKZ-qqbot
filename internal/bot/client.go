// Package bot 封装 QQ 官方机器人（botgo SDK）的核心接入：
// openapi 客户端（client.go）、网关连接（gateway.go）、
// 事件注册（event.go）与业务分发（handler.go）。
package bot

import (
	"context"
	"fmt"

	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/XBDJ504764827/LumiBot/internal/config"
)

// Client 是 QQ 官方机器人客户端：openapi 与 gateway 的聚合与生命周期管理。
type Client struct {
	cfg     *config.Config
	logger  *zap.Logger
	api     openapi.OpenAPI
	handler *Handler
	gateway *Gateway
}

// NewClient 构建机器人客户端（依赖注入：配置、日志、openapi、事件分发器）。
func NewClient(cfg *config.Config, logger *zap.Logger, api openapi.OpenAPI, h *Handler) *Client {
	return &Client{cfg: cfg, logger: logger, api: api, handler: h}
}

// NewOpenAPI 构建 openapi 客户端（工厂函数）。
//
// 按运行环境选择沙箱 / 正式网关，BOT_DEBUG=true 时开启 SDK 调试输出。
// 返回的 openapi.OpenAPI 同时满足 message.MessageAPI，可直接注入消息发送器。
func NewOpenAPI(botCfg config.BotConfig, env config.Env, logger *zap.Logger) openapi.OpenAPI {
	tokenSource := newTokenSource(botCfg)

	var api openapi.OpenAPI
	if env == config.EnvProd {
		api = botgo.NewOpenAPI(botCfg.AppID, tokenSource).WithTimeout(botCfg.Timeout)
	} else {
		api = botgo.NewSandboxOpenAPI(botCfg.AppID, tokenSource).WithTimeout(botCfg.Timeout)
	}

	if botCfg.Debug {
		api = api.SetDebug(true)
		logger.Warn("BOT_DEBUG 已开启，openapi 调试输出启用（请勿在生产环境长期开启）")
	}
	return api
}

// OpenAPI 暴露 openapi 客户端（供外部组件使用）。
func (c *Client) OpenAPI() openapi.OpenAPI {
	return c.api
}

// Start 启动机器人：拉取网关信息 → 注册事件 → 连接 QQ 网关。
//
// 阻塞直到网关连接终止或 ctx 取消（优雅关闭）。
// 连接建立后的断线重连由 botgo session manager 内部完成。
func (c *Client) Start(ctx context.Context) error {
	// 1. 拉取 websocket 网关信息（含 shard 数量与连接限制）
	wsInfo, err := c.api.WS(ctx, nil, "")
	if err != nil {
		return fmt.Errorf("获取 QQ websocket 网关信息失败: %w", err)
	}

	// 2. 注册事件回调，返回鉴权所需的 intent 集合
	intents := RegisterEvents(c.handler, c.logger)
	c.logger.Info("QQ 网关事件处理器注册完成",
		zap.String("url", wsInfo.URL),
		zap.Uint32("shards", wsInfo.Shards),
		zap.Int("intents", int(intents)),
	)

	// 3. 创建网关连接并启动
	c.gateway = NewGateway(wsInfo, newTokenSource(c.cfg.Bot), &intents, c.logger)

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.gateway.Start()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		c.logger.Info("收到退出信号，停止 QQ 网关连接")
		return ctx.Err()
	}
}

// newTokenSource 构建 oauth2 token source（appid + secret 换取 access token）。
func newTokenSource(botCfg config.BotConfig) oauth2.TokenSource {
	return token.NewQQBotTokenSource(&token.QQBotCredentials{
		AppID:     botCfg.AppID,
		AppSecret: botCfg.Secret,
	})
}
