package prometheus

import (
	"fmt"
	"time"

	prommodel "github.com/prometheus/common/model"
)

// TimeDurationToPromStr converts from std duration to prom string duration.
func TimeDurationToPromStr(t time.Duration) string {
	return prommodel.Duration(t).String()
}

// PromStrToTimeDuration converts from prom string duration to std duration.
func PromStrToTimeDuration(t string) (time.Duration, error) {
	d, err := prommodel.ParseDuration(t)
	if err != nil {
		return 0, fmt.Errorf("could not parse prom duration %q: %w", t, err)
	}
	return time.Duration(d), nil
}

// LabelsToPromFilter converts a map of labels to a Prometheus filter string.
func LabelsToPromFilter(labels map[string]string) string {
	metricFilters := prommodel.LabelSet{}
	for k, v := range labels {
		metricFilters[prommodel.LabelName(k)] = prommodel.LabelValue(v)
	}

	return metricFilters.String()
}
