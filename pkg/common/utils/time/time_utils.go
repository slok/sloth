package time

import (
	"time"
)

// CalculateStepsForTimeRange calculates the step duration for a given time range and number of steps.
func CalculateStepsForTimeRange(from, to time.Time, steps int) time.Duration {
	totalDuration := to.Sub(from)
	step := totalDuration / time.Duration(steps)

	// Round step to minutes.
	if step < time.Minute {
		step = time.Minute
	}
	step = (time.Duration(int(step.Minutes())) * time.Minute)

	return step
}

// RoundTimeToDay rounds a time to the start of the day (00:00:00).
func RoundTimeToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// WeekMonday returns the Monday of the week for the given time.
func WeekMonday(t time.Time) time.Time {
	diff := time.Duration(t.Weekday() - 1)
	if diff < 0 {
		diff = 6
	}

	return RoundTimeToDay(t).Add(-1 * diff * 24 * time.Hour) // Remove the diff days until monday.
}

// MonthFirst returns the first TS of the month for the given time.
func MonthFirst(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// QuarterFirst returns the first TS of the quarter for the given time.
func QuarterFirst(t time.Time) time.Time {
	// Gets the first day of the quarter the time is in.
	month := ((t.Month()-1)/3)*3 + 1
	return time.Date(t.Year(), month, 1, 0, 0, 0, 0, t.Location())
}

// YearFirst returns the first TS of the year for the given time.
func YearFirst(t time.Time) time.Time {
	// Gets the first day of the year the time is in.
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

// EndOfWeek returns the last TS of the week for the given time.
func EndOfWeek(t time.Time) time.Time {
	return WeekMonday(t).Add(7*24*time.Hour - 1)
}

// EndOfMonth returns the last TS of the month for the given time.
func EndOfMonth(t time.Time) time.Time {
	return SetEndOfDay(LastDayOfMonth(t))
}

// EndOfQuarter returns the last TS of the quarter for the given time.
func EndOfQuarter(t time.Time) time.Time {
	qf := QuarterFirst(t)
	qe := NextMonths(qf, 2) // We are at first month of quarter, add 2 to get to last month.
	return SetEndOfDay(LastDayOfMonth(qe))
}

func EndOfYear(t time.Time) time.Time {
	nextYear := time.Date(t.Year()+1, 1, 1, 0, 0, 0, 0, t.Location())
	return SetEndOfDay(nextYear.Add(-24 * time.Hour))
}

// LastDayOfMonth returns the last day of the month for the given time.
func LastDayOfMonth(t time.Time) time.Time {
	year := t.Year()
	firstOfNextMonth := time.Date(year, t.Month()+1, 1, 0, 0, 0, 0, t.Location())
	lastOfMonth := firstOfNextMonth.Add(-24 * time.Hour)
	return lastOfMonth
}

// NextMonths returns the time after adding the specified number of months to the given time.
func NextMonths(t time.Time, months int) time.Time {
	year := t.Year()
	month := t.Month() + time.Month(months)
	if month > 12 {
		month = month % 12
		year++
	}

	return time.Date(year, month, t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// SetEndOfDay sets the time to the end of the day (23:59:59.999999999) in a given TS.
func SetEndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}
