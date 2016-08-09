package mskit

import(
	mshttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	"github.com/go-kit/kit/endpoint"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"errors"
	"strings"
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
	resp.Data["ret"] = 1
	resp.Data["error"] =errors.New("Not allowed.") 
	return resp
}


// DecodeRequest adds a restservice used for endpoint.
/*
需要在nginx上配置
proxy_set_header Remote_addr $remote_addr;
*/
func (c *RestApi)DecodeRequest(r *http.Request) (request interface{}, err error){
	
	req := Request{Queries : make(map[string]interface{}), }
	
	req.Method =  r.Method
	
	_,req.Params,_ = MsRest.Router.Lookup(r.Method,r.URL.EscapedPath()) 
		
	values := r.URL.Query()
	
	accept := r.Header.Get("Accept")
	ss := strings.Split(accept,";")
	
	for _,s := range ss{
		sv := strings.Split(s,"=")
		
		if len(sv)>1 && strings.TrimSpace(sv[0])== "version" {
			req.Version = sv[1]
		}	
		
	} 
	
	for k,v := range values {
		req.Queries[k] = v
	}
	
	req.Body,err = ioutil.ReadAll(r.Body)
	
	ip := r.Header.Get("X-Real-IP")
	
    if (ip=="") {
       req.RemoteAddr = r.RemoteAddr
    }else{
    	req.RemoteAddr = ip
    }
    
	
	
	return req,nil
}

func (c *RestApi)Prepare(r *Request)(interface{},error){
	return nil,nil
} 

/*
*该方法是在response返回之前调用，用于增加一下个性化的头信息
*/
func (c *RestApi)Finish(w http.ResponseWriter)(error){
	
	if w == nil {
		return errors.New("writer is nil ")
	}
	
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type,Origin,Accept,Content-Range,Content-Description,Content-Disposition")
	w.Header().Add("Access-Control-Allow-Methods", "PUT,GET,POST,DELETE,OPTIONS")
	
	return nil
} 

// EncodeResponse adds a restservice used for endpoint.
func (c *RestApi)EncodeResponse(w http.ResponseWriter,response interface{}) error{
	
	if response == nil {
		response = ""
	}
	
	w.Header().Set("Allow", "HEAD,GET,PUT,DELETE,OPTIONS,POST");
	
	c.Finish(w)
	
	err := json.NewEncoder(w).Encode(response)

	return err
}

