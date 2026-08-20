package command

import (
	"os"
	"strings"
	"testing"
)

func TestFileAuditorWritesJSONL(t *testing.T) {
	path := t.TempDir() + "/qq-command-audit.jsonl"
	auditor, err := NewFileAuditor(path)
	if err != nil {
		t.Fatalf("NewFileAuditor() error = %v", err)
	}
	if err := auditor.Record(AuditEvent{Event: "command_bind", Command: "/bind", OpenID: "openid-1", Result: "success"}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if err := auditor.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), `"event":"command_bind"`) || !strings.Contains(string(content), `"openid":"openid-1"`) {
		t.Errorf("audit content = %q", content)
	}
}
