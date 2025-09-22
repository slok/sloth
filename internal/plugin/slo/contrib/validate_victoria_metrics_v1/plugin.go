package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/VictoriaMetrics/metricsql"

	"github.com/slok/sloth/pkg/common/validation"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/contrib/validate_victoria_metrics/v1"
)

func NewPlugin(_ json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	return plugin{
		appUtils: appUtils,
	}, nil
}

type plugin struct {
	appUtils pluginslov1.AppUtils
}

func (p plugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	err := validation.ValidateSLO(request.SLO, VictoriaMetricsDialectValidator)
	if err != nil {
		return fmt.Errorf("invalid slo %q: %w", request.SLO.ID, err)
	}

	return nil
}

// VictoriaMetricsDialectValidator is the SLO flavour validator for victoria metrics backends dialect.
var VictoriaMetricsDialectValidator = victoriaMetricsDialectValidator{
	promValidator: validation.PromQLDialectValidator,
}

type victoriaMetricsDialectValidator struct {
	promValidator validation.SLODialectValidator
}

var promExprTplAllowedFakeData = map[string]string{"window": "1m"}

func (v victoriaMetricsDialectValidator) ValidateLabelKey(key string) error {
	return v.promValidator.ValidateLabelKey(key)
}

func (v victoriaMetricsDialectValidator) ValidateLabelValue(label string) error {
	return v.promValidator.ValidateLabelValue(label)
}

func (v victoriaMetricsDialectValidator) ValidateAnnotationKey(key string) error {
	return v.promValidator.ValidateAnnotationKey(key)
}

func (v victoriaMetricsDialectValidator) ValidateAnnotationValue(annot string) error {
	return v.promValidator.ValidateAnnotationValue(annot)
}

func (victoriaMetricsDialectValidator) ValidateQueryExpression(queryExpression string) error {
	if queryExpression == "" {
		return fmt.Errorf("query is required")
	}

	// The expressions set by users can have some allowed templated data.
	// We are rendering the expression with fake data so prometheus can
	// have a final expr and check if is correct.
	tpl, err := template.New("expr").Parse(queryExpression)
	if err != nil {
		return err
	}

	var tplB bytes.Buffer
	err = tpl.Execute(&tplB, promExprTplAllowedFakeData)
	if err != nil {
		return err
	}

	_, err = metricsql.Parse(tplB.String())

	return err
}
