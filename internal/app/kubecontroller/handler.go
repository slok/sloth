package kubecontroller

import (
	"context"
	"fmt"

	"github.com/spotahome/kooper/v2/controller"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/sloth/internal/app/generate"
	"github.com/slok/sloth/internal/info"
	"github.com/slok/sloth/internal/k8sprometheus"
	"github.com/slok/sloth/internal/log"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
)

// SpecLoader Knows how to load a Kubernetes Spec into an app model.
type SpecLoader interface {
	LoadSpec(ctx context.Context, spec *slothv1.PrometheusServiceLevel) (*k8sprometheus.SLOGroup, error)
}

// Generator Knows how to generate SLO prometheus rules from app SLO model.
type Generator interface {
	Generate(ctx context.Context, r generate.Request) (*generate.Response, error)
}

// Repository knows how to store generated SLO Prometheus rules.
type Repository interface {
	StoreSLOs(ctx context.Context, kmeta k8sprometheus.K8sMeta, slos []k8sprometheus.StorageSLO) error
}

// KubeStatusStorer knows how to set the status of Prometheus service levels Kubernetes CRD.
type KubeStatusStorer interface {
	EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error
}

// HandlerConfig is the controller handler configuration.
type HandlerConfig struct {
	Generator        Generator
	SpecLoader       SpecLoader
	Repository       Repository
	KubeStatusStorer KubeStatusStorer
	ExtraLabels      map[string]string
	Logger           log.Logger
}

func (c *HandlerConfig) defaults() error {
	if c.Generator == nil {
		return fmt.Errorf("generator is required")
	}

	if c.SpecLoader == nil {
		c.SpecLoader = k8sprometheus.CRSpecLoader
	}

	if c.KubeStatusStorer == nil {
		return fmt.Errorf("kubernetes status storer is required")
	}

	if c.ExtraLabels == nil {
		c.ExtraLabels = map[string]string{}
	}

	if c.Repository == nil {
		return fmt.Errorf("repository is required")
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"service": "kubecontroller.Handler"})

	return nil
}

type handler struct {
	specLoader       SpecLoader
	generator        Generator
	repository       Repository
	kubeStatusStorer KubeStatusStorer
	extraLabels      map[string]string
	logger           log.Logger
}

func NewHandler(config HandlerConfig) (controller.Handler, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return &handler{
		specLoader:       config.SpecLoader,
		generator:        config.Generator,
		repository:       config.Repository,
		kubeStatusStorer: config.KubeStatusStorer,
		extraLabels:      config.ExtraLabels,
		logger:           config.Logger,
	}, nil
}

func (h handler) Handle(ctx context.Context, obj runtime.Object) error {
	switch v := obj.(type) {
	case *slothv1.PrometheusServiceLevel:
		return h.handlePrometheusServiceLevelV1(ctx, v)
	default:
		h.logger.Warningf("Unsuported Kubernetes object type: %s", obj.GetObjectKind())
	}

	return nil
}

func (h handler) handlePrometheusServiceLevelV1(ctx context.Context, psl *slothv1.PrometheusServiceLevel) (err error) {
	ctx = h.logger.SetValuesOnCtx(ctx, log.Kv{"ns": psl.Namespace, "name": psl.Name})
	logger := h.logger.WithCtxValues(ctx)

	// If the received object is being deleted, ignore.
	deleteInProgress := !psl.DeletionTimestamp.IsZero()
	if deleteInProgress {
		logger.Debugf("Received object is in deletion process, ignoring")
		return nil
	}

	// Store the status with the result of the handling process every time we
	// process a CR.
	defer func() {
		storedErr := h.kubeStatusStorer.EnsurePrometheusServiceLevelStatus(ctx, psl, err)
		if storedErr != nil {
			logger.Errorf("Could not set PrometheusServiceLevel CRD status: %s", storedErr)
		}
	}()

	// Load From CRD to model.
	model, err := h.specLoader.LoadSpec(ctx, psl)
	if err != nil {
		return fmt.Errorf("could not load CR spec into model: %w", err)
	}

	// Generate rules.
	req := generate.Request{
		Info: info.Info{
			Version: info.Version,
			Mode:    info.ModeControllerGenKubernetes,
			Spec:    fmt.Sprintf("%s/%s", slothv1.SchemeGroupVersion.Group, slothv1.SchemeGroupVersion.Version),
		},
		ExtraLabels: h.extraLabels,
		SLOGroup:    model.SLOGroup,
	}
	resp, err := h.generator.Generate(ctx, req)
	if err != nil {
		return fmt.Errorf("could not generate SLOs: %w", err)
	}

	// Store on k8s as Prometheus operator Rules.
	storageSLOs := make([]k8sprometheus.StorageSLO, 0, len(resp.PrometheusSLOs))
	for _, s := range resp.PrometheusSLOs {
		storageSLOs = append(storageSLOs, k8sprometheus.StorageSLO{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}
	err = h.repository.StoreSLOs(ctx, model.K8sMeta, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOs: %w", err)
	}

	return nil
}
