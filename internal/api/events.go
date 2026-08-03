package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/event"
)

// EventsHandler 处理 POST /api/v1/events：外部系统事件上报入口。
//
// 处理流程：
//
//	HTTP Request → 鉴权（X-API-Key）→ 解析 JSON → 校验 → 补全字段
//	→ 发布到 Event Bus → 202 Accepted
type EventsHandler struct {
	bus    event.Bus
	keys   map[string]struct{} // 合法 API Key 集合
	logger *zap.Logger
}

// NewEventsHandler 构建事件接收处理器（依赖注入：事件总线、API Key 列表、日志）。
func NewEventsHandler(bus event.Bus, apiKeys []string, logger *zap.Logger) *EventsHandler {
	keys := make(map[string]struct{}, len(apiKeys))
	for _, k := range apiKeys {
		if k != "" {
			keys[k] = struct{}{}
		}
	}
	return &EventsHandler{bus: bus, keys: keys, logger: logger}
}

// ServeHTTP 实现 http.Handler。
func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. API Key 鉴权（当前简单校验，后续扩展 JWT / 签名验证）
	if !h.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// 2. 解析 JSON 事件
	var ev event.Event
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	// 3. 校验格式（source / event_type 必填，level 取值合法）
	if err := ev.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// 4. 补全字段（ID / Timestamp / Level / Data）
	ev.Normalize()

	// 5. 发布到 Event Bus
	if err := h.bus.Publish(r.Context(), ev); err != nil {
		h.logger.Error("事件发布失败",
			zap.String("event_id", ev.ID),
			zap.String("source", ev.Source),
			zap.String("event_type", ev.EventType),
			zap.Error(err),
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "event publish failed"})
		return
	}

	// 6. 记录接收日志并响应
	h.logger.Info("event received",
		zap.String("event_id", ev.ID),
		zap.String("source", ev.Source),
		zap.String("event_type", ev.EventType),
		zap.String("level", ev.Level),
	)
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":   "accepted",
		"event_id": ev.ID,
	})
}

// authorized 校验 X-API-Key 是否合法（fail-closed：未配置 Key 时拒绝所有请求）。
func (h *EventsHandler) authorized(r *http.Request) bool {
	if len(h.keys) == 0 {
		return false
	}
	key := r.Header.Get("X-API-Key")
	_, ok := h.keys[key]
	return ok
}

// writeJSON 输出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
