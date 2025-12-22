package k8s_test

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/slok/sloth/internal/log"
	pluginenginek8stransform "github.com/slok/sloth/internal/pluginengine/k8stransform"
	storagek8s "github.com/slok/sloth/internal/storage/k8s"
	"github.com/slok/sloth/pkg/common/model"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	slothclientsetfake "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned/fake"
)

var originalSource = model.PromSLOGroupSource{
	K8sSlothV1: &slothv1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-name",
			UID:  types.UID("test-uid"),
		},
	},
}

var testPromSLOGroupResult = model.PromSLOGroupResult{
	OriginalSource: originalSource,
	SLOResults: []model.PromSLOResult{
		{
			SLO: model.PromSLO{ID: "testa"},
			PrometheusRules: model.PromSLORules{
				SLIErrorRecRules: model.PromRuleGroup{
					Name: "sloth-slo-sli-recordings-testa",
					Rules: []rulefmt.Rule{
						{
							Record: "test:record-a1",
							Expr:   "test-expr-a1",
							Labels: map[string]string{"test-label": "a-1"},
						},
						{
							Record: "test:record-a2",
							Expr:   "test-expr-a2",
							Labels: map[string]string{"test-label": "a-2"},
						},
					}},
				MetadataRecRules: model.PromRuleGroup{
					Name: "sloth-slo-meta-recordings-testa",
					Rules: []rulefmt.Rule{
						{
							Record: "test:record-a3",
							Expr:   "test-expr-a3",
							Labels: map[string]string{"test-label": "a-3"},
						},
						{
							Record: "test:record-a4",
							Expr:   "test-expr-a4",
							Labels: map[string]string{"test-label": "a-4"},
						},
					}},
				AlertRules: model.PromRuleGroup{
					Name:     "sloth-slo-alerts-testa",
					Interval: 15 * time.Minute, // Custom interval.
					Rules: []rulefmt.Rule{
						{
							Alert:       "testAlertA1",
							Expr:        "test-expr-a1",
							Labels:      map[string]string{"test-label": "a-1"},
							Annotations: map[string]string{"test-annot": "a-1"},
						},
						{
							Alert:       "testAlertA2",
							Expr:        "test-expr-a2",
							Labels:      map[string]string{"test-label": "a-2"},
							Annotations: map[string]string{"test-annot": "a-2"},
						},
					}},
			},
		},
		{
			SLO: model.PromSLO{ID: "testb"},
			PrometheusRules: model.PromSLORules{
				SLIErrorRecRules: model.PromRuleGroup{
					Name: "sloth-slo-sli-recordings-testb",
					Rules: []rulefmt.Rule{
						{
							Record: "test:record-b1",
							Expr:   "test-expr-b1",
							Labels: map[string]string{"test-label": "b-1"},
						},
					}},
				MetadataRecRules: model.PromRuleGroup{
					Name: "sloth-slo-meta-recordings-testb",
					Rules: []rulefmt.Rule{
						{
							Record: "test:record-b2",
							Expr:   "test-expr-b2",
							Labels: map[string]string{"test-label": "b-2"},
						},
					}},
				AlertRules: model.PromRuleGroup{
					Name: "sloth-slo-alerts-testb",
					Rules: []rulefmt.Rule{
						{
							Alert:       "testAlertB1",
							Expr:        "test-expr-b1",
							Labels:      map[string]string{"test-label": "b-1"},
							Annotations: map[string]string{"test-annot": "b-1"},
						},
					}},
				ExtraRules: []model.PromRuleGroup{
					{
						Name:     "sloth-slo-extra-rules-000-testb",
						Interval: 42 * time.Minute,
						Rules: []rulefmt.Rule{
							{
								Alert:       "testAlertZ1",
								Expr:        "test-expr-z1",
								Labels:      map[string]string{"test-label": "z-1"},
								Annotations: map[string]string{"test-annot": "z-1"},
							},
						}},
					{}, // Should be skipped.
					{
						Name: "sloth-slo-extra-rules-001-testb",
						Rules: []rulefmt.Rule{
							{
								Alert:       "testAlertZ2",
								Expr:        "test-expr-z2",
								Labels:      map[string]string{"test-label": "z-2"},
								Annotations: map[string]string{"test-annot": "z-2"},
							},
							{
								Alert:       "testAlertZ3",
								Expr:        "test-expr-z3",
								Labels:      map[string]string{"test-label": "z-3"},
								Annotations: map[string]string{"test-annot": "z-3"},
							},
						},
					},
				},
			},
		},
	}}

