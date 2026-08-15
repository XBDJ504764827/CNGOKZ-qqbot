package event

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/zap"
)

// MemoryBus 基于内存的进程内事件总线实现。
//
// 特性：
//   - 同一事件类型支持多个订阅者（按注册顺序同步执行）
//   - 单个订阅者失败不影响其他订阅者（错误记录日志并聚合返回）
//   - 订阅 / 退订线程安全
//
// 适用于单实例部署；多实例部署时可替换为 Redis Bus 实现。
type MemoryBus struct {
	logger *zap.Logger

	mu          sync.RWMutex
	subscribers map[string][]*subscriptionEntry
	nextID      uint64
}

// subscriptionEntry 单个订阅条目。
type subscriptionEntry struct {
	id      uint64
	handler Handler
}

// NewMemoryBus 构建内存事件总线。
func NewMemoryBus(logger *zap.Logger) *MemoryBus {
	return &MemoryBus{
		logger:      logger,
		subscribers: make(map[string][]*subscriptionEntry),
	}
}

// Publish 发布事件，同步分发给该事件类型的所有订阅者。
func (b *MemoryBus) Publish(ctx context.Context, event Event) error {
	entries := b.snapshot(event.EventType)
	if len(entries) == 0 {
		b.logger.Debug("事件无订阅者",
			zap.String("event_id", event.ID),
			zap.String("event_type", event.EventType),
		)
		return nil
	}

	var errs []error
	for _, entry := range entries {
		if err := entry.handler.HandleEvent(ctx, event); err != nil {
			b.logger.Error("事件处理失败",
				zap.String("event_id", event.ID),
				zap.String("event_type", event.EventType),
				zap.Error(err),
			)
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Subscribe 订阅指定事件类型，同一类型可注册多个订阅者。
func (b *MemoryBus) Subscribe(eventType string, handler Handler) Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	entry := &subscriptionEntry{id: b.nextID, handler: handler}
	b.subscribers[eventType] = append(b.subscribers[eventType], entry)

	return &memorySubscription{bus: b, eventType: eventType, id: entry.id}
}

// snapshot 返回指定事件类型的订阅者快照（避免执行期间持锁）。
func (b *MemoryBus) snapshot(eventType string) []*subscriptionEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]*subscriptionEntry(nil), b.subscribers[eventType]...)
}

// remove 移除指定订阅条目。
func (b *MemoryBus) remove(eventType string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entries := b.subscribers[eventType]
	for i, entry := range entries {
		if entry.id == id {
			b.subscribers[eventType] = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}

// memorySubscription 内存订阅句柄。
type memorySubscription struct {
	bus       *MemoryBus
	eventType string
	id        uint64
	once      sync.Once
}

// Unsubscribe 取消订阅（幂等，可多次调用）。
func (s *memorySubscription) Unsubscribe() {
	s.once.Do(func() {
		s.bus.remove(s.eventType, s.id)
	})
}
