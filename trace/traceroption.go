package trace

import (
	"net/http"

	"github.com/libra9z/mskit/v4/log"
)

// TracerOption allows for functional options to our Zipkin tracing middleware.
type TracerOption func(o *TracerOptions)

// Name sets the name for an instrumented transport endpoint. If name is omitted
// at tracing middleware creation, the method of the transport or transport rpc
// name is used.
func Name(name string) TracerOption {
	return func(o *TracerOptions) {
		o.Name = name
	}
}

// Tags adds default tags to our Zipkin transport spans.
func Tags(tags map[string]string) TracerOption {
	return func(o *TracerOptions) {
		for k, v := range tags {
			o.Tags[k] = v
		}
	}
}

// Logger adds a Go kit logger to our Zipkin Middleware to log SpanContext
// extract / inject errors if they occur. Default is Noop.
func Logger(logger log.Logger) TracerOption {
	return func(o *TracerOptions) {
		o.Logger = logger
	}
}

// AllowPropagation instructs the tracer to allow or deny propagation of the
// span context between this instrumented client or service and its peers. If
// the instrumented client connects to services outside its own platform or if
// the instrumented service receives requests from untrusted clients it is
// strongly advised to disallow propagation. Propagation between services inside
// your own platform benefit from propagation. Default for both TraceClient and
// TraceServer is to allow propagation.
func AllowPropagation(propagate bool) TracerOption {
	return func(o *TracerOptions) {
		o.Propagate = propagate
	}
}

// RequestSampler allows one to set the sampling decision based on the details
// found in the http.Request.
func RequestSampler(sampleFunc func(r *http.Request) bool) TracerOption {
	return func(o *TracerOptions) {
		o.RequestSampler = sampleFunc
	}
}

type TracerOptions struct {
	Tags           map[string]string
	Name           string
	Logger         log.Logger
	Propagate      bool
	RequestSampler func(r *http.Request) bool
}
