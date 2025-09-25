package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	PluginID      = "sloth.dev/core/metadata_rules/v1"
)

func NewPlugin(_ json.RawMessage, _ pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	return plugin{}, nil
}

type plugin struct{}

func (plugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	metadataRules, err := generateMetadataRecordingRules(ctx, request.Info, request.SLO, request.MWMBAlertGroup)
	if err != nil {
		return err
	}
	result.SLORules.MetadataRecRules.Rules = metadataRules
	return nil
}

func generateMetadataRecordingRules(ctx context.Context, info model.Info, slo model.PromSLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	labels := utilsdata.MergeLabels(conventions.GetSLOIDPromLabels(slo), slo.Labels)

	sloObjectiveRatio := slo.Objective / 100

	sloFilter := promutils.LabelsToPromFilter(conventions.GetSLOIDPromLabels(slo))

	var currentBurnRateExpr bytes.Buffer
	err := burnRateRecordingExprTpl.Execute(&currentBurnRateExpr, map[string]string{
		"SLIErrorMetric":         conventions.GetSLIErrorMetric(alerts.PageQuick.ShortWindow),
		"MetricFilter":           sloFilter,
		"SLOIDName":              conventions.PromSLOIDLabelName,
		"SLOLabelName":           conventions.PromSLONameLabelName,
		"SLOServiceName":         conventions.PromSLOServiceLabelName,
		"ErrorBudgetRatioMetric": conventions.PromMetaSLOErrorBudgetRatioMetric,
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
		"ErrorBudgetRatioMetric": conventions.PromMetaSLOErrorBudgetRatioMetric,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render period burn rate prometheus metadata recording rule expression: %w", err)
	}

	rules := []rulefmt.Rule{
		// SLO Objective.
		{
			Record: conventions.PromMetaSLOObjectiveRatioMetric,
			Expr:   fmt.Sprintf(`vector(%g)`, sloObjectiveRatio),
			Labels: labels,
		},

		// Error budget.
		{
			Record: conventions.PromMetaSLOErrorBudgetRatioMetric,
			Expr:   fmt.Sprintf(`vector(1-%g)`, sloObjectiveRatio),
			Labels: labels,
		},

		// Total period.
		{
			Record: conventions.PromMetaSLOTimePeriodDaysMetric,
			Expr:   fmt.Sprintf(`vector(%g)`, slo.TimeWindow.Hours()/24),
			Labels: labels,
		},

		// Current burning speed.
		{
			Record: conventions.PromMetaSLOCurrentBurnRateRatioMetric,
			Expr:   currentBurnRateExpr.String(),
			Labels: labels,
		},

		// Total period burn rate.
		{
			Record: conventions.PromMetaSLOPeriodBurnRateRatioMetric,
			Expr:   periodBurnRateExpr.String(),
			Labels: labels,
		},

		// Total Error budget remaining period.
		{
			Record: conventions.PromMetaSLOPeriodErrorBudgetRemainingRatioMetric,
			Expr:   fmt.Sprintf(`1 - %s%s`, conventions.PromMetaSLOPeriodBurnRateRatioMetric, sloFilter),
			Labels: labels,
		},

		// Info.
		{
			Record: conventions.PromMetaSLOInfoMetric,
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
