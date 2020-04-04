package rpcx

import (
	"context"
	"github.com/libra9z/mskit/metadata"
	"github.com/opentracing/opentracing-go"
	"github.com/smallnest/rpcx/share"
	"github.com/libra9z/mskit/log"
	"github.com/libra9z/mskit/trace"
)


func RpcxClientOpenTracing(tracer trace.Tracer, options ...trace.TracerOption) ClientOption {
	config := trace.TracerOptions{
		Tags:      make(map[string]string),
		Name:      "",
		Logger:    log.Mslog,
		Propagate: true,
	}

	for _, option := range options {
		option(&config)
	}

	clientBefore := ClientBefore(
		func(ctx context.Context, mmd *map[string]string) context.Context {
			ctx = context.WithValue(ctx,share.ReqMetaDataKey,*mmd)
			if tracer == nil {
				return ctx
			}
			if span := opentracing.SpanFromContext(ctx); span != nil {
				// There's nothing we can do with an error here.
				var md metadata.MD
				md = make(metadata.MD)
				for k,v := range *mmd {
					md[k] = v
				}
				if err := tracer.GetOpenTracer().Inject(span.Context(), opentracing.TextMap, metadataReaderWriter{&md}); err != nil {
					config.Logger.Log("err", err)
				}
			}else {
				span,ctx = tracer.StartSpanFromContext(tracer.GetServiceName(),ctx)
			}
			return ctx
		},
	)

	clientAfter := ClientAfter(
		func(ctx context.Context, _ map[string]string, _ map[string]string) context.Context {
			if span := opentracing.SpanFromContext(ctx); span != nil {
				span.Finish()
			}

			return ctx
		},
	)

	clientFinalizer := ClientFinalizer(
		func(ctx context.Context, err error) {
			if span := opentracing.SpanFromContext(ctx); span != nil {
				span.Finish()
			}
		},
	)

	return func(c *Client) {
		clientBefore(c)
		clientAfter(c)
		clientFinalizer(c)
	}

}
