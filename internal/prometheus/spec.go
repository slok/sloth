package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/ghodss/yaml"
)

// SpecVersionV1 is the Prometheus V1 ID version.
const SpecVersionV1 = "prometheus/v1"

type alertSpec struct {
	Disable     bool              `json:"disable"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type specV1 struct {
	Version string            `json:"version"`
	Service string            `json:"service"`
	Labels  map[string]string `json:"labels"`
	SLOs    []struct {
		Name      string            `json:"name"`
		Objective float64           `json:"objective"`
		Labels    map[string]string `json:"labels"`
		SLI       struct {
			ErrorQuery string `json:"error_query"`
			TotalQuery string `json:"total_query"`
		} `json:"sli"`
		Alerting struct {
			Name        string            `json:"name" validate:"required"`
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
			PageAlert   *alertSpec        `json:"page_alert"`
			TicketAlert *alertSpec        `json:"ticket_alert"`
		} `json:"alerting,omitempty"`
	} `json:"slos"`
}

type yamlSpecLoader bool

// YAMLSpecLoader knows how to load YAML specs and converts them to a model.
const YAMLSpecLoader = yamlSpecLoader(false)

func (y yamlSpecLoader) LoadSpec(ctx context.Context, data []byte) ([]SLO, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("spec is required")
	}

	s := specV1{}
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall YAML spec correctly: %w", err)
	}

	// Check version.
	if s.Version != SpecVersionV1 {
		return nil, fmt.Errorf("invalid spec version, should be %q", SpecVersionV1)
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

func (yamlSpecLoader) mapSpecToModel(spec specV1) ([]SLO, error) {
	models := make([]SLO, 0, len(spec.SLOs))
	for _, specSLO := range spec.SLOs {
		slo := SLO{
			ID:         fmt.Sprintf("%s-%s", spec.Service, specSLO.Name),
			Name:       specSLO.Name,
			Service:    spec.Service,
			TimeWindow: 30 * 24 * time.Hour, // Default and for now the only one supported.
			SLI: CustomSLI{
				ErrorQuery: specSLO.SLI.ErrorQuery,
				TotalQuery: specSLO.SLI.TotalQuery,
			},
			Objective:        specSLO.Objective,
			Labels:           mergeLabels(spec.Labels, specSLO.Labels),
			PageAlertMeta:    AlertMeta{Disable: true},
			WarningAlertMeta: AlertMeta{Disable: true},
		}

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

	return models, nil
}
