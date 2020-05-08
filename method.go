package jsonrpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/intel-go/fastjson"
	"github.com/lebedevars/di"
)

type (
	Handler interface {
		Provider() interface{}
		SetContext(context.Context)
		GetContext() context.Context
		Handle(interface{}) (interface{}, *Error)
	}

	methodData struct {
		Handler Handler
		Params  interface{}
		Result  interface{}
	}

	methodRepository struct {
		m           sync.RWMutex
		methods     map[string]methodData
		middlewares []Middleware
		container   *di.Container
		logger      Logger
	}

	MethodDeclaration struct {
		Method  string
		Handler Handler
		Params  interface{}
	}
)

// newMethodRepository creates a new method repository
func newMethodRepository(logger Logger, container *di.Container) *methodRepository {
	return &methodRepository{
		m:         sync.RWMutex{},
		methods:   map[string]methodData{},
		container: container,
		logger:    logger,
	}
}

// RegisterMethod adds method to method repository and registers its handler in container
func (mr *methodRepository) registerMethod(declaration *MethodDeclaration) error {
	if declaration.Method == "" {
		return errors.New("method name is empty")
	}

	if declaration.Handler == nil {
		return errors.New("Handler is empty")
	}

	if mr.container == nil {
		return errors.New("container is nil")
	}

	mr.m.Lock()
	defer mr.m.Unlock()
	mr.methods[declaration.Method] = methodData{
		Handler: declaration.Handler,
		Params:  declaration.Params,
	}

	err := mr.container.Register(declaration.Handler.Provider(), di.Scoped)
	if err != nil {
		return err
	}

	return nil
}

func (mr *methodRepository) GetMethodData(r *Request) (methodData, *Error) {
	if r.Method == "" || r.Version != Version {
		return methodData{}, ErrInvalidParams()
	}

	mr.m.RLock()
	md, ok := mr.methods[r.Method]
	mr.m.RUnlock()
	if !ok {
		return methodData{}, ErrMethodNotFound()
	}

	return md, nil
}

func (mr *methodRepository) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rs, batch, err := ParseRequest(r)
	if err != nil {
		err := SendResponse(w, []*Response{
			{
				Version: Version,
				Error:   err,
			},
		}, false)
		if err != nil {
			fmt.Fprint(w, "Failed to encode error objects")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	resp := make([]*Response, len(rs))
	for i := range rs {
		resp[i] = mr.InvokeMethod(r.Context(), rs[i])
	}

	if err := SendResponse(w, resp, batch); err != nil {
		fmt.Fprint(w, "Failed to encode result objects")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (mr *methodRepository) InvokeMethod(c context.Context, r *Request) *Response {
	var md methodData
	res := NewResponse(r)
	md, res.Error = mr.GetMethodData(r)
	if res.Error != nil {
		return res
	}

	container := mr.container.WithContext("tracking_id", r.ID)
	h, err := container.Get(reflect.TypeOf(md.Handler))
	if err != nil {
		mr.logger.Error(err)
		res.Error = ErrInternal()
		return res
	}

	handlerInterface, ok := h.(Handler)
	if !ok {
		mr.logger.Error("Handler type assertion failed")
		res.Error = ErrInternal()
		return res
	}

	handlerInterface.SetContext(c)
	paramsReflectValue := reflect.ValueOf(md.Params)
	if paramsReflectValue.Kind() == reflect.Ptr {
		paramsReflectValue = paramsReflectValue.Elem()
	}

	expectedParamType := paramsReflectValue.Type()
	paramsUnmarshalled := reflect.New(expectedParamType).Interface()
	if r.Params != nil && len(*r.Params) > 0 {
		err = fastjson.Unmarshal(*r.Params, paramsUnmarshalled)
		if err != nil {
			mr.logger.Error(err)
			res.Error = ErrInternal()
			return res
		}
	}

	wrappedHandler := buildChain(handlerInterface, func(handler Handler, params interface{}) (interface{}, *Error) {
		return handler.Handle(params)
	}, mr.middlewares)

	res.Result, res.Error = wrappedHandler(handlerInterface, paramsUnmarshalled)
	if res.Error != nil {
		res.Result = nil
	}
	return res
}
