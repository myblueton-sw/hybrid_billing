package hawkeye

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestTeamLines(t *testing.T) {
	h := newHarness(t, nil)
	h.open(stmt("r1", "s1", 10))
	cases := []struct {
		name string
		line Line
	}{
		{"team", Line{Team: "", Cost: "1", Currency: "KRW", Measured: true}},
		{"a team name that was not encoded", Line{Team: "\ud300 A", Cost: "1", Currency: "KRW", Measured: true}},
		{"cost", Line{Team: "t", Cost: "x", Currency: "KRW", Measured: true}},
		{"a negative cost", Line{Team: "t", Cost: "-1", Currency: "KRW", Measured: true}},
		{"a cost beyond the input bound", Line{Team: "t", Cost: "0.00001", Currency: "KRW", Measured: true}},
		{"currency", Line{Team: "t", Cost: "1", Currency: "krw", Measured: true}},
		{"energy", Line{Team: "t", Cost: "1", Currency: "KRW", Measured: true, Energy: "lots"}},
		{"negative energy", Line{Team: "t", Cost: "1", Currency: "KRW", Measured: true, Energy: "-1"}},
	}
	for _, c := range cases {
		before := h.r.dump()
		_, err := h.r.Post("ws1", "r1", c.line)
		wantErr(t, err, ErrInvalid, c.name)
		h.unchanged(before, c.name)
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "r1", ErrNotFound}, {"ws1", "r9", ErrNotFound}, {"", "r1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.Post(c.ws, c.id, team("t", "1"))
		wantErr(t, err, c.want, "post")
	}
	first := Line{Team: "team-a", Cost: "100.50", Currency: "KRW", Measured: true, Energy: "12.500"}
	h.post("r1", first)
	before := h.r.dump()
	same := first
	same.Cost, same.Energy = "100.5", "12.5"
	h.post("r1", same)
	for name, change := range map[string]func(*Line){
		"cost": func(l *Line) { l.Cost = "100.51" }, "currency": func(l *Line) { l.Currency = "USD" }, "measured": func(l *Line) { l.Measured = false },
		"energy": func(l *Line) { l.Energy = "12.6" }, "no energy": func(l *Line) { l.Energy = "" },
	} {
		other := first
		change(&other)
		_, err := h.r.Post("ws1", "r1", other)
		wantErr(t, err, ErrConflict, name)
	}
	h.unchanged(before, "repeats and conflicts")
	h.post("r1", Line{Team: "team-b", Cost: "0", Currency: "USD", Measured: false}, Line{Team: "team-c", Cost: "0.5", Currency: "EUR", Measured: true})
	before = h.r.dump()
	_, err := h.r.Post("ws1", "r1", team("team-d", "1"))
	wantErr(t, err, ErrRejected, "a line beyond the declared number")
	h.unchanged(before, "a line too many")
	four := stmt("r2", "s2", 11)
	four.Lines = 9
	h.open(four)
	h.post("r2", Line{Team: "a", Cost: "1", Currency: "USD", Measured: true}, Line{Team: "b", Cost: "1", Currency: "EUR", Measured: true})
	before = h.r.dump()
	_, err = h.r.Post("ws1", "r2", Line{Team: "c", Cost: "1", Currency: "JPY", Measured: true})
	wantErr(t, err, ErrRejected, "a fourth currency among declared ones and lines")
	h.unchanged(before, "the currency limit")
	h.post("r2", Line{Team: "c", Cost: "1", Currency: "KRW", Measured: true})
	h.clock.advance(time.Hour)
	h.close("r1")
	view := h.view("r1")
	if view.Lines != 3 || !reflect.DeepEqual(view.Sums, map[string]string{"KRW": "100.5", "USD": "0", "EUR": "0.5"}) || !view.Changed.Equal(start.Add(time.Hour)) || !view.Opened.Equal(start) || !view.WasClosed {
		t.Fatalf("%+v", view)
	}
	if got := h.verdict("r1"); got != "not_eligible[unmeasured_lines totals_differ]" {
		t.Fatalf("a line in a currency the header does not declare shows as a difference: %s", got)
	}
	before = h.r.dump()
	if state, err := h.r.Post("ws1", "r1", first); err != nil || state != Closed {
		t.Fatal(state, err)
	}
	_, err = h.r.Post("ws1", "r1", team("team-z", "1"))
	wantErr(t, err, ErrRejected, "a line for a closed reception")
	if state, err := h.r.Close("ws1", "r1"); err != nil || state != Closed {
		t.Fatal(state, err)
	}
	h.unchanged(before, "after closing")
	lines, err := h.r.Lines("ws1", "r1", 0, 5)
	want := []Line{{Team: "team-a", Cost: "100.5", Currency: "KRW", Measured: true, Energy: "12.5"}, {Team: "team-b", Cost: "0", Currency: "USD"}, {Team: "team-c", Cost: "0.5", Currency: "EUR", Measured: true}}
	if err != nil || !reflect.DeepEqual(lines, want) {
		t.Fatalf("%+v %v", lines, err)
	}
	lines[0].Cost = "999"
	if again, _ := h.r.Lines("ws1", "r1", 0, 1); len(again) != 1 || again[0].Cost != "100.5" {
		t.Fatal("lines share state")
	}
	if page, _ := h.r.Lines("ws1", "r1", 1, 1); len(page) != 1 || page[0].Team != "team-b" {
		t.Fatal(page)
	}
	if page, _ := h.r.Lines("ws1", "r1", 2, 5); len(page) != 1 || page[0].Team != "team-c" {
		t.Fatal(page)
	}
	for _, after := range []int{3, 30} {
		if page, err := h.r.Lines("ws1", "r1", after, 5); err != nil || page != nil {
			t.Fatal(page, err)
		}
	}
	_, err = h.r.Lines("ws1", "r2", 0, 5)
	wantErr(t, err, ErrRejected, "lines of an open reception")
	for _, c := range []struct {
		ws, id       string
		after, limit int
		want         error
	}{{"ws1", "r1", -1, 1, ErrInvalid}, {"ws1", "r1", 0, 0, ErrInvalid}, {"ws1", "r1", 0, 6, ErrInvalid}, {"", "r1", 0, 1, ErrInvalid}, {"ws1", "", 0, 1, ErrInvalid}, {"ws2", "r1", 0, 1, ErrNotFound}, {"ws1", "r9", 0, 1, ErrNotFound}} {
		_, err := h.r.Lines(c.ws, c.id, c.after, c.limit)
		wantErr(t, err, c.want, fmt.Sprint(c))
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "r1", ErrNotFound}, {"ws1", "r9", ErrNotFound}, {"", "r1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.Close(c.ws, c.id)
		wantErr(t, err, c.want, "close")
		_, err = h.r.Eligibility(c.ws, c.id)
		wantErr(t, err, c.want, "eligibility")
		_, err = h.r.Reception(c.ws, c.id)
		wantErr(t, err, c.want, "reception")
		wantErr(t, h.r.Void(c.ws, c.id, VoidedBy), c.want, "void")
	}
	view = h.view("r1")
	view.Spec.Totals["KRW"], view.Sums["KRW"] = "9", "9"
	if again := h.view("r1"); again.Spec.Totals["KRW"] != "300" || again.Sums["KRW"] != "100.5" {
		t.Fatal("view shares state")
	}
}

