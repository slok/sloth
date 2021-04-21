package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/prometheus/prometheus/pkg/rulefmt"

	"github.com/slok/sloth/internal/alert"
)

// sliRulesgenFunc knows how to generate an SLI recording rule for a specific time window.
type sliRulesgenFunc func(slo SLO, window time.Duration) (*rulefmt.Rule, error)

type sliRecordingRulesGenerator struct {
	genFunc sliRulesgenFunc
}

// SLIRecordingRulesGenerator knows how to generate the SLI prometheus recording rules
// form an SLO. Normally these rules are used by the SLO alerts.
var SLIRecordingRulesGenerator = sliRecordingRulesGenerator{genFunc: defaultSLIRecordGenerator}

func (s sliRecordingRulesGenerator) GenerateSLIRecordingRules(ctx context.Context, slo SLO, alerts alert.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	// Get the windows we need the recording rules.
	windows := getAlertGroupWindows(alerts)
	windows = append(windows, slo.TimeWindow) // Add the total time window as a handy helper.

	// Generate the rules
	rules := make([]rulefmt.Rule, 0, len(windows))
	for _, window := range windows {
		rule, err := s.genFunc(slo, window)
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

func defaultSLIRecordGenerator(slo SLO, window time.Duration) (*rulefmt.Rule, error) {
	// Generate our first level of template by assembling the error and total expressions.
	sliExprTpl := fmt.Sprintf(`(%s)
/
(%s)
`, slo.SLI.ErrorQuery, slo.SLI.TotalQuery)

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
			slo.Labels,
			slo.GetSLOIDPromLabels(),
			map[string]string{
				sloWindowLabelName: strWindow,
			}),
	}, nil
}

type metadataRecordingRulesGenerator bool

// MetadataRecordingRulesGenerator knows how to generate the metadata prometheus recording rules
// from an SLO.
const MetadataRecordingRulesGenerator = metadataRecordingRulesGenerator(false)

func (m metadataRecordingRulesGenerator) GenerateMetadataRecordingRules(ctx context.Context, slo SLO, alerts alert.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	labels := mergeLabels(slo.Labels, slo.GetSLOIDPromLabels())

	// Metatada Recordings.
	const (
		metricSLOObjectiveRatio                  = "slo:objective:ratio"
		metricSLOErrorBudgetRatio                = "slo:error_budget:ratio"
		metricSLOTimePeriodDays                  = "slo:time_period:days"
		metricSLOCurrentBurnRateRatio            = "slo:current_burn_rate:ratio"
		metricSLOPeriodBurnRateRatio             = "slo:period_burn_rate:ratio"
		metricSLOPeriodErrorBudgetRemainingRatio = "slo:period_error_budget_remaining:ratio"
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
	}

	return rules, nil
}

var burnRateRecordingExprTpl = template.Must(template.New("burnRateExpr").Option("missingkey=error").Parse(`{{ .SLIErrorMetric }}{{ .MetricFilter }}
/ on({{ .SLOIDName }}, {{ .SLOLabelName }}, {{ .SLOServiceName }}) group_left
{{ .ErrorBudgetRatioMetric }}{{ .MetricFilter }}
`))
