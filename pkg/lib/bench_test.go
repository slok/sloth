package lib_test

import (
	"context"
	"io"
	"testing"

	"github.com/slok/sloth/pkg/lib"
)

func BenchmarkLibGenerateAndWrite(b *testing.B) {
	const sloSpec = `
---
version: "prometheus/v1"
service: "myservice"
labels:
  owner: "myteam"
  repo: "myorg/myservice"
  tier: "2"
slos:
  # We allow failing (5xx and 429) 1 request every 1000 requests (99.9%).
  - name: "requests-availability"
    objective: 99.9
    description: "Common SLO based on availability for HTTP request responses."
    labels:
      category: availability
    sli:
      events:
        error_query: sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))
        total_query: sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))
    alerting:
      name: "MyServiceHighErrorRate"
      labels:
        category: "availability"
      annotations:
        # Overwrite default Sloth SLO alert summmary on ticket and page alerts.
        summary: "High error rate on 'myservice' requests responses"
      page_alert:
        labels:
          severity: "pageteam"
          routing_key: "myteam"
      ticket_alert:
        labels:
          severity: "slack"
          slack_channel: "#alerts-myteam"
`

	gen, err := lib.NewPrometheusSLOGenerator(lib.PrometheusSLOGeneratorConfig{
		ExtraLabels: map[string]string{"source": "slothlib-example"},
	})
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		ctx := context.Background()

		slo, err := gen.GenerateFromRaw(ctx, []byte(sloSpec))
		if err != nil {
			b.Fatal(err)
		}

		err = gen.WriteResultAsPrometheusStd(ctx, *slo, io.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}
}
