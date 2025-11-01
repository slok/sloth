package fs

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"sync"

	"github.com/slok/sloth/internal/log"
	pluginenginek8stransform "github.com/slok/sloth/internal/pluginengine/k8stransform"
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

type K8sTransformPluginLoader interface {
	LoadRawPlugin(ctx context.Context, src string) (*pluginenginek8stransform.Plugin, error)
}

type FilePluginRepo struct {
	fss                []fs.FS
	sloPluginLoader    SLOPluginLoader
	sliPluginLoader    SLIPluginLoader
	k8sTransformLoader K8sTransformPluginLoader
	sloPluginCache     map[string]pluginengineslo.Plugin
	sliPluginCache     map[string]pluginenginesli.SLIPlugin
	k8sTransformCache  map[string]pluginenginek8stransform.Plugin
	logger             log.Logger
	mu                 sync.RWMutex
	failOnError        bool
}

// NewFilePluginRepo returns a new FilePluginRepo that loads SLI and SLO plugins from the given file system.
func NewFilePluginRepo(logger log.Logger, failOnError bool, sliPluginLoader SLIPluginLoader, sloPluginLoader SLOPluginLoader, k8sTransformLoader K8sTransformPluginLoader, fss ...fs.FS) (*FilePluginRepo, error) {
	r := &FilePluginRepo{
		fss:                fss,
		sliPluginLoader:    sliPluginLoader,
		sloPluginLoader:    sloPluginLoader,
		k8sTransformLoader: k8sTransformLoader,
		sloPluginCache:     map[string]pluginengineslo.Plugin{},
		sliPluginCache:     map[string]pluginenginesli.SLIPlugin{},
		k8sTransformCache:  map[string]pluginenginek8stransform.Plugin{},
		logger:             logger,
		failOnError:        failOnError,
	}

	err := r.Reload(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not load plugins: %w", err)
	}

	return r, nil
}

var pluginNameRegex = regexp.MustCompile("plugin.go$")

func (r *FilePluginRepo) Reload(ctx context.Context) error {
	sloPlugins, sliPlugins, k8sTransformPlugins, err := r.loadPlugins(ctx, r.fss...)
	if err != nil {
		return fmt.Errorf("could not load plugins: %w", err)
	}

	// Set loaded plugins.
	r.mu.Lock()
	r.sloPluginCache = sloPlugins
	r.sliPluginCache = sliPlugins
	r.k8sTransformCache = k8sTransformPlugins
	r.mu.Unlock()

	r.logger.WithValues(log.Kv{"slo-plugins": len(sloPlugins), "sli-plugins": len(sliPlugins), "k8s-transform-plugins": len(k8sTransformPlugins)}).Infof("Plugins loaded")
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

func (r *FilePluginRepo) GetK8sTransformPlugin(ctx context.Context, id string) (*pluginenginek8stransform.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.k8sTransformCache[id]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found: %w", id, commonerrors.ErrNotFound)
	}

	return &p, nil
}

func (r *FilePluginRepo) ListK8sTransformPlugins(ctx context.Context) (map[string]pluginenginek8stransform.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.k8sTransformCache, nil
}

func (r *FilePluginRepo) loadPlugins(ctx context.Context, fss ...fs.FS) (map[string]pluginengineslo.Plugin, map[string]pluginenginesli.SLIPlugin, map[string]pluginenginek8stransform.Plugin, error) {
	sloPlugins := map[string]pluginengineslo.Plugin{}
	sliPlugins := map[string]pluginenginesli.SLIPlugin{}
	k8sTransformPlugins := map[string]pluginenginek8stransform.Plugin{}

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

			// Try K8s transform plugin.
			k8sTransformPlugin, k8sErr := r.k8sTransformLoader.LoadRawPlugin(ctx, pluginData)
			if k8sErr == nil {
				_, ok := k8sTransformPlugins[k8sTransformPlugin.ID]
				if ok {
					return fmt.Errorf("plugin %q already loaded", k8sTransformPlugin.ID)
				}
				k8sTransformPlugins[k8sTransformPlugin.ID] = *k8sTransformPlugin
				r.logger.WithValues(log.Kv{"k8s-transform-plugin-id": k8sTransformPlugin.ID}).Debugf("K8s transform plugin discovered and loaded")
				return nil
			}

			err = fmt.Errorf("could not load %q as any kind of plugin: (SLI plugin error: %w | SLO plugin error: %w | K8s transform plugin error: %w)", path, sliErr, sloErr, k8sErr)
			if r.failOnError {
				return err
			}
			r.logger.Errorf(err.Error())

			return nil
		})
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not walk dir: %w", err)
		}
	}

	return sloPlugins, sliPlugins, k8sTransformPlugins, nil
}
