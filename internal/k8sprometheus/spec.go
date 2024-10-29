package k8sprometheus

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/sloth/internal/prometheus"
	k8sprometheusv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	"github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned/scheme"
	prometheuspluginv1 "github.com/slok/sloth/pkg/prometheus/plugin/v1"
)

type SLIPluginRepo interface {
	GetSLIPlugin(ctx context.Context, id string) (*prometheus.SLIPlugin, error)
}

// YAMLSpecLoader knows how to load Kubernetes ServiceLevel YAML specs and converts them to a model.
type YAMLSpecLoader struct {
	windowPeriod time.Duration
	pluginsRepo  SLIPluginRepo
	decoder      runtime.Decoder
}

// NewYAMLSpecLoader returns a YAML spec loader.
func NewYAMLSpecLoader(pluginsRepo SLIPluginRepo, windowPeriod time.Duration) YAMLSpecLoader {
	return YAMLSpecLoader{
		windowPeriod: windowPeriod,
		pluginsRepo:  pluginsRepo,
		decoder:      scheme.Codecs.UniversalDeserializer(),
	}
}

var (
	specTypeV1RegexKind       = regexp.MustCompile(`(?m)^kind: +['"]?PrometheusServiceLevel['"]? *$`)
	specTypeV1RegexAPIVersion = regexp.MustCompile(`(?m)^apiVersion: +['"]?sloth.slok.dev\/v1['"]? *$`)
)

func (y YAMLSpecLoader) IsSpecType(ctx context.Context, data []byte) bool {
	return specTypeV1RegexKind.Match(data) && specTypeV1RegexAPIVersion.Match(data)
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

	m, err := mapSpecToModel(ctx, y.windowPeriod, y.pluginsRepo, kslo)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	return m, nil
}

type CRSpecLoader struct {
	windowPeriod time.Duration
	pluginsRepo  SLIPluginRepo
}

// CRSpecLoader knows how to load Kubernetes CRD specs and converts them to a model.

func NewCRSpecLoader(pluginsRepo SLIPluginRepo, windowPeriod time.Duration) CRSpecLoader {
	return CRSpecLoader{
		windowPeriod: windowPeriod,
		pluginsRepo:  pluginsRepo,
	}
}

func (c CRSpecLoader) LoadSpec(ctx context.Context, spec *k8sprometheusv1.PrometheusServiceLevel) (*SLOGroup, error) {
	return mapSpecToModel(ctx, c.windowPeriod, c.pluginsRepo, spec)
}

func mapSpecToModel(ctx context.Context, defaultWindowPeriod time.Duration, pluginsRepo SLIPluginRepo, kspec *k8sprometheusv1.PrometheusServiceLevel) (*SLOGroup, error) {
	slos := make([]prometheus.SLO, 0, len(kspec.Spec.SLOs))
	spec := kspec.Spec
	for _, specSLO := range kspec.Spec.SLOs {
		slo := prometheus.SLO{
			ID:              fmt.Sprintf("%s-%s", spec.Service, specSLO.Name),
			Name:            specSLO.Name,
			Description:     specSLO.Description,
			Service:         spec.Service,
			TimeWindow:      defaultWindowPeriod,
			Objective:       specSLO.Objective,
			Labels:          mergeLabels(spec.Labels, specSLO.Labels),
			InfoLabels:      specSLO.InfoLabels,
			PageAlertMeta:   prometheus.AlertMeta{Disable: true},
			TicketAlertMeta: prometheus.AlertMeta{Disable: true},
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
			plugin, err := pluginsRepo.GetSLIPlugin(ctx, specSLO.SLI.Plugin.ID)
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
			slo.TicketAlertMeta = prometheus.AlertMeta{
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
