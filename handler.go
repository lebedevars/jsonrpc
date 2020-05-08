package jsonrpc

import "context"

type (
	BaseHandler struct {
		Ctx context.Context
	}
)

func (h *BaseHandler) SetContext(ctx context.Context) {
	h.Ctx = ctx
}

func (h *BaseHandler) GetContext() context.Context {
	return h.Ctx
}
