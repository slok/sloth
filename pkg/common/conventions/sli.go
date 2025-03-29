package conventions

import (
	"fmt"
	"time"

	promutils "github.com/slok/sloth/pkg/common/utils/prometheus"
)

// GetSLIErrorMetric returns the SLI error Prometheus metric name.
func GetSLIErrorMetric(window time.Duration) string {
	return fmt.Sprintf(PromSLIErrorMetricFmt, promutils.TimeDurationToPromStr(window))
}