func TestEveryReasonOfEligibility(t *testing.T) {
	cases := []struct {
		name  string
		set   func(*Spec)
		lines []Line
		want  string
	}{
		{"eligible", nil, []Line{team("a", "100"), team("b", "150.5"), team("c", "49.50")}, "eligible[]"},
		{"a mock", func(s *Spec) { s.Mock = true }, []Line{team("a", "100"), team("b", "100"), team("c", "100")}, "not_eligible[mock]"},
		{"provisional", func(s *Spec) { s.Status = Provisional }, []Line{team("a", "100"), team("b", "100"), team("c", "100")}, "not_eligible[provisional]"},
		{"estimated", func(s *Spec) { s.Tier = Estimated }, []Line{team("a", "100"), team("b", "100"), team("c", "100")}, "not_eligible[not_measured]"},
		{"measurement unavailable", func(s *Spec) { s.Tier = Unavailable }, []Line{team("a", "100"), team("b", "100"), team("c", "100")}, "not_eligible[not_measured]"},
		{"coverage just below the requirement", func(s *Spec) { s.Coverage = 9499 }, []Line{team("a", "100"), team("b", "100"), team("c", "100")}, "not_eligible[coverage_below_required]"},
		{"coverage at the requirement", func(s *Spec) { s.Coverage = 9500 }, []Line{team("a", "100"), team("b", "100"), team("c", "100")}, "eligible[]"},
		{"a declaration derived from the lines", func(s *Spec) { s.Declaration = Derived }, []Line{team("a", "100"), team("b", "100"), team("c", "100")}, "not_eligible[declaration_derived]"},
		{"an unmeasured team reported as zero", func(s *Spec) { s.Totals = map[string]string{"KRW": "200"} }, []Line{team("a", "100"), team("b", "100"), {Team: "c", Cost: "0", Currency: "KRW"}}, "not_eligible[unmeasured_lines]"},
		{"an unmeasured team with a cost", nil, []Line{team("a", "100"), team("b", "100"), {Team: "c", Cost: "100", Currency: "KRW"}}, "not_eligible[unmeasured_lines]"},
		{"a line missing", func(s *Spec) { s.Totals = map[string]string{"KRW": "200"} }, []Line{team("a", "100"), team("b", "100")}, "not_eligible[incomplete_list]"},
		{"no line at all", nil, nil, "not_eligible[incomplete_list totals_differ]"},
		{"a sum that differs by the smallest unit", nil, []Line{team("a", "100"), team("b", "100"), team("c", "100.0001")}, "not_eligible[totals_differ]"},
		{"a sum that differs by a power of ten", func(s *Spec) { s.Totals = map[string]string{"KRW": "30"} }, []Line{team("a", "100"), team("b", "100"), team("c", "100")}, "not_eligible[totals_differ]"},
		{"a sum that is a tenth of the declared one", func(s *Spec) { s.Totals = map[string]string{"KRW": "3"} }, []Line{team("a", "0.1"), team("b", "0.1"), team("c", "0.1")}, "not_eligible[totals_differ]"},
		{"a declared currency without a line", func(s *Spec) { s.Totals = map[string]string{"KRW": "300", "USD": "0"} }, []Line{team("a", "100"), team("b", "100"), team("c", "100")}, "not_eligible[totals_differ]"},
		{"two currencies met", func(s *Spec) { s.Totals = map[string]string{"KRW": "200", "USD": "0.0001"} }, []Line{team("a", "100"), team("b", "100"), {Team: "c", Cost: "0.0001", Currency: "USD", Measured: true}}, "eligible[]"},
		{"a zero total met by a line of zero", func(s *Spec) { s.Totals = map[string]string{"KRW": "300", "USD": "0"} }, []Line{team("a", "100"), team("b", "200"), {Team: "c", Cost: "0", Currency: "USD", Measured: true}}, "eligible[]"},
		{"everything at once", func(s *Spec) {
			s.Mock, s.Status, s.Tier, s.Coverage, s.Declaration = true, Provisional, Unavailable, 0, Derived
		},
			[]Line{team("a", "100"), {Team: "b", Cost: "0", Currency: "USD"}}, "not_eligible[mock provisional not_measured coverage_below_required declaration_derived unmeasured_lines incomplete_list totals_differ]"},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		spec := stmt("r1", "s1", 10)
		if c.set != nil {
			c.set(&spec)
		}
		h.open(spec)
		open := []string{NotClosed}
		for _, reason := range strings.Fields(strings.TrimSuffix(strings.SplitN(c.want, "[", 2)[1], "]")) {
			if reason != UnmeasuredLines && reason != IncompleteList && reason != TotalsDiffer && reason != SumOutsideProfile {
				open = append(open, reason)
			}
		}
		wantOpen := "not_eligible[" + strings.Join(open, " ") + "]"
		h.post("r1", c.lines...)
		if got := h.verdict("r1"); got != wantOpen {
			t.Fatalf("%s while open:\n got %s\nwant %s", c.name, got, wantOpen)
		}
		h.close("r1")
		if got := h.verdict("r1"); got != c.want {
			t.Fatalf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
		wantCounting := ""
		if c.want == "eligible[]" {
			wantCounting = "r1"
		}
		if got := h.counting(10); got != wantCounting {
			t.Fatalf("%s: counting %q", c.name, got)
		}
		if err := h.r.Void("ws1", "r1", VoidedBy); err != nil {
			t.Fatal(err)
		}
		if got := h.verdict("r1"); got != "not_eligible[voided]" || h.counting(10) != "" {
			t.Fatalf("%s voided: %s", c.name, got)
		}
	}
	lax, err := NewRegistry(testLimits(), 0, testProfile(t), (&testClock{at: start}).read)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, r: lax, clock: &testClock{at: start}}
	none := stmt("r1", "s1", 10)
	none.Coverage = 0
	h.full(none)
	if got := h.verdict("r1"); got != "eligible[]" {
		t.Fatalf("a required coverage of zero switches the check off: %s", got)
	}
	small, _ := decimal.NewProfile(decimal.ProfileSpec{ID: "small", Revision: "1", InputIntegerDigits: 3, InputScale: 2, ResultIntegerDigits: 3, ResultScale: 2, IntermediateDigits: 10})
	tight, err := NewRegistry(testLimits(), 9500, small, (&testClock{at: start}).read)
	if err != nil {
		t.Fatal(err)
	}
	h = &harness{t: t, r: tight, clock: &testClock{at: start}}
	big := stmt("r1", "s1", 10)
	big.Totals = map[string]string{"KRW": "999"}
	h.open(big)
	h.post("r1", team("a", "600"), team("b", "600"), team("c", "600"))
	h.close("r1")
	if got := h.verdict("r1"); got != "not_eligible[totals_differ sum_outside_profile]" || h.view("r1").Sums["KRW"] != "1800" {
		t.Fatalf("%s %v", got, h.view("r1").Sums)
	}
}

