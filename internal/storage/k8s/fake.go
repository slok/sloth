package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/common/model"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	slothclientsetfake "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned/fake"
	plugink8stransformv1 "github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1"
)

type FakeApiserverRepository struct {
	ksvc ApiserverRepository
}

// NewFakeApiserverRepository returns a new Kubernetes Service that will fake Kubernetes operations
// using fake clients.
func NewFakeApiserverRepository(logger log.Logger, k8sTransformPlugin plugink8stransformv1.Plugin) (*FakeApiserverRepository, error) {
	// Setup fake dynamic client with ConfigMap scheme.
	// Important: When adding new k8s transform plugins we need to add their
	// resources to the fake discovery client to be able to fake them.
	dynamicCli := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	fakeDiscovery := &fakediscovery.FakeDiscovery{Fake: &k8stesting.Fake{Resources: []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "configmaps", Namespaced: true, Kind: "ConfigMap"},
			},
		},
		{
			GroupVersion: "monitoring.coreos.com/v1",
			APIResources: []metav1.APIResource{
				{Name: "prometheusrules", Namespaced: true, Kind: "PrometheusRule"},
			},
		},
	}}}

	c, err := NewApiserverRepository(ApiserverRepositoryConfig{
		SlothCli:           slothclientsetfake.NewClientset(prometheusServiceLevelFakes...),
		DynamicCli:         dynamicCli,
		DiscoveryCli:       fakeDiscovery,
		K8sTransformPlugin: k8sTransformPlugin,
		Logger:             logger,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create fake k8s apiserver repository: %w", err)
	}

	return &FakeApiserverRepository{
		ksvc: *c,
	}, nil
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

func (r FakeApiserverRepository) StoreSLOs(ctx context.Context, kmeta model.K8sMeta, slos model.PromSLOGroupResult) error {
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
