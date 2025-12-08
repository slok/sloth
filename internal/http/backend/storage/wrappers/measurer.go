package wrappers

import (
	"context"
	"time"

	"github.com/slok/sloth/internal/http/backend/metrics"
	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
)

type measuredServiceGetter struct {
	orig    storage.ServiceGetter
	metrics metrics.Recorder
}

func NewMeasuredServiceGetter(orig storage.ServiceGetter, metricsRecorder metrics.Recorder) storage.ServiceGetter {
	return measuredServiceGetter{
		orig:    orig,
		metrics: metricsRecorder,
	}
}

func (m measuredServiceGetter) ListAllServiceAndAlerts(ctx context.Context) (sa []storage.ServiceAndAlerts, err error) {
	t0 := time.Now()
	defer func() {
		m.metrics.MeasureStorageOperationDuration(ctx, "ListAllServiceAndAlerts", time.Since(t0), err)
	}()

	return m.orig.ListAllServiceAndAlerts(ctx)
}

func (m measuredServiceGetter) ListServiceAndAlertsByServiceSearch(ctx context.Context, serviceSearchInput string) (sa []storage.ServiceAndAlerts, err error) {
	t0 := time.Now()
	defer func() {
		m.metrics.MeasureStorageOperationDuration(ctx, "ListServiceAndAlertsByServiceSearch", time.Since(t0), err)
	}()
	return m.orig.ListServiceAndAlertsByServiceSearch(ctx, serviceSearchInput)
}

type measuredSLOGetter struct {
	orig    storage.SLOGetter
	metrics metrics.Recorder
}

func NewMeasuredSLOGetter(orig storage.SLOGetter, metricsRecorder metrics.Recorder) storage.SLOGetter {
	return measuredSLOGetter{
		orig:    orig,
		metrics: metricsRecorder,
	}
}

func (m measuredSLOGetter) ListSLOInstantDetailsService(ctx context.Context, serviceID string) (sloDetails []storage.SLOInstantDetails, err error) {
	t0 := time.Now()
	defer func() {
		m.metrics.MeasureStorageOperationDuration(ctx, "ListSLOInstantDetailsService", time.Since(t0), err)
	}()
	return m.orig.ListSLOInstantDetailsService(ctx, serviceID)
}

func (m measuredSLOGetter) ListSLOInstantDetailsServiceBySLOSearch(ctx context.Context, serviceID, sloSearchInput string) (sloDetails []storage.SLOInstantDetails, err error) {
	t0 := time.Now()
	defer func() {
		m.metrics.MeasureStorageOperationDuration(ctx, "ListSLOInstantDetailsServiceBySLOSearch", time.Since(t0), err)
	}()
	return m.orig.ListSLOInstantDetailsServiceBySLOSearch(ctx, serviceID, sloSearchInput)
}

func (m measuredSLOGetter) ListSLOInstantDetails(ctx context.Context) (sloDetails []storage.SLOInstantDetails, err error) {
	t0 := time.Now()
	defer func() {
		m.metrics.MeasureStorageOperationDuration(ctx, "ListSLOInstantDetails", time.Since(t0), err)
	}()
	return m.orig.ListSLOInstantDetails(ctx)
}

func (m measuredSLOGetter) ListSLOInstantDetailsBySLOSearch(ctx context.Context, sloSearchInput string) (sloDetails []storage.SLOInstantDetails, err error) {
	t0 := time.Now()
	defer func() {
		m.metrics.MeasureStorageOperationDuration(ctx, "ListSLOInstantDetailsBySLOSearch", time.Since(t0), err)
	}()
	return m.orig.ListSLOInstantDetailsBySLOSearch(ctx, sloSearchInput)
}

func (m measuredSLOGetter) GetSLOInstantDetails(ctx context.Context, sloID string) (sloDetail *storage.SLOInstantDetails, err error) {
	t0 := time.Now()
	defer func() {
		m.metrics.MeasureStorageOperationDuration(ctx, "GetSLOInstantDetails", time.Since(t0), err)
	}()
	return m.orig.GetSLOInstantDetails(ctx, sloID)
}

func (m measuredSLOGetter) GetSLIAvailabilityInRange(ctx context.Context, sloID string, from, to time.Time, step time.Duration) (dataPoints []model.DataPoint, err error) {
	t0 := time.Now()
	defer func() {
		m.metrics.MeasureStorageOperationDuration(ctx, "GetSLIAvailabilityInRange", time.Since(t0), err)
	}()
	return m.orig.GetSLIAvailabilityInRange(ctx, sloID, from, to, step)
}

func (m measuredSLOGetter) GetSLIAvailabilityInRangeAutoStep(ctx context.Context, sloID string, from, to time.Time) (dataPoints []model.DataPoint, err error) {
	t0 := time.Now()
	defer func() {
		m.metrics.MeasureStorageOperationDuration(ctx, "GetSLIAvailabilityInRangeAutoStep", time.Since(t0), err)
	}()
	return m.orig.GetSLIAvailabilityInRangeAutoStep(ctx, sloID, from, to)
}
