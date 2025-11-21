package model

import "time"

type Service struct {
	ID string
}

type SLO struct {
	ID             string
	Name           string
	ServiceID      string
	Objective      float64
	PeriodDuration time.Duration
}

type SLOBudgetDetails struct {
	SLOID                     string
	BurningBudgetPercent      float64 // Percentage of error budget burning.
	BurnedBudgetWindowPercent float64 // Percentage of error budget burned in the period window.
}

type SLOAlerts struct {
	SLOID         string
	FiringPage    *Alert
	FiringWarning *Alert
}

type Alert struct {
	Name string
}

type DataPoint struct {
	Value   float64
	Missing bool // Easier than using float64 nil pointers.
	TS      time.Time
}
