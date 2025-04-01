package generate

import (
	"context"
	"fmt"

	"github.com/slok/sloth/internal/log"
)

func newProcessorForSLIRecordingRulesGenerator(logger log.Logger, gen SLIRecordingRulesGenerator) sloProcessor {
	return sloProcessorFunc(func(ctx context.Context, req *sloProcessortRequest, res *sloProcessortResult) error {
		rules, err := gen.GenerateSLIRecordingRules(ctx, req.SLO, req.Alerts)
		if err != nil {
			return fmt.Errorf("could not generate Prometheus sli recording rules: %w", err)
		}
		res.SLORules.SLIErrorRecRules.Rules = rules

		logger.WithValues(log.Kv{"rules": len(rules)}).Infof("SLI recording rules generated")
		return nil
	})
}

func newProcessorForMetaRecordingRulesGenerator(logger log.Logger, gen MetadataRecordingRulesGenerator) sloProcessor {
	return sloProcessorFunc(func(ctx context.Context, req *sloProcessortRequest, res *sloProcessortResult) error {
		rules, err := gen.GenerateMetadataRecordingRules(ctx, req.Info, req.SLO, req.Alerts)
		if err != nil {
			return fmt.Errorf("could not generate Prometheus metadata recording rules: %w", err)
		}
		res.SLORules.MetadataRecRules.Rules = rules

		logger.WithValues(log.Kv{"rules": len(rules)}).Infof("Metadata recording rules generated")
		return nil
	})
}

func newProcessorForSLOAlertRulesGenerator(logger log.Logger, gen SLOAlertRulesGenerator) sloProcessor {
	return sloProcessorFunc(func(ctx context.Context, req *sloProcessortRequest, res *sloProcessortResult) error {
		rules, err := gen.GenerateSLOAlertRules(ctx, req.SLO, req.Alerts)
		if err != nil {
			return fmt.Errorf("could not generate Prometheus alert rules: %w", err)
		}
		res.SLORules.AlertRules.Rules = rules

		logger.WithValues(log.Kv{"rules": len(rules)}).Infof("SLO alert rules generated")
		return nil
	})
}
