package lib_test

import (
	"bytes"
	"io/fs"
	"os"
	"testing"
	"text/template"
	"time"

	"github.com/slok/sloth/internal/info"
	"github.com/slok/sloth/pkg/common/model"
	"github.com/slok/sloth/pkg/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests try being as real as possible by using the library to tests against the CLI integration tests.
// Regarding the features, the CLI offers file and IO operations that the library doesn't offer. However the
// core SLO generator logic is the same (the CLI uses the public library under the hood).
func TestLibAsCLIIntegration(t *testing.T) {
	testWindowsFS := os.DirFS("../../test/integration/prometheus/windows")
	testPluginsFS := os.DirFS("../../test/integration/prometheus/plugins")

	tests := map[string]struct {
		config          func() lib.PrometheusSLOGeneratorConfig
		inFilePath      string
		resultFormatter func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte
		expOutFilePath  string
		expGenErr       bool
	}{
		"Invalid spec case.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{CallerAgent: lib.CallerAgentCLI}
			},
			inFilePath: "../../test/integration/prometheus/testdata/in-invalid-version.yaml",
			expGenErr:  true,
		},

		"Prometheus case.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{CallerAgent: lib.CallerAgentCLI}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-base.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-base.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				var b bytes.Buffer
				err := gen.WriteResultAsPrometheusStd(t.Context(), result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"Kubernetes case.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{CallerAgent: lib.CallerAgentCLI}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-base-k8s.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-base-k8s.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				var b bytes.Buffer
				kmeta := model.K8sMeta{Name: "svc", Namespace: "test-ns"}
				err := gen.WriteResultAsK8sPrometheusOperator(t.Context(), kmeta, result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"OpenSLO case.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{CallerAgent: lib.CallerAgentCLI}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-openslo.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-openslo.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				var b bytes.Buffer
				err := gen.WriteResultAsPrometheusStd(t.Context(), result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"Default 28d window period case.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{
					DefaultSLOPeriod: 28 * 24 * time.Hour,
					CallerAgent:      lib.CallerAgentCLI,
				}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-base.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-base-28d.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				var b bytes.Buffer
				err := gen.WriteResultAsPrometheusStd(t.Context(), result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"Custom 7d window period case.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{
					DefaultSLOPeriod: 7 * 24 * time.Hour,
					CallerAgent:      lib.CallerAgentCLI,
					WindowsFS:        testWindowsFS,
				}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-base.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-base-custom-windows-7d.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				var b bytes.Buffer
				err := gen.WriteResultAsPrometheusStd(t.Context(), result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"Extra labels case.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{
					CallerAgent: lib.CallerAgentCLI,
					ExtraLabels: map[string]string{"exk1": "exv1", "exk2": "exv2"},
				}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-base.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-base-extra-labels.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				var b bytes.Buffer
				err := gen.WriteResultAsPrometheusStd(t.Context(), result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"No alerts case.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{CallerAgent: lib.CallerAgentCLI}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-base.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-base-no-alerts.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {

				for i := range result.SLOResults {
					result.SLOResults[i].PrometheusRules.AlertRules = model.PromRuleGroup{}
				}

				var b bytes.Buffer
				err := gen.WriteResultAsPrometheusStd(t.Context(), result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"No recording rules case.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{CallerAgent: lib.CallerAgentCLI}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-base.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-base-no-recordings.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				// Remove alerts.
				for i := range result.SLOResults {
					result.SLOResults[i].PrometheusRules.SLIErrorRecRules = model.PromRuleGroup{}
					result.SLOResults[i].PrometheusRules.MetadataRecRules = model.PromRuleGroup{}
				}

				var b bytes.Buffer
				err := gen.WriteResultAsPrometheusStd(t.Context(), result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"SLI plugin usage.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{
					CallerAgent: lib.CallerAgentCLI,
					PluginsFS:   []fs.FS{testPluginsFS},
				}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-sli-plugin.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-sli-plugin.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				var b bytes.Buffer
				err := gen.WriteResultAsPrometheusStd(t.Context(), result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"SLO plugin usage.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{
					CallerAgent: lib.CallerAgentCLI,
					PluginsFS:   []fs.FS{testPluginsFS},
				}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-slo-plugin.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-slo-plugin.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				var b bytes.Buffer
				err := gen.WriteResultAsPrometheusStd(t.Context(), result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"SLO plugin K8s usage.": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{
					CallerAgent: lib.CallerAgentCLI,
					PluginsFS:   []fs.FS{testPluginsFS},
				}
			},
			inFilePath:     "../../test/integration/prometheus/testdata/in-slo-plugin-k8s.yaml",
			expOutFilePath: "../../test/integration/prometheus/testdata/out-slo-plugin-k8s.yaml.tpl",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				var b bytes.Buffer
				kmeta := model.K8sMeta{Name: "svc", Namespace: "test-ns"}
				err := gen.WriteResultAsK8sPrometheusOperator(t.Context(), kmeta, result, &b)
				require.NoError(t, err)
				return b.Bytes()
			},
		},

		"A multifile case (Not supported).": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{CallerAgent: lib.CallerAgentCLI}
			},
			inFilePath: "../../test/integration/prometheus/testdata/in-multifile.yaml",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				return nil
			},
			expGenErr: true,
		},

		"A multifile Kubernetes case (Not supported).": {
			config: func() lib.PrometheusSLOGeneratorConfig {
				return lib.PrometheusSLOGeneratorConfig{CallerAgent: lib.CallerAgentCLI}
			},
			inFilePath: "../../test/integration/prometheus/testdata/in-multifile-k8s.yaml",
			resultFormatter: func(t *testing.T, gen *lib.PrometheusSLOGenerator, result model.PromSLOGroupResult) []byte {
				return nil
			},
			expGenErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			gen, err := lib.NewPrometheusSLOGenerator(test.config())
			require.NoError(err)

			// Generate.
			expInData, err := os.ReadFile(test.inFilePath)
			require.NoError(err)
			result, err := gen.GenerateFromRaw(t.Context(), expInData)
			if test.expGenErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				// Check result.
				resultOutData := test.resultFormatter(t, gen, *result)
				expOutData := getExpData(t, test.expOutFilePath)
				assert.Equal(string(expOutData), string(resultOutData))
			}
		})
	}
}

func getExpData(t *testing.T, path string) []byte {
	expOutData, err := os.ReadFile(path)
	require.NoError(t, err)

	var b bytes.Buffer
	err = template.Must(template.New("").Parse(string(expOutData))).Execute(&b, map[string]string{
		"version": info.Version,
	})
	require.NoError(t, err)

	return b.Bytes()
}
