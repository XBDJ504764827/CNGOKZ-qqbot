package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/XBDJ504764827/LumiBot/internal/message"
)

// GroupMentionRequest LumiAdmin 请求在群内 @玩家 的请求体。
type GroupMentionRequest struct {
	GroupID       string `json:"group_id"`
	MentionOpenID string `json:"mention_openid"`
	Content       string `json:"content"`
	Operator      string `json:"operator"`
}

// GroupMentionHandler 处理 POST /api/v1/messages/group-mention。
//
// LumiAdmin 管理员在后台点击「群内 @玩家」后调用该接口，由 LumiBot
// 在指定群内 @出该玩家。群 ID 与 openid 必须是绑定时记录的群场景标识。
type GroupMentionHandler struct {
	sender *message.Sender
	logger *zap.Logger
}

// NewGroupMentionHandler 构建群内 @玩家 处理器。
func NewGroupMentionHandler(sender *message.Sender, logger *zap.Logger) *GroupMentionHandler {
	return &GroupMentionHandler{sender: sender, logger: logger}
}

// ServeHTTP 实现 http.Handler。
func (h *GroupMentionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.sender == nil {
		WriteJSON(w, http.StatusServiceUnavailable, "群内通知未启用")
		return
	}

	var req GroupMentionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, "invalid json body")
		return
	}

	req.GroupID = strings.TrimSpace(req.GroupID)
	req.MentionOpenID = strings.TrimSpace(req.MentionOpenID)
	req.Content = strings.TrimSpace(req.Content)
	if req.GroupID == "" || req.MentionOpenID == "" {
		WriteJSON(w, http.StatusBadRequest, "group_id / mention_openid 不能为空")
		return
	}
	if len([]rune(req.Content)) > 200 {
		WriteJSON(w, http.StatusBadRequest, "通知内容不能超过 200 字")
		return
	}

	msg, err := h.sender.SendGroupMention(r.Context(), req.GroupID, req.MentionOpenID, req.Content)
	if err != nil {
		h.logger.Warn("群内@玩家发送失败",
			zap.String("group_id", req.GroupID),
			zap.String("openid", req.MentionOpenID),
			zap.String("operator", req.Operator),
			zap.Error(err),
		)
		WriteJSON(w, http.StatusBadGateway, "发送失败: "+err.Error())
		return
	}

	messageID := ""
	if msg != nil {
		messageID = msg.ID
	}
	h.logger.Info("群内@玩家发送成功",
		zap.String("group_id", req.GroupID),
		zap.String("openid", req.MentionOpenID),
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
