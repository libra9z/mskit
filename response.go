package mskit

import (

)

type Response struct {
	Data	map[interface{}]interface{}
}

func NewResponse() *Response {
	return new( Response)
}

func (r *Response)GetErrorResponse(resp interface{}) interface{}{
	
	return nil
}


func (r *Response)GetSuccessResponse(resp interface{}) interface{}{
	
	return nil
}