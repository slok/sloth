package prometheus

import (
	"sort"
	"time"

	prommodel "github.com/prometheus/common/model"

	"github.com/slok/sloth/internal/alert"
)

func mergeLabels(ms ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}

func labelsToPromFilter(labels map[string]string) string {
	metricFilters := prommodel.LabelSet{}
	for k, v := range labels {
		metricFilters[prommodel.LabelName(k)] = prommodel.LabelValue(v)
	}

	return metricFilters.String()
}

// Pretty simple durations for prometheus.
func timeDurationToPromStr(t time.Duration) string {
	return prommodel.Duration(t).String()
}

// getAlertGroupWindows gets all the time windows from a multiwindow multiburn alert group.
func getAlertGroupWindows(alerts alert.MWMBAlertGroup) []time.Duration {
	// Use a map to avoid duplicated windows.
	windows := map[string]time.Duration{
		alerts.PageQuick.ShortWindow.String():   alerts.PageQuick.ShortWindow,
		alerts.PageQuick.LongWindow.String():    alerts.PageQuick.LongWindow,
		alerts.PageSlow.ShortWindow.String():    alerts.PageSlow.ShortWindow,
		alerts.PageSlow.LongWindow.String():     alerts.PageSlow.LongWindow,
		alerts.TicketQuick.ShortWindow.String(): alerts.TicketQuick.ShortWindow,
		alerts.TicketQuick.LongWindow.String():  alerts.TicketQuick.LongWindow,
		alerts.TicketSlow.ShortWindow.String():  alerts.TicketSlow.ShortWindow,
		alerts.TicketSlow.LongWindow.String():   alerts.TicketSlow.LongWindow,
	}

	res := make([]time.Duration, 0, len(windows))
	for _, w := range windows {
		res = append(res, w)
	}
	sort.SliceStable(res, func(i, j int) bool { return res[i] < res[j] })

	return res
}
