package custom

import (
	"reflect"

	_ "github.com/caarlos0/env/v11" // Used only by yaegi plugins, not by Sloth.
)

//go:generate yaegi extract --name custom github.com/caarlos0/env/v11

//go:generate yaegi extract --name custom github.com/prometheus/common/model
//go:generate yaegi extract --name custom github.com/prometheus/prometheus/model/rulefmt
//go:generate yaegi extract --name custom github.com/prometheus/prometheus/promql/parser

//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/prometheus/plugin/slo/v1
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/common/conventions
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/common/model
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/common/utils/data
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/common/utils/prometheus
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/common/validation

//go:generate yaegi extract --name custom github.com/VictoriaMetrics/metricsql

// Symbols variable stores the map of custom Yaegi symbols per package.
var Symbols = map[string]map[string]reflect.Value{}
