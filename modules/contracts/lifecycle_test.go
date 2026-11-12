package contracts

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func day(t testing.TB, text string) Day {
	t.Helper()
	var year, month, dayOfMonth int
	if _, err := fmt.Sscanf(text, "%d-%d-%d", &year, &month, &dayOfMonth); err != nil {
		t.Fatal(err)
	}
	made, err := NewDay(year, month, dayOfMonth)
	if err != nil {
		t.Fatalf("%s: %v", text, err)
	}
	return made
}

func span(t testing.TB, start, end string) Interval {
	t.Helper()
	made, err := Between(day(t, start), day(t, end))
	if err != nil {
		t.Fatalf("%s to %s: %v", start, end, err)
	}
	return made
}

func ev(t testing.TB, text string) Event {
	t.Helper()
	parts := strings.Fields(text)
	event := Event{ID: parts[0], Kind: Kind(parts[1]), From: day(t, parts[2])}
	if len(parts) > 3 {
		fmt.Sscanf(parts[3], "%d", &event.Revision)
	}
	return event
}

func evs(t testing.TB, texts ...string) []Event {
	t.Helper()
	events := []Event{}
	for _, text := range texts {
		events = append(events, ev(t, text))
	}
	return events
}

func shape(segments []Segment) string {
	parts := []string{}
	for _, segment := range segments {
		parts = append(parts, fmt.Sprintf("%s %s %d", segment.Interval, segment.State, segment.Revision))
	}
	return strings.Join(parts, "; ")
}

