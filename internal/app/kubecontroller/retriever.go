package kubecontroller

import (
	"context"

	"github.com/spotahome/kooper/v2/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
)

// RetrieverKubernetesRepository is the service to manage k8s resources by the Kubernetes controller retrievers.
type RetrieverKubernetesRepository interface {
	ListPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (*slothv1.PrometheusServiceLevelList, error)
	WatchPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (watch.Interface, error)
}

// NewPrometheusServiceLevelsRetriver returns the retriever for Prometheus service levels events.
func NewPrometheusServiceLevelsRetriver(ns string, labelSelector labels.Selector, repo RetrieverKubernetesRepository) controller.Retriever {
	return controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = labelSelector.String()
			return repo.ListPrometheusServiceLevels(context.Background(), ns, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = labelSelector.String()
			return repo.WatchPrometheusServiceLevels(context.Background(), ns, options)
		},
	})
}
