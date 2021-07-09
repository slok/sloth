package k8sprometheus

import (
	"context"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringclientset "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	monitoringclientsetfake "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/fake"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/slok/sloth/internal/log"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	slothclientset "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned"
	slothclientsetfake "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned/fake"
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

func (k KubernetesService) ListPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (*slothv1.PrometheusServiceLevelList, error) {
	return k.slothCli.SlothV1().PrometheusServiceLevels(ns).List(ctx, opts)
}

func (k KubernetesService) WatchPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (watch.Interface, error) {
	return k.slothCli.SlothV1().PrometheusServiceLevels(ns).Watch(ctx, opts)
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

// EnsurePrometheusServiceLevelStatus updates the status of a PrometheusServiceLeve, be aware that updating
// an status will trigger a watch update event on a controller.
// In case of no error we will update "last correct Prometheus operation rules generated" TS so we can be in
// a infinite loop of handling, the handler should break this loop somehow (e.g: if ok and last generated < 5m, ignore).
func (k KubernetesService) EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error {
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

	_, err = k.slothCli.SlothV1().PrometheusServiceLevels(slo.Namespace).UpdateStatus(ctx, slo, metav1.UpdateOptions{})
	return err
}

type DryRunKubernetesService struct {
	svc    KubernetesService
	logger log.Logger
}

// NewKubernetesServiceDryRun returns a new Kubernetes Service that will dry-run that will only do real ReadOnly operations.
func NewKubernetesServiceDryRun(svc KubernetesService, logger log.Logger) DryRunKubernetesService {
	return DryRunKubernetesService{
		svc:    svc,
		logger: logger.WithValues(log.Kv{"service": "k8sprometheus.DryRunService"}),
	}
}

func (d DryRunKubernetesService) ListPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (*slothv1.PrometheusServiceLevelList, error) {
	return d.svc.ListPrometheusServiceLevels(ctx, ns, opts)
}

func (d DryRunKubernetesService) WatchPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (watch.Interface, error) {
	return d.svc.WatchPrometheusServiceLevels(ctx, ns, opts)
}

func (d DryRunKubernetesService) EnsurePrometheusRule(ctx context.Context, pr *monitoringv1.PrometheusRule) error {
	d.logger.Infof("Dry run EnsurePrometheusRule")
	return nil
}

func (d DryRunKubernetesService) EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error {
	d.logger.Infof("Dry run EnsurePrometheusServiceLevelStatus")
	return nil
}

type FakeKubernetesService struct {
	ksvc KubernetesService
}

// NewKubernetesServiceFake returns a new Kubernetes Service that will fake Kubernetes operations
// using fake clients.
func NewKubernetesServiceFake(logger log.Logger) FakeKubernetesService {
	return FakeKubernetesService{
		ksvc: NewKubernetesService(
			slothclientsetfake.NewSimpleClientset(prometheusServiceLevelFakes...),
			monitoringclientsetfake.NewSimpleClientset(),
			logger),
	}
}

func (f FakeKubernetesService) ListPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (*slothv1.PrometheusServiceLevelList, error) {
	return f.ksvc.ListPrometheusServiceLevels(ctx, ns, opts)
}

func (f FakeKubernetesService) WatchPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (watch.Interface, error) {
	return f.ksvc.WatchPrometheusServiceLevels(ctx, ns, opts)
}

func (f FakeKubernetesService) EnsurePrometheusRule(ctx context.Context, pr *monitoringv1.PrometheusRule) error {
	return f.ksvc.EnsurePrometheusRule(ctx, pr)
}

func (f FakeKubernetesService) EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error {
	return f.ksvc.EnsurePrometheusServiceLevelStatus(ctx, slo, err)
}

var prometheusServiceLevelFakes = []runtime.Object{
	&slothv1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake01",
			Labels: map[string]string{
				"prometheus": "default",
			},
		},
		Spec: slothv1.PrometheusServiceLevelSpec{
			Service: "svc01",
			Labels: map[string]string{
				"globalk1": "globalv1",
			},
			SLOs: []slothv1.SLO{
				{
					Name:      "slo01",
					Objective: 99.9,
					Labels: map[string]string{
						"slo01k1": "slo01v1",
					},
					SLI: slothv1.SLI{Events: &slothv1.SLIEvents{
						ErrorQuery: `sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))`,
						TotalQuery: `sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))`,
					}},
					Alerting: slothv1.Alerting{
						Name: "myServiceAlert",
						Labels: map[string]string{
							"alert01k1": "alert01v1",
						},
						Annotations: map[string]string{
							"alert02k1": "alert02v1",
						},
						PageAlert:   slothv1.Alert{},
						TicketAlert: slothv1.Alert{},
					},
				},
				{
					Name:      "slo02",
					Objective: 99.99,
					SLI: slothv1.SLI{Raw: &slothv1.SLIRaw{
						ErrorRatioQuery: `
sum(rate(http_request_duration_seconds_count{job="myservice2",code=~"(5..|429)"}[{{.window}}]))
/
sum(rate(http_request_duration_seconds_count{job="myservice2"}[{{.window}}]))
`,
					}},
					Alerting: slothv1.Alerting{
						PageAlert:   slothv1.Alert{Disable: true},
						TicketAlert: slothv1.Alert{Disable: true},
					},
				},
			},
		},
	},
}