func TestTheCasesOfTheAcceptance(t *testing.T) {
	h := newHarness(t, nil)
	h.full(stmt("first", "stmt-2026-10-a", 10))
	if h.counting(10) != "first" {
		t.Fatal("one statement of a month counts")
	}
	h.full(stmt("second", "stmt-2026-10-b", 10))
	if a, b := h.verdict("first"), h.verdict("second"); a != "not_eligible[period_conflict] with [second] of 1" || b != "not_eligible[period_conflict] with [first] of 1" || h.counting(10) != "" {
		t.Fatalf("another statement identifier in the same month: %s / %s", a, b)
	}
	if err := h.r.Void("ws1", "second", WrongReport); err != nil {
		t.Fatal(err)
	}
	h.full(superseding("second-again", "stmt-2026-10-b", 10, "stmt-2026-10-a"))
	if a, b := h.verdict("first"), h.verdict("second-again"); a != "not_eligible[superseded]" || b != "eligible[]" || h.counting(10) != "second-again" {
		t.Fatalf("with a supersession exactly one counts: %s / %s", a, b)
	}
	for _, cut := range []int{500, 2000} {
		for _, declaration := range []Declaration{SourceHeader, Derived} {
			other := newHarness(t, nil)
			spec := stmt("r1", "s1", 10)
			spec.Declaration = declaration
			spec.Lines, spec.Totals = cut+700, map[string]string{"KRW": fmt.Sprint((cut + 700) * 10)}
			if declaration == Derived {
				spec.Lines, spec.Totals = cut, map[string]string{"KRW": fmt.Sprint(cut * 10)}
			}
			other.open(spec)
			for index := 0; index < cut; index++ {
				other.post("r1", team(fmt.Sprintf("team-%04d", index), "10"))
			}
			other.close("r1")
			want := "not_eligible[incomplete_list totals_differ]"
			if declaration == Derived {
				want = "not_eligible[declaration_derived]"
			}
			if got := other.verdict("r1"); got != want || other.counting(10) != "" {
				t.Fatalf("a list cut off at %d with a %s declaration: %s", cut, declaration, got)
			}
			if declaration == SourceHeader {
				if err := other.r.Void("ws1", "r1", WrongReport); err != nil {
					t.Fatal(err)
				}
				whole := spec
				whole.ID = "r2"
				other.open(whole)
				for index := 0; index < cut+700; index++ {
					other.post("r2", team(fmt.Sprintf("team-%04d", index), "10"))
				}
				other.close("r2")
				if got := other.verdict("r2"); got != "eligible[]" || other.counting(10) != "r2" {
					t.Fatalf("the whole list reported again under a new reception: %s", got)
				}
				page, err := other.r.Lines("ws1", "r2", 0, 5)
				if err != nil || len(page) != 5 || page[0].Team != "team-0000" || page[1].Team != "team-0001" || page[4].Team != "team-0004" {
					t.Fatalf("lines are read in the order of teams: %v %v", page, err)
				}
			}
		}
	}
	mock := stmt("mock", "stmt-2026-11-test", 11)
	mock.Mock = true
	h.full(mock)
	h.full(stmt("real", "stmt-2026-11", 11))
	if a, b := h.verdict("mock"), h.verdict("real"); a != "not_eligible[mock]" || b != "eligible[]" || h.counting(11) != "real" {
		t.Fatalf("a mock neither counts nor conflicts: %s / %s", a, b)
	}
}

