package prometheus

import (
	"github.com/slok/sloth/pkg/common/model"
)

// TODO(slok): Remove after migration to pkg/common/model package.
type SLO = model.PromSLO
type SLI = model.PromSLI
type SLIEvents = model.PromSLIEvents
type AlertMeta = model.PromAlertMeta
type SLOGroup = model.PromSLOGroup
type SLORules = model.PromSLORules
type SLIRaw = model.PromSLIRaw
