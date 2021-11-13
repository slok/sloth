package tests

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chartDir = "../"

func TestChartServiceAccount(t *testing.T) {
	tests := map[string]struct {
		name       string
		valuesFile string
		namespace  string
		values     map[string]string
		expErr     bool
		expTplFile string
	}{
		"A chart without values should render correctly.": {
			name:       "sloth",
			valuesFile: "testdata/input/default.yaml",
			expTplFile: "testdata/output/sa_default.yaml",
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			valuesFile: "testdata/input/custom.yaml",
			namespace:  "custom",
			values:     map[string]string{},
			expTplFile: "testdata/output/sa_custom.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Prepare.
			options := &helm.Options{
				ValuesFiles:    []string{test.valuesFile},
				SetValues:      test.values,
				KubectlOptions: &k8s.KubectlOptions{Namespace: test.namespace},
			}

			// Execute.
			gotTpl, err := helm.RenderTemplateE(t, options, chartDir, test.name, []string{"templates/service-account.yaml"})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, gotTpl)
			}
		})
	}
}

func TestChartDeployment(t *testing.T) {
	tests := map[string]struct {
		name       string
		valuesFile string
		namespace  string
		values     map[string]string
		expErr     bool
		expTplFile string
	}{
		"A chart without values should render correctly.": {
			name:       "sloth",
			valuesFile: "testdata/input/default.yaml",
			expTplFile: "testdata/output/deployment_default.yaml",
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			valuesFile: "testdata/input/custom.yaml",
			namespace:  "custom",
			values:     map[string]string{},
			expTplFile: "testdata/output/deployment_custom.yaml",
		},

		"A chart with values without metrics and plugins should render correctly.": {
			name:       "test",
			valuesFile: "testdata/input/custom.yaml",
			namespace:  "custom",
			values: map[string]string{
				"commonPlugins.enabled": "false",
				"metrics.enabled":       "false",
			},
			expTplFile: "testdata/output/deployment_custom_no_extras.yaml",
		},

		"A chart with custom slo config should render correctly.": {
			name:       "test",
			valuesFile: "testdata/input/custom.yaml",
			namespace:  "custom",
			values: map[string]string{
				"commonPlugins.enabled":   "false",
				"customSloConfig.enabled": "true",
			},
			expTplFile: "testdata/output/deployment_custom_slo_config.yaml",
		},
	}

	checksumNormalizer := regexp.MustCompile(`checksum/config: [a-z0-9]+`)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Prepare.
			options := &helm.Options{
				ValuesFiles:    []string{test.valuesFile},
				SetValues:      test.values,
				KubectlOptions: &k8s.KubectlOptions{Namespace: test.namespace},
			}

			// Execute.
			gotTpl, err := helm.RenderTemplateE(t, options, chartDir, test.name, []string{"templates/deployment.yaml"})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				gotTpl := checksumNormalizer.ReplaceAllString(gotTpl, "checksum/config: <checksum>")

				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, gotTpl)
			}
		})
	}
}

func TestChartPodMonitor(t *testing.T) {
	tests := map[string]struct {
		name       string
		valuesFile string
		namespace  string
		values     map[string]string
		expErr     bool
		expTplFile string
	}{
		"A chart without values should render correctly.": {
			name:       "sloth",
			valuesFile: "testdata/input/default.yaml",
			expTplFile: "testdata/output/pod_monitor_default.yaml",
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			valuesFile: "testdata/input/custom.yaml",
			namespace:  "custom",
			values:     map[string]string{},
			expTplFile: "testdata/output/pod_monitor_custom.yaml",
		},

		"A chart with values without metrics and plugins should correctly.": {
			name:       "test",
			valuesFile: "testdata/input/custom.yaml",
			namespace:  "custom",
			values: map[string]string{
				"commonPlugins.enabled": "false",
				"metrics.enabled":       "false",
			},
			expErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Prepare.
			options := &helm.Options{
				ValuesFiles:    []string{test.valuesFile},
				SetValues:      test.values,
				KubectlOptions: &k8s.KubectlOptions{Namespace: test.namespace},
			}

			// Execute.
			gotTpl, err := helm.RenderTemplateE(t, options, chartDir, test.name, []string{"templates/pod-monitor.yaml"})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, gotTpl)
			}
		})
	}
}

func TestChartClusterRole(t *testing.T) {
	tests := map[string]struct {
		name       string
		valuesFile string
		namespace  string
		values     map[string]string
		expErr     bool
		expTplFile string
	}{
		"A chart without values should render correctly.": {
			name:       "sloth",
			valuesFile: "testdata/input/default.yaml",
			expTplFile: "testdata/output/cluster_role_default.yaml",
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			valuesFile: "testdata/input/custom.yaml",
			namespace:  "custom",
			values:     map[string]string{},
			expTplFile: "testdata/output/cluster_role_custom.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Prepare.
			options := &helm.Options{
				ValuesFiles:    []string{test.valuesFile},
				SetValues:      test.values,
				KubectlOptions: &k8s.KubectlOptions{Namespace: test.namespace},
			}

			// Execute.
			gotTpl, err := helm.RenderTemplateE(t, options, chartDir, test.name, []string{"templates/cluster-role.yaml"})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, gotTpl)
			}
		})
	}
}

func TestChartClusterRoleBinding(t *testing.T) {
	tests := map[string]struct {
		name       string
		valuesFile string
		namespace  string
		values     map[string]string
		expErr     bool
		expTplFile string
	}{
		"A chart without values should render correctly.": {
			name:       "sloth",
			valuesFile: "testdata/input/default.yaml",
			expTplFile: "testdata/output/cluster_role_binding_default.yaml",
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			valuesFile: "testdata/input/custom.yaml",
			namespace:  "custom",
			values:     map[string]string{},
			expTplFile: "testdata/output/cluster_role_binding_custom.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Prepare.
			options := &helm.Options{
				ValuesFiles:    []string{test.valuesFile},
				SetValues:      test.values,
				KubectlOptions: &k8s.KubectlOptions{Namespace: test.namespace},
			}

			// Execute.
			gotTpl, err := helm.RenderTemplateE(t, options, chartDir, test.name, []string{"templates/cluster-role-binding.yaml"})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, gotTpl)
			}
		})
	}
}

func TestChartConfigMap(t *testing.T) {
	tests := map[string]struct {
		name       string
		valuesFile string
		namespace  string
		values     map[string]string
		expErr     bool
		expTplFile string
	}{
		"A chart without values should not render a configmap.": {
			name:       "sloth",
			valuesFile: "testdata/input/default.yaml",
			expErr: true,
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			valuesFile: "testdata/input/custom.yaml",
			namespace:  "custom",
			values:     map[string]string{
				"customSloConfig.enabled": "true",
			},
			expTplFile: "testdata/output/configmap_slo_config.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Prepare.
			options := &helm.Options{
				ValuesFiles:    []string{test.valuesFile},
				SetValues:      test.values,
				KubectlOptions: &k8s.KubectlOptions{Namespace: test.namespace},
			}

			// Execute.
			gotTpl, err := helm.RenderTemplateE(t, options, chartDir, test.name, []string{"templates/configmap.yaml"})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, gotTpl)
			}
		})
	}
}
