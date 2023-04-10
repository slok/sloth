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
		"01 Discovery of good specs should validate correctly.": {
			valCmdArgs: "--input ./testdata/validate/good --ignore-slo-duplicates",
		},

		"02 Discovery of bad specs should validate with failures.": {
			valCmdArgs: "--input ./testdata/validate/bad --ignore-slo-duplicates",
			expErr:     true,
		},

		"03 Discovery of all specs should validate with failures.": {
			valCmdArgs: "--input ./testdata/validate",
			expErr:     true,
		},

		"04 Discovery of all specs excluding bads should validate correctly.": {
			valCmdArgs: "--input ./testdata/validate --fs-exclude bad --ignore-slo-duplicates",
		},

		"05 Discovery of all specs including only good should validate correctly.": {
			valCmdArgs: "--input ./testdata/validate --fs-include good --ignore-slo-duplicates",
		},

		"06 Discovery of none specs should fail.": {
			valCmdArgs: "--input ./testdata/validate --fs-exclude .*",
			expErr:     true,
		},

		"07 Discovery of all specs excluding bad and including a bad one should validate correctly because exclude has preference.": {
			valCmdArgs: "--input ./testdata/validate --fs-exclude bad --fs-include .*-aa.*  --ignore-slo-duplicates",
		},

		"DUP_01_A It fails when finds bad Prom specs with slo duplicates.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/prom",
			expErr:     true,
		},

		"DUP_01_B It fails when finds bad Prom multifile spec with slo duplicates.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/bad-prom-multi-duplicates.yaml",
			expErr:     true,
		},

		"DUP_02 It fails when finds bad K8S specs with slo duplicates.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/k8s",
			expErr:     true,
		},

		"DUP_03 It fails when finds bad OpenSLO specs with slo duplicates.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/openslo",
			expErr:     true,
		},

		"DUP_04_A It succeeds on Prom specs having slo duplicates if ignore-slo-duplicates specified.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/prom --ignore-slo-duplicates",
		},

		"DUP_04_B It succeeds on Prom multifile specs having slo duplicates if ignore-slo-duplicates specified.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/bad-prom-multi-duplicates.yaml --ignore-slo-duplicates",
		},

		"DUP_05 It succeeds on K8S specs having slo duplicates if ignore-slo-duplicates specified.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/k8s --ignore-slo-duplicates",
		},

		"DUP_06 It succeeds on OpenSLO specs having slo duplicates if ignore-slo-duplicates specified.": {
			valCmdArgs: "--input ./testdata/validate_with_duplicates/openslo --ignore-slo-duplicates",
		},

		"DUP_07 It succeeds on a good Prom multifile spec without slo duplicates.": {
			valCmdArgs: "--input ./testdata/validate/good/good-multi.yaml",
		},

		"DUP_08 It succeeds on a good K8S multifile spec without slo duplicates.": {
			valCmdArgs: "--input ./testdata/validate/good/good-multi-k8s.yaml",
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
