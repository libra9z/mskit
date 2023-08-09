package metrics

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/libra9z/mskit/v4/endpoint"
	"github.com/libra9z/mskit/v4/log"
	"github.com/libra9z/mskit/v4/rest"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type metricsMmConf struct {
	counter metrics.Counter
	latency metrics.Histogram
}

var imc *metricsMmConf

func InitMetrics(namespace, subsystem string) {
	imc = &metricsMmConf{
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "request_count",
			Help:      "接收到的请求总数.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "request_latency_microseconds",
			Help:      "请求的总时长（毫秒）.",
		}, []string{"method"}),
	}
}
func MetricsMiddleware(logger log.Logger) rest.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			if request == nil {
				return nil, errors.New("no request available")
			}

			req := request.(*rest.Mcontext)
			var method string
			uri := strings.Replace(req.Request.URL.Path, "/", "_", -1)
			method = req.Method + "_" + uri
			//metrics
			defer func(begin time.Time, method string) {
				imc.counter.With("method", method).Add(1)
				imc.latency.With("method", method).Observe(time.Since(begin).Seconds())
			}(time.Now(), method)

			return next(ctx, req)
		}
	}
}
