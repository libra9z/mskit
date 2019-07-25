package metrics

import (
	"github.com/go-kit/kit/metrics"
)

type MsMetrics struct {
	Mtype 	string
	counter metrics.Counter
	latency metrics.Histogram
	gauge 	metrics.Gauge
}

type MetricsOption struct {
	PushAddress	string

}

func NewMetrics(mtype string,options ...MetricsOption) *MsMetrics {

	mm := new(MsMetrics)
	mm.Mtype = mtype

	switch mtype {
	case "prometheus":
	case "statsd":
	case "graphite":
	}

	return mm
}