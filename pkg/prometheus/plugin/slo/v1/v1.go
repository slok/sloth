package v1

import (
	"context"
	"encoding/json"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/common/model"
)

// Version is this plugin type version.
const Version = "prometheus/slo/v1"

// PluginVersion is the version of the plugin (e.g: `prometheus/slo/v1`).
type PluginVersion = string

const PluginVersionName = "PluginVersion"

// PluginID is the ID of the plugin (e.g: sloth.dev/my-test-plugin/v1).
type PluginID = string

const PluginIDName = "PluginID"

// AppUtils are app utils plugins can use in their logic.
type AppUtils struct {
	Logger         log.Logger
	QueryValidator model.QueryValidator
}

type Request struct {
	// Info about the application and execution, normally used as metadata.
	Info model.Info
	// The SLO to process and generate the final Prom rules.
	SLO model.PromSLO
	// The SLO MWMBAlertGroup selected.
	MWMBAlertGroup model.MWMBAlertGroup
}

type Result struct {
	SLORules model.PromSLORules
}

// PluginFactoryName is the required name for the plugin factory.
const PluginFactoryName = "NewPlugin"

type PluginFactory = func(config json.RawMessage, appUtils AppUtils) (Plugin, error)

// Plugin knows how to process SLOs in a chain of plugins.
// * The plugin processor can change the result argument of the SLO processing with the resulting prometheus rules.
// * The plugin processor can also modify the request object, but this is not recommended as it can lead to unexpected behavior.
//
// This is the type the SLO plugins need to implement.
type Plugin interface {
	ProcessSLO(ctx context.Context, request *Request, result *Result) error
}
