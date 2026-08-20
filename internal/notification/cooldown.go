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
	// Reserve 为一次通知发送创建临时占位。
	// 占位在 Commit 前不会开始冷却窗口；发送失败时必须调用 Release。
	// 同一 key 在占位期间不能被再次获取。
	Reserve(key string, window time.Duration) (Reservation, bool)
	// Reset 清空全部冷却状态（测试与未来管理接口使用）。
	Reset()
}

// Reservation 表示一次尚未完成的通知发送。
//
// Commit 必须在所有目标发送成功后调用；失败路径调用 Release，
// 让后续重试重新获得发送机会。方法可以安全地重复调用。
type Reservation interface {
	Commit()
	Release()
}

// MemoryCooldown 基于内存的冷却实现（单实例适用）。
type MemoryCooldown struct {
	mu        sync.Mutex
	last      map[string]time.Time
	pending   map[string]uint64
	nextToken uint64
}

// NewMemoryCooldown 构建内存冷却器。
func NewMemoryCooldown() *MemoryCooldown {
	return &MemoryCooldown{
		last:    make(map[string]time.Time),
		pending: make(map[string]uint64),
	}
}

// Allow 实现 Cooldown 接口。
func (c *MemoryCooldown) Allow(key string, window time.Duration) bool {
	reservation, ok := c.Reserve(key, window)
	if !ok {
		return false
	}
	reservation.Commit()
	return true
}

// Reserve 实现 Cooldown 接口。
func (c *MemoryCooldown) Reserve(key string, window time.Duration) (Reservation, bool) {
	if window <= 0 {
		return noopReservation{}, true
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.pending[key]; ok {
		return nil, false
	}
	now := time.Now()
	if last, ok := c.last[key]; ok && now.Sub(last) < window {
		return nil, false
	}

	c.nextToken++
	token := c.nextToken
	c.pending[key] = token
	return &memoryReservation{cooldown: c, key: key, token: token}, true
}

// memoryReservation 是 MemoryCooldown 的一次发送占位。
type memoryReservation struct {
	cooldown *MemoryCooldown
	key      string
	token    uint64
}

func (r *memoryReservation) Commit() {
	r.cooldown.mu.Lock()
	defer r.cooldown.mu.Unlock()
	if token, ok := r.cooldown.pending[r.key]; ok && token == r.token {
		delete(r.cooldown.pending, r.key)
		r.cooldown.last[r.key] = time.Now()
	}
}

func (r *memoryReservation) Release() {
	r.cooldown.mu.Lock()
	defer r.cooldown.mu.Unlock()
	if token, ok := r.cooldown.pending[r.key]; ok && token == r.token {
		delete(r.cooldown.pending, r.key)
	}
}

// noopReservation 用于关闭冷却时保持统一的提交/释放流程。
type noopReservation struct{}

func (noopReservation) Commit()  {}
func (noopReservation) Release() {}

// Reset 清空全部冷却状态。
func (c *MemoryCooldown) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last = make(map[string]time.Time)
	c.pending = make(map[string]uint64)
}
