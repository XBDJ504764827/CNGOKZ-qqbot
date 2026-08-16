// Package qqapproval 实现「通过 QQ 聊天审批白名单申请」的完整闭环：
//
//	管理员私聊收到带按钮的新申请 → 点「通过/拒绝」→ 通过直接回写；
//	拒绝则反问原因，管理员打字回复后再回写；并发时由 LumiAdmin 数据库
//	原子保证只有一方成功，其余收到「已被他人审批」提示。
//
// 组件：
//   - BuildKeyboard     为申请构建「通过/拒绝」按钮
//   - HandleInteraction  处理按钮点击（INTERACTION_CREATE 事件）
//   - HandleC2CText      处理「等待填拒绝原因」时的管理员文本回复
//   - 回写走 LumiAdmin 审批接口 POST /api/integration/qq/whitelist/:id/review
package qqapproval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/keyboard"
	"go.uber.org/zap"
)

// Sender 最小发送接口，隔离具体实现以便测试。
type Sender interface {
	SendC2CMessage(ctx context.Context, userID, content string) (*dto.Message, error)
	SendC2CMessageWithKeyboard(ctx context.Context, userID, content string, buttons *keyboard.CustomKeyboard) (*dto.Message, error)
}

// 按钮动作标识（写在一行避免 gofmt 分段）。
const (
	BtnApprove = "approve"
	BtnReject  = "reject"
)

// pendingEntry 待填拒绝原因的一次挂起。
type pendingEntry struct {
	whitelistID string
	nickname    string
	expiresAt   time.Time
}

// Service 审批服务。
type Service struct {
	sender  Sender
	baseURL string // LumiAdmin 地址，如 http://127.0.0.1:8081
	token   string // LumiAdmin QQ Integration Token
	http    *http.Client
	pending map[string]*pendingEntry // openid -> 待填拒绝原因
	names   map[string]string        // whitelistID -> 玩家昵称（按钮点击时还原回复用）
	nameMu  sync.Mutex
	reqTTL  time.Duration
	logger  *zap.Logger
}

// Options 构建参数。
type Options struct {
	Sender     Sender
	BaseURL    string
	Token      string
	PendingTTL time.Duration // 拒绝原因等待超时
	HTTPClient *http.Client
}

// New 构建审批服务。
func New(opts Options, logger *zap.Logger) (*Service, error) {
	if opts.Sender == nil {
		return nil, fmt.Errorf("qqapproval: sender 不能为空")
	}
	ttl := opts.PendingTTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Service{
		sender:  opts.Sender,
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		token:   opts.Token,
		http:    httpClient,
		pending: make(map[string]*pendingEntry),
		names:   make(map[string]string),
		reqTTL:  ttl,
		logger:  logger,
	}, nil
}

// Enabled 是否已配置 LumiAdmin 审批回写（未配置则不启用按钮审批）。
func (s *Service) Enabled() bool {
	return s.baseURL != "" && s.token != ""
}

// reviewPayload 上报给 LumiAdmin 的审批请求体。
type reviewPayload struct {
	Action string `json:"action"`           // approve | reject
	OpenID string `json:"openid"`           // 审批者 QQ openid
	Reason string `json:"reason,omitempty"` // reject 时必填
	Force  bool   `json:"force"`
}

