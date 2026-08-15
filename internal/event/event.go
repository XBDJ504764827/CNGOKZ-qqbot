package event

import "context"

// Bus 事件总线接口。
//
// 当前实现：MemoryBus（内存，单实例可用）。
// 扩展能力：Redis Pub/Sub（RedisBus 实现同一接口，替换内存实现即可
// 支持多实例水平扩展，业务代码无需改动）。
type Bus interface {
	// Publish 发布事件，同步分发给该事件类型的所有订阅者。
	// 单个订阅者失败不影响其他订阅者执行。
	Publish(ctx context.Context, event Event) error
	// Subscribe 订阅指定事件类型，返回可退订的 Subscription。
	// 同一事件类型支持多个订阅者，按注册顺序依次执行。
	Subscribe(eventType string, handler Handler) Subscription
}

// Subscription 事件订阅句柄。
type Subscription interface {
	// Unsubscribe 取消订阅，之后不再收到该事件类型的分发。
	Unsubscribe()
}
