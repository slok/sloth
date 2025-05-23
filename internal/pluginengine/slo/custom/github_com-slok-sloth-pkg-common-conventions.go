// Code generated by 'yaegi extract github.com/slok/sloth/pkg/common/conventions'. DO NOT EDIT.

package custom

import (
	"github.com/slok/sloth/pkg/common/conventions"
	"go/constant"
	"go/token"
	"reflect"
)

func init() {
	Symbols["github.com/slok/sloth/pkg/common/conventions/conventions"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"GetSLIErrorMetric":         reflect.ValueOf(conventions.GetSLIErrorMetric),
		"GetSLOIDPromLabels":        reflect.ValueOf(conventions.GetSLOIDPromLabels),
		"NameRegexp":                reflect.ValueOf(&conventions.NameRegexp).Elem(),
		"PromSLIErrorMetricFmt":     reflect.ValueOf(constant.MakeFromLiteral("\"slo:sli_error:ratio_rate%s\"", token.STRING, 0)),
		"PromSLOIDLabelName":        reflect.ValueOf(constant.MakeFromLiteral("\"sloth_id\"", token.STRING, 0)),
		"PromSLOModeLabelName":      reflect.ValueOf(constant.MakeFromLiteral("\"sloth_mode\"", token.STRING, 0)),
		"PromSLONameLabelName":      reflect.ValueOf(constant.MakeFromLiteral("\"sloth_slo\"", token.STRING, 0)),
		"PromSLOObjectiveLabelName": reflect.ValueOf(constant.MakeFromLiteral("\"sloth_objective\"", token.STRING, 0)),
		"PromSLOServiceLabelName":   reflect.ValueOf(constant.MakeFromLiteral("\"sloth_service\"", token.STRING, 0)),
		"PromSLOSeverityLabelName":  reflect.ValueOf(constant.MakeFromLiteral("\"sloth_severity\"", token.STRING, 0)),
		"PromSLOSpecLabelName":      reflect.ValueOf(constant.MakeFromLiteral("\"sloth_spec\"", token.STRING, 0)),
		"PromSLOVersionLabelName":   reflect.ValueOf(constant.MakeFromLiteral("\"sloth_version\"", token.STRING, 0)),
		"PromSLOWindowLabelName":    reflect.ValueOf(constant.MakeFromLiteral("\"sloth_window\"", token.STRING, 0)),
		"TplWindowRegex":            reflect.ValueOf(&conventions.TplWindowRegex).Elem(),
	}
}
