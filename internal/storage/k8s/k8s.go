package k8s

import (
	"context"
	"fmt"
	"time"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/common/model"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	slothclientset "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned"
	plugink8stransformv1 "github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1"
)

type ApiserverRepositoryConfig struct {
	SlothCli           slothclientset.Interface
	DynamicCli         dynamic.Interface
	DiscoveryCli       discovery.DiscoveryInterface
	K8sTransformPlugin plugink8stransformv1.Plugin
	Logger             log.Logger
}

type ApiserverRepository struct {
	slothCli     slothclientset.Interface
	dynamicCli   dynamic.Interface
	restMapper   meta.RESTMapper
	k8sTransform plugink8stransformv1.Plugin
	logger       log.Logger
}

func (c *ApiserverRepositoryConfig) defaults() error {
	if c.SlothCli == nil {
		return fmt.Errorf("SlothCli must be set")
	}

	if c.K8sTransformPlugin == nil {
		return fmt.Errorf("k8s transform plugin must be set")
	}

	if c.DynamicCli == nil {
		return fmt.Errorf("dynamic CLI must be set")
	}

	if c.DiscoveryCli == nil {
		return fmt.Errorf("discovery CLI must be set")
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}

	return nil
}

// NewApiserverRepository returns a new Kubernetes Apiserver storage.
func NewApiserverRepository(config ApiserverRepositoryConfig) (*ApiserverRepository, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	apiGroupResources, err := restmapper.GetAPIGroupResources(config.DiscoveryCli)
	if err != nil {
		return nil, fmt.Errorf("failed to get API group resources: %w", err)
	}

	return &ApiserverRepository{
		slothCli:     config.SlothCli,
		dynamicCli:   config.DynamicCli,
		k8sTransform: config.K8sTransformPlugin,
		restMapper:   restmapper.NewDiscoveryRESTMapper(apiGroupResources),
		logger:       config.Logger.WithValues(log.Kv{"service": "storage.k8s."}),
	}, nil

}

func (r ApiserverRepository) ListPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (*slothv1.PrometheusServiceLevelList, error) {
	return r.slothCli.SlothV1().PrometheusServiceLevels(ns).List(ctx, opts)
}

func (r ApiserverRepository) WatchPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (watch.Interface, error) {
	return r.slothCli.SlothV1().PrometheusServiceLevels(ns).Watch(ctx, opts)
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

func (r ApiserverRepository) StoreSLOs(ctx context.Context, kmeta model.K8sMeta, slos model.PromSLOGroupResult) error {
	ownRef := metav1.OwnerReference{
		Kind:       "PrometheusServiceLevel",
		APIVersion: "sloth.slok.dev/v1",
		Name:       slos.OriginalSource.K8sSlothV1.Name,
		UID:        slos.OriginalSource.K8sSlothV1.UID,
	}

	// Transform to k8s objects.
	k8sObjs, err := r.k8sTransform.TransformK8sObjects(ctx, kmeta, slos)
	if err != nil {
		return fmt.Errorf("could not transform to k8s objects: %w", err)
	}

	// Add object reference.
	for _, obj := range k8sObjs.Items {
		ownRefs := obj.GetOwnerReferences()
		ownRefs = append(ownRefs, ownRef)
		obj.SetOwnerReferences(ownRefs)
	}

	// Ensure k8s objects.
	err = r.ensureK8sObjects(ctx, k8sObjs.Items)
	if err != nil {
		return fmt.Errorf("could not ensure k8s objects: %w", err)
	}
	return nil
}

func (r ApiserverRepository) ensureK8sObjects(ctx context.Context, objs []*unstructured.Unstructured) error {
	logger := r.logger.WithCtxValues(ctx)

	for _, obj := range objs {
		obj := obj.DeepCopy()

		l := obj.GetLabels()
		if l == nil {
			l = map[string]string{}
		}
		l["app.kubernetes.io/component"] = "SLO"
		l["app.kubernetes.io/managed-by"] = "sloth"
		obj.SetLabels(l)

		gvk := obj.GroupVersionKind()
		mapping, err := r.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("could not get GVR for GVK %v: %w", gvk, err)
		}

		// With the GVR get the proper cli.
		gvr := mapping.Resource
		namespace := obj.GetNamespace()
		var resource dynamic.ResourceInterface
		if namespace != "" {
			resource = r.dynamicCli.Resource(gvr).Namespace(namespace)
		} else {
			resource = r.dynamicCli.Resource(gvr)
		}

		// Try setting the resources (Create or override).
		name := obj.GetName()
		stored, err := resource.Get(ctx, name, metav1.GetOptions{})
		logger = logger.WithValues(log.Kv{"gvr": gvr.String(), "namespace": namespace, "name": name})

		if err != nil {
			if !kubeerrors.IsNotFound(err) {
				return fmt.Errorf("could not get object %s/%s: %w", namespace, name, err)
			}
			_, err = resource.Create(ctx, obj, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("could not create object %s/%s: %w", namespace, name, err)
			}
			logger.Debugf("Resource has been created")
			continue
		}

		obj.SetResourceVersion(stored.GetResourceVersion())
		_, err = resource.Update(ctx, obj, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("could not update object %s/%s: %w", namespace, name, err)
		}

		logger.Debugf("Resource has been overwritten")
	}

	return nil
}
