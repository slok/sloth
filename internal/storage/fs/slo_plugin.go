package fs

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"sync"

	"github.com/slok/sloth/internal/log"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	commonerrors "github.com/slok/sloth/pkg/common/errors"
)

type SLOPluginLoader interface {
	LoadRawPlugin(ctx context.Context, src string) (*pluginengineslo.Plugin, error)
}

//go:generate mockery --case underscore --output fsmock --outpkg fsmock --name SLOPluginLoader

type FileSLOPluginRepo struct {
	fss          []fs.FS
	pluginLoader SLOPluginLoader
	cache        map[string]pluginengineslo.Plugin
	logger       log.Logger
	mu           sync.RWMutex
}

// NewFileSLOPluginRepo returns a new FileSLOPluginRepo that loads plugins from the given file system.
// The plugin file should be called "plugin.go".
func NewFileSLOPluginRepo(logger log.Logger, pluginLoader SLOPluginLoader, fss ...fs.FS) (*FileSLOPluginRepo, error) {
	r := &FileSLOPluginRepo{
		fss:          fss,
		pluginLoader: pluginLoader,
		cache:        map[string]pluginengineslo.Plugin{},
		logger:       logger,
	}

	err := r.Reload(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not load plugins: %w", err)
	}

	return r, nil
}

var sloPluginNameRegex = regexp.MustCompile("plugin.go$")

func (r *FileSLOPluginRepo) Reload(ctx context.Context) error {
	plugins, err := r.loadPlugins(ctx, r.fss...)
	if err != nil {
		return fmt.Errorf("could not load plugins: %w", err)
	}

	// Set loaded plugins.
	r.mu.Lock()
	r.cache = plugins
	r.mu.Unlock()

	r.logger.WithValues(log.Kv{"plugins": len(plugins)}).Infof("SLO plugins loaded")
	return nil
}

func (r *FileSLOPluginRepo) GetSLOPlugin(ctx context.Context, id string) (*pluginengineslo.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.cache[id]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found: %w", id, commonerrors.ErrNotFound)
	}

	return &p, nil
}

func (r *FileSLOPluginRepo) ListSLOPlugins(ctx context.Context) (map[string]pluginengineslo.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.cache, nil
}

func (r *FileSLOPluginRepo) loadPlugins(ctx context.Context, fss ...fs.FS) (map[string]pluginengineslo.Plugin, error) {
	allPlugins := map[string]pluginengineslo.Plugin{}

	for _, f := range fss {
		err := fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			if !sloPluginNameRegex.MatchString(path) {
				return nil
			}

			pluginData, err := fs.ReadFile(f, path)
			if err != nil {
				return fmt.Errorf("could not read %q plugin data: %w", path, err)
			}

			plugin, err := r.pluginLoader.LoadRawPlugin(ctx, string(pluginData))
			if err != nil {
				return fmt.Errorf("could not load %q plugin: %w", path, err)
			}

			_, ok := allPlugins[plugin.ID]
			if ok {
				return fmt.Errorf("plugin %q already loaded", plugin.ID)
			}
			allPlugins[plugin.ID] = *plugin
			r.logger.WithValues(log.Kv{"plugin-id": plugin.ID}).Debugf("SLO plugin discovered and loaded")

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("could not walk dir: %w", err)
		}
	}

	return allPlugins, nil
}
