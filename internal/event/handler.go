package event

import "context"

// Handler 事件处理器接口。
//
// 业务侧处理器（如 internal/notification）实现该接口后，
// 通过 Bus.Subscribe 注册即可接收事件：
//
//	bus.Subscribe(event.EventServerOffline, notifyHandler)
//
// 亦可用 HandlerFunc 快速注册匿名函数：
//
//	bus.Subscribe(event.EventServerOffline,
//	    event.HandlerFunc(func(ctx context.Context, e event.Event) error { ... }))
type Handler interface {
	// HandleEvent 处理单个事件，返回 error 表示处理失败。
	HandleEvent(ctx context.Context, event Event) error
}

// HandlerFunc 函数式适配器，允许将普通函数用作 Handler。
type HandlerFunc func(ctx context.Context, event Event) error

// HandleEvent 实现 Handler 接口。
func (f HandlerFunc) HandleEvent(ctx context.Context, event Event) error {
	return f(ctx, event)
}
