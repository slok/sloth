package lib_test

import (
	"bytes"
	"testing"

	"github.com/slok/sloth/pkg/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateFromRawAlertKeyAliasesProduceIdenticalRules(t *testing.T) {
	tests := map[string]struct {
		canonical string
		alias     string
	}{
		"Prometheus v1 page_alert/ticket_alert vs fast_burn/slow_burn.": {
			canonical: `
version: "prometheus/v1"
service: "myservice"
slos:
  - name: "requests-availability"
    objective: 99.9
    sli:
      events:
        error_query: sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))
        total_query: sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))
    alerting:
      name: MyServiceHighErrorRate
      page_alert:
        labels:
          severity: critical
      ticket_alert:
        labels:
          severity: warning
`,
			alias: `
version: "prometheus/v1"
service: "myservice"
slos:
  - name: "requests-availability"
    objective: 99.9
    sli:
      events:
        error_query: sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))
        total_query: sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))
    alerting:
      name: MyServiceHighErrorRate
      fast_burn:
        labels:
          severity: critical
      slow_burn:
        labels:
          severity: warning
`,
		},

		"Kubernetes CRD pageAlert/ticketAlert vs fastBurn/slowBurn.": {
			canonical: `
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: myservice-slos
  namespace: monitoring
spec:
  service: "myservice"
  slos:
    - name: "requests-availability"
      objective: 99.9
      sli:
        events:
          errorQuery: sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))
          totalQuery: sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))
      alerting:
        name: MyServiceHighErrorRate
        pageAlert:
          labels:
            severity: critical
        ticketAlert:
          labels:
            severity: warning
`,
			alias: `
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: myservice-slos
  namespace: monitoring
spec:
  service: "myservice"
  slos:
    - name: "requests-availability"
      objective: 99.9
      sli:
        events:
          errorQuery: sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))
          totalQuery: sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))
      alerting:
        name: MyServiceHighErrorRate
        fastBurn:
          labels:
            severity: critical
        slowBurn:
          labels:
            severity: warning
`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			gen, err := lib.NewPrometheusSLOGenerator(lib.PrometheusSLOGeneratorConfig{
				CallerAgent: lib.CallerAgentCLI,
			})
			require.NoError(err)

			gotCanonical, err := gen.GenerateFromRaw(t.Context(), []byte(test.canonical))
			require.NoError(err)
			gotAlias, err := gen.GenerateFromRaw(t.Context(), []byte(test.alias))
			require.NoError(err)

			require.Equal(len(gotCanonical.SLOResults), len(gotAlias.SLOResults))
			for i := range gotCanonical.SLOResults {
				assert.Equal(gotCanonical.SLOResults[i].PrometheusRules, gotAlias.SLOResults[i].PrometheusRules)
				assert.Equal(gotCanonical.SLOResults[i].SLO.PageAlertMeta, gotAlias.SLOResults[i].SLO.PageAlertMeta)
				assert.Equal(gotCanonical.SLOResults[i].SLO.TicketAlertMeta, gotAlias.SLOResults[i].SLO.TicketAlertMeta)
			}

			var canonicalYAML, aliasYAML bytes.Buffer
			require.NoError(gen.WriteResultAsPrometheusStd(t.Context(), *gotCanonical, &canonicalYAML))
			require.NoError(gen.WriteResultAsPrometheusStd(t.Context(), *gotAlias, &aliasYAML))
			assert.Equal(canonicalYAML.String(), aliasYAML.String())

			out := aliasYAML.String()
			assert.Contains(out, "sloth_severity: page")
			assert.Contains(out, "sloth_severity: ticket")
			assert.NotContains(out, "sloth_severity: fast_burn")
			assert.NotContains(out, "sloth_severity: slow_burn")
			assert.NotContains(out, "sloth_severity: fastBurn")
			assert.NotContains(out, "sloth_severity: slowBurn")
			assert.Contains(out, "(page)")
			assert.Contains(out, "(ticket)")

			sawPage, sawTicket := false, false
			for _, res := range gotAlias.SLOResults {
				for _, rule := range res.PrometheusRules.AlertRules.Rules {
					switch rule.Labels["sloth_severity"] {
					case "page":
						sawPage = true
						assert.Contains(rule.Annotations["title"], "(page)")
					case "ticket":
						sawTicket = true
						assert.Contains(rule.Annotations["title"], "(ticket)")
					case "fast_burn", "slow_burn", "fastBurn", "slowBurn":
						t.Errorf("GenerateFromRaw() sloth_severity = %q, want page or ticket", rule.Labels["sloth_severity"])
					}
					for k, v := range rule.Labels {
						if v == "fast_burn" || v == "slow_burn" || v == "fastBurn" || v == "slowBurn" {
							t.Errorf("GenerateFromRaw() rule label %s=%q, want no burn-rate alias values", k, v)
						}
					}
				}
			}
			assert.True(sawPage, "GenerateFromRaw() missing sloth_severity=page")
			assert.True(sawTicket, "GenerateFromRaw() missing sloth_severity=ticket")
		})
	}
}

func TestGenerateFromRawConflictingAlertKeysFail(t *testing.T) {
	tests := map[string]struct {
		spec           string
		expErrContains []string
	}{
		"Prometheus v1 page_alert and fast_burn together should fail.": {
			spec: `
version: "prometheus/v1"
service: "myservice"
slos:
  - name: "requests-availability"
    objective: 99.9
    sli:
      events:
        error_query: sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))
        total_query: sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))
    alerting:
      name: MyServiceHighErrorRate
      page_alert:
        labels:
          severity: critical
      fast_burn:
        labels:
          severity: critical
`,
			expErrContains: []string{"page_alert", "fast_burn"},
		},
		"Kubernetes CRD pageAlert and fastBurn together should fail.": {
			spec: `
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: myservice-slos
spec:
  service: "myservice"
  slos:
    - name: "requests-availability"
      objective: 99.9
      sli:
        events:
          errorQuery: sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))
          totalQuery: sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))
      alerting:
        name: MyServiceHighErrorRate
        pageAlert:
          labels:
            severity: critical
        fastBurn:
          labels:
            severity: critical
`,
			expErrContains: []string{"pageAlert", "fastBurn"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			gen, err := lib.NewPrometheusSLOGenerator(lib.PrometheusSLOGeneratorConfig{
				CallerAgent: lib.CallerAgentCLI,
			})
			require.NoError(err)

			_, err = gen.GenerateFromRaw(t.Context(), []byte(test.spec))
			if !assert.Error(err) {
				return
			}
			for _, s := range test.expErrContains {
				assert.Contains(err.Error(), s)
			}
		})
	}
}
