package io

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"

	kubernetesmodelmap "github.com/slok/sloth/internal/kubernetes/modelmap"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/storage"
	"github.com/slok/sloth/pkg/common/model"
)

func NewIOWriterPrometheusOperatorYAMLRepo(writer io.Writer, logger log.Logger) IOWriterPrometheusOperatorYAMLRepo {
	return IOWriterPrometheusOperatorYAMLRepo{
		writer:  writer,
		encoder: json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil),
		logger:  logger.WithValues(log.Kv{"svc": "storage.io.IOWriterPrometheusOperatorYAMLRepo"}),
	}
}

// IOWriterPrometheusOperatorYAMLRepo knows to store all the SLO rules (recordings and alerts)
// grouped in an IOWriter in Kubernetes prometheus operator YAML format.
type IOWriterPrometheusOperatorYAMLRepo struct {
	writer  io.Writer
	encoder runtime.Encoder
	logger  log.Logger
}

func (i IOWriterPrometheusOperatorYAMLRepo) StoreSLOs(ctx context.Context, kmeta storage.K8sMeta, slos model.PromSLOGroupResult) error {
	rule, err := kubernetesmodelmap.MapModelToPrometheusOperator(ctx, kmeta, slos)
	if err != nil {
		return fmt.Errorf("could not map model to Prometheus operator CR: %w", err)
	}

	var b bytes.Buffer
	err = i.encoder.Encode(rule, &b)
	if err != nil {
		return fmt.Errorf("could encode prometheus operator object: %w", err)
	}

	rulesYaml := writeYAMLTopDisclaimer(b.Bytes())
	_, err = i.writer.Write(rulesYaml)
	if err != nil {
		return fmt.Errorf("could not write top disclaimer: %w", err)
	}

	return nil
}
