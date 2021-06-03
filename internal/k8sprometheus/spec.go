package k8sprometheus

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/sloth/internal/prometheus"
	k8sprometheusv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	"github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned/scheme"
	prometheuspluginv1 "github.com/slok/sloth/pkg/prometheus/plugin/v1"
)

// YAMLSpecLoader knows how to load Kubernetes ServiceLevel YAML specs and converts them to a model.
type YAMLSpecLoader struct {
	plugins map[string]prometheus.SLIPlugin
	decoder runtime.Decoder
}

// NewYAMLSpecLoader returns a YAML spec loader.
func NewYAMLSpecLoader(plugins map[string]prometheus.SLIPlugin) YAMLSpecLoader {
	return YAMLSpecLoader{
		plugins: plugins,
		decoder: scheme.Codecs.UniversalDeserializer(),
	}
}

func (y YAMLSpecLoader) LoadSpec(ctx context.Context, data []byte) (*SLOGroup, error) {
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

	m, err := mapSpecToModel(ctx, y.plugins, kslo)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	return m, nil
}

type CRSpecLoader struct {
	plugins map[string]prometheus.SLIPlugin
}

// CRSpecLoader knows how to load Kubernetes CRD specs and converts them to a model.

func NewCRSpecLoader(plugins map[string]prometheus.SLIPlugin) CRSpecLoader {
	return CRSpecLoader{
		plugins: plugins,
	}
}

func (c CRSpecLoader) LoadSpec(ctx context.Context, spec *k8sprometheusv1.PrometheusServiceLevel) (*SLOGroup, error) {
	return mapSpecToModel(ctx, c.plugins, spec)
}

func mapSpecToModel(ctx context.Context, plugins map[string]prometheus.SLIPlugin, kspec *k8sprometheusv1.PrometheusServiceLevel) (*SLOGroup, error) {
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

		if specSLO.SLI.Plugin != nil {
			plugin, ok := plugins[specSLO.SLI.Plugin.ID]
			if !ok {
				return nil, fmt.Errorf("unknown plugin: %q", specSLO.SLI.Plugin.ID)
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

			slo.SLI.Raw = &prometheus.SLIRaw{
				ErrorRatioQuery: rawQuery,
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
