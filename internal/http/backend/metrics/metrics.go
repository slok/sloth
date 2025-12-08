package metrics

import (
	"context"
	"time"
)

type Recorder interface {
	MeasureStorageOperationDuration(ctx context.Context, op string, t time.Duration, err error)
	MeasurePrometheusStorageBackgroundCacheRefresh(ctx context.Context, t time.Duration, err error)
	MeasurePrometheusAPIClientOperation(ctx context.Context, op string, t time.Duration, err error)
}

type noopRecorder bool

var NoopRecorder Recorder = noopRecorder(false)

func (r noopRecorder) MeasureStorageOperationDuration(ctx context.Context, op string, t time.Duration, err error) {
}

func (r noopRecorder) MeasurePrometheusStorageBackgroundCacheRefresh(ctx context.Context, t time.Duration, err error) {
}

func (r noopRecorder) MeasurePrometheusAPIClientOperation(ctx context.Context, op string, t time.Duration, err error) {
}
