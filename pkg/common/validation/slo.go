package validation

import (
	"fmt"

	"github.com/slok/sloth/pkg/common/conventions"
	commonerrors "github.com/slok/sloth/pkg/common/errors"
	"github.com/slok/sloth/pkg/common/model"
)

func isValidName(name string) error {
	if name == "" {
		return commonerrors.ErrRequired
	}

	if !conventions.NameRegexp.MatchString(name) {
		return fmt.Errorf("name must start and end with an alphanumeric and can only contain alphanumeric, '.', '_', and '-'")
	}

	return nil
}

func isValidSLIQueryTemplate(sliQuery string) error {
	if sliQuery == "" {
		return fmt.Errorf("query template is required: %w", commonerrors.ErrRequired)
	}

	if !conventions.TplWindowRegex.MatchString(sliQuery) {
		return fmt.Errorf("template must contain the {{ .window }} variable")
	}

	return nil
}

func isValidSLOSLI(slo model.PromSLO, dialect SLODialectValidator) error {
	sli := slo.SLI

	if sli.Events == nil && sli.Raw == nil {
		return fmt.Errorf("at least one SLI type is required")
	}

	if sli.Events != nil && sli.Raw != nil {
		return fmt.Errorf("only one SLI type is allowed")
	}

	switch {
	case sli.Events != nil:
		// If they are the same, they are invalid.
		if sli.Events.ErrorQuery == sli.Events.TotalQuery {
			return fmt.Errorf("both error and total queries can't be the same")
		}

		if err := isValidSLIQueryTemplate(sli.Events.ErrorQuery); err != nil {
			return fmt.Errorf("sli error query template: %w", err)
		}

		if err := isValidSLIQueryTemplate(sli.Events.TotalQuery); err != nil {
			return fmt.Errorf("sli total query template: %w", err)
		}

		if err := dialect.ValidateQueryExpression(sli.Events.ErrorQuery); err != nil {
			return fmt.Errorf("sli error query expression: %w", err)
		}

		if err := dialect.ValidateQueryExpression(sli.Events.TotalQuery); err != nil {
			return fmt.Errorf("sli total query expression: %w", err)
		}

	case sli.Raw != nil:
		if err := isValidSLIQueryTemplate(sli.Raw.ErrorRatioQuery); err != nil {
			return fmt.Errorf("sli raw query template: %w", err)
		}

		if err := dialect.ValidateQueryExpression(sli.Raw.ErrorRatioQuery); err != nil {
			return fmt.Errorf("sli raw query expression: %w", err)
		}
	}

	return nil
}

func isValidSLOAlert(slo model.PromSLO, dialect SLODialectValidator) error {
	if err := isValidAlert(slo.PageAlertMeta, dialect); err != nil {
		return fmt.Errorf("page alert: %w", err)
	}

	if err := isValidAlert(slo.TicketAlertMeta, dialect); err != nil {
		return fmt.Errorf("ticket alert: %w", err)
	}

	return nil
}

func isValidAlert(alert model.PromAlertMeta, dialect SLODialectValidator) error {
	if alert.Disable {
		return nil
	}

	if alert.Name == "" {
		return fmt.Errorf("alert name is required")
	}

	for k, v := range alert.Labels {
		if err := dialect.ValidateLabelKey(k); err != nil {
			return fmt.Errorf("invalid alert label key %q: %w", k, err)
		}
		if err := dialect.ValidateLabelValue(v); err != nil {
			return fmt.Errorf("invalid alert label value %q: %w", v, err)
		}
	}

	for k, v := range alert.Annotations {
		if err := dialect.ValidateAnnotationKey(k); err != nil {
			return fmt.Errorf("invalid alert annotation key %q: %w", k, err)
		}
		if err := dialect.ValidateAnnotationValue(v); err != nil {
			return fmt.Errorf("invalid alert annotation value %q: %w", v, err)
		}
	}

	return nil
}

func isValidPlugins(slo model.PromSLO) error {
	if slo.Plugins.OverrideDefaultPlugins && len(slo.Plugins.Plugins) == 0 {
		return fmt.Errorf("override default plugins is set but no plugins are defined")
	}

	for _, p := range slo.Plugins.Plugins {
		if p.ID == "" {
			return fmt.Errorf("plugin ID is required")
		}
	}

	return nil
}

// SLODialectValidator is the interface that all SLO dialects must implement to validate
// SLOs. A dialect can me Prometheus PromQL, or VictoriaMetrics metricsQL for example.
type SLODialectValidator interface {
	ValidateLabelKey(k string) error
	ValidateLabelValue(k string) error
	ValidateAnnotationKey(k string) error
	ValidateAnnotationValue(k string) error
	ValidateQueryExpression(queryExpression string) error
}

func ValidateSLO(slo model.PromSLO, dialect SLODialectValidator) error {
	if err := isValidName(slo.ID); err != nil {
		return fmt.Errorf("invalid SLO ID: %w", err)
	}

	if err := isValidName(slo.Name); err != nil {
		return fmt.Errorf("invalid SLO name: %w", err)
	}

	if err := isValidName(slo.Service); err != nil {
		return fmt.Errorf("invalid SLO service: %w", err)
	}

	if slo.TimeWindow == 0 {
		return fmt.Errorf("time window is required")
	}

	if slo.Objective <= 0 || slo.Objective > 100 {
		return fmt.Errorf("objective must >0 and <=100")
	}

	for k, v := range slo.Labels {
		if err := dialect.ValidateLabelKey(k); err != nil {
			return fmt.Errorf("invalid SLO label key %q: %w", k, err)
		}
		if err := dialect.ValidateLabelValue(v); err != nil {
			return fmt.Errorf("invalid SLO label value %q: %w", v, err)
		}
	}

	if err := isValidSLOSLI(slo, dialect); err != nil {
		return fmt.Errorf("invalid SLI: %w", err)
	}

	if err := isValidSLOAlert(slo, dialect); err != nil {
		return fmt.Errorf("invalid alert: %w", err)
	}

	if err := isValidPlugins(slo); err != nil {
		return fmt.Errorf("invalid plugins: %w", err)
	}

	return nil
}
