package k8sprometheus

import (
	"github.com/go-playground/validator/v10"

	"github.com/slok/sloth/internal/prometheus"
)

// K8sMeta is the Kubernetes metadata simplified.
type K8sMeta struct {
	Name        string `validate:"required"`
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
}

// SLOGroup is a Kubernetes SLO group. Is created based on a regular Prometheus
// SLO model and Kubernetes data.
type SLOGroup struct {
	K8sMeta K8sMeta
	SLOs    []prometheus.SLO
}

// Validate validates the SLO.
func (s SLOGroup) Validate() error {
	err := modelSpecValidate.Struct(s)
	if err != nil {
		return err
	}

	for _, slo := range s.SLOs {
		err := slo.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

var modelSpecValidate = func() *validator.Validate {
	return validator.New()
}()
