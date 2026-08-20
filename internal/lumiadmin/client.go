// Package lumiadmin 提供 LumiBot 调用 LumiAdmin QQ 集成接口的客户端。
package lumiadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const whitelistStatusPath = "/api/integration/qq/whitelist/status"

// ErrNotConfigured 表示 LumiAdmin 查询接口未配置。
var ErrNotConfigured = errors.New("LumiAdmin 白名单查询接口未配置")

// APIError 表示 LumiAdmin 返回的 HTTP 错误。
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("LumiAdmin 请求失败（HTTP %d）", e.StatusCode)
	}
	return fmt.Sprintf("LumiAdmin 请求失败（HTTP %d）：%s", e.StatusCode, e.Message)
}

// WhitelistStatusQuerier 是白名单状态查询的最小依赖接口。
type WhitelistStatusQuerier interface {
	QueryWhitelistStatus(ctx context.Context, steamInput string) (*WhitelistStatusResponse, error)
}

// WhitelistStatusResponse 是 LumiAdmin 返回的白名单查询结果。
type WhitelistStatusResponse struct {
	SteamID64 string            `json:"steamid64"`
	SteamID   string            `json:"steamid,omitempty"`
	Items     []WhitelistRecord `json:"items"`
}

// WhitelistRecord 是某一次白名单申请/审核记录。
type WhitelistRecord struct {
	ID              string  `json:"id"`
	SteamID64       string  `json:"steamid64"`
	SteamID         string  `json:"steamid,omitempty"`
	Status          string  `json:"status"`
	AppliedAt       string  `json:"applied_at"`
	ApprovedAt      *string `json:"approved_at,omitempty"`
	RejectedAt      *string `json:"rejected_at,omitempty"`
	RevokedAt       *string `json:"revoked_at,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// Client 是 LumiAdmin HTTP 客户端。
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient 构建 LumiAdmin 客户端。
// baseURL 应为 LumiAdmin 根地址，例如 http://127.0.0.1:3001。
func NewClient(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   strings.TrimSpace(token),
		http:    httpClient,
	}
}

// Enabled 表示查询接口所需配置是否完整。
func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != "" && c.token != ""
}

// QueryWhitelistStatus 查询某个 Steam 标识对应的全部白名单历史记录。
func (c *Client) QueryWhitelistStatus(ctx context.Context, steamInput string) (*WhitelistStatusResponse, error) {
	if !c.Enabled() {
		return nil, ErrNotConfigured
	}
	steamInput = strings.TrimSpace(steamInput)
	if steamInput == "" {
		return nil, &APIError{StatusCode: http.StatusBadRequest, Message: "请输入 Steam 标识符"}
	}

	u, err := url.Parse(c.baseURL + whitelistStatusPath)
	if err != nil {
		return nil, fmt.Errorf("构造 LumiAdmin 请求地址失败: %w", err)
	}
	query := u.Query()
	query.Set("steam_input", steamInput)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("构造 LumiAdmin 请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-qq-token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 LumiAdmin 白名单查询接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var envelope struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&envelope)
		return nil, &APIError{StatusCode: resp.StatusCode, Message: envelope.Error}
	}

	var result WhitelistStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 LumiAdmin 白名单查询响应失败: %w", err)
	}
	return &result, nil
}
