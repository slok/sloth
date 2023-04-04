package prometheus_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/test/integration/prometheus"
)

func TestPrometheusValidate(t *testing.T) {
	// Tests config.
	config := prometheus.NewConfig(t)

	// Tests.
	tests := map[string]struct {
		valCmdArgs string
		expErr     bool
	}{
		"Discovery of good specs should validate correctly.": {
			valCmdArgs: "--input ./testdata/validate/good",
		},

		"Discovery of bad specs should validate with failures.": {
			valCmdArgs: "--input ./testdata/validate/bad",
			expErr:     true,
		},

		"Discovery of all specs should validate with failures.": {
			valCmdArgs: "--input ./testdata/validate",
			expErr:     true,
		},

		"Discovery of all specs excluding bads should validate correctly.": {
			valCmdArgs: "--input ./testdata/validate --fs-exclude bad",
		},

		"Discovery of all specs including only good should validate correctly.": {
			valCmdArgs: "--input ./testdata/validate --fs-include good",
		},

		"Discovery of none specs should fail.": {
			valCmdArgs: "--input ./testdata/validate --fs-exclude .*",
			expErr:     true,
		},

		"Discovery of all specs excluding bad and including a bad one should validate correctly because exclude has preference.": {
			valCmdArgs: "--input ./testdata/validate --fs-exclude bad --fs-include .*-aa.*",
		},

		"Discovery of bad Prom specs with duplicates should validate with failures.": {
			valCmdArgs: "--input ./testdata/validate/bad/duplicates/prom",
		},

		"Discovery of bad K8S specs with duplicates k8s validate with failures.": {
			valCmdArgs: "--input ./testdata/validate/bad/duplicates/prom",
		},

		"Discovery of bad K8S specs with duplicates OpenSlo validate with failures.": {
			valCmdArgs: "--input ./testdata/validate/bad/duplicates/openslo",
		},

		"Discovery of bad Prom multifile specs with duplicates validate with failures.": {
			valCmdArgs: "--input ./testdata/validate/bad/duplicates/bad-prom-multi-duplicates.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			// Run with context to stop on test end.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			_, _, err := prometheus.RunSlothValidate(ctx, config, test.valCmdArgs)

			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
