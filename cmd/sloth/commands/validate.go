package commands

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"time"

	prometheusmodel "github.com/prometheus/common/model"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/k8sprometheus"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/openslo"
	"github.com/slok/sloth/internal/prometheus"
)

type validateCommand struct {
	slosInput            string
	slosExcludeRegex     string
	slosIncludeRegex     string
	extraLabels          map[string]string
	sliPluginsPaths      []string
	sloPeriodWindowsPath string
	sloPeriod            string
}

// NewValidateCommand returns the validate command.
func NewValidateCommand(app *kingpin.Application) Command {
	c := &validateCommand{extraLabels: map[string]string{}}
	cmd := app.Command("validate", "Validates the SLO manifests and generation of Prometheus SLOs.")
	cmd.Flag("input", "SLO spec discovery path, will discover recursively all YAML files.").Short('i').Required().StringVar(&c.slosInput)
	cmd.Flag("fs-exclude", "Filter regex to ignore matched discovered SLO file paths.").Short('e').StringVar(&c.slosExcludeRegex)
	cmd.Flag("fs-include", "Filter regex to include matched discovered SLO file paths, everything else will be ignored. Exclude has preference.").Short('n').StringVar(&c.slosIncludeRegex)
	cmd.Flag("extra-labels", "Extra labels that will be added to all the generated Prometheus rules ('key=value' form, can be repeated).").Short('l').StringMapVar(&c.extraLabels)
	cmd.Flag("sli-plugins-path", "The path to SLI plugins (can be repeated), if not set it disable plugins support.").Short('p').StringsVar(&c.sliPluginsPaths)
	cmd.Flag("slo-period-windows-path", "The directory path to custom SLO period windows catalog (replaces default ones).").StringVar(&c.sloPeriodWindowsPath)
	cmd.Flag("default-slo-period", "The default SLO period windows to be used for the SLOs.").Default("30d").StringVar(&c.sloPeriod)

	return c
}

func (v validateCommand) Name() string { return "validate" }
func (v validateCommand) Run(ctx context.Context, config RootConfig) error {
	logger := config.Logger.WithValues(log.Kv{"window": v.sloPeriod})

	// SLO period.
	sp, err := prometheusmodel.ParseDuration(v.sloPeriod)
	if err != nil {
		return fmt.Errorf("invalid SLO period duration: %w", err)
	}
	sloPeriod := time.Duration(sp)

	// Set up files discovery filter regex.
	var excludeRegex *regexp.Regexp
	var includeRegex *regexp.Regexp
	if v.slosExcludeRegex != "" {
		r, err := regexp.Compile(v.slosExcludeRegex)
		if err != nil {
			return fmt.Errorf("invalid exclude regex: %w", err)
		}
		excludeRegex = r
	}
	if v.slosIncludeRegex != "" {
		r, err := regexp.Compile(v.slosIncludeRegex)
		if err != nil {
			return fmt.Errorf("invalid include regex: %w", err)
		}
		includeRegex = r
	}

	// Discover SLOs.
	sloPaths, err := discoverSLOManifests(logger, excludeRegex, includeRegex, v.slosInput)
	if err != nil {
		return fmt.Errorf("could not discover files: %w", err)
	}
	if len(sloPaths) == 0 {
		return fmt.Errorf("0 slo specs have been discovered")
	}

	// Load plugins.
	pluginRepo, err := createPluginLoader(ctx, logger, v.sliPluginsPaths)
	if err != nil {
		return err
	}

	// Windows repository.
	var wfs fs.FS
	if v.sloPeriodWindowsPath != "" {
		wfs = os.DirFS(v.sloPeriodWindowsPath)
	}
	windowsRepo, err := alert.NewFSWindowsRepo(alert.FSWindowsRepoConfig{
		FS:     wfs,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("could not load SLO period windows repository: %w", err)
	}

	// Check if the default slo period is supported by our windows repo.
	_, err = windowsRepo.GetWindows(ctx, sloPeriod)
	if err != nil {
		return fmt.Errorf("invalid default slo period: %w", err)
	}

	// Create Spec loaders.
	promYAMLLoader := prometheus.NewYAMLSpecLoader(pluginRepo, sloPeriod)
	kubeYAMLLoader := k8sprometheus.NewYAMLSpecLoader(pluginRepo, sloPeriod)
	openSLOYAMLLoader := openslo.NewYAMLSpecLoader(sloPeriod)

	// For every file load the data and start the validation process:
	validations := []*fileValidation{}
	totalValidations := 0
	for _, input := range sloPaths {
		// Get SLO spec data.
		slxData, err := os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("could not read SLOs spec file data: %w", err)
		}

		// Split YAMLs in case we have multiple yaml files in a single file.
		splittedSLOsData := splitYAML(slxData)

		gen := generator{
			logger:      log.Noop,
			windowsRepo: windowsRepo,
			extraLabels: v.extraLabels,
		}

		// Prepare file validation result and start validation result for every SLO in the file.
		// TODO(slok): Add service meta to validation.
		validation := &fileValidation{File: input}
		validations = append(validations, validation)
		for _, data := range splittedSLOsData {
			totalValidations++

			dataB := []byte(data)
			// Match the spec type to know how to validate.
			switch {
			case promYAMLLoader.IsSpecType(ctx, dataB):
				slos, promErr := promYAMLLoader.LoadSpec(ctx, dataB)
				if promErr == nil {
					err := gen.GeneratePrometheus(ctx, *slos, io.Discard)
					if err != nil {
						validation.Errs = []error{fmt.Errorf("Could not generate Prometheus format rules: %w", err)}
					}
					continue
				}

				validation.Errs = []error{fmt.Errorf("Tried loading raw prometheus SLOs spec, it couldn't: %w", promErr)}

			case kubeYAMLLoader.IsSpecType(ctx, dataB):
				sloGroup, k8sErr := kubeYAMLLoader.LoadSpec(ctx, dataB)
				if k8sErr == nil {
					err := gen.GenerateKubernetes(ctx, *sloGroup, io.Discard)
					if err != nil {
						validation.Errs = []error{fmt.Errorf("could not generate Kubernetes format rules: %w", err)}
					}
					continue
				}

				validation.Errs = []error{fmt.Errorf("Tried loading Kubernetes prometheus SLOs spec, it couldn't: %w", k8sErr)}

			case openSLOYAMLLoader.IsSpecType(ctx, dataB):
				slos, openSLOErr := openSLOYAMLLoader.LoadSpec(ctx, dataB)
				if openSLOErr == nil {
					err := gen.GenerateOpenSLO(ctx, *slos, io.Discard)
					if err != nil {
						validation.Errs = []error{fmt.Errorf("Could not generate OpenSLO format rules: %w", err)}
					}
					continue
				}

				validation.Errs = []error{fmt.Errorf("Tried loading OpenSLO SLOs spec, it couldn't: %s", openSLOErr)}

			default:
				validation.Errs = []error{fmt.Errorf("Unknown spec type")}
			}
		}

		// Don't wait until the end to show validation per file.
		logger := logger.WithValues(log.Kv{"file": validation.File})
		logger.Debugf("File validated")
		for _, err := range validation.Errs {
			logger.Errorf("%s", err)
		}
	}

	// Check if we need to return an error.
	for _, v := range validations {
		if len(v.Errs) != 0 {
			return fmt.Errorf("validation failed")
		}
	}

	logger.WithValues(log.Kv{"slo-specs": totalValidations}).Infof("Validation succeeded")
	return nil
}

type fileValidation struct {
	File string
	Errs []error
}
