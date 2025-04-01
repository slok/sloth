package k8s

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

func (r ApiserverRepository) EnsurePrometheusRule(ctx context.Context, pr *monitoringv1.PrometheusRule) error {
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

func (r DryRunApiserverRepository) EnsurePrometheusRule(ctx context.Context, pr *monitoringv1.PrometheusRule) error {
	r.logger.Infof("Dry run EnsurePrometheusRule")
	return nil
}

func (r DryRunApiserverRepository) EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error {
	r.logger.Infof("Dry run EnsurePrometheusServiceLevelStatus")
	return nil
}

type FakeApiserverRepository struct {
	ksvc ApiserverRepository
}

// NewFakeApiserverRepository returns a new Kubernetes Service that will fake Kubernetes operations
// using fake clients.
func NewFakeApiserverRepository(logger log.Logger) FakeApiserverRepository {
	return FakeApiserverRepository{
		ksvc: NewApiserverRepository(
			slothclientsetfake.NewSimpleClientset(prometheusServiceLevelFakes...),
			monitoringclientsetfake.NewSimpleClientset(),
			logger),
	}
}

func (r FakeApiserverRepository) ListPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (*slothv1.PrometheusServiceLevelList, error) {
	return r.ksvc.ListPrometheusServiceLevels(ctx, ns, opts)
}

func (r FakeApiserverRepository) WatchPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (watch.Interface, error) {
	return r.ksvc.WatchPrometheusServiceLevels(ctx, ns, opts)
}

func (r FakeApiserverRepository) EnsurePrometheusRule(ctx context.Context, pr *monitoringv1.PrometheusRule) error {
	return r.ksvc.EnsurePrometheusRule(ctx, pr)
}

func (r FakeApiserverRepository) EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error {
	return r.ksvc.EnsurePrometheusServiceLevelStatus(ctx, slo, err)
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
