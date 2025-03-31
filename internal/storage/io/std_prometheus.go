package io

import (
	"context"
	"fmt"
	"io"

	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v2"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/common/model"
)

var (
	// ErrNoSLORules will be used when there are no rules to store. The upper layer
	// could ignore or handle the error in cases where there wasn't an output.
	ErrNoSLORules = fmt.Errorf("0 SLO Prometheus rules generated")
)

func NewStdPrometheusGroupedRulesYAMLRepo(writer io.Writer, logger log.Logger) StdPrometheusGroupedRulesYAMLRepo {
	return StdPrometheusGroupedRulesYAMLRepo{
		writer: writer,
		logger: logger.WithValues(log.Kv{"svc": "storageio.StdPrometheusGroupedRulesYAMLRepo"}),
	}
}

// StdPrometheusGroupedRulesYAMLRepo knows to store all the SLO rules (recordings and alerts)
// grouped in an IOWriter in YAML format, that is compatible with Prometheus.
type StdPrometheusGroupedRulesYAMLRepo struct {
	writer io.Writer
	logger log.Logger
}

type StdPrometheusStorageSLO struct {
	SLO   model.PromSLO
	Rules model.PromSLORules
}

// StoreSLOs will store the recording and alert prometheus rules, if grouped is false it will
// split and store as 2 different groups the alerts and the recordings, if true
// it will be save as a single group.
func (r StdPrometheusGroupedRulesYAMLRepo) StoreSLOs(ctx context.Context, slos []StdPrometheusStorageSLO) error {
	if len(slos) == 0 {
		return fmt.Errorf("slo rules required")
	}

	ruleGroups := stdPromRuleGroupsYAMLv2{}
	for _, slo := range slos {
		if len(slo.Rules.SLIErrorRecRules.Rules) > 0 {
			ruleGroups.Groups = append(ruleGroups.Groups, stdPromRuleGroupYAMLv2{
				Name:  fmt.Sprintf("sloth-slo-sli-recordings-%s", slo.SLO.ID),
				Rules: slo.Rules.SLIErrorRecRules.Rules,
			})
		}

		if len(slo.Rules.MetadataRecRules.Rules) > 0 {
			ruleGroups.Groups = append(ruleGroups.Groups, stdPromRuleGroupYAMLv2{
				Name:  fmt.Sprintf("sloth-slo-meta-recordings-%s", slo.SLO.ID),
				Rules: slo.Rules.MetadataRecRules.Rules,
			})
		}

		if len(slo.Rules.AlertRules.Rules) > 0 {
			ruleGroups.Groups = append(ruleGroups.Groups, stdPromRuleGroupYAMLv2{
				Name:  fmt.Sprintf("sloth-slo-alerts-%s", slo.SLO.ID),
				Rules: slo.Rules.AlertRules.Rules,
			})
		}
	}

	// If we don't have anything to store, error so we can increase the reliability
	// because maybe this was due to an unintended error (typos, misconfig, too many disable...).
	if len(ruleGroups.Groups) == 0 {
		return ErrNoSLORules
	}

	// Convert to YAML (Prometheus rule format).
	rulesYaml, err := yaml.Marshal(ruleGroups)
	if err != nil {
		return fmt.Errorf("could not format rules: %w", err)
	}

	rulesYaml = writeYAMLTopDisclaimer(rulesYaml)
	_, err = r.writer.Write(rulesYaml)
	if err != nil {
		return fmt.Errorf("could not write top disclaimer: %w", err)
	}

	logger := r.logger.WithCtxValues(ctx)
	logger.WithValues(log.Kv{"groups": len(ruleGroups.Groups)}).Infof("Prometheus rules written")

	return nil
}

// these types are defined to support yaml v2 (instead of the new Prometheus
// YAML v3 that has some problems with marshaling).
type stdPromRuleGroupsYAMLv2 struct {
	Groups []stdPromRuleGroupYAMLv2 `yaml:"groups"`
}

type stdPromRuleGroupYAMLv2 struct {
	Name     string             `yaml:"name"`
	Interval prommodel.Duration `yaml:"interval,omitempty"`
	Rules    []rulefmt.Rule     `yaml:"rules"`
}
