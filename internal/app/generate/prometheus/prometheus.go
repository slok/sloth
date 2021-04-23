package prometheus

import (
	"context"
	"fmt"

	"github.com/prometheus/prometheus/pkg/rulefmt"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/prometheus"
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
		c.AlertGenerator = alert.AlertGenerator
	}

	if c.SLIRecordingRulesGenerator == nil {
		c.SLIRecordingRulesGenerator = prometheus.SLIRecordingRulesGenerator
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
	GenerateMWMBAlerts(ctx context.Context, slo alert.SLO) (*alert.MWMBAlertGroup, error)
}

// SLIRecordingRulesGenerator knows how to generate SLI recording rules.
type SLIRecordingRulesGenerator interface {
	GenerateSLIRecordingRules(ctx context.Context, slo prometheus.SLO, alerts alert.MWMBAlertGroup) ([]rulefmt.Rule, error)
}

// MetadataRecordingRulesGenerator knows how to generate metadata recording rules.
type MetadataRecordingRulesGenerator interface {
	GenerateMetadataRecordingRules(ctx context.Context, slo prometheus.SLO, alerts alert.MWMBAlertGroup) ([]rulefmt.Rule, error)
}

// SLOAlertRulesGenerator knows hot to generate SLO alert rules.
type SLOAlertRulesGenerator interface {
	GenerateSLOAlertRules(ctx context.Context, slo prometheus.SLO, alerts alert.MWMBAlertGroup) ([]rulefmt.Rule, error)
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

type GenerateRequest struct {
	SLOs []prometheus.SLO
}

type SLOResult struct {
	SLO      prometheus.SLO
	Alerts   alert.MWMBAlertGroup
	SLORules prometheus.SLORules
}

type GenerateResponse struct {
	PrometheusSLOs []SLOResult
}

func (s Service) Generate(ctx context.Context, r GenerateRequest) (*GenerateResponse, error) {
	if len(r.SLOs) == 0 {
		return nil, fmt.Errorf("slos are required")
	}

	// Alert generation.
	results := make([]SLOResult, 0, len(r.SLOs))
	for _, slo := range r.SLOs {
		result, err := s.generateSLO(ctx, slo)
		if err != nil {
			return nil, fmt.Errorf("could not generate %q slo: %w", slo.ID, err)
		}

		results = append(results, *result)
	}

	return &GenerateResponse{
		PrometheusSLOs: results,
	}, nil
}

func (s Service) generateSLO(ctx context.Context, slo prometheus.SLO) (*SLOResult, error) {
	// Validate before using the SLO.
	err := slo.Validate()
	if err != nil {
		return nil, fmt.Errorf("slo is invalid: %w", err)
	}

	logger := s.logger.WithValues(log.Kv{"slo": slo.ID})

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
	logger.Infof("Multiwindow-multiburn alerts generated")

	// Generate SLI recording rules.
	sliRecordingRules, err := s.sliRecordRuleGen.GenerateSLIRecordingRules(ctx, slo, *as)
	if err != nil {
		return nil, fmt.Errorf("could not generate Prometheus sli recording rules: %w", err)
	}
	logger.WithValues(log.Kv{"rules": len(sliRecordingRules)}).Infof("SLI recording rules generated")

	// Generate Metadata recording rules.
	metaRecordingRules, err := s.metaRecordRuleGen.GenerateMetadataRecordingRules(ctx, slo, *as)
	if err != nil {
		return nil, fmt.Errorf("could not generate Prometheus metadata recording rules: %w", err)
	}
	logger.WithValues(log.Kv{"rules": len(metaRecordingRules)}).Infof("Metadata recording rules generated")

	// Generate Alert rules.
	alertRules, err := s.alertRuleGen.GenerateSLOAlertRules(ctx, slo, *as)
	if err != nil {
		return nil, fmt.Errorf("could not generate Prometheus alert rules: %w", err)
	}
	logger.WithValues(log.Kv{"rules": len(alertRules)}).Infof("SLO alert rules generated")

	return &SLOResult{
		SLO:    slo,
		Alerts: *as,
		SLORules: prometheus.SLORules{
			SLIErrorRecRules: sliRecordingRules,
			MetadataRecRules: metaRecordingRules,
			AlertRules:       alertRules,
		},
	}, nil
}
