package storage

import "github.com/slok/sloth/pkg/common/model"

// K8sMeta is the Kubernetes metadata simplified used for storage purposes.
type K8sMeta struct {
	Kind        string `validate:"required"`
	APIVersion  string `validate:"required"`
	Name        string `validate:"required"`
	UID         string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
}

// SLORulesResult is a common type used to store final SLO rules result in batches.
type SLORulesResult struct {
	SLO   model.PromSLO
	Rules model.PromSLORules
}
