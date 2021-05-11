package k8sprometheus

import (
	"github.com/go-playground/validator/v10"

	"github.com/slok/sloth/internal/prometheus"
)

// K8sMeta is the Kubernetes metadata simplified.
type K8sMeta struct {
	Kind        string `validate:"required"`
	APIVersion  string `validate:"required"`
	Name        string `validate:"required"`
	UID         string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
}

// SLOGroup is a Kubernetes SLO group. Is created based on a regular Prometheus
// SLO model and Kubernetes data.
type SLOGroup struct {
	K8sMeta K8sMeta
	prometheus.SLOGroup
}

// Validate validates the SLO.
func (s SLOGroup) Validate() error {
	err := modelSpecValidate.Struct(s.K8sMeta)
	if err != nil {
		return err
	}

	err = s.SLOGroup.Validate()
	if err != nil {
		return err
	}

	return nil
}

var modelSpecValidate = func() *validator.Validate {
	return validator.New()
}()
