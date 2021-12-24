package opentrace


import (
	"net"
	"net/http"
	"strconv"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/go-kit/kit/log"
	"github.com/libra9z/mskit/v4/rest"
)

// ContextToHTTP returns an http RequestFunc that injects an OpenTracing Span
// found in `ctx` into the http headers. If no such Span can be found, the
// RequestFunc is a noop.
func ContextToHTTP(tracer opentracing.Tracer, logger log.Logger) rest.RequestFunc {
	return func(c *rest.Mcontext, _ http.ResponseWriter) {
		// Try to find a Span in the Context.
		if span := opentracing.SpanFromContext(c.Ctx); span != nil {
			// Add standard OpenTracing tags.
			ext.HTTPMethod.Set(span, c.Request.Method)
			ext.HTTPUrl.Set(span, c.Request.URL.String())
			host, portString, err := net.SplitHostPort(c.Request.URL.Host)
			if err == nil {
				ext.PeerHostname.Set(span, host)
				if port, err := strconv.Atoi(portString); err == nil {
					ext.PeerPort.Set(span, uint16(port))
				}
			} else {
				ext.PeerHostname.Set(span, c.Request.URL.Host)
			}

			// There's nothing we can do with any errors here.
			if err = tracer.Inject(
				span.Context(),
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(c.Request.Header),
			); err != nil {
				logger.Log("err", err)
			}
		}
	}
}

// HTTPToContext returns an http RequestFunc that tries to join with an
// OpenTracing trace found in `req` and starts a new Span called
// `operationName` accordingly. If no trace could be found in `req`, the Span
// will be a trace root. The Span is incorporated in the returned Context and
// can be retrieved with opentracing.SpanFromContext(ctx).
func HTTPToContext(tracer opentracing.Tracer, operationName string, logger log.Logger) rest.RequestFunc {
	return func(c *rest.Mcontext, _ http.ResponseWriter) {
		// Try to join to a trace propagated in `req`.
		var span opentracing.Span
		wireContext, err := tracer.Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(c.Request.Header),
		)
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			logger.Log("err", err)
		}

		span = tracer.StartSpan(operationName, ext.RPCServerOption(wireContext))
		ext.HTTPMethod.Set(span, c.Request.Method)
		ext.HTTPUrl.Set(span, c.Request.URL.String())
		c.Ctx= opentracing.ContextWithSpan(c.Ctx, span)
	}
}