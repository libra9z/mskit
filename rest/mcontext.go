package rest

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/libra9z/httprouter"
	"github.com/libra9z/mskit/binding"
	"github.com/libra9z/mskit/trace"
)

// Content-Type MIME of the most common data formats.
const (
	MIMEJSON              = binding.MIMEJSON
	MIMEHTML              = binding.MIMEHTML
	MIMEXML               = binding.MIMEXML
	MIMEXML2              = binding.MIMEXML2
	MIMEPlain             = binding.MIMEPlain
	MIMEPOSTForm          = binding.MIMEPOSTForm
	MIMEMultipartPOSTForm = binding.MIMEMultipartPOSTForm
	MIMEYAML              = binding.MIMEYAML
)

type Mcontext struct {
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
	Userid        string //admin user prefix with 'a' ,user table prefix with 'u'
	Tracer        trace.Tracer
	AuthedOrgids  []int64

	// This mutex protect Keys map
	mu sync.RWMutex

	// queryCache use url.ParseQuery cached the param query result from c.Request.URL.Query()
	queryCache url.Values

	// formCache use url.ParseQuery cached PostForm contains the parsed form data from POST, PATCH,
	// or PUT body parameters.
	formCache url.Values

	// SameSite allows a server to define a cookie attribute making it impossible for
	// the browser to send this cookie along with cross-site requests.
	sameSite http.SameSite
}

type Request Mcontext

const (
	maxParam               = 50
	CONTENT_TYPE_FORM      = 1
	CONTENT_TYPE_XML       = 2
	CONTENT_TYPE_JSON      = 3
	CONTENT_TYPE_MULTIFORM = 4
	CONTENT_TYPE_SENML     = 5
)

func NewContext() *Mcontext {
	return &Mcontext{
		Queries: make(map[string]interface{}),
	}
}

// func (r *Request) GetString(key string) []string {
// 	var ret []string
// 	for k, v := range r.Queries {
// 		if k == key {
// 			ret = v.([]string)
// 			break
// 		}
// 	}

// 	return ret
// }
// func (r *Request) GetInt(key string) []int {
// 	var ret []int

// 	for k, v := range r.Queries {
// 		if k == key {
// 			s := v.([]string)

// 			for _, si := range s {
// 				iv, _ := strconv.ParseInt(si, 10, 64)
// 				ret = append(ret, int(iv))
// 			}

// 			break
// 		}
// 	}
// 	return ret
// }

// func (r *Request) GetInt64(key string) []int64 {
// 	var ret []int64

// 	for k, v := range r.Queries {
// 		if k == key {
// 			s := v.([]string)

// 			for _, si := range s {
// 				iv, _ := strconv.ParseInt(si, 10, 64)
// 				ret = append(ret, iv)
// 			}

// 			break
// 		}
// 	}
// 	return ret
// }

func (r *Mcontext) SetAuthorized(auth bool) {
	r.IsAuthorized = auth
}

func (r *Mcontext) SetUserid(uid string) {
	r.Userid = uid
}

func (r *Mcontext) GetUserid() string {
	return r.Userid
}

