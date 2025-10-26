package k8s

import (
	"context"
	"fmt"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringclientset "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	kubernetesmodelmap "github.com/slok/sloth/internal/kubernetes/modelmap"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/common/model"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	slothclientset "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned"
)

type ApiserverRepository struct {
	slothCli      slothclientset.Interface
	monitoringCli monitoringclientset.Interface
	logger        log.Logger
}

// NewApiserverRepository returns a new Kubernetes Apiserver storage.
func NewApiserverRepository(slothCli slothclientset.Interface, monitoringCli monitoringclientset.Interface, logger log.Logger) ApiserverRepository {
	return ApiserverRepository{
		slothCli:      slothCli,
		monitoringCli: monitoringCli,
		logger:        logger.WithValues(log.Kv{"service": "storage.k8s."}),
	}
}

func (r ApiserverRepository) ListPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (*slothv1.PrometheusServiceLevelList, error) {
	return r.slothCli.SlothV1().PrometheusServiceLevels(ns).List(ctx, opts)
}

func (r ApiserverRepository) WatchPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (watch.Interface, error) {
	return r.slothCli.SlothV1().PrometheusServiceLevels(ns).Watch(ctx, opts)
}

// EnsurePrometheusServiceLevelStatus updates the status of a PrometheusServiceLeve, be aware that updating
// an status will trigger a watch update event on a controller.
// In case of no error we will update "last correct Prometheus operation rules generated" TS so we can be in
// a infinite loop of handling, the handler should break this loop somehow (e.g: if ok and last generated < 5m, ignore).
func (r ApiserverRepository) EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error {
	slo = slo.DeepCopy()

	slo.Status.PromOpRulesGenerated = false
	slo.Status.PromOpRulesGeneratedSLOs = 0
	slo.Status.ProcessedSLOs = len(slo.Spec.SLOs)
	slo.Status.ObservedGeneration = slo.Generation

	if err == nil {
		slo.Status.PromOpRulesGenerated = true
		slo.Status.PromOpRulesGeneratedSLOs = len(slo.Spec.SLOs)
		slo.Status.LastPromOpRulesSuccessfulGenerated = &metav1.Time{Time: time.Now().UTC()}
	}

	_, err = r.slothCli.SlothV1().PrometheusServiceLevels(slo.Namespace).UpdateStatus(ctx, slo, metav1.UpdateOptions{})
	return err
}

func (r ApiserverRepository) StoreSLOs(ctx context.Context, kmeta model.K8sMeta, slos model.PromSLOGroupResult) error {
	// Map to the Prometheus operator CRD.
	rule, err := kubernetesmodelmap.MapModelToPrometheusOperator(ctx, kmeta, slos)
	if err != nil {
		return fmt.Errorf("could not map model to Prometheus operator CR: %w", err)
	}

	// Add object reference.
	rule.ObjectMeta.OwnerReferences = append(rule.ObjectMeta.OwnerReferences, metav1.OwnerReference{
		Kind:       "PrometheusServiceLevel",
		APIVersion: "sloth.slok.dev/v1",
		Name:       slos.OriginalSource.K8sSlothV1.Name,
		UID:        slos.OriginalSource.K8sSlothV1.UID,
	})

	// Create on API server.
	err = r.ensurePrometheusRule(ctx, rule)
	if err != nil {
		return fmt.Errorf("could not ensure Prometheus operator rule CR: %w", err)
	}

	return nil
}

func (r ApiserverRepository) ensurePrometheusRule(ctx context.Context, pr *monitoringv1.PrometheusRule) error {
	logger := r.logger.WithCtxValues(ctx)
	pr = pr.DeepCopy()
	stored, err := r.monitoringCli.MonitoringV1().PrometheusRules(pr.Namespace).Get(ctx, pr.Name, metav1.GetOptions{})
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}
		_, err = r.monitoringCli.MonitoringV1().PrometheusRules(pr.Namespace).Create(ctx, pr, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		logger.Debugf("monitoringv1.PrometheusRule has been created")

		return nil
	}

	// Force overwrite.
	pr.ObjectMeta.ResourceVersion = stored.ResourceVersion
	_, err = r.monitoringCli.MonitoringV1().PrometheusRules(pr.Namespace).Update(ctx, pr, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	logger.Debugf("monitoringv1.PrometheusRule has been overwritten")

	return nil
}
