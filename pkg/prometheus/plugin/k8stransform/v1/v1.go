package v1

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/slok/sloth/pkg/common/model"
)

// Version is this plugin type version.
const Version = "prometheus/k8stransform/v1"

// PluginVersion is the version of the plugin (e.g: `prometheus/k8stransform/v1`).
type PluginVersion = string

const PluginVersionName = "PluginVersion"

// PluginID is the ID of the plugin (e.g: sloth.dev/my-test-plugin/v1).
type PluginID = string

const PluginIDName = "PluginID"

type K8sObjects struct {
	Items []*unstructured.Unstructured
}

// PluginFactoryName is the required name for the plugin factory.
const PluginFactoryName = "NewPlugin"

type PluginFactory = func() (Plugin, error)

// Plugin knows how to transform K8s objects, these transformers should be simple and
// only focused on transforming K8s objects generated from SLOs.
type Plugin interface {
	TransformK8sObjects(ctx context.Context, kmeta model.K8sMeta, sloResult model.PromSLOGroupResult) (*K8sObjects, error)
}
