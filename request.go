package mskit

import (

)


type Request struct {
	Value	map[string]interface{}
	Body	[]byte
	Method	string
}

func NewRequest() *Request {
	return new( Request)
}

