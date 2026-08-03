// Package bot 封装 QQ 官方机器人（botgo SDK）客户端的初始化与生命周期管理：
// openapi 客户端、oauth2 token source、websocket 长连接会话。
package bot

import (
	"context"
	"fmt"

	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/config"
	"github.com/XBDJ504764827/LumiBot/internal/handler"
)

// Client 管理 botgo 连接。
type Client struct {
	cfg     *config.Config
	logger  *zap.Logger
	handler handler.Handler

	openapi openapi.OpenAPI
}

// NewClient 构建机器人客户端（依赖注入：配置、日志、消息处理器）。
func NewClient(cfg *config.Config, logger *zap.Logger, h handler.Handler) *Client {
	return &Client{cfg: cfg, logger: logger, handler: h}
}

// OpenAPI 暴露 openapi 客户端，供消息处理器在后续阶段调用
// （如回复消息、查询频道/群信息、发送管理员通知）。
func (c *Client) OpenAPI() openapi.OpenAPI {
	return c.openapi
}

// Start 建立并维持与 QQ 网关的长连接，阻塞直到连接终止或 ctx 取消。
//
// 注意：botgo 的本地 session manager 不提供停止接口，
// ctx 取消时本方法直接返回，进程随后退出即可终止后台 goroutine。
func (c *Client) Start(ctx context.Context) error {
	// oauth2 token source：appid + secret 换取 access token
	tokenSource := token.NewQQBotTokenSource(&token.QQBotCredentials{
		AppID:     c.cfg.Bot.AppID,
		AppSecret: c.cfg.Bot.Secret,
	})

	// 按运行环境选择沙箱 / 正式网关
	if c.cfg.Env == config.EnvProd {
		c.openapi = botgo.NewOpenAPI(c.cfg.Bot.AppID, tokenSource).WithTimeout(c.cfg.Bot.Timeout)
	} else {
		c.openapi = botgo.NewSandboxOpenAPI(c.cfg.Bot.AppID, tokenSource).WithTimeout(c.cfg.Bot.Timeout)
	}
	c.logger.Info("openapi 初始化完成", zap.String("env", string(c.cfg.Env)))

	// 拉取 websocket 网关信息（含 shard 数量与连接限制）
	wsInfo, err := c.openapi.WS(ctx, nil, "")
	if err != nil {
		return fmt.Errorf("获取 QQ websocket 网关信息失败: %w", err)
	}

	// 注册事件回调，返回鉴权所需的 intent 集合
	intents := RegisterEvents(c.handler, c.logger)
	c.logger.Info("QQ 网关事件处理器注册完成",
		zap.String("url", wsInfo.URL),
		zap.Uint32("shards", wsInfo.Shards),
		zap.Int("intents", int(intents)),
	)

	manager := botgo.NewSessionManager()
	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.Start(wsInfo, tokenSource, &intents)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		c.logger.Info("收到退出信号，停止 QQ 网关连接")
		return ctx.Err()
	}
}
