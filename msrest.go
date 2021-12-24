package mskit

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/libra9z/httprouter"
	"net/http"
	"github.com/libra9z/mskit/v4/grace"
	. "github.com/libra9z/mskit/v4/rest"
	"github.com/libra9z/mskit/v4/trace"
)

var (
	MsRest *grace.MicroService
)

func init() {
	//logger = kitlog.NewLogfmtLogger(os.Stdout)
	MsRest = New()
}

// NewApp returns a new msrest application.
func New() *grace.MicroService {
	router := httprouter.New()
	ms := &grace.MicroService{Router: router, Server: &http.Server{}}
	return ms
}

/**
	包方法:
**/
func RegisterRestService(path string, rest RestService, middlewares ...RestMiddleware) {
	MsRest.RegisterRestService(path, rest, middlewares...)
}

func RegisterServiceWithTracer(path string, rest RestService, tracer trace.Tracer, logger log.Logger, middlewares ...RestMiddleware) {
	MsRest.RegisterServiceWithTracer(path, rest, tracer, logger, middlewares...)
}

func Handler(method, path string, handler http.Handler) {
	MsRest.Router.Handler(method, path, handler)
}
func HandlerFunc(method, path string, handler http.Handler) {
	MsRest.Router.Handler(method, path, handler)
}

func Serve(params ...string) {
	if MsRest != nil {
		MsRest.Serve(params...)
	} else {
		fmt.Printf("no rest service avaliable.\n")
	}
}

func ServeFiles(path string, root http.FileSystem) {
	if MsRest != nil {
		MsRest.Router.ServeFiles(path, root)
	} else {
		fmt.Printf("no rest service avaliable.\n")
	}
}
