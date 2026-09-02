package command

import (
	"context"

	"github.com/tencent-connect/botgo/dto"
)

// Handler 是消息指令处理器。
// handled=true 表示该指令已被消费（无论是否成功），不再交给后续流程。
type Handler interface {
	Handle(ctx context.Context, msg *dto.Message) (handled bool, err error)
}

// Router 按注册顺序分发指令；未匹配的消息交给后续流程。
type Router struct {
	handlers []Handler
}

// NewRouter 构建指令路由器。
func NewRouter(handlers ...Handler) *Router {
	return &Router{handlers: handlers}
}

// Handle 依次尝试各处理器；命中或出错即返回。
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
