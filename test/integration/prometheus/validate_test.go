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
			valCmdArgs: "--input ./testdata/validate/good --ignore-slo-duplicates",
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
			valCmdArgs: "--input ./testdata/validate --fs-exclude bad --ignore-slo-duplicates",
		},

		"Discovery of all specs including only good should validate correctly.": {
			valCmdArgs: "--input ./testdata/validate --fs-include good --ignore-slo-duplicates",
		},

		"Discovery of none specs should fail.": {
			valCmdArgs: "--input ./testdata/validate --fs-exclude .*",
			expErr:     true,
		},

		"Discovery of all specs excluding bad and including a bad one should validate correctly because exclude has preference.": {
			valCmdArgs: "--input ./testdata/validate --fs-exclude bad --fs-include .*-aa.* --ignore-slo-duplicates",
		},

		"Discovery of specs with duplicates should validate with failures.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates",
			expErr:     true,
		},

		"Discovery of specs with duplicates and ignore flag should validate correctly.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates --ignore-slo-duplicates",
			expErr:     false,
		},

		"Discovery of prom specs with duplicates should validate with failures.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/prom",
			expErr:     true,
		},

		"Discovery of k8s specs with duplicates should validate with failures.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/k8s",
			expErr:     true,
		},

		"Discovery of openslo specs with duplicates should validate with failures.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/openslo",
			expErr:     true,
		},

		"Discovery of multi-file prom specs with duplicates should validate with failures.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/bad-prom-multi-duplicates.yaml",
			expErr:     true,
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
