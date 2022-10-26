package tests

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/slok/go-helm-template/helm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var slothChart = helm.MustLoadChart(context.TODO(), os.DirFS("../"))

var versionNormalizer = regexp.MustCompile(`helm\.sh/chart: sloth-[0-9\\.]+`)

func normalizeVersion(tpl string) string {
	return versionNormalizer.ReplaceAllString(tpl, "helm.sh/chart: sloth-<version>")
}

func TestChartServiceAccount(t *testing.T) {
	tests := map[string]struct {
		name       string
		namespace  string
		values     func() map[string]interface{}
		expErr     bool
		expTplFile string
	}{
		"A chart without values should render correctly.": {
			name:       "sloth",
			namespace:  "default",
			values:     defaultValues,
			expTplFile: "testdata/output/sa_default.yaml",
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			namespace:  "custom",
			values:     customValues,
			expTplFile: "testdata/output/sa_custom.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Execute.
			gotTpl, err := helm.Template(context.TODO(), helm.TemplateConfig{
				Chart:       slothChart,
				Namespace:   test.namespace,
				ReleaseName: test.name,
				Values:      test.values(),
				ShowFiles:   []string{"templates/service-account.yaml"},
			})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, normalizeVersion(gotTpl))
			}
		})
	}
}

func TestChartDeployment(t *testing.T) {
	checksumNormalizer := regexp.MustCompile(`checksum/config: [a-z0-9]+`)

	tests := map[string]struct {
		name       string
		namespace  string
		values     func() map[string]interface{}
		expErr     bool
		expTplFile string
	}{
		"A chart without values should render correctly.": {
			name:       "sloth",
			namespace:  "default",
			values:     defaultValues,
			expTplFile: "testdata/output/deployment_default.yaml",
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			namespace:  "custom",
			values:     customValues,
			expTplFile: "testdata/output/deployment_custom.yaml",
		},

		"A chart with values without metrics and plugins should render correctly.": {
			name:      "test",
			namespace: "custom",
			values: func() map[string]interface{} {
				v := customValues()
				v["commonPlugins"].(msi)["enabled"] = false
				v["metrics"].(msi)["enabled"] = false

				return v
			},
			expTplFile: "testdata/output/deployment_custom_no_extras.yaml",
		},

		"A chart with custom slo config should render correctly.": {
			name:      "test",
			namespace: "custom",
			values: func() map[string]interface{} {
				v := customValues()
				v["commonPlugins"].(msi)["enabled"] = false
				v["customSloConfig"].(msi)["enabled"] = true

				return v
			},
			expTplFile: "testdata/output/deployment_custom_slo_config.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			gotTpl, err := helm.Template(context.TODO(), helm.TemplateConfig{
				Chart:       slothChart,
				Namespace:   test.namespace,
				ReleaseName: test.name,
				Values:      test.values(),
				ShowFiles:   []string{"templates/deployment.yaml"},
			})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				gotTpl := checksumNormalizer.ReplaceAllString(gotTpl, "checksum/config: <checksum>")

				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, normalizeVersion(gotTpl))
			}
		})
	}
}

func TestChartPodMonitor(t *testing.T) {
	tests := map[string]struct {
		name       string
		namespace  string
		values     func() map[string]interface{}
		expErr     bool
		expTplFile string
	}{
		"A chart without values should render correctly.": {
			name:       "sloth",
			namespace:  "default",
			values:     defaultValues,
			expTplFile: "testdata/output/pod_monitor_default.yaml",
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			namespace:  "custom",
			values:     customValues,
			expTplFile: "testdata/output/pod_monitor_custom.yaml",
		},

		"A chart with values without metrics and plugins should correctly.": {
			name:      "test",
			namespace: "custom",
			values: func() map[string]interface{} {
				v := customValues()
				v["commonPlugins"].(msi)["enabled"] = false
				v["metrics"].(msi)["enabled"] = false

				return v
			},
			expErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Execute.
			gotTpl, err := helm.Template(context.TODO(), helm.TemplateConfig{
				Chart:       slothChart,
				Namespace:   test.namespace,
				ReleaseName: test.name,
				Values:      test.values(),
				ShowFiles:   []string{"templates/pod-monitor.yaml"},
			})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, normalizeVersion(gotTpl))
			}
		})
	}
}

func TestChartClusterRole(t *testing.T) {
	tests := map[string]struct {
		name       string
		namespace  string
		values     func() map[string]interface{}
		expErr     bool
		expTplFile string
	}{
		"A chart without values should render correctly.": {
			name:       "sloth",
			namespace:  "default",
			values:     defaultValues,
			expTplFile: "testdata/output/cluster_role_default.yaml",
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			namespace:  "custom",
			values:     customValues,
			expTplFile: "testdata/output/cluster_role_custom.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Execute.
			gotTpl, err := helm.Template(context.TODO(), helm.TemplateConfig{
				Chart:       slothChart,
				Namespace:   test.namespace,
				ReleaseName: test.name,
				Values:      test.values(),
				ShowFiles:   []string{"templates/cluster-role.yaml"},
			})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, normalizeVersion(gotTpl))
			}
		})
	}
}

func TestChartClusterRoleBinding(t *testing.T) {
	tests := map[string]struct {
		name       string
		namespace  string
		values     func() map[string]interface{}
		expErr     bool
		expTplFile string
	}{
		"A chart without values should render correctly.": {
			name:       "sloth",
			namespace:  "default",
			values:     defaultValues,
			expTplFile: "testdata/output/cluster_role_binding_default.yaml",
		},

		"A chart with custom values should render correctly.": {
			name:       "test",
			namespace:  "custom",
			values:     customValues,
			expTplFile: "testdata/output/cluster_role_binding_custom.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Execute.
			gotTpl, err := helm.Template(context.TODO(), helm.TemplateConfig{
				Chart:       slothChart,
				Namespace:   test.namespace,
				ReleaseName: test.name,
				Values:      test.values(),
				ShowFiles:   []string{"templates/cluster-role-binding.yaml"},
			})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, normalizeVersion(gotTpl))
			}
		})
	}
}

func TestChartConfigMap(t *testing.T) {
	tests := map[string]struct {
		name       string
		namespace  string
		values     func() map[string]interface{}
		expErr     bool
		expTplFile string
	}{
		"A chart without values should not render a configmap.": {
			name:      "sloth",
			namespace: "default",
			values:    defaultValues,
			expErr:    true,
		},

		"A chart with custom values should render correctly.": {
			name:      "test",
			namespace: "custom",
			values: func() map[string]interface{} {
				v := customValues()
				v["customSloConfig"].(msi)["enabled"] = true

				return v
			},
			expTplFile: "testdata/output/configmap_slo_config.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Execute.
			gotTpl, err := helm.Template(context.TODO(), helm.TemplateConfig{
				Chart:       slothChart,
				Namespace:   test.namespace,
				ReleaseName: test.name,
				Values:      test.values(),
				ShowFiles:   []string{"templates/configmap.yaml"},
			})

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				expTpl, err := os.ReadFile(test.expTplFile)
				require.NoError(err)
				expTplS := strings.TrimSpace(string(expTpl))

				assert.Equal(expTplS, normalizeVersion(gotTpl))
			}
		})
	}
}
