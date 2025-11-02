package ui

import (
	gohttpmetrics "github.com/slok/go-http-metrics/metrics"
)

// MetricsRecorder is the service used to record metrics in the HTTP API handler.
type MetricsRecorder interface {
	gohttpmetrics.Recorder
}

var noopMetricsRecorder = struct {
	gohttpmetrics.Recorder
}{
	Recorder: gohttpmetrics.Dummy,
}
