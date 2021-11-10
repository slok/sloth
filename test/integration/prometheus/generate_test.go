package prometheus_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/test/integration/prometheus"
	"github.com/slok/sloth/test/integration/testutils"
)

type expecteOutLoader struct {
	version string
}

func (e expecteOutLoader) mustLoadExp(path string) string {
	fileData, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	tmpl := template.Must(template.New("").Parse(string(fileData)))

	data := map[string]string{"version": e.version}
	var b bytes.Buffer
	err = tmpl.Execute(&b, data)
	if err != nil {
		panic(err)
	}

	return b.String()
}

func TestPrometheusGenerate(t *testing.T) {
	// Tests config.
	config := prometheus.NewConfig(t)
	version, err := testutils.SlothVersion(context.TODO(), config.Binary)
	require.NoError(t, err)

	expectLoader := expecteOutLoader{version: version}

	// Tests.
	tests := map[string]struct {
		genCmdArgs string
		expOut     string
		expErr     bool
	}{
		"Generate should generate the correct rules for all the SLOs.": {
			genCmdArgs: "--input ./testdata/in-base.yaml",
			expOut:     expectLoader.mustLoadExp("./testdata/out-base.yaml.tpl"),
		},

		"Generate should generate the correct rules for all the SLOs (Kubernetes).": {
			genCmdArgs: "--input ./testdata/in-base-k8s.yaml",
			expOut:     expectLoader.mustLoadExp("./testdata/out-base-k8s.yaml.tpl"),
		},

		"Generate without alerts should generate the correct recording rules for all the SLOs.": {
			genCmdArgs: "--input ./testdata/in-base.yaml --disable-alerts",
			expOut:     expectLoader.mustLoadExp("./testdata/out-base-no-alerts.yaml.tpl"),
		},

		"Generate without recordings should generate the correct alert rules for all the SLOs.": {
			genCmdArgs: "--input ./testdata/in-base.yaml --disable-recordings",
			expOut:     expectLoader.mustLoadExp("./testdata/out-base-no-recordings.yaml.tpl"),
		},

		"Generate with extra labels should generate the correct rules for all the SLOs.": {
			genCmdArgs: "--input ./testdata/in-base.yaml -l exk1=exv1 -l exk2=exv2",
			expOut:     expectLoader.mustLoadExp("./testdata/out-base-extra-labels.yaml.tpl"),
		},

		"Generate with plugins should generate the correct rules for all the SLOs.": {
			genCmdArgs: "--input ./testdata/in-plugin.yaml",
			expOut:     expectLoader.mustLoadExp("./testdata/out-plugin.yaml.tpl"),
		},

		"Generate using multifile YAML in single file should generate the correct rules for all the SLOs.": {
			genCmdArgs: "--input ./testdata/in-multifile.yaml",
			expOut:     expectLoader.mustLoadExp("./testdata/out-multifile.yaml.tpl"),
		},

		"Generate using multifile YAML in single file should generate the correct rules for all the SLOs (Kubernetes).": {
			genCmdArgs: "--input ./testdata/in-multifile-k8s.yaml",
			expOut:     expectLoader.mustLoadExp("./testdata/out-multifile-k8s.yaml.tpl"),
		},

		"Generate using OpenSLO YAML should generate Prometheus rules.": {
			genCmdArgs: "--input ./testdata/in-openslo.yaml",
			expOut:     expectLoader.mustLoadExp("./testdata/out-openslo.yaml.tpl"),
		},

		"Generate using 28 day time window should generate Prometheus rules.": {
			genCmdArgs: "--default-slo-period 28d --input ./testdata/in-base.yaml",
			expOut:     expectLoader.mustLoadExp("./testdata/out-base-28d.yaml.tpl"),
		},

		"Generate using custom 7 day time window should generate Prometheus rules.": {
			genCmdArgs: "--default-slo-period 7d --input ./testdata/in-base.yaml --slo-period-windows-path ./windows",
			expOut:     expectLoader.mustLoadExp("./testdata/out-base-custom-windows-7d.yaml.tpl"),
		},

		"Generate using invalid version should fail.": {
			genCmdArgs: "--input ./testdata/in-invalid-version.yaml",
			expErr:     true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			// Run with context to stop on test end.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			out, _, err := prometheus.RunSlothGenerate(ctx, config, test.genCmdArgs)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expOut, string(out))
			}
		})
	}
}
