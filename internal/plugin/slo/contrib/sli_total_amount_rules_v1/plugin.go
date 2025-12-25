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
	PluginVersion                 = "prometheus/slo/v1"
	PluginID                      = "sloth.dev/contrib/sli_total_amount/v1"
	sliTotalAmountMetric          = "slo:sli_total:amount"
	sliTotalAmountGroupNamePrefix = "sloth-slo-sli-total-amount-"
)

type PluginConfig struct{}

func NewPlugin(c json.RawMessage, _ pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	cfg := &PluginConfig{}
	err := json.Unmarshal(c, cfg)
	if err != nil {
		return nil, err
	}

	return plugin{cfg: *cfg}, nil
}

type plugin struct {
	cfg PluginConfig
}

func (p plugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	if request.SLO.SLI.Events == nil || request.SLO.SLI.Events.TotalQuery == "" {
		return fmt.Errorf("SLI event type with TotalQuery required")
	}

	rules, err := p.generateSLITotalRecordingRules(ctx, request.SLO, request.MWMBAlertGroup)
	if err != nil {
		return err
	}

	customGroup := model.PromRuleGroup{
		Name:     sliTotalAmountGroupNamePrefix + request.SLO.ID,
		Interval: 0, // or set as needed
		Rules:    rules,
	}

	result.SLORules.ExtraRules = append(result.SLORules.ExtraRules, customGroup)
	return nil
}

func (p plugin) generateSLITotalRecordingRules(ctx context.Context, slo model.PromSLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	windows := alerts.TimeDurationWindows()
	windows = append(windows, slo.TimeWindow)

	labels := utilsdata.MergeLabels(conventions.GetSLOIDPromLabels(slo), slo.Labels)
	rules := make([]rulefmt.Rule, 0, len(windows))

	for _, window := range windows {
		windowStr := promutils.TimeDurationToPromStr(window)
		recordName := sliTotalAmountMetric + windowStr

		tpl, err := template.New("totalQuery").Option("missingkey=error").Parse(slo.SLI.Events.TotalQuery)
		if err != nil {
			return nil, fmt.Errorf("could not create template for %s: %w", recordName, err)
		}

		var buf bytes.Buffer
		err = tpl.Execute(&buf, map[string]string{
			conventions.TplSLIQueryWindowVarName: windowStr,
		})
		if err != nil {
			return nil, fmt.Errorf("could not render TotalQuery for %s: %w", recordName, err)
		}

		rule := rulefmt.Rule{
			Record: recordName,
			Expr:   buf.String(),
			Labels: labels,
		}
		rules = append(rules, rule)
	}
	return rules, nil
}
