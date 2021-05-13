package kubecontroller

import (
	"context"

	"github.com/spotahome/kooper/v2/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
)

// RetrieverKubernetesRepository is the service to manage k8s resources by the Kubernetes controller retrievers.
type RetrieverKubernetesRepository interface {
	ListPrometheusServiceLevels(ctx context.Context, ns string, labelSelector map[string]string) (*slothv1.PrometheusServiceLevelList, error)
	WatchPrometheusServiceLevels(ctx context.Context, ns string, labelSelector map[string]string) (watch.Interface, error)
}

// NewPrometheusServiceLevelsRetriver returns the retriever for Prometheus service levels events.
func NewPrometheusServiceLevelsRetriver(ns string, repo RetrieverKubernetesRepository) controller.Retriever {
	return controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return repo.ListPrometheusServiceLevels(context.TODO(), ns, map[string]string{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return repo.WatchPrometheusServiceLevels(context.TODO(), ns, map[string]string{})
		},
	})
}
