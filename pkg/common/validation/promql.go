package validation

import (
	"bytes"
	"fmt"
	"text/template"

	prommodel "github.com/prometheus/common/model"
	promqlparser "github.com/prometheus/prometheus/promql/parser"
)

// PromQLDialectValidator is the SLO flavour validator for prometheus backends dialect: PromQL.
const PromQLDialectValidator = promQLDialectValidator(false)

type promQLDialectValidator bool

func (promQLDialectValidator) ValidateLabelKey(k string) error {
	if k == prommodel.MetricNameLabel {
		return fmt.Errorf("the label key %q is not allowed", prommodel.MetricNameLabel)
	}
	if !prommodel.LabelName(k).IsValid() {
		return fmt.Errorf("the label key %q is not valid", k)
	}

	return nil
}

func (promQLDialectValidator) ValidateLabelValue(k string) error {
	if k == "" {
		return fmt.Errorf("the label value is required")
	}

	if !prommodel.LabelValue(k).IsValid() {
		return fmt.Errorf("the label value %q is not valid", k)
	}

	return nil
}

func (promQLDialectValidator) ValidateAnnotationKey(k string) error {
	if !prommodel.LabelName(k).IsValid() {
		return fmt.Errorf("the annotation key %q is not valid", k)
	}

	return nil
}

func (promQLDialectValidator) ValidateAnnotationValue(k string) error {
	if k == "" {
		return fmt.Errorf("the annotation value is required")
	}

	return nil
}

var promExprTplAllowedFakeData = map[string]string{"window": "1m"}

func (promQLDialectValidator) ValidateQueryExpression(queryExpression string) error {
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

	_, err = promqlparser.ParseExpr(tplB.String())

	return err
}
