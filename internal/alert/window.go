package alert

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/slok/sloth/internal/log"
	alertwindowsv1 "github.com/slok/sloth/pkg/prometheus/alertwindows/v1"
)

var (
	//go:embed windows
	// Raw embedded alert windows.
	//
	// Warning, Go embed will ignore `_*` files.
	embeddedWindows embed.FS
)

type Window struct {
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

func (w Window) Validate() error {
	if w.LongWindow == 0 {
		return fmt.Errorf("long window is required")
	}

	if w.ShortWindow == 0 {
		return fmt.Errorf("short window is required")
	}

	if w.ErrorBudgetPercent == 0 {
		return fmt.Errorf("error budget is required")
	}

	return nil
}

// Windows has the information of the windows for multiwindow-multiburn SLO alerting.
// Its a matrix of values with:
// - Alert severity: ["page", "ticket"].
// - Measuring period: ["long", "short"].
type Windows struct {
	SLOPeriod   time.Duration
	PageQuick   Window
	PageSlow    Window
	TicketQuick Window
	TicketSlow  Window
}

func (w Windows) Validate() error {
	if w.SLOPeriod == 0 {
		return fmt.Errorf("slo period is required")
	}

	err := w.PageQuick.Validate()
	if err != nil {
		return fmt.Errorf("invalid page quick: %w", err)
	}

	err = w.PageSlow.Validate()
	if err != nil {
		return fmt.Errorf("invalid page slow: %w", err)
	}

	err = w.TicketQuick.Validate()
	if err != nil {
		return fmt.Errorf("invalid ticket quick: %w", err)
	}

	err = w.TicketSlow.Validate()
	if err != nil {
		return fmt.Errorf("invalid ticket slow: %w", err)
	}

	return nil
}

// Error budget speeds based on a full time window, however once we have the factor (speed)
// the value can be used with any time window.
func (w Windows) GetSpeedPageQuick() float64 {
	return w.getBurnRateFactor(w.SLOPeriod, float64(w.PageQuick.ErrorBudgetPercent), w.PageQuick.LongWindow)
}
func (w Windows) GetSpeedPageSlow() float64 {
	return w.getBurnRateFactor(w.SLOPeriod, float64(w.PageSlow.ErrorBudgetPercent), w.PageSlow.LongWindow)
}
func (w Windows) GetSpeedTicketQuick() float64 {
	return w.getBurnRateFactor(w.SLOPeriod, float64(w.TicketQuick.ErrorBudgetPercent), w.TicketQuick.LongWindow)
}
func (w Windows) GetSpeedTicketSlow() float64 {
	return w.getBurnRateFactor(w.SLOPeriod, float64(w.TicketSlow.ErrorBudgetPercent), w.TicketSlow.LongWindow)
}

// getBurnRateFactor calculates the burnRateFactor (speed) needed to consume all the error budget available percent
// in a specific time window taking into account the total time window.
func (w Windows) getBurnRateFactor(totalWindow time.Duration, errorBudgetPercent float64, consumptionWindow time.Duration) float64 {
	// First get the total hours required to consume the % of the error budget in the total window.
	hoursRequiredConsumption := errorBudgetPercent * totalWindow.Hours() / 100

	// Now calculate how much is the factor required for the hours consumption, in case we would need to use
	// a different time window (e.g: hours required: 36h, if we want to do it in 6h: would be `x6`).
	speed := hoursRequiredConsumption / consumptionWindow.Hours()

	return speed
}

type FSWindowsRepoConfig struct {
	FS     fs.FS
	Logger log.Logger
}

func (c *FSWindowsRepoConfig) defaults() error {
	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"svc": "alert.WindowsRepo"})

	return nil
}

type FSWindowsRepo struct {
	windows map[time.Duration]Windows
	loader  windowLoader
	logger  log.Logger
}

