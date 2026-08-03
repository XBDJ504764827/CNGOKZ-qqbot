package notification

import (
	"sync"
	"time"
)

// Cooldown 通知冷却接口：同一事件类型在冷却窗口内的重复事件只放行一次。
//
// 当前实现：MemoryCooldown（进程内）。
// 扩展能力：Redis 实现同一接口（如 RedisCooldown），支持多实例共享冷却状态，
// 业务层无需改动。
type Cooldown interface {
	// Allow 判断 key 是否允许发送通知。
	// 距离上次放行超过 window 时返回 true 并记录本次时间；否则返回 false。
	// window <= 0 时始终放行。
	Allow(key string, window time.Duration) bool
	// Reset 清空全部冷却状态（测试与未来管理接口使用）。
	Reset()
}

// MemoryCooldown 基于内存的冷却实现（单实例适用）。
type MemoryCooldown struct {
	mu   sync.Mutex
	last map[string]time.Time
}

// NewMemoryCooldown 构建内存冷却器。
func NewMemoryCooldown() *MemoryCooldown {
	return &MemoryCooldown{last: make(map[string]time.Time)}
}

// Allow 实现 Cooldown 接口。
func (c *MemoryCooldown) Allow(key string, window time.Duration) bool {
	if window <= 0 {
		return true
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	last, ok := c.last[key]
	if ok && now.Sub(last) < window {
		return false
	}
	c.last[key] = now
	return true
}

// Reset 清空全部冷却状态。
func (c *MemoryCooldown) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last = make(map[string]time.Time)
}
