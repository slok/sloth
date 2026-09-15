package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// UnmarshalJSON implements json.Unmarshaler.
//
// It rejects documents that set both a canonical MWMB class key and its alias
// (pageAlert and fastBurn, or ticketAlert and slowBurn) when both objects are
// non-empty. Empty objects on either key are ignored because the Kubernetes
// unstructured converter always emits zero-value structs (omitempty does not
// apply to struct kinds), which would otherwise look like pageAlert and
// fastBurn both being set.
func (a *Alerting) UnmarshalJSON(data []byte) error {
	if err := rejectExclusiveKeys(data, "pageAlert", "fastBurn"); err != nil {
		return err
	}
	if err := rejectExclusiveKeys(data, "ticketAlert", "slowBurn"); err != nil {
		return err
	}
	type alerting Alerting
	return json.Unmarshal(data, (*alerting)(a))
}

// ResolvedPageAlert returns the page-class MWMB alert configuration.
//
// FastBurn is an alias of PageAlert. Setting both is an error.
func (a Alerting) ResolvedPageAlert() (Alert, error) {
	return exclusiveAlert(a.PageAlert, a.FastBurn, "pageAlert", "fastBurn")
}

// ResolvedTicketAlert returns the ticket-class MWMB alert configuration.
//
// SlowBurn is an alias of TicketAlert. Setting both is an error.
func (a Alerting) ResolvedTicketAlert() (Alert, error) {
	return exclusiveAlert(a.TicketAlert, a.SlowBurn, "ticketAlert", "slowBurn")
}

func rejectExclusiveKeys(data []byte, canonical, alias string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	canonicalRaw, hasCanonical := raw[canonical]
	aliasRaw, hasAlias := raw[alias]
	if hasCanonical && hasAlias && !isEmptyAlertJSON(canonicalRaw) && !isEmptyAlertJSON(aliasRaw) {
		return fmt.Errorf("%s and %s are mutually exclusive", canonical, alias)
	}
	return nil
}

func isEmptyAlertJSON(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return true
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}
	return len(obj) == 0
}

func exclusiveAlert(canonical, alias Alert, canonicalName, aliasName string) (Alert, error) {
	if alertPresent(canonical) && alertPresent(alias) {
		return Alert{}, fmt.Errorf("%s and %s are mutually exclusive", canonicalName, aliasName)
	}
	if alertPresent(alias) {
		return alias, nil
	}
	return canonical, nil
}

func alertPresent(a Alert) bool {
	return a.Disable || len(a.Labels) > 0 || len(a.Annotations) > 0
}
