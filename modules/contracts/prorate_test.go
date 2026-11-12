package contracts

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func profile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic", Revision: "1", InputIntegerDigits: 12, InputScale: 4, ResultIntegerDigits: 14, ResultScale: 4, IntermediateDigits: 40})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func dec(t testing.TB, text string) decimal.Decimal {
	t.Helper()
	made, err := decimal.Parse(profile(t), text)
	if err != nil {
		t.Fatalf("%s: %v", text, err)
	}
	return made
}

func prices(t testing.TB, pairs ...string) map[uint64]decimal.Decimal {
	t.Helper()
	amounts := map[uint64]decimal.Decimal{}
	for index := 0; index < len(pairs); index += 2 {
		var revision uint64
		fmt.Sscanf(pairs[index], "%d", &revision)
		amounts[revision] = dec(t, pairs[index+1])
	}
	return amounts
}

func basisFor(t testing.TB, start, end string) Basis {
	return Basis{WorkspaceID: "w1", ContractID: "c1", ChargeID: "base", Period: span(t, start, end), Point: 1, Days: ActualDays, Suspended: SkipSuspended, Scale: 0, Mode: decimal.HalfUp}
}

func lines(result Result) string {
	parts := []string{}
	for _, line := range result.Lines {
		parts = append(parts, fmt.Sprintf("%s r%d %dd %s", line.Interval, line.Revision, line.Days, line.Amount))
	}
	return strings.Join(parts, "; ")
}

func revisions(result Result) string {
	parts := []string{}
	for _, entry := range result.Revisions {
		parts = append(parts, fmt.Sprintf("r%d %s of %s", entry.Revision, entry.Total, entry.Price))
	}
	return strings.Join(parts, "; ")
}

