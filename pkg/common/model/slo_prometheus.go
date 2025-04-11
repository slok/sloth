package model

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"text/template"
	"time"

	openslov1alpha "github.com/OpenSLO/oslo/pkg/manifest/v1alpha"
	"github.com/VictoriaMetrics/metricsql"
	"github.com/go-playground/validator/v10"
	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	promqlparser "github.com/prometheus/prometheus/promql/parser"

	k8sprometheusv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	prometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
)

// SLI represents an SLI with custom error and total expressions.
type PromSLI struct {
	Raw    *PromSLIRaw
	Events *PromSLIEvents
}

type PromSLIRaw struct {
	ErrorRatioQuery string `validate:"required,prom_expr,template_vars"`
}

type PromSLIEvents struct {
	ErrorQuery string `validate:"required,prom_expr,template_vars"`
	TotalQuery string `validate:"required,prom_expr,template_vars"`
}

// AlertMeta is the metadata of an alert settings.
type PromAlertMeta struct {
	Disable     bool
	Name        string            `validate:"required_if_enabled"`
	Labels      map[string]string `validate:"dive,keys,prom_label_key,endkeys,required,prom_label_value"`
	Annotations map[string]string `validate:"dive,keys,prom_annot_key,endkeys,required"`
}

// PromSLO represents a service level objective configuration.
type PromSLO struct {
	ID              string `validate:"required,name"`
	Name            string `validate:"required,name"`
	Description     string
	Service         string            `validate:"required,name"`
	SLI             PromSLI           `validate:"required"`
	TimeWindow      time.Duration     `validate:"required"`
	Objective       float64           `validate:"gt=0,lte=100"`
	Labels          map[string]string `validate:"dive,keys,prom_label_key,endkeys,required,prom_label_value"`
	PageAlertMeta   PromAlertMeta
	TicketAlertMeta PromAlertMeta
	Plugins         SLOPlugins
}

type SLOPlugins struct {
	Plugins []PromSLOPluginMetadata
}

type PromSLOPluginMetadata struct {
	ID       string
	Config   any
	Priority int
}

type PromSLOGroup struct {
	SLOs           []PromSLO
	OriginalSource PromSLOGroupSource
}

// Used to store the original source of the SLO group in case we need to make low-level decision
// based on where the SLOs came from.
type PromSLOGroupSource struct {
	K8sSlothV1     *k8sprometheusv1.PrometheusServiceLevel
	SlothV1        *prometheusv1.Spec
	OpenSLOV1Alpha *openslov1alpha.SLO
}

// QueryValidator knows which validator to use (promql, metricsql).
type QueryValidator struct {
	PromQL    bool
	MetricsQL bool
}

// Validate validates the SLO.
func (s PromSLO) Validate(dialect QueryValidator) error {
	if dialect.PromQL {
		return modelSpecValidatePromQL.Struct(s)
	} else if dialect.MetricsQL {
		return modelSpecValidateMetricsQL.Struct(s)
	}

	return errors.New("no validator set for SLOGroup")
}

var modelSpecValidate = func() *validator.Validate {
	v := validator.New()
	commonValidations(v)
	return v
}

var modelSpecValidatePromQL = func() *validator.Validate {
	v := validator.New()
	// More information on prometheus validators logic: https://github.com/prometheus/prometheus/blob/df80dc4d3970121f2f76cba79050983ffb3cdbb0/pkg/rulefmt/rulefmt.go#L188-L208
	mustRegisterValidation(v, "prom_expr", validatePromQLExpression)
	commonValidations(v)
	return v
}()
var modelSpecValidateMetricsQL = func() *validator.Validate {
	v := validator.New()
	mustRegisterValidation(v, "prom_expr", validateMetricsQLExpression)
	commonValidations(v)
	return v
}()