func TestPeriodConflicts(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Listed = 2 })
	from, to := month(10)
	span := func(id string, begin, end time.Time) Spec {
		spec := stmt(id, "s-"+id, 10)
		spec.PeriodStart, spec.PeriodEnd = begin, end
		return spec
	}
	h.full(span("base", from, to))
	h.full(span("before", from.Add(-10*day), from))
	h.full(span("after", to, to.Add(10*day)))
	for _, id := range []string{"base", "before", "after"} {
		if got := h.verdict(id); got != "eligible[]" {
			t.Fatalf("periods that only touch do not overlap: %s %s", id, got)
		}
	}
	other := span("elsewhere", from, to)
	other.Source = "hawk-2"
	h.full(other)
	far := span("base", from, to)
	far.WorkspaceID = "ws2"
	h.r.Open(far)
	if got := h.verdict("base"); got != "eligible[]" {
		t.Fatalf("another source and another workspace do not conflict: %s", got)
	}
	h.open(span("nested", from.Add(day), from.Add(2*day)))
	if got := h.verdict("base"); got != "not_eligible[period_conflict] with [nested] of 1" {
		t.Fatalf("an open reception conflicts: %s", got)
	}
	if got := h.verdict("nested"); got != "not_eligible[not_closed period_conflict] with [base] of 1" {
		t.Fatal(got)
	}
	h.open(span("edge", to.Add(-time.Second), to.Add(day)))
	never := span("never", from, to)
	never.Status, never.Tier = Provisional, Unavailable
	h.open(never)
	if got := h.verdict("base"); got != "not_eligible[period_conflict] with [edge nested] of 3" {
		t.Fatalf("a reception that can never be eligible conflicts, and the list is cut at the bound: %s", got)
	}
	if got := h.verdict("after"); got != "not_eligible[period_conflict] with [edge] of 1" {
		t.Fatal(got)
	}
	if got := h.verdict("never"); got != "not_eligible[not_closed provisional not_measured period_conflict] with [base edge] of 3" {
		t.Fatal(got)
	}
	mock := span("mock", from, to)
	mock.Mock = true
	h.open(mock)
	if got := h.verdict("mock"); got != "not_eligible[not_closed mock]" {
		t.Fatalf("a mock is given no conflict: %s", got)
	}
	for _, id := range []string{"nested", "edge"} {
		if err := h.r.Void("ws1", id, WrongReport); err != nil {
			t.Fatal(err)
		}
	}
	if got := h.verdict("base"); got != "not_eligible[period_conflict] with [never] of 1" || h.verdict("after") != "eligible[]" {
		t.Fatal(got)
	}
	h.full(func() Spec {
		s := superseding("newer", "s-newer", 10, "s-never")
		s.PeriodStart, s.PeriodEnd = to.Add(20*day), to.Add(30*day)
		return s
	}())
	if got := h.verdict("base"); got != "eligible[]" {
		t.Fatalf("a conflict removed by a supersession: %s", got)
	}
	if got := h.verdict("never"); got != "not_eligible[not_closed provisional not_measured superseded]" {
		t.Fatalf("a superseded reception is given no conflict: %s", got)
	}
	twin := span("twin", from, to)
	twin.Revision = 5
	h.full(twin)
	if h.counting(10) != "" {
		t.Fatal("two receptions of the same period")
	}
	if err := h.r.Void("ws1", "twin", VoidedBy); err != nil {
		t.Fatal(err)
	}
	if h.counting(10) != "base" {
		t.Fatal("voiding the newer of two is a choice for the older")
	}
}

