package v1

import (
	"encoding/json"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestAlertingUnmarshalJSONExclusiveKeys(t *testing.T) {
	tests := map[string]struct {
		raw          string
		wantErr      bool
		wantContains []string
		wantPageSev  string
	}{
		"pageAlert only": {
			raw:         `{"name":"a","pageAlert":{"labels":{"severity":"critical"}}}`,
			wantPageSev: "critical",
		},
		"fastBurn only": {
			raw:         `{"name":"a","fastBurn":{"labels":{"severity":"critical"}}}`,
			wantPageSev: "critical",
		},
		"pageAlert and fastBurn": {
			raw:          `{"name":"a","pageAlert":{"labels":{"severity":"page"}},"fastBurn":{"labels":{"severity":"critical"}}}`,
			wantErr:      true,
			wantContains: []string{"pageAlert", "fastBurn"},
		},
		"ticketAlert and slowBurn": {
			raw:          `{"name":"a","ticketAlert":{"disable":true},"slowBurn":{"labels":{"severity":"warning"}}}`,
			wantErr:      true,
			wantContains: []string{"ticketAlert", "slowBurn"},
		},
		"empty fastBurn with pageAlert is ignored": {
			raw:         `{"name":"a","pageAlert":{"labels":{"severity":"critical"}},"fastBurn":{}}`,
			wantPageSev: "critical",
		},
		"empty pageAlert with fastBurn is ignored": {
			raw:         `{"name":"a","pageAlert":{},"fastBurn":{"labels":{"severity":"critical"}}}`,
			wantPageSev: "critical",
		},
		"empty fastBurn and slowBurn with empty pageAlert and ticketAlert": {
			raw: `{"name":"a","pageAlert":{},"ticketAlert":{},"fastBurn":{},"slowBurn":{}}`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var got Alerting
			err := json.Unmarshal([]byte(test.raw), &got)
			if test.wantErr {
				if err == nil {
					t.Fatalf("UnmarshalJSON(%s) error = nil, want error", test.raw)
				}
				for _, s := range test.wantContains {
					if !strings.Contains(err.Error(), s) {
						t.Errorf("UnmarshalJSON(%s) error = %q, want substring %q", test.raw, err.Error(), s)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("UnmarshalJSON(%s) error = %v, want nil", test.raw, err)
			}
			page, err := got.ResolvedPageAlert()
			if err != nil {
				t.Fatalf("ResolvedPageAlert() error = %v, want nil", err)
			}
			if page.Labels["severity"] != test.wantPageSev {
				t.Errorf("ResolvedPageAlert().Labels[severity] = %q, want %q", page.Labels["severity"], test.wantPageSev)
			}
		})
	}
}

func TestAlertingUnstructuredRoundTripKeepsPageAlert(t *testing.T) {
	in := Alerting{
		Name: "myServiceAlert",
		PageAlert: Alert{
			Labels: map[string]string{"severity": "critical"},
		},
		TicketAlert: Alert{Disable: true},
	}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&in)
	if err != nil {
		t.Fatalf("ToUnstructured() error = %v, want nil", err)
	}
	raw, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}

	var got Alerting
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("UnmarshalJSON(%s) error = %v, want nil", raw, err)
	}
	page, err := got.ResolvedPageAlert()
	if err != nil {
		t.Fatalf("ResolvedPageAlert() error = %v, want nil", err)
	}
	if page.Labels["severity"] != "critical" {
		t.Errorf("ResolvedPageAlert().Labels[severity] = %q, want %q", page.Labels["severity"], "critical")
	}
	ticket, err := got.ResolvedTicketAlert()
	if err != nil {
		t.Fatalf("ResolvedTicketAlert() error = %v, want nil", err)
	}
	if !ticket.Disable {
		t.Errorf("ResolvedTicketAlert().Disable = %v, want true", ticket.Disable)
	}
}

func TestAlertingUnstructuredRoundTripKeepsFastBurn(t *testing.T) {
	in := Alerting{
		Name: "myServiceAlert",
		FastBurn: Alert{
			Labels: map[string]string{"severity": "critical"},
		},
		SlowBurn: Alert{Disable: true},
	}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&in)
	if err != nil {
		t.Fatalf("ToUnstructured() error = %v, want nil", err)
	}
	raw, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}

	var got Alerting
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("UnmarshalJSON(%s) error = %v, want nil", raw, err)
	}
	page, err := got.ResolvedPageAlert()
	if err != nil {
		t.Fatalf("ResolvedPageAlert() error = %v, want nil", err)
	}
	if page.Labels["severity"] != "critical" {
		t.Errorf("ResolvedPageAlert().Labels[severity] = %q, want %q", page.Labels["severity"], "critical")
	}
	ticket, err := got.ResolvedTicketAlert()
	if err != nil {
		t.Fatalf("ResolvedTicketAlert() error = %v, want nil", err)
	}
	if !ticket.Disable {
		t.Errorf("ResolvedTicketAlert().Disable = %v, want true", ticket.Disable)
	}
}
