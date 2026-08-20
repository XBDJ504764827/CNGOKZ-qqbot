package qqapproval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEvent 是一条白名单 QQ 审批审计记录。
type AuditEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	Event         string    `json:"event"`
	WhitelistID   string    `json:"whitelist_id"`
	Nickname      string    `json:"nickname,omitempty"`
	InteractionID string    `json:"interaction_id,omitempty"`
	OpenID        string    `json:"openid,omitempty"`
	Action        string    `json:"action,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	Result        string    `json:"result,omitempty"`
	Error         string    `json:"error,omitempty"`
}

// Auditor 持久化审批审计记录。
type Auditor interface {
	Record(AuditEvent) error
}

// FileAuditor 将审计记录以 JSON Lines 追加到文件，方便日志采集和后续导入。
type FileAuditor struct {
	mu   sync.Mutex
	file *os.File
}

// NewFileAuditor 创建审计文件及其父目录。
func NewFileAuditor(path string) (*FileAuditor, error) {
	if path == "" {
		return nil, fmt.Errorf("审批审计文件路径不能为空")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("创建审批审计目录: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("打开审批审计文件: %w", err)
	}
	return &FileAuditor{file: f}, nil
}

// Record 追加单条审计记录。
func (a *FileAuditor) Record(event AuditEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	return json.NewEncoder(a.file).Encode(event)
}

// Close 关闭审计文件。
func (a *FileAuditor) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.file.Close()
}
