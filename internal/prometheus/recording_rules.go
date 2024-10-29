package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"text/template"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/info"
)

// sliRulesgenFunc knows how to generate an SLI recording rule for a specific time window.
type sliRulesgenFunc func(slo SLO, window time.Duration, alerts alert.MWMBAlertGroup) (*rulefmt.Rule, error)

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

func optimizedFactorySLIRecordGenerator(slo SLO, window time.Duration, alerts alert.MWMBAlertGroup) (*rulefmt.Rule, error) {
	// Optimize the rules that are for the total period time window.
	if window == slo.TimeWindow {
		return optimizedSLIRecordGenerator(slo, window, alerts.PageQuick.ShortWindow)
	}

	return factorySLIRecordGenerator(slo, window, alerts)
}

func (s sliRecordingRulesGenerator) GenerateSLIRecordingRules(ctx context.Context, slo SLO, alerts alert.MWMBAlertGroup) ([]rulefmt.Rule, error) {
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

func factorySLIRecordGenerator(slo SLO, window time.Duration, alerts alert.MWMBAlertGroup) (*rulefmt.Rule, error) {
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

func rawSLIRecordGenerator(slo SLO, window time.Duration, alerts alert.MWMBAlertGroup) (*rulefmt.Rule, error) {
	// Render with our templated data.
	sliExprTpl := fmt.Sprintf(`(%s)`, slo.SLI.Raw.ErrorRatioQuery)
	tpl, err := template.New("sliExpr").Option("missingkey=error").Parse(sliExprTpl)
	if err != nil {
		return nil, fmt.Errorf("could not create SLI expression template data: %w", err)
	}

	strWindow := timeDurationToPromStr(window)
	var b bytes.Buffer
	err = tpl.Execute(&b, map[string]string{
		tplKeyWindow: strWindow,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render SLI expression template: %w", err)
	}

	return &rulefmt.Rule{
		Record: slo.GetSLIErrorMetric(window),
		Expr:   b.String(),
		Labels: mergeLabels(
			slo.GetSLOIDPromLabels(),
			map[string]string{
				sloWindowLabelName: strWindow,
			},
			slo.Labels,
		),
	}, nil
}

func eventsSLIRecordGenerator(slo SLO, window time.Duration, alerts alert.MWMBAlertGroup) (*rulefmt.Rule, error) {
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

	strWindow := timeDurationToPromStr(window)
	var b bytes.Buffer
	err = tpl.Execute(&b, map[string]string{
		tplKeyWindow: strWindow,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render SLI expression template: %w", err)
	}

	return &rulefmt.Rule{
		Record: slo.GetSLIErrorMetric(window),
		Expr:   b.String(),
		Labels: mergeLabels(
			slo.GetSLOIDPromLabels(),
			map[string]string{
				sloWindowLabelName: strWindow,
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

	shortWindowSLIRec := slo.GetSLIErrorMetric(shortWindow)
	filter := labelsToPromFilter(slo.GetSLOIDPromLabels())

	// Render with our templated data.
	tpl, err := template.New("sliExpr").Option("missingkey=error").Parse(sliExprTplFmt)
	if err != nil {
		return nil, fmt.Errorf("could not create SLI expression template data: %w", err)
	}

	strWindow := timeDurationToPromStr(window)
	var b bytes.Buffer
	err = tpl.Execute(&b, map[string]string{
		"metric":    shortWindowSLIRec,
		"filter":    filter,
		"window":    strWindow,
		"windowKey": sloWindowLabelName,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render SLI expression template: %w", err)
	}

	return &rulefmt.Rule{
		Record: slo.GetSLIErrorMetric(window),
		Expr:   b.String(),
		Labels: mergeLabels(
			slo.GetSLOIDPromLabels(),
			map[string]string{
				sloWindowLabelName: strWindow,
			},
			slo.Labels,
		),
	}, nil
}

type metadataRecordingRulesGenerator bool

// MetadataRecordingRulesGenerator knows how to generate the metadata prometheus recording rules
// from an SLO.
const MetadataRecordingRulesGenerator = metadataRecordingRulesGenerator(false)

func (m metadataRecordingRulesGenerator) GenerateMetadataRecordingRules(ctx context.Context, info info.Info, slo SLO, alerts alert.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	labels := mergeLabels(slo.GetSLOIDPromLabels(), slo.Labels)

	infoLabels := mergeLabels(labels, map[string]string{
		sloVersionLabelName:   info.Version,
		sloModeLabelName:      string(info.Mode),
		sloSpecLabelName:      info.Spec,
		sloObjectiveLabelName: strconv.FormatFloat(slo.Objective, 'f', -1, 64),
	})

	infoLabels = mergeLabels(infoLabels, slo.InfoLabels)

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

	sloFilter := labelsToPromFilter(slo.GetSLOIDPromLabels())

	var currentBurnRateExpr bytes.Buffer
	err := burnRateRecordingExprTpl.Execute(&currentBurnRateExpr, map[string]string{
		"SLIErrorMetric":         slo.GetSLIErrorMetric(alerts.PageQuick.ShortWindow),
		"MetricFilter":           sloFilter,
		"SLOIDName":              sloIDLabelName,
		"SLOLabelName":           sloNameLabelName,
		"SLOServiceName":         sloServiceLabelName,
		"ErrorBudgetRatioMetric": metricSLOErrorBudgetRatio,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render current burn rate prometheus metadata recording rule expression: %w", err)
	}

	var periodBurnRateExpr bytes.Buffer
	err = burnRateRecordingExprTpl.Execute(&periodBurnRateExpr, map[string]string{
		"SLIErrorMetric":         slo.GetSLIErrorMetric(slo.TimeWindow),
		"MetricFilter":           sloFilter,
		"SLOIDName":              sloIDLabelName,
		"SLOLabelName":           sloNameLabelName,
		"SLOServiceName":         sloServiceLabelName,
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
			Labels: infoLabels,
		},
	}

	return rules, nil
}

var burnRateRecordingExprTpl = template.Must(template.New("burnRateExpr").Option("missingkey=error").Parse(`{{ .SLIErrorMetric }}{{ .MetricFilter }}
/ on({{ .SLOIDName }}, {{ .SLOLabelName }}, {{ .SLOServiceName }}) group_left
{{ .ErrorBudgetRatioMetric }}{{ .MetricFilter }}
`))
