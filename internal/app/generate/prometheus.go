package generate

import (
	"context"
	"fmt"

	"github.com/prometheus/prometheus/model/rulefmt"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/prometheus"
	"github.com/slok/sloth/pkg/common/model"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
)

// ServiceConfig is the application service configuration.
type ServiceConfig struct {
	AlertGenerator              AlertGenerator
	SLIRecordingRulesGenerator  SLIRecordingRulesGenerator
	MetaRecordingRulesGenerator MetadataRecordingRulesGenerator
	SLOAlertRulesGenerator      SLOAlertRulesGenerator
	Logger                      log.Logger
}

func (c *ServiceConfig) defaults() error {
	if c.AlertGenerator == nil {
		return fmt.Errorf("alert generator is required")
	}

	if c.SLIRecordingRulesGenerator == nil {
		c.SLIRecordingRulesGenerator = prometheus.OptimizedSLIRecordingRulesGenerator
	}

	if c.MetaRecordingRulesGenerator == nil {
		c.MetaRecordingRulesGenerator = prometheus.MetadataRecordingRulesGenerator
	}

	if c.SLOAlertRulesGenerator == nil {
		c.SLOAlertRulesGenerator = prometheus.SLOAlertRulesGenerator
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"svc": "generate.prometheus.Service"})

	return nil
}

// AlertGenerator knows how to generate multiwindow multi-burn SLO alerts.
type AlertGenerator interface {
	GenerateMWMBAlerts(ctx context.Context, slo alert.SLO) (*model.MWMBAlertGroup, error)
}

// SLIRecordingRulesGenerator knows how to generate SLI recording rules.
type SLIRecordingRulesGenerator interface {
	GenerateSLIRecordingRules(ctx context.Context, slo prometheus.SLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error)
}

// MetadataRecordingRulesGenerator knows how to generate metadata recording rules.
type MetadataRecordingRulesGenerator interface {
	GenerateMetadataRecordingRules(ctx context.Context, info model.Info, slo prometheus.SLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error)
}

// SLOAlertRulesGenerator knows hot to generate SLO alert rules.
type SLOAlertRulesGenerator interface {
	GenerateSLOAlertRules(ctx context.Context, slo prometheus.SLO, alerts model.MWMBAlertGroup) ([]rulefmt.Rule, error)
}

// Service is the application service for the generation of SLO for Prometheus.
type Service struct {
	alertGen          AlertGenerator
	sliRecordRuleGen  SLIRecordingRulesGenerator
	metaRecordRuleGen MetadataRecordingRulesGenerator
	alertRuleGen      SLOAlertRulesGenerator
	logger            log.Logger
}

// NewService returns a new Prometheus application service.
func NewService(config ServiceConfig) (*Service, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &Service{
		alertGen:          config.AlertGenerator,
		sliRecordRuleGen:  config.SLIRecordingRulesGenerator,
		metaRecordRuleGen: config.MetaRecordingRulesGenerator,
		alertRuleGen:      config.SLOAlertRulesGenerator,
		logger:            config.Logger,
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

func (s Service) generateSLO(ctx context.Context, info model.Info, slo prometheus.SLO) (*model.PromSLORules, error) {
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

	// Set default processors.
	processors := []sloProcessor{
		newProcessorForMetaRecordingRulesGenerator(logger, s.metaRecordRuleGen),
		newProcessorForSLIRecordingRulesGenerator(logger, s.sliRecordRuleGen),
		newProcessorForSLOAlertRulesGenerator(logger, s.alertRuleGen),
	}

	req := &sloProcessortRequest{
		Info:   info,
		Alerts: *as,
		SLO:    slo,
	}
	res := &sloProcessortResult{}
	for _, p := range processors {
		err := p.ProcessSLO(ctx, req, res)
		if err != nil {
			return nil, fmt.Errorf("slo processor failed: %w", err)
		}
	}

	return &res.SLORules, nil
}
