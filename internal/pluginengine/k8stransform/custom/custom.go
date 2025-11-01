package custom

import (
	"reflect"

	_ "github.com/caarlos0/env/v11" // Used only by yaegi plugins, not by Sloth.
)

//go:generate yaegi extract --name custom github.com/caarlos0/env/v11

//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/common/conventions
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/common/model
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/common/utils/data
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/common/utils/prometheus
//go:generate yaegi extract --name custom github.com/slok/sloth/pkg/common/utils/k8s

//go:generate yaegi extract --name custom k8s.io/apimachinery/pkg/apis/meta/v1/unstructured

// Symbols variable stores the map of custom Yaegi symbols per package.
var Symbols = map[string]map[string]reflect.Value{}
