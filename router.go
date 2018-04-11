package mskit

import (
	"encoding/json"
	"errors"
	"github.com/go-kit/kit/endpoint"
	mshttp "github.com/go-kit/kit/transport/http"
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"
)

type Middleware func(endpoint.Endpoint) endpoint.Endpoint

type RestMiddleware struct {
	Middle Middleware
	Object interface{}
}

var (
	// ErrTwoZeroes is an arbitrary business rule for the Add method.
	ErrTwoZeroes = errors.New("can't sum two zeroes")

	// ErrIntOverflow protects the Add method. We've decided that this error
	// indicates a misbehaving service and should count against e.g. circuit
	// breakers. So, we return it directly in endpoints, to illustrate the
	// difference. In a real service, this probably wouldn't be the case.
	ErrIntOverflow = errors.New("integer overflow")

	// ErrMaxSizeExceeded protects the Concat method.
	ErrMaxSizeExceeded = errors.New("result exceeds maximum size")
)

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	code := http.StatusInternalServerError
	msg := err.Error()


	switch err {
	case ErrTwoZeroes, ErrMaxSizeExceeded, ErrIntOverflow:
		code = http.StatusBadRequest
	}


	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errorWrapper{Error: msg})
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

type errorWrapper struct {
	Error string `json:"error"`
}

func (rm *RestMiddleware) GetMiddleware() func(interface{}) Middleware {
	return func(inter interface{}) Middleware {
		return rm.Middle
	}
}

type RestService interface {
	Get(*Request) (interface{}, error)
	Post(*Request) (interface{}, error)
	Delete(*Request) (interface{}, error)
	Put(*Request) (interface{}, error)
	Head(*Request) (interface{}, error)
	Patch(*Request) (interface{}, error)
	Options(*Request) (interface{}, error)
	Trace(*Request) (interface{}, error)

	//response relate interface
	GetErrorResponse() interface{}
	DecodeRequest(context.Context, *http.Request) (request interface{}, err error)
	EncodeResponse(context.Context, http.ResponseWriter, interface{}) error
}

