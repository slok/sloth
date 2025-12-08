package prometheus_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	metricsprometheus "github.com/slok/sloth/internal/http/backend/metrics/prometheus"
)

func TestPrometheusMetricsRecorder(t *testing.T) {
	tests := map[string]struct {
		measure    func(t *testing.T, r metricsprometheus.Recorder)
		expMetrics string
	}{
		"Measuring Prometheus Storage Background Cache Refresh should measure correctly.": {
			measure: func(t *testing.T, r metricsprometheus.Recorder) {
				r.MeasurePrometheusStorageBackgroundCacheRefresh(t.Context(), 1500*time.Millisecond, nil)
				r.MeasurePrometheusStorageBackgroundCacheRefresh(t.Context(), 500*time.Millisecond, nil)
				r.MeasurePrometheusStorageBackgroundCacheRefresh(t.Context(), 2500*time.Millisecond, fmt.Errorf("some error"))
			},
			expMetrics: `
				# HELP sloth_storage_prometheus_cache_background_refresh_duration_seconds Duration histogram of Prometheus storage cache refresh operations.
				# TYPE sloth_storage_prometheus_cache_background_refresh_duration_seconds histogram
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="0.005"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="0.01"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="0.025"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="0.05"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="0.1"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="0.25"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="0.5"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="1"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="2.5"} 1
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="5"} 1
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="10"} 1
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="false",le="+Inf"} 1
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_sum{success="false"} 2.5
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_count{success="false"} 1
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="0.005"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="0.01"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="0.025"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="0.05"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="0.1"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="0.25"} 0
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="0.5"} 1
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="1"} 1
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="2.5"} 2
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="5"} 2
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="10"} 2
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_bucket{success="true",le="+Inf"} 2
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_sum{success="true"} 2
				sloth_storage_prometheus_cache_background_refresh_duration_seconds_count{success="true"} 2
			`,
		},

		"Measuring Prometheus API Client Operation should measure correctly.": {
			measure: func(t *testing.T, r metricsprometheus.Recorder) {
				r.MeasurePrometheusAPIClientOperation(t.Context(), "Query", 1200*time.Millisecond, nil)
				r.MeasurePrometheusAPIClientOperation(t.Context(), "Query", 800*time.Millisecond, nil)
				r.MeasurePrometheusAPIClientOperation(t.Context(), "QueryRange", 3000*time.Millisecond, fmt.Errorf("some error"))
			},
			expMetrics: `
				# HELP sloth_prometheus_api_client_operation_duration_seconds Duration histogram of Prometheus API client operations.
            	# TYPE sloth_prometheus_api_client_operation_duration_seconds histogram
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="0.005"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="0.01"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="0.025"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="0.05"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="0.1"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="0.25"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="0.5"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="1"} 1
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="2.5"} 2
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="5"} 2
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="10"} 2
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="Query",success="true",le="+Inf"} 2
            	sloth_prometheus_api_client_operation_duration_seconds_sum{operation="Query",success="true"} 2
            	sloth_prometheus_api_client_operation_duration_seconds_count{operation="Query",success="true"} 2
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="0.005"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="0.01"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="0.025"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="0.05"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="0.1"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="0.25"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="0.5"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="1"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="2.5"} 0
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="5"} 1
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="10"} 1
            	sloth_prometheus_api_client_operation_duration_seconds_bucket{operation="QueryRange",success="false",le="+Inf"} 1
            	sloth_prometheus_api_client_operation_duration_seconds_sum{operation="QueryRange",success="false"} 3
            	sloth_prometheus_api_client_operation_duration_seconds_count{operation="QueryRange",success="false"} 1
			`,
		},
		"Measuring Storage Operation Duration should measure correctly.": {
			measure: func(t *testing.T, r metricsprometheus.Recorder) {
				r.MeasureStorageOperationDuration(t.Context(), "GetSLOs", 700*time.Millisecond, nil)
				r.MeasureStorageOperationDuration(t.Context(), "GetSLOs", 400*time.Millisecond, nil)
				r.MeasureStorageOperationDuration(t.Context(), "GetServices", 2000*time.Millisecond, fmt.Errorf("some error"))
			},
			expMetrics: `
				# HELP sloth_storage_operation_duration_seconds Duration histogram of storage operations.
            	# TYPE sloth_storage_operation_duration_seconds histogram
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="0.005"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="0.01"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="0.025"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="0.05"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="0.1"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="0.25"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="0.5"} 1
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="1"} 2
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="2.5"} 2
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="5"} 2
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="10"} 2
            	sloth_storage_operation_duration_seconds_bucket{operation="GetSLOs",success="true",le="+Inf"} 2
            	sloth_storage_operation_duration_seconds_sum{operation="GetSLOs",success="true"} 1.1
            	sloth_storage_operation_duration_seconds_count{operation="GetSLOs",success="true"} 2
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="0.005"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="0.01"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="0.025"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="0.05"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="0.1"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="0.25"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="0.5"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="1"} 0
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="2.5"} 1
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="5"} 1
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="10"} 1
            	sloth_storage_operation_duration_seconds_bucket{operation="GetServices",success="false",le="+Inf"} 1
            	sloth_storage_operation_duration_seconds_sum{operation="GetServices",success="false"} 2
            	sloth_storage_operation_duration_seconds_count{operation="GetServices",success="false"} 1
			`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			reg := prometheus.NewRegistry()
			rec := metricsprometheus.NewRecorder(reg)

			test.measure(t, rec)

			// Check metrics.
			err := testutil.GatherAndCompare(reg, strings.NewReader(test.expMetrics))
			assert.NoError(err)
		})
	}
}
