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
	PluginID      = "sloth.dev/core/sli_rules/v1"
)

type PluginConfig struct {
	Optimized bool
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
	genFunc := factorySLIRecordGenerator
	if p.cfg.Optimized {
		genFunc = optimizedFactorySLIRecordGenerator
	}

	sliRules, err := generateSLIRecordingRules(ctx, request.SLO, request.MWMBAlertGroup, genFunc)
	if err != nil {
		return err
	}
	result.SLORules.SLIErrorRecRules.Rules = sliRules

	return nil
}

func generateSLIRecordingRules(ctx context.Context, slo model.PromSLO, alerts model.MWMBAlertGroup, genFunc sliRulesgenFunc) ([]rulefmt.Rule, error) {
	// Get the windows we need the recording rules.
	windows := getAlertGroupWindows(alerts)
	windows = append(windows, slo.TimeWindow) // Add the total time window as a handy helper.

	// Generate the rules
	rules := make([]rulefmt.Rule, 0, len(windows))
	for _, window := range windows {
		rule, err := genFunc(slo, window, alerts)
		if err != nil {
			return nil, fmt.Errorf("could not create %q SLO rule for window %s: %w", slo.ID, window, err)
		}
		rules = append(rules, *rule)
	}

	return rules, nil
}

// sliRulesgenFunc knows how to generate an SLI recording rule for a specific time window.
type sliRulesgenFunc func(slo model.PromSLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error)

// OptimizedSLIRecordingRulesGenerator knows how to generate the SLI prometheus recording rules
// from an SLO optimizing where it can.
// Normally these rules are used by the SLO alerts.
func optimizedFactorySLIRecordGenerator(slo model.PromSLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error) {
	// Optimize the rules that are for the total period time window.
	if window == slo.TimeWindow {
		return optimizedSLIRecordGenerator(slo, window, alerts.PageQuick.ShortWindow)
	}

	return factorySLIRecordGenerator(slo, window, alerts)
}

const (
	tplKeyWindow = "window"
)

// factorySLIRecordGenerator knows how to generate the SLI prometheus recording rules
// form an SLO.
// Normally these rules are used by the SLO alerts.
func factorySLIRecordGenerator(slo model.PromSLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error) {
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

func rawSLIRecordGenerator(slo model.PromSLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error) {
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

func eventsSLIRecordGenerator(slo model.PromSLO, window time.Duration, alerts model.MWMBAlertGroup) (*rulefmt.Rule, error) {
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
func optimizedSLIRecordGenerator(slo model.PromSLO, window, shortWindow time.Duration) (*rulefmt.Rule, error) {
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
