package slo_test

import (
	"testing"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/log"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	"github.com/slok/sloth/pkg/common/model"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

func TestPlugin(t *testing.T) {
	tests := map[string]struct {
		pluginSrc  string
		execPlugin func(t *testing.T, p pluginengineslo.Plugin)
		expErr     bool
	}{
		"Empty plugin should fail.": {
			pluginSrc:  "",
			execPlugin: func(t *testing.T, p pluginengineslo.Plugin) {},
			expErr:     true,
		},

		"An invalid plugin syntax should fail": {
			pluginSrc:  `package test{`,
			execPlugin: func(t *testing.T, p pluginengineslo.Plugin) {},
			expErr:     true,
		},

		"A plugin without the required version, should fail.": {
			pluginSrc: `package test
import (
	"context"
	"encoding/json"

	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v2"
	PluginID      = "sloth.dev/noop/v1"
)

func NewPlugin(_ json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	return noopPlugin{
		appUtils: appUtils,
	}, nil
}

type noopPlugin struct{
	appUtils pluginslov1.AppUtils
}

func (noopPlugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	return nil
}	
`,
			execPlugin: func(t *testing.T, p pluginengineslo.Plugin) {},
			expErr:     true,
		},

		"A plugin without the plugin ID, should fail.": {
			pluginSrc: `package test
import (
	"context"
	"encoding/json"

	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = ""
)

func NewPlugin(_ json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	return noopPlugin{
		appUtils: appUtils,
	}, nil
}

type noopPlugin struct{}

func (noopPlugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	return nil
}	
`,
			execPlugin: func(t *testing.T, p pluginengineslo.Plugin) {},
			expErr:     true,
		},

		"A plugin without the plugin factory, should fail.": {
			pluginSrc: `package test
import (
	"context"
	"encoding/json"

	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/noop/v1"
)

func NewPlugin2(_ json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	return noopPlugin{
		appUtils: appUtils,
	}, nil
}

type noopPlugin struct{}

func (noopPlugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	return nil
}	
`,
			execPlugin: func(t *testing.T, p pluginengineslo.Plugin) {},
			expErr:     true,
		},

		"A correct plugin should execute the plugin.": {
			pluginSrc: `package noopv1

import (
	"context"
	"encoding/json"

	"github.com/prometheus/prometheus/model/rulefmt"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/test/v1"
)

func NewPlugin(_ json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	return test{
		appUtils: appUtils,
	}, nil
}

type test struct{
	appUtils pluginslov1.AppUtils
}

func (test) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	result.SLORules.MetadataRecRules.Rules = []rulefmt.Rule{
		{Expr: "test1"},
		{Alert: "test2"},
	}
	
	return nil
}
`,
			execPlugin: func(t *testing.T, p pluginengineslo.Plugin) {
				var queryValidator model.QueryValidator
				queryValidator.MetricsQL = true
				plugin, err := p.PluginV1Factory(nil, pluginslov1.AppUtils{Logger: log.Noop, QueryValidator: queryValidator})
				require.NoError(t, err)
				gotResp := &pluginslov1.Result{}
				err = plugin.ProcessSLO(t.Context(), &pluginslov1.Request{}, gotResp)
				require.NoError(t, err)
				expResp := pluginslov1.Result{SLORules: model.PromSLORules{MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
					{Expr: "test1"},
					{Alert: "test2"},
				}}},
				}
				assert.Equal(t, expResp, *gotResp)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			plugin, err := pluginengineslo.PluginLoader.LoadRawPlugin(t.Context(), test.pluginSrc)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				test.execPlugin(t, *plugin)
			}
		})
	}
}
