// Package api 提供 LumiBot 内置 HTTP 服务。
//
// 路由：
//   - GET  /health        健康检查
//   - POST /api/v1/events 外部系统事件上报（Event Bus 入口，见 events.go）
//   - POST /api/message/send（预留）LumiAdmin 通知推送
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/auth"
	"github.com/XBDJ504764827/LumiBot/internal/config"
	"github.com/XBDJ504764827/LumiBot/internal/event"
)

// Server 内置 HTTP 服务。
type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
}

// NewServer 构建 HTTP 服务并注册路由（依赖注入：事件总线 + 事件 API 配置）。
func NewServer(cfg config.HTTPConfig, eventCfg config.EventConfig, logger *zap.Logger, bus event.Bus) *Server {
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("GET /health", handleHealth)

	// 外部系统事件上报（统一事件系统入口）
	// 中间件链：认证（X-API-Key）→ 限流（按 Key）→ 业务 Handler
	rateWindow := eventCfg.RateWindow
	if rateWindow <= 0 {
		rateWindow = time.Minute // 兜底：未配置时固定 1 分钟窗口
	}
	var events http.Handler = NewEventsHandler(bus, logger)
	events = auth.RateLimitMiddleware(
		auth.NewMemoryRateLimiter(eventCfg.RateLimit, rateWindow),
		logger,
	)(events)
	events = auth.NewAuthenticator(eventCfg.APIKeys).Middleware(logger)(events)
	mux.Handle("POST /api/v1/events", events)

	// 预留：LumiAdmin 通知推送
	// mux.HandleFunc("POST /api/message/send", s.handleMessageSend)

	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Addr,
			Handler:      mux,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		logger: logger,
	}
}

// Start 启动 HTTP 服务，阻塞直到服务退出或 ctx 取消（优雅关闭）。
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("HTTP 服务启动", zap.String("addr", s.httpServer.Addr))
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		s.logger.Info("收到退出信号，HTTP 服务优雅关闭")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	}
}

// handleHealth 健康检查：GET /health → {"status":"ok"}。
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
