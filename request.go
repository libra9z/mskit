package mskit

import (
	"strconv"
	"github.com/libra9z/httprouter"

	)

type Request struct {
	IsAuthorized	bool
	Version	string
	Params 	httprouter.Params		
	Queries map[string]interface{}
	Body    []byte
	Method  string
}

const(
	maxParam = 50
	)

func NewRequest() *Request {
	return &Request{
		Queries:   make(map[string]interface{}),
	}
}

func (r *Request)GetString(key string) []string{
	var ret []string
	for k,v := range r.Queries {
		if k == key {
			ret = v.([]string)
			break
		}
	}
	
	return ret
}
func (r *Request)GetInt(key string) []int{
	var ret []int

	for k,v := range r.Queries {
		if k == key {
			s := v.([]string)
			
			for _,si := range s {
				iv,_ := strconv.ParseInt(si,10,64)
				ret = append(ret,int(iv))
			}
			
			break
		}
	}
	return ret
}

func (r *Request)GetInt64(key string) []int64{
	var ret []int64

	for k,v := range r.Queries {
		if k == key {
			s := v.([]string)
			
			for _,si := range s {
				iv,_ := strconv.ParseInt(si,10,64)
				ret = append(ret,iv)
			}
			
			break
		}
	}
	return ret
}

func (r *Request)SetAuthorized( auth bool ){
	r.IsAuthorized = auth
}