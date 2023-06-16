package generate

import (
	"context"
	"fmt"

	"github.com/prometheus/prometheus/model/rulefmt"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/info"
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
	GenerateMWMBAlerts(ctx context.Context, slo alert.SLO) (*alert.MWMBAlertGroup, error)
}

// SLIRecordingRulesGenerator knows how to generate SLI recording rules.
type SLIRecordingRulesGenerator interface {
	GenerateSLIRecordingRules(ctx context.Context, slo prometheus.SLO, alerts alert.MWMBAlertGroup) ([]rulefmt.Rule, error)
}

// MetadataRecordingRulesGenerator knows how to generate metadata recording rules.
type MetadataRecordingRulesGenerator interface {
	GenerateMetadataRecordingRules(ctx context.Context, info info.Info, slo prometheus.SLO, alerts alert.MWMBAlertGroup) ([]rulefmt.Rule, error)
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

type Request struct {
	// Info about the application and execution, normally used as metadata.
	Info info.Info
	// ExtraLabels are the extra labels added to the SLOs on execution time.
	ExtraLabels map[string]string
	// disable prometheus promql support if needed
	DisablePromExprValidation bool
	// SLOGroup are the SLOs group that will be used to generate the SLO results and Prom rules.
	SLOGroup prometheus.SLOGroup
}

type SLOResult struct {
	SLO      prometheus.SLO
	Alerts   alert.MWMBAlertGroup
	SLORules prometheus.SLORules
}

type Response struct {
	PrometheusSLOs []SLOResult
}

func (s Service) Generate(ctx context.Context, r Request) (*Response, error) {
	if r.DisablePromExprValidation == false {
		err := r.SLOGroup.Validate()
		if err != nil {
			return nil, fmt.Errorf("invalid SLO group: %w", err)
		}
	}

	// Generate Prom rules.
	results := make([]SLOResult, 0, len(r.SLOGroup.SLOs))
	for _, slo := range r.SLOGroup.SLOs {
		// Add extra labels.
		slo.Labels = mergeLabels(slo.Labels, r.ExtraLabels)

		// Generate SLO result.
		result, err := s.generateSLO(ctx, r.Info, slo)
		if err != nil {
			return nil, fmt.Errorf("could not generate %q slo: %w", slo.ID, err)
		}

		results = append(results, *result)
	}

	return &Response{
		PrometheusSLOs: results,
	}, nil
}

func (s Service) generateSLO(ctx context.Context, info info.Info, slo prometheus.SLO) (*SLOResult, error) {
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
	logger.Infof("Multiwindow-multiburn alerts generated")

	// Generate SLI recording rules.
	sliRecordingRules, err := s.sliRecordRuleGen.GenerateSLIRecordingRules(ctx, slo, *as)
	if err != nil {
		return nil, fmt.Errorf("could not generate Prometheus sli recording rules: %w", err)
	}
	logger.WithValues(log.Kv{"rules": len(sliRecordingRules)}).Infof("SLI recording rules generated")

	// Generate Metadata recording rules.
	metaRecordingRules, err := s.metaRecordRuleGen.GenerateMetadataRecordingRules(ctx, info, slo, *as)
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

func mergeLabels(ms ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}
