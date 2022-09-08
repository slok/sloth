package k8sprometheus

import (
	"context"
	"github.com/slok/sloth/internal/prometheus"
)

type SLIPluginRepo interface {
	GetSLIPlugin(ctx context.Context, id string) (*prometheus.SLIPlugin, error)
}

type YamlSpecLoader interface {
	IsSpecType(ctx context.Context, data []byte) bool
	LoadSpec(ctx context.Context, data []byte) (*SLOGroup, error)
}
