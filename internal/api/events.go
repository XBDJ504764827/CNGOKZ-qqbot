package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/event"
)

// EventsHandler 处理 POST /api/v1/events：外部系统事件上报入口。
//
// 认证（X-API-Key）与限流由 internal/auth 中间件负责（见 server.go），
// 本 Handler 只负责：解析 JSON → 校验 → 补全字段 → 发布到 Event Bus。
//
// 处理流程：
//
//	HTTP Request → [auth 中间件] → [限流中间件] → 本 Handler → Event Bus
type EventsHandler struct {
	bus    event.Bus
	logger *zap.Logger
}

// NewEventsHandler 构建事件接收处理器（依赖注入：事件总线、日志）。
func NewEventsHandler(bus event.Bus, logger *zap.Logger) *EventsHandler {
	return &EventsHandler{bus: bus, logger: logger}
}

// ServeHTTP 实现 http.Handler。
func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. 解析 JSON 事件
	var ev event.Event
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		WriteJSON(w, http.StatusBadRequest, "invalid json body")
		return
	}

	// 2. 校验格式（source / event_type 必填，level 取值合法）
	if err := ev.Validate(); err != nil {
		WriteJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	// 3. 补全字段（ID / Timestamp / Level / Data）
	ev.Normalize()

	// 4. 发布到 Event Bus
	if err := h.bus.Publish(r.Context(), ev); err != nil {
		h.logger.Error("事件发布失败",
			zap.String("event_id", ev.ID),
			zap.String("source", ev.Source),
			zap.String("event_type", ev.EventType),
			zap.Error(err),
		)
		WriteJSON(w, http.StatusInternalServerError, "event publish failed")
		return
	}

	// 5. 记录接收日志并响应
	h.logger.Info("event received",
		zap.String("event_id", ev.ID),
		zap.String("source", ev.Source),
		zap.String("event_type", ev.EventType),
		zap.String("level", ev.Level),
	)
	WriteEventAccepted(w, ev.ID)
}

// apiResponse 统一 API 响应格式。
//
//	成功: {"success": true,  "event_id": "..."}
//	失败: {"success": false, "error": "..."}
type apiResponse struct {
	Success bool   `json:"success"`
	EventID string `json:"event_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// WriteEventAccepted 输出事件接收成功响应（202 Accepted）。
func WriteEventAccepted(w http.ResponseWriter, eventID string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(apiResponse{Success: true, EventID: eventID})
}

// WriteJSON 输出统一错误响应：{"success":false,"error":"..."}。
// 供内部及 internal/auth 中间件复用。
func WriteJSON(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{Success: false, Error: msg})
}
