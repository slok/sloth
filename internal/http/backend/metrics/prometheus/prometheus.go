package prometheus

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	Prefix = "sloth"
)

type Recorder struct {
	reg prometheus.Registerer

	storagePromCacheLatency *prometheus.HistogramVec
	storageOperationLatency *prometheus.HistogramVec
	promAPICliLatency       *prometheus.HistogramVec
}

func NewRecorder(reg prometheus.Registerer) Recorder {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	r := &Recorder{
		reg: reg,

		storagePromCacheLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Prefix,
				Subsystem: "storage_prometheus",
				Name:      "cache_background_refresh_duration_seconds",
				Help:      "Duration histogram of Prometheus storage cache refresh operations.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"success"},
		),

		storageOperationLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Prefix,
				Subsystem: "storage",
				Name:      "operation_duration_seconds",
				Help:      "Duration histogram of storage operations.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation", "success"},
		),

		promAPICliLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Prefix,
				Subsystem: "prometheus_api_client",
				Name:      "operation_duration_seconds",
				Help:      "Duration histogram of Prometheus API client operations.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation", "success"},
		),
	}

	r.init()

	return *r
}

func (r Recorder) init() {
	// Register our collectors.
	r.reg.MustRegister(
		r.storagePromCacheLatency,
		r.promAPICliLatency,
		r.storageOperationLatency,
	)
}

func (r Recorder) MeasurePrometheusStorageBackgroundCacheRefresh(ctx context.Context, t time.Duration, err error) {
	r.storagePromCacheLatency.WithLabelValues(strconv.FormatBool(err == nil)).Observe(t.Seconds())
}

func (r Recorder) MeasurePrometheusAPIClientOperation(ctx context.Context, op string, t time.Duration, err error) {
	r.promAPICliLatency.WithLabelValues(op, strconv.FormatBool(err == nil)).Observe(t.Seconds())
}

func (r Recorder) MeasureStorageOperationDuration(ctx context.Context, op string, t time.Duration, err error) {
	r.storageOperationLatency.WithLabelValues(op, strconv.FormatBool(err == nil)).Observe(t.Seconds())
}
