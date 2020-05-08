package jsonrpc

import (
	"fmt"
	"net/http"

	"github.com/lebedevars/di"
)

// Mux is a wrapper for http.ServeMux which calls JSON-RPC handlers for registered methods
type Mux struct {
	*http.ServeMux
	container *di.Container
	logger    Logger
}

// NewJsonRpcMux returns a new JSON-RPC mux
func NewJsonRpcMux(mux *http.ServeMux, container *di.Container, logger Logger) *Mux {
	return &Mux{ServeMux: mux, container: container, logger: logger}
}

// RegisterMethods registers methods and wraps them with middlewars
func (s *Mux) RegisterMethods(pattern string, methods []*MethodDeclaration, middlewares ...Middleware) error {
	mr := newMethodRepository(s.logger, s.container)
	for _, method := range methods {
		err := mr.registerMethod(method)
		if err != nil {
			return fmt.Errorf("error registering method: %w", err)
		}
	}

	mr.middlewares = middlewares
	s.Handle(pattern, mr)
	return nil
}