func TestCountingAndTheSequenceNumber(t *testing.T) {
	h := newHarness(t, nil)
	from, to := month(10)
	if out, err := h.r.Counting("ws1", "hawk-1", from, to); err != ErrNotFound || out.Sequence != 0 {
		t.Fatal(out, err)
	}
	for _, c := range []struct {
		ws, source string
		from, to   time.Time
	}{{"", "hawk-1", from, to}, {"ws1", "", from, to}, {"ws1", "hawk-1", to, from}, {"ws1", "hawk-1", from, from}, {"ws1", "hawk-1", from.Add(time.Millisecond), to}, {"ws1", "hawk-1", from, to.Add(time.Nanosecond)}, {"ws1", "hawk-1", from, from.Add(41 * day)}} {
		_, err := h.r.Counting(c.ws, c.source, c.from, c.to)
		wantErr(t, err, ErrInvalid, "counting")
	}
	expect := func(reception string, sequence uint64) {
		t.Helper()
		out, err := h.r.Counting("ws1", "hawk-1", from, to)
		if out.Reception != reception || out.Sequence != sequence || (err == nil) != (reception != "") || (reception != "" && out.Statement != "s1") {
			t.Fatalf("%+v %v, want %q at %d", out, err, reception, sequence)
		}
	}
	spec := stmt("r1", "s1", 10)
	h.open(spec)
	expect("", 1)
	h.r.Open(spec)
	h.r.Open(stmt("r1", "s9", 10))
	h.r.Open(stmt("r2", "s1", 10))
	expect("", 1)
	h.post("r1", team("a", "100"))
	h.post("r1", team("a", "100"))
	h.r.Post("ws1", "r1", team("a", "101"))
	expect("", 2)
	h.post("r1", team("b", "100"), team("c", "100"))
	h.r.Post("ws1", "r1", team("d", "100"))
	h.close("r1")
	h.close("r1")
	expect("r1", 5)
	if out, err := h.r.Counting("ws1", "hawk-1", from, to.Add(-time.Second)); err != ErrNotFound || out.Reception != "" || out.Sequence != 5 {
		t.Fatalf("only the exact period counts: %+v %v", out, err)
	}
	if out, err := h.r.Counting("ws1", "hawk-1", from.In(time.FixedZone("kst", 9*3600)), to); err != nil || out.Reception != "r1" {
		t.Fatalf("the same instants in another zone: %+v %v", out, err)
	}
	other := stmt("x1", "s1", 10)
	other.Source = "hawk-2"
	h.full(other)
	expect("r1", 5)
	if out, err := h.r.Counting("ws2", "hawk-1", from, to); err != ErrNotFound || out.Sequence != 0 {
		t.Fatal(out, err)
	}
	h.r.Void("ws1", "r1", "nonsense")
	if err := h.r.Void("ws1", "r1", ReissuedAtSource); err != nil {
		t.Fatal(err)
	}
	h.r.Void("ws1", "r1", ReissuedAtSource)
	h.r.Void("ws1", "r1", VoidedBy)
	expect("", 6)
	h.r.Close("ws1", "r1")
	h.r.Post("ws1", "r1", team("z", "1"))
	expect("", 6)
	for _, reason := range []VoidReason{WithdrawnAtSource, ReissuedAtSource, WrongReport, VoidedBy} {
		one := newHarness(t, nil)
		many := stmt("r1", "s1", 10)
		many.Lines = 20
		one.open(many)
		one.post("r1", team("b", "1"), team("a", "1"))
		for index := 9; index >= 0; index-- {
			one.post("r1", team(fmt.Sprintf("t%02d", index), "1"))
		}
		for _, bad := range []VoidReason{"", "superseded", "other"} {
			wantErr(t, one.r.Void("ws1", "r1", bad), ErrInvalid, "void reason")
		}
		one.clock.advance(time.Hour)
		if err := one.r.Void("ws1", "r1", reason); err != nil {
			t.Fatal(err)
		}
		view := one.view("r1")
		if view.State != Voided || view.VoidReason != reason || view.WasClosed || !view.Changed.Equal(start.Add(time.Hour)) || view.Sums != nil {
			t.Fatalf("%+v", view)
		}
		before := one.r.dump()
		if err := one.r.Void("ws1", "r1", reason); err != nil {
			t.Fatal(err)
		}
		next := WrongReport
		if reason == WrongReport {
			next = VoidedBy
		}
		wantErr(t, one.r.Void("ws1", "r1", next), ErrConflict, "voided with another reason")
		_, err := one.r.Close("ws1", "r1")
		wantErr(t, err, ErrRejected, "closing a reception voided while open")
		_, err = one.r.Post("ws1", "r1", team("c", "1"))
		wantErr(t, err, ErrRejected, "a line for a voided reception")
		if state, err := one.r.Post("ws1", "r1", team("a", "1")); err != nil || state != Voided {
			t.Fatal(state, err)
		}
		one.unchanged(before, "after voiding")
		if lines, err := one.r.Lines("ws1", "r1", 0, 5); err != nil || len(lines) != 5 || lines[0].Team != "a" || lines[1].Team != "b" || lines[2].Team != "t00" || lines[4].Team != "t02" {
			t.Fatalf("the lines of a voided reception stay readable in order: %v %v", lines, err)
		}
	}
	closed := newHarness(t, nil)
	closed.full(stmt("r1", "s1", 10))
	closed.r.Void("ws1", "r1", WithdrawnAtSource)
	if state, err := closed.r.Close("ws1", "r1"); err != nil || state != Voided || !closed.view("r1").WasClosed {
		t.Fatal(state, err)
	}
}

