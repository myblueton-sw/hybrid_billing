package closing

import "time"

func wall(at time.Time, location *time.Location) time.Time {
	local := at.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), local.Second(), 0, time.UTC)
}

func localInstant(day time.Time, minutes int, location *time.Location) time.Time {
	target := day.Add(time.Duration(minutes) * time.Minute)
	high := 0
	offsets := map[int]bool{}
	for hours := -48; hours <= 48; hours++ {
		_, offset := target.Add(time.Duration(hours) * time.Hour).In(location).Zone()
		if len(offsets) == 0 || offset > high {
			high = offset
		}
		offsets[offset] = true
	}
	var best time.Time
	for offset := range offsets {
		candidate := target.Add(-time.Duration(offset) * time.Second)
		if wall(candidate, location).Equal(target) && (best.IsZero() || candidate.Before(best)) {
			best = candidate
		}
	}
	if !best.IsZero() {
		return best
	}
	for at := target.Add(-time.Duration(high) * time.Second); ; at = at.Add(time.Minute) {
		if wall(at, location).After(target) {
			return at
		}
	}
}

func parseDate(text string) (time.Time, bool) {
	parsed, err := time.Parse(time.DateOnly, text)
	return parsed, err == nil && parsed.Format(time.DateOnly) == text
}

type calendar struct {
	view     CalendarView
	location *time.Location
	weekend  map[time.Weekday]bool
	holidays map[string]bool
	from     time.Time
	to       time.Time
}

func (c *calendar) business(day time.Time) bool {
	return !c.weekend[day.Weekday()] && !c.holidays[day.Format(time.DateOnly)]
}

func (c *calendar) deadline(period string, rule Rule) (time.Time, error) {
	start, _ := time.Parse("2006-01", period)
	day := start.AddDate(0, 1, -1)
	if day.Before(c.from) || day.After(c.to) {
		return time.Time{}, refuse(RefusalOutsideCalendar)
	}
	count := 0
	for step := 0; step < maxSearch; step++ {
		day = day.AddDate(0, 0, 1)
		if day.After(c.to) {
			return time.Time{}, refuse(RefusalOutsideCalendar)
		}
		if !rule.Business || c.business(day) {
			count++
		}
		if count == rule.Days {
			return localInstant(day, c.view.Calendar.Close, c.location), nil
		}
	}
	return time.Time{}, refuse(RefusalOutsideCalendar)
}
