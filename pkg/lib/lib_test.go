package lib_test

import (
	"context"
	"os"

	slotk8sv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	"github.com/slok/sloth/pkg/lib"
	slothprometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ExamplePrometheusSLOGenerator_GenerateFromRaw() {
	sloSpec := []byte(`
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
`)

	ctx := context.Background()

	gen, err := lib.NewPrometheusSLOGenerator(lib.PrometheusSLOGeneratorConfig{
		ExtraLabels: map[string]string{"source": "slothlib-example"},
	})
	if err != nil {
		panic(err)
	}

	// Generate SLO and write result.
	slo, err := gen.GenerateFromRaw(ctx, sloSpec)
	if err != nil {
		panic(err)
	}

	err = lib.WriteResultAsPrometheusStd(ctx, *slo, os.Stdout)
	if err != nil {
		panic(err)
	}
}

func ExamplePrometheusSLOGenerator_GenerateFromSlothV1() {
	sloSpec := slothprometheusv1.Spec{
		Service: "myservice",
		Labels: map[string]string{
			"owner": "myteam",
			"repo":  "myorg/myservice",
			"tier":  "2",
		},
		SLOs: []slothprometheusv1.SLO{
			{
				Name:        "requests-availability",
				Objective:   99.9,
				Description: "Common SLO based on availability for HTTP request responses.",
				SLI: slothprometheusv1.SLI{
					Events: &slothprometheusv1.SLIEvents{
						ErrorQuery: `sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))`,
						TotalQuery: `sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))`,
					},
				},
				Alerting: slothprometheusv1.Alerting{
					Name:        "MyServiceHighErrorRate",
					Labels:      map[string]string{"category": "availability"},
					Annotations: map[string]string{"summary": "High error rate on 'myservice' requests responses"},
					PageAlert: slothprometheusv1.Alert{
						Labels: map[string]string{
							"severity":    "page",
							"routing_key": "myteam",
						},
					},
					TicketAlert: slothprometheusv1.Alert{
						Labels: map[string]string{
							"severity":      "slack",
							"slack_channel": "#alerts-myteam",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()

	gen, err := lib.NewPrometheusSLOGenerator(lib.PrometheusSLOGeneratorConfig{
		ExtraLabels: map[string]string{"source": "slothlib-example"},
	})
	if err != nil {
		panic(err)
	}

	// Generate SLO and write result.
	slo, err := gen.GenerateFromSlothV1(ctx, sloSpec)
	if err != nil {
		panic(err)
	}

	err = lib.WriteResultAsPrometheusStd(ctx, *slo, os.Stdout)
	if err != nil {
		panic(err)
	}
}

func ExamplePrometheusSLOGenerator_GenerateFromK8sV1() {
	sloSpec := slotk8sv1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test01",
			Labels: map[string]string{
				"prometheus": "default",
			},
		},
		Spec: slotk8sv1.PrometheusServiceLevelSpec{
			Service: "svc01",
			Labels: map[string]string{
				"globalk1": "globalv1",
			},
			SLOs: []slotk8sv1.SLO{
				{
					Name:      "slo01",
					Objective: 99.9,
					Labels: map[string]string{
						"slo01k1": "slo01v1",
					},
					SLI: slotk8sv1.SLI{Events: &slotk8sv1.SLIEvents{
						ErrorQuery: `sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))`,
						TotalQuery: `sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))`,
					}},
					Alerting: slotk8sv1.Alerting{
						Name: "myServiceAlert",
						Labels: map[string]string{
							"alert01k1": "alert01v1",
						},
						Annotations: map[string]string{
							"alert02k1": "alert02v1",
						},
						PageAlert:   slotk8sv1.Alert{},
						TicketAlert: slotk8sv1.Alert{},
					},
				},
			},
		},
	}

	ctx := context.Background()

	gen, err := lib.NewPrometheusSLOGenerator(lib.PrometheusSLOGeneratorConfig{
		ExtraLabels: map[string]string{"source": "slothlib-example"},
	})
	if err != nil {
		panic(err)
	}

	// Generate SLO and write result.
	slo, err := gen.GenerateFromK8sV1(ctx, sloSpec)
	if err != nil {
		panic(err)
	}

	kmeta := lib.K8sMeta{
		Name:      "sloth-slo-gen-" + sloSpec.ObjectMeta.Name,
		Namespace: sloSpec.ObjectMeta.Namespace,
	}
	err = lib.WriteResultAsK8sPrometheusOperator(ctx, kmeta, *slo, os.Stdout)
	if err != nil {
		panic(err)
	}
}
