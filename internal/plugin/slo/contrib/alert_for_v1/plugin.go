package plugin

import (
	"context"
	"encoding/json"

	prommodel "github.com/prometheus/common/model"

	"github.com/slok/sloth/pkg/common/conventions"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/contrib/alert_for/v1"
)

func NewPlugin(_ json.RawMessage, _ pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	return plugin{}, nil
}

type plugin struct{}

func (p plugin) ProcessSLO(_ context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	src := request.OriginalSource.SlothV1
	if src == nil {
		return nil
	}

	var pageFor prommodel.Duration
	var ticketFor prommodel.Duration
	found := false
	for _, specSLO := range src.SLOs {
		if specSLO.Name != request.SLO.Name {
			continue
		}

		pageFor = specSLO.Alerting.PageAlert.For
		ticketFor = specSLO.Alerting.TicketAlert.For
		found = true
		break
	}

	if !found || (pageFor == 0 && ticketFor == 0) {
		return nil
	}

	pageSeverity := request.MWMBAlertGroup.PageQuick.Severity.String()
	ticketSeverity := request.MWMBAlertGroup.TicketQuick.Severity.String()

	for i := range result.SLORules.AlertRules.Rules {
		rule := &result.SLORules.AlertRules.Rules[i]
		if rule.Labels == nil {
			continue
		}

		switch rule.Labels[conventions.PromSLOSeverityLabelName] {
		case pageSeverity:
			if pageFor != 0 {
				rule.For = pageFor
			}
		case ticketSeverity:
			if ticketFor != 0 {
				rule.For = ticketFor
			}
		}
	}

	return nil
}
