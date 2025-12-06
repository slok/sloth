package time_test

import (
	"testing"
	"time"

	utilstime "github.com/slok/sloth/pkg/common/utils/time"
	"github.com/stretchr/testify/assert"
)

func TestCalculateStepsForTimeRange(t *testing.T) {
	tests := map[string]struct {
		from   time.Time
		to     time.Time
		steps  int
		expDur time.Duration
	}{
		"Calculates steps for time range correctly.": {
			from:   time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
			to:     time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			steps:  30,
			expDur: 24 * time.Hour,
		},

		"Calculates steps for time range correctly (between months).": {
			from:   time.Date(2025, 10, 15, 0, 0, 0, 0, time.UTC),
			to:     time.Date(2025, 11, 15, 0, 0, 0, 0, time.UTC),
			steps:  31,
			expDur: 24 * time.Hour,
		},

		"Calculates steps for time range correctly (between months february).": {
			from:   time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC),
			to:     time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
			steps:  28,
			expDur: 24 * time.Hour,
		},

		"Calculates steps for time range correctly (between years).": {
			from:   time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
			to:     time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			steps:  31,
			expDur: 24 * time.Hour,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotDur := utilstime.CalculateStepsForTimeRange(test.from, test.to, test.steps)
			assert.Equal(test.expDur, gotDur)
		})
	}
}

func TestRoundTimeToDay(t *testing.T) {
	tests := map[string]struct {
		ts    time.Time
		expTS time.Time
	}{
		"Rounds time to day correctly.": {
			ts:    time.Date(2025, 11, 23, 15, 45, 30, 0, time.UTC),
			expTS: time.Date(2025, 11, 23, 0, 0, 0, 0, time.UTC),
		},
		"Rounds time to day correctly at midnight.": {
			ts:    time.Date(2025, 11, 23, 0, 0, 0, 0, time.UTC),
			expTS: time.Date(2025, 11, 23, 0, 0, 0, 0, time.UTC),
		},
		"Rounds time to day correctly before midnight.": {
			ts:    time.Date(2025, 11, 23, 23, 59, 59, 0, time.UTC),
			expTS: time.Date(2025, 11, 23, 0, 0, 0, 0, time.UTC),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotDur := utilstime.RoundTimeToDay(test.ts)
			assert.Equal(test.expTS, gotDur)
		})
	}
}

func TestWeekMonday(t *testing.T) {
	tests := map[string]struct {
		ts    time.Time
		expTS time.Time
	}{
		"Gets week monday correctly for wednesday.": {
			ts:    time.Date(2025, 11, 19, 15, 45, 30, 0, time.UTC), // Wednesday.
			expTS: time.Date(2025, 11, 17, 0, 0, 0, 0, time.UTC),    // Monday.
		},
		"Gets week monday correctly for monday.": {
			ts:    time.Date(2025, 11, 17, 10, 0, 0, 0, time.UTC), // Monday.
			expTS: time.Date(2025, 11, 17, 0, 0, 0, 0, time.UTC),  // Monday.
		},
		"Gets week monday correctly for sunday.": {
			ts:    time.Date(2025, 11, 23, 23, 59, 59, 0, time.UTC), // Sunday.
			expTS: time.Date(2025, 11, 17, 0, 0, 0, 0, time.UTC),    // Monday.
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotDur := utilstime.WeekMonday(test.ts)
			assert.Equal(test.expTS, gotDur)
		})
	}
}

func TestMonthFirst(t *testing.T) {
	tests := map[string]struct {
		ts    time.Time
		expTS time.Time
	}{
		"Gets month first correctly for a random day.": {
			ts:    time.Date(2025, 11, 19, 15, 45, 30, 0, time.UTC), // Wednesday.
			expTS: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),     // First of the month.
		},
		"Gets month first correctly for the first day.": {
			ts:    time.Date(2025, 11, 1, 10, 0, 0, 0, time.UTC), // First of the month.
			expTS: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),  // First of the month.
		},
		"Gets month first correctly for the last day.": {
			ts:    time.Date(2025, 11, 30, 23, 59, 59, 0, time.UTC), // Last of the month.
			expTS: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),     // First of the month.
		},
		"Gets month first correctly for February.": {
			ts:    time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC), // Middle of February (leap year).
			expTS: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),   // First of February.
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotDur := utilstime.MonthFirst(test.ts)
			assert.Equal(test.expTS, gotDur)
		})
	}
}

