// Code generated by 'yaegi extract github.com/slok/sloth/pkg/common/validation'. DO NOT EDIT.

package custom

import (
	"github.com/slok/sloth/pkg/common/validation"
	"reflect"
)

func init() {
	Symbols["github.com/slok/sloth/pkg/common/validation/validation"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"PromQLDialectValidator": reflect.ValueOf(validation.PromQLDialectValidator),
		"ValidateSLO":            reflect.ValueOf(validation.ValidateSLO),

		// type definitions
		"SLODialectValidator": reflect.ValueOf((*validation.SLODialectValidator)(nil)),

		// interface wrapper definitions
		"_SLODialectValidator": reflect.ValueOf((*_github_com_slok_sloth_pkg_common_validation_SLODialectValidator)(nil)),
	}
}

// _github_com_slok_sloth_pkg_common_validation_SLODialectValidator is an interface wrapper for SLODialectValidator type
type _github_com_slok_sloth_pkg_common_validation_SLODialectValidator struct {
	IValue                   interface{}
	WValidateAnnotationKey   func(k string) error
	WValidateAnnotationValue func(k string) error
	WValidateLabelKey        func(k string) error
	WValidateLabelValue      func(k string) error
	WValidateQueryExpression func(queryExpression string) error
}

func (W _github_com_slok_sloth_pkg_common_validation_SLODialectValidator) ValidateAnnotationKey(k string) error {
	return W.WValidateAnnotationKey(k)
}
func (W _github_com_slok_sloth_pkg_common_validation_SLODialectValidator) ValidateAnnotationValue(k string) error {
	return W.WValidateAnnotationValue(k)
}
func (W _github_com_slok_sloth_pkg_common_validation_SLODialectValidator) ValidateLabelKey(k string) error {
	return W.WValidateLabelKey(k)
}
func (W _github_com_slok_sloth_pkg_common_validation_SLODialectValidator) ValidateLabelValue(k string) error {
	return W.WValidateLabelValue(k)
}
func (W _github_com_slok_sloth_pkg_common_validation_SLODialectValidator) ValidateQueryExpression(queryExpression string) error {
	return W.WValidateQueryExpression(queryExpression)
}
