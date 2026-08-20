package lumiadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const banStatusPath = "/api/integration/qq/ban/status"

// BanStatusQuerier 是 /ban 指令依赖的最小查询接口。
type BanStatusQuerier interface {
	QueryBanStatus(ctx context.Context, steamInput string) (*BanStatusResponse, error)
}

// BanStatusResponse 是 LumiAdmin 返回的本地封禁和全球封禁结果。
type BanStatusResponse struct {
	SteamID64  string            `json:"steamid64"`
	SteamID    string            `json:"steamid,omitempty"`
	LocalBans  []LocalBanRecord  `json:"local_bans"`
	GlobalBans []GlobalBanRecord `json:"global_bans"`
}

// LocalBanRecord 是网站自身的封禁历史记录。
type LocalBanRecord struct {
	ID              string  `json:"id"`
	Player          *string `json:"player,omitempty"`
	SteamID         string  `json:"steam_id"`
	BanType         string  `json:"ban_type"`
	DurationMinutes int     `json:"duration_minutes"`
	ExpiresAt       *string `json:"expires_at,omitempty"`
	Reason          string  `json:"reason"`
	Status          string  `json:"status"`
	Source          string  `json:"source"`
	RemovedReason   *string `json:"removed_reason,omitempty"`
	RemovedAt       *string `json:"removed_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

// GlobalBanRecord 是 KZTimer Global 在 LumiAdmin 本地同步的全球封禁记录。
type GlobalBanRecord struct {
	ID             int64   `json:"id"`
	PlayerName     *string `json:"player_name,omitempty"`
	SteamID64      string  `json:"steamid64"`
	SteamID        *string `json:"steam_id,omitempty"`
	BanType        string  `json:"ban_type"`
	Notes          *string `json:"notes,omitempty"`
	Stats          *string `json:"stats,omitempty"`
	ExpiresOn      *string `json:"expires_on,omitempty"`
	CreatedOn      *string `json:"created_on,omitempty"`
	UpdatedOn      *string `json:"updated_on,omitempty"`
	IsExpired      bool    `json:"is_expired"`
	ManualUnbanned bool    `json:"manual_unbanned"`
}

// QueryBanStatus 查询指定 Steam 标识对应的全部本地/全球封禁历史。
func (c *Client) QueryBanStatus(ctx context.Context, steamInput string) (*BanStatusResponse, error) {
	if !c.Enabled() {
		return nil, ErrNotConfigured
	}
	steamInput = strings.TrimSpace(steamInput)
	if steamInput == "" {
		return nil, &APIError{StatusCode: http.StatusBadRequest, Message: "请输入 Steam 标识符"}
	}

	u, err := url.Parse(c.baseURL + banStatusPath)
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
		return nil, fmt.Errorf("请求 LumiAdmin 封禁查询接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var envelope struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&envelope)
		return nil, &APIError{StatusCode: resp.StatusCode, Message: envelope.Error}
	}

	var result BanStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 LumiAdmin 封禁查询响应失败: %w", err)
	}
	return &result, nil
}