func TestQuarterFirst(t *testing.T) {
	tests := map[string]struct {
		ts    time.Time
		expTS time.Time
	}{
		"Gets quarter first correctly for a random day in Q1.": {
			ts:    time.Date(2025, 2, 19, 15, 45, 30, 0, time.UTC), // February.
			expTS: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),     // First of Q1.
		},
		"Gets quarter first correctly for a random day in Q2.": {
			ts:    time.Date(2025, 5, 10, 10, 0, 0, 0, time.UTC), // May.
			expTS: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),   // First of Q2.
		},
		"Gets quarter first correctly for a random day in Q3.": {
			ts:    time.Date(2025, 8, 25, 23, 59, 59, 0, time.UTC), // August.
			expTS: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),     // First of Q3.
		},
		"Gets quarter first correctly for a random day in Q4.": {
			ts:    time.Date(2025, 11, 5, 12, 0, 0, 0, time.UTC), // November.
			expTS: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),  // First of Q4.
		},
		"Gets quarter first correctly for the first day of Q1.": {
			ts:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), // First of Q1.
			expTS: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), // First of Q1.
		},
		"Gets quarter first correctly for the last day of Q4.": {
			ts:    time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC), // Last of Q4.
			expTS: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),     // First of Q4.
		},
		"Gets quarter first correctly for leap year Q1.": {
			ts:    time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC), // February (leap year).
			expTS: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),   // First of Q1.
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotDur := utilstime.QuarterFirst(test.ts)
			assert.Equal(test.expTS, gotDur)
		})
	}
}

func TestYearFirst(t *testing.T) {
	tests := map[string]struct {
		ts    time.Time
		expTS time.Time
	}{
		"Gets year first correctly for a random day.": {
			ts:    time.Date(2025, 11, 19, 15, 45, 30, 0, time.UTC), // November.
			expTS: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),      // First of the year.
		},
		"Gets year first correctly for the first day.": {
			ts:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), // First of the year.
			expTS: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),  // First of the year.
		},
		"Gets year first correctly for the last day.": {
			ts:    time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC), // Last of the year.
			expTS: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),      // First of the year.
		},
		"Gets year first correctly for leap year.": {
			ts:    time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC), // Middle of February (leap year).
			expTS: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),   // First of the year.
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotDur := utilstime.YearFirst(test.ts)
			assert.Equal(test.expTS, gotDur)
		})
	}
}

func TestEndOfWeek(t *testing.T) {
	tests := map[string]struct {
		ts    time.Time
		expTS time.Time
	}{
		"Gets end of week correctly for wednesday.": {
			ts:    time.Date(2025, 11, 19, 15, 45, 30, 0, time.UTC),         // Wednesday.
			expTS: time.Date(2025, 11, 23, 23, 59, 59, 999999999, time.UTC), // Sunday.
		},
		"Gets end of week correctly for monday.": {
			ts:    time.Date(2025, 11, 17, 10, 0, 0, 0, time.UTC),           // Monday.
			expTS: time.Date(2025, 11, 23, 23, 59, 59, 999999999, time.UTC), // Sunday.
		},
		"Gets end of week correctly for sunday.": {
			ts:    time.Date(2025, 11, 23, 23, 59, 59, 0, time.UTC),         // Sunday.
			expTS: time.Date(2025, 11, 23, 23, 59, 59, 999999999, time.UTC), // Sunday.
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotDur := utilstime.EndOfWeek(test.ts)
			assert.Equal(test.expTS, gotDur)
		})
	}
}

