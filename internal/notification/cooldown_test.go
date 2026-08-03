package notification

import (
	"testing"
	"time"
)

func TestMemoryCooldown_Allow(t *testing.T) {
	c := NewMemoryCooldown()

	// 窗口内第二次调用被拒绝
	if !c.Allow("SERVER_OFFLINE", time.Minute) {
		t.Fatal("first Allow should return true")
	}
	if c.Allow("SERVER_OFFLINE", time.Minute) {
		t.Fatal("second Allow within window should return false")
	}
	// 不同 key 互不影响
	if !c.Allow("SYSTEM_WARNING", time.Minute) {
		t.Fatal("different key should be allowed")
	}
}

func TestMemoryCooldown_WindowExpired(t *testing.T) {
	c := NewMemoryCooldown()

	if !c.Allow("SERVER_OFFLINE", time.Minute) {
		t.Fatal("first Allow should return true")
	}
	// 模拟窗口过期：冷却器不依赖外部时钟，通过 Reset 或短窗口验证
	c.Reset()
	if !c.Allow("SERVER_OFFLINE", time.Minute) {
		t.Fatal("Allow after Reset should return true")
	}
}

func TestMemoryCooldown_ZeroWindow(t *testing.T) {
	c := NewMemoryCooldown()
	for i := 0; i < 3; i++ {
		if !c.Allow("SERVER_OFFLINE", 0) {
			t.Fatalf("Allow with zero window should always return true (call %d)", i+1)
		}
	}
}
