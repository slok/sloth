package fs_test

import (
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/log"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	storagefs "github.com/slok/sloth/internal/storage/fs"
	"github.com/slok/sloth/internal/storage/fs/fsmock"
)

func TestFileSLOPluginRepoListSLOPlugins(t *testing.T) {
	tests := map[string]struct {
		fss        func() []fs.FS
		mock       func(mpl *fsmock.SLOPluginLoader)
		expPlugins map[string]pluginengineslo.Plugin
		expLoadErr bool
		expErr     bool
	}{
		"Having no files, should return empty list of plugins.": {
			fss:        func() []fs.FS { return nil },
			mock:       func(mpl *fsmock.SLOPluginLoader) {},
			expPlugins: map[string]pluginengineslo.Plugin{},
		},

		"Having plugins in multiple FS and directories, should return all plugins.": {
			fss: func() []fs.FS {
				m1 := make(fstest.MapFS)
				m1["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m1["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}
				m1["m1/plx/pl3/plugin.go"] = &fstest.MapFile{Data: []byte("p3")}

				m2 := make(fstest.MapFS)
				m2["m2/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p4")}
				m2["m2/plx/pl3/plugin.go"] = &fstest.MapFile{Data: []byte("p5")}
				m2["m2/plugin-test.go"] = &fstest.MapFile{Data: []byte("p8")} // Ignored.

				m3 := make(fstest.MapFS)
				m3["m3/plx/pl3/plugin.go"] = &fstest.MapFile{Data: []byte("p6")}
				m3["m3/plx/pl3/plugin.yaml"] = &fstest.MapFile{Data: []byte("p7")} // Ignored.

				return []fs.FS{m1, m2, m3}
			},
			mock: func(mpl *fsmock.SLOPluginLoader) {
				mpl.On("LoadRawPlugin", mock.Anything, "p1").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)
				mpl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(&pluginengineslo.Plugin{ID: "p2"}, nil)
				mpl.On("LoadRawPlugin", mock.Anything, "p3").Once().Return(&pluginengineslo.Plugin{ID: "p3"}, nil)
				mpl.On("LoadRawPlugin", mock.Anything, "p4").Once().Return(&pluginengineslo.Plugin{ID: "p4"}, nil)
				mpl.On("LoadRawPlugin", mock.Anything, "p5").Once().Return(&pluginengineslo.Plugin{ID: "p5"}, nil)
				mpl.On("LoadRawPlugin", mock.Anything, "p6").Once().Return(&pluginengineslo.Plugin{ID: "p6"}, nil)
			},
			expPlugins: map[string]pluginengineslo.Plugin{
				"p1": {ID: "p1"},
				"p2": {ID: "p2"},
				"p3": {ID: "p3"},
				"p4": {ID: "p4"},
				"p5": {ID: "p5"},
				"p6": {ID: "p6"},
			},
		},

		"Having a plugin loaded with the same ID multiple times should fail.": {
			fss: func() []fs.FS {
				m1 := make(fstest.MapFS)
				m1["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m1["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}

				return []fs.FS{m1}
			},
			mock: func(mpl *fsmock.SLOPluginLoader) {
				mpl.On("LoadRawPlugin", mock.Anything, "p1").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)
				mpl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)

			},
			expLoadErr: true,
		},

		"Having an error while loading a plugin, should fail.": {
			fss: func() []fs.FS {
				m1 := make(fstest.MapFS)
				m1["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m1["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}

				return []fs.FS{m1}
			},
			mock: func(mpl *fsmock.SLOPluginLoader) {
				mpl.On("LoadRawPlugin", mock.Anything, "p1").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)
				mpl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))

			},
			expLoadErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mpl := fsmock.NewSLOPluginLoader(t)
			test.mock(mpl)

			// Create repository and load plugins.
			repo, err := storagefs.NewFileSLOPluginRepo(log.Noop, mpl, test.fss()...)
			if test.expLoadErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			plugins, err := repo.ListSLOPlugins(t.Context())
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expPlugins, plugins)
			}
		})
	}
}

func TestFileSLOPluginRepoGetSLOPlugin(t *testing.T) {
	tests := map[string]struct {
		pluginID  string
		fss       func() []fs.FS
		mock      func(mpl *fsmock.SLOPluginLoader)
		expPlugin pluginengineslo.Plugin
		expErr    bool
	}{
		"Having no files, should return empty list of plugins.": {
			pluginID: "test",
			fss:      func() []fs.FS { return nil },
			mock:     func(mpl *fsmock.SLOPluginLoader) {},
			expErr:   true,
		},

		"Getting a correct plugin, should return the plugin.": {
			pluginID: "p2",
			fss: func() []fs.FS {
				m := make(fstest.MapFS)
				m["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}
				return []fs.FS{m}
			},
			mock: func(mpl *fsmock.SLOPluginLoader) {
				mpl.On("LoadRawPlugin", mock.Anything, "p1").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)
				mpl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(&pluginengineslo.Plugin{ID: "p2"}, nil)
			},
			expPlugin: pluginengineslo.Plugin{ID: "p2"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mpl := fsmock.NewSLOPluginLoader(t)
			test.mock(mpl)

			// Create repository and load plugins.
			repo, err := storagefs.NewFileSLOPluginRepo(log.Noop, mpl, test.fss()...)
			require.NoError(err)

			plugin, err := repo.GetSLOPlugin(t.Context(), test.pluginID)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expPlugin, *plugin)
			}
		})
	}
}
