package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	// Check requirements for this type of SLOs.
	if request.SLO.SLI.Events == nil || request.SLO.SLI.Events.ErrorQuery == "" || request.SLO.SLI.Events.TotalQuery == "" {
		return fmt.Errorf("denominator corrected SLI requires SLI event type")
	}

	// Generate and override SLI recordings.
	sliRules, err := generateSLIRecordingRules(ctx, request.SLO, request.MWMBAlertGroup)
	if err != nil {
		return err
	}
	result.SLORules.SLIErrorRecRules.Rules = sliRules

	// Add required new metadata recordings with the correction factor.
	windows := getAlertGroupWindows(request.MWMBAlertGroup)
	windows = append(windows, request.SLO.TimeWindow) // Add the total time window as a handy helper.
	metadataLabels := utilsdata.MergeLabels(conventions.GetSLOIDPromLabels(request.SLO), request.SLO.Labels)
	for _, window := range windows {
		rule, err := createNumeratorCorrection(request.SLO, metadataLabels, window, request.SLO.TimeWindow)
		if err != nil {
			return fmt.Errorf("could not create numerator rule: %v", err)
		}
		result.SLORules.MetadataRecRules.Rules = append(result.SLORules.MetadataRecRules.Rules, *rule)
	}

	return nil
}

func generateSLIRecordingRules(ctx context.Context, slo model.PromSLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error) {
	// Get the windows we need the recording rules.
	windows := getAlertGroupWindows(alerts)
	windows = append(windows, slo.TimeWindow) // Add the total time window as a handy helper.

	// Generate the rules
	rules := make([]rulefmt.Rule, 0, len(windows))
	for _, window := range windows {
		rule, err := denominatorCorrectedSLIRecordGenerator(slo, window, alerts)
		if err != nil {
			return nil, fmt.Errorf("could not create %q SLO rule for window %s: %w", slo.ID, window, err)
		}
		rules = append(rules, *rule)
	}

	return rules, nil
}

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

func denominatorCorrectedSLIRecordGenerator(slo model.PromSLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error) {
	const sliExprTplFmt = `(
slo:numerator_correction:ratio{{.window}}{{.filter}}
* on()
%s
)
/
(%s)
`

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
		"filter":     promutils.LabelsToPromFilter(conventions.GetSLOIDPromLabels(slo)),
		"windowKey":  conventions.PromSLOWindowLabelName,
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

const (
	tplKeyWindow = "window"
)

func createNumeratorCorrection(slo model.PromSLO, labels map[string]string, currentWindow, totalWindow time.Duration) (*rulefmt.Rule, error) {
	windowString := promutils.TimeDurationToPromStr(currentWindow)
	metricSLONumeratorCorrection := fmt.Sprintf("slo:numerator_correction:ratio%s", windowString)

	tpl, err := template.New("sliExpr").Option("missingkey=error").Parse(slo.SLI.Events.TotalQuery)
	if err != nil {
		return nil, fmt.Errorf("could not create %s expression template data: %w", metricSLONumeratorCorrection, err)
	}

	var numeratorBuffer bytes.Buffer
	err = tpl.Execute(&numeratorBuffer, map[string]string{
		tplKeyWindow: windowString,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create numerator for %s: %w", metricSLONumeratorCorrection, err)
	}

	denominatorWindow := promutils.TimeDurationToPromStr(totalWindow)
	var denominatorBuffer bytes.Buffer
	err = tpl.Execute(&denominatorBuffer, map[string]string{
		tplKeyWindow: denominatorWindow,
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