func TestDays(t *testing.T) {
	for _, c := range []struct {
		year, month, day int
		valid            bool
	}{{2026, 2, 28, true}, {2026, 2, 29, false}, {2028, 2, 29, true}, {2100, 2, 29, false}, {2000, 2, 29, true}, {1, 1, 1, true}, {9999, 12, 31, true},
		{0, 1, 1, false}, {10000, 1, 1, false}, {2026, 13, 1, false}, {2026, 0, 1, false}, {2026, 4, 31, false}, {2026, 1, 0, false}, {2026, 1, 32, false}} {
		made, err := NewDay(c.year, c.month, c.day)
		if (err == nil) != c.valid || made.Valid() != c.valid {
			t.Errorf("%d-%d-%d: %v %v", c.year, c.month, c.day, made, err)
		}
		if c.valid && made.String() != fmt.Sprintf("%04d-%02d-%02d", c.year, c.month, c.day) {
			t.Errorf("%d-%d-%d reads %s", c.year, c.month, c.day, made)
		}
	}
	if (Day{}).Valid() || (Day{}).String() != "" {
		t.Errorf("the zero day is valid")
	}
	for _, c := range []struct {
		start, end string
		days       int
	}{{"2026-06-01", "2026-07-01", 30}, {"2026-07-01", "2026-08-01", 31}, {"2026-02-01", "2026-03-01", 28}, {"2028-02-01", "2028-03-01", 29},
		{"2026-12-16", "2027-01-16", 31}, {"2026-01-01", "2027-01-01", 365}, {"2028-01-01", "2029-01-01", 366}, {"2026-06-10", "2026-06-11", 1},
		{"0001-01-01", "9999-12-31", 3652058}} {
		if got := span(t, c.start, c.end).Days(); got != c.days {
			t.Errorf("%s to %s: %d days; want %d", c.start, c.end, got, c.days)
		}
		moved, err := day(t, c.start).Add(c.days)
		if err != nil || moved != day(t, c.end) {
			t.Errorf("%s plus %d: %s %v", c.start, c.days, moved, err)
		}
		back, err := day(t, c.end).Add(-c.days)
		if err != nil || back != day(t, c.start) {
			t.Errorf("%s minus %d: %s %v", c.end, c.days, back, err)
		}
	}
	first, last := day(t, "0001-01-01"), day(t, "9999-12-31")
	if _, err := first.Add(-1); !errors.Is(err, ErrInvalid) {
		t.Errorf("a day before the first: %v", err)
	}
	if _, err := last.Add(1); !errors.Is(err, ErrInvalid) {
		t.Errorf("a day after the last: %v", err)
	}
	for _, far := range []int{1<<57 + 719162, 1 << 62, -(1 << 62), 3652059, -3652059, 3652060} {
		if moved, err := first.Add(far); !errors.Is(err, ErrInvalid) {
			t.Errorf("the first day plus %d: %s %v", far, moved, err)
		}
	}
	if moved, err := first.Add(3652058); err != nil || moved != last {
		t.Errorf("the first day plus the whole range: %s %v", moved, err)
	}
	if moved, err := last.Add(-3652058); err != nil || moved != first {
		t.Errorf("the last day minus the whole range: %s %v", moved, err)
	}
	if _, err := (Day{}).Add(1); !errors.Is(err, ErrInvalid) {
		t.Errorf("the zero day moved: %v", err)
	}
	if !first.Before(last) || last.Before(first) || first.Before(first) {
		t.Errorf("order of days")
	}
	same := day(t, "2026-06-01")
	for name, pair := range map[string][2]Day{"an empty interval": {same, same}, "a reversed interval": {last, first}, "no start": {{}, same}, "no end": {same, {}}} {
		if _, err := Between(pair[0], pair[1]); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if (Interval{}).Valid() {
		t.Errorf("the zero interval is valid")
	}
	if got := span(t, "2026-06-01", "2026-07-01"); got.Start() != day(t, "2026-06-01") || got.End() != day(t, "2026-07-01") || got.String() != "[2026-06-01,2026-07-01)" {
		t.Errorf("interval %s", got)
	}
	reference := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if day(t, "2026-06-01").date() != reference {
		t.Errorf("the day does not map to the calendar date")
	}
}

func TestValidLifecycles(t *testing.T) {
	valid := map[string][]string{
		"empty":                    {},
		"a start":                  {"a start 2026-06-01 1"},
		"a full life":              {"a start 2026-06-01 1", "b suspend 2026-06-05", "c change 2026-06-06 2", "d resume 2026-06-08", "e renew 2026-06-09", "f change 2026-06-10 3", "g end 2026-06-20"},
		"ended while suspended":    {"a start 2026-06-01 1", "b suspend 2026-06-05", "c end 2026-06-06"},
		"renewed while suspended":  {"a start 2026-06-01 1", "b suspend 2026-06-05", "c renew 2026-06-06"},
		"back to an earlier terms": {"a start 2026-06-01 1", "b change 2026-06-05 2", "c change 2026-06-06 1"},
		"given in another order":   {"g end 2026-06-20", "a start 2026-06-01 1"},
	}
	for name, texts := range valid {
		if !validLifecycle(ordered(evs(t, texts...))) {
			t.Errorf("%s is refused", name)
		}
	}
	invalid := map[string][]string{
		"no start":                      {"a suspend 2026-06-01"},
		"an end first":                  {"a end 2026-06-01"},
		"a change first":                {"a change 2026-06-01 1"},
		"a renewal first":               {"a renew 2026-06-01"},
		"two starts":                    {"a start 2026-06-01 1", "b start 2026-06-05 2"},
		"a start after the end":         {"a start 2026-06-01 1", "b end 2026-06-05", "c start 2026-06-06 2"},
		"suspended twice":               {"a start 2026-06-01 1", "b suspend 2026-06-05", "c suspend 2026-06-06"},
		"resumed while active":          {"a start 2026-06-01 1", "b resume 2026-06-05"},
		"a change to the same terms":    {"a start 2026-06-01 1", "b change 2026-06-05 1"},
		"an event after the end":        {"a start 2026-06-01 1", "b end 2026-06-05", "c renew 2026-06-06"},
		"a change after the end":        {"a start 2026-06-01 1", "b end 2026-06-05", "c change 2026-06-06 2"},
		"two ends":                      {"a start 2026-06-01 1", "b end 2026-06-05", "c end 2026-06-06"},
		"start and end on one day":      {"a start 2026-06-01 1", "b end 2026-06-01"},
		"resume and change on one day":  {"a start 2026-06-01 1", "b suspend 2026-06-03", "c resume 2026-06-05", "d change 2026-06-05 2"},
		"renew and change on one day":   {"a start 2026-06-01 1", "b renew 2026-06-05", "c change 2026-06-05 2"},
		"an end before the start":       {"a end 2026-05-01", "b start 2026-06-01 1"},
		"a start without terms":         {"a start 2026-06-01"},
		"a change without terms":        {"a start 2026-06-01 1", "b change 2026-06-05"},
		"a suspension with terms":       {"a start 2026-06-01 1", "b suspend 2026-06-05 2"},
		"an end with terms":             {"a start 2026-06-01 1", "b end 2026-06-05 1"},
		"a renewal with terms":          {"a start 2026-06-01 1", "b renew 2026-06-05 1"},
		"a resume with terms":           {"a start 2026-06-01 1", "b suspend 2026-06-03", "c resume 2026-06-05 1"},
		"an unknown kind":               {"a start 2026-06-01 1", "b pause 2026-06-05"},
		"an identifier that is text":    {"a start 2026-06-01 1", "b/c end 2026-06-05"},
		"a suspension after suspension": {"a start 2026-06-01 1", "b suspend 2026-06-05", "c renew 2026-06-06", "d suspend 2026-06-07"},
	}
	for name, texts := range invalid {
		if validLifecycle(ordered(evs(t, texts...))) {
			t.Errorf("%s is accepted", name)
		}
		if _, err := Segments(evs(t, texts...), span(t, "2026-06-01", "2026-07-01")); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: segments %v", name, err)
		}
	}
	if validLifecycle([]Event{{ID: "a", Kind: Start, Revision: 1}}) {
		t.Errorf("an event without a day is accepted")
	}
}

