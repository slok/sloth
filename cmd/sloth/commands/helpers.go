package commands

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/prometheus"
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

func createPluginLoader(ctx context.Context, logger log.Logger, paths []string) (*prometheus.FileSLIPluginRepo, error) {
	config := prometheus.FileSLIPluginRepoConfig{
		Paths:  paths,
		Logger: logger,
	}
	sliPluginRepo, err := prometheus.NewFileSLIPluginRepo(config)
	if err != nil {
		return nil, fmt.Errorf("could not create file SLI plugin repository: %w", err)
	}

	return sliPluginRepo, nil
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
