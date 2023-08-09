package trace

import (
	"context"
	"github.com/libra9z/mskit/v4/log"
	"github.com/libra9z/mskit/v4/rest"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	otrace "go.opentelemetry.io/otel/trace"
	"net/http"
)

const (
	OT_EXPORTER_ZIPKIN = "zipkin"
	OT_EXPORTER_JAEGER = "jaeger"
)

type openTelemetry struct {
	Name          string
	ServiceName   string
	logger        log.Logger
	flushOnFinish bool
	tp            *sdktrace.TracerProvider

	Tags           map[string]string
	Propagate      bool
	RequestSampler func(r *http.Request) bool
	exporterType   string
	exporterUrl    string
	address        string //微服务监听的地址和端口号 例如: 192.168.0.9:7811
}

var _ Tracer = (*openTelemetry)(nil)

func NewOpentelemetryTracer(logger log.Logger, name, servicename, exportertype, exporterurl, address string, tags map[string]string, Propagate, flushOnFinish bool, RequestSampler func(r *http.Request) bool) (Tracer, error) {
	o := &openTelemetry{
		logger:         logger,
		Name:           name,
		ServiceName:    servicename,
		exporterType:   exportertype,
		exporterUrl:    exporterurl,
		address:        address,
		Tags:           tags,
		Propagate:      Propagate,
		flushOnFinish:  flushOnFinish,
		RequestSampler: RequestSampler,
	}
	var err error
	o.tp, err = tracerProvider(o.exporterType, o.exporterUrl, o.ServiceName, "production")
	if err != nil {
		logger.Error("exporter=connnection error.")
		return nil, err
	}
	otel.SetTracerProvider(o.tp)
	return o, nil
}

func tracerProvider(exporterType, url, service, env string) (*sdktrace.TracerProvider, error) {

	var tp *sdktrace.TracerProvider

	switch exporterType {
	case OT_EXPORTER_ZIPKIN:
		//logg := olog.New(os.Stderr, "[zipkin-exporter]: ", olog.Ldate|olog.Ltime|olog.Lmsgprefix)
		exporter, err := zipkin.New(url)
		//exporter, err := zipkin.New(url, zipkin.WithLogger(logg))
		if err != nil {
			return nil, err
		}
		batcher := sdktrace.NewBatchSpanProcessor(exporter)

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(batcher),
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(service),
			)),
		)
	case OT_EXPORTER_JAEGER:
		// Create the Jaeger exporter
		exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
		if err != nil {
			return nil, err
		}
		tp = sdktrace.NewTracerProvider(
			// Always be sure to batch in production.
			sdktrace.WithBatcher(exp),
			// Record information about this application in a Resource.
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(service),
				attribute.String("environment", env),
			)),
		)
	}

	return tp, nil
}

func (t *openTelemetry) GetServiceName() string {
	return t.ServiceName
}

func (t *openTelemetry) GetTraceName() string {
	return t.Name
}

func (t *openTelemetry) GetTracer() (string, interface{}) {
	return TRACER_TYPE_OPENTELEMETRY, t.tp.Tracer(t.Name)
}

func (t *openTelemetry) HTTPServerTrace(operatename string) rest.ServerOption {

	var span otrace.Span

	serverBefore := rest.ServerBefore(
		func(c *rest.Mcontext, w http.ResponseWriter) error {
			var name string

			if t.Name != "" {
				name = t.Name
			} else {
				name = operatename
			}

			tr := t.tp.Tracer(t.Name)
			c.Ctx, span = tr.Start(c.Ctx, name, otrace.WithSpanKind(otrace.SpanKindServer))
			return nil
		},
	)

	serverAfter := rest.ServerAfter(
		func(c *rest.Mcontext, _ http.ResponseWriter) error {
			span.End()
			return nil
		},
	)

	serverFinalizer := rest.ServerFinalizer(
		func(ctx context.Context, code int, r *http.Request) {
			if t.flushOnFinish {
				t.tp.ForceFlush(ctx)
			}
		},
	)

	return func(s *rest.Engine) {
		serverBefore(s)
		serverAfter(s)
		serverFinalizer(s)
	}
}
