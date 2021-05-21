package prometheus

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"

	prometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
)

type yamlSpecLoader bool

// YAMLSpecLoader knows how to load YAML specs and converts them to a model.
const YAMLSpecLoader = yamlSpecLoader(false)

func (y yamlSpecLoader) LoadSpec(ctx context.Context, data []byte) (*SLOGroup, error) {
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

	m, err := y.mapSpecToModel(s)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	return m, nil
}

func (yamlSpecLoader) mapSpecToModel(spec prometheusv1.Spec) (*SLOGroup, error) {
	models := make([]SLO, 0, len(spec.SLOs))
	for _, specSLO := range spec.SLOs {
		slo := SLO{
			ID:               fmt.Sprintf("%s-%s", spec.Service, specSLO.Name),
			Name:             specSLO.Name,
			Description:      specSLO.Description,
			Service:          spec.Service,
			TimeWindow:       30 * 24 * time.Hour, // Default and for now the only one supported.
			Objective:        specSLO.Objective,
			Labels:           mergeLabels(spec.Labels, specSLO.Labels),
			PageAlertMeta:    AlertMeta{Disable: true},
			WarningAlertMeta: AlertMeta{Disable: true},
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

		// Set alerts.
		if !specSLO.Alerting.PageAlert.Disable {
			slo.PageAlertMeta = AlertMeta{
				Name:        specSLO.Alerting.Name,
				Labels:      mergeLabels(specSLO.Alerting.Labels, specSLO.Alerting.PageAlert.Labels),
				Annotations: mergeLabels(specSLO.Alerting.Annotations, specSLO.Alerting.PageAlert.Annotations),
			}
		}

		if !specSLO.Alerting.TicketAlert.Disable {
			slo.WarningAlertMeta = AlertMeta{
				Name:        specSLO.Alerting.Name,
				Labels:      mergeLabels(specSLO.Alerting.Labels, specSLO.Alerting.TicketAlert.Labels),
				Annotations: mergeLabels(specSLO.Alerting.Annotations, specSLO.Alerting.TicketAlert.Annotations),
			}
		}

		models = append(models, slo)
	}

	return &SLOGroup{SLOs: models}, nil
}
