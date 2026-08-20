package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEvent 是 QQ 指令审计记录。
type AuditEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	Command   string    `json:"command"`
	OpenID    string    `json:"openid,omitempty"`
	Source    string    `json:"source,omitempty"`
	Result    string    `json:"result"`
}

// Auditor 持久化 QQ 指令审计记录。
type Auditor interface {
	Record(AuditEvent) error
}

// FileAuditor 将 QQ 指令审计记录以 JSONL 追加到独立文件。
type FileAuditor struct {
	mu   sync.Mutex
	file *os.File
}

// NewFileAuditor 创建 QQ 指令审计文件及其父目录。
func NewFileAuditor(path string) (*FileAuditor, error) {
	if path == "" {
		return nil, fmt.Errorf("QQ 指令审计文件路径不能为空")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("创建 QQ 指令审计目录: %w", err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("打开 QQ 指令审计文件: %w", err)
	}
	return &FileAuditor{file: file}, nil
}

// Record 追加一条审计记录。
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
