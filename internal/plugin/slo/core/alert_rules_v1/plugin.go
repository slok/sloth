package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/prometheus/prometheus/model/rulefmt"

	"github.com/slok/sloth/pkg/common/conventions"
	"github.com/slok/sloth/pkg/common/model"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
	promutils "github.com/slok/sloth/pkg/common/utils/prometheus"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/core/alert_rules/v1"
)

func NewPlugin(_ json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	return plugin{
		appUtils: appUtils,
	}, nil
}

type plugin struct {
	appUtils pluginslov1.AppUtils
}

func (p plugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	rules, err := p.generateSLOAlertRules(ctx, request.SLO, request.MWMBAlertGroup)
	if err != nil {
		return err
	}

	result.SLORules.AlertRules.Rules = rules

	return nil
}

func (p plugin) generateSLOAlertRules(ctx context.Context, slo model.PromSLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	rules := []rulefmt.Rule{}

	// Generate Page alerts.
	if !slo.PageAlertMeta.Disable {
		rule, err := defaultSLOAlertGenerator(slo, slo.PageAlertMeta, alerts.PageQuick, alerts.PageSlow)
		if err != nil {
			return nil, fmt.Errorf("could not create page alert: %w", err)
		}

		rules = append(rules, *rule)
	}

	// Generate Ticket alerts.
	if !slo.TicketAlertMeta.Disable {
		rule, err := defaultSLOAlertGenerator(slo, slo.TicketAlertMeta, alerts.TicketQuick, alerts.TicketSlow)
		if err != nil {
			return nil, fmt.Errorf("could not create ticket alert: %w", err)
		}

		rules = append(rules, *rule)
	}

	return rules, nil
}

func defaultSLOAlertGenerator(slo model.PromSLO, sloAlert model.PromAlertMeta, quick, slow model.MWMBAlert) (*rulefmt.Rule, error) {
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
		Annotations: utilsdata.MergeLabels(extraAnnotations, sloAlert.Annotations),
		Labels:      utilsdata.MergeLabels(extraLabels, sloAlert.Labels),
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
