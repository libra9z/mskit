package mskit

import (
	"context"
	"errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"
	"github.com/go-kit/kit/tracing/zipkin"
	mshttp "github.com/go-kit/kit/transport/http"
	"github.com/libra9z/httprouter"
	stdopentracing "github.com/opentracing/opentracing-go"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"net"
	"net/http"
	"fmt"
	//"os"
	"strconv"
	"time"
)


// App defines msrest application with a new PatternServeMux.
type MicroService struct {
	Router *httprouter.Router
	Server *http.Server
	logger log.Logger
	tracer stdopentracing.Tracer
	zipkinTracer *stdzipkin.Tracer
}

var (
	// BeeApp is an application instance
	MsRest *MicroService
)

func init() {
	//logger = kitlog.NewLogfmtLogger(os.Stdout)
	MsRest = New()
}


// NewApp returns a new msrest application.
func New() *MicroService {
	router := httprouter.New()
	ms := &MicroService{Router: router, Server: &http.Server{}}
	return ms
}


func regRoute(r *httprouter.Router,path string, handler http.Handler) {

	r.Handler("GET", path, handler)
	r.Handler("POST", path, handler)
	r.Handler("PUT", path, handler)
	r.Handler("PATCH", path, handler)
	r.Handler("DELETE", path, handler)
	r.Handler("HEAD", path, handler)
	r.Handler("OPTIONS", path, handler)
	r.Handler("TRACE", path, handler)
}


// Run Rest MicroService.
/**
* params 为可变参数
* 第一个参数为ip host
* 第二个参数为ip port
* 第三个参数为ServerTimeOut
* 第四个参数为协议是否为Tcp4 or tcp6，bool值：true or false
 */
func (ms *MicroService) Serve(params ...string) {

	if len(params) < 2 {
		fmt.Printf("err: no host port parameters set.\n")
		return
	}

	addr := params[0] + ":" + params[1]


	if len(params) > 2 {
		ServerTimeOut, _ := strconv.ParseInt(params[2], 10, 64)
		ms.Server.ReadTimeout = time.Duration(ServerTimeOut) * time.Second
		ms.Server.WriteTimeout = time.Duration(ServerTimeOut) * time.Second
	}

	var isListenTCP4 bool = false

	if len(params) > 3 {
		isListenTCP4, _ = strconv.ParseBool(params[3])
	}

	// run normal mode

	ms.Server.Handler = ms.Router
	ms.Server.Addr = addr
	fmt.Printf("http server Running on : %s\n", ms.Server.Addr)
	if isListenTCP4 {
		ln, err := net.Listen("tcp4", ms.Server.Addr)
		if err != nil {
			fmt.Printf("ListenAndServe: %v\n", err)
			time.Sleep(100 * time.Microsecond)
			return
		}
		if err = ms.Server.Serve(ln); err != nil {
			fmt.Printf("ListenAndServe: %v\n", err)
			time.Sleep(100 * time.Microsecond)
			return
		}
	} else {
		if err := ms.Server.ListenAndServe(); err != nil {
			fmt.Printf("ListenAndServe: %v\n", err)
			time.Sleep(100 * time.Microsecond)
		}
	}

}

func (ms *MicroService) NewRestEndpoint(svc RestService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		if request == nil {
			return nil, errors.New("no request avaliable.")
		}

		req := request.(Request)

		var ret interface{}
		var err error
		switch req.Method {
		case "GET":
			ret, err = svc.Get(&req)
		case "POST":
			ret, err = svc.Post(&req)
		case "PUT":
			ret, err = svc.Put(&req)
		case "DELETE":
			ret, err = svc.Delete(&req)
		case "HEAD":
			ret, err = svc.Head(&req)
		case "PATCH":
			ret, err = svc.Patch(&req)
		case "OPTIONS":
			ret, err = svc.Options(&req)
		case "TRACE":
			ret, err = svc.Trace(&req)
		case "CONNECT":
		}

		if err != nil {
			return svc.GetErrorResponse(), nil
		}
		return ret, nil
	}
}


func (ms *MicroService) SetLogger(logger log.Logger) {
	ms.logger = logger
}

func (ms *MicroService) GetLogger() log.Logger {
	return ms.logger
}

func (ms *MicroService) SetTracer(tracer stdopentracing.Tracer) {
	ms.tracer = tracer
}

func (ms *MicroService) GetTracer() stdopentracing.Tracer {
	return ms.tracer
}

func (ms *MicroService) SetZipkinTracer(zipkinTracer *stdzipkin.Tracer) {
	ms.zipkinTracer = zipkinTracer
}

func (ms *MicroService) GetZipkinTracer() *stdzipkin.Tracer {
	return ms.zipkinTracer
}

func (ms *MicroService) NewHttpHandler(path string,rest RestService,middlewares ...RestMiddleware) *mshttp.Server {

	svc := ms.NewRestEndpoint(rest)

	for i := 0; i < len(middlewares); i++ {
		svc = middlewares[i].GetMiddleware()(middlewares[i].Object)(svc)
	}

	var zipkinServer mshttp.ServerOption
	var options []mshttp.ServerOption

	if ms.zipkinTracer != nil {
		zipkinServer = zipkin.HTTPServerTrace(ms.zipkinTracer)
		options = []mshttp.ServerOption{
			mshttp.ServerErrorEncoder(errorEncoder),
			mshttp.ServerErrorLogger(ms.logger),
			zipkinServer,
		}
	}else{
		options = []mshttp.ServerOption{
			mshttp.ServerErrorEncoder(errorEncoder),
			mshttp.ServerErrorLogger(ms.logger),
		}
	}

	if ms.tracer != nil {
		svc = opentracing.TraceServer(ms.tracer, path)(svc)
		options = append(options, mshttp.ServerBefore(opentracing.HTTPToContext(ms.tracer, path, ms.logger)))
	}

	handler := mshttp.NewServer(
		svc,
		rest.DecodeRequest,
		rest.EncodeResponse,
		options...,
	)

	return handler
}

func (ms *MicroService) RegisterServiceWithTracer(path string, rest RestService, tracer stdopentracing.Tracer, logger log.Logger, middlewares ...RestMiddleware) {

	ms.SetLogger(logger)
	ms.SetTracer(tracer)

	handler := ms.NewHttpHandler(path,rest,middlewares...)
	regRoute(ms.Router,path,handler)
}


func (ms *MicroService) RegisterRestService(path string, rest RestService, middlewares ...RestMiddleware) {

	handler := ms.NewHttpHandler(path,rest,middlewares...)
	regRoute(ms.Router,path,handler)
}

func (ms *MicroService) Handler(method, path string, handler http.Handler) {
	ms.Router.Handler(method, path, handler)
}
func (ms *MicroService) HandlerFunc(method, path string, handler http.Handler) {
	ms.Router.Handler(method, path, handler)
}


/**
	包方法:
**/
func RegisterRestService(path string, rest RestService,middlewares ...RestMiddleware) {
	MsRest.RegisterRestService(path,rest,middlewares...)
}

func RegisterServiceWithTracer(path string, rest RestService, tracer stdopentracing.Tracer, logger log.Logger,middlewares ...RestMiddleware) {
	MsRest.RegisterServiceWithTracer(path,rest,tracer,logger,middlewares...)
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
	}else{
		fmt.Printf("no rest service avaliable.\n")
	}
}

func ServeFiles(path string,root http.FileSystem) {
	if MsRest != nil {
		MsRest.Router.ServeFiles(path,root)
	}else{
		fmt.Printf("no rest service avaliable.\n")
	}
}