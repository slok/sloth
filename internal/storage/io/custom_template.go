package io

import (
	"context"
	"fmt"
	"io"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/registry/conversion"
	"github.com/go-sprout/sprout/registry/encoding" // Used to easy yaml encoding.
	"github.com/go-sprout/sprout/registry/maps"
	"github.com/go-sprout/sprout/registry/slices"
	"github.com/go-sprout/sprout/registry/std"
	"github.com/go-sprout/sprout/registry/strings"
	"github.com/go-sprout/sprout/registry/time"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/common/model"
)

func NewCustomGoTemplateRepo(writer io.Writer, logger log.Logger, tplData []byte) (*CustomGoTemplateRepo, error) {
	handler := sprout.New()
	err := handler.AddRegistries(
		conversion.NewRegistry(),
		std.NewRegistry(),
		encoding.NewRegistry(),
		maps.NewRegistry(),
		slices.NewRegistry(),
		strings.NewRegistry(),
		time.NewRegistry(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create sprout handler: %w", err)
	}

	tpl, err := template.New("slo").Funcs(handler.Build()).Parse(string(tplData))
	if err != nil {
		return nil, fmt.Errorf("could not parse custom Go template: %w", err)
	}

	return &CustomGoTemplateRepo{
		writer: writer,
		logger: logger.WithValues(log.Kv{"svc": "storageio.CustomGoTemplateRepo"}),
		tpl:    tpl,
	}, nil
}

type CustomGoTemplateRepo struct {
	writer io.Writer
	logger log.Logger
	tpl    *template.Template
}

// TplSLOGroupResult is the result of generating standard Prometheus SLO rules from SLO definitions as SLO group.
type TplSLOGroupResult struct {
	SLOGroup  model.PromSLOGroup
	SLOResult []TplSLOResult
}

// TplSLOResult is the result of generating standard Prometheus SLO rules from SLO definitions.
type TplSLOResult struct {
	SLO             model.PromSLO
	PrometheusRules model.PromSLORules
}

// StoreSLOs will store the recording and alert prometheus rules using a custom Go template.
func (r CustomGoTemplateRepo) StoreSLOs(ctx context.Context, extraTplContext map[string]string, result TplSLOGroupResult) error {
	if len(result.SLOResult) == 0 {
		return fmt.Errorf("slo rules required")
	}

	return r.tpl.Execute(r.writer, result)
}