func commonValidations(v *validator.Validate) {
	mustRegisterValidation(v, "prom_label_key", validatePromLabelKey)
	mustRegisterValidation(v, "prom_label_value", validatePromLabelValue)
	mustRegisterValidation(v, "prom_annot_key", validatePromAnnotKey)
	mustRegisterValidation(v, "name", validateName)
	mustRegisterValidation(v, "required_if_enabled", validateRequiredEnabledAlertName)
	mustRegisterValidation(v, "template_vars", validateTemplateVars)
	v.RegisterStructValidation(validateOneSLI, PromSLI{})
	v.RegisterStructValidation(validateSLIEvents, PromSLIEvents{})
}

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

// validatePromQLExpression implements validator.CustomTypeFunc by validating
// a prometheus expression.
func validatePromQLExpression(fl validator.FieldLevel) bool {
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

// validateMetricsQLExpression implements validator.CustomTypeFunc by validating
// a VictoriaMetrics expression.
func validateMetricsQLExpression(fl validator.FieldLevel) bool {
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

	_, err = metricsql.Parse(tplB.String())
	return err == nil
}

// Names must:
// - Start and end with an alphanumeric.
// - Contain alphanumeric, `.`, '_', and '-'.
var (
	nameRegexp = regexp.MustCompile("^[A-Za-z0-9][-A-Za-z0-9_.]*[A-Za-z0-9]$")
)

// validateName implements validator.CustomTypeFunc by validating
// a regular name.
func validateName(fl validator.FieldLevel) bool {
	s, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	return nameRegexp.MatchString(s)
}

func validateRequiredEnabledAlertName(fl validator.FieldLevel) bool {
	alertMeta, ok := fl.Parent().Interface().(PromAlertMeta)
	if !ok {
		return false
	}

	if alertMeta.Disable {
		return true
	}

	return alertMeta.Name != ""
}

const PromQueryTPLKeyWindow = "window"

var tplWindowRegex = regexp.MustCompile(fmt.Sprintf(`{{ *\.%s *}}`, PromQueryTPLKeyWindow))

// validateTemplateVars implements validator.CustomTypeFunc by validating
// an SLI template has all the required fields.
func validateTemplateVars(fl validator.FieldLevel) bool {
	v, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	return tplWindowRegex.MatchString(v)
}

// validateSLIEvents validates that both SLI event queries are different.
func validateSLIEvents(sl validator.StructLevel) {
	s, ok := sl.Current().Interface().(PromSLIEvents)
	if !ok {
		sl.ReportError(s, "", "SLIEvents", "not_sli_events", "")
		return
	}

	// If empty we don't need to check.
	if s.ErrorQuery == "" || s.TotalQuery == "" {
		return
	}

	// If different, they are valid.
	if s.ErrorQuery == s.TotalQuery {
		sl.ReportError(s, "", "", "sli_events_queries_different", "")
		return
	}
}

// validateOneSLI validates only one SLI type is set and configured.
func validateOneSLI(sl validator.StructLevel) {
	sli, ok := sl.Current().Interface().(PromSLI)
	if !ok {
		sl.ReportError(sli, "", "SLI", "not_sli", "")
		return
	}

	// Check only one SLI type is set.
	sliSet := false
	sliType := reflect.ValueOf(sli)
	strNumFields := sliType.NumField()
	for i := 0; i < strNumFields; i++ {
		f := sliType.Field(i)
		if f.IsNil() {
			continue
		}
		// We already have one SLI type set.
		if sliSet {
			sl.ReportError(sli, "", "", "one_sli_type", "")
		}
		sliSet = true
	}

	// No SLI types set.
	if !sliSet {
		sl.ReportError(sli, "", "", "sli_type_required", "")
	}
}

// PromSLORules are the prometheus rules required by an SLO.
type PromSLORules struct {
	SLIErrorRecRules PromRuleGroup
	MetadataRecRules PromRuleGroup
	AlertRules       PromRuleGroup
}

// PromRuleGroup are regular prometheus group of rules.
type PromRuleGroup struct {
	Interval time.Duration
	Rules    []rulefmt.Rule
}
