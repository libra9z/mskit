package trace

import (
	"context"
	"errors"
	"github.com/libra9z/mskit/v4/log"
	"github.com/libra9z/mskit/v4/rest"
	"github.com/opentracing/opentracing-go"
	zkOt "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/openzipkin/zipkin-go/reporter"
	rhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"net/http"
	"os"
	"strconv"
)

const (
	ZIPKIN_REPORTER_TYPE_HTTP  = "http"
	ZIPKIN_REPORTER_TYPE_KAFKA = "kafka"
	ZIPKIN_REPORTER_TYPE_LOG   = "log"
)

var _ Tracer = (*zipkinTracer)(nil)

type zipkinTracer struct {
	zipkinTracer  *zipkin.Tracer
	zkTracer      opentracing.Tracer
	Name          string
	ServiceName   string
	logger        log.Logger
	flushOnFinish bool

	Tags           map[string]string
	Propagate      bool
	RequestSampler func(r *http.Request) bool
	reporterType   string
	reportUrl      string
	address        string //微服务监听的地址和端口号 例如: 192.168.0.9:7811
}

func NewZipkinTracer(log log.Logger, name, servicename, reportertype, reporturl, address string, tags map[string]string, Propagate, flushOnFinish bool, RequestSampler func(r *http.Request) bool) (Tracer, error) {
	zt := &zipkinTracer{
		logger:         log,
		Name:           name,
		ServiceName:    servicename,
		Tags:           tags,
		Propagate:      Propagate,
		RequestSampler: RequestSampler,
		reporterType:   reportertype,
		reportUrl:      reporturl,
		address:        address,
		flushOnFinish:  flushOnFinish,
	}

	if zt.reporterType == "" {
		zt.reporterType = ZIPKIN_REPORTER_TYPE_HTTP
	}
	var reporter reporter.Reporter
	var err error
	switch zt.reporterType {
	case ZIPKIN_REPORTER_TYPE_HTTP:
		reporter = rhttp.NewReporter(zt.reportUrl)
		if reporter == nil {
			return nil, errors.New("cannot creater a new zipkin reporter")
		}

	}
	ep, err := zipkin.NewEndpoint(zt.ServiceName, zt.address)
	zt.zipkinTracer, err = zipkin.NewTracer(
		reporter, zipkin.WithLocalEndpoint(ep), //zipkin.WithSharedSpans(true), zipkin.WithNoopTracer(useNoopTracer),
	)
	if err != nil {
		zt.logger.Error("error=%v", err)
		os.Exit(1)
	}
	zt.zkTracer = zkOt.Wrap(zt.zipkinTracer)
	opentracing.SetGlobalTracer(zt.zkTracer)
	return zt, nil
}

func (t *zipkinTracer) GetServiceName() string {
	return t.ServiceName
}

func (t *zipkinTracer) GetTraceName() string {
	return t.Name
}
func (t *zipkinTracer) GetTracer() (string, interface{}) {
	return TRACER_TYPE_ZIPKIN, t.zipkinTracer
}

func (t *zipkinTracer) HTTPServerTrace(operatename string) rest.ServerOption {
	serverBefore := rest.ServerBefore(
		func(c *rest.Mcontext, w http.ResponseWriter) error {
			var (
				spanContext model.SpanContext
				name        string
			)

			if t.Name != "" {
				name = t.Name
			} else {
				name = operatename
			}

			if t.Propagate {
				spanContext = t.zipkinTracer.Extract(b3.ExtractHTTP(c.Request))

				if spanContext.Sampled == nil && t.RequestSampler != nil {
					sample := t.RequestSampler(c.Request)
					spanContext.Sampled = &sample
				}

				if spanContext.Err != nil {
					t.logger.Error("error=%v", spanContext.Err)
				}
			}

			tags := map[string]string{
				string(zipkin.TagHTTPMethod): c.Request.Method,
				string(zipkin.TagHTTPPath):   c.Request.URL.Path,
			}

			span := t.zipkinTracer.StartSpan(
				name,
				zipkin.Kind(model.Server),
				zipkin.Tags(t.Tags),
				zipkin.Tags(tags),
				zipkin.Parent(spanContext),
				zipkin.FlushOnFinish(t.flushOnFinish),
			)

			c.Ctx = zipkin.NewContext(c.Ctx, span)
			return nil
		},
	)

	serverAfter := rest.ServerAfter(
		func(c *rest.Mcontext, _ http.ResponseWriter) error {
			if span := zipkin.SpanFromContext(c.Ctx); span != nil {
				span.Finish()
			}

			return nil
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
