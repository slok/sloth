package prometheus

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"gopkg.in/yaml.v2"

	prometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
	prometheuspluginv1 "github.com/slok/sloth/pkg/prometheus/plugin/v1"
)

type SLIPluginRepo interface {
	GetSLIPlugin(ctx context.Context, id string) (*SLIPlugin, error)
}

// YAMLSpecLoader knows how to load YAML specs and converts them to a model.
type YAMLSpecLoader struct {
	windowPeriod time.Duration
	pluginsRepo  SLIPluginRepo
}

// NewYAMLSpecLoader returns a YAML spec loader.
func NewYAMLSpecLoader(pluginsRepo SLIPluginRepo, windowPeriod time.Duration) YAMLSpecLoader {
	return YAMLSpecLoader{
		windowPeriod: windowPeriod,
		pluginsRepo:  pluginsRepo,
	}
}

var specTypeV1Regex = regexp.MustCompile(`(?m)^version: +['"]?prometheus\/v1['"]? *$`)

func (y YAMLSpecLoader) IsSpecType(ctx context.Context, data []byte) bool {
	return specTypeV1Regex.Match(data)
}

func (y YAMLSpecLoader) LoadSpec(ctx context.Context, data []byte) (*SLOGroup, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("spec is required")
	}

	s := prometheusv1.Spec{}
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall YAML spec correctly: %w", err)
	}

	// Check version.
	if s.Version != prometheusv1.Version {
		return nil, fmt.Errorf("invalid spec version, should be %q", prometheusv1.Version)
	}

	// Check at least we have one SLO.
	if len(s.SLOs) == 0 {
		return nil, fmt.Errorf("at least one SLO is required")
	}

	m, err := y.mapSpecToModel(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	return m, nil
}

func (y YAMLSpecLoader) mapSpecToModel(ctx context.Context, spec prometheusv1.Spec) (*SLOGroup, error) {
	models := make([]SLO, 0, len(spec.SLOs))
	for _, specSLO := range spec.SLOs {
		slo := SLO{
			ID:                    fmt.Sprintf("%s-%s", spec.Service, specSLO.Name),
			RuleGroupInterval:     specSLO.Interval.RuleGroupInterval,
			SLIErrorRulesInterval: specSLO.Interval.SLIErrorRulesInterval,
			MetadataRulesInterval: specSLO.Interval.MetadataRulesInterval,
			AlertRulesInterval:    specSLO.Interval.AlertRulesInterval,
			Name:                  specSLO.Name,
			Description:           specSLO.Description,
			Service:               spec.Service,
			TimeWindow:            y.windowPeriod,
			Objective:             specSLO.Objective,
			Labels:                mergeLabels(spec.Labels, specSLO.Labels),
			PageAlertMeta:         AlertMeta{Disable: true},
			TicketAlertMeta:       AlertMeta{Disable: true},
		}

		// Set SLIs.
		if specSLO.SLI.Events != nil {
			slo.SLI.Events = &SLIEvents{
				ErrorQuery: specSLO.SLI.Events.ErrorQuery,
				TotalQuery: specSLO.SLI.Events.TotalQuery,
			}
		}

		if specSLO.SLI.Raw != nil {
			slo.SLI.Raw = &SLIRaw{
				ErrorRatioQuery: specSLO.SLI.Raw.ErrorRatioQuery,
			}
		}

		if specSLO.SLI.Plugin != nil {
			plugin, err := y.pluginsRepo.GetSLIPlugin(ctx, specSLO.SLI.Plugin.ID)
			if err != nil {
				return nil, fmt.Errorf("could not get plugin: %w", err)
			}

			meta := map[string]string{
				prometheuspluginv1.SLIPluginMetaService:   spec.Service,
				prometheuspluginv1.SLIPluginMetaSLO:       specSLO.Name,
				prometheuspluginv1.SLIPluginMetaObjective: fmt.Sprintf("%f", specSLO.Objective),
			}

			rawQuery, err := plugin.Func(ctx, meta, spec.Labels, specSLO.SLI.Plugin.Options)
			if err != nil {
				return nil, fmt.Errorf("plugin %q execution error: %w", specSLO.SLI.Plugin.ID, err)
			}

			slo.SLI.Raw = &SLIRaw{
				ErrorRatioQuery: rawQuery,
			}
		}

		// Set alerts.
		if !specSLO.Alerting.PageAlert.Disable {
			slo.PageAlertMeta = AlertMeta{
				Name:        specSLO.Alerting.Name,
				Labels:      mergeLabels(specSLO.Alerting.Labels, specSLO.Alerting.PageAlert.Labels),
				Annotations: mergeLabels(specSLO.Alerting.Annotations, specSLO.Alerting.PageAlert.Annotations),
			}
		}

		if !specSLO.Alerting.TicketAlert.Disable {
			slo.TicketAlertMeta = AlertMeta{
				Name:        specSLO.Alerting.Name,
				Labels:      mergeLabels(specSLO.Alerting.Labels, specSLO.Alerting.TicketAlert.Labels),
				Annotations: mergeLabels(specSLO.Alerting.Annotations, specSLO.Alerting.TicketAlert.Annotations),
			}
		}

		models = append(models, slo)
	}

	return &SLOGroup{SLOs: models}, nil
}
