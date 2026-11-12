package contracts

import "sort"

func ordered(events []Event) []Event {
	sorted := append([]Event(nil), events...)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].From.Before(sorted[b].From) })
	return sorted
}

func step(position Position, event Event) (Position, bool) {
	running := position.State == Active || position.State == Suspended
	switch event.Kind {
	case Start:
		return Position{State: Active, Revision: event.Revision}, position.State == NotStarted
	case Change:
		return Position{State: position.State, Revision: event.Revision}, running && event.Revision != position.Revision
	case Renew:
		return position, running
	case Suspend:
		return Position{State: Suspended, Revision: position.Revision}, position.State == Active
	case Resume:
		return Position{State: Active, Revision: position.Revision}, position.State == Suspended
	case End:
		return Position{State: Ended}, running
	}
	return position, false
}

func validLifecycle(sorted []Event) bool {
	position := Position{State: NotStarted}
	for index, event := range sorted {
		if !event.valid() || (index > 0 && !sorted[index-1].From.Before(event.From)) {
			return false
		}
		next, allowed := step(position, event)
		if !allowed {
			return false
		}
		position = next
	}
	return true
}

func ended(sorted []Event) (Day, bool) {
	if len(sorted) == 0 || sorted[len(sorted)-1].Kind != End {
		return Day{}, false
	}
	return sorted[len(sorted)-1].From, true
}

func positionOn(sorted []Event, day Day) Position {
	position := Position{State: NotStarted}
	for _, event := range sorted {
		if day.Before(event.From) {
			break
		}
		position, _ = step(position, event)
	}
	return position
}

func Segments(events []Event, period Interval) ([]Segment, error) {
	sorted := ordered(events)
	if !period.Valid() || !validLifecycle(sorted) {
		return nil, ErrInvalid
	}
	position := Position{State: NotStarted}
	cursor := period.start
	segments := []Segment{}
	for _, event := range sorted {
		if !event.From.Before(period.end) {
			break
		}
		next, _ := step(position, event)
		if period.start.Before(event.From) && next != position {
			segments = append(segments, Segment{Interval: Interval{start: cursor, end: event.From}, State: position.State, Revision: position.Revision})
			cursor = event.From
		}
		position = next
	}
	return append(segments, Segment{Interval: Interval{start: cursor, end: period.end}, State: position.State, Revision: position.Revision}), nil
}
