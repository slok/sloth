package model

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Service struct {
	ID string
}

type ServiceStats struct {
	ServiceID                         string
	TotalSLOs                         int
	SLOsCurrentlyBurningOverBudget    int
	SLOsAlreadyConsumedBudgetOnPeriod int
}

type SLO struct {
	ID             string // ID is unique for an SLO and grouped SLOs.
	SlothID        string // SlothID is the ID set by Sloth on Prometheus (if grouped by labels SLO they will share this ID).
	Name           string
	ServiceID      string
	Objective      float64
	GroupLabels    map[string]string // Some SLOs are grouped by labels under the umbrella of the same SLO spec.
	IsGrouped      bool
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

func SLOGroupLabelsIDMarshal(sloID string, labels map[string]string) string {
	lps := []string{}
	for k, v := range labels {
		lps = append(lps, fmt.Sprintf("%s=%s", k, v))
	}
	sort.SliceStable(lps, func(i, j int) bool { return lps[i] < lps[j] })
	lpb64 := base64.URLEncoding.EncodeToString([]byte(strings.Join(lps, ",")))

	return fmt.Sprintf("%s:%s", sloID, lpb64)
}

func SLOGroupLabelsIDUnmarshal(id string) (sloID string, labels map[string]string, err error) {
	id, b64Labels, ok := strings.Cut(id, ":")
	if !ok {
		return id, nil, nil
	}

	decodedLabels, err := base64.URLEncoding.DecodeString(b64Labels)
	if err != nil {
		return "", nil, fmt.Errorf("could not decode base64 labels: %w", err)
	}

	labels = map[string]string{}
	for _, lp := range strings.Split(string(decodedLabels), ",") {
		k, v, ok := strings.Cut(lp, "=")
		if !ok || k == "" || v == "" {
			return "", nil, fmt.Errorf("invalid label pair in decoded labels: %q", lp)
		}

		labels[k] = v
	}

	return id, labels, nil
}
