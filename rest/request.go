package rest

import (
	"github.com/libra9z/httprouter"
	"github.com/libra9z/mskit/trace"
	"net/http"
	"strconv"
)

type Request struct {
	IsAuthorized  bool
	LicExpired    bool
	Version       string
	Params        httprouter.Params
	Queries       map[string]interface{}
	Body          []byte
	Method        string
	RemoteAddr    string
	OriginRequest *http.Request
	ContentType   int
	Userid		  string  //admin user prefix with 'a' ,user table prefix with 'u'
	Tracer        trace.Tracer
	AuthedOrgids  []int64
}

const (
	maxParam               = 50
	CONTENT_TYPE_FORM      = 1
	CONTENT_TYPE_XML       = 2
	CONTENT_TYPE_JSON      = 3
	CONTENT_TYPE_MULTIFORM = 4
	CONTENT_TYPE_SENML 		= 5
)

func NewRequest() *Request {
	return &Request{
		Queries: make(map[string]interface{}),
	}
}

func (r *Request) GetString(key string) []string {
	var ret []string
	for k, v := range r.Queries {
		if k == key {
			ret = v.([]string)
			break
		}
	}

	return ret
}
func (r *Request) GetInt(key string) []int {
	var ret []int

	for k, v := range r.Queries {
		if k == key {
			s := v.([]string)

			for _, si := range s {
				iv, _ := strconv.ParseInt(si, 10, 64)
				ret = append(ret, int(iv))
			}

			break
		}
	}
	return ret
}

func (r *Request) GetInt64(key string) []int64 {
	var ret []int64

	for k, v := range r.Queries {
		if k == key {
			s := v.([]string)

			for _, si := range s {
				iv, _ := strconv.ParseInt(si, 10, 64)
				ret = append(ret, iv)
			}

			break
		}
	}
	return ret
}

func (r *Request) SetAuthorized(auth bool) {
	r.IsAuthorized = auth
}

func (r *Request) SetUserid(uid string) {
	r.Userid = uid
}

func (r *Request) GetUserid() string {
	return r.Userid
}

func (r *Request) GetContentType() string {
	var ct string
	switch r.ContentType {
	case CONTENT_TYPE_FORM:
		ct = "application/x-www-form-urlencoded"
	case CONTENT_TYPE_XML:
		ct = "application/xml"
	case CONTENT_TYPE_JSON:
		ct = "application/json"
	case CONTENT_TYPE_SENML:
		ct = "application/senml+json"
	case CONTENT_TYPE_MULTIFORM:
		ct = "multipart/form-data"
	}
	return ct
}
