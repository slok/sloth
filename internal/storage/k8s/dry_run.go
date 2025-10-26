package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/common/model"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
)

type DryRunApiserverRepository struct {
	svc    ApiserverRepository
	logger log.Logger
}

// NewDryRunApiserverRepository returns a new Kubernetes Service that will dry-run that will only do real ReadOnly operations.
func NewDryRunApiserverRepository(svc ApiserverRepository, logger log.Logger) DryRunApiserverRepository {
	return DryRunApiserverRepository{
		svc:    svc,
		logger: logger.WithValues(log.Kv{"service": "storage.k8s.DryRunApiserverRepository"}),
	}
}

func (r DryRunApiserverRepository) ListPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (*slothv1.PrometheusServiceLevelList, error) {
	return r.svc.ListPrometheusServiceLevels(ctx, ns, opts)
}

func (r DryRunApiserverRepository) WatchPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (watch.Interface, error) {
	return r.svc.WatchPrometheusServiceLevels(ctx, ns, opts)
}

func (r DryRunApiserverRepository) EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error {
	r.logger.Infof("Dry run EnsurePrometheusServiceLevelStatus")
	return nil
}

func (r DryRunApiserverRepository) StoreSLOs(ctx context.Context, kmeta model.K8sMeta, slos model.PromSLOGroupResult) error {
	r.logger.Infof("Dry run StoreSLOs")
	return nil
}
