// LumiBot 是 CNGOKZ 社区生态中的 QQ 官方机器人服务。
//
// 本文件仅负责依赖装配与生命周期管理（依赖注入），
// 具体能力分别由 internal/ 下的 config、logger、bot、message、api 包提供。
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tencent-connect/botgo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/XBDJ504764827/LumiBot/internal/api"
	"github.com/XBDJ504764827/LumiBot/internal/bot"
	"github.com/XBDJ504764827/LumiBot/internal/config"
	"github.com/XBDJ504764827/LumiBot/internal/event"
	"github.com/XBDJ504764827/LumiBot/internal/logger"
	"github.com/XBDJ504764827/LumiBot/internal/message"
	"github.com/XBDJ504764827/LumiBot/internal/notification"
	"github.com/XBDJ504764827/LumiBot/internal/qqapproval"
)

// main 启动流程：
//
//	加载配置 → 初始化 Logger → 创建 Bot Client → 注册事件 Handler → 连接 QQ Gateway
//
// 监听 SIGINT / SIGTERM，优雅关闭（关闭 Gateway 连接、HTTP 服务）。
func main() {
	// 1. 加载配置（.env 文件 + 环境变量）
	cfg, err := config.Load()
	if err != nil {
		fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化结构化日志
	zapLogger, err := logger.New(cfg.Log)
	if err != nil {
		fatalf("初始化日志失败: %v", err)
	}
	defer func() { _ = zapLogger.Sync() }()

	// 3. 将 zap 注入 botgo SDK，统一日志输出
	botgo.SetLogger(logger.NewBotgoAdapter(zapLogger))

	// 4. 依赖装配（依赖注入）
	openAPI := bot.NewOpenAPI(cfg.Bot, cfg.Env, zapLogger)    // 创建 Bot Client 基础：openapi
	sender := message.NewSender(openAPI, zapLogger)           // 消息发送能力
	receiver := message.NewReceiver(zapLogger)                // 消息接收处理
	botHandler := bot.NewHandler(receiver, sender, zapLogger) // 注册事件 Handler（业务分发）
	botClient := bot.NewClient(cfg, zapLogger, openAPI, botHandler)

	// QQ 聊天审批服务（白名单按钮审批 → 回写 LumiAdmin）
	approvalSvc, err := qqapproval.New(qqapproval.Options{
		Sender:           sender,
		InteractionAcker: openAPI,
		BaseURL:          cfg.LumiAdmin.CallbackBaseURL,
		Token:            cfg.LumiAdmin.IntegrationToken,
	}, zapLogger)
	if err != nil {
		fatalf("初始化 QQ 审批服务失败: %v", err)
	}
	if approvalSvc.Enabled() {
		// 按钮点击 → 审批
		botHandler.SetInteractionHandler(approvalSvc.HandleInteraction)
		// 私聊文本 → 「等待填拒绝原因」时消费
		receiver.OnUserText = approvalSvc.HandleUserText
	} else {
		zapLogger.Warn("LumiAdmin 审批未配置（LUMIADMIN_CALLBACK_URL / LUMIADMIN_QQ_TOKEN），按钮审批已禁用")
	}

	// 统一事件系统：Event Bus + 通知服务（订阅关键事件，规则/模板/冷却见 internal/notification）
	eventBus := event.NewMemoryBus(zapLogger)
	notifyService := notification.NewServiceWithButtons(cfg.Notification, sender, approvalSvc, zapLogger)
	eventBus.Subscribe(event.EventSystemWarning, notifyService)
	eventBus.Subscribe(event.EventServerOffline, notifyService)
	eventBus.Subscribe(event.EventServerOnline, notifyService)
	eventBus.Subscribe(event.EventForumReportCreated, notifyService)
	eventBus.Subscribe(event.EventAdminAction, notifyService)
	eventBus.Subscribe(event.EventWhitelistRequestCreated, notifyService)

	httpServer := api.NewServer(cfg.HTTP, cfg.Event, zapLogger, eventBus)

	// 5. 生命周期管理：监听退出信号，优雅关闭
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error { return httpServer.Start(ctx) })
	eg.Go(func() error { return botClient.Start(ctx) }) // 连接 QQ Gateway

	zapLogger.Info("LumiBot 启动完成",
		zap.String("env", string(cfg.Env)),
		zap.String("http_addr", cfg.HTTP.Addr),
	)

	if err := eg.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			zapLogger.Info("LumiBot 已优雅退出")
			return
		}
		zapLogger.Error("LumiBot 异常退出", zap.Error(err))
		os.Exit(1)
	}
	zapLogger.Info("LumiBot 已退出")
}

// fatalf 在日志系统初始化前输出致命错误并退出。
func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "%s [FATAL] %s\n",
		time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
	os.Exit(1)
}
