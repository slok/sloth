package fs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	pluginsli "github.com/slok/sloth/internal/plugin/sli"
	"github.com/slok/sloth/internal/storage/fs"
	"github.com/slok/sloth/internal/storage/fs/fsmock"
)

func TestSLIPluginRepoListSLIPlugins(t *testing.T) {
	tests := map[string]struct {
		mock       func(mfm *fsmock.FileManager, mpl *fsmock.SLIPluginLoader)
		expPlugins map[string]pluginsli.SLIPlugin
		expErr     bool
	}{
		"Having no files, should return empty list of plugins.": {
			mock: func(mfm *fsmock.FileManager, mpl *fsmock.SLIPluginLoader) {
				mfm.On("FindFiles", mock.Anything, "./", mock.Anything).Once().Return([]string{}, nil)
			},
			expPlugins: map[string]pluginsli.SLIPlugin{},
		},

		"Having multiple files, should return multiple plugins.": {
			mock: func(mfm *fsmock.FileManager, mpl *fsmock.SLIPluginLoader) {
				mfm.On("FindFiles", mock.Anything, "./", mock.Anything).Once().Return([]string{
					"./test_plugin_1.go",
					"./test_plugin_2.go",
					"./test2/test_plugin_3.go",
					"./test3/test4/test_plugin_4.go",
				}, nil)

				mfm.On("ReadFile", mock.Anything, "./test_plugin_1.go").Once().Return([]byte(`test1`), nil)
				mfm.On("ReadFile", mock.Anything, "./test_plugin_2.go").Once().Return([]byte(`test2`), nil)
				mfm.On("ReadFile", mock.Anything, "./test2/test_plugin_3.go").Once().Return([]byte(`test3`), nil)
				mfm.On("ReadFile", mock.Anything, "./test3/test4/test_plugin_4.go").Once().Return([]byte(`test4`), nil)

				mpl.On("LoadRawSLIPlugin", mock.Anything, "test1").Once().Return(&pluginsli.SLIPlugin{ID: "test1"}, nil)
				mpl.On("LoadRawSLIPlugin", mock.Anything, "test2").Once().Return(&pluginsli.SLIPlugin{ID: "test2"}, nil)
				mpl.On("LoadRawSLIPlugin", mock.Anything, "test3").Once().Return(&pluginsli.SLIPlugin{ID: "test3"}, nil)
				mpl.On("LoadRawSLIPlugin", mock.Anything, "test4").Once().Return(&pluginsli.SLIPlugin{ID: "test4"}, nil)
			},
			expPlugins: map[string]pluginsli.SLIPlugin{
				"test1": {ID: "test1"},
				"test2": {ID: "test2"},
				"test3": {ID: "test3"},
				"test4": {ID: "test4"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mfm := fsmock.NewFileManager(t)
			mpl := fsmock.NewSLIPluginLoader(t)
			test.mock(mfm, mpl)

			// Create repository and load plugins.
			config := fs.FileSLIPluginRepoConfig{
				FileManager:  mfm,
				PluginLoader: mpl,
				Paths:        []string{"./"},
			}
			repo, err := fs.NewFileSLIPluginRepo(config)
			require.NoError(err)

			plugins, err := repo.ListSLIPlugins(t.Context())
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expPlugins, plugins)
			}
		})
	}
}

func TestSLIPluginRepoGetSLIPlugin(t *testing.T) {
	tests := map[string]struct {
		pluginID  string
		mock      func(mfm *fsmock.FileManager, mpl *fsmock.SLIPluginLoader)
		expPlugin pluginsli.SLIPlugin
		expErr    bool
	}{
		"Having a missing plugin, should fail.": {
			pluginID: "test3",
			mock: func(mfm *fsmock.FileManager, mpl *fsmock.SLIPluginLoader) {
				mfm.On("FindFiles", mock.Anything, "./", mock.Anything).Once().Return([]string{
					"./test_plugin_1.go",
					"./test_plugin_2.go",
				}, nil)

				mfm.On("ReadFile", mock.Anything, "./test_plugin_1.go").Once().Return([]byte(`test1`), nil)
				mfm.On("ReadFile", mock.Anything, "./test_plugin_2.go").Once().Return([]byte(`test2`), nil)

				mpl.On("LoadRawSLIPlugin", mock.Anything, "test1").Once().Return(&pluginsli.SLIPlugin{ID: "test1"}, nil)
				mpl.On("LoadRawSLIPlugin", mock.Anything, "test2").Once().Return(&pluginsli.SLIPlugin{ID: "test2"}, nil)
			},
			expErr: true,
		},

		"Having a correct plugin, should return the plugin.": {
			pluginID: "test2",
			mock: func(mfm *fsmock.FileManager, mpl *fsmock.SLIPluginLoader) {
				mfm.On("FindFiles", mock.Anything, "./", mock.Anything).Once().Return([]string{
					"./test_plugin_1.go",
					"./test_plugin_2.go",
				}, nil)

				mfm.On("ReadFile", mock.Anything, "./test_plugin_1.go").Once().Return([]byte(`test1`), nil)
				mfm.On("ReadFile", mock.Anything, "./test_plugin_2.go").Once().Return([]byte(`test2`), nil)

				mpl.On("LoadRawSLIPlugin", mock.Anything, "test1").Once().Return(&pluginsli.SLIPlugin{ID: "test1"}, nil)
				mpl.On("LoadRawSLIPlugin", mock.Anything, "test2").Once().Return(&pluginsli.SLIPlugin{ID: "test2"}, nil)
			},
			expPlugin: pluginsli.SLIPlugin{ID: "test2"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mfm := fsmock.NewFileManager(t)
			mpl := fsmock.NewSLIPluginLoader(t)
			test.mock(mfm, mpl)

			// Create repository and load plugins.
			config := fs.FileSLIPluginRepoConfig{
				FileManager:  mfm,
				PluginLoader: mpl,
				Paths:        []string{"./"},
			}
			repo, err := fs.NewFileSLIPluginRepo(config)
			require.NoError(err)

			plugin, err := repo.GetSLIPlugin(t.Context(), test.pluginID)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expPlugin, *plugin)
			}
		})
	}
}
