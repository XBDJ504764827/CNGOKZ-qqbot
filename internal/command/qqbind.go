// Package command 处理 QQ 消息中的用户指令。
//
// 当前支持：
// - /bind：用户私聊机器人获取自己的 QQ OpenID（供网站自助绑定）
// - @机器人 <UUID>：用户在 QQ 群发送绑定码，完成 Steam 与 QQ 的绑定
package command

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"go.uber.org/zap"
)

// 绑定码正则：UUID v4（8-4-4-4-12 hex）
var uuidRe = regexpMustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// QQBindHandler 处理 QQ 群中的 @机器人 <绑定码> 消息：
// 从消息中解析绑定码与发送者 openid，调用 LumiAdmin 完成绑定并回复结果。
type QQBindHandler struct {
	sender    MessageSender
	logger    *zap.Logger
	apiURL    string // LumiAdmin API 地址（不含末尾斜杠）
	apiToken  string // LumiAdmin QQ 集成令牌
	http      *http.Client
	groupOnly bool // 仅处理群消息（@机器人场景）

	mu       sync.Mutex
	requests map[string][]time.Time
	now      func() time.Time
}

// QQBindHandlerOptions 构建参数。
type QQBindHandlerOptions struct {
	Sender   MessageSender
	Logger   *zap.Logger
	APIURL   string
	APIToken string
}

// NewQQBindHandler 构建 QQ 群绑定处理器。
func NewQQBindHandler(opts QQBindHandlerOptions) *QQBindHandler {
	return &QQBindHandler{
		sender:    opts.Sender,
		logger:    opts.Logger,
		apiURL:    strings.TrimRight(opts.APIURL, "/"),
		apiToken:  opts.APIToken,
		http:      &http.Client{Timeout: 8 * time.Second},
		groupOnly: true,
		requests:  make(map[string][]time.Time),
		now:       time.Now,
	}
}

// Handle 处理消息：仅在群消息且内容为「绑定码（纯 UUID）」时消费。
//
// 说明：QQ 官方机器人群消息事件 GROUP_AT_MESSAGE_CREATE 的 Content
// 通常包含 @机器人 的文本表示（如 "@bot 或 @BOT"），这里做宽松匹配：
// 提取内容中的第一个 UUID；若内容不含 UUID 则忽略。
func (h *QQBindHandler) Handle(ctx context.Context, msg *dto.Message) (bool, error) {
	if msg == nil || msg.Author == nil {
		return false, nil
	}

	content := strings.TrimSpace(msg.Content)
	code := extractUUID(content)
	if code == "" {
		return false, nil
	}

	// 只在群消息中处理（GroupID 非空）
	if msg.GroupID == "" {
		h.logger.Info("QQ 绑定码消息忽略（非群消息）", zap.String("user_id", msg.Author.ID))
		return false, nil
	}

	openid := msg.Author.ID
	if openid == "" {
		h.logger.Warn("QQ 绑定码消息缺少发送者 OpenID")
		return true, nil
	}

	if !h.allow(openid) {
		h.logger.Warn("QQ 绑定请求过于频繁", zap.String("openid", openid))
		return true, h.reply(ctx, msg, "绑定请求过于频繁，请稍后再试。")
	}

	result := h.bind(ctx, code, openid)
	if err := h.reply(ctx, msg, result); err != nil {
		h.logger.Error("QQ 绑定回复失败", zap.String("openid", openid), zap.Error(err))
		return true, err
	}
	h.logger.Info("QQ 群绑定请求处理完成",
		zap.String("openid", openid),
		zap.String("code", code),
		zap.String("message", result),
	)
	return true, nil
}

// bind 调用 LumiAdmin 绑定 API。
func (h *QQBindHandler) bind(ctx context.Context, code, openid string) string {
	if h.apiURL == "" || h.apiToken == "" {
		h.logger.Warn("LumiAdmin 未配置，无法执行 QQ 绑定")
		return "绑定功能未启用（LumiAdmin 集成未配置），请联系管理员。"
	}

	body := fmt.Sprintf(`{"code":%q,"qq_openid":%q}`, code, openid)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/qq/bind", h.apiURL),
		strings.NewReader(body),
	)
	if err != nil {
		h.logger.Error("构造绑定请求失败", zap.Error(err))
		return "绑定失败，请稍后重试。"
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", h.apiToken)

	resp, err := h.http.Do(req)
	if err != nil {
		h.logger.Warn("LumiAdmin 绑定请求失败", zap.String("code", code), zap.Error(err))
		return "绑定失败：无法连接管理后台，请稍后重试。"
	}
	defer func() { _ = resp.Body.Close() }()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var payload struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	_ = json.Unmarshal(data, &payload)

	if resp.StatusCode != http.StatusOK {
		h.logger.Warn("LumiAdmin 绑定返回异常状态",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(data)),
		)
		return "绑定失败：服务暂时不可用，请稍后重试。"
	}
	if !payload.Success {
		switch payload.Error {
		case "code_invalid":
			return "绑定失败：绑定码不存在，请回网站重新生成。"
		case "code_used":
			return "绑定失败：该绑定码已使用过，请回网站重新生成。"
		case "code_expired":
			return "绑定失败：绑定码已过期（10 分钟有效），请回网站重新生成。"
		default:
			return "绑定失败：" + payload.Message
		}
	}
	return "✅ 绑定成功！已关联您的 QQ 与 Steam 账号，管理员可通过 QQ 联系方式追溯。"
}

// reply 通过私聊向发送者回复绑定结果。
//
// 注意：群消息中无法直接 @ 回复普通用户（QQ 官方机器人在群内仅能回复私聊），
// 因此通过 C2C 私聊告知用户结果。
func (h *QQBindHandler) reply(ctx context.Context, msg *dto.Message, content string) error {
	if h.sender == nil {
		return fmt.Errorf("QQ 绑定回复失败：sender 未配置")
	}
	openid := msg.Author.ID
	if openid == "" {
		return fmt.Errorf("QQ 绑定回复失败：消息缺少用户 OpenID")
	}
	_, err := h.sender.SendC2CMessage(ctx, openid, content)
	return err
}

// allow 限流：单个 OpenID 在窗口内最多 qqBindRateLimit 次。
func (h *QQBindHandler) allow(openid string) bool {
	if openid == "" {
		return true
	}
	now := h.now()
	cutoff := now.Add(-time.Minute)

	h.mu.Lock()
	defer h.mu.Unlock()

	requests := h.requests[openid]
	kept := requests[:0]
	for _, timestamp := range requests {
		if timestamp.After(cutoff) {
			kept = append(kept, timestamp)
		}
	}
	if len(kept) >= bindRateLimit {
		h.requests[openid] = kept
		return false
	}
	h.requests[openid] = append(kept, now)
	return true
}

// extractUUID 从消息内容中提取第一个 UUID v4（宽松：任意 8-4-4-4-12 hex 段）。
func extractUUID(content string) string {
	fields := strings.Fields(content)
	for _, field := range fields {
		field = strings.Trim(field, `"'，,。；;！!@「」『』()（）`)
		if uuidRe.MatchString(field) {
			return strings.ToLower(field)
		}
	}
	return ""
}

// regexpMustCompile 编译正则（防误用 panic）。
func regexpMustCompile(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}