func (r *Mcontext) GetContentType() string {
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

/************************************/
/******** METADATA MANAGEMENT********/
/************************************/

// Set is used to store a new key/value pair exclusively for this Context.
// It also lazy initializes  c.Keys if it was not used previously.
func (c *Mcontext) Set(key string, value interface{}) {
	c.mu.Lock()
	if c.Queries == nil {
		c.Queries = make(map[string]interface{})
	}

	c.Queries[key] = value
	c.mu.Unlock()
}

// Get returns the value for the given key, ie: (value, true).
// If the value does not exists it returns (nil, false)
func (c *Mcontext) Get(key string) (value interface{}, exists bool) {
	c.mu.RLock()
	value, exists = c.Queries[key]
	c.mu.RUnlock()
	return
}

// MustGet returns the value for the given key if it exists, otherwise it panics.
func (c *Mcontext) MustGet(key string) interface{} {
	if value, exists := c.Get(key); exists {
		return value
	}
	panic("Key \"" + key + "\" does not exist")
}

// GetString returns the value associated with the key as a string.
func (c *Mcontext) GetString(key string) (s string) {
	if val, ok := c.Get(key); ok && val != nil {
		s, _ = val.(string)
	}
	return
}

// GetBool returns the value associated with the key as a boolean.
func (c *Mcontext) GetBool(key string) (b bool) {
	if val, ok := c.Get(key); ok && val != nil {
		b, _ = val.(bool)
	}
	return
}

// GetInt returns the value associated with the key as an integer.
func (c *Mcontext) GetInt(key string) (i int) {
	if val, ok := c.Get(key); ok && val != nil {
		i, _ = val.(int)
	}
	return
}

// GetInt64 returns the value associated with the key as an integer.
func (c *Mcontext) GetInt64(key string) (i64 int64) {
	if val, ok := c.Get(key); ok && val != nil {
		i64, _ = val.(int64)
	}
	return
}

// GetFloat64 returns the value associated with the key as a float64.
func (c *Mcontext) GetFloat64(key string) (f64 float64) {
	if val, ok := c.Get(key); ok && val != nil {
		f64, _ = val.(float64)
	}
	return
}

// GetTime returns the value associated with the key as time.
func (c *Mcontext) GetTime(key string) (t time.Time) {
	if val, ok := c.Get(key); ok && val != nil {
		t, _ = val.(time.Time)
	}
	return
}

// GetDuration returns the value associated with the key as a duration.
func (c *Mcontext) GetDuration(key string) (d time.Duration) {
	if val, ok := c.Get(key); ok && val != nil {
		d, _ = val.(time.Duration)
	}
	return
}

// GetInt returns the value associated with the key as an integer.
func (c *Mcontext) GetIntSlice(key string) (i []int) {
	if val, ok := c.Get(key); ok && val != nil {
		i, _ = val.([]int)
	}
	return
}

// GetInt64 returns the value associated with the key as an integer.
func (c *Mcontext) GetInt64Slice(key string) (i64 []int64) {
	if val, ok := c.Get(key); ok && val != nil {
		i64, _ = val.([]int64)
	}
	return
}

// GetStringSlice returns the value associated with the key as a slice of strings.
func (c *Mcontext) GetStringSlice(key string) (ss []string) {
	if val, ok := c.Get(key); ok && val != nil {
		ss, _ = val.([]string)
	}
	return
}

// GetStringMap returns the value associated with the key as a map of interfaces.
func (c *Mcontext) GetStringMap(key string) (sm map[string]interface{}) {
	if val, ok := c.Get(key); ok && val != nil {
		sm, _ = val.(map[string]interface{})
	}
	return
}

// GetStringMapString returns the value associated with the key as a map of strings.
func (c *Mcontext) GetStringMapString(key string) (sms map[string]string) {
	if val, ok := c.Get(key); ok && val != nil {
		sms, _ = val.(map[string]string)
	}
	return
}

// GetStringMapStringSlice returns the value associated with the key as a map to a slice of strings.
func (c *Mcontext) GetStringMapStringSlice(key string) (smss map[string][]string) {
	if val, ok := c.Get(key); ok && val != nil {
		smss, _ = val.(map[string][]string)
	}
	return
}

/************************************/
/************ INPUT DATA ************/
/************************************/

// Param returns the value of the URL param.
// It is a shortcut for c.Params.ByName(key)
//     router.GET("/user/:id", func(c *gin.Context) {
//         // a GET Context to /user/john
//         id := c.Param("id") // id == "john"
//     })
func (c *Mcontext) Param(key string) string {
	return c.Params.ByName(key)
}

// Query returns the keyed url query value if it exists,
// otherwise it returns an empty string `("")`.
// It is shortcut for `c.Context.URL.Query().Get(key)`
//     GET /path?id=1234&name=Manu&value=
// 	   c.Query("id") == "1234"
// 	   c.Query("name") == "Manu"
// 	   c.Query("value") == ""
// 	   c.Query("wtf") == ""
func (c *Mcontext) Query(key string) string {
	value, _ := c.GetQuery(key)
	return value
}

// DefaultQuery returns the keyed url query value if it exists,
// otherwise it returns the specified defaultValue string.
// See: Query() and GetQuery() for further information.
//     GET /?name=Manu&lastname=
//     c.DefaultQuery("name", "unknown") == "Manu"
//     c.DefaultQuery("id", "none") == "none"
//     c.DefaultQuery("lastname", "none") == ""
func (c *Mcontext) DefaultQuery(key, defaultValue string) string {
	if value, ok := c.GetQuery(key); ok {
		return value
	}
	return defaultValue
}

// GetQuery is like Query(), it returns the keyed url query value
// if it exists `(value, true)` (even when the value is an empty string),
// otherwise it returns `("", false)`.
// It is shortcut for `c.Context.URL.Query().Get(key)`
//     GET /?name=Manu&lastname=
//     ("Manu", true) == c.GetQuery("name")
//     ("", false) == c.GetQuery("id")
//     ("", true) == c.GetQuery("lastname")
func (c *Mcontext) GetQuery(key string) (string, bool) {
	if values, ok := c.GetQueryArray(key); ok {
		return values[0], ok
	}
	return "", false
}

// QueryArray returns a slice of strings for a given query key.
// The length of the slice depends on the number of params with the given key.
func (c *Mcontext) QueryArray(key string) []string {
	values, _ := c.GetQueryArray(key)
	return values
}

func (c *Mcontext) getQueryCache() {
	if c.queryCache == nil {
		c.queryCache = c.OriginRequest.URL.Query()
	}
}

// GetQueryArray returns a slice of strings for a given query key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Mcontext) GetQueryArray(key string) ([]string, bool) {
	c.getQueryCache()
	if values, ok := c.queryCache[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}

// QueryMap returns a map for a given query key.
func (c *Mcontext) QueryMap(key string) map[string]string {
	dicts, _ := c.GetQueryMap(key)
	return dicts
}

// GetQueryMap returns a map for a given query key, plus a boolean value
// whether at least one value exists for the given key.
func (c *Mcontext) GetQueryMap(key string) (map[string]string, bool) {
	c.getQueryCache()
	return c.get(c.queryCache, key)
}

// get is an internal method and returns a map which satisfy conditions.
func (c *Mcontext) get(m map[string][]string, key string) (map[string]string, bool) {
	dicts := make(map[string]string)
	exist := false
	for k, v := range m {
		if i := strings.IndexByte(k, '['); i >= 1 && k[0:i] == key {
			if j := strings.IndexByte(k[i+1:], ']'); j >= 1 {
				exist = true
				dicts[k[i+1:][:j]] = v[0]
			}
		}
	}
	return dicts, exist
}
