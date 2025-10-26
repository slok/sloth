package lib

import (
	"context"
	"io"

	storageio "github.com/slok/sloth/internal/storage/io"
	"github.com/slok/sloth/pkg/common/model"
	"github.com/slok/sloth/pkg/lib/log"
)

// WriteResultAsPrometheusStd writes the SLO results into the writer as a Prometheus standard rules file.
// More information in:
//   - https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#recording-rules.
//   - https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/.
func WriteResultAsPrometheusStd(ctx context.Context, slo model.PromSLOGroupResult, w io.Writer) error {
	repo := storageio.NewStdPrometheusGroupedRulesYAMLRepo(w, log.Noop)
	return repo.StoreSLOs(ctx, slo)
}

// WriteResultAsK8sPrometheusOperator writes the SLO results into the writer as a Prometheus Operator CRD file.
// More information in: https://prometheus-operator.dev/docs/api-reference/api/#monitoring.coreos.com/v1.PrometheusRule.
func WriteResultAsK8sPrometheusOperator(ctx context.Context, k8sMeta model.K8sMeta, slo model.PromSLOGroupResult, w io.Writer) error {
	repo := storageio.NewIOWriterPrometheusOperatorYAMLRepo(w, log.Noop)
	return repo.StoreSLOs(ctx, k8sMeta, slo)
}
