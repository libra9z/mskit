package zipkin

import (
	"context"
	"github.com/go-kit/kit/log"
	"github.com/libra9z/mskit/trace"
	"github.com/libra9z/mskit/rest"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"net/http"
	"strconv"
)

func HTTPServerTrace(tracer *zipkin.Tracer, options ...trace.TracerOption) rest.ServerOption {
	config := trace.TracerOptions{
		Tags:      make(map[string]string),
		Name:      "",
		Logger:    log.NewNopLogger(),
		Propagate: true,
	}

	for _, option := range options {
		option(&config)
	}

	serverBefore := rest.ServerBefore(
		func(c *rest.Mcontext, w http.ResponseWriter)  {
			var (
				spanContext model.SpanContext
				name        string
			)

			if config.Name != "" {
				name = config.Name
			} else {
				name = c.Request.Method
			}

			if config.Propagate {
				spanContext = tracer.Extract(b3.ExtractHTTP(c.Request))

				if spanContext.Sampled == nil && config.RequestSampler != nil {
					sample := config.RequestSampler(c.Request)
					spanContext.Sampled = &sample
				}

				if spanContext.Err != nil {
					config.Logger.Log("err", spanContext.Err)
				}
			}

			tags := map[string]string{
				string(zipkin.TagHTTPMethod): c.Request.Method,
				string(zipkin.TagHTTPPath):   c.Request.URL.Path,
			}

			span := tracer.StartSpan(
				name,
				zipkin.Kind(model.Server),
				zipkin.Tags(config.Tags),
				zipkin.Tags(tags),
				zipkin.Parent(spanContext),
				zipkin.FlushOnFinish(false),
			)

			c.Ctx = zipkin.NewContext(c.Ctx, span)
		},
	)

	serverAfter := rest.ServerAfter(
		func(c *rest.Mcontext, _ http.ResponseWriter) {
			if span := zipkin.SpanFromContext(c.Ctx); span != nil {
				span.Finish()
			}
		},
	)

	serverFinalizer := rest.ServerFinalizer(
		func(ctx context.Context, code int, r *http.Request) {
			if span := zipkin.SpanFromContext(ctx); span != nil {
				zipkin.TagHTTPStatusCode.Set(span, strconv.Itoa(code))
				if code > 399 {
					// set http status as error tag (if already set, this is a noop)
					zipkin.TagError.Set(span, http.StatusText(code))
				}
				if rs, ok := ctx.Value(rest.ContextKeyResponseSize).(int64); ok {
					zipkin.TagHTTPResponseSize.Set(span, strconv.FormatInt(rs, 10))
				}

				// calling span.Finish() a second time is a noop, if we didn't get to
				// ServerAfter we can at least time the early bail out by calling it
				// here.
				span.Finish()
				// send span to the Reporter
				span.Flush()
			}
		},
	)

	return func(s *rest.Engine) {
		serverBefore(s)
		serverAfter(s)
		serverFinalizer(s)
	}
}