package jsonrpc

// MiddlewareFunc is a function that wraps a call to Handler.Handle()
type MiddlewareFunc func(h Handler, params interface{}) (interface{}, *Error)

// Middleware is a function that allows chaining of MiddlewareFuncs and effectively represents a middleware
type Middleware func(h Handler, next MiddlewareFunc) MiddlewareFunc

// buildChain - a function that takes a handler function, a list of middlewares
// and creates a new application stack as a single MiddlewareFunc
// first element of the array will be called first in stack
func buildChain(h Handler, last MiddlewareFunc, m []Middleware) MiddlewareFunc {
	// if there are no more middlewares, we just return the
	// handlerfunc, as we are done recursing.
	if len(m) == 0 {
		return last
	}
	// otherwise pop the middleware from the list,
	// and call build chain recursively as it's parameter
	tailHandler := buildChain(h, last, m[1:])
	return m[0](h, tailHandler)
}
