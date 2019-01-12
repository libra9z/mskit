package rpcx

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/smallnest/rpcx/share"
	. "platform/mskit/log"
)

// Generate span info from context, generate a new span when context is empty or
// will generate span from parentSpan
func GenerateSpanWithContext(ctx context.Context, tracer opentracing.Tracer ,operationName string) (opentracing.Span, context.Context, error) {

	md := ctx.Value(share.ReqMetaDataKey)
	var span opentracing.Span
	var parentSpan opentracing.Span

	//tracer := opentracing.GlobalTracer()
	if md != nil {
		carrier := opentracing.TextMapCarrier(md.(map[string]string))
		spanContext, err := tracer.Extract(opentracing.TextMap, carrier)
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			Mslog.Log("funcname","GenerateSpanWithContext","error", err)
		} else {
			parentSpan = tracer.StartSpan(operationName, ext.RPCServerOption(spanContext))
		}
	}

	if parentSpan != nil {
		span = opentracing.GlobalTracer().StartSpan(operationName, opentracing.ChildOf(parentSpan.Context()))
	} else {
		span = opentracing.StartSpan(operationName)
	}

	metadata := opentracing.TextMapCarrier(make(map[string]string))
	err := tracer.Inject(span.Context(), opentracing.TextMap, metadata)
	if err != nil {
		return nil, nil, err
	}
	ctx = context.WithValue(context.Background(), share.ReqMetaDataKey, (map[string]string)(metadata))
	return span, ctx, nil
}