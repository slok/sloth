package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/plugin"
	pluginenginesli "github.com/slok/sloth/internal/pluginengine/sli"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	storagefs "github.com/slok/sloth/internal/storage/fs"
	"github.com/slok/sloth/pkg/common/model"
)

var (
	splitMarkRe  = regexp.MustCompile("(?m)^---")
	rmCommentsRe = regexp.MustCompile("(?m)^#.*$")
)

func splitYAML(data []byte) []string {
	// Santize.
	data = bytes.TrimSpace(data)
	data = rmCommentsRe.ReplaceAll(data, []byte(""))

	// Split (YAML can declare multiple files in the same file using `---`).
	dataSplit := splitMarkRe.Split(string(data), -1)

	// Remove empty splits.
	nonEmptyData := []string{}
	for _, d := range dataSplit {
		d = strings.TrimSpace(d)
		if d != "" {
			nonEmptyData = append(nonEmptyData, d)
		}
	}

	return nonEmptyData
}

func createPluginLoader(ctx context.Context, logger log.Logger, paths []string) (*storagefs.FilePluginRepo, error) {
	fss := []fs.FS{
		plugin.EmbeddedDefaultSLOPlugins,
	}
	for _, p := range paths {
		fss = append(fss, os.DirFS(p))
	}

	pluginsRepo, err := storagefs.NewFilePluginRepo(logger, pluginenginesli.PluginLoader, pluginengineslo.PluginLoader, fss...)
	if err != nil {
		return nil, fmt.Errorf("could not create file SLO and SLI plugins repository: %w", err)
	}

	return pluginsRepo, nil
}

func discoverSLOManifests(logger log.Logger, exclude, include *regexp.Regexp, path string) ([]string, error) {
	logger = logger.WithValues(log.Kv{"svc": "SLODiscovery"})

	paths := []string{}
	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Directories and non YAML files don't need to be handled.
		extension := strings.ToLower(filepath.Ext(path))
		if info.IsDir() || (extension != ".yml" && extension != ".yaml") {
			return nil
		}

		// Filter by exclude or include (exclude has preference).
		if exclude != nil && exclude.MatchString(path) {
			logger.Debugf("Excluding path due to exclude filter %s", path)
			return nil
		}
		if include != nil && !include.MatchString(path) {
			logger.Debugf("Excluding path due to include filter %s", path)
			return nil
		}

		// If we reach here, path discovered.
		paths = append(paths, path)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not find files recursively: %w", err)
	}

	return paths, nil
}

func mapCmdPluginToModel(ctx context.Context, jsonPlugins []string) ([]model.PromSLOPluginMetadata, error) {
	type jsonPlugin struct {
		ID       string          `json:"id"`
		Config   json.RawMessage `json:"config"`
		Priority int             `json:"priority"`
	}

	plugins := []model.PromSLOPluginMetadata{}
	for _, jp := range jsonPlugins {
		p := &jsonPlugin{}
		err := json.Unmarshal([]byte(jp), p)
		if err != nil {
			return nil, fmt.Errorf("could not load cmd plugin json config %q: %w", jp, err)
		}

		plugins = append(plugins, model.PromSLOPluginMetadata{
			ID:       p.ID,
			Config:   p.Config,
			Priority: p.Priority,
		})
	}

	return plugins, nil
}