func TestSegments(t *testing.T) {
	life := []string{"a start 2026-06-01 1", "b suspend 2026-06-11", "c resume 2026-06-21", "r renew 2026-06-23", "d change 2026-06-25 2", "e end 2026-07-11"}
	cases := []struct {
		name       string
		events     []string
		start, end string
		want       string
	}{
		{"the whole month", life, "2026-06-01", "2026-07-01",
			"[2026-06-01,2026-06-11) active 1; [2026-06-11,2026-06-21) suspended 1; [2026-06-21,2026-06-25) active 1; [2026-06-25,2026-07-01) active 2"},
		{"before the start", life, "2026-05-01", "2026-06-01", "[2026-05-01,2026-06-01) not_started 0"},
		{"over the start", life, "2026-05-25", "2026-06-05", "[2026-05-25,2026-06-01) not_started 0; [2026-06-01,2026-06-05) active 1"},
		{"over the end", life, "2026-07-01", "2026-08-01", "[2026-07-01,2026-07-11) active 2; [2026-07-11,2026-08-01) ended 0"},
		{"after the end", life, "2026-08-01", "2026-09-01", "[2026-08-01,2026-09-01) ended 0"},
		{"from the end day", life, "2026-07-11", "2026-07-12", "[2026-07-11,2026-07-12) ended 0"},
		{"until the end day", life, "2026-07-10", "2026-07-11", "[2026-07-10,2026-07-11) active 2"},
		{"an event on the first day of the period", life, "2026-06-11", "2026-06-21", "[2026-06-11,2026-06-21) suspended 1"},
		{"an event on the day after the period", life, "2026-06-05", "2026-06-11", "[2026-06-05,2026-06-11) active 1"},
		{"an event on the last day of the period", life, "2026-06-05", "2026-06-12", "[2026-06-05,2026-06-11) active 1; [2026-06-11,2026-06-12) suspended 1"},
		{"a renewal alone cuts nothing", life, "2026-06-22", "2026-06-25", "[2026-06-22,2026-06-25) active 1"},
		{"over everything", life, "2026-05-01", "2026-09-01",
			"[2026-05-01,2026-06-01) not_started 0; [2026-06-01,2026-06-11) active 1; [2026-06-11,2026-06-21) suspended 1; [2026-06-21,2026-06-25) active 1; [2026-06-25,2026-07-11) active 2; [2026-07-11,2026-09-01) ended 0"},
		{"no events", nil, "2026-06-01", "2026-07-01", "[2026-06-01,2026-07-01) not_started 0"},
		{"a contract that never ends", life[:1], "2030-01-01", "2030-02-01", "[2030-01-01,2030-02-01) active 1"},
	}
	for _, c := range cases {
		got, err := Segments(evs(t, c.events...), span(t, c.start, c.end))
		if err != nil || shape(got) != c.want {
			t.Errorf("%s:\n got %s (%v)\nwant %s", c.name, shape(got), err, c.want)
		}
	}
	if _, err := Segments(evs(t, life...), Interval{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("no period: %v", err)
	}
	events := evs(t, life...)
	reversed := append([]Event(nil), events...)
	for a, b := 0, len(reversed)-1; a < b; a, b = a+1, b-1 {
		reversed[a], reversed[b] = reversed[b], reversed[a]
	}
	one, _ := Segments(events, span(t, "2026-05-01", "2026-09-01"))
	two, _ := Segments(reversed, span(t, "2026-05-01", "2026-09-01"))
	if shape(one) != shape(two) || reversed[0].ID != "e" {
		t.Errorf("the order the events are given in matters, or the caller's slice was sorted")
	}
	for name, c := range map[string]struct {
		on   string
		want Position
	}{"before the start": {"2026-05-31", Position{NotStarted, 0}}, "the first day": {"2026-06-01", Position{Active, 1}}, "the day before suspension": {"2026-06-10", Position{Active, 1}},
		"the first suspended day": {"2026-06-11", Position{Suspended, 1}}, "the day of the change": {"2026-06-25", Position{Active, 2}},
		"the last served day": {"2026-07-10", Position{Active, 2}}, "the end day": {"2026-07-11", Position{Ended, 0}}, "long after": {"2030-01-01", Position{Ended, 0}}} {
		if got := positionOn(ordered(events), day(t, c.on)); got != c.want {
			t.Errorf("%s: %+v", name, got)
		}
	}
}
