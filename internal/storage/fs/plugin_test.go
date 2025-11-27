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
	pluginenginesli "github.com/slok/sloth/internal/pluginengine/sli"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	storagefs "github.com/slok/sloth/internal/storage/fs"
	"github.com/slok/sloth/internal/storage/fs/fsmock"
)

func TestFilePluginRepoListSLOPlugins(t *testing.T) {
	tests := map[string]struct {
		failOnError bool
		fss         func() []fs.FS
		mock        func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader)
		expPlugins  map[string]pluginengineslo.Plugin
		expLoadErr  bool
		expErr      bool
	}{
		"Having no files, should return empty list of plugins.": {
			fss: func() []fs.FS { return nil },
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
			},
			expPlugins: map[string]pluginengineslo.Plugin{},
		},

		"Having plugins in multiple FS and directories, should return all plugins.": {
			fss: func() []fs.FS {
				m1 := make(fstest.MapFS)
				m1["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m1["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}
				m1["m1/plx/pl3/plugin.go"] = &fstest.MapFile{Data: []byte("p3")}
				m1["m1/..pl4/plugin.go"] = &fstest.MapFile{Data: []byte("p7")} // Ignored.

				m2 := make(fstest.MapFS)
				m2["m2/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p4")}
				m2["m2/plx/pl3/plugin.go"] = &fstest.MapFile{Data: []byte("p5")}
				m2["m2/plugin-test.go"] = &fstest.MapFile{Data: []byte("p8")} // Ignored.

				m3 := make(fstest.MapFS)
				m3["m3/plx/pl3/plugin.go"] = &fstest.MapFile{Data: []byte("p6")}
				m3["m3/plx/pl3/plugin.yaml"] = &fstest.MapFile{Data: []byte("p9")} // Ignored.

				return []fs.FS{m1, m2, m3}
			},
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p1").Once().Return(nil, fmt.Errorf("something"))
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p3").Once().Return(nil, fmt.Errorf("something"))
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p4").Once().Return(nil, fmt.Errorf("something"))
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p5").Once().Return(nil, fmt.Errorf("something"))
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p6").Once().Return(nil, fmt.Errorf("something"))

				mslopl.On("LoadRawPlugin", mock.Anything, "p1").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)
				mslopl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(&pluginengineslo.Plugin{ID: "p2"}, nil)
				mslopl.On("LoadRawPlugin", mock.Anything, "p3").Once().Return(&pluginengineslo.Plugin{ID: "p3"}, nil)
				mslopl.On("LoadRawPlugin", mock.Anything, "p4").Once().Return(&pluginengineslo.Plugin{ID: "p4"}, nil)
				mslopl.On("LoadRawPlugin", mock.Anything, "p5").Once().Return(&pluginengineslo.Plugin{ID: "p5"}, nil)
				mslopl.On("LoadRawPlugin", mock.Anything, "p6").Once().Return(&pluginengineslo.Plugin{ID: "p6"}, nil)
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
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p1").Once().Return(nil, fmt.Errorf("something"))
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))

				mslopl.On("LoadRawPlugin", mock.Anything, "p1").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)
				mslopl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)

			},
			expLoadErr: true,
		},

		"Having an error while loading a plugin, should not fail but don't load the failed plugin.": {
			fss: func() []fs.FS {
				m1 := make(fstest.MapFS)
				m1["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m1["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}

				return []fs.FS{m1}
			},
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p1").Once().Return(nil, fmt.Errorf("something"))
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))

				mslopl.On("LoadRawPlugin", mock.Anything, "p1").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)
				mslopl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))

				mk8stl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))
			},
			expPlugins: map[string]pluginengineslo.Plugin{
				"p1": {ID: "p1"},
			},
		},

		"Having an error while loading a plugin on strict mode, should fail.": {
			failOnError: true,
			fss: func() []fs.FS {
				m1 := make(fstest.MapFS)
				m1["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m1["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}

				return []fs.FS{m1}
			},
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p1").Once().Return(nil, fmt.Errorf("something"))
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))

				mslopl.On("LoadRawPlugin", mock.Anything, "p1").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)
				mslopl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))

				mk8stl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))
			},
			expLoadErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mslopl := fsmock.NewSLOPluginLoader(t)
			mslipl := fsmock.NewSLIPluginLoader(t)
			mk8stl := fsmock.NewK8sTransformPluginLoader(t)
			test.mock(mslopl, mslipl, mk8stl)

			// Create repository and load plugins.
			repo, err := storagefs.NewFilePluginRepo(log.Noop, test.failOnError, mslipl, mslopl, mk8stl, test.fss()...)
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

