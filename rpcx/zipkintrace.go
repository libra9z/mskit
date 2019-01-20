package rpcx

import (
	"context"
	"platform/mskit/log"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
)


func RpcxClientTrace(tracer *zipkin.Tracer, options ...TracerOption) ClientOption {
	config := tracerOptions{
		tags:      make(map[string]string),
		name:      "",
		logger:    log.Mslog,
		propagate: true,
	}

	for _, option := range options {
		option(&config)
	}

	clientBefore := ClientBefore(
		func(ctx context.Context, md *MD) context.Context {
			var (
				spanContext model.SpanContext
				name        string
			)

			if config.name != "" {
				name = config.name
			} else {
				name = ctx.Value(ContextKeyRequestMethod).(string)
			}

			if parent := zipkin.SpanFromContext(ctx); parent != nil {
				spanContext = parent.Context()
			}

			span := tracer.StartSpan(
				name,
				zipkin.Kind(model.Client),
				zipkin.Tags(config.tags),
				zipkin.Parent(spanContext),
				zipkin.FlushOnFinish(false),
			)

			if config.propagate {
				mm := make(map[string]string)
				for k,v := range *md {
					mm[k] = v
				}
				if err := InjectRpcx(&mm)(span.Context()); err != nil {
					config.logger.Log("err", err)
				}
			}

			return zipkin.NewContext(ctx, span)
		},
	)

	clientAfter := ClientAfter(
		func(ctx context.Context, _ MD, _ MD) context.Context {
			if span := zipkin.SpanFromContext(ctx); span != nil {
				span.Finish()
			}

			return ctx
		},
	)

	clientFinalizer := ClientFinalizer(
		func(ctx context.Context, err error) {
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
		},
	)

	return func(c *Client) {
		clientBefore(c)
		clientAfter(c)
		clientFinalizer(c)
	}

}
