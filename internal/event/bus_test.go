package event

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func newTestBus() *MemoryBus {
	return NewMemoryBus(zap.NewNop())
}

// TestPublish_SingleSubscriber 验证发布事件后订阅者收到事件。
func TestPublish_SingleSubscriber(t *testing.T) {
	bus := newTestBus()
	var got *Event
	bus.Subscribe("TYPE_A", HandlerFunc(func(_ context.Context, e Event) error {
		got = &e
		return nil
	}))

	ev := Event{ID: "evt-1", Source: "Test", EventType: "TYPE_A", Level: LevelInfo}
	if err := bus.Publish(context.Background(), ev); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if got == nil || got.ID != "evt-1" {
		t.Fatalf("subscriber got %+v, want event evt-1", got)
	}
}

// TestPublish_MultipleSubscribers 验证同一事件多个订阅者全部执行。
func TestPublish_MultipleSubscribers(t *testing.T) {
	bus := newTestBus()

	var mu sync.Mutex
	executed := make([]string, 0, 3)
	for _, name := range []string{"handler-1", "handler-2", "handler-3"} {
		name := name
		bus.Subscribe("SERVER_OFFLINE", HandlerFunc(func(_ context.Context, _ Event) error {
			mu.Lock()
			executed = append(executed, name)
			mu.Unlock()
			return nil
		}))
	}

	if err := bus.Publish(context.Background(), Event{ID: "evt-2", EventType: "SERVER_OFFLINE"}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if len(executed) != 3 {
		t.Fatalf("executed %d handlers, want 3: %v", len(executed), executed)
	}
}

// TestPublish_HandlerFailure 验证单个订阅者失败不影响其他订阅者。
func TestPublish_HandlerFailure(t *testing.T) {
	bus := newTestBus()

	var otherCalled bool
	bus.Subscribe("TYPE_A", HandlerFunc(func(_ context.Context, _ Event) error {
		return errors.New("handler failed")
	}))
	bus.Subscribe("TYPE_A", HandlerFunc(func(_ context.Context, _ Event) error {
		otherCalled = true
		return nil
	}))

	err := bus.Publish(context.Background(), Event{EventType: "TYPE_A"})
	if err == nil {
		t.Fatal("Publish() expected error from failing handler, got nil")
	}
	if !otherCalled {
		t.Fatal("second handler should still be executed after first failure")
	}
}

// TestSubscribe_Unsubscribe 验证退订后不再收到事件。
func TestSubscribe_Unsubscribe(t *testing.T) {
	bus := newTestBus()

	var count int
	sub := bus.Subscribe("TYPE_A", HandlerFunc(func(_ context.Context, _ Event) error {
		count++
		return nil
	}))

	ev := Event{EventType: "TYPE_A"}
	_ = bus.Publish(context.Background(), ev)
	sub.Unsubscribe()
	_ = bus.Publish(context.Background(), ev)
	sub.Unsubscribe() // 幂等

	if count != 1 {
		t.Fatalf("handler executed %d times, want 1 (after unsubscribe)", count)
	}
}

// TestPublish_UnsubscribedType 验证未订阅的事件类型不执行任何 handler。
func TestPublish_UnsubscribedType(t *testing.T) {
	bus := newTestBus()
	var called bool
	bus.Subscribe("TYPE_A", HandlerFunc(func(_ context.Context, _ Event) error {
		called = true
		return nil
	}))

	if err := bus.Publish(context.Background(), Event{EventType: "TYPE_B"}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if called {
		t.Fatal("handler should not be called for unsubscribed event type")
	}
}
