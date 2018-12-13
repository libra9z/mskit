package mskit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	code := http.StatusInternalServerError
	msg := err.Error()


	switch err {
	case ErrTwoZeroes, ErrMaxSizeExceeded, ErrIntOverflow:
		code = http.StatusBadRequest
	}


	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errorWrapper{Error: msg})
}

type errorWrapper struct {
	Error string `json:"error"`
}

type RestService interface {
	Get(*Request) (interface{}, error)
	Post(*Request) (interface{}, error)
	Delete(*Request) (interface{}, error)
	Put(*Request) (interface{}, error)
	Head(*Request) (interface{}, error)
	Patch(*Request) (interface{}, error)
	Options(*Request) (interface{}, error)
	Trace(*Request) (interface{}, error)

	//response relate interface
	GetErrorResponse() interface{}
	DecodeRequest(context.Context, *http.Request) (request interface{}, err error)
	EncodeResponse(context.Context, http.ResponseWriter, interface{}) error
}
