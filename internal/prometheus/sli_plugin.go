package prometheus

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/prometheus/plugin/v1"
	pluginv1 "github.com/slok/sloth/pkg/prometheus/plugin/v1"
)

// FileManager knows how to manage files.
type FileManager interface {
	FindFiles(ctx context.Context, root string, matcher *regexp.Regexp) (paths []string, err error)
	ReadFile(ctx context.Context, path string) (data []byte, err error)
}

//go:generate mockery --case underscore --output prometheusmock --outpkg prometheusmock --name FileManager

type fileManager struct{}

func (f fileManager) FindFiles(ctx context.Context, root string, matcher *regexp.Regexp) ([]string, error) {
	paths := []string{}
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if matcher.MatchString(path) {
			paths = append(paths, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not find files recursively: %w", err)
	}

	return paths, nil
}

func (f fileManager) ReadFile(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

type SLIPlugin struct {
	ID   string
	Func plugin.SLIPlugin
}

type FileSLIPluginRepoConfig struct {
	FileManager FileManager
	Paths       []string
	Logger      log.Logger
}

func (c *FileSLIPluginRepoConfig) defaults() error {
	if c.FileManager == nil {
		c.FileManager = fileManager{}
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"svc": "storage.FileSLIPlugin"})

	return nil
}

func NewFileSLIPluginRepo(config FileSLIPluginRepoConfig) (*FileSLIPluginRepo, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	f := &FileSLIPluginRepo{
		fileManager:  config.FileManager,
		pluginLoader: sliPluginLoader{},
		paths:        config.Paths,
		logger:       config.Logger,
	}

	err = f.Reload(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not load plugins: %w", err)
	}

	return f, nil
}

// FileSLIPluginRepo will provide the plugins loaded from files.
// To be able to provide a simple and safe plugin system to the user we have set some
// rules/requirements that a plugin must implement:
//
// - The plugin must be in a `plugin.go` file inside a directory.
// - All the plugin must be in the `plugin.go` file.
// - The plugin can't import anything apart from the Go standard library.
// - `reflect` and `unsafe` packages can't be used.
//
// These rules provide multiple things:
// - Easy discovery of plugins without the need to provide extra data (import paths, path sanitization...).
// - Safety because we don't allow adding external packages easily.
// - Force keeping the plugins simple, small and without smart code.
// - Force avoiding DRY in small plugins and embrace WET to have independent plugins.
type FileSLIPluginRepo struct {
	pluginLoader sliPluginLoader
	fileManager  FileManager
	paths        []string
	plugins      map[string]SLIPlugin
	mu           sync.RWMutex
	logger       log.Logger
}

var sliPluginNameRegex = regexp.MustCompile("plugin.go$")

// Reload will reload all the plugins again from the paths.
func (f *FileSLIPluginRepo) Reload(ctx context.Context) error {
	// Discover plugins.
	paths := map[string]struct{}{}
	for _, path := range f.paths {
		discoveredPaths, err := f.fileManager.FindFiles(ctx, path, sliPluginNameRegex)
		if err != nil {
			return fmt.Errorf("could not discover SLI plugins: %w", err)
		}
		for _, dPath := range discoveredPaths {
			paths[dPath] = struct{}{}
		}
	}

	// Load the plugins.
	plugins := map[string]SLIPlugin{}
	for path := range paths {
		pluginData, err := f.fileManager.ReadFile(ctx, path)
		if err != nil {
			return fmt.Errorf("could not read %q plugin data: %w", path, err)
		}

		// Create the plugin.
		plugin, err := f.pluginLoader.LoadRawSLIPlugin(ctx, string(pluginData))
		if err != nil {
			return fmt.Errorf("could not load %q plugin: %w", path, err)
		}

		// Check collision.
		_, ok := plugins[plugin.ID]
		if ok {
			return fmt.Errorf("2 or more plugins with the same %q ID have been loaded", plugin.ID)
		}

		plugins[plugin.ID] = *plugin
		f.logger.WithValues(log.Kv{"plugin-id": plugin.ID, "plugin-path": path}).Debugf("SLI plugin loaded")
	}

	// Set loaded plugins.
	f.mu.Lock()
	f.plugins = plugins
	f.mu.Unlock()

	f.logger.WithValues(log.Kv{"plugins": len(plugins)}).Infof("SLI plugins loaded")

	return nil
}

func (f *FileSLIPluginRepo) ListSLIPlugins(ctx context.Context) (map[string]SLIPlugin, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.plugins, nil
}

func (f *FileSLIPluginRepo) GetSLIPlugin(ctx context.Context, id string) (*SLIPlugin, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	p, ok := f.plugins[id]
	if !ok {
		return nil, fmt.Errorf("plugin %q missing", id)
	}

	return &p, nil
}

// sliPluginLoader knows how to load Go SLI plugins using Yaegi.
type sliPluginLoader struct{}

var packageRegexp = regexp.MustCompile(`(?m)^package +([^\s]+) *$`)

// LoadRawSLIPlugin knows how to load plugins using Yaegi from source data not files,
// thats why, this implementation will not support any import library except standard
// library.
//
// The load process will search for:
// - A function called `SLIPlugin` to obtain the plugin func.
// - A constant called `SLIPluginID` to obtain the plugin ID.
// - A constant called `SLIPluginVersion` to obtain the plugin version.
func (s sliPluginLoader) LoadRawSLIPlugin(ctx context.Context, src string) (*SLIPlugin, error) {
	// Load the plugin in a new interpreter.
	// For each plugin we need to use an independent interpreter to avoid name collisions.
	yaegiInterp, err := s.newYaeginInterpreter()
	if err != nil {
		return nil, fmt.Errorf("could not create a new Yaegi interpreter: %w", err)
	}

	_, err = yaegiInterp.EvalWithContext(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("could not evaluate plugin source code: %w", err)
	}

	// Discover package name.
	packageMatch := packageRegexp.FindStringSubmatch(src)
	if len(packageMatch) != 2 {
		return nil, fmt.Errorf("invalid plugin source code, could not get package name")
	}
	packageName := packageMatch[1]

	// Get plugin version and check if is a known one.
	pluginVerTmp, err := yaegiInterp.EvalWithContext(ctx, fmt.Sprintf("%s.SLIPluginVersion", packageName))
	if err != nil {
		return nil, fmt.Errorf("could not get plugin version: %w", err)
	}

	pluginVer, ok := pluginVerTmp.Interface().(pluginv1.SLIPluginVersion)
	if !ok || (pluginVer != pluginv1.Version) {
		return nil, fmt.Errorf("unsuported plugin version: %s", pluginVer)
	}

	// Get plugin ID.
	pluginIDTmp, err := yaegiInterp.EvalWithContext(ctx, fmt.Sprintf("%s.SLIPluginID", packageName))
	if err != nil {
		return nil, fmt.Errorf("could not get plugin ID: %w", err)
	}

	pluginID, ok := pluginIDTmp.Interface().(pluginv1.SLIPluginID)
	if !ok {
		return nil, fmt.Errorf("invalid SLI plugin ID type")
	}

	// Get plugin logic.
	pluginFuncTmp, err := yaegiInterp.EvalWithContext(ctx, fmt.Sprintf("%s.SLIPlugin", packageName))
	if err != nil {
		return nil, fmt.Errorf("could not get plugin: %w", err)
	}

	pluginFunc, ok := pluginFuncTmp.Interface().(pluginv1.SLIPlugin)
	if !ok {
		return nil, fmt.Errorf("invalid SLI plugin type")
	}

	return &SLIPlugin{
		ID:   pluginID,
		Func: pluginFunc,
	}, nil
}

func (s sliPluginLoader) newYaeginInterpreter() (*interp.Interpreter, error) {
	i := interp.New(interp.Options{})
	err := i.Use(stdlib.Symbols)
	if err != nil {
		return nil, fmt.Errorf("could not use stdlib symbols: %w", err)
	}

	return i, nil
}
