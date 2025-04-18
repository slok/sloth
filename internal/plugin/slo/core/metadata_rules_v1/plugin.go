package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"text/template"

	prommodel "github.com/prometheus/common/model"
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

	sloFilter := promutils.LabelsToPromFilter(labels)
	sloGroup := labelsToPromGroup(labels)

	var currentBurnRateExpr bytes.Buffer
	err := burnRateRecordingExprTpl.Execute(&currentBurnRateExpr, map[string]string{
		"SLIErrorMetric":         conventions.GetSLIErrorMetric(alerts.PageQuick.ShortWindow),
		"MetricFilter":           sloFilter,
		"SLOGroup":               sloGroup,
		"ErrorBudgetRatioMetric": metricSLOErrorBudgetRatio,
	})
	if err != nil {
		return nil, fmt.Errorf("could not render current burn rate prometheus metadata recording rule expression: %w", err)
	}

	var periodBurnRateExpr bytes.Buffer
	err = burnRateRecordingExprTpl.Execute(&periodBurnRateExpr, map[string]string{
		"SLIErrorMetric":         conventions.GetSLIErrorMetric(slo.TimeWindow),
		"MetricFilter":           sloFilter,
		"SLOGroup":               sloGroup,
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

// labelsToPromGroup converts a map of labels to a Prometheus filter string.
func labelsToPromGroup(labels map[string]string) string {
	metricGroup := prommodel.LabelNames{}
	for k, _ := range labels {
		metricGroup = append(metricGroup, prommodel.LabelName(k))
	}

	sort.Sort(metricGroup)
	return metricGroup.String()
}

var burnRateRecordingExprTpl = template.Must(template.New("burnRateExpr").Option("missingkey=error").Parse(`{{ .SLIErrorMetric }}{{ .MetricFilter }}
/ on({{ .SLOGroup }}) group_left
{{ .ErrorBudgetRatioMetric }}{{ .MetricFilter }}
`))