func TestReceptionsOfASourceInPages(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Page = 3 })
	if rows, err := h.r.Receptions("ws1", "hawk-1", "", 3); err != nil || rows != nil {
		t.Fatal(rows, err)
	}
	if h.r.dump() != "" {
		t.Fatal("a read created something")
	}
	for index, id := range []string{"r4", "r2", "r6", "r8"} {
		h.open(stmt(id, "s"+id, 3+index))
	}
	mock := stmt("r5", "s-mock", 9)
	mock.Mock = true
	h.open(mock)
	h.open(superseding("r7", "s-new", 10, "sr2"))
	h.r.Void("ws1", "r6", WrongReport)
	other := stmt("r3", "s-other", 10)
	other.Source = "hawk-2"
	h.open(other)
	first, err := h.r.Receptions("ws1", "hawk-1", "", 3)
	from, to := month(4)
	want := []Row{{ID: "r2", Statement: "sr2", PeriodStart: from, PeriodEnd: to, State: Open, Superseded: true}, {ID: "r4", Statement: "sr4", State: Open}, {ID: "r5", Statement: "s-mock", State: Open, Mock: true}}
	want[1].PeriodStart, want[1].PeriodEnd = month(3)
	want[2].PeriodStart, want[2].PeriodEnd = month(9)
	if err != nil || !reflect.DeepEqual(first, want) {
		t.Fatalf("%+v %v", first, err)
	}
	h.open(stmt("r1", "s-early", 11))
	h.open(stmt("r55", "s-mid", 12))
	next, _ := h.r.Receptions("ws1", "hawk-1", "r5", 3)
	if len(next) != 3 || next[0].ID != "r55" || next[1].ID != "r6" || next[1].State != Voided || next[2].ID != "r7" {
		t.Fatalf("a page does not shift when receptions are added: %+v", next)
	}
	if last, _ := h.r.Receptions("ws1", "hawk-1", "r7", 3); len(last) != 1 || last[0].ID != "r8" {
		t.Fatal(last)
	}
	if none, err := h.r.Receptions("ws1", "hawk-1", "r8", 3); err != nil || none != nil {
		t.Fatal(none, err)
	}
	if absent, _ := h.r.Receptions("ws1", "hawk-1", "r45", 1); len(absent) != 1 || absent[0].ID != "r5" {
		t.Fatalf("the lower bound need not exist: %+v", absent)
	}
	if rows, err := h.r.Receptions("ws2", "hawk-1", "", 3); err != nil || rows != nil {
		t.Fatal(rows, err)
	}
	for _, c := range []struct {
		ws, source, after string
		limit             int
	}{{"", "hawk-1", "", 1}, {"ws1", "", "", 1}, {"ws1", "hawk-1", "a b", 1}, {"ws1", "hawk-1", "", 0}, {"ws1", "hawk-1", "", 4}} {
		_, err := h.r.Receptions(c.ws, c.source, c.after, c.limit)
		wantErr(t, err, ErrInvalid, fmt.Sprint(c))
	}
}

