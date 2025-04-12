package fs

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"sync"

	"github.com/slok/sloth/internal/log"
	pluginenginesli "github.com/slok/sloth/internal/pluginengine/sli"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	commonerrors "github.com/slok/sloth/pkg/common/errors"
)

type SLIPluginLoader interface {
	LoadRawSLIPlugin(ctx context.Context, src string) (*pluginenginesli.SLIPlugin, error)
}

type SLOPluginLoader interface {
	LoadRawPlugin(ctx context.Context, src string) (*pluginengineslo.Plugin, error)
}

type FilePluginRepo struct {
	fss             []fs.FS
	sloPluginLoader SLOPluginLoader
	sliPluginLoader SLIPluginLoader
	sloPluginCache  map[string]pluginengineslo.Plugin
	sliPluginCache  map[string]pluginenginesli.SLIPlugin
	logger          log.Logger
	mu              sync.RWMutex
}

// NewFilePluginRepo returns a new FilePluginRepo that loads SLI and SLO plugins from the given file system.
func NewFilePluginRepo(logger log.Logger, sliPluginLoader SLIPluginLoader, sloPluginLoader SLOPluginLoader, fss ...fs.FS) (*FilePluginRepo, error) {
	r := &FilePluginRepo{
		fss:             fss,
		sliPluginLoader: sliPluginLoader,
		sloPluginLoader: sloPluginLoader,
		sloPluginCache:  map[string]pluginengineslo.Plugin{},
		sliPluginCache:  map[string]pluginenginesli.SLIPlugin{},
		logger:          logger,
	}

	err := r.Reload(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not load plugins: %w", err)
	}

	return r, nil
}

var pluginNameRegex = regexp.MustCompile("plugin.go$")

func (r *FilePluginRepo) Reload(ctx context.Context) error {
	sloPlugins, sliPlugins, err := r.loadPlugins(ctx, r.fss...)
	if err != nil {
		return fmt.Errorf("could not load plugins: %w", err)
	}

	// Set loaded plugins.
	r.mu.Lock()
	r.sloPluginCache = sloPlugins
	r.sliPluginCache = sliPlugins
	r.mu.Unlock()

	r.logger.WithValues(log.Kv{"slo-plugins": len(sloPlugins), "sli-plugins": len(sliPlugins)}).Infof("Plugins loaded")
	return nil
}

func (r *FilePluginRepo) GetSLOPlugin(ctx context.Context, id string) (*pluginengineslo.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.sloPluginCache[id]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found: %w", id, commonerrors.ErrNotFound)
	}

	return &p, nil
}

func (r *FilePluginRepo) ListSLOPlugins(ctx context.Context) (map[string]pluginengineslo.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.sloPluginCache, nil
}

func (r *FilePluginRepo) GetSLIPlugin(ctx context.Context, id string) (*pluginenginesli.SLIPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.sliPluginCache[id]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found: %w", id, commonerrors.ErrNotFound)
	}

	return &p, nil
}

func (r *FilePluginRepo) ListSLIPlugins(ctx context.Context) (map[string]pluginenginesli.SLIPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.sliPluginCache, nil
}

func (r *FilePluginRepo) loadPlugins(ctx context.Context, fss ...fs.FS) (map[string]pluginengineslo.Plugin, map[string]pluginenginesli.SLIPlugin, error) {
	sloPlugins := map[string]pluginengineslo.Plugin{}
	sliPlugins := map[string]pluginenginesli.SLIPlugin{}

	for _, f := range fss {
		err := fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			if !pluginNameRegex.MatchString(path) {
				return nil
			}

			pluginDataBytes, err := fs.ReadFile(f, path)
			if err != nil {
				return fmt.Errorf("could not read %q plugin data: %w", path, err)
			}
			pluginData := string(pluginDataBytes)

			// Try SLI plugin if not, SLO plugin.
			sliPlugin, sliErr := r.sliPluginLoader.LoadRawSLIPlugin(ctx, pluginData)
			if sliErr == nil {
				_, ok := sliPlugins[sliPlugin.ID]
				if ok {
					return fmt.Errorf("plugin %q already loaded", sliPlugin.ID)
				}
				sliPlugins[sliPlugin.ID] = *sliPlugin
				r.logger.WithValues(log.Kv{"sli-plugin-id": sliPlugin.ID}).Debugf("SLI plugin discovered and loaded")
				return nil
			}

			// Try SLO plugin.
			sloPlugin, sloErr := r.sloPluginLoader.LoadRawPlugin(ctx, pluginData)
			if sloErr == nil {
				_, ok := sloPlugins[sloPlugin.ID]
				if ok {
					return fmt.Errorf("plugin %q already loaded", sloPlugin.ID)
				}
				sloPlugins[sloPlugin.ID] = *sloPlugin
				r.logger.WithValues(log.Kv{"slo-plugin-id": sloPlugin.ID}).Debugf("SLO plugin discovered and loaded")
				return nil
			}

			r.logger.Errorf("could not load %q as SLI or SLO plugin: (SLI plugin error: %s | SLO plugin error: %s)", path, sliErr, sloErr)

			return nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("could not walk dir: %w", err)
		}
	}

	return sloPlugins, sliPlugins, nil
}
