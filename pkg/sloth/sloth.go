package sloth

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/app/generate"
	"github.com/slok/sloth/internal/info"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/prometheus"
	prometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
)

// SLI reprensents an SLI with custom error and total expressions.
type SLI struct {
	Raw    *SLIRaw
	Events *SLIEvents
}

type SLIRaw struct {
	ErrorRatioQuery string
}

type SLIEvents struct {
	ErrorQuery string
	TotalQuery string
}

// AlertMeta is the metadata of an alert settings.
type AlertMeta struct {
	Disable     bool
	Name        string
	Labels      map[string]string
	Annotations map[string]string
}

// SLO represents a service level objective configuration.
type SLO struct {
	ID              string
	Name            string
	Description     string
	Service         string
	SLI             SLI
	TimeWindow      time.Duration
	Objective       float64
	Labels          map[string]string
	PageAlertMeta   AlertMeta
	TicketAlertMeta AlertMeta
}

type SLOGroup struct {
	SLOs []SLO
}

func ToPrometheusSLOGroup(group SLOGroup) prometheus.SLOGroup {
	slos := make([]prometheus.SLO, 0, len(group.SLOs))
	for _, s := range group.SLOs {
		var (
			sliRaw    *prometheus.SLIRaw
			sliEvents *prometheus.SLIEvents
		)
		if s.SLI.Raw != nil {
			sliRaw = &prometheus.SLIRaw{
				ErrorRatioQuery: s.SLI.Raw.ErrorRatioQuery,
			}
		}
		if s.SLI.Events != nil {
			sliEvents = &prometheus.SLIEvents{
				ErrorQuery: s.SLI.Events.ErrorQuery,
				TotalQuery: s.SLI.Events.TotalQuery,
			}
		}
		slos = append(slos, prometheus.SLO{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			Service:     s.Service,
			SLI: prometheus.SLI{
				Raw:    sliRaw,
				Events: sliEvents,
			},
			TimeWindow:      s.TimeWindow,
			Objective:       s.Objective,
			Labels:          s.Labels,
			PageAlertMeta:   prometheus.AlertMeta(s.PageAlertMeta),
			TicketAlertMeta: prometheus.AlertMeta(s.TicketAlertMeta),
		})
	}
	return prometheus.SLOGroup{
		SLOs: slos,
	}
}

type AlertWindow struct {
	// ErrorBudgetPercent is the error budget % consumed for a full time window.
	// Google gives us some defaults in its SRE workbook that work correctly most of the times:
	// - Page quick:   2%
	// - Page slow:    5%
	// - Ticket quick: 10%
	// - Ticket slow:  10%
	ErrorBudgetPercent float64
	// ShortWindow is the small window used on the alerting part to stop alerting
	// during a long window because we consumed a lot of error budget but the problem
	// is already gone.
	ShortWindow time.Duration
	// LongWindow is the long window used to alert based on the errors happened on that
	// long window.
	LongWindow time.Duration
}

// AlertWindows has the information of the windows for multiwindow-multiburn SLO alerting.
// Its a matrix of values with:
// - Alert severity: ["page", "ticket"].
// - Measuring period: ["long", "short"].
type AlertWindows struct {
	SLOPeriod   time.Duration
	PageQuick   AlertWindow
	PageSlow    AlertWindow
	TicketQuick AlertWindow
	TicketSlow  AlertWindow
}

type AlertWindowRepo interface {
	GetWindows(ctx context.Context, period time.Duration) (*AlertWindows, error)
}

type alertWindowRepoWrapper struct {
	repo AlertWindowRepo
}

func (w alertWindowRepoWrapper) GetWindows(ctx context.Context, period time.Duration) (*alert.Windows, error) {
	windows, err := w.repo.GetWindows(ctx, period)
	if err != nil {
		return nil, err
	}
	return &alert.Windows{
		SLOPeriod:   windows.SLOPeriod,
		PageQuick:   alert.Window(windows.PageQuick),
		PageSlow:    alert.Window(windows.PageSlow),
		TicketQuick: alert.Window(windows.TicketQuick),
		TicketSlow:  alert.Window(windows.TicketSlow),
	}, nil
}

type WindowsConfig struct {
	Windows *AlertWindows
}

func (c WindowsConfig) GetWindows(ctx context.Context, period time.Duration) (*AlertWindows, error) {
	return c.Windows, nil
}

type Logger interface {
	log.Logger
}

var NoOpLogger = log.Noop

func GeneratePrometheus(ctx context.Context, logger Logger, windowsRepo AlertWindowRepo, disableRecs, disableAlerts, disableOptimizedRules bool, extraLabels map[string]string, slos SLOGroup, out io.Writer) error {
	logger.Infof("Generating from Prometheus spec")
	info := info.Info{
		Version: info.Version,
		Mode:    info.ModeCLIGenPrometheus,
		Spec:    prometheusv1.Version,
	}

	wrappedWindowsRepo := alertWindowRepoWrapper{windowsRepo}
	result, err := generateRules(ctx, logger, info, wrappedWindowsRepo, disableRecs, disableAlerts, disableOptimizedRules, extraLabels, ToPrometheusSLOGroup(slos))
	if err != nil {
		return err
	}

	repo := prometheus.NewIOWriterGroupedRulesYAMLRepo(out, logger)
	storageSLOs := make([]prometheus.StorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, prometheus.StorageSLO{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}

	err = repo.StoreSLOs(ctx, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOS: %w", err)
	}

	return nil
}

// generate is the main generator logic that all the spec types and storers share. Mainly
// has the logic of the generate app service.
func generateRules(ctx context.Context, logger log.Logger, info info.Info, windowsRepo alert.WindowsRepo, disableRecs, disableAlerts, disableOptimizedRules bool, extraLabels map[string]string, slos prometheus.SLOGroup) (*generate.Response, error) {
	// Disable recording rules if required.
	var sliRuleGen generate.SLIRecordingRulesGenerator = generate.NoopSLIRecordingRulesGenerator
	var metaRuleGen generate.MetadataRecordingRulesGenerator = generate.NoopMetadataRecordingRulesGenerator
	if !disableRecs {
		// Disable optimized rules if required.
		sliRuleGen = prometheus.OptimizedSLIRecordingRulesGenerator
		if disableOptimizedRules {
			sliRuleGen = prometheus.SLIRecordingRulesGenerator
		}
		metaRuleGen = prometheus.MetadataRecordingRulesGenerator
	}

	// Disable alert rules if required.
	var alertRuleGen generate.SLOAlertRulesGenerator = generate.NoopSLOAlertRulesGenerator
	if !disableAlerts {
		alertRuleGen = prometheus.SLOAlertRulesGenerator
	}

	// Generate.
	controller, err := generate.NewService(generate.ServiceConfig{
		AlertGenerator:              alert.NewGenerator(windowsRepo),
		SLIRecordingRulesGenerator:  sliRuleGen,
		MetaRecordingRulesGenerator: metaRuleGen,
		SLOAlertRulesGenerator:      alertRuleGen,
		Logger:                      logger,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create application service: %w", err)
	}

	result, err := controller.Generate(ctx, generate.Request{
		ExtraLabels: extraLabels,
		Info:        info,
		SLOGroup:    slos,
	})
	if err != nil {
		return nil, fmt.Errorf("could not generate prometheus rules: %w", err)
	}

	return result, nil
}
