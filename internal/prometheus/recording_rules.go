package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"text/template"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"

	"github.com/slok/sloth/pkg/common/conventions"
	"github.com/slok/sloth/pkg/common/model"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
	promutils "github.com/slok/sloth/pkg/common/utils/prometheus"
)

// sliRulesgenFunc knows how to generate an SLI recording rule for a specific time window.
type sliRulesgenFunc func(slo SLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error)

type sliRecordingRulesGenerator struct {
	genFunc sliRulesgenFunc
}

// OptimizedSLIRecordingRulesGenerator knows how to generate the SLI prometheus recording rules
// from an SLO optimizing where it can.
// Normally these rules are used by the SLO alerts.
var OptimizedSLIRecordingRulesGenerator = sliRecordingRulesGenerator{genFunc: optimizedFactorySLIRecordGenerator}

// SLIRecordingRulesGenerator knows how to generate the SLI prometheus recording rules
// form an SLO.
// Normally these rules are used by the SLO alerts.
var SLIRecordingRulesGenerator = sliRecordingRulesGenerator{genFunc: factorySLIRecordGenerator}

func optimizedFactorySLIRecordGenerator(slo SLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error) {
	// Optimize the rules that are for the total period time window.
	if window == slo.TimeWindow {
		return optimizedSLIRecordGenerator(slo, window, alerts.PageQuick.ShortWindow)
	}

	return factorySLIRecordGenerator(slo, window, alerts)
}

func (s sliRecordingRulesGenerator) GenerateSLIRecordingRules(ctx context.Context, slo SLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	// Get the windows we need the recording rules.
	windows := getAlertGroupWindows(alerts)
	windows = append(windows, slo.TimeWindow) // Add the total time window as a handy helper.

	// Generate the rules
	rules := make([]rulefmt.Rule, 0, len(windows))
	for _, window := range windows {
		rule, err := s.genFunc(slo, window, alerts)
		if err != nil {
			return nil, fmt.Errorf("could not create %q SLO rule for window %s: %w", slo.ID, window, err)
		}
		rules = append(rules, *rule)
	}

	return rules, nil
}

const (
	tplKeyWindow = "window"
)

func factorySLIRecordGenerator(slo SLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error) {
	switch {
	// Event based SLI.
	case slo.SLI.Events != nil:
		return eventsSLIRecordGenerator(slo, window, alerts)
	// Raw based SLI.
	case slo.SLI.Raw != nil:
		return rawSLIRecordGenerator(slo, window, alerts)
	}

	return nil, fmt.Errorf("invalid SLI type")
}

