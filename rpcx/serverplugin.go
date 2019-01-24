package rpcx

import (
	"context"
	"fmt"
	"platform/mskit/log"
	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/smallnest/rpcx/protocol"
)

type ZipkinTracePlugin struct {
	tracer 		*zipkin.Tracer
}

func NewZipkinTracePlugin(ziptracer *zipkin.Tracer) *ZipkinTracePlugin {
	z := &ZipkinTracePlugin{
		tracer:ziptracer,
	}

	return z
}

func (p *ZipkinTracePlugin) PostReadRequest(ctx context.Context, r *protocol.Message, e error) error {
	config := tracerOptions{
		tags:      make(map[string]string),
		name:      "",
		logger:    log.Mslog,
		propagate: true,
	}

	var (
		spanContext model.SpanContext
		name        string
		tags        = make(map[string]string)
	)

	rpcMethod, ok := ctx.Value(ContextKeyRequestMethod).(string)
	if !ok {
		config.logger.Log("err", "unable to retrieve method name: missing rpcx interceptor hook")
	} else {
		tags["rpcx.method"] = rpcMethod
	}

	config.logger.Log("rpcx.method",rpcMethod)

	if config.name != "" {
		name = config.name
	} else {
		name = rpcMethod
	}

	if config.propagate {
		spanContext = p.tracer.Extract(ExtractRpcx(&r.Metadata))
		if spanContext.Err != nil {
			config.logger.Log("err", spanContext.Err)
		}
	}

	fmt.Printf("Metadata=%+v\n",r)

	span := p.tracer.StartSpan(
		name,
		zipkin.Kind(model.Server),
		zipkin.Tags(config.tags),
		zipkin.Tags(tags),
		zipkin.Parent(spanContext),
		zipkin.FlushOnFinish(false),
	)
	ctx = zipkin.NewContext(ctx, span)
	return nil
}

func (p *ZipkinTracePlugin) PostWriteResponse(ctx context.Context, req *protocol.Message, res *protocol.Message, err error) error {

	if span := zipkin.SpanFromContext(ctx); span != nil {
		if err != nil {
			zipkin.TagError.Set(span, err.Error())
		}
		// calling span.Finish() a second time is a noop, if we didn't get to
		// ClientAfter we can at least time the early bail out by calling it
		// here.
		span.Finish()
		// send span to the Reporter
		span.Flush()
	}

	return nil
}