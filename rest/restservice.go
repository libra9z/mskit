package rest

import (
	"context"
	"errors"
	"net/http"

	"github.com/libra9z/httprouter"
)

var (
	// ErrTwoZeroes is an arbitrary business rule for the Add method.
	ErrTwoZeroes = errors.New("can't sum two zeroes")

	// ErrIntOverflow protects the Add method. We've decided that this error
	// indicates a misbehaving service and should count against e.g. circuit
	// breakers. So, we return it directly in endpoints, to illustrate the
	// difference. In a real service, this probably wouldn't be the case.
	ErrIntOverflow = errors.New("integer overflow")

	// ErrMaxSizeExceeded protects the Concat method.
	ErrMaxSizeExceeded = errors.New("result exceeds maximum size")
)

type RestService interface {
	Get(context.Context, *Mcontext) (interface{}, error)
	Post(context.Context, *Mcontext) (interface{}, error)
	Delete(context.Context, *Mcontext) (interface{}, error)
	Put(context.Context, *Mcontext) (interface{}, error)
	Head(context.Context, *Mcontext) (interface{}, error)
	Patch(context.Context, *Mcontext) (interface{}, error)
	Options(context.Context, *Mcontext) (interface{}, error)
	Trace(context.Context, *Mcontext) (interface{}, error)

	Prepare(r *Mcontext) (*Mcontext, error)
	Finish(w http.ResponseWriter, response interface{}) error

	//response relate interface
	SetRouter(router *httprouter.Router)
	GetErrorResponse() interface{}
	DecodeRequest(context.Context, *http.Request) (request interface{}, err error)
	EncodeResponse(context.Context, http.ResponseWriter, interface{}) error
}
