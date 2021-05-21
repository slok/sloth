package k8sprometheus

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/sloth/internal/prometheus"
	k8sprometheusv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	"github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned/scheme"
)

type yamlSpecLoader struct {
	decoder runtime.Decoder
}

// YAMLSpecLoader knows how to load Kubernetes ServiceLevel YAML specs and converts them to a model.
var YAMLSpecLoader = yamlSpecLoader{
	decoder: scheme.Codecs.UniversalDeserializer(),
}

func (y yamlSpecLoader) LoadSpec(ctx context.Context, data []byte) (*SLOGroup, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("spec is required")
	}

	obj, _, err := y.decoder.Decode([]byte(data), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decode kubernetes object %w", err)
	}

	kslo, ok := obj.(*k8sprometheusv1.PrometheusServiceLevel)
	if !ok {
		return nil, fmt.Errorf("can't type assert runtime.Object to v1.PrometheusServiceLeve")
	}

	// Check at least we have one SLO.
	if len(kslo.Spec.SLOs) == 0 {
		return nil, fmt.Errorf("at least one SLO is required")
	}

	m, err := mapSpecToModel(kslo)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	return m, nil
}

type crSpecLoader bool

// CRSpecLoader knows how to load Kubernetes CRD specs and converts them to a model.
const CRSpecLoader = crSpecLoader(false)

func (c crSpecLoader) LoadSpec(ctx context.Context, spec *k8sprometheusv1.PrometheusServiceLevel) (*SLOGroup, error) {
	return mapSpecToModel(spec)
}

func mapSpecToModel(kspec *k8sprometheusv1.PrometheusServiceLevel) (*SLOGroup, error) {
	slos := make([]prometheus.SLO, 0, len(kspec.Spec.SLOs))
	spec := kspec.Spec
	for _, specSLO := range kspec.Spec.SLOs {
		slo := prometheus.SLO{
			ID:               fmt.Sprintf("%s-%s", spec.Service, specSLO.Name),
			Name:             specSLO.Name,
			Description:      specSLO.Description,
			Service:          spec.Service,
			TimeWindow:       30 * 24 * time.Hour, // Default and for now the only one supported.
			Objective:        specSLO.Objective,
			Labels:           mergeLabels(spec.Labels, specSLO.Labels),
			PageAlertMeta:    prometheus.AlertMeta{Disable: true},
			WarningAlertMeta: prometheus.AlertMeta{Disable: true},
		}

		// Set SLIs.
		if specSLO.SLI.Events != nil {
			slo.SLI.Events = &prometheus.SLIEvents{
				ErrorQuery: specSLO.SLI.Events.ErrorQuery,
				TotalQuery: specSLO.SLI.Events.TotalQuery,
			}
		}

		if specSLO.SLI.Raw != nil {
			slo.SLI.Raw = &prometheus.SLIRaw{
				ErrorRatioQuery: specSLO.SLI.Raw.ErrorRatioQuery,
			}
		}

		// Set alerts.
		if !specSLO.Alerting.PageAlert.Disable {
			slo.PageAlertMeta = prometheus.AlertMeta{
				Name:        specSLO.Alerting.Name,
				Labels:      mergeLabels(specSLO.Alerting.Labels, specSLO.Alerting.PageAlert.Labels),
				Annotations: mergeLabels(specSLO.Alerting.Annotations, specSLO.Alerting.PageAlert.Annotations),
			}
		}

		if !specSLO.Alerting.TicketAlert.Disable {
			slo.WarningAlertMeta = prometheus.AlertMeta{
				Name:        specSLO.Alerting.Name,
				Labels:      mergeLabels(specSLO.Alerting.Labels, specSLO.Alerting.TicketAlert.Labels),
				Annotations: mergeLabels(specSLO.Alerting.Annotations, specSLO.Alerting.TicketAlert.Annotations),
			}
		}

		slos = append(slos, slo)
	}

	res := &SLOGroup{
		K8sMeta: K8sMeta{
			Kind:        "PrometheusServiceLevel",
			APIVersion:  "sloth.slok.dev/v1",
			UID:         string(kspec.UID),
			Name:        kspec.Name,
			Namespace:   kspec.Namespace,
			Labels:      kspec.Labels,
			Annotations: kspec.Annotations,
		},
		SLOGroup: prometheus.SLOGroup{SLOs: slos},
	}

	return res, nil
}
