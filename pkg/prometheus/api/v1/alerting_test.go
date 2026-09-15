package v1

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAlertingUnmarshalJSONExclusiveKeys(t *testing.T) {
	tests := map[string]struct {
		raw           string
		wantErr       bool
		wantContains  []string
		wantPageSev   string
		wantTicketOff bool
	}{
		"page_alert only": {
			raw:         `{"name":"a","page_alert":{"labels":{"severity":"critical"}}}`,
			wantPageSev: "critical",
		},
		"fast_burn only": {
			raw:         `{"name":"a","fast_burn":{"labels":{"severity":"critical"}}}`,
			wantPageSev: "critical",
		},
		"page_alert and fast_burn": {
			raw:          `{"name":"a","page_alert":{"labels":{"severity":"page"}},"fast_burn":{"labels":{"severity":"critical"}}}`,
			wantErr:      true,
			wantContains: []string{"page_alert", "fast_burn"},
		},
		"ticket_alert and slow_burn": {
			raw:          `{"name":"a","ticket_alert":{"disable":true},"slow_burn":{"labels":{"severity":"warning"}}}`,
			wantErr:      true,
			wantContains: []string{"ticket_alert", "slow_burn"},
		},
		"fast_burn plus ticket_alert": {
			raw:           `{"name":"a","fast_burn":{"labels":{"severity":"critical"}},"ticket_alert":{"disable":true}}`,
			wantPageSev:   "critical",
			wantTicketOff: true,
		},
		"empty fast_burn with page_alert is ignored": {
			raw:         `{"name":"a","page_alert":{"labels":{"severity":"critical"}},"fast_burn":{}}`,
			wantPageSev: "critical",
		},
		"empty page_alert with fast_burn is ignored": {
			raw:         `{"name":"a","page_alert":{},"fast_burn":{"labels":{"severity":"critical"}}}`,
			wantPageSev: "critical",
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
			ticket, err := got.ResolvedTicketAlert()
			if err != nil {
				t.Fatalf("ResolvedTicketAlert() error = %v, want nil", err)
			}
			if ticket.Disable != test.wantTicketOff {
				t.Errorf("ResolvedTicketAlert().Disable = %v, want %v", ticket.Disable, test.wantTicketOff)
			}
		})
	}
}
