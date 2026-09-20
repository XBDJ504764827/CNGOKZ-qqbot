// Package lumiadmin 封装对 LumiAdmin 后端的调用。
//
// 用途：
//   - 私聊验证码绑定：玩家在网站生成验证码后，私聊机器人发送验证码，
//     LumiBot 将 QQ openid 回传 LumiAdmin 完成 Steam↔QQ 绑定；
//   - 玩家私聊消息回传：按 openid 归属到 Steam，供管理员聊天面板查看。
package lumiadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Config 调用 LumiAdmin 所需配置。
type Config struct {
	BaseURL string
	QQToken string
	Timeout time.Duration
}

// Client LumiAdmin HTTP 客户端。
type Client struct {
	cfg    Config
	client *http.Client
	logger *zap.Logger
}

// NewClient 构建客户端。BaseURL 或 Token 为空时返回 nil（禁用集成）。
func NewClient(baseURL, token string, timeout time.Duration, logger *zap.Logger) *Client {
	if baseURL == "" || token == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{
		cfg:    Config{BaseURL: baseURL, QQToken: token, Timeout: timeout},
		client: &http.Client{Timeout: timeout},
		logger: logger,
	}
}

// BindVerifyRequest QQ 绑定校验请求体。
type BindVerifyRequest struct {
	Code       string `json:"code"`
	QQOpenID   string `json:"qq_openid"`
	QQUsername string `json:"qq_username,omitempty"`
}

// BindOutcome 绑定结果（与 LumiAdmin 的 BindOutcome 序列化对齐）。
type BindOutcome struct {
	Result    string `json:"result"`
	SteamID64 string `json:"steamid64,omitempty"`
	QQOpenID  string `json:"qq_openid,omitempty"`
	Already   bool   `json:"already,omitempty"`
	Max       int    `json:"max,omitempty"`
	Message   string `json:"message,omitempty"`
}

type bindVerifyResponse struct {
	Outcome BindOutcome `json:"outcome"`
}

// VerifyAndBind 调用 LumiAdmin 校验验证码并完成绑定。
func (c *Client) VerifyAndBind(ctx context.Context, req BindVerifyRequest) (BindOutcome, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return BindOutcome{}, fmt.Errorf("序列化绑定请求失败: %w", err)
	}

	url := c.cfg.BaseURL + "/api/integration/qq/bind/verify"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return BindOutcome{}, fmt.Errorf("构建绑定请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-QQ-Token", c.cfg.QQToken)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return BindOutcome{}, fmt.Errorf("请求 LumiAdmin 失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	if resp.StatusCode != http.StatusOK {
		return BindOutcome{}, fmt.Errorf("LumiAdmin 返回 HTTP %d: %s", resp.StatusCode, string(data))
	}

	var parsed bindVerifyResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return BindOutcome{}, fmt.Errorf("解析绑定响应失败: %w", err)
	}
	return parsed.Outcome, nil
}

// ChatInboundRequest 玩家私聊消息回传请求体。
type ChatInboundRequest struct {
	QQOpenID string `json:"qq_openid"`
	Content  string `json:"content"`
}

// RecordInboundChat 将玩家私聊消息回传 LumiAdmin（按 openid 归属到 Steam）。
func (c *Client) RecordInboundChat(ctx context.Context, openID, content string) error {
	return c.recordInbound(ctx, ChatInboundRequest{QQOpenID: openID, Content: content})
}

// recordInbound 内部实现：回传玩家私聊消息。
func (c *Client) recordInbound(ctx context.Context, req ChatInboundRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化私聊消息失败: %w", err)
	}

	url := c.cfg.BaseURL + "/api/integration/qq/chat/inbound"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("构建私聊消息请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-QQ-Token", c.cfg.QQToken)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("请求 LumiAdmin 失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024))
		return fmt.Errorf("LumiAdmin 返回 HTTP %d: %s", resp.StatusCode, string(data))
	}
	return nil
}
