package prometheus_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/prometheus"
	"github.com/slok/sloth/internal/prometheus/prometheusmock"
)

func TestSLIPluginLoader(t *testing.T) {
	tests := map[string]struct {
		pluginSrc   string
		pluginID    string
		meta        map[string]string
		labels      map[string]string
		options     map[string]string
		expPluginID string
		expSLIQuery string
		expErrLoad  bool
		expErr      bool
	}{
		"Plugin without version should fail on load.": {
			pluginSrc: `
package testplugin

import "context"

const SLIPluginVersion = "prometheus/v1"

func SLIPlugin(ctx context.Context, meta, labels, options map[string]string) (string, error) {
	return "test_query{}", nil
}
`,
			expErrLoad: true,
		},

		"Basic plugin should load and return a correct SLI.": {
			pluginSrc: `
package testplugin

import "context"

const (
	SLIPluginID      = "test_plugin"
	SLIPluginVersion = "prometheus/v1"
)


func SLIPlugin(ctx context.Context, meta, labels, options map[string]string) (string, error) {
	return "test_query{}", nil
}
`,
			expPluginID: "test_plugin",
			expSLIQuery: "test_query{}",
		},

		"Plugin with meta and options should load and return a correct SLI.": {
			pluginSrc: `
package testplugin

import "context"

import "fmt"

const (
	SLIPluginID      = "test_plugin"
	SLIPluginVersion = "prometheus/v1"
)

func SLIPlugin(ctx context.Context, meta, labels, options map[string]string) (string, error) {
	return fmt.Sprintf("test_query{mk1=\"%s\",lk1=\"%s\",k1=\"%s\",k2=\"%s\"}", meta["mk1"], labels["lk1"], options["k1"], options["k2"]), nil
}
		`,
			meta:        map[string]string{"mk1": "mv1"},
			labels:      map[string]string{"lk1": "lv1"},
			options:     map[string]string{"k1": "v1", "k2": "v2"},
			expSLIQuery: `test_query{mk1="mv1",lk1="lv1",k1="v1",k2="v2"}`,
			expPluginID: "test_plugin",
		},

		"Plugin with error should return errors.": {
			pluginSrc: `
package testplugin

import "context"

import "fmt"

const (
	SLIPluginID      = "test_plugin"
	SLIPluginVersion = "prometheus/v1"
)

func SLIPlugin(ctx context.Context, meta, labels, options map[string]string) (string, error) {
	return "", fmt.Errorf("something")
}
		`,
			meta:        map[string]string{"mk1": "mv1"},
			labels:      map[string]string{"lk1": "lv1"},
			options:     map[string]string{"k1": "v1", "k2": "v2"},
			expPluginID: "test_plugin",
			expErr:      true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Mock the plugin files.
			mfm := &prometheusmock.FileManager{}
			mfm.On("FindFiles", mock.Anything, "./", mock.Anything).Once().Return([]string{"testplugin/test.go"}, nil)
			mfm.On("ReadFile", mock.Anything, "testplugin/test.go").Once().Return([]byte(test.pluginSrc), nil)

			// Create repository and load plugins.
			config := prometheus.FileSLIPluginRepoConfig{
				FileManager: mfm,
				Paths:       []string{"./"},
			}
			repo, err := prometheus.NewFileSLIPluginRepo(config)
			if test.expErrLoad {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			// Get plugin.
			plugin, err := repo.GetSLIPlugin(context.TODO(), test.expPluginID)
			require.NoError(err)

			// Check.
			assert.Equal(test.expPluginID, plugin.ID)

			gotSLIQuery, err := plugin.Func(context.TODO(), test.meta, test.labels, test.options)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expSLIQuery, gotSLIQuery)
			}
		})
	}
}