// BuildKeyboard 为一条白名单申请构建「通过/拒绝」按钮。
// whitelistID 用于回写时定位；allowOpenids 预留（当前经 data.openids 天然限定），可暂时忽略。
// 同时登记 whitelistID -> nickname，供按钮点击时还原玩家昵称（QQ 的 button_data 承载不下昵称，
// 且可能含 : 等字符，不能塞进按钮数据，故用服务内部映射按 whitelistID 找回）。
func (s *Service) BuildKeyboard(whitelistID, nickname string, allowOpenids []string) *keyboard.CustomKeyboard {
	s.nameMu.Lock()
	s.names[whitelistID] = nickname
	s.nameMu.Unlock()
	return &keyboard.CustomKeyboard{
		Rows: []*keyboard.Row{
			{Buttons: []*keyboard.Button{
				{ID: fmt.Sprintf("%s:%s:%s", BtnApprove, whitelistID, uuid.NewString()),
					RenderData: &keyboard.RenderData{Label: "通过 ✅", Style: 4},
					Action:     &keyboard.Action{Type: keyboard.ActionTypeCallback, Data: fmt.Sprintf("%s:%s", BtnApprove, whitelistID)}},
				{ID: fmt.Sprintf("%s:%s:%s", BtnReject, whitelistID, uuid.NewString()),
					RenderData: &keyboard.RenderData{Label: "拒绝 ❌", Style: 3},
					Action:     &keyboard.Action{Type: keyboard.ActionTypeCallback, Data: fmt.Sprintf("%s:%s", BtnReject, whitelistID)}},
			}},
		},
	}
}

// BeginRejectReason 发起一次「等待拒绝原因」挂起。
func (s *Service) BeginRejectReason(openid, whitelistID, nickname string) {
	s.pending[openid] = &pendingEntry{
		whitelistID: whitelistID,
		nickname:    nickname,
		expiresAt:   time.Now().Add(s.reqTTL),
	}
}

// PeekPending 读取（不删除）该 openid 的挂起项，超时自动清除。测试用。
func (s *Service) PeekPending(openid string) bool {
	return s.peekPending(openid) != nil
}

func (s *Service) peekPending(openid string) *pendingEntry {
	e, ok := s.pending[openid]
	if !ok {
		return nil
	}
	if time.Now().After(e.expiresAt) {
		delete(s.pending, openid)
		return nil
	}
	return e
}

// nameFor 按 whitelistID 取回登记的玩家昵称；未知则返回空。
func (s *Service) nameFor(whitelistID string) string {
	s.nameMu.Lock()
	defer s.nameMu.Unlock()
	return s.names[whitelistID]
}

func (s *Service) takePending(openid string) *pendingEntry {
	e := s.peekPending(openid)
	if e != nil {
		delete(s.pending, openid)
	}
	return e
}

// review 调用 LumiAdmin 审批接口。
func (s *Service) review(ctx context.Context, whitelistID, action, openid, reason string, force bool) error {
	u := fmt.Sprintf("%s/api/integration/qq/whitelist/%s/review", s.baseURL, url.PathEscape(whitelistID))
	body, _ := json.Marshal(reviewPayload{
		Action: action,
		OpenID: openid,
		Reason: reason,
		Force:  force,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-qq-token", s.token)

	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("请求 LumiAdmin 审批接口失败: %w", err)
	}
	defer resp.Body.Close()

	var envelope struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&envelope)

	if resp.StatusCode == http.StatusConflict {
		return &ReviewError{
			AlreadyReviewed: true,
			Message:         envelope.Error,
		}
	}
	if resp.StatusCode >= 300 || !envelope.OK {
		msg := envelope.Error
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("LumiAdmin 审批失败(%d): %s", resp.StatusCode, msg)
	}
	return nil
}

// ReviewError 审批结果错误。
type ReviewError struct {
	AlreadyReviewed bool
	Message         string
}

func (e *ReviewError) Error() string {
	if e.Message == "" {
		return "该申请已被他人审批，无法重复操作"
	}
	return e.Message
}

// Approve 处理「通过」点击：直接回写。成功返回 ""，否则返回错误。
func (s *Service) Approve(ctx context.Context, whitelistID, openid string) error {
	return s.review(ctx, whitelistID, "approve", openid, "", false)
}

// SubmitRejectReason 提交「拒绝」的原因并回写。返回被拒玩家昵称（无挂起时为空）。
func (s *Service) SubmitRejectReason(ctx context.Context, openid, reason string) (string, error) {
	e := s.takePending(openid)
	if e == nil {
		return "", nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", fmt.Errorf("拒绝原因不能为空")
	}
	err := s.review(ctx, e.whitelistID, "reject", openid, reason, false)
	if err != nil {
		// 失败时恢复挂起，允许重试
		s.BeginRejectReason(openid, e.whitelistID, e.nickname)
		return "", err
	}
	return e.nickname, nil
}

