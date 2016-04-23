package mskit

import(
	mshttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	"github.com/go-kit/kit/endpoint"
	"net/http"
	
)


type Middleware func(endpoint.Endpoint) endpoint.Endpoint

type RestMiddleware struct {
	Middle	Middleware
	Object	interface{}
}

func(rm *RestMiddleware)GetMiddleware() (func (interface{}) Middleware) {
	return func(inter interface{}) Middleware{
		return rm.Middle
	}
}


type RestService interface {
	Get(*Request)(interface{},error)
	Post(*Request)(interface{},error)
	Delete(*Request)(interface{},error)
	Put(*Request)(interface{},error)
	Head(*Request)(interface{},error)
	Patch(*Request)(interface{},error)
	Options(*Request)(interface{},error)
	Trace(*Request)(interface{},error)
	
	//response relate interface
	GetErrorResponse() interface{}
	DecodeRequest(*http.Request) (request interface{}, err error)
	EncodeResponse(http.ResponseWriter, interface{}) error
}


func MakeRestEndpoint(svc RestService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(Request)
		
		var ret interface{}
		var err error
		switch req.Method {
		case "GET":
			ret,err = svc.Get(&req)
		case "POST":
			ret,err = svc.Post(&req)
		case "PUT":
			ret,err = svc.Put(&req)
		case "DELETE":
			ret,err = svc.Delete(&req)
		case "HEAD":
			ret,err = svc.Head(&req)
		case "PATCH":
			ret,err = svc.Patch(&req)
		case "OPTIONS":
			ret,err = svc.Options(&req)
		case "TRACE":
			ret,err = svc.Trace(&req)
		case "CONNECT":
		}
		
		if err != nil {
			return svc.GetErrorResponse(), nil
		}
		return ret, nil
	}
}

func RegisterRestService(path string,rest RestService, middlewares... RestMiddleware ) {
	ctx := context.Background()
	
	svc := MakeRestEndpoint(rest)
	
	
	for i:=0;i<len(middlewares);i++ {
		svc = middlewares[i].GetMiddleware()(middlewares[i].Object)(svc)
	}
	
	handler := mshttp.NewServer(
		ctx,
		svc,
		rest.DecodeRequest,
		rest.EncodeResponse,
	)
	
	MsRest.Router.Handler("GET",path,handler)
	MsRest.Router.Handler("POST",path,handler)
	MsRest.Router.Handler("PUT",path,handler)
	MsRest.Router.Handler("PATCH",path,handler)
	MsRest.Router.Handler("DELETE",path,handler)
	MsRest.Router.Handler("HEAD",path,handler)
	MsRest.Router.Handler("OPTIONS",path,handler)
	MsRest.Router.Handler("TRACE",path,handler)
}


func (ms *MicroService)RegisterRestService(path string,rest RestService, middlewares... RestMiddleware ) {
	ctx := context.Background()
	
	svc := MakeRestEndpoint(rest)
	
	
	for i:=0;i<len(middlewares);i++ {
		svc = middlewares[i].GetMiddleware()(middlewares[i].Object)(svc)
	}
	
	handler := mshttp.NewServer(
		ctx,
		svc,
		rest.DecodeRequest,
		rest.EncodeResponse,
	)
	
	ms.Router.Handler("GET",path,handler)
	ms.Router.Handler("POST",path,handler)
	ms.Router.Handler("PUT",path,handler)
	ms.Router.Handler("PATCH",path,handler)
	ms.Router.Handler("DELETE",path,handler)
	ms.Router.Handler("HEAD",path,handler)
	ms.Router.Handler("OPTIONS",path,handler)
	ms.Router.Handler("TRACE",path,handler)
}

