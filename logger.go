package jsonrpc

// Logger describes what methods the library needs from a logger to log internal errors
type Logger interface {
	Error(args ...interface{})
}