func TestRefusalsAreFixedValuesAndNothingIsPriced(t *testing.T) {
	h := newHarness(t, nil)
	h.open(stmt("r1", "s1", 10))
	secret := "customer-text"
	changed := stmt("r1", "s1", 10)
	changed.CostOrigin = secret
	_, _, conflict := h.r.Open(changed)
	_, _, invalid := h.r.Open(stmt(secret+" x", "s2", 10))
	_, missing := h.r.Post("ws1", secret, team(secret, "1"))
	_, _, rejected := h.r.Open(superseding("r2", "s2", 10, secret))
	for index, pair := range []struct{ got, want error }{{conflict, ErrConflict}, {invalid, ErrInvalid}, {missing, ErrNotFound}, {rejected, ErrRejected}} {
		if pair.got != pair.want {
			t.Fatalf("%d: %v", index, pair.got)
		}
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	floating := regexp.MustCompile(`float32|float64|big\.Float|big\.Rat|ParseFloat|FormatFloat|math\.|%[0-9.]*[fgeFGE]\b|complex`)
	pricing := regexp.MustCompile(`decimal\.(Mul|Quo|DivideRound|Round|Quanta|RoundToQuantum)\b`)
	checked := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		checked++
		text := strings.ReplaceAll(string(content), `"math/big"`, "")
		if found := floating.FindString(text); found != "" {
			t.Fatalf("%s uses %q", file, found)
		}
		if found := pricing.FindString(text); found != "" {
			t.Fatalf("%s multiplies or divides a decimal: %q", file, found)
		}
	}
	if checked != 4 || !floating.MatchString("x := float64(1)") || !pricing.MatchString("decimal.Mul(p, cost, price)") || !pricing.MatchString("decimal.Quo(p, a, b)") {
		t.Fatalf("%d source files checked", checked)
	}
}

