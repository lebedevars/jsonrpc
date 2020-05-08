package jsonrpc

import (
	"log"
	"net/http"
	"time"

	"github.com/lebedevars/di"
	"go.uber.org/zap"
)

type TestHandler struct {
	BaseHandler
	logger *zap.SugaredLogger
}

type TestParams struct {
	Text string
}

type TestResult struct {
	Text string
}

func (h *TestHandler) Provider() interface{} {
	return func(logger *zap.SugaredLogger) *TestHandler {
		return &TestHandler{logger: logger}
	}
}

func (h *TestHandler) Handle(params interface{}) (interface{}, *Error) {
	h.logger.Info("hello world")
	p := params.(*TestParams)
	return TestResult{Text: p.Text}, nil
}

func main() {
	zapLogger, _ := zap.NewDevelopment()
	logger := zapLogger.Sugar()

	c := di.NewContainer()
	err := c.Register(func(params di.ContextParams) *zap.SugaredLogger {
		return logger.With("tracking_id", params.GetValue("tracking_id"))
	}, di.Scoped)

	jsonRpcMux := NewJsonRpcMux(http.NewServeMux(), c, logger)
	err = jsonRpcMux.RegisterMethods("/rpc/v1/", []*MethodDeclaration{{
		Method:  "test",
		Handler: &TestHandler{},
		Params:  &TestParams{},
	}})
	if err != nil {
		logger.Fatal(err)
	}

	err = c.Build()
	if err != nil {
		logger.Fatal(err)
	}

	jsonRpcMux.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      jsonRpcMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