func rawSLIRecordGenerator(slo SLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error) {
	// Render with our templated data.
	sliExprTpl := fmt.Sprintf(`(%s)`, slo.SLI.Raw.ErrorRatioQuery)
	tpl, err := template.New("sliExpr").Option("missingkey=error").Parse(sliExprTpl)
	if err != nil {
		return nil, fmt.Errorf("could not create SLI expression template data: %w", err)
	}

	strWindow := promutils.TimeDurationToPromStr(window)
	var b bytes.Buffer
	err = tpl.Execute(&b, map[string]string{
		tplKeyWindow: strWindow,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render SLI expression template: %w", err)
	}

	return &rulefmt.Rule{
		Record: conventions.GetSLIErrorMetric(window),
		Expr:   b.String(),
		Labels: utilsdata.MergeLabels(
			conventions.GetSLOIDPromLabels(slo),
			map[string]string{
				conventions.PromSLOWindowLabelName: strWindow,
			},
			slo.Labels,
		),
	}, nil
}

func eventsSLIRecordGenerator(slo SLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error) {
	const sliExprTplFmt = `(%s)
/
(%s)
`
	// Generate our first level of template by assembling the error and total expressions.
	sliExprTpl := fmt.Sprintf(sliExprTplFmt, slo.SLI.Events.ErrorQuery, slo.SLI.Events.TotalQuery)

	// Render with our templated data.
	tpl, err := template.New("sliExpr").Option("missingkey=error").Parse(sliExprTpl)
	if err != nil {
		return nil, fmt.Errorf("could not create SLI expression template data: %w", err)
	}

	strWindow := promutils.TimeDurationToPromStr(window)
	var b bytes.Buffer
	err = tpl.Execute(&b, map[string]string{
		tplKeyWindow: strWindow,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render SLI expression template: %w", err)
	}

	return &rulefmt.Rule{
		Record: conventions.GetSLIErrorMetric(window),
		Expr:   b.String(),
		Labels: utilsdata.MergeLabels(
			conventions.GetSLOIDPromLabels(slo),
			map[string]string{
				conventions.PromSLOWindowLabelName: strWindow,
			},
			slo.Labels,
		),
	}, nil
}

// optimizedSLIRecordGenerator gets a SLI recording rule from other SLI recording rules. This optimization
// will make Prometheus consume less CPU and memory, however the result will be less accurate. Used wisely
// is a good tradeoff. For example on calculating informative metrics like total period window (30d).
//
// The way this optimization is made is using one SLI recording rule (the one with the shortest window to
// reduce the downsampling, e.g 5m) and make an average over time on that rule for the window time range.
func optimizedSLIRecordGenerator(slo SLO, window, shortWindow time.Duration) (*rulefmt.Rule, error) {
	// Averages over ratios (average over average) is statistically incorrect, so we do
	// aggregate all ratios on the time window and then divide with the aggregation of all the full ratios
	// that is 1 (thats why we can use `count`), giving use a correct ratio of ratios:
	// - https://prometheus.io/docs/practices/rules/
	// - https://math.stackexchange.com/questions/95909/why-is-an-average-of-an-average-usually-incorrect
	const sliExprTplFmt = `sum_over_time({{.metric}}{{.filter}}[{{.window}}])
/ ignoring ({{.windowKey}})
count_over_time({{.metric}}{{.filter}}[{{.window}}])
`

	if window == shortWindow {
		return nil, fmt.Errorf("can't optimize using the same shortwindow as the window to optimize")
	}

	shortWindowSLIRec := conventions.GetSLIErrorMetric(shortWindow)
	filter := promutils.LabelsToPromFilter(conventions.GetSLOIDPromLabels(slo))

	// Render with our templated data.
	tpl, err := template.New("sliExpr").Option("missingkey=error").Parse(sliExprTplFmt)
	if err != nil {
		return nil, fmt.Errorf("could not create SLI expression template data: %w", err)
	}

	strWindow := promutils.TimeDurationToPromStr(window)
	var b bytes.Buffer
	err = tpl.Execute(&b, map[string]string{
		"metric":    shortWindowSLIRec,
		"filter":    filter,
		"window":    strWindow,
		"windowKey": conventions.PromSLOWindowLabelName,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render SLI expression template: %w", err)
	}

	return &rulefmt.Rule{
		Record: conventions.GetSLIErrorMetric(window),
		Expr:   b.String(),
		Labels: utilsdata.MergeLabels(
			conventions.GetSLOIDPromLabels(slo),
			map[string]string{
				conventions.PromSLOWindowLabelName: strWindow,
			},
			slo.Labels,
		),
	}, nil
}

type metadataRecordingRulesGenerator bool

// MetadataRecordingRulesGenerator knows how to generate the metadata prometheus recording rules
// from an SLO.
const MetadataRecordingRulesGenerator = metadataRecordingRulesGenerator(false)

func (m metadataRecordingRulesGenerator) GenerateMetadataRecordingRules(ctx context.Context, info model.Info, slo SLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	labels := utilsdata.MergeLabels(conventions.GetSLOIDPromLabels(slo), slo.Labels)

	// Metatada Recordings.
	const (
		metricSLOObjectiveRatio                  = "slo:objective:ratio"
		metricSLOErrorBudgetRatio                = "slo:error_budget:ratio"
		metricSLOTimePeriodDays                  = "slo:time_period:days"
		metricSLOCurrentBurnRateRatio            = "slo:current_burn_rate:ratio"
		metricSLOPeriodBurnRateRatio             = "slo:period_burn_rate:ratio"
		metricSLOPeriodErrorBudgetRemainingRatio = "slo:period_error_budget_remaining:ratio"
		metricSLOInfo                            = "sloth_slo_info"
	)

	sloObjectiveRatio := slo.Objective / 100

	sloFilter := promutils.LabelsToPromFilter(conventions.GetSLOIDPromLabels(slo))

	var currentBurnRateExpr bytes.Buffer
	err := burnRateRecordingExprTpl.Execute(&currentBurnRateExpr, map[string]string{
		"SLIErrorMetric":         conventions.GetSLIErrorMetric(alerts.PageQuick.ShortWindow),
		"MetricFilter":           sloFilter,
		"SLOIDName":              conventions.PromSLOIDLabelName,
		"SLOLabelName":           conventions.PromSLONameLabelName,
		"SLOServiceName":         conventions.PromSLOServiceLabelName,
		"ErrorBudgetRatioMetric": metricSLOErrorBudgetRatio,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render current burn rate prometheus metadata recording rule expression: %w", err)
	}

	var periodBurnRateExpr bytes.Buffer
	err = burnRateRecordingExprTpl.Execute(&periodBurnRateExpr, map[string]string{
		"SLIErrorMetric":         conventions.GetSLIErrorMetric(slo.TimeWindow),
		"MetricFilter":           sloFilter,
		"SLOIDName":              conventions.PromSLOIDLabelName,
		"SLOLabelName":           conventions.PromSLONameLabelName,
		"SLOServiceName":         conventions.PromSLOServiceLabelName,
		"ErrorBudgetRatioMetric": metricSLOErrorBudgetRatio,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render period burn rate prometheus metadata recording rule expression: %w", err)
	}

	rules := []rulefmt.Rule{
		// SLO Objective.
		{
			Record: metricSLOObjectiveRatio,
			Expr:   fmt.Sprintf(`vector(%g)`, sloObjectiveRatio),
			Labels: labels,
		},

		// Error budget.
		{
			Record: metricSLOErrorBudgetRatio,
			Expr:   fmt.Sprintf(`vector(1-%g)`, sloObjectiveRatio),
			Labels: labels,
		},

		// Total period.
		{
			Record: metricSLOTimePeriodDays,
			Expr:   fmt.Sprintf(`vector(%g)`, slo.TimeWindow.Hours()/24),
			Labels: labels,
		},

		// Current burning speed.
		{
			Record: metricSLOCurrentBurnRateRatio,
			Expr:   currentBurnRateExpr.String(),
			Labels: labels,
		},

		// Total period burn rate.
		{
			Record: metricSLOPeriodBurnRateRatio,
			Expr:   periodBurnRateExpr.String(),
			Labels: labels,
		},

		// Total Error budget remaining period.
		{
			Record: metricSLOPeriodErrorBudgetRemainingRatio,
			Expr:   fmt.Sprintf(`1 - %s%s`, metricSLOPeriodBurnRateRatio, sloFilter),
			Labels: labels,
		},

		// Info.
		{
			Record: metricSLOInfo,
			Expr:   `vector(1)`,
			Labels: utilsdata.MergeLabels(labels, map[string]string{
				conventions.PromSLOVersionLabelName:   info.Version,
				conventions.PromSLOModeLabelName:      string(info.Mode),
				conventions.PromSLOSpecLabelName:      info.Spec,
				conventions.PromSLOObjectiveLabelName: strconv.FormatFloat(slo.Objective, 'f', -1, 64),
			}),
		},
	}

	return rules, nil
}

var burnRateRecordingExprTpl = template.Must(template.New("burnRateExpr").Option("missingkey=error").Parse(`{{ .SLIErrorMetric }}{{ .MetricFilter }}
/ on({{ .SLOIDName }}, {{ .SLOLabelName }}, {{ .SLOServiceName }}) group_left
{{ .ErrorBudgetRatioMetric }}{{ .MetricFilter }}
`))

// getAlertGroupWindows gets all the time windows from a multiwindow multiburn alert group.
func getAlertGroupWindows(alerts model.MWMBAlertGroup) []time.Duration {
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
