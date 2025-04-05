package custom

import (
	"reflect"
)

//go:generate yaegi extract --name custom github.com/prometheus/common/model github.com/prometheus/prometheus/model/rulefmt github.com/prometheus/prometheus/promql/parser
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/prometheus/plugin/slo/v1 github.com/slok/sloth/pkg/common/conventions github.com/slok/sloth/pkg/common/model github.com/slok/sloth/pkg/common/utils/data github.com/slok/sloth/pkg/common/utils/prometheus

// Symbols variable stores the map of custom Yaegi symbols per package.
var Symbols = map[string]map[string]reflect.Value{}
