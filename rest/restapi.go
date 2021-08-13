package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-kit/kit/metrics"
	"github.com/libra9z/httprouter"
)

var _ RestService = (*RestApi)(nil)

const DefaultContextKey = "MskitABContext"
// HandlerFunc defines the handler used by gin middleware as return value.
type BeforeFunc func(*Mcontext,http.ResponseWriter)

// HandlersChain defines a HandlerFunc array.
type BeforesChain []BeforeFunc

// Last returns the last handler in the chain. ie. the last handler is the main one.
func (c BeforesChain) Last() BeforeFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}

// HandlerFunc defines the handler used by gin middleware as return value.
type AfterFunc func(*Mcontext,http.ResponseWriter)

// HandlersChain defines a HandlerFunc array.
type AftersChain []AfterFunc

// Last returns the last handler in the chain. ie. the last handler is the main one.
func (c AftersChain) Last() AfterFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}
type RestApi struct {
	Request   *http.Request
	Router    *httprouter.Router
	Counter   metrics.Counter
	Gauge     metrics.Gauge
	Histogram metrics.Histogram
	after 		AftersChain
	before 		BeforesChain
	mc 			*Mcontext
}

func (c *RestApi) After() AftersChain {
	return c.after
}

func (c *RestApi) Before() BeforesChain {
	return c.before
}

func (c *RestApi) AfterUse( handlerFunc ...AfterFunc) {
	c.after = append(c.after,handlerFunc...)
}

func (c *RestApi) BeforeUse(handlerFunc ...BeforeFunc) {
	c.before = append(c.before,handlerFunc...)
}

func (c *RestApi) Mcontext()  *Mcontext{
	return c.mc
}

func (c *RestApi) SetMcontext(mc *Mcontext)  {
	c.mc = mc
}

// Get adds a request function to handle GET request.
func (c *RestApi) SetRouter(r *httprouter.Router) {
	c.Router = r
}

// Get adds a request function to handle GET request.
func (c *RestApi) Get(ctx context.Context, r *Mcontext) (interface{}, error) {
	return nil, nil
}

// Post adds a request function to handle POST request.
func (c *RestApi) Post(ctx context.Context, r *Mcontext) (interface{}, error) {
	return nil, nil
}

// Delete adds a request function to handle DELETE request.
func (c *RestApi) Delete(ctx context.Context, r *Mcontext) (interface{}, error) {
	return nil, nil
}

// Put adds a request function to handle PUT request.
func (c *RestApi) Put(ctx context.Context, r *Mcontext) (interface{}, error) {
	return nil, nil
}

// Head adds a request function to handle HEAD request.
func (c *RestApi) Head(ctx context.Context, r *Mcontext) (interface{}, error) {
	return nil, nil
}

// Patch adds a request function to handle PATCH request.
func (c *RestApi) Patch(ctx context.Context, r *Mcontext) (interface{}, error) {
	return nil, nil
}

// Options adds a request function to handle OPTIONS request.
func (c *RestApi) Options(ctx context.Context, r *Mcontext) (interface{}, error) {
	return nil, nil
}

// Options adds a request function to handle OPTIONS request.
func (c *RestApi) Trace(ctx context.Context, r *Mcontext) (interface{}, error) {
	return nil, nil
}

// GetErrorResponse adds a restservice used for endpoint.
func (c *RestApi) GetErrorResponse() interface{} {
	return nil
}

// DecodeRequest adds a restservice used for endpoint.
/*
需要在nginx上配置
proxy_set_header Remote_addr $remote_addr;
*/
func (c *RestApi) DecodeRequest(ctx context.Context, r *http.Request,w http.ResponseWriter) (request interface{}, err error) {

	c.Request = r

	req := Mcontext{}

	req.reset()
	req.Method = r.Method
	//req.writermem.reset(w)
	req.Queries = make(map[string]interface{})

	if c.Router == nil {
		fmt.Printf("no router set.\n")
		return nil, errors.New("no router set.")
	}

	_, req.Params, _ = c.Router.Lookup(r.Method, r.URL.EscapedPath())

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

	req.Request = r

	if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		req.Body, err = ioutil.ReadAll(r.Body)

		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			req.ContentType = CONTENT_TYPE_JSON
		} else if strings.Contains(r.Header.Get("Content-Type"), "application/xml") ||
			strings.Contains(r.Header.Get("Content-Type"), "text/xml") {
			req.ContentType = CONTENT_TYPE_XML
		} else if strings.Contains(r.Header.Get("Content-Type"), "x-www-form-urlencoded") {
			req.ContentType = CONTENT_TYPE_FORM
		}
	} else {
		req.ContentType = CONTENT_TYPE_MULTIFORM
	}

	c.mc = &req
	c.mc.writermem.reset(w)
	c.Cors(w)

	return c.mc,err

}

func (c *RestApi) Prepare(r *Mcontext) (*Mcontext, error) {
	return r, nil
}

/*
*该方法是在response返回之前调用，用于增加一下个性化的头信息
 */
 func (c *RestApi) Cors(w http.ResponseWriter) error {

	if w == nil {
		return errors.New("writer is nil ")
	}
	if !c.mc.EnableCors {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type,Origin,Accept,Content-Range,Content-Description,Content-Disposition")
		w.Header().Add("Access-Control-Allow-Methods", "PUT,GET,POST,DELETE,OPTIONS")

		c.mc.Header("Access-Control-Allow-Origin", "*")
		c.mc.Header("Access-Control-Allow-Headers", "Content-Type,Origin,Accept,Content-Range,Content-Description,Content-Disposition")
		c.mc.Header("Access-Control-Allow-Methods", "PUT,GET,POST,DELETE,OPTIONS")
	}

	return nil
}

func (c *RestApi) Finish(w http.ResponseWriter, response interface{}) error {

	if w == nil {
		return errors.New("writer is nil ")
	}

	w.Header().Set("Content-Type",MIMEJSON)

	err := json.NewEncoder(w).Encode(response)
	return err
}

// EncodeResponse adds a restservice used for endpoint.
func (c *RestApi) EncodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {

	var err error

	if response == nil {
		response = ""
	}
	w.Header().Set("Allow", "HEAD,GET,PUT,DELETE,OPTIONS,POST")

	//
	//for _,f := range c.After() {
	//	f(c.mc,w)
	//}

	if !c.mc.useContextWriter {
		err = c.Finish(w, response)
	}

	return err
}

type errorWrapper struct {
	Error string `json:"error"`
}

func (c *RestApi)ErrorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	code := http.StatusInternalServerError
	msg := err.Error()

	switch err {
	case ErrTwoZeroes, ErrMaxSizeExceeded, ErrIntOverflow:
		code = http.StatusBadRequest
	}

	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errorWrapper{Error: msg})
}
