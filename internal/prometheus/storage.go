package prometheus

import (
	"context"
	"fmt"
	"io"

	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v2"

	"github.com/slok/sloth/internal/info"
	"github.com/slok/sloth/internal/log"
)

var (
	// ErrNoSLORules will be used when there are no rules to store. The upper layer
	// could ignore or handle the error in cases where there wasn't an output.
	ErrNoSLORules = fmt.Errorf("0 SLO Prometheus rules generated")
)

func NewIOWriterGroupedRulesYAMLRepo(writer io.Writer, logger log.Logger) IOWriterGroupedRulesYAMLRepo {
	return IOWriterGroupedRulesYAMLRepo{
		writer: writer,
		logger: logger.WithValues(log.Kv{"svc": "storage.IOWriter", "format": "yaml"}),
	}
}

// IOWriterGroupedRulesYAMLRepo knows to store all the SLO rules (recordings and alerts)
// grouped in an IOWriter in YAML format, that is compatible with Prometheus.
type IOWriterGroupedRulesYAMLRepo struct {
	writer io.Writer
	logger log.Logger
}

type StorageSLO struct {
	SLO   SLO
	Rules SLORules
}

// StoreSLOs will store the recording and alert prometheus rules, if grouped is false it will
// split and store as 2 different groups the alerts and the recordings, if true
// it will be save as a single group.
func (i IOWriterGroupedRulesYAMLRepo) StoreSLOs(ctx context.Context, slos []StorageSLO) error {
	if len(slos) == 0 {
		return fmt.Errorf("slo rules required")
	}

	ruleGroups := ruleGroupsYAMLv2{}
	for _, slo := range slos {
		if len(slo.Rules.SLIErrorRecRules) > 0 {

			group := ruleGroupYAMLv2{
				Name:  fmt.Sprintf("sloth-slo-sli-recordings-%s", slo.SLO.ID),
				Rules: slo.Rules.SLIErrorRecRules,
			}

			var ruleGroupIntervalDuration prommodel.Duration
			var err error

			switch {
			case slo.SLO.SLIErrorRulesInterval.String() != "0s":
				ruleGroupIntervalDuration, err = prommodel.ParseDuration(slo.SLO.SLIErrorRulesInterval.String())
				if err != nil {
					return fmt.Errorf("could not parse rule_group interval duration for alerts %w", err)
				} else {
					group.RuleGroupInterval = ruleGroupIntervalDuration
				}
			case slo.SLO.RuleGroupInterval.String() != "0s":
				ruleGroupIntervalDuration, err = prommodel.ParseDuration(slo.SLO.RuleGroupInterval.String())
				if err != nil {
					return fmt.Errorf("could not parse default ('all') rule_group interval duration %w", err)
				} else {
					group.RuleGroupInterval = ruleGroupIntervalDuration
				}
			}

			ruleGroups.Groups = append(ruleGroups.Groups, group)
		}

		if len(slo.Rules.MetadataRecRules) > 0 {

			group := ruleGroupYAMLv2{
				Name:  fmt.Sprintf("sloth-slo-meta-recordings-%s", slo.SLO.ID),
				Rules: slo.Rules.MetadataRecRules,
			}

			var ruleGroupIntervalDuration prommodel.Duration
			var err error

			switch {
			case slo.SLO.MetadataRulesInterval.String() != "0s":
				ruleGroupIntervalDuration, err = prommodel.ParseDuration(slo.SLO.MetadataRulesInterval.String())
				if err != nil {
					return fmt.Errorf("could not parse rule_group interval duration for alerts %w", err)
				} else {
					group.RuleGroupInterval = ruleGroupIntervalDuration
				}
			case slo.SLO.RuleGroupInterval.String() != "0s":
				ruleGroupIntervalDuration, err = prommodel.ParseDuration(slo.SLO.RuleGroupInterval.String())
				if err != nil {
					return fmt.Errorf("could not parse default ('all') rule_group interval duration %w", err)
				} else {
					group.RuleGroupInterval = ruleGroupIntervalDuration
				}
			}

			ruleGroups.Groups = append(ruleGroups.Groups, group)
		}

		if len(slo.Rules.AlertRules) > 0 {

			group := ruleGroupYAMLv2{
				Name:  fmt.Sprintf("sloth-slo-alerts-%s", slo.SLO.ID),
				Rules: slo.Rules.AlertRules,
			}

			var ruleGroupIntervalDuration prommodel.Duration
			var err error

			switch {
			case slo.SLO.AlertRulesInterval.String() != "0s":
				ruleGroupIntervalDuration, err = prommodel.ParseDuration(slo.SLO.AlertRulesInterval.String())
				if err != nil {
					return fmt.Errorf("could not parse rule_group interval duration for alerts %w", err)
				} else {
					group.RuleGroupInterval = ruleGroupIntervalDuration
				}
			case slo.SLO.RuleGroupInterval.String() != "0s":
				ruleGroupIntervalDuration, err = prommodel.ParseDuration(slo.SLO.RuleGroupInterval.String())
				if err != nil {
					return fmt.Errorf("could not parse default ('all') rule_group interval duration %w", err)
				} else {
					group.RuleGroupInterval = ruleGroupIntervalDuration
				}
			}

			ruleGroups.Groups = append(ruleGroups.Groups, group)
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

	rulesYaml = writeTopDisclaimer(rulesYaml)
	_, err = i.writer.Write(rulesYaml)
	if err != nil {
		return fmt.Errorf("could not write top disclaimer: %w", err)
	}

	logger := i.logger.WithCtxValues(ctx)
	logger.WithValues(log.Kv{"groups": len(ruleGroups.Groups)}).Infof("Prometheus rules written")

	return nil
}

var disclaimer = fmt.Sprintf(`
---
# Code generated by Sloth (%s): https://github.com/slok/sloth.
# DO NOT EDIT.

`, info.Version)

func writeTopDisclaimer(bs []byte) []byte {
	return append([]byte(disclaimer), bs...)
}

// these types are defined to support yaml v2 (instead of the new Prometheus
// YAML v3 that has some problems with marshaling).
type ruleGroupsYAMLv2 struct {
	Groups []ruleGroupYAMLv2 `yaml:"groups"`
}

type ruleGroupYAMLv2 struct {
	Name              string             `yaml:"name"`
	RuleGroupInterval prommodel.Duration `yaml:"interval,omitempty"`
	Rules             []rulefmt.Rule     `yaml:"rules"`
}
