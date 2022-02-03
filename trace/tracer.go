package trace

import (
	"github.com/go-kit/kit/log"
	"github.com/libra9z/mskit/v4/rest"
	"net/http"
)

const(
	TRACER_TYPE_OPENTRACING		= "opentracing"
	TRACER_TYPE_ZIPKIN			= "zipkin"
	TRACER_TYPE_OPENTELEMETRY	= "opentelemetry"
)

type Tracer interface {
	GetServiceName()(string)
	GetTraceName()(string)
	GetTracer()(string,interface{})
	HTTPServerTrace(operatename string) rest.ServerOption
}

type trace struct {
	Tags           map[string]string
	Name           string
	Logger         log.Logger
	Propagate      bool
	RequestSampler func(r *http.Request) bool

	withTracer 			bool
	withZipkinTracer 	bool
	ServiceName		 	string
	tracerType			string
	tracer 				Tracer

	reporterType 	string
	reportUrl		string
	address			string  //微服务监听的地址和端口号 例如: 192.168.0.9:7811

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
func WithTraceType(typ string) TraceOption {
	return func(t *trace){t.tracerType =typ}
}
func WithTraceName(name string) TraceOption {
	return func(t *trace){ t.Name = name }
}
func WithTraceTags(tags map[string]string) TraceOption {
	return func(t *trace){ t.Tags = tags }
}
func WithTraceLog(logger log.Logger) TraceOption {
	return func(t *trace){ t.Logger = logger }
}
func WithAllowPropagation(propagate bool) TraceOption {
	return func(t *trace){ t.Propagate = propagate }
}
func WithReporterType(reporterType string) TraceOption{
	return func(t *trace){ t.reporterType = reporterType }
}
func WithReporterURL(reporterurl string) TraceOption{
	return func(t *trace){ t.reportUrl = reporterurl }
}
func WithAddress(address string) TraceOption{
	return func(t *trace){ t.address = address }
}

func(t *trace)GetServiceName() string{
	return t.ServiceName
}
func(t *trace)GetTraceName() string{
	return t.Name
}
func(t *trace)HTTPServerTrace(operatename string) rest.ServerOption {
	return t.tracer.HTTPServerTrace(operatename)
}
func(t *trace)GetTracer()(string,interface{}) {
	return t.tracer.GetTracer()
}

func NewTracer(options... TraceOption) Tracer {
	t := &trace{}
	for _,option := range options{
		option(t)
	}
	var err error
	switch t.tracerType {
	case TRACER_TYPE_ZIPKIN:
		t.tracer,err = NewZipkinTracer(t.Logger,t.Name,t.ServiceName,t.reporterType,t.reportUrl,t.address,t.Tags,t.Propagate,true,t.RequestSampler)
	case TRACER_TYPE_OPENTELEMETRY:
		t.tracer,err = NewOpentelemetryTracer(t.Logger,t.Name,t.ServiceName,t.reporterType,t.reportUrl,t.address,t.Tags,t.Propagate,true,t.RequestSampler)
	}

	if err != nil {
		t.Logger.Log("error",err.Error())
		return nil
	}

	return t
}