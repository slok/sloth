package lib

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"time"

	openslov1alpha "github.com/OpenSLO/oslo/pkg/manifest/v1alpha"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/app/generate"
	"github.com/slok/sloth/internal/info"
	storagefs "github.com/slok/sloth/internal/storage/fs"
	storageio "github.com/slok/sloth/internal/storage/io"
	"github.com/slok/sloth/pkg/common/model"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
	kubernetesv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	"github.com/slok/sloth/pkg/lib/log"
	prometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
)

// CallerAgent is the agent calling the library.
type CallerAgent string

const (
	// CallerAgentCLI is the caller agent when using sloth as CLI.
	CallerAgentCLI CallerAgent = "cli"
	// CallerAgentCLI is the caller agent when using sloth as API.
	CallerAgentAPI CallerAgent = "api"
)

var allCallerAgents = map[CallerAgent]struct{}{
	CallerAgentCLI: {},
	CallerAgentAPI: {},
}

// PrometheusSLOGenerator is the configuration for the Prometheus SLO generator.
type PrometheusSLOGeneratorConfig struct {
	// WindowsFS is the FS where custom SLO definition period windows exist (When not set default Sloth windows will be used).
	WindowsFS fs.FS
	// PluginsFS are the FSs where custom SLO and SLI plugins exist.
	PluginsFS []fs.FS
	// StrictPlugins makes the plugin loader fail when a plugin can't be loaded.
	StrictPlugins bool
	// DefaultSLOPeriod is the default SLO period to use when not specified in the SLO definition.
	DefaultSLOPeriod time.Duration
	// DisableDefaultPlugins disables the default SLO plugins, normally used along with custom SLO plugins to fully customize Sloth behavior.
	DisableDefaultPlugins bool
	// CMDSLOPlugins are SLO plugins defined at app level, in other words, they will be executed on all SLOs unless these override the SLO plugin chain.
	CMDSLOPlugins []model.PromSLOPluginMetadata
	// ExtraLabels are labels that will be added to all SLOs.
	ExtraLabels map[string]string
	// CallerAgent is the agent calling the library (The identity or form of calling it).
	CallerAgent CallerAgent
	// Logger is the logger to use for the library.
	Logger log.Logger
}

func (c *PrometheusSLOGeneratorConfig) defaults() error {
	if c.DefaultSLOPeriod == 0 {
		c.DefaultSLOPeriod = 30 * 24 * time.Hour // 30 days.
	}

	if c.CallerAgent == "" {
		c.CallerAgent = CallerAgentAPI
	}

	if _, ok := allCallerAgents[c.CallerAgent]; !ok {
		return fmt.Errorf("invalid caller agent: %q", c.CallerAgent)
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}

	return nil
}

// PrometheusSLOGenerator is a Prometheus SLO rules generator from the Sloth supported SLO definitions.
type PrometheusSLOGenerator struct {
	genSvc            generate.Service
	promYAMLLoader    storageio.SlothPrometheusYAMLSpecLoader
	kubeYAMLLoader    storageio.K8sSlothPrometheusYAMLSpecLoader
	openSLOYAMLLoader storageio.OpenSLOYAMLSpecLoader
	pluginsRepo       *storagefs.FilePluginRepo
	extraLabels       map[string]string
	agent             CallerAgent
	logger            log.Logger
}

func NewPrometheusSLOGenerator(config PrometheusSLOGeneratorConfig) (*PrometheusSLOGenerator, error) {
	ctx := context.Background()

	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create alert windows repo.
	windowsRepo, err := alert.NewFSWindowsRepo(alert.FSWindowsRepoConfig{
		FS:     config.WindowsFS,
		Logger: config.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("could not load SLO period windows repository: %w", err)
	}
	_, err = windowsRepo.GetWindows(ctx, config.DefaultSLOPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid default slo period: %w", err)
	}

	// Create plugin repo.
	pluginRepo, err := createPluginLoader(ctx, config.Logger, config.PluginsFS, config.StrictPlugins)
	if err != nil {
		return nil, fmt.Errorf("could not create plugin repository: %w", err)
	}

	defSLOPlugins := []generate.SLOProcessor{}
	if !config.DisableDefaultPlugins {
		// Load default slo plugins.
		defSLOPlugins, err = createDefaultSLOPlugins(config.Logger, false, false)
		if err != nil {
			return nil, fmt.Errorf("could not create default slo plugins: %w", err)
		}
	}

	// Create the final generator service.
	genSvc, err := generate.NewService(generate.ServiceConfig{
		AlertGenerator:  alert.NewGenerator(windowsRepo),
		DefaultPlugins:  defSLOPlugins,
		SLOPluginGetter: pluginRepo,
		ExtraPlugins:    config.CMDSLOPlugins,
		Logger:          config.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create application service: %w", err)
	}

	return &PrometheusSLOGenerator{
		genSvc:            *genSvc,
		promYAMLLoader:    storageio.NewSlothPrometheusYAMLSpecLoader(pluginRepo, config.DefaultSLOPeriod),
		kubeYAMLLoader:    storageio.NewK8sSlothPrometheusYAMLSpecLoader(pluginRepo, config.DefaultSLOPeriod),
		openSLOYAMLLoader: storageio.NewOpenSLOYAMLSpecLoader(config.DefaultSLOPeriod),
		pluginsRepo:       pluginRepo,
		extraLabels:       config.ExtraLabels,
		agent:             config.CallerAgent,
		logger:            config.Logger,
	}, nil
}

// GenerateFromRaw generates SLO rules from raw data, it will infer what type of SLO spec receives. This method is the most
// generic one as the user doesn't need to know what type of SLO spec is using.
// For more custom programmatic usage use the other Go struct API spec generators.
func (p PrometheusSLOGenerator) GenerateFromRaw(ctx context.Context, data []byte) (*model.PromSLOGroupResult, error) {
	// For now we only support yaml specs, so this is safe to do.
	yamlData := utilsdata.SplitYAML(data)
	if len(yamlData) > 1 {
		return nil, fmt.Errorf("multiple specs in the same file are not supported")
	}

	// Match the spec type to know how to generate.
	switch {
	case p.promYAMLLoader.IsSpecType(ctx, data):
		apiSpec, err := p.promYAMLLoader.LoadAPI(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("tried loading raw prometheus SLOs spec, it couldn't: %w", err)
		}

		return p.GenerateFromSlothV1(ctx, *apiSpec)

	case p.kubeYAMLLoader.IsSpecType(ctx, data):
		apiSpec, err := p.kubeYAMLLoader.LoadAPI(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("tried loading Kubernetes prometheus SLOs spec, it couldn't: %w", err)
		}

		return p.GenerateFromK8sV1(ctx, *apiSpec)

	case p.openSLOYAMLLoader.IsSpecType(ctx, data):
		apiSpec, err := p.openSLOYAMLLoader.LoadAPI(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("tried loading OpenSLO SLOs spec, it couldn't: %w", err)
		}

		return p.GenerateFromOpenSLOV1Alpha(ctx, *apiSpec)

	default:
		return nil, fmt.Errorf("invalid spec, could not load with any of the supported spec types")
	}
}

// GenerateFromSlothV1 generates SLOs from a Sloth Prometheus SLO definition spec struct.
func (p PrometheusSLOGenerator) GenerateFromSlothV1(ctx context.Context, spec prometheusv1.Spec) (*model.PromSLOGroupResult, error) {
	spec.Version = prometheusv1.Version // Force version in case is missing(we already know what it is with the type).

	sloGroup, err := p.promYAMLLoader.MapSpecToModel(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	var mode model.Mode
	switch p.agent {
	case CallerAgentCLI:
		mode = model.ModeCLIGenPrometheus
	case CallerAgentAPI:
		mode = model.ModeAPIGenPrometheus
	}

	info := model.Info{
		Version: info.Version,
		Mode:    mode,
		Spec:    prometheusv1.Version,
	}
	req := generate.Request{
		Info:        info,
		ExtraLabels: p.extraLabels,
		SLOGroup:    *sloGroup,
	}

	return p.generateFromModel(ctx, req)
}

// GenerateFromK8sV1 generates SLO rules from a Kubernetes Sloth CR SLO definition spec struct.
func (p PrometheusSLOGenerator) GenerateFromK8sV1(ctx context.Context, spec kubernetesv1.PrometheusServiceLevel) (*model.PromSLOGroupResult, error) {
	sloGroup, err := p.kubeYAMLLoader.MapSpecToModel(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	var mode model.Mode
	switch p.agent {
	case CallerAgentCLI:
		mode = model.ModeCLIGenKubernetes
	case CallerAgentAPI:
		mode = model.ModeAPIGenKubernetes
	}

	info := model.Info{
		Version: info.Version,
		Mode:    mode,
		Spec:    fmt.Sprintf("%s/%s", kubernetesv1.SchemeGroupVersion.Group, kubernetesv1.SchemeGroupVersion.Version),
	}
	req := generate.Request{
		Info:        info,
		ExtraLabels: p.extraLabels,
		SLOGroup:    *sloGroup,
	}

	return p.generateFromModel(ctx, req)
}

// GenerateFromOpenSLOV1Alpha generates SLO rules from an OpenSLO SLO definition spec struct.
func (p PrometheusSLOGenerator) GenerateFromOpenSLOV1Alpha(ctx context.Context, spec openslov1alpha.SLO) (*model.PromSLOGroupResult, error) {
	sloGroup, err := p.openSLOYAMLLoader.MapSpecToModel(spec)
	if err != nil {
		return nil, fmt.Errorf("could not map to model: %w", err)
	}

	var mode model.Mode
	switch p.agent {
	case CallerAgentCLI:
		mode = model.ModeCLIGenOpenSLO
	case CallerAgentAPI:
		mode = model.ModeAPIGenOpenSLO
	}

	info := model.Info{
		Version: info.Version,
		Mode:    mode,
		Spec:    openslov1alpha.APIVersion,
	}
	req := generate.Request{
		Info:        info,
		ExtraLabels: p.extraLabels,
		SLOGroup:    *sloGroup,
	}

	return p.generateFromModel(ctx, req)
}

func (p PrometheusSLOGenerator) generateFromModel(ctx context.Context, req generate.Request) (*model.PromSLOGroupResult, error) {
	res, err := p.genSvc.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not generate Prometheus SLO rules: %w", err)
	}

	result := []model.PromSLOResult{}
	for _, r := range res.PrometheusSLOs {
		result = append(result, model.PromSLOResult{
			SLO:             r.SLO,
			PrometheusRules: r.SLORules,
		})
	}

	return &model.PromSLOGroupResult{
		OriginalSource: req.SLOGroup.OriginalSource,
		SLOResults:     result,
	}, nil
}

// WriteResultAsPrometheusStd writes the SLO results into the writer as a Prometheus standard rules file.
// More information in:
//   - https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#recording-rules.
//   - https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/.
func (p PrometheusSLOGenerator) WriteResultAsPrometheusStd(ctx context.Context, slo model.PromSLOGroupResult, w io.Writer) error {
	repo := storageio.NewStdPrometheusGroupedRulesYAMLRepo(w, p.logger)
	return repo.StoreSLOs(ctx, slo)
}

// WriteResultAsK8sPrometheusOperator writes the SLO results into the writer as a Prometheus Operator CRD file.
// More information in: https://prometheus-operator.dev/docs/api-reference/api/#monitoring.coreos.com/v1.PrometheusRule.
func (p PrometheusSLOGenerator) WriteResultAsK8sPrometheusOperator(ctx context.Context, k8sMeta model.K8sMeta, slo model.PromSLOGroupResult, w io.Writer) error {
	repo := storageio.NewIOWriterPrometheusOperatorYAMLRepo(w, p.logger)
	return repo.StoreSLOs(ctx, k8sMeta, slo)
}
