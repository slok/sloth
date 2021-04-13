package prometheus

import (
	"bytes"
	"text/template"
	"time"

	"github.com/go-playground/validator/v10"
	prommodel "github.com/prometheus/common/model"
	promqlparser "github.com/prometheus/prometheus/promql/parser"

	"github.com/slok/sloth/internal/model"
)

type CustomSLI struct {
	ErrorQuery string `validate:"required,prom_expr"`
	TotalQuery string `validate:"required,prom_expr"`
}

func (CustomSLI) IsSLI() {}

type AlertMeta struct {
	Disable     bool
	Name        string            `validate:"required"`
	Labels      map[string]string `validate:"dive,keys,prom_label_key,endkeys,required,prom_label_value"`
	Annotations map[string]string `validate:"dive,keys,prom_annot_key,endkeys,required"`
}

type SLO struct {
	ID               string    `validate:"required"`
	Service          string    `validate:"required"`
	SLI              CustomSLI `validate:"required"`
	TimeWindow       time.Duration
	Objective        float64           `validate:"gt=0,lte=100"`
	Labels           map[string]string `validate:"dive,keys,prom_label_key,endkeys,required,prom_label_value"`
	PageAlertMeta    AlertMeta
	WarningAlertMeta AlertMeta
}

func (s SLO) GetID() string                { return s.ID }
func (s SLO) GetService() string           { return s.Service }
func (s SLO) GetSLI() model.SLI            { return s.SLI }
func (s SLO) GetTimeWindow() time.Duration { return s.TimeWindow }
func (s SLO) GetObjective() float64        { return s.Objective }
func (s SLO) Validate() error {
	return modelSpecValidate.Struct(s)
}

var modelSpecValidate = func() *validator.Validate {
	v := validator.New()

	// Prometheus validators.
	// More info here: https://github.com/prometheus/prometheus/blob/df80dc4d3970121f2f76cba79050983ffb3cdbb0/pkg/rulefmt/rulefmt.go#L188-L208
	mustRegisterValidation(v, "prom_expr", validatePromExpression)
	mustRegisterValidation(v, "prom_label_key", validatePromLabelKey)
	mustRegisterValidation(v, "prom_label_value", validatePromLabelValue)
	mustRegisterValidation(v, "prom_annot_key", validatePromAnnotKey)

	return v
}()

// mustRegisterValidation is a helper so we panic on start if we can't register a validator.
func mustRegisterValidation(v *validator.Validate, tag string, fn validator.Func) {
	err := v.RegisterValidation(tag, fn)
	if err != nil {
		panic(err)
	}
}

// validatePromAnnotKey implements validator.CustomTypeFunc by validating
// a prometheus annotation key.
func validatePromAnnotKey(fl validator.FieldLevel) bool {
	k, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	return prommodel.LabelName(k).IsValid()
}

// validatePromLabel implements validator.CustomTypeFunc by validating
// a prometheus label key.
func validatePromLabelKey(fl validator.FieldLevel) bool {
	k, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	return prommodel.LabelName(k).IsValid() && k != prommodel.MetricNameLabel
}

// validatePromLabelValue implements validator.CustomTypeFunc by validating
// a prometheus label value.
func validatePromLabelValue(fl validator.FieldLevel) bool {
	v, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	return prommodel.LabelValue(v).IsValid()
}

var promExprTplAllowedFakeData = map[string]string{
	"window": "1m",
}

// validatePromExpression implements validator.CustomTypeFunc by validating
// a prometheus expression.
func validatePromExpression(fl validator.FieldLevel) bool {
	expr, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	// The expressions set by users can have some allowed templated data
	// we are rendering the expression with fake data so prometheus can
	// have a final expr and check if is correct.
	tpl, err := template.New("expr").Parse(expr)
	if err != nil {
		return false
	}

	var tplB bytes.Buffer
	err = tpl.Execute(&tplB, promExprTplAllowedFakeData)
	if err != nil {
		return false
	}

	_, err = promqlparser.ParseExpr(tplB.String())
	return err == nil
}