// hasPending 该 openid 是否有未过期的「等待拒绝原因」挂起。
func (s *Service) hasPending(openid string) bool {
	return s.peekPending(openid) != nil
}

// ParseAction 从按钮点击数据解析出 (action, whitelistID)。
// 兼容两种来源：Action.Data（`action:whitelistID`）与 Button.ID（`action:whitelistID:uuid`）。
func ParseAction(buttonData string) (action, whitelistID string, ok bool) {
	if buttonData == "" {
		return "", "", false
	}
	parts := strings.Split(buttonData, ":")
	if len(parts) < 1 {
		return "", "", false
	}
	action = parts[0]
	if action != BtnApprove && action != BtnReject {
		return "", "", false
	}
	if len(parts) < 2 || parts[1] == "" {
		return "", "", false
	}
	return action, parts[1], true
}

// HandleUserText 处理管理员私聊文本：若处于「等待拒绝原因」挂起则消费并提交；否则不消费。
// 返回 (handled, err)，handled=true 表示该条消息已被消费。调用方（receiver）据此跳过日志。
func (s *Service) HandleUserText(ctx context.Context, openid, content string) (bool, error) {
	if !s.hasPending(openid) {
		return false, nil
	}
	content = strings.TrimSpace(content)
	if content == "" {
		_, _ = s.sender.SendC2CMessage(ctx, openid, "拒绝原因不能为空，请重新输入。")
		return true, nil
	}
	nickname, err := s.SubmitRejectReason(ctx, openid, content)
	if err != nil {
		var re *ReviewError
		if ok := errors.As(err, &re); ok && re.AlreadyReviewed {
			_, _ = s.sender.SendC2CMessage(ctx, openid,
				"该申请已被其他管理员审批了，无需重复操作。")
			return true, nil
		}
		s.logger.Warn("QQ 审批：提交拒绝失败", zap.String("openid", openid), zap.Error(err))
		_, _ = s.sender.SendC2CMessage(ctx, openid,
			fmt.Sprintf("提交拒绝失败：%v\n请更换原因后再次发送。", err))
		return true, nil
	}
	if nickname == "" {
		return true, nil
	}
	_, _ = s.sender.SendC2CMessage(ctx, openid,
		fmt.Sprintf("已拒绝玩家「%s」的白名单申请。", nickname))
	return true, nil
}

// DoAction 处理一次已解析的按钮点击（action + whitelistID），并向申请人发送结果回复。
// nickname 为申请玩家昵称（用于提示），为空时按 whitelistID 从登记映射找回。
func (s *Service) DoAction(ctx context.Context, openid, action, whitelistID, nickname string) error {
	if nickname == "" {
		nickname = s.nameFor(whitelistID)
	}
	switch action {
	case BtnApprove:
		err := s.Approve(ctx, whitelistID, openid)
		if err == nil {
			_, _ = s.sender.SendC2CMessage(ctx, openid,
				fmt.Sprintf("玩家「%s」的白名单申请已通过 ✅", nickname))
			return nil
		}
		var re *ReviewError
		if ok := errors.As(err, &re); ok && re.AlreadyReviewed {
			_, _ = s.sender.SendC2CMessage(ctx, openid,
				"该申请已被其他管理员审批了，无需重复操作。")
			return nil
		}
		s.logger.Warn("QQ 审批：通过失败", zap.String("whitelist_id", whitelistID), zap.Error(err))
		_, _ = s.sender.SendC2CMessage(ctx, openid,
			fmt.Sprintf("操作失败：%v", err))
	case BtnReject:
		// 进入「等待拒绝原因」状态，反问原因
		s.BeginRejectReason(openid, whitelistID, nickname)
		_, _ = s.sender.SendC2CMessage(ctx, openid,
			fmt.Sprintf("请回复拒绝玩家「%s」的申请原因（5 分钟内有效）：", nickname))
	default:
		s.logger.Debug("忽略未知按钮动作", zap.String("action", action))
	}
	return nil
}
