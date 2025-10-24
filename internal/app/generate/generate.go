package generate

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/log"
	plugincorealertrulesv1 "github.com/slok/sloth/internal/plugin/slo/core/alert_rules_v1"
	plugincoremetadatarulesv1 "github.com/slok/sloth/internal/plugin/slo/core/metadata_rules_v1"
	plugincorenoopv1 "github.com/slok/sloth/internal/plugin/slo/core/noop_v1"
	plugincoreslirulesv1 "github.com/slok/sloth/internal/plugin/slo/core/sli_rules_v1"
	plugincorevalidatev1 "github.com/slok/sloth/internal/plugin/slo/core/validate_v1"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	"github.com/slok/sloth/pkg/common/conventions"
	commonerrors "github.com/slok/sloth/pkg/common/errors"
	"github.com/slok/sloth/pkg/common/model"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
)

// Default plugins.
var (
	NoopPlugin, _ = NewSLOProcessorFromSLOPluginV1(plugincorenoopv1.NewPlugin, log.Noop, nil)
)

type noopSLOPluginGetter bool

func (noopSLOPluginGetter) GetSLOPlugin(ctx context.Context, id string) (*pluginengineslo.Plugin, error) {
	return nil, commonerrors.ErrNotFound
}

type SLOPluginGetter interface {
	GetSLOPlugin(ctx context.Context, id string) (*pluginengineslo.Plugin, error)
}

// ServiceConfig is the application service configuration.
type ServiceConfig struct {
	AlertGenerator  AlertGenerator
	DefaultPlugins  []SLOProcessor
	ExtraPlugins    []model.PromSLOPluginMetadata
	SLOPluginGetter SLOPluginGetter
	Logger          log.Logger
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
	if c.DefaultPlugins == nil {
		pluginValidate, err := NewSLOProcessorFromSLOPluginV1(
			plugincorevalidatev1.NewPlugin,
			c.Logger.WithValues(log.Kv{"plugin": plugincorevalidatev1.PluginID}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("could not create SLO validate plugin: %w", err)
		}

		pluginSLI, err := NewSLOProcessorFromSLOPluginV1(
			plugincoreslirulesv1.NewPlugin,
			c.Logger.WithValues(log.Kv{"plugin": plugincoreslirulesv1.PluginID}),
			plugincoreslirulesv1.PluginConfig{},
		)
		if err != nil {
			return fmt.Errorf("could not create SLI rules plugin: %w", err)
		}

		pluginMeta, err := NewSLOProcessorFromSLOPluginV1(
			plugincoremetadatarulesv1.NewPlugin,
			c.Logger.WithValues(log.Kv{"plugin": plugincoremetadatarulesv1.PluginID}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("could not create metadata rules plugin: %w", err)
		}

		pluginAlert, err := NewSLOProcessorFromSLOPluginV1(
			plugincorealertrulesv1.NewPlugin,
			c.Logger.WithValues(log.Kv{"plugin": plugincorealertrulesv1.PluginID}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("could not create alert rules plugin: %w", err)
		}

		c.DefaultPlugins = []SLOProcessor{
			pluginValidate,
			pluginSLI,
			pluginMeta,
			pluginAlert,
		}
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
	extraPlugins    []model.PromSLOPluginMetadata
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
		defaultPlugins:  config.DefaultPlugins,
		extraPlugins:    config.ExtraPlugins,
		logger:          config.Logger,
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
	err := s.validateSLOGroup(r.SLOGroup)
	if err != nil {
		return nil, fmt.Errorf("invalid SLO group: %w", err)
	}

	// Generate Prom rules.
	results := make([]SLOResult, 0, len(r.SLOGroup.SLOs))
	for _, slo := range r.SLOGroup.SLOs {
		// Add extra labels.
		slo.Labels = utilsdata.MergeLabels(slo.Labels, r.ExtraLabels)

		// Generate SLO result.
		result, err := s.generateSLO(ctx, r.Info, r.SLOGroup, slo)
		if err != nil {
			return nil, fmt.Errorf("could not generate %q slo: %w", slo.ID, err)
		}

		// Set safe defaults on rules result.
		setDefaultsPromSLORulesResult(slo, result)

		results = append(results, SLOResult{SLO: slo, SLORules: *result})
	}

	return &Response{
		PrometheusSLOs: results,
	}, nil
}

func (s Service) generateSLO(ctx context.Context, info model.Info, sloGroup model.PromSLOGroup, slo model.PromSLO) (*model.PromSLORules, error) {
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

	// Get SLO plugins based on the priority, default plugins are `0` priority
	// so, we split the plugins in two slices, pre default (<0) and post default (>=0).
	// That way we create the final processor list: pre-default + default + post-default.
	preDefault := []SLOProcessor{}
	postDefault := []SLOProcessor{}
	sloPluginMetadata := []model.PromSLOPluginMetadata{}
	if !slo.Plugins.OverridePlugins {
		sloPluginMetadata = append(sloPluginMetadata, s.extraPlugins...) // App level SLO plugins.
	}
	sloPluginMetadata = append(sloPluginMetadata, slo.Plugins.Plugins...) // SLO (group and/or individual) level plugins.
	slices.SortStableFunc(sloPluginMetadata, func(a, b model.PromSLOPluginMetadata) int {
		return cmp.Compare(a.Priority, b.Priority)
	})

	for _, p := range sloPluginMetadata {
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

		if p.Priority < 0 {
			preDefault = append(preDefault, processor)
		} else {
			postDefault = append(postDefault, processor)
		}
	}

	// Prepare processors.
	sloProcessors := preDefault
	// Add default plugins if we don't want to override the plugins.
	if !slo.Plugins.OverridePlugins {
		sloProcessors = append(sloProcessors, s.defaultPlugins...)
	}
	sloProcessors = append(sloProcessors, postDefault...)

	req := &SLOProcessorRequest{
		Info:           info,
		MWMBAlertGroup: *as,
		SLO:            slo,
		SLOGroup:       sloGroup,
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

func (s Service) validateSLOGroup(sloGroup model.PromSLOGroup) error {
	if len(sloGroup.SLOs) == 0 {
		return fmt.Errorf("at least one SLO is required")
	}

	// Check SLO IDs not repeated.
	sloIDs := map[string]struct{}{}
	for _, slo := range sloGroup.SLOs {
		_, ok := sloIDs[slo.ID]
		if ok {
			return fmt.Errorf("SLO ID %q is repeated", slo.ID)
		}
		sloIDs[slo.ID] = struct{}{}
	}

	return nil
}

// helper function to set the required safe defaults generated SLO prom rules.
func setDefaultsPromSLORulesResult(slo model.PromSLO, rules *model.PromSLORules) {
	// Set rule result naming defaults.
	if rules.SLIErrorRecRules.Name == "" {
		rules.SLIErrorRecRules.Name = conventions.PromRuleGroupNameSLOSLIPrefix + slo.ID
	}
	if rules.MetadataRecRules.Name == "" {
		rules.MetadataRecRules.Name = conventions.PromRuleGroupNameSLOMetadataPrefix + slo.ID
	}
	if rules.AlertRules.Name == "" {
		rules.AlertRules.Name = conventions.PromRuleGroupNameSLOAlertsPrefix + slo.ID
	}
	for i, extraRuleGroup := range rules.ExtraRules {
		if extraRuleGroup.Name == "" {
			extraRuleGroup.Name = fmt.Sprintf("%s%03d-%s", conventions.PromRuleGroupNameSLOExtraRulesPrefix, i, slo.ID)
		}
	}
}
