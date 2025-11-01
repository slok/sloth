package slo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	pluginenginek8stransform "github.com/slok/sloth/internal/pluginengine/k8stransform"
	"github.com/slok/sloth/pkg/common/model"
	plugink8stransformv1 "github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1"
)

func TestPlugin(t *testing.T) {
	tests := map[string]struct {
		pluginSrc  string
		execPlugin func(t *testing.T, p pluginenginek8stransform.Plugin)
		expErr     bool
	}{
		"Empty plugin should fail.": {
			pluginSrc:  "",
			execPlugin: func(t *testing.T, p pluginenginek8stransform.Plugin) {},
			expErr:     true,
		},

		"An invalid plugin syntax should fail": {
			pluginSrc:  `package test{`,
			execPlugin: func(t *testing.T, p pluginenginek8stransform.Plugin) {},
			expErr:     true,
		},

		"A plugin without the required version, should fail.": {
			pluginSrc: `package test
import (
	"context"
	"encoding/json"

	plugink8stransformv1 "github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1"
)

const (
	PluginVersion = "prometheus/k8stransform/v2"
	PluginID      = "sloth.dev/noop/v1"
)

func NewPlugin() (plugink8stransformv1.Plugin, error) {
	return noopPlugin{}, nil
}

type noopPlugin struct{}
func (noopPlugin) TransformK8sObjects(ctx context.Context, kmeta model.K8sMeta, sloResult model.PromSLOGroupResult) (*plugink8stransformv1.K8sObjects, error) {
	return &plugink8stransformv1.K8sObjects{}, nil
}
`,
			execPlugin: func(t *testing.T, p pluginenginek8stransform.Plugin) {},
			expErr:     true,
		},

		"A plugin without the plugin ID, should fail.": {
			pluginSrc: `package test
import (
	"context"
	"encoding/json"

	plugink8stransformv1 "github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1"
)

const (
	PluginVersion = "prometheus/k8stransform/v1"
	PluginID      = ""
)

func NewPlugin() (plugink8stransformv1.Plugin, error) {
	return noopPlugin{}, nil
}

type noopPlugin struct{}

func (noopPlugin) TransformK8sObjects(ctx context.Context, kmeta model.K8sMeta, sloResult model.PromSLOGroupResult) (*plugink8stransformv1.K8sObjects, error) {
	return &plugink8stransformv1.K8sObjects{}, nil
}
`,
			execPlugin: func(t *testing.T, p pluginenginek8stransform.Plugin) {},
			expErr:     true,
		},

		"A plugin without the plugin factory, should fail.": {
			pluginSrc: `package test
import (
	"context"
	"encoding/json"

	plugink8stransformv1 "github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1"
)

const (
	PluginVersion = "prometheus/k8stransform/v1"
	PluginID      = "sloth.dev/noop/v1"
)

func NewPlugin2() (plugink8stransformv1.Plugin, error) {
	return noopPlugin{}, nil
}

type noopPlugin struct{}

func (noopPlugin) TransformK8sObjects(ctx context.Context, kmeta model.K8sMeta, sloResult model.PromSLOGroupResult) (*plugink8stransformv1.K8sObjects, error) {
	return &plugink8stransformv1.K8sObjects{}, nil
}
`,
			execPlugin: func(t *testing.T, p pluginenginek8stransform.Plugin) {},
			expErr:     true,
		},

		"A correct plugin should execute the plugin.": {
			pluginSrc: `package plugin

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/slok/sloth/pkg/common/model"
	plugink8stransformv1 "github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1"
)

const (
	PluginVersion = "prometheus/k8stransform/v1"
	PluginID      = "sloth.dev/test-transform/v1"
)

func NewPlugin() (plugink8stransformv1.Plugin, error) {
	return noopPlugin{}, nil
}

type noopPlugin struct{}

type m map[string]any

func (noopPlugin) TransformK8sObjects(ctx context.Context, kmeta model.K8sMeta, sloResult model.PromSLOGroupResult) (*plugink8stransformv1.K8sObjects, error) {
	objs := []*unstructured.Unstructured{
		{
			Object: m{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": m{
					"name":      sloResult.SLOResults[0].SLO.ID,
					"namespace": "default",
				},
				"spec": m{
					"containers": []m{
						{
							"name":  sloResult.SLOResults[0].SLO.ID,
							"image": "nginx:latest",
						},
					},
				},
			},
		},
		{
			Object: m{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": m{
					"name": sloResult.SLOResults[0].SLO.ID + "-cfg",
					"labels": m{
						"slo": "true",
					},
				},
				"data": m{
					"description": sloResult.SLOResults[0].SLO.Description,
				},
			},
		},
	}
	return &plugink8stransformv1.K8sObjects{
		Items: objs,
	}, nil
}`,
			execPlugin: func(t *testing.T, p pluginenginek8stransform.Plugin) {
				plugin, err := p.PluginK8sTransformV1()
				require.NoError(t, err)

				sloResults := model.PromSLOGroupResult{
					SLOResults: []model.PromSLOResult{
						{SLO: model.PromSLO{
							ID:          "my-slo",
							Description: "My SLO description",
						}},
					},
				}
				kMeta := model.K8sMeta{}
				objs, err := plugin.TransformK8sObjects(t.Context(), kMeta, sloResults)
				require.NoError(t, err)

				expObjs := plugink8stransformv1.K8sObjects{
					Items: []*unstructured.Unstructured{
						{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name":      "my-slo",
									"namespace": "default",
								},
								"spec": map[string]interface{}{
									"containers": []map[string]interface{}{
										{
											"name":  "my-slo",
											"image": "nginx:latest",
										},
									},
								},
							},
						},
						{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "ConfigMap",
								"metadata": map[string]interface{}{
									"name": "my-slo-cfg",
									"labels": map[string]interface{}{
										"slo": "true",
									},
								},
								"data": map[string]interface{}{
									"description": "My SLO description",
								},
							},
						},
					},
				}
				assert.Equal(t, expObjs, *objs)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			plugin, err := pluginenginek8stransform.PluginLoader.LoadRawPlugin(t.Context(), test.pluginSrc)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				test.execPlugin(t, *plugin)
			}
		})
	}
}
