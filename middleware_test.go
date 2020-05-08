package jsonrpc

import (
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testHandler struct {
	BaseHandler
}

func (h *testHandler) Provider() interface{} {
	return func(logger *zap.SugaredLogger) *TestHandler {
		return &TestHandler{logger: logger}
	}
}
func (h *testHandler) Handle(interface{}) (interface{}, *Error) {
	return nil, nil
}

type ResultAccum struct {
	Pre  []int
	Post []int
}

func NewTestMw(order int, accum *ResultAccum) Middleware {
	return func(h Handler, next MiddlewareFunc) MiddlewareFunc {
		return func(handler Handler, params interface{}) (interface{}, *Error) {
			accum.Pre = append(accum.Pre, order)
			r, err := next(handler, params)
			accum.Post = append(accum.Post, order)
			return r, err
		}
	}
}

// TestBuildChain проверяет, что все хенделеры вызываются в ожиданном порядке
// и ничего не теряется между ними
// и демонстрирует как пользоваться middleware
func TestBuildChain(t *testing.T) {

	accum := &ResultAccum{}
	mw1 := NewTestMw(1, accum)
	mw2 := NewTestMw(2, accum)
	mw3 := NewTestMw(3, accum)
	mw4 := NewTestMw(4, accum)

	lasthandlerCalled := 0
	last := func(handler Handler, params interface{}) (interface{}, *Error) {
		lasthandlerCalled++
		err := ErrInternal()
		err.Message = "Test error"
		return "Test result", err
	}

	h := &testHandler{}
	wrappedHandle := buildChain(h, last, []Middleware{mw1, mw2, mw3, mw4})

	result, err := wrappedHandle(h, nil)

	require.NotNil(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Test error", err.Message)
	assert.Equal(t, "Test result", result)
	assert.Equal(t, 1, lasthandlerCalled)

	assert.Equal(t, []int{1, 2, 3, 4}, accum.Pre)
	assert.Equal(t, []int{4, 3, 2, 1}, accum.Post)

}

func TestBuildEmptyChain(t *testing.T) {
	h := &testHandler{}
	last := func(h Handler, params interface{}) (interface{}, *Error) {
		return "Test result", nil
	}

	assert.NotPanics(t, func() {
		res, err := buildChain(h, last, nil)(h, nil)
		assert.Nil(t, err)
		assert.Equal(t, "Test result", res)
	})

}
