package prometheus

import (
	"fmt"
	"time"

	"github.com/slok/sloth/pkg/common/conventions"
	"github.com/slok/sloth/pkg/common/model"
)

// TODO(slok): Remove after migration to pkg/common/model package.
type SLO = model.PromSLO
type SLI = model.PromSLI
type SLIEvents = model.PromSLIEvents
type AlertMeta = model.PromAlertMeta
type SLOGroup = model.PromSLOGroup
type SLORules = model.PromSLORules
type SLIRaw = model.PromSLIRaw

// getSLIErrorMetric returns the SLI error metric.
func getSLIErrorMetric(window time.Duration) string {
	return fmt.Sprintf(conventions.PromSLIErrorMetricFmt, timeDurationToPromStr(window))
}

// getSLOIDPromLabels returns the ID labels of an SLO, these can be used to identify
// an SLO recorded metrics and alerts.
func getSLOIDPromLabels(s SLO) map[string]string {
	return map[string]string{
		conventions.PromSLOIDLabelName:      s.ID,
		conventions.PromSLONameLabelName:    s.Name,
		conventions.PromSLOServiceLabelName: s.Service,
	}
}
