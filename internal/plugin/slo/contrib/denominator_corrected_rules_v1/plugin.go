package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/slok/sloth/pkg/common/conventions"
	"github.com/slok/sloth/pkg/common/model"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
	promutils "github.com/slok/sloth/pkg/common/utils/prometheus"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/contrib/denominator_corrected_rules/v1"
)

const (
	numeratorCorrectionMetric = "slo:numerator_correction:ratio" // The correction factor metric name.
)

type PluginConfig struct {
	DisableOptimized bool `json:"disableOptimized,omitempty"`
}

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
	// Check requirements for this type of SLOs.
	if request.SLO.SLI.Events == nil || request.SLO.SLI.Events.ErrorQuery == "" || request.SLO.SLI.Events.TotalQuery == "" {
		return fmt.Errorf("denominator corrected SLI requires SLI event type")
	}

	// Generate and override SLI recordings.
	sliRules, err := p.generateSLIRecordingRules(ctx, request.SLO, request.MWMBAlertGroup)
	if err != nil {
		return err
	}
	result.SLORules.SLIErrorRecRules.Rules = sliRules

	// Add required new metadata recordings with the correction factor.
	metaRules, err := p.generateMetaRecordingRules(ctx, request.SLO, request.MWMBAlertGroup)
	if err != nil {
		return err
	}
	result.SLORules.MetadataRecRules.Rules = append(result.SLORules.MetadataRecRules.Rules, metaRules...)

	return nil
}

func (p plugin) generateSLIRecordingRules(ctx context.Context, slo model.PromSLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	// Get the windows we need the recording rules.
	windows := alerts.TimeDurationWindows()
	windows = append(windows, slo.TimeWindow) // Add the total time window as a handy helper.

	// Generate the rules
	rules := make([]rulefmt.Rule, 0, len(windows))
	for _, window := range windows {
		rule, err := p.denominatorCorrectedSLIRecordGenerator(slo, window, alerts)
		if err != nil {
			return nil, fmt.Errorf("could not create %q SLO rule for window %s: %w", slo.ID, window, err)
		}
		rules = append(rules, *rule)
	}

	return rules, nil
}

func (p plugin) generateMetaRecordingRules(ctx context.Context, slo model.PromSLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	metadataLabels := utilsdata.MergeLabels(conventions.GetSLOIDPromLabels(slo), slo.Labels)
	rules := []rulefmt.Rule{}
	for _, window := range alerts.TimeDurationWindows() {
		rule, err := createNumeratorCorrection(slo, metadataLabels, window, slo.TimeWindow)
		if err != nil {
			return nil, fmt.Errorf("could not create numerator rule: %v", err)
		}
		rules = append(rules, *rule)
	}
	return rules, nil
}

func (p plugin) denominatorCorrectedSLIRecordGenerator(slo model.PromSLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error) {
	const (
		sliExprTplFmt = `(
{{.numeratorCorrectionMetric}}{{.window}}{{.filter}}
* on()
%s
)
/
(%s)
`
		sliExprTotalWindowTplFmt = `(%s)
/
(%s)
`
		// For more information about optimized SLI check `sloth.dev/core/sli_rules/v1` plugin.
		sliExprTotalWindowOptimizedTplFmt = `sum_over_time(%s[{{.window}}])
/ ignoring ({{.windowKey}})
count_over_time(%s[{{.window}}])
`
	)

	sliExprTpl := ""
	switch {
	// Last window (total window) when not optimized.
	case window == slo.TimeWindow && p.cfg.DisableOptimized:
		sliExprTpl = fmt.Sprintf(sliExprTotalWindowTplFmt, slo.SLI.Events.ErrorQuery, slo.SLI.Events.TotalQuery)

	// Last window (total window) when optimized.
	case window == slo.TimeWindow && !p.cfg.DisableOptimized:
		shortWindowSLIRec := conventions.GetSLIErrorMetric(alerts.PageQuick.ShortWindow)
		filter := promutils.LabelsToPromFilter(conventions.GetSLOIDPromLabels(slo))
		metric := shortWindowSLIRec + filter
		sliExprTpl = fmt.Sprintf(sliExprTotalWindowOptimizedTplFmt, metric, metric)

	// Regular SLI.
	default:
		sliExprTpl = fmt.Sprintf(sliExprTplFmt, slo.SLI.Events.ErrorQuery, slo.SLI.Events.TotalQuery)
	}
	// Render with our templated data.
	tpl, err := template.New("sliExpr").Option("missingkey=error").Parse(sliExprTpl)
	if err != nil {
		return nil, fmt.Errorf("could not create SLI expression template data: %w", err)
	}

	strWindow := promutils.TimeDurationToPromStr(window)

	var b bytes.Buffer
	err = tpl.Execute(&b, map[string]string{
		"numeratorCorrectionMetric": numeratorCorrectionMetric,
		"window":                    strWindow,
		"filter":                    promutils.LabelsToPromFilter(conventions.GetSLOIDPromLabels(slo)),
		"windowKey":                 conventions.PromSLOWindowLabelName,
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

func createNumeratorCorrection(slo model.PromSLO, labels map[string]string, currentWindow, totalWindow time.Duration) (*rulefmt.Rule, error) {
	windowString := promutils.TimeDurationToPromStr(currentWindow)
	metricSLONumeratorCorrection := numeratorCorrectionMetric + windowString

	tpl, err := template.New("sliExpr").Option("missingkey=error").Parse(slo.SLI.Events.TotalQuery)
	if err != nil {
		return nil, fmt.Errorf("could not create %s expression template data: %w", metricSLONumeratorCorrection, err)
	}

	var numeratorBuffer bytes.Buffer
	err = tpl.Execute(&numeratorBuffer, map[string]string{
		conventions.TplSLIQueryWindowVarName: windowString,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create numerator for %s: %w", metricSLONumeratorCorrection, err)
	}

	denominatorWindow := promutils.TimeDurationToPromStr(totalWindow)
	var denominatorBuffer bytes.Buffer
	err = tpl.Execute(&denominatorBuffer, map[string]string{
		conventions.TplSLIQueryWindowVarName: denominatorWindow,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create denominator for %s: %w", metricSLONumeratorCorrection, err)
	}

	return &rulefmt.Rule{
		Record: metricSLONumeratorCorrection,
		Expr:   fmt.Sprintf(`(%s)/(%s)`, numeratorBuffer.String(), denominatorBuffer.String()),
		Labels: labels,
	}, nil
}