func TestReasonOrdersExitsAndRepeatsOnSupersededStatements(t *testing.T) {
	h := newHarness(t, nil)
	cut := stmt("r1", "s1", 10)
	cut.Declaration = Derived
	h.open(cut)
	first := team("a", "100")
	h.post("r1", first, team("b", "100"))
	h.close("r1")
	if state, _, err := h.r.Open(cut); err != nil || state != Closed {
		t.Fatalf("a repeated open of a closed reception: %v %v", state, err)
	}
	if _, occupant, err := h.r.Open(stmt("r1x", "s1", 10)); err != ErrConflict || occupant != "r1" {
		t.Fatalf("a second reception while the first is closed: %q %v", occupant, err)
	}
	h.open(superseding("r2", "s2", 11, "s1"))
	if got := h.verdict("r1"); got != "not_eligible[declaration_derived superseded incomplete_list totals_differ]" {
		t.Fatal(got)
	}
	before := h.r.dump()
	if state, err := h.r.Post("ws1", "r1", first); err != nil || state != Closed {
		t.Fatalf("a repeated line of a superseded statement: %v %v", state, err)
	}
	if state, err := h.r.Close("ws1", "r1"); err != nil || state != Closed {
		t.Fatalf("a repeated close of a superseded statement: %v %v", state, err)
	}
	h.unchanged(before, "repeats on a superseded statement")
	h.open(stmt("o1", "s-o1", 9))
	h.post("o1", first)
	h.open(superseding("o2", "s-o2", 8, "s-o1"))
	if state, err := h.r.Post("ws1", "o1", first); err != nil || state != Open {
		t.Fatalf("a repeated line of a reception superseded while open: %v %v", state, err)
	}
	h.open(stmt("r3", "s3", 12))
	h.post("r3", team("a", "100"), team("b", "100"), Line{Team: "c", Cost: "100", Currency: "KRW"})
	h.close("r3")
	h.open(stmt("r4", "s4", 12))
	if got := h.verdict("r3"); got != "not_eligible[period_conflict unmeasured_lines] with [r4] of 1" {
		t.Fatal(got)
	}
	if err := h.r.Void("ws1", "r2", WithdrawnAtSource); err != nil {
		t.Fatal(err)
	}
	if h.counting(10) != "" || h.counting(11) != "" {
		t.Fatal("after a supersession and a void nothing counts")
	}
	h.full(stmt("r5", "s5", 10))
	if got := h.verdict("r5"); got != "eligible[]" || h.counting(10) != "r5" {
		t.Fatalf("a further statement is the exit: %s", got)
	}
	h.open(stmt("v1", "s-v1", 7))
	if err := h.r.Void("ws1", "v1", WrongReport); err != nil {
		t.Fatal(err)
	}
	h.open(superseding("v2", "s-v2", 7, "s-v1"))
	if h.view("v1").SupersededBy != "s-v2" {
		t.Fatal("a statement whose reception was voided may be named")
	}
}
