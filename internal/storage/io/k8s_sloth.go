package io

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/sloth/pkg/common/model"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
	k8sprometheusv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	"github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned/scheme"
	prometheuspluginv1 "github.com/slok/sloth/pkg/prometheus/plugin/v1"
)

type K8sSlothPrometheusCRSpecLoader struct {
	windowPeriod time.Duration
	pluginsRepo  SLIPluginRepo
}

// NewK8sSlothPrometheusCRSpecLoader knows how to load Kubernetes CRD specs and converts them to a model.
func NewK8sSlothPrometheusCRSpecLoader(pluginsRepo SLIPluginRepo, windowPeriod time.Duration) K8sSlothPrometheusCRSpecLoader {
	return K8sSlothPrometheusCRSpecLoader{
		windowPeriod: windowPeriod,
		pluginsRepo:  pluginsRepo,
	}
}

func (c K8sSlothPrometheusCRSpecLoader) LoadSpec(ctx context.Context, spec *k8sprometheusv1.PrometheusServiceLevel) (*model.PromSLOGroup, error) {
	return mapSpecToModel(ctx, c.windowPeriod, c.pluginsRepo, spec)
}

// K8sSlothPrometheusYAMLSpecLoader knows how to load Kubernetes ServiceLevel YAML specs and converts them to a model.
type K8sSlothPrometheusYAMLSpecLoader struct {
	windowPeriod time.Duration
	pluginsRepo  SLIPluginRepo
	decoder      runtime.Decoder
}

// NewK8sSlothPrometheusYAMLSpecLoader returns a YAML spec loader.
func NewK8sSlothPrometheusYAMLSpecLoader(pluginsRepo SLIPluginRepo, windowPeriod time.Duration) K8sSlothPrometheusYAMLSpecLoader {
	return K8sSlothPrometheusYAMLSpecLoader{
		windowPeriod: windowPeriod,
		pluginsRepo:  pluginsRepo,
		decoder:      scheme.Codecs.UniversalDeserializer(),
	}
}

var (
	k8sSlothPromSpecTypeV1RegexKind       = regexp.MustCompile(`(?m)^kind: +['"]?PrometheusServiceLevel['"]? *$`)
	k8sSlothPromSpecTypeV1RegexAPIVersion = regexp.MustCompile(`(?m)^apiVersion: +['"]?sloth.slok.dev\/v1['"]? *$`)
)

func (l K8sSlothPrometheusYAMLSpecLoader) IsSpecType(ctx context.Context, data []byte) bool {
	return k8sSlothPromSpecTypeV1RegexKind.Match(data) && k8sSlothPromSpecTypeV1RegexAPIVersion.Match(data)
}

func (l K8sSlothPrometheusYAMLSpecLoader) LoadSpec(ctx context.Context, data []byte) (*model.PromSLOGroup, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("spec is required")
	}

	obj, _, err := l.decoder.Decode([]byte(data), nil, nil)
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

	m, err := mapSpecToModel(ctx, l.windowPeriod, l.pluginsRepo, kslo)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	return m, nil
}

func mapSpecToModel(ctx context.Context, defaultWindowPeriod time.Duration, pluginsRepo SLIPluginRepo, kspec *k8sprometheusv1.PrometheusServiceLevel) (*model.PromSLOGroup, error) {
	slos := make([]model.PromSLO, 0, len(kspec.Spec.SLOs))
	spec := kspec.Spec
	for _, specSLO := range kspec.Spec.SLOs {
		slo := model.PromSLO{
			ID:              fmt.Sprintf("%s-%s", spec.Service, specSLO.Name),
			Name:            specSLO.Name,
			Description:     specSLO.Description,
			Service:         spec.Service,
			TimeWindow:      defaultWindowPeriod,
			Objective:       specSLO.Objective,
			Labels:          utilsdata.MergeLabels(spec.Labels, specSLO.Labels),
			PageAlertMeta:   model.PromAlertMeta{Disable: true},
			TicketAlertMeta: model.PromAlertMeta{Disable: true},
		}

		// Set SLIs.
		if specSLO.SLI.Events != nil {
			slo.SLI.Events = &model.PromSLIEvents{
				ErrorQuery: specSLO.SLI.Events.ErrorQuery,
				TotalQuery: specSLO.SLI.Events.TotalQuery,
			}
		}

		if specSLO.SLI.Raw != nil {
			slo.SLI.Raw = &model.PromSLIRaw{
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

			slo.SLI.Raw = &model.PromSLIRaw{
				ErrorRatioQuery: rawQuery,
			}
		}

		// Set alerts.
		if !specSLO.Alerting.PageAlert.Disable {
			slo.PageAlertMeta = model.PromAlertMeta{
				Name:        specSLO.Alerting.Name,
				Labels:      utilsdata.MergeLabels(specSLO.Alerting.Labels, specSLO.Alerting.PageAlert.Labels),
				Annotations: utilsdata.MergeLabels(specSLO.Alerting.Annotations, specSLO.Alerting.PageAlert.Annotations),
			}
		}

		if !specSLO.Alerting.TicketAlert.Disable {
			slo.TicketAlertMeta = model.PromAlertMeta{
				Name:        specSLO.Alerting.Name,
				Labels:      utilsdata.MergeLabels(specSLO.Alerting.Labels, specSLO.Alerting.TicketAlert.Labels),
				Annotations: utilsdata.MergeLabels(specSLO.Alerting.Annotations, specSLO.Alerting.TicketAlert.Annotations),
			}
		}

		slos = append(slos, slo)
	}

	res := &model.PromSLOGroup{
		SLOs: slos,
		OriginalSource: model.PromSLOGroupSource{
			K8sSlothV1: kspec,
		},
	}

	return res, nil
}
