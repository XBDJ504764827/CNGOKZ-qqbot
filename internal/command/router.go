package command

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
)

// Handler 是消息指令处理器。
type Handler interface {
	Handle(ctx context.Context, msg *dto.Message) (handled bool, err error)
}

// Router 按注册顺序分发指令；未匹配的消息交给后续文本处理流程。
type Router struct {
	handlers []Handler
}

func NewRouter(handlers ...Handler) *Router {
	return &Router{handlers: handlers}
}

func (r *Router) Handle(ctx context.Context, msg *dto.Message) (bool, error) {
	for _, handler := range r.handlers {
		if handler == nil {
			continue
		}
		handled, err := handler.Handle(ctx, msg)
		if handled || err != nil {
			return handled, err
		}
	}
	return false, nil
}
