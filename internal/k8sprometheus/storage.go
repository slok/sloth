package k8sprometheus

import (
	"context"
	"fmt"
	managedPromAPIV1 "github.com/GoogleCloudPlatform/prometheus-engine/pkg/operator/apis/monitoring/v1"
	promOpAPIV1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/slok/sloth/internal/prometheus"
)

var (
	// ErrNoSLORules will be used when there are no rules to store. The upper layer
	// could ignore or handle the error in cases where there wasn't an output.
	ErrNoSLORules = fmt.Errorf("0 SLO Prometheus rules generated")
)

type StorageSLO struct {
	SLO   prometheus.SLO
	Rules prometheus.SLORules
}

//go:generate mockery --case underscore --output k8sprometheusmock --outpkg k8sprometheusmock --name PrometheusRulesEnsurer

type PrometheusRulesEnsurer interface {
	EnsurePrometheusRule(ctx context.Context, pr *promOpAPIV1.PrometheusRule) error
	EnsureManagedPrometheusRule(ctx context.Context, pr *managedPromAPIV1.Rules) error
}
