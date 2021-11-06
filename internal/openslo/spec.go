package openslo

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	openslov1alpha "github.com/OpenSLO/oslo/pkg/manifest/v1alpha"
	"gopkg.in/yaml.v2"

	"github.com/slok/sloth/internal/prometheus"
)

type YAMLSpecLoader struct {
	windowPeriod time.Duration
}

// YAMLSpecLoader knows how to load YAML specs and converts them to a model.
func NewYAMLSpecLoader(windowPeriod time.Duration) YAMLSpecLoader {
	return YAMLSpecLoader{
		windowPeriod: windowPeriod,
	}
}

var (
	specTypeV1AlphaRegexKind       = regexp.MustCompile(`(?m)^kind: +['"]?SLO['"]? *$`)
	specTypeV1AlphaRegexAPIVersion = regexp.MustCompile(`(?m)^apiVersion: +['"]?openslo\/v1alpha['"]? *$`)
)

func (y YAMLSpecLoader) IsSpecType(ctx context.Context, data []byte) bool {
	return specTypeV1AlphaRegexKind.Match(data) && specTypeV1AlphaRegexAPIVersion.Match(data)
}

func (y YAMLSpecLoader) LoadSpec(ctx context.Context, data []byte) (*prometheus.SLOGroup, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("spec is required")
	}

	s := openslov1alpha.SLO{}
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall YAML spec correctly: %w", err)
	}

	// Check version.
	if s.APIVersion != openslov1alpha.APIVersion {
		return nil, fmt.Errorf("invalid spec version, should be %q", openslov1alpha.APIVersion)
	}

	// Check at least we have one SLO.
	if len(s.Spec.Objectives) == 0 {
		return nil, fmt.Errorf("at least one SLO is required")
	}

	// Validate time windows are correct.
	err = y.validateTimeWindow(s)
	if err != nil {
		return nil, fmt.Errorf("invalid SLO time windows: %w", err)
	}

	m, err := y.mapSpecToModel(s)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	return m, nil
}

func (y YAMLSpecLoader) mapSpecToModel(spec openslov1alpha.SLO) (*prometheus.SLOGroup, error) {
	slos, err := y.getSLOs(spec)
	if err != nil {
		return nil, fmt.Errorf("could not map SLOs correctly: %w", err)
	}

	return &prometheus.SLOGroup{SLOs: slos}, nil
}

// validateTimeWindow will validate that Sloth only supports 30 day based time windows
// we need this because time windows are a required by OpenSLO.
func (YAMLSpecLoader) validateTimeWindow(spec openslov1alpha.SLO) error {
	if len(spec.Spec.TimeWindows) == 0 {
		return nil
	}

	if len(spec.Spec.TimeWindows) > 1 {
		return fmt.Errorf("only 1 time window is supported")
	}

	t := spec.Spec.TimeWindows[0]
	if strings.ToLower(t.Unit) != "day" {
		return fmt.Errorf("only days based time windows are supported")
	}

	return nil
}

var errorRatioRawQueryTpl = template.Must(template.New("").Parse(`
  1 - (
    (
      {{ .good }}
    )
    /
    (
      {{ .total }}
    )
  )
`))

// getSLI gets the SLI from the OpenSLO slo objective, we only support ratio based openSLO objectives,
// however we will convert to a raw based sloth SLI because the ratio queries that we have differ from
// Sloth. Sloth uses bad/total events, OpenSLO uses good/total events. We get the ratio using good events
// and then rest to 1, to get a raw error ratio query.
func (y YAMLSpecLoader) getSLI(spec openslov1alpha.SLOSpec, slo openslov1alpha.Objective) (*prometheus.SLI, error) {
	if slo.RatioMetrics == nil {
		return nil, fmt.Errorf("missing ratioMetrics")
	}

	good := slo.RatioMetrics.Good
	total := slo.RatioMetrics.Total

	if good.Source != "prometheus" && good.Source != "sloth" {
		return nil, fmt.Errorf("prometheus or sloth query ratio 'good' source is required")
	}

	if total.Source != "prometheus" && good.Source != "sloth" {
		return nil, fmt.Errorf("prometheus or sloth query ratio 'total' source is required")
	}

	if good.QueryType != "promql" {
		return nil, fmt.Errorf("unsupported 'good' indicator query type: %s", good.QueryType)
	}

	if total.QueryType != "promql" {
		return nil, fmt.Errorf("unsupported 'total' indicator query type: %s", total.QueryType)
	}

	// Map as good and total events as a raw query.
	var b bytes.Buffer
	err := errorRatioRawQueryTpl.Execute(&b, map[string]string{"good": good.Query, "total": total.Query})
	if err != nil {
		return nil, fmt.Errorf("could not execute mapping SLI template: %w", err)
	}

	return &prometheus.SLI{Raw: &prometheus.SLIRaw{
		ErrorRatioQuery: b.String(),
	}}, nil
}

// getSLOs will try getting all the objectives as individual SLOs, this way we can map
// to what Sloth understands as an SLO, that OpenSLO understands as a list of objectives
// for the same SLO.
func (y YAMLSpecLoader) getSLOs(spec openslov1alpha.SLO) ([]prometheus.SLO, error) {
	res := []prometheus.SLO{}

	for idx, slo := range spec.Spec.Objectives {
		sli, err := y.getSLI(spec.Spec, slo)
		if err != nil {
			return nil, fmt.Errorf("could not map SLI: %w", err)
		}

		timeWindow := y.windowPeriod
		if len(spec.Spec.TimeWindows) > 0 {
			timeWindow = time.Duration(spec.Spec.TimeWindows[0].Count) * 24 * time.Hour
		}

		// TODO(slok): Think about using `slo.Value` insted of idx (`slo.Value` is not mandatory).
		res = append(res, prometheus.SLO{
			ID:              fmt.Sprintf("%s-%s-%d", spec.Spec.Service, spec.Metadata.Name, idx),
			Name:            fmt.Sprintf("%s-%d", spec.Metadata.Name, idx),
			Service:         spec.Spec.Service,
			Description:     spec.Spec.Description,
			TimeWindow:      timeWindow,
			SLI:             *sli,
			Objective:       *slo.BudgetTarget * 100, // OpenSLO uses ratios, we use percents.
			PageAlertMeta:   prometheus.AlertMeta{Disable: true},
			TicketAlertMeta: prometheus.AlertMeta{Disable: true},
		})
	}

	return res, nil
}
