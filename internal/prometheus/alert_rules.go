package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/prometheus/prometheus/model/rulefmt"

	"github.com/slok/sloth/pkg/common/conventions"
	"github.com/slok/sloth/pkg/common/model"
	promutils "github.com/slok/sloth/pkg/common/utils/prometheus"
)

// genFunc knows how to generate an SLI recording rule for a specific time window.
type alertGenFunc func(slo SLO, sloAlert AlertMeta, quick, slow model.MWMBAlert) (*rulefmt.Rule, error)

type sloAlertRulesGenerator struct {
	alertGenFunc alertGenFunc
}

// SLOAlertRulesGenerator knows how to generate the SLO prometheus alert rules
// from an SLO.
var SLOAlertRulesGenerator = sloAlertRulesGenerator{alertGenFunc: defaultSLOAlertGenerator}

func (s sloAlertRulesGenerator) GenerateSLOAlertRules(ctx context.Context, slo SLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	rules := []rulefmt.Rule{}

	// Generate Page alerts.
	if !slo.PageAlertMeta.Disable {
		rule, err := s.alertGenFunc(slo, slo.PageAlertMeta, alerts.PageQuick, alerts.PageSlow)
		if err != nil {
			return nil, fmt.Errorf("could not create page alert: %w", err)
		}

		rules = append(rules, *rule)
	}

	// Generate Ticket alerts.
	if !slo.TicketAlertMeta.Disable {
		rule, err := s.alertGenFunc(slo, slo.TicketAlertMeta, alerts.TicketQuick, alerts.TicketSlow)
		if err != nil {
			return nil, fmt.Errorf("could not create ticket alert: %w", err)
		}

		rules = append(rules, *rule)
	}

	return rules, nil
}

func defaultSLOAlertGenerator(slo SLO, sloAlert AlertMeta, quick, slow model.MWMBAlert) (*rulefmt.Rule, error) {
	// Generate the filter labels based on the SLO ids.
	metricFilter := promutils.LabelsToPromFilter(conventions.GetSLOIDPromLabels(slo))

	// Render the alert template.
	tplData := struct {
		MetricFilter         string
		ErrorBudgetRatio     float64
		QuickShortMetric     string
		QuickShortBurnFactor float64
		QuickLongMetric      string
		QuickLongBurnFactor  float64
		SlowShortMetric      string
		SlowShortBurnFactor  float64
		SlowQuickMetric      string
		SlowQuickBurnFactor  float64
		WindowLabel          string
	}{
		MetricFilter:         metricFilter,
		ErrorBudgetRatio:     quick.ErrorBudget / 100, // Any(quick or slow) should work because are the same.
		QuickShortMetric:     conventions.GetSLIErrorMetric(quick.ShortWindow),
		QuickShortBurnFactor: quick.BurnRateFactor,
		QuickLongMetric:      conventions.GetSLIErrorMetric(quick.LongWindow),
		QuickLongBurnFactor:  quick.BurnRateFactor,
		SlowShortMetric:      conventions.GetSLIErrorMetric(slow.ShortWindow),
		SlowShortBurnFactor:  slow.BurnRateFactor,
		SlowQuickMetric:      conventions.GetSLIErrorMetric(slow.LongWindow),
		SlowQuickBurnFactor:  slow.BurnRateFactor,
		WindowLabel:          conventions.PromSLOWindowLabelName,
	}
	var expr bytes.Buffer
	err := mwmbAlertTpl.Execute(&expr, tplData)
	if err != nil {
		return nil, fmt.Errorf("could not render alert expression: %w", err)
	}

	// Add specific annotations.
	severity := quick.Severity.String() // Any(quick or slow) should work because are the same.
	extraAnnotations := map[string]string{
		"title":   fmt.Sprintf("(%s) {{$labels.%s}} {{$labels.%s}} SLO error budget burn rate is too fast.", severity, conventions.PromSLOServiceLabelName, conventions.PromSLONameLabelName),
		"summary": fmt.Sprintf("{{$labels.%s}} {{$labels.%s}} SLO error budget burn rate is over expected.", conventions.PromSLOServiceLabelName, conventions.PromSLONameLabelName),
	}

	// Add specific labels. We don't add the labels from the rules because we will
	// inherit on the alerts, this way we avoid warnings of overrided labels.
	extraLabels := map[string]string{
		conventions.PromSLOSeverityLabelName: severity,
	}

	return &rulefmt.Rule{
		Alert:       sloAlert.Name,
		Expr:        expr.String(),
		Annotations: mergeLabels(extraAnnotations, sloAlert.Annotations),
		Labels:      mergeLabels(extraLabels, sloAlert.Labels),
	}, nil
}

// Multiburn multiwindow alert template.
var mwmbAlertTpl = template.Must(template.New("mwmbAlertTpl").Option("missingkey=error").Parse(`(
    max({{ .QuickShortMetric }}{{ .MetricFilter}} > ({{ .QuickShortBurnFactor }} * {{ .ErrorBudgetRatio }})) without ({{ .WindowLabel }})
    and
    max({{ .QuickLongMetric }}{{ .MetricFilter}} > ({{ .QuickLongBurnFactor }} * {{ .ErrorBudgetRatio }})) without ({{ .WindowLabel }})
)
or
(
    max({{ .SlowShortMetric }}{{ .MetricFilter }} > ({{ .SlowShortBurnFactor }} * {{ .ErrorBudgetRatio }})) without ({{ .WindowLabel }})
    and
    max({{ .SlowQuickMetric }}{{ .MetricFilter }} > ({{ .SlowQuickBurnFactor }} * {{ .ErrorBudgetRatio }})) without ({{ .WindowLabel }})
)
`))
