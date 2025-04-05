package generate

import (
	"context"
	"fmt"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/log"
	plugincorealertrulesv1 "github.com/slok/sloth/internal/plugin/slo/core/alert_rules_v1"
	plugincoremetadatarulesv1 "github.com/slok/sloth/internal/plugin/slo/core/metadata_rules_v1"
	plugincorenoopsv1 "github.com/slok/sloth/internal/plugin/slo/core/noop_v1"
	plugincoreslirulesv1 "github.com/slok/sloth/internal/plugin/slo/core/sli_rules_v1"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	commonerrors "github.com/slok/sloth/pkg/common/errors"
	"github.com/slok/sloth/pkg/common/model"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
)

// Default plugins.
var (
	NoopPlugin, _ = NewSLOProcessorFromSLOPluginV1(plugincorenoopsv1.NewPlugin, log.Noop, nil)
)

type noopSLOPluginGetter bool

func (noopSLOPluginGetter) GetSLOPlugin(ctx context.Context, id string) (*pluginengineslo.Plugin, error) {
	return nil, commonerrors.ErrNotFound
}

type SLOPluginGetter interface {
	GetSLOPlugin(ctx context.Context, id string) (*pluginengineslo.Plugin, error)
}

//go:generate mockery --case underscore --output generatemock --outpkg generatemock --name SLOPluginGetter

// ServiceConfig is the application service configuration.
type ServiceConfig struct {
	AlertGenerator            AlertGenerator
	SLIRulesGenSLOPlugin      SLOProcessor
	AlertRulesGenSLOPlugin    SLOProcessor
	MetadataRulesGenSLOPlugin SLOProcessor
	SLOPluginGetter           SLOPluginGetter
	Logger                    log.Logger
}

func (c *ServiceConfig) defaults() error {
	if c.AlertGenerator == nil {
		return fmt.Errorf("alert generator is required")
	}

	if c.SLOPluginGetter == nil {
		c.SLOPluginGetter = noopSLOPluginGetter(false)
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"svc": "generate.prometheus.Service"})

	// Default plugins.
	if c.SLIRulesGenSLOPlugin == nil {
		plugin, err := NewSLOProcessorFromSLOPluginV1(
			plugincoreslirulesv1.NewPlugin,
			c.Logger.WithValues(log.Kv{"plugin": plugincoreslirulesv1.PluginID}),
			plugincoreslirulesv1.PluginConfig{Optimized: true},
		)
		if err != nil {
			return fmt.Errorf("could not create SLI rules plugin: %w", err)
		}
		c.SLIRulesGenSLOPlugin = plugin
	}
	if c.AlertRulesGenSLOPlugin == nil {
		plugin, err := NewSLOProcessorFromSLOPluginV1(
			plugincorealertrulesv1.NewPlugin,
			c.Logger.WithValues(log.Kv{"plugin": plugincorealertrulesv1.PluginID}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("could not create alert rules plugin: %w", err)
		}
		c.AlertRulesGenSLOPlugin = plugin
	}
	if c.MetadataRulesGenSLOPlugin == nil {
		plugin, err := NewSLOProcessorFromSLOPluginV1(
			plugincoremetadatarulesv1.NewPlugin,
			c.Logger.WithValues(log.Kv{"plugin": plugincoremetadatarulesv1.PluginID}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("could not create metadata rules plugin: %w", err)
		}
		c.MetadataRulesGenSLOPlugin = plugin
	}

	return nil
}

// AlertGenerator knows how to generate multiwindow multi-burn SLO alerts.
type AlertGenerator interface {
	GenerateMWMBAlerts(ctx context.Context, slo alert.SLO) (*model.MWMBAlertGroup, error)
}

// Service is the application service for the generation of SLO for Prometheus.
type Service struct {
	alertGen        AlertGenerator
	sloPluginGetter SLOPluginGetter
	defaultPlugins  []SLOProcessor
	logger          log.Logger
}

// NewService returns a new Prometheus application service.
func NewService(config ServiceConfig) (*Service, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &Service{
		alertGen:        config.AlertGenerator,
		sloPluginGetter: config.SLOPluginGetter,
		defaultPlugins: []SLOProcessor{
			config.SLIRulesGenSLOPlugin,
			config.AlertRulesGenSLOPlugin,
			config.MetadataRulesGenSLOPlugin,
		},
		logger: config.Logger,
	}, nil
}

type Request struct {
	// Info about the application and execution, normally used as metadata.
	Info model.Info
	// ExtraLabels are the extra labels added to the SLOs on execution time.
	ExtraLabels map[string]string
	// SLOGroup are the SLOs group that will be used to generate the SLO results and Prom rules.
	SLOGroup model.PromSLOGroup
}

type SLOResult struct {
	SLO      model.PromSLO
	SLORules model.PromSLORules
}

type Response struct {
	PrometheusSLOs []SLOResult
}

func (s Service) Generate(ctx context.Context, r Request) (*Response, error) {
	err := r.SLOGroup.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid SLO group: %w", err)
	}

	// Generate Prom rules.
	results := make([]SLOResult, 0, len(r.SLOGroup.SLOs))
	for _, slo := range r.SLOGroup.SLOs {
		// Add extra labels.
		slo.Labels = utilsdata.MergeLabels(slo.Labels, r.ExtraLabels)

		// Generate SLO result.
		result, err := s.generateSLO(ctx, r.Info, slo)
		if err != nil {
			return nil, fmt.Errorf("could not generate %q slo: %w", slo.ID, err)
		}

		results = append(results, SLOResult{SLO: slo, SLORules: *result})
	}

	return &Response{
		PrometheusSLOs: results,
	}, nil
}

func (s Service) generateSLO(ctx context.Context, info model.Info, slo model.PromSLO) (*model.PromSLORules, error) {
	logger := s.logger.WithCtxValues(ctx).WithValues(log.Kv{"slo": slo.ID})

	// Generate the MWMB alerts.
	alertSLO := alert.SLO{
		ID:         slo.ID,
		Objective:  slo.Objective,
		TimeWindow: slo.TimeWindow,
	}
	as, err := s.alertGen.GenerateMWMBAlerts(ctx, alertSLO)
	if err != nil {
		return nil, fmt.Errorf("could not generate SLO alerts: %w", err)
	}
	logger.Debugf("Multiwindow-multiburn alerts generated")

	// Generate plugins. Add default plugins and then the ones of each of the SLO.
	sloProcessors := s.defaultPlugins
	for _, p := range slo.Plugins.Plugins {
		pf, err := s.sloPluginGetter.GetSLOPlugin(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("could not get SLO plugin %q: %w", p.ID, err)
		}
		var processor SLOProcessor
		switch {
		case pf.PluginV1Factory != nil:
			processor, err = NewSLOProcessorFromSLOPluginV1(pf.PluginV1Factory, logger.WithValues(log.Kv{"plugin": pf.ID}), p.Config)
			if err != nil {
				return nil, fmt.Errorf("could create SLO plugin %q: %w", p.ID, err)
			}
		}

		sloProcessors = append(sloProcessors, processor)
	}

	req := &SLOProcessorRequest{
		Info:           info,
		MWMBAlertGroup: *as,
		SLO:            slo,
	}
	res := &SLOProcessorResult{}
	for _, p := range sloProcessors {
		err := p.ProcessSLO(ctx, req, res)
		if err != nil {
			return nil, fmt.Errorf("slo processor failed: %w", err)
		}
	}

	return &res.SLORules, nil
}