func TestEndOfMonth(t *testing.T) {
	tests := map[string]struct {
		ts    time.Time
		expTS time.Time
	}{
		"Gets end of month correctly for a random day.": {
			ts:    time.Date(2025, 11, 19, 15, 45, 30, 0, time.UTC),         // November.
			expTS: time.Date(2025, 11, 30, 23, 59, 59, 999999999, time.UTC), // November 30th.
		},

		"Gets end of month correctly for a december.": {
			ts:    time.Date(2025, 12, 19, 15, 45, 30, 0, time.UTC),         // December.
			expTS: time.Date(2025, 12, 31, 23, 59, 59, 999999999, time.UTC), // December 31st.
		},
		"Gets end of month correctly for February non-leap year.": {
			ts:    time.Date(2025, 2, 15, 12, 0, 0, 0, time.UTC),           // February.
			expTS: time.Date(2025, 2, 28, 23, 59, 59, 999999999, time.UTC), // February 28th.
		},
		"Gets end of month correctly for February leap year.": {
			ts:    time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC),           // February.
			expTS: time.Date(2024, 2, 29, 23, 59, 59, 999999999, time.UTC), // February 29th.
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotDur := utilstime.EndOfMonth(test.ts)
			assert.Equal(test.expTS, gotDur)
		})
	}
}

func TestEndOfQuarter(t *testing.T) {
	tests := map[string]struct {
		ts    time.Time
		expTS time.Time
	}{
		"Gets end of quarter correctly for a random day in Q1.": {
			ts:    time.Date(2025, 2, 19, 15, 45, 30, 0, time.UTC),         // February.
			expTS: time.Date(2025, 3, 31, 23, 59, 59, 999999999, time.UTC), // End of Q1.
		},
		"Gets end of quarter correctly for a random day in Q2.": {
			ts:    time.Date(2025, 5, 10, 10, 0, 0, 0, time.UTC),           // May.
			expTS: time.Date(2025, 6, 30, 23, 59, 59, 999999999, time.UTC), // End of Q2.
		},
		"Gets end of quarter correctly for a random day in Q3.": {
			ts:    time.Date(2025, 8, 25, 23, 59, 59, 0, time.UTC),         // August.
			expTS: time.Date(2025, 9, 30, 23, 59, 59, 999999999, time.UTC), // End of Q3.
		},
		"Gets end of quarter correctly for a random day in Q4.": {
			ts:    time.Date(2025, 11, 5, 12, 0, 0, 0, time.UTC),            // November.
			expTS: time.Date(2025, 12, 31, 23, 59, 59, 999999999, time.UTC), // End of Q4.
		},
		"Gets end of quarter correctly for the last day of Q1.": {
			ts:    time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC),            // End of Q1.
			expTS: time.Date(2025, 3, 31, 23, 59, 59, 999999999, time.UTC), // End of Q1.
		},
		"Gets end of quarter correctly for the last day of Q4.": {
			ts:    time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),         // End of Q4.
			expTS: time.Date(2025, 12, 31, 23, 59, 59, 999999999, time.UTC), // End of Q4.
		},
		"Gets end of quarter correctly for leap year Q1.": {
			ts:    time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),           // February (leap year).
			expTS: time.Date(2024, 3, 31, 23, 59, 59, 999999999, time.UTC), // End of Q1.
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotDur := utilstime.EndOfQuarter(test.ts)
			assert.Equal(test.expTS, gotDur)
		})
	}
}

func TestEndOfYear(t *testing.T) {
	tests := map[string]struct {
		ts    time.Time
		expTS time.Time
	}{
		"Gets end of year correctly for a random day.": {
			ts:    time.Date(2025, 11, 19, 15, 45, 30, 0, time.UTC),         // November.
			expTS: time.Date(2025, 12, 31, 23, 59, 59, 999999999, time.UTC), // December 31st.
		},
		"Gets end of year correctly for leap year.": {
			ts:    time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC),            // Middle of February (leap year).
			expTS: time.Date(2024, 12, 31, 23, 59, 59, 999999999, time.UTC), // December 31st.
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotDur := utilstime.EndOfYear(test.ts)
			assert.Equal(test.expTS, gotDur)
		})
	}
}
