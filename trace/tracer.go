package trace

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/openzipkin/zipkin-go"
)

type Tracer interface {
	GetOpenTracer()(opentracing.Tracer)
	GetZipkinTracer()(*zipkin.Tracer)
	GetServiceName()(string)
	StartSpanFromContext(name string,ctx context.Context) (opentracing.Span,context.Context)
}

type trace struct {
	withTracer 			bool
	withZipkinTracer 	bool
	ServiceName		 	string
	tracer 				opentracing.Tracer
	zipkinTracer		*zipkin.Tracer
}

type TraceOption func(*trace)

func WithTracerOption(istracer bool) TraceOption {
	return func(t *trace){t.withTracer = istracer}
}
func WithZipkinTracerOption(istracer bool) TraceOption {
	return func(t *trace){t.withZipkinTracer = istracer}
}

func WithServiceNameOption(serviceName string) TraceOption {
	return func(t *trace){t.ServiceName = serviceName}
}

func OpenTracerOption(tracer opentracing.Tracer) TraceOption {
	return func(t *trace){t.tracer = tracer}
}
func ZipkinTracerOption(tracer *zipkin.Tracer) TraceOption {
	return func(t *trace){t.zipkinTracer = tracer}
}
func(t *trace)GetOpenTracer() opentracing.Tracer{
	return t.tracer
}
func(t *trace)GetZipkinTracer() *zipkin.Tracer{
	return t.zipkinTracer
}

func(t *trace)StartSpanFromContext(name string,ctx context.Context) (opentracing.Span,context.Context) {
	return opentracing.StartSpanFromContext(ctx,name)
}

func(t *trace)GetServiceName() string{
	return t.ServiceName
}

func NewTracer(options... TraceOption) Tracer {
	t := &trace{}
	for _,option := range options{
		option(t)
	}

	return t
}