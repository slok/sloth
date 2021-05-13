package k8sprometheus

import (
	"context"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringclientset "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/slok/sloth/internal/log"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	slothclientset "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned"
)

type KubernetesService struct {
	slothCli      slothclientset.Interface
	monitoringCli monitoringclientset.Interface
	logger        log.Logger
}

// NewKubernetesService returns a new Kubernetes Service.
func NewKubernetesService(slothCli slothclientset.Interface, monitoringCli monitoringclientset.Interface, logger log.Logger) KubernetesService {
	return KubernetesService{
		slothCli:      slothCli,
		monitoringCli: monitoringCli,
		logger:        logger.WithValues(log.Kv{"service": "k8sprometheus.Service"}),
	}
}

func (k KubernetesService) ListPrometheusServiceLevels(ctx context.Context, ns string, labelSelector map[string]string) (*slothv1.PrometheusServiceLevelList, error) {
	return k.slothCli.SlothV1().PrometheusServiceLevels(ns).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector).String(),
	})
}

func (k KubernetesService) WatchPrometheusServiceLevels(ctx context.Context, ns string, labelSelector map[string]string) (watch.Interface, error) {
	return k.slothCli.SlothV1().PrometheusServiceLevels(ns).Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector).String(),
	})
}

func (k KubernetesService) EnsurePrometheusRule(ctx context.Context, pr *monitoringv1.PrometheusRule) error {
	logger := k.logger.WithCtxValues(ctx)
	pr = pr.DeepCopy()
	stored, err := k.monitoringCli.MonitoringV1().PrometheusRules(pr.Namespace).Get(ctx, pr.Name, metav1.GetOptions{})
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}
		_, err = k.monitoringCli.MonitoringV1().PrometheusRules(pr.Namespace).Create(ctx, pr, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		logger.Debugf("monitoringv1.PrometheusRule has been created")

		return nil
	}

	// Force overwrite.
	pr.ObjectMeta.ResourceVersion = stored.ResourceVersion
	_, err = k.monitoringCli.MonitoringV1().PrometheusRules(pr.Namespace).Update(ctx, pr, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	logger.Debugf("monitoringv1.PrometheusRule has been overwritten")

	return nil
}

func (k KubernetesService) EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error {
	slo = slo.DeepCopy()

	slo.Status.PromOpRulesGenerated = false
	slo.Status.PromOpRulesGeneratedSLOs = 0
	slo.Status.ProcessedSLOs = len(slo.Spec.SLOs)
	slo.Status.LastPromOpRulesGeneration = &metav1.Time{Time: time.Now().UTC()}
	if err == nil {
		slo.Status.PromOpRulesGenerated = true
		slo.Status.PromOpRulesGeneratedSLOs = len(slo.Spec.SLOs)
		slo.Status.LastPromOpRulesSuccessfulGeneration = slo.Status.LastPromOpRulesGeneration
	}

	_, err = k.slothCli.SlothV1().PrometheusServiceLevels(slo.Namespace).UpdateStatus(ctx, slo, metav1.UpdateOptions{})
	return err
}
