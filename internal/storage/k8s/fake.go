package k8s

import (
	"context"

	monitoringclientsetfake "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/storage"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	slothclientsetfake "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned/fake"
)

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

func (r FakeApiserverRepository) EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error {
	return r.ksvc.EnsurePrometheusServiceLevelStatus(ctx, slo, err)
}

func (r FakeApiserverRepository) StoreSLOs(ctx context.Context, kmeta storage.K8sMeta, slos []storage.SLORulesResult) error {
	return r.ksvc.StoreSLOs(ctx, kmeta, slos)
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