func prorated(t testing.TB, basis Basis, amounts map[uint64]decimal.Decimal, events ...string) Result {
	t.Helper()
	segments, err := Segments(evs(t, events...), basis.Period)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Prorate(profile(t), basis, amounts, segments)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestTheAcceptanceCases(t *testing.T) {
	p := profile(t)
	basis := basisFor(t, "2026-06-01", "2026-07-01")
	price := prices(t, "1", "300000")
	issued := prorated(t, basis, price, "a start 2026-06-01 1")
	if issued.Total.String() != "300000" || lines(issued) != "[2026-06-01,2026-07-01) r1 30d 300000" || issued.PeriodDays != 30 {
		t.Errorf("the advance bill for the full period: %s total %s", lines(issued), issued.Total)
	}
	later := basis
	later.Point = 2
	served := prorated(t, later, price, "a start 2026-06-01 1", "b end 2026-06-11")
	if served.Total.String() != "100000" || lines(served) != "[2026-06-01,2026-06-11) r1 10d 100000" || revisions(served) != "r1 100000 of 300000" {
		t.Errorf("300,000 for 30 days of which 10: %s total %s", lines(served), served.Total)
	}
	withArrears, err := decimal.Add(p, served.Total, dec(t, "50000"))
	if err != nil || withArrears.String() != "150000" {
		t.Errorf("with the arrears addition of 50,000: %s %v", withArrears, err)
	}
	correction, err := Correction(p, issued, served)
	if err != nil || correction.Total.String() != "-200000" || len(correction.Revisions) != 1 || correction.Revisions[0].Amount.String() != "-200000" ||
		correction.Before.Basis.Point != 1 || correction.After.Basis.Point != 2 {
		t.Errorf("the correction of the advance bill: %+v %v", correction, err)
	}
}

func TestProrationTables(t *testing.T) {
	june := []string{"2026-06-01", "2026-07-01"}
	july := []string{"2026-07-01", "2026-08-01"}
	three := []string{"a start 2026-07-01 1", "b suspend 2026-07-11", "c resume 2026-07-21"}
	cases := []struct {
		name      string
		period    []string
		suspended SuspendedRule
		scale     int
		mode      decimal.Mode
		amounts   []string
		events    []string
		lines     string
		total     string
		revisions string
	}{
		{"a price change in mid-period", june, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000", "2", "360000"},
			[]string{"a start 2026-06-01 1", "b change 2026-06-11 2"},
			"[2026-06-01,2026-06-11) r1 10d 100000; [2026-06-11,2026-07-01) r2 20d 240000", "340000", "r1 100000 of 300000; r2 240000 of 360000"},
		{"suspended days skipped", june, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000"},
			[]string{"a start 2026-06-01 1", "b suspend 2026-06-11", "c resume 2026-06-21"},
			"[2026-06-01,2026-06-11) r1 10d 100000; [2026-06-21,2026-07-01) r1 10d 100000", "200000", "r1 200000 of 300000"},
		{"suspended days billed", june, BillSuspended, 0, decimal.HalfUp, []string{"1", "300000"},
			[]string{"a start 2026-06-01 1", "b suspend 2026-06-11", "c resume 2026-06-21"},
			"[2026-06-01,2026-06-11) r1 10d 100000; [2026-06-11,2026-06-21) r1 10d 100000; [2026-06-21,2026-07-01) r1 10d 100000", "300000", "r1 300000 of 300000"},
		{"a period before the start", june, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000"}, []string{"a start 2026-07-01 1"}, "", "0", ""},
		{"a period after the end", june, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000"}, []string{"a start 2026-05-01 1", "b end 2026-06-01"}, "", "0", ""},
		{"a start and an end inside the period", june, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000"}, []string{"a start 2026-06-11 1", "b end 2026-06-21"},
			"[2026-06-11,2026-06-21) r1 10d 100000", "100000", "r1 100000 of 300000"},
		{"no events at all", june, SkipSuspended, 0, decimal.HalfUp, nil, nil, "", "0", ""},
		{"ten days of twenty-eight", []string{"2026-02-01", "2026-03-01"}, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000"},
			[]string{"a start 2026-02-01 1", "b end 2026-02-11"}, "[2026-02-01,2026-02-11) r1 10d 107143", "107143", "r1 107143 of 300000"},
		{"ten days of twenty-eight to the cent", []string{"2026-02-01", "2026-03-01"}, SkipSuspended, 2, decimal.HalfUp, []string{"1", "300000"},
			[]string{"a start 2026-02-01 1", "b end 2026-02-11"}, "[2026-02-01,2026-02-11) r1 10d 107142.86", "107142.86", "r1 107142.86 of 300000"},
		{"ten days of twenty-nine", []string{"2028-02-01", "2028-03-01"}, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000"},
			[]string{"a start 2028-02-01 1", "b end 2028-02-11"}, "[2028-02-01,2028-02-11) r1 10d 103448", "103448", "r1 103448 of 300000"},
		{"ten days of thirty-one", july, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000"},
			[]string{"a start 2026-07-01 1", "b end 2026-07-11"}, "[2026-07-01,2026-07-11) r1 10d 96774", "96774", "r1 96774 of 300000"},
		{"over the twenty-ninth of February", []string{"2028-02-15", "2028-03-15"}, SkipSuspended, 0, decimal.HalfUp, []string{"1", "290000"},
			[]string{"a start 2028-02-15 1", "b end 2028-03-01"}, "[2028-02-15,2028-03-01) r1 15d 150000", "150000", "r1 150000 of 290000"},
		{"over a year boundary", []string{"2026-12-16", "2027-01-16"}, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000"},
			[]string{"a start 2026-12-16 1", "b end 2027-01-01"}, "[2026-12-16,2027-01-01) r1 16d 154839", "154839", "r1 154839 of 300000"},
		{"a full period of thirty-one days", july, SkipSuspended, 0, decimal.Floor, []string{"1", "100000"}, []string{"a start 2026-01-01 1"},
			"[2026-07-01,2026-08-01) r1 31d 100000", "100000", "r1 100000 of 100000"},
		{"three segments, half up", july, BillSuspended, 0, decimal.HalfUp, []string{"1", "100000"}, three,
			"[2026-07-01,2026-07-11) r1 10d 32258; [2026-07-11,2026-07-21) r1 10d 32258; [2026-07-21,2026-08-01) r1 11d 35484", "100000", "r1 100000 of 100000"},
		{"three segments, half even", july, BillSuspended, 0, decimal.HalfEven, []string{"1", "100000"}, three,
			"[2026-07-01,2026-07-11) r1 10d 32258; [2026-07-11,2026-07-21) r1 10d 32258; [2026-07-21,2026-08-01) r1 11d 35484", "100000", "r1 100000 of 100000"},
		{"three segments, floor", july, BillSuspended, 0, decimal.Floor, []string{"1", "100000"}, three,
			"[2026-07-01,2026-07-11) r1 10d 32258; [2026-07-11,2026-07-21) r1 10d 32258; [2026-07-21,2026-08-01) r1 11d 35484", "100000", "r1 100000 of 100000"},
		{"three segments, toward zero", july, BillSuspended, 0, decimal.TowardZero, []string{"1", "100000"}, three,
			"[2026-07-01,2026-07-11) r1 10d 32258; [2026-07-11,2026-07-21) r1 10d 32258; [2026-07-21,2026-08-01) r1 11d 35484", "100000", "r1 100000 of 100000"},
		{"three segments, ceiling", july, BillSuspended, 0, decimal.Ceiling, []string{"1", "100000"}, three,
			"[2026-07-01,2026-07-11) r1 10d 32259; [2026-07-11,2026-07-21) r1 10d 32258; [2026-07-21,2026-08-01) r1 11d 35483", "100000", "r1 100000 of 100000"},
		{"three segments, away from zero", july, BillSuspended, 0, decimal.AwayFromZero, []string{"1", "100000"}, three,
			"[2026-07-01,2026-07-11) r1 10d 32259; [2026-07-11,2026-07-21) r1 10d 32258; [2026-07-21,2026-08-01) r1 11d 35483", "100000", "r1 100000 of 100000"},
		{"a credit in three segments, half up", july, BillSuspended, 0, decimal.HalfUp, []string{"1", "-100000"}, three,
			"[2026-07-01,2026-07-11) r1 10d -32258; [2026-07-11,2026-07-21) r1 10d -32258; [2026-07-21,2026-08-01) r1 11d -35484", "-100000", "r1 -100000 of -100000"},
		{"a credit in three segments, floor", july, BillSuspended, 0, decimal.Floor, []string{"1", "-100000"}, three,
			"[2026-07-01,2026-07-11) r1 10d -32259; [2026-07-11,2026-07-21) r1 10d -32258; [2026-07-21,2026-08-01) r1 11d -35483", "-100000", "r1 -100000 of -100000"},
		{"a credit in three segments, ceiling", july, BillSuspended, 0, decimal.Ceiling, []string{"1", "-100000"}, three,
			"[2026-07-01,2026-07-11) r1 10d -32258; [2026-07-11,2026-07-21) r1 10d -32258; [2026-07-21,2026-08-01) r1 11d -35484", "-100000", "r1 -100000 of -100000"},
		{"a credit in three segments, toward zero", july, BillSuspended, 0, decimal.TowardZero, []string{"1", "-100000"}, three,
			"[2026-07-01,2026-07-11) r1 10d -32258; [2026-07-11,2026-07-21) r1 10d -32258; [2026-07-21,2026-08-01) r1 11d -35484", "-100000", "r1 -100000 of -100000"},
		{"ten days of thirty-one, floor", july, SkipSuspended, 0, decimal.Floor, []string{"1", "100000"}, three[:2],
			"[2026-07-01,2026-07-11) r1 10d 32258", "32258", "r1 32258 of 100000"},
		{"the mirrored credit, floor", july, SkipSuspended, 0, decimal.Floor, []string{"1", "-100000"}, three[:2],
			"[2026-07-01,2026-07-11) r1 10d -32259", "-32259", "r1 -32259 of -100000"},
		{"a zero line for a share that is not zero", june, BillSuspended, 0, decimal.HalfUp, []string{"1", "1"},
			[]string{"a start 2026-06-01 1", "b suspend 2026-06-11", "c resume 2026-06-21"},
			"[2026-06-01,2026-06-11) r1 10d 0; [2026-06-11,2026-06-21) r1 10d 1; [2026-06-21,2026-07-01) r1 10d 0", "1", "r1 1 of 1"},
		{"a zero amount", june, SkipSuspended, 0, decimal.HalfUp, []string{"1", "0", "2", "300000"}, []string{"a start 2026-06-01 1", "b change 2026-06-16 2"},
			"[2026-06-01,2026-06-16) r1 15d 0; [2026-06-16,2026-07-01) r2 15d 150000", "150000", "r1 0 of 0; r2 150000 of 300000"},
		{"an amount for terms that are not billed", june, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000", "9", "-1.2345"}, []string{"a start 2026-06-01 1"},
			"[2026-06-01,2026-07-01) r1 30d 300000", "300000", "r1 300000 of 300000"},
		{"terms returning after a change", june, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000", "2", "600000"},
			[]string{"a start 2026-06-01 1", "b change 2026-06-11 2", "c change 2026-06-21 1"},
			"[2026-06-01,2026-06-11) r1 10d 100000; [2026-06-11,2026-06-21) r2 10d 200000; [2026-06-21,2026-07-01) r1 10d 100000", "400000", "r1 200000 of 300000; r2 200000 of 600000"},
		{"later terms with the lower number", june, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000", "2", "600000"},
			[]string{"a start 2026-06-01 2", "b change 2026-06-21 1"},
			"[2026-06-01,2026-06-21) r2 20d 400000; [2026-06-21,2026-07-01) r1 10d 100000", "500000", "r1 100000 of 300000; r2 400000 of 600000"},
		{"to four decimals", july, SkipSuspended, 4, decimal.HalfEven, []string{"1", "0.0031"}, three[:2], "[2026-07-01,2026-07-11) r1 10d 0.001", "0.001", "r1 0.001 of 0.0031"},
		{"suspended and skipped terms need no amount", june, SkipSuspended, 0, decimal.HalfUp, []string{"1", "300000"},
			[]string{"a start 2026-06-01 1", "b suspend 2026-06-11", "c change 2026-06-16 2", "d end 2026-06-21"},
			"[2026-06-01,2026-06-11) r1 10d 100000", "100000", "r1 100000 of 300000"},
	}
	for _, c := range cases {
		basis := basisFor(t, c.period[0], c.period[1])
		basis.Suspended, basis.Scale, basis.Mode = c.suspended, c.scale, c.mode
		got := prorated(t, basis, prices(t, c.amounts...), c.events...)
		if lines(got) != c.lines || got.Total.String() != c.total || revisions(got) != c.revisions {
			t.Errorf("%s:\n got %s | total %s | %s\nwant %s | total %s | %s", c.name, lines(got), got.Total, revisions(got), c.lines, c.total, c.revisions)
		}
		if got.Basis != basis || got.ProfileID != "synthetic" || got.ProfileRevision != "1" || got.PeriodDays != basis.Period.Days() {
			t.Errorf("%s: the result does not carry its basis: %+v", c.name, got)
		}
	}
}

func manual(t testing.TB, start string, states ...any) (Interval, []Segment) {
	t.Helper()
	cursor := day(t, start)
	segments := []Segment{}
	for index := 0; index < len(states); index += 3 {
		next, err := cursor.Add(states[index].(int))
		if err != nil {
			t.Fatal(err)
		}
		segments = append(segments, Segment{Interval: Interval{start: cursor, end: next}, State: states[index+1].(State), Revision: uint64(states[index+2].(int))})
		cursor = next
	}
	return Interval{start: day(t, start), end: cursor}, segments
}

func TestALineCanBeOffByExactlyOneUnit(t *testing.T) {
	period, segments := manual(t, "2026-06-01", 1, Active, 1, 2, Suspended, 1, 3, Active, 1)
	basis := basisFor(t, "2026-06-01", "2026-06-07")
	basis.Period, basis.Suspended, basis.Mode = period, BillSuspended, decimal.HalfEven
	got, err := Prorate(profile(t), basis, prices(t, "1", "3"), segments)
	if err != nil || lines(got) != "[2026-06-01,2026-06-02) r1 1d 0; [2026-06-02,2026-06-04) r1 2d 2; [2026-06-04,2026-06-07) r1 3d 1" || got.Total.String() != "3" {
		t.Errorf("%s total %s %v", lines(got), got.Total, err)
	}
}

func TestProrationRefusals(t *testing.T) {
	p := profile(t)
	period, segments := manual(t, "2026-06-01", 10, Active, 1, 10, Suspended, 1, 10, Active, 2)
	good := basisFor(t, "2026-06-01", "2026-07-01")
	amounts := prices(t, "1", "300000", "2", "360000")
	if good.Period != period {
		t.Fatal("the fixture period")
	}
	if _, err := Prorate(p, good, amounts, segments); err != nil {
		t.Fatalf("the valid call: %v", err)
	}
	for name, change := range map[string]func(*Basis){
		"no workspace": func(b *Basis) { b.WorkspaceID = "" }, "no contract": func(b *Basis) { b.ContractID = "" }, "no charge": func(b *Basis) { b.ChargeID = "" },
		"a charge that is text": func(b *Basis) { b.ChargeID = "base fee" }, "no period": func(b *Basis) { b.Period = Interval{} },
		"no day basis": func(b *Basis) { b.Days = "" }, "a thirty-day convention": func(b *Basis) { b.Days = "thirty_days" },
		"no rule for suspended days": func(b *Basis) { b.Suspended = "" }, "an unknown rule for suspended days": func(b *Basis) { b.Suspended = "half" },
		"a negative scale": func(b *Basis) { b.Scale = -1 }, "no rounding mode": func(b *Basis) { b.Mode = 0 }, "an unknown rounding mode": func(b *Basis) { b.Mode = 99 },
		"another period than the segments cover": func(b *Basis) { b.Period = span(t, "2026-06-01", "2026-06-30") },
	} {
		basis := good
		change(&basis)
		if _, err := Prorate(p, basis, amounts, segments); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	over := good
	over.Scale = 5
	if _, err := Prorate(p, over, amounts, segments); !errors.Is(err, decimal.ErrInvalidScale) {
		t.Errorf("a scale beyond the profile: %v", err)
	}
	idlePeriod, idle := manual(t, "2026-06-01", 30, NotStarted, 0)
	if over.Period != idlePeriod {
		t.Fatal("the fixture period")
	}
	if _, err := Prorate(p, over, nil, idle); !errors.Is(err, decimal.ErrInvalidScale) {
		t.Errorf("a scale beyond the profile when nothing is billed: %v", err)
	}
	if got, err := Prorate(p, good, nil, idle); err != nil || got.Total.String() != "0" || len(got.Lines) != 0 || len(got.Revisions) != 0 {
		t.Errorf("a period in which nothing is billed: %+v %v", got, err)
	}
	zero := good
	zero.Point, zero.Scale = 0, 0
	if _, err := Prorate(p, zero, amounts, segments); err != nil {
		t.Errorf("knowledge point 0 and scale 0 are values, not absences: %v", err)
	}
	if _, err := Prorate(decimal.Profile{}, good, amounts, segments); !errors.Is(err, decimal.ErrInvalidProfile) {
		t.Errorf("no profile: %v", err)
	}
	gap := append([]Segment(nil), segments...)
	gap[1].Interval.start, _ = gap[1].Interval.start.Add(1)
	overlap := append([]Segment(nil), segments...)
	overlap[1].Interval.start, _ = overlap[1].Interval.start.Add(-1)
	unknown := append([]Segment(nil), segments...)
	unknown[1].State = "paused"
	empty := append([]Segment(nil), segments...)
	empty[1].Interval.end = empty[1].Interval.start
	for name, given := range map[string][]Segment{
		"no segments": nil, "only the billable part": {segments[0], segments[2]}, "one chunk": segments[:1], "the tail": segments[1:], "the wrong order": {segments[2], segments[1], segments[0]},
		"an empty segment between its neighbours": {segments[0], {Interval: Interval{start: segments[1].Interval.start, end: segments[1].Interval.start}, State: Active, Revision: 1}, segments[1], segments[2]},
		"a reversed segment between its neighbours": {segments[0], {Interval: Interval{start: segments[0].Interval.end, end: segments[2].Interval.start}, State: Suspended, Revision: 1},
			{Interval: Interval{start: segments[2].Interval.start, end: segments[1].Interval.start}, State: Active, Revision: 1}, segments[1], segments[2]},
		"a gap": gap, "an overlap": overlap, "an unknown state": unknown, "an empty segment": empty, "a segment twice": {segments[0], segments[0], segments[1], segments[2]},
	} {
		if _, err := Prorate(p, good, amounts, given); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, given := range map[string]map[uint64]decimal.Decimal{
		"no amount for billed terms": prices(t, "1", "300000"), "no amounts": nil, "an amount that is not set": {1: dec(t, "300000"), 2: {}},
		"more decimals than the scale": prices(t, "1", "300000", "2", "100.5"), "amounts of both signs": prices(t, "1", "300000", "2", "-360000"),
	} {
		if _, err := Prorate(p, good, given, segments); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	cents := good
	cents.Scale = 2
	if _, err := Prorate(p, cents, prices(t, "1", "100.005", "2", "1"), segments); !errors.Is(err, ErrInvalid) {
		t.Errorf("100.005 at two decimals: %v", err)
	}
	if got, err := Prorate(p, cents, prices(t, "1", "100.5", "2", "0"), segments); err != nil || got.Total.String() != "33.5" {
		t.Errorf("a zero amount beside a positive one: %v %s", err, got.Total)
	}
	if got, err := Prorate(p, good, prices(t, "1", "0", "2", "-360000"), segments); err != nil || got.Total.String() != "-120000" {
		t.Errorf("a zero amount beside a negative one: %v %s", err, got.Total)
	}
	year, served := manual(t, "2026-01-01", 365, Active, 1)
	yearly := good
	yearly.Period = year
	if _, err := Prorate(p, yearly, prices(t, "1", "999999999999"), served); !errors.Is(err, decimal.ErrResultBound) {
		t.Errorf("an amount whose product with the days passes the bound: %v", err)
	}
	if got, err := Prorate(p, yearly, prices(t, "1", "100000000000"), served); err != nil || got.Total.String() != "100000000000" {
		t.Errorf("an amount whose product with the days fits: %v %s", err, got.Total)
	}
	if got, err := Prorate(p, good, prices(t, "1", "999999999999", "2", "999999999999"), segments); err != nil || got.Total.String() != "666666666666" {
		t.Errorf("the largest input over a month: %v %s", err, got.Total)
	}
}

func TestCorrections(t *testing.T) {
	p := profile(t)
	basis := basisFor(t, "2026-06-01", "2026-07-01")
	price := prices(t, "1", "300000", "2", "360000")
	first := prorated(t, basis, price, "a start 2026-06-01 1")
	second := basis
	second.Point = 2
	changed := prorated(t, second, price, "a start 2026-06-01 1", "b change 2026-06-11 2")
	netted, err := Correction(p, first, changed)
	if err != nil || netted.Total.String() != "40000" || len(netted.Revisions) != 2 || netted.Revisions[0].Revision != 1 || netted.Revisions[0].Amount.String() != "-200000" ||
		netted.Revisions[1].Revision != 2 || netted.Revisions[1].Amount.String() != "240000" {
		t.Errorf("the net of two terms: %+v %v", netted, err)
	}
	third := basis
	third.Point = 3
	ended := prorated(t, third, price, "a start 2026-06-01 1", "b change 2026-06-11 2", "c end 2026-06-16")
	onward, err := Correction(p, changed, ended)
	if err != nil || onward.Total.String() != "-180000" || onward.Revisions[0].Amount.String() != "0" || onward.Revisions[1].Amount.String() != "-180000" {
		t.Errorf("the second correction: %+v %v", onward, err)
	}
	whole, err := Correction(p, first, ended)
	sum, _ := decimal.Add(p, netted.Total, onward.Total)
	if err != nil || whole.Total.String() != "-140000" || !sum.Equal(whole.Total) {
		t.Errorf("corrections over three knowledge points do not add up: %s and %s", sum, whole.Total)
	}
	back, err := Correction(p, ended, first)
	if err != nil || back.Total.String() != "140000" || back.Revisions[1].Amount.String() != "-60000" {
		t.Errorf("the reverse correction: %+v %v", back, err)
	}
	same, err := Correction(p, first, first)
	if err != nil || same.Total.String() != "0" || same.Revisions[0].Amount.String() != "0" {
		t.Errorf("a result against itself: %+v %v", same, err)
	}
	other, _ := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic", Revision: "2", InputIntegerDigits: 12, InputScale: 4, ResultIntegerDigits: 14, ResultScale: 4, IntermediateDigits: 40})
	named, _ := decimal.NewProfile(decimal.ProfileSpec{ID: "another", Revision: "1", InputIntegerDigits: 12, InputScale: 4, ResultIntegerDigits: 14, ResultScale: 4, IntermediateDigits: 40})
	for name, profile := range map[string]decimal.Profile{"another revision of the profile": other, "another profile": named} {
		if _, err := Correction(profile, first, changed); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := Correction(decimal.Profile{}, first, changed); !errors.Is(err, decimal.ErrInvalidProfile) {
		t.Errorf("no profile: %v", err)
	}
	for name, change := range map[string]func(*Basis){
		"another workspace": func(b *Basis) { b.WorkspaceID = "w2" }, "another contract": func(b *Basis) { b.ContractID = "c2" }, "another charge": func(b *Basis) { b.ChargeID = "other" },
		"another period": func(b *Basis) { b.Period = span(t, "2026-07-01", "2026-07-31") }, "another rule for suspended days": func(b *Basis) { b.Suspended = BillSuspended },
		"another scale": func(b *Basis) { b.Scale = 2 }, "another rounding mode": func(b *Basis) { b.Mode = decimal.Floor },
	} {
		moved := second
		change(&moved)
		segments, _ := Segments(evs(t, "a start 2026-06-01 1", "b change 2026-06-11 2"), moved.Period)
		after, err := Prorate(p, moved, price, segments)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := Correction(p, first, after); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
		if _, err := Correction(p, after, first); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s, reversed: %v", name, err)
		}
	}
	repriced := prorated(t, second, prices(t, "1", "330000", "2", "360000"), "a start 2026-06-01 1", "b change 2026-06-11 2")
	if _, err := Correction(p, first, repriced); !errors.Is(err, ErrInvalid) {
		t.Errorf("another price for terms billed in both: %v", err)
	}
	elsewhere := prorated(t, second, prices(t, "1", "300000", "2", "1"), "a start 2026-06-01 1", "b change 2026-06-11 2")
	if _, err := Correction(p, first, elsewhere); err != nil {
		t.Errorf("another price for terms billed in one result only: %v", err)
	}
	forged := changed
	forged.Revisions = []RevisionTotal{{Revision: 1, Price: dec(t, "300000"), Total: dec(t, "250000")}}
	if _, err := Correction(p, first, forged); !errors.Is(err, ErrInvalid) {
		t.Errorf("a result whose totals per terms do not sum to its total: %v", err)
	}
	doubled := first
	doubled.Revisions = []RevisionTotal{{Revision: 1, Price: dec(t, "300000"), Total: dec(t, "100000")}, {Revision: 1, Price: dec(t, "300000"), Total: dec(t, "200000")}}
	for name, pair := range map[string][2]Result{"before": {doubled, changed}, "after": {changed, doubled}} {
		if _, err := Correction(p, pair[0], pair[1]); !errors.Is(err, ErrInvalid) {
			t.Errorf("terms listed twice in the result %s: %v", name, err)
		}
	}
	unpriced := first
	unpriced.Revisions = []RevisionTotal{{Revision: 1, Total: dec(t, "300000")}}
	if _, err := Correction(p, unpriced, changed); !errors.Is(err, ErrInvalid) {
		t.Errorf("terms without a price: %v", err)
	}
	apart := prorated(t, second, price, "a start 2026-06-01 2")
	if _, err := Correction(p, unpriced, apart); !errors.Is(err, ErrInvalid) {
		t.Errorf("terms without a price that the other result does not bill: %v", err)
	}
	for attempt := 0; attempt < 40; attempt++ {
		many, err := Correction(p, apart, prorated(t, third, prices(t, "1", "300000", "2", "360000", "3", "30000"), "a start 2026-06-01 3", "b change 2026-06-11 1", "c change 2026-06-21 2"))
		if err != nil || len(many.Revisions) != 3 || many.Revisions[0].Revision != 1 || many.Revisions[1].Revision != 2 || many.Revisions[2].Revision != 3 ||
			many.Revisions[0].Amount.String() != "100000" || many.Revisions[1].Amount.String() != "-240000" || many.Revisions[2].Amount.String() != "10000" || many.Total.String() != "-130000" {
			t.Fatalf("the differences of three terms, in order: %+v %v", many.Revisions, err)
		}
	}
	if _, err := Correction(p, Result{}, first); !errors.Is(err, ErrInvalid) {
		t.Errorf("an empty result: %v", err)
	}
	if _, err := Correction(p, first, Result{Basis: first.Basis, ProfileID: "synthetic", ProfileRevision: "1"}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a result without a total: %v", err)
	}
}
