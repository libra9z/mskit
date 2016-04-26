package mskit

import(
	mshttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	"github.com/go-kit/kit/endpoint"
	"net/http"
	"errors"
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
		
		if request == nil {
			return nil,errors.New("no request avaliable.")
		}
		
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


type RestApi struct{

}

// Get adds a request function to handle GET request.
func (c *RestApi) Get(r *Request)(interface{},error) {
	return nil,nil
}

// Post adds a request function to handle POST request.
func (c *RestApi) Post(r *Request)(interface{},error) {
	return nil,nil
}

// Delete adds a request function to handle DELETE request.
func (c *RestApi) Delete(r *Request)(interface{},error) {
	return nil,nil
}

// Put adds a request function to handle PUT request.
func (c *RestApi) Put(r *Request)(interface{},error) {
	return nil,nil
}

// Head adds a request function to handle HEAD request.
func (c *RestApi) Head(r *Request)(interface{},error){
	return nil,nil
}

// Patch adds a request function to handle PATCH request.
func (c *RestApi) Patch(r *Request)(interface{},error){
	return nil,nil
}

// Options adds a request function to handle OPTIONS request.
func (c *RestApi) Options(r *Request)(interface{},error){
	return nil,nil
}

// Options adds a request function to handle OPTIONS request.
func (c *RestApi) Trace(r *Request)(interface{},error){
	return nil,nil
}

// GetErrorResponse adds a restservice used for endpoint.
func (c *RestApi)GetErrorResponse() interface{} {
	resp := NewResponse()
	resp.Data["ret"] = "1"
	resp.Data["error"] =errors.New("Not allowed.") 
	return resp
}
// DecodeRequest adds a restservice used for endpoint.
func (c *RestApi)DecodeRequest(*http.Request) (request interface{}, err error){
	return nil,nil
}
// EncodeResponse adds a restservice used for endpoint.
func (c *RestApi)EncodeResponse(http.ResponseWriter, interface{}) error{
	return nil
}

