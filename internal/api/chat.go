package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/message"
)

// ChatRequest LumiAdmin 请求私聊（C2C）推送玩家的请求体。
type ChatRequest struct {
	OpenID   string `json:"openid"`
	Content  string `json:"content"`
	Operator string `json:"operator"`
}

// ChatHandler 处理 POST /api/v1/messages/chat。
//
// LumiAdmin 管理员在后台聊天面板发送消息时调用，由 LumiBot 私聊推送给玩家。
// 玩家 openid 为绑定时记录的 C2C openid。
type ChatHandler struct {
	sender *message.Sender
	logger *zap.Logger
}

// NewChatHandler 构建私聊推送处理器。
func NewChatHandler(sender *message.Sender, logger *zap.Logger) *ChatHandler {
	return &ChatHandler{sender: sender, logger: logger}
}

// ServeHTTP 实现 http.Handler。
func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.sender == nil {
		WriteJSON(w, http.StatusServiceUnavailable, "私聊推送未启用")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, "invalid json body")
		return
	}

	req.OpenID = strings.TrimSpace(req.OpenID)
	req.Content = strings.TrimSpace(req.Content)
	if req.OpenID == "" || req.Content == "" {
		WriteJSON(w, http.StatusBadRequest, "openid / content 不能为空")
		return
	}
	if len([]rune(req.Content)) > 500 {
		WriteJSON(w, http.StatusBadRequest, "消息内容不能超过 500 字")
		return
	}

	msg, err := h.sender.SendC2CMessage(r.Context(), req.OpenID, req.Content)
	if err != nil {
		h.logger.Warn("私聊推送失败",
			zap.String("openid", req.OpenID),
			zap.String("operator", req.Operator),
			zap.Error(err),
		)
		// 透传底层错误（含 QQ 主动消息权限提示），供 LumiAdmin 落库与展示
		WriteJSON(w, http.StatusBadGateway, "发送失败: "+err.Error())
		return
	}

	messageID := ""
	if msg != nil {
		messageID = msg.ID
	}
	h.logger.Info("私聊推送成功",
		zap.String("openid", req.OpenID),
		zap.String("operator", req.Operator),
		zap.String("message_id", messageID),
	)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":    true,
		"message_id": messageID,
	})
}