func TestApiserverRepositoryStoreSLOsWithK8sTransformPlugins(t *testing.T) {
	// Create a simple k8s transform plugin that creates configmaps as unstructured objects.
	pluginSrc := `package plugin

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/slok/sloth/pkg/common/model"
	plugink8stransformv1 "github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1"
)

const (
	PluginVersion = "prometheus/k8stransform/v1"
	PluginID      = "sloth.dev/test-transform-test/v1"
)

func NewPlugin() (plugink8stransformv1.Plugin, error) {
	return testPlugin{}, nil
}

type testPlugin struct{}

type m map[string]any

func (testPlugin) TransformK8sObjects(ctx context.Context, kmeta model.K8sMeta, sloResult model.PromSLOGroupResult) (*plugink8stransformv1.K8sObjects, error) {
	objs := []*unstructured.Unstructured{}

	// Create a ConfigMap for each SLO.
	for _, slo := range sloResult.SLOResults {
		// Convert labels and annotations to map[string]interface{}.
		labels := m{}
		for k, v := range kmeta.Labels {
			labels[k] = v
		}
		annotations := m{}
		for k, v := range kmeta.Annotations {
			annotations[k] = v
		}

		obj := &unstructured.Unstructured{
			Object: m{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": m{
					"name":        kmeta.Name + "-" + slo.SLO.ID,
					"namespace":   kmeta.Namespace,
					"labels":      labels,
					"annotations": annotations,
				},
				"data": m{
					"slo_id": slo.SLO.ID,
				},
			},
		}
		objs = append(objs, obj)
	}

	return &plugink8stransformv1.K8sObjects{
		Items: objs,
	}, nil
}`

	tests := map[string]struct {
		k8sMeta model.K8sMeta
		slos    model.PromSLOGroupResult
		expObjs []unstructured.Unstructured
		expErr  bool
	}{
		"Having multiple SLOs should create unstructured objects correctly.": {
			k8sMeta: model.K8sMeta{
				Name:        "test-name",
				Namespace:   "test-ns",
				Labels:      map[string]string{"lk1": "lv1"},
				Annotations: map[string]string{"ak1": "av1"},
			},
			slos: testPromSLOGroupResult,
			expObjs: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "ConfigMap",
						"metadata": map[string]interface{}{
							"name":      "test-name-testa",
							"namespace": "test-ns",
							"labels": map[string]interface{}{
								"lk1":                          "lv1",
								"app.kubernetes.io/component":  "SLO",
								"app.kubernetes.io/managed-by": "sloth",
							},
							"annotations": map[string]interface{}{
								"ak1": "av1",
							},
							"ownerReferences": []interface{}{
								map[string]interface{}{
									"apiVersion": "sloth.slok.dev/v1",
									"kind":       "PrometheusServiceLevel",
									"name":       "test-name",
									"uid":        "test-uid",
								},
							},
						},
						"data": map[string]interface{}{
							"slo_id": "testa",
						},
					},
				},
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "ConfigMap",
						"metadata": map[string]interface{}{
							"name":      "test-name-testb",
							"namespace": "test-ns",
							"labels": map[string]interface{}{
								"lk1":                          "lv1",
								"app.kubernetes.io/component":  "SLO",
								"app.kubernetes.io/managed-by": "sloth",
							},
							"annotations": map[string]interface{}{
								"ak1": "av1",
							},
							"ownerReferences": []interface{}{
								map[string]interface{}{
									"apiVersion": "sloth.slok.dev/v1",
									"kind":       "PrometheusServiceLevel",
									"name":       "test-name",
									"uid":        "test-uid",
								},
							},
						},
						"data": map[string]interface{}{
							"slo_id": "testb",
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Load the k8s transform plugin.
			pluginFactory, err := pluginenginek8stransform.PluginLoader.LoadRawPlugin(context.TODO(), pluginSrc)
			require.NoError(err)

			plugin, err := pluginFactory.PluginK8sTransformV1()
			require.NoError(err)

			// Create fake clients.
			slothCLI := slothclientsetfake.NewClientset()

			// Setup fake dynamic client with ConfigMap scheme.
			scheme := runtime.NewScheme()
			err = corev1.AddToScheme(scheme)
			require.NoError(err)
			dynamicCli := fakedynamic.NewSimpleDynamicClient(scheme)
			fakeDiscovery := &fakediscovery.FakeDiscovery{Fake: &k8stesting.Fake{Resources: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{{Name: "configmaps", Namespaced: true, Kind: "ConfigMap"}},
				},
			}}}

			repo, err := storagek8s.NewApiserverRepository(storagek8s.ApiserverRepositoryConfig{
				SlothCli:           slothCLI,
				DynamicCli:         dynamicCli,
				DiscoveryCli:       fakeDiscovery,
				K8sTransformPlugin: plugin,
				Logger:             log.Noop,
			})
			require.NoError(err)

			err = repo.StoreSLOs(context.TODO(), test.k8sMeta, test.slos)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				gotObjs, err := dynamicCli.Resource(corev1.SchemeGroupVersion.WithResource("configmaps")).Namespace("").List(context.TODO(), metav1.ListOptions{})
				require.NoError(err)
				assert.Equal(test.expObjs, gotObjs.Items)
			}
		})
	}
}
