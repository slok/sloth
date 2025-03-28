// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/slok/sloth/pkg/kubernetes/gen/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// PrometheusServiceLevels returns a PrometheusServiceLevelInformer.
	PrometheusServiceLevels() PrometheusServiceLevelInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// PrometheusServiceLevels returns a PrometheusServiceLevelInformer.
func (v *version) PrometheusServiceLevels() PrometheusServiceLevelInformer {
	return &prometheusServiceLevelInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