func MakeRestEndpoint(svc RestService) endpoint.Endpoint {
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

func RegisterRestService(path string, rest RestService,middlewares ...RestMiddleware) {
	//ctx := context.Background()

	svc := MakeRestEndpoint(rest)

	for i := 0; i < len(middlewares); i++ {
		svc = middlewares[i].GetMiddleware()(middlewares[i].Object)(svc)
	}

	options := []mshttp.ServerOption{
		mshttp.ServerErrorEncoder(errorEncoder),
	}


	handler := mshttp.NewServer(
		//ctx,
		svc,
		rest.DecodeRequest,
		rest.EncodeResponse,
		options...,
	)

	MsRest.Router.Handler("GET", path, handler)
	MsRest.Router.Handler("POST", path, handler)
	MsRest.Router.Handler("PUT", path, handler)
	MsRest.Router.Handler("PATCH", path, handler)
	MsRest.Router.Handler("DELETE", path, handler)
	MsRest.Router.Handler("HEAD", path, handler)
	MsRest.Router.Handler("OPTIONS", path, handler)
	MsRest.Router.Handler("TRACE", path, handler)
}

func RegisterServiceWithTracer(path string, rest RestService, tracer stdopentracing.Tracer, logger log.Logger,middlewares ...RestMiddleware) {
	//ctx := context.Background()

	svc := MakeRestEndpoint(rest)

	for i := 0; i < len(middlewares); i++ {
		svc = middlewares[i].GetMiddleware()(middlewares[i].Object)(svc)
	}

	options := []mshttp.ServerOption{
		mshttp.ServerErrorEncoder(errorEncoder),
		mshttp.ServerErrorLogger(logger),
	}
	if tracer != nil {
		svc = opentracing.TraceServer(tracer, path)(svc)
		options = append(options, mshttp.ServerBefore(opentracing.HTTPToContext(tracer, path, logger)))
	}


	handler := mshttp.NewServer(
		//ctx,
		svc,
		rest.DecodeRequest,
		rest.EncodeResponse,
		options...,
	)

	MsRest.Router.Handler("GET", path, handler)
	MsRest.Router.Handler("POST", path, handler)
	MsRest.Router.Handler("PUT", path, handler)
	MsRest.Router.Handler("PATCH", path, handler)
	MsRest.Router.Handler("DELETE", path, handler)
	MsRest.Router.Handler("HEAD", path, handler)
	MsRest.Router.Handler("OPTIONS", path, handler)
	MsRest.Router.Handler("TRACE", path, handler)
}

func Handler(method, path string, handler http.Handler) {
	MsRest.Router.Handler(method, path, handler)
}
func HandlerFunc(method, path string, handler http.Handler) {
	MsRest.Router.Handler(method, path, handler)
}



func (ms *MicroService) RegisterRestService(path string, rest RestService, middlewares ...RestMiddleware) {
	//ctx := context.Background()

	svc := MakeRestEndpoint(rest)

	for i := 0; i < len(middlewares); i++ {
		svc = middlewares[i].GetMiddleware()(middlewares[i].Object)(svc)
	}

	options := []mshttp.ServerOption{
		mshttp.ServerErrorEncoder(errorEncoder),
	}

	handler := mshttp.NewServer(
		//ctx,
		svc,
		rest.DecodeRequest,
		rest.EncodeResponse,
		options...,
	)

	ms.Router.Handler("GET", path, handler)
	ms.Router.Handler("POST", path, handler)
	ms.Router.Handler("PUT", path, handler)
	ms.Router.Handler("PATCH", path, handler)
	ms.Router.Handler("DELETE", path, handler)
	ms.Router.Handler("HEAD", path, handler)
	ms.Router.Handler("OPTIONS", path, handler)
	ms.Router.Handler("TRACE", path, handler)
}


func (ms *MicroService) RegisterServiceWithTracer(path string, rest RestService, tracer stdopentracing.Tracer, logger log.Logger, middlewares ...RestMiddleware) {
	//ctx := context.Background()

	svc := MakeRestEndpoint(rest)

	for i := 0; i < len(middlewares); i++ {
		svc = middlewares[i].GetMiddleware()(middlewares[i].Object)(svc)
	}

	options := []mshttp.ServerOption{
		mshttp.ServerErrorEncoder(errorEncoder),
		mshttp.ServerErrorLogger(logger),
	}
	if tracer != nil {
		svc = opentracing.TraceServer(tracer, path)(svc)
		options = append(options, mshttp.ServerBefore(opentracing.HTTPToContext(tracer, path, logger)))
	}

	handler := mshttp.NewServer(
		//ctx,
		svc,
		rest.DecodeRequest,
		rest.EncodeResponse,
		options...,
	)

	ms.Router.Handler("GET", path, handler)
	ms.Router.Handler("POST", path, handler)
	ms.Router.Handler("PUT", path, handler)
	ms.Router.Handler("PATCH", path, handler)
	ms.Router.Handler("DELETE", path, handler)
	ms.Router.Handler("HEAD", path, handler)
	ms.Router.Handler("OPTIONS", path, handler)
	ms.Router.Handler("TRACE", path, handler)
}


func (ms *MicroService) Handler(method, path string, handler http.Handler) {
	ms.Router.Handler(method, path, handler)
}
func (ms *MicroService) HandlerFunc(method, path string, handler http.Handler) {
	ms.Router.Handler(method, path, handler)
}



type RestApi struct {
	Request 		*http.Request
}

// Get adds a request function to handle GET request.
func (c *RestApi) Get(r *Request) (interface{}, error) {
	return nil, nil
}

// Post adds a request function to handle POST request.
func (c *RestApi) Post(r *Request) (interface{}, error) {
	return nil, nil
}

// Delete adds a request function to handle DELETE request.
func (c *RestApi) Delete(r *Request) (interface{}, error) {
	return nil, nil
}

// Put adds a request function to handle PUT request.
func (c *RestApi) Put(r *Request) (interface{}, error) {
	return nil, nil
}

// Head adds a request function to handle HEAD request.
func (c *RestApi) Head(r *Request) (interface{}, error) {
	return nil, nil
}

// Patch adds a request function to handle PATCH request.
func (c *RestApi) Patch(r *Request) (interface{}, error) {
	return nil, nil
}

// Options adds a request function to handle OPTIONS request.
func (c *RestApi) Options(r *Request) (interface{}, error) {
	return nil, nil
}

// Options adds a request function to handle OPTIONS request.
func (c *RestApi) Trace(r *Request) (interface{}, error) {
	return nil, nil
}

// GetErrorResponse adds a restservice used for endpoint.
func (c *RestApi) GetErrorResponse() interface{} {
	resp := NewResponse()
	resp.Data["ret"] = 1
	resp.Data["error"] = errors.New("Not allowed.")
	return resp
}

// DecodeRequest adds a restservice used for endpoint.
/*
需要在nginx上配置
proxy_set_header Remote_addr $remote_addr;
*/
func (c *RestApi) DecodeRequest(_ context.Context, r *http.Request) (request interface{}, err error) {

	c.Request = r

	req := Request{Queries: make(map[string]interface{})}

	req.Method = r.Method

	_, req.Params, _ = MsRest.Router.Lookup(r.Method, r.URL.EscapedPath())

	values := r.URL.Query()

	accept := r.Header.Get("Accept")
	ss := strings.Split(accept, ";")

	for _, s := range ss {
		sv := strings.Split(s, "=")

		if len(sv) > 1 && strings.TrimSpace(sv[0]) == "version" {
			req.Version = sv[1]
		}

	}

	for k, v := range values {
		req.Queries[k] = v
	}

	ip := r.Header.Get("X-Real-IP")

	if ip == "" {
		req.RemoteAddr = r.RemoteAddr
	} else {
		req.RemoteAddr = ip
	}

	req.OriginRequest = r

	if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		req.Body, err = ioutil.ReadAll(r.Body)

		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			req.ContentType = CONTENT_TYPE_JSON
		}else if strings.Contains(r.Header.Get("Content-Type"), "application/xml") ||
			strings.Contains(r.Header.Get("Content-Type"), "text/xml") {
			req.ContentType = CONTENT_TYPE_XML
		}else if strings.Contains(r.Header.Get("Content-Type"), "x-www-form-urlencoded") {
			req.ContentType = CONTENT_TYPE_FORM
		}
	}else {
		req.ContentType = CONTENT_TYPE_MULTIFORM
	}


	return req, nil
}

func (c *RestApi) Prepare(r *Request) (interface{}, error) {
	return nil, nil
}

/*
*该方法是在response返回之前调用，用于增加一下个性化的头信息
 */
func (c *RestApi) Finish(w http.ResponseWriter) error {

	if w == nil {
		return errors.New("writer is nil ")
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type,Origin,Accept,Content-Range,Content-Description,Content-Disposition")
	w.Header().Add("Access-Control-Allow-Methods", "PUT,GET,POST,DELETE,OPTIONS")

	return nil
}

// EncodeResponse adds a restservice used for endpoint.
func (c *RestApi) EncodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {

	if response == nil {
		response = ""
	}

	w.Header().Set("Allow", "HEAD,GET,PUT,DELETE,OPTIONS,POST")

	c.Finish(w)

	err := json.NewEncoder(w).Encode(response)

	return err
}


