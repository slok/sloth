package io

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"k8s.io/apimachinery/pkg/util/yaml"

	pluginenginesli "github.com/slok/sloth/internal/pluginengine/sli"
	"github.com/slok/sloth/pkg/common/model"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
	prometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
	prometheuspluginv1 "github.com/slok/sloth/pkg/prometheus/plugin/v1"
)

type SLIPluginRepo interface {
	GetSLIPlugin(ctx context.Context, id string) (*pluginenginesli.SLIPlugin, error)
}

// SlothPrometheusYAMLSpecLoader knows how to load sloth prometheus YAML specs and converts them to a model.
type SlothPrometheusYAMLSpecLoader struct {
	windowPeriod time.Duration
	pluginsRepo  SLIPluginRepo
}

// NewSlothPrometheusYAMLSpecLoader returns a YAML spec loader.
func NewSlothPrometheusYAMLSpecLoader(pluginsRepo SLIPluginRepo, windowPeriod time.Duration) SlothPrometheusYAMLSpecLoader {
	return SlothPrometheusYAMLSpecLoader{
		windowPeriod: windowPeriod,
		pluginsRepo:  pluginsRepo,
	}
}

var slothPromSpecTypeV1Regex = regexp.MustCompile(`(?m)^version: +['"]?prometheus/v1['"]?\r?\n? *$`)

func (l SlothPrometheusYAMLSpecLoader) IsSpecType(ctx context.Context, data []byte) bool {
	return slothPromSpecTypeV1Regex.Match(data)
}

func (l SlothPrometheusYAMLSpecLoader) LoadSpec(ctx context.Context, data []byte) (*model.PromSLOGroup, error) {
	s, err := l.LoadAPI(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("could not load API: %w", err)
	}

	m, err := l.MapSpecToModel(ctx, *s)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	return m, nil
}

func (l SlothPrometheusYAMLSpecLoader) LoadAPI(ctx context.Context, data []byte) (*prometheusv1.Spec, error) {
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

	return &s, nil
}

func (l SlothPrometheusYAMLSpecLoader) MapSpecToModel(ctx context.Context, spec prometheusv1.Spec) (*model.PromSLOGroup, error) {
	models := make([]model.PromSLO, 0, len(spec.SLOs))

	// Get group plugins if any.
	var groupSLOPlugins []model.PromSLOPluginMetadata
	groupOverridePlugins := spec.SLOPlugins.OverridePrevious
	for _, plugin := range spec.SLOPlugins.Chain {
		groupSLOPlugins = append(groupSLOPlugins, model.PromSLOPluginMetadata{
			ID:       plugin.ID,
			Config:   plugin.Config,
			Priority: plugin.Priority,
		})
	}

	for _, specSLO := range spec.SLOs {
		plugins := append([]model.PromSLOPluginMetadata{}, groupSLOPlugins...) // Add group plugins if any.

		// If we need to override the previous plugins at SLO level we need to remove the group plugins.
		overridePlugins := groupOverridePlugins
		if specSLO.Plugins.OverridePrevious {
			plugins = []model.PromSLOPluginMetadata{}
			overridePlugins = true
		}

		for _, plugin := range specSLO.Plugins.Chain {
			plugins = append(plugins, model.PromSLOPluginMetadata{
				ID:       plugin.ID,
				Config:   plugin.Config,
				Priority: plugin.Priority,
			})
		}

		slo := model.PromSLO{
			ID:              fmt.Sprintf("%s-%s", spec.Service, specSLO.Name),
			Name:            specSLO.Name,
			Description:     specSLO.Description,
			Service:         spec.Service,
			TimeWindow:      l.windowPeriod,
			Objective:       specSLO.Objective,
			Labels:          utilsdata.MergeLabels(spec.Labels, specSLO.Labels),
			PageAlertMeta:   model.PromAlertMeta{Disable: true},
			TicketAlertMeta: model.PromAlertMeta{Disable: true},
			Plugins: model.SLOPlugins{
				OverridePlugins: overridePlugins,
				Plugins:         plugins,
			},
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
			plugin, err := l.pluginsRepo.GetSLIPlugin(ctx, specSLO.SLI.Plugin.ID)
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

		models = append(models, slo)
	}

	return &model.PromSLOGroup{
		SLOs:           models,
		OriginalSource: model.PromSLOGroupSource{SlothV1: &spec},
	}, nil
}
