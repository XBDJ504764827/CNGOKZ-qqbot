// LumiBot 是 CNGOKZ 社区生态中的 QQ 官方机器人服务。
//
// 本文件仅负责依赖装配与生命周期管理（依赖注入），
// 具体能力分别由 internal/ 下的 config、logger、bot、handler、api 包提供。
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
	"github.com/XBDJ504764827/LumiBot/internal/handler"
	"github.com/XBDJ504764827/LumiBot/internal/logger"
)

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
	msgHandler := handler.NewDefaultHandler(zapLogger)
	botClient := bot.NewClient(cfg, zapLogger, msgHandler)
	httpServer := api.NewServer(cfg.HTTP, zapLogger)

	// 5. 生命周期管理：监听退出信号，优雅关闭
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error { return httpServer.Start(ctx) })
	eg.Go(func() error { return botClient.Start(ctx) })

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
