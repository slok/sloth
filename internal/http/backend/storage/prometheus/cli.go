package prometheus

import (
	"context"
	"time"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/slok/sloth/internal/http/backend/metrics"
)

// PrometheusAPIClient is an interface that defines the methods we use from the Prometheus client.
// We define it so we can add flexibility like easily mocking in tests or wrap it for functionality.
type PrometheusAPIClient interface {
	prometheusv1.API
}

func NewMeasuredPrometheusAPIClient(metricsRecorder metrics.Recorder, promcli PrometheusAPIClient) PrometheusAPIClient {
	return measuredPrometheusAPIClient{
		PrometheusAPIClient: promcli,
		metricsRecorder:     metricsRecorder,
	}
}

type measuredPrometheusAPIClient struct {
	PrometheusAPIClient
	metricsRecorder metrics.Recorder
}

func (m measuredPrometheusAPIClient) Query(ctx context.Context, query string, ts time.Time, opts ...prometheusv1.Option) (v model.Value, w prometheusv1.Warnings, err error) {
	start := time.Now()
	defer func() {
		m.metricsRecorder.MeasurePrometheusAPIClientOperation(ctx, "Query", time.Since(start), err)
	}()
	return m.PrometheusAPIClient.Query(ctx, query, ts, opts...)
}

func (m measuredPrometheusAPIClient) QueryRange(ctx context.Context, query string, r prometheusv1.Range, opts ...prometheusv1.Option) (v model.Value, w prometheusv1.Warnings, err error) {
	start := time.Now()
	defer func() {
		m.metricsRecorder.MeasurePrometheusAPIClientOperation(ctx, "QueryRange", time.Since(start), err)
	}()
	return m.PrometheusAPIClient.QueryRange(ctx, query, r, opts...)
}
