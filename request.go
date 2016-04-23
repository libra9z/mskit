package mskit

import (

)


type Request struct {
	value	map[string]interface{}
	body	[]byte
	Method	string
}

func NewRequest() *Request {
	return new( Request)
}