func NewFSWindowsRepo(config FSWindowsRepoConfig) (*FSWindowsRepo, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	f := &FSWindowsRepo{
		windows: map[time.Duration]Windows{},
		logger:  config.Logger,
	}

	// Load default windows if not custom windows are specified.
	if config.FS == nil {
		err = f.load(context.Background(), embeddedWindows)
		if err != nil {
			return nil, fmt.Errorf("could not initialize default windows: %w", err)
		}
	} else {
		config.Logger.Infof("Using custom slo period windows catalog")
		err = f.load(context.Background(), config.FS)
		if err != nil {
			return nil, fmt.Errorf("could not initialize custom windows: %w", err)
		}
	}

	config.Logger.WithValues(log.Kv{"windows": len(f.windows)}).Infof("SLO period windows loaded")

	return f, nil
}

func (f *FSWindowsRepo) load(ctx context.Context, windowsFS fs.FS) error {
	err := fs.WalkDir(windowsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := fs.ReadFile(windowsFS, path)
		if err != nil {
			return fmt.Errorf("could not read  %q  alert windows data from file: %w", path, err)
		}

		windows, err := f.loader.LoadWindow(ctx, data)
		if err != nil {
			return fmt.Errorf("could not load %q alert windows: %w", path, err)
		}

		// Check if it was already loaded so we avoid conflicts and load race conditions, then add to the catalog.
		storedWindows, ok := f.windows[windows.SLOPeriod]
		if ok {
			// If is the same spec, just warn and don't fail.
			if !reflect.DeepEqual(storedWindows, *windows) {
				return fmt.Errorf("%q slo period is already loaded", windows.SLOPeriod)
			}
			f.logger.Warningf("Identical %q slo periods have been loaded multiple times", windows.SLOPeriod)
			return nil
		}
		f.windows[windows.SLOPeriod] = *windows

		return nil
	})

	if err != nil {
		return fmt.Errorf("could not discover period windows: %w", err)
	}

	return nil
}

func (f *FSWindowsRepo) GetWindows(ctx context.Context, period time.Duration) (*Windows, error) {
	w, ok := f.windows[period]
	if !ok {
		return nil, fmt.Errorf("window period %s missing", period)
	}

	return &w, nil
}

type windowLoader struct{}

func (l windowLoader) LoadWindow(ctx context.Context, data []byte) (*Windows, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("spec is required")
	}

	s := alertwindowsv1.AlertWindows{}
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall YAML spec correctly: %w", err)
	}

	// Check version.
	if s.APIVersion != alertwindowsv1.APIVersion || s.Kind != alertwindowsv1.Kind {
		return nil, fmt.Errorf("invalid spec version")
	}

	// Map to model.
	w := &Windows{
		SLOPeriod: time.Duration(s.Spec.SLOPeriod),
		PageQuick: Window{
			ErrorBudgetPercent: s.Spec.Page.Quick.ErrorBudgetPercent,
			ShortWindow:        time.Duration(s.Spec.Page.Quick.ShortWindow),
			LongWindow:         time.Duration(s.Spec.Page.Quick.LongWindow),
		},
		PageSlow: Window{
			ErrorBudgetPercent: s.Spec.Page.Slow.ErrorBudgetPercent,
			ShortWindow:        time.Duration(s.Spec.Page.Slow.ShortWindow),
			LongWindow:         time.Duration(s.Spec.Page.Slow.LongWindow),
		},
		TicketQuick: Window{
			ErrorBudgetPercent: s.Spec.Ticket.Quick.ErrorBudgetPercent,
			ShortWindow:        time.Duration(s.Spec.Ticket.Quick.ShortWindow),
			LongWindow:         time.Duration(s.Spec.Ticket.Quick.LongWindow),
		},
		TicketSlow: Window{
			ErrorBudgetPercent: s.Spec.Ticket.Slow.ErrorBudgetPercent,
			ShortWindow:        time.Duration(s.Spec.Ticket.Slow.ShortWindow),
			LongWindow:         time.Duration(s.Spec.Ticket.Slow.LongWindow),
		},
	}

	err = w.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid alerting window: %w", err)
	}
	return w, nil
}
