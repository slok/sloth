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
		meta        map[string]interface{}
		options     map[string]interface{}
		expPluginID string
		expSLIQuery string
		expErr      bool
	}{
		"Basic plugin should load and return a correct SLI.": {
			pluginSrc: `
package testplugin

const SLIPluginID = "test_plugin"

func SLIPlugin(meta map[string]interface{}, options map[string]interface{}) (string, error) {
	return "test_query{}", nil
}
`,
			expPluginID: "test_plugin",
			expSLIQuery: "test_query{}",
		},

		"Plugin with meta and options should load and return a correct SLI.": {
			pluginSrc: `
package testplugin

import "fmt"

const SLIPluginID = "test_plugin"

func SLIPlugin(meta map[string]interface{}, options map[string]interface{}) (string, error) {
	return fmt.Sprintf("test_query{mk1=\"%s\",k1=\"%s\",k2=\"%s\"}", meta["mk1"], options["k1"], options["k2"]), nil
}
		`,
			meta:        map[string]interface{}{"mk1": "mv1"},
			options:     map[string]interface{}{"k1": "v1", "k2": "v2"},
			expSLIQuery: `test_query{mk1="mv1",k1="v1",k2="v2"}`,
			expPluginID: "test_plugin",
		},

		"Plugin with error should return errors.": {
			pluginSrc: `
package testplugin

import "fmt"

const SLIPluginID = "test_plugin"

func SLIPlugin(meta map[string]interface{}, options map[string]interface{}) (string, error) {
	return "", fmt.Errorf("something")
}
		`,
			meta:        map[string]interface{}{"mk1": "mv1"},
			options:     map[string]interface{}{"k1": "v1", "k2": "v2"},
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
			require.NoError(err)

			plugins, err := repo.ListSLIPlugins(context.TODO())
			require.NoError(err)

			// Execute pluginand check.
			assert.Len(plugins, 1)
			plugin, ok := plugins[test.expPluginID]
			assert.True(ok)
			assert.Equal(test.expPluginID, plugin.ID)

			gotSLIQuery, err := plugin.Func(test.meta, test.options)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expSLIQuery, gotSLIQuery)
			}
		})
	}
}
