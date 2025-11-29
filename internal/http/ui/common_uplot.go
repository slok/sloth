package ui

// Check: https://github.com/leeoniya/uPlot/tree/master/docs.
type uPlotSLIChart struct {
	Title        string     `json:"title"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	TSs          []int      `json:"timestamps"`
	SLIs         []*float64 `json:"sli_values"`
	SLOObjective float64    `json:"slo_objective"`
}

func (u *uPlotSLIChart) defaults() error {
	if u.Title == "" {
		u.Title = "SLI over time"
	}

	if u.Height == 0 {
		u.Height = 400
	}
	return nil
}

// Check: https://github.com/leeoniya/uPlot/tree/master/docs.
type uPlotBudgetBurnChart struct {
	Title         string     `json:"title"`
	ColorLineOk   bool       `json:"color_line_ok"`
	Width         int        `json:"width"`
	Height        int        `json:"height"`
	TSs           []int      `json:"timestamps"`
	RealBurned    []*float64 `json:"real_burned_values"`
	PerfectBurned []*float64 `json:"perfect_burned_values"`
}

func (u *uPlotBudgetBurnChart) defaults() error {
	if u.Title == "" {
		u.Title = "Budget Burn"
	}

	if u.Height == 0 {
		u.Height = 400
	}
	return nil
}

func float64Ptr(f float64) *float64 {
	return &f
}