func TestFilePluginRepoGetSLOPlugin(t *testing.T) {
	tests := map[string]struct {
		pluginID  string
		fss       func() []fs.FS
		mock      func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader)
		expPlugin pluginengineslo.Plugin
		expErr    bool
	}{
		"Having no files, should fail.": {
			pluginID: "test",
			fss:      func() []fs.FS { return nil },
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
			},
			expErr: true,
		},

		"Getting a correct plugin, should return the plugin.": {
			pluginID: "p2",
			fss: func() []fs.FS {
				m := make(fstest.MapFS)
				m["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}
				return []fs.FS{m}
			},
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p1").Once().Return(nil, fmt.Errorf("something"))
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))

				mslopl.On("LoadRawPlugin", mock.Anything, "p1").Once().Return(&pluginengineslo.Plugin{ID: "p1"}, nil)
				mslopl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(&pluginengineslo.Plugin{ID: "p2"}, nil)
			},
			expPlugin: pluginengineslo.Plugin{ID: "p2"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mslopl := fsmock.NewSLOPluginLoader(t)
			mslipl := fsmock.NewSLIPluginLoader(t)
			mk8stl := fsmock.NewK8sTransformPluginLoader(t)
			test.mock(mslopl, mslipl, mk8stl)

			// Create repository and load plugins.
			repo, err := storagefs.NewFilePluginRepo(log.Noop, false, mslipl, mslopl, mk8stl, test.fss()...)
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

func TestFilePluginRepoListSLIPlugins(t *testing.T) {
	tests := map[string]struct {
		failOnError bool
		fss         func() []fs.FS
		mock        func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader)
		expPlugins  map[string]pluginenginesli.SLIPlugin
		expLoadErr  bool
		expErr      bool
	}{
		"Having no files, should return empty list of plugins.": {
			fss: func() []fs.FS { return nil },
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
			},
			expPlugins: map[string]pluginenginesli.SLIPlugin{},
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
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p1").Once().Return(&pluginenginesli.SLIPlugin{ID: "p1"}, nil)
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p2").Once().Return(&pluginenginesli.SLIPlugin{ID: "p2"}, nil)
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p3").Once().Return(&pluginenginesli.SLIPlugin{ID: "p3"}, nil)
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p4").Once().Return(&pluginenginesli.SLIPlugin{ID: "p4"}, nil)
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p5").Once().Return(&pluginenginesli.SLIPlugin{ID: "p5"}, nil)
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p6").Once().Return(&pluginenginesli.SLIPlugin{ID: "p6"}, nil)
			},
			expPlugins: map[string]pluginenginesli.SLIPlugin{
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
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p1").Once().Return(&pluginenginesli.SLIPlugin{ID: "p1"}, nil)
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p2").Once().Return(&pluginenginesli.SLIPlugin{ID: "p1"}, nil)
			},
			expLoadErr: true,
		},

		"Having an error while loading a plugin, should not fail but don't load the failed plugin.": {
			fss: func() []fs.FS {
				m1 := make(fstest.MapFS)
				m1["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m1["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}

				return []fs.FS{m1}
			},
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p1").Once().Return(&pluginenginesli.SLIPlugin{ID: "p1"}, nil)
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))
				mslopl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))
				mk8stl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))
			},
			expPlugins: map[string]pluginenginesli.SLIPlugin{
				"p1": {ID: "p1"},
			},
		},

		"Having an error while loading a plugin on strict mode, should fail.": {
			failOnError: true,
			fss: func() []fs.FS {
				m1 := make(fstest.MapFS)
				m1["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m1["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}

				return []fs.FS{m1}
			},
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p1").Once().Return(&pluginenginesli.SLIPlugin{ID: "p1"}, nil)
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))
				mslopl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))
				mk8stl.On("LoadRawPlugin", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))
			},
			expLoadErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mslopl := fsmock.NewSLOPluginLoader(t)
			mslipl := fsmock.NewSLIPluginLoader(t)
			mk8stl := fsmock.NewK8sTransformPluginLoader(t)
			test.mock(mslopl, mslipl, mk8stl)

			// Create repository and load plugins.
			repo, err := storagefs.NewFilePluginRepo(log.Noop, test.failOnError, mslipl, mslopl, mk8stl, test.fss()...)
			if test.expLoadErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			plugins, err := repo.ListSLIPlugins(t.Context())
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expPlugins, plugins)
			}
		})
	}
}

func TestFilePluginRepoGetSLIPlugin(t *testing.T) {
	tests := map[string]struct {
		pluginID  string
		fss       func() []fs.FS
		mock      func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader)
		expPlugin pluginenginesli.SLIPlugin
		expErr    bool
	}{
		"Having no files, should fail.": {
			pluginID: "test",
			fss:      func() []fs.FS { return nil },
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
			},
			expErr: true,
		},

		"Getting a correct plugin, should return the plugin.": {
			pluginID: "p2",
			fss: func() []fs.FS {
				m := make(fstest.MapFS)
				m["m1/pl1/plugin.go"] = &fstest.MapFile{Data: []byte("p1")}
				m["m1/pl2/plugin.go"] = &fstest.MapFile{Data: []byte("p2")}
				return []fs.FS{m}
			},
			mock: func(mslopl *fsmock.SLOPluginLoader, mslipl *fsmock.SLIPluginLoader, mk8stl *fsmock.K8sTransformPluginLoader) {
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p1").Once().Return(&pluginenginesli.SLIPlugin{ID: "p1"}, nil)
				mslipl.On("LoadRawSLIPlugin", mock.Anything, "p2").Once().Return(&pluginenginesli.SLIPlugin{ID: "p2"}, nil)
			},
			expPlugin: pluginenginesli.SLIPlugin{ID: "p2"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mslopl := fsmock.NewSLOPluginLoader(t)
			mslipl := fsmock.NewSLIPluginLoader(t)
			mk8stl := fsmock.NewK8sTransformPluginLoader(t)
			test.mock(mslopl, mslipl, mk8stl)

			// Create repository and load plugins.
			repo, err := storagefs.NewFilePluginRepo(log.Noop, false, mslipl, mslopl, mk8stl, test.fss()...)
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
