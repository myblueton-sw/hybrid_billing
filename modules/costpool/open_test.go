package costpool

import (
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/allocation"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestConstructionAndRulesForStrings(t *testing.T) {
	clock := func() time.Time { return start }
	good := profileOf(t, 12, 14)
	if _, err := NewRegistry(testLimits(), good, clock); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Limits){
		"pools": func(l *Limits) { l.Pools = 0 }, "versions": func(l *Limits) { l.Versions = 0 }, "assets": func(l *Limits) { l.Assets = 0 }, "components": func(l *Limits) { l.Components = 0 },
		"targets": func(l *Limits) { l.Targets = 0 }, "page": func(l *Limits) { l.Page = 0 }, "listed": func(l *Limits) { l.Listed = 0 }, "period": func(l *Limits) { l.MaxPeriod = 0 }, "intermediate": func(l *Limits) { l.Intermediate = 0 },
	} {
		limits := testLimits()
		change(&limits)
		_, err := NewRegistry(limits, good, clock)
		wantErr(t, err, ErrInvalid, name)
	}
	_, err := NewRegistry(testLimits(), decimal.Profile{}, clock)
	wantErr(t, err, ErrInvalid, "no profile")
	_, err = NewRegistry(testLimits(), good, nil)
	wantErr(t, err, ErrInvalid, "no clock")
	long, longer := strings.Repeat("a", 128), strings.Repeat("a", 129)
	for id, want := range map[string]bool{"a": true, "A.z_0:9-": true, long: true, longer: false, "": false, "a b": false, "a/b": false, "kim@example": false, "é": false} {
		if validID(id) != want {
			t.Errorf("identifier %q", id)
		}
		if id == "" {
			continue
		}
		weight, _ := decimal.Parse(good, "1")
		_, err := allocation.LargestRemainderV1(allocation.Request{Profile: good, Limits: allocation.Limits{MaxTargets: 2, MaxIntermediateDigits: 40}, Encoding: allocation.EncodingASCIIBytewiseV1,
			Total: weight, Quantum: weight, Mode: decimal.HalfEven, Targets: []allocation.Target{{ID: id, Weight: weight}}})
		if (err == nil) != want {
			t.Errorf("the allocation and this package differ on the identifier %q: %v", id, err)
		}
	}
	for code, want := range map[string]bool{"KRW": true, "USD": true, "krw": false, "KR": false, "KRWW": false, "K1W": false, "1RW": false, "KR1": false, "KRw": false, "kRW": false, "KrW": false} {
		if validCurrency(code) != want {
			t.Errorf("currency %q", code)
		}
	}
}

func TestHeaderRefusalsRepeatsAndSupersession(t *testing.T) {
	h := newHarness(t, nil)
	from, to := month(10)
	malformed := map[string]func(*Spec){
		"workspace":                   func(s *Spec) { s.WorkspaceID = "" },
		"pool":                        func(s *Spec) { s.Pool = "dc 1" },
		"owner":                       func(s *Spec) { s.Owner = "" },
		"version zero":                func(s *Spec) { s.Version = 0 },
		"a version above the maximum": func(s *Spec) { s.Version = 1000000 },
		"currency":                    func(s *Spec) { s.Currency = "krw" },
		"a quantum of two":            func(s *Spec) { s.Quantum = "2" },
		"a quantum of ten":            func(s *Spec) { s.Quantum = "10" },
		"a quantum written otherwise": func(s *Spec) { s.Quantum = "1.0" },
		"no quantum":                  func(s *Spec) { s.Quantum = "" },
		"a period backwards":          func(s *Spec) { s.PeriodStart, s.PeriodEnd = to, from },
		"an empty period":             func(s *Spec) { s.PeriodEnd = from },
		"a start with a fraction":     func(s *Spec) { s.PeriodStart = from.Add(time.Millisecond) },
		"an end with a fraction":      func(s *Spec) { s.PeriodEnd = to.Add(time.Millisecond) },
		"a period that is too long":   func(s *Spec) { s.PeriodEnd = from.Add(40*day + time.Second) },
		"superseding itself":          func(s *Spec) { s.Supersedes = s.Version },
		"superseding a later one":     func(s *Spec) { s.Supersedes = s.Version + 1 },
		"superseding a negative one":  func(s *Spec) { s.Supersedes = -1 },
	}
	for name, change := range malformed {
		spec := header(5, 10, 0)
		change(&spec)
		wantErr(t, second(h.r.Open(spec)), ErrInvalid, name)
	}
	if h.r.dump() != "" {
		t.Fatal("a refused header wrote")
	}
	h = newHarness(t, func(l *Limits) { l.Pools = 5 })
	for _, quantum := range []string{"1", "0.1", "0.01", "0.001", "0.0001"} {
		spec := header(1, 10, 0)
		spec.Pool, spec.Quantum = "q-"+strings.ReplaceAll(quantum, ".", "_"), quantum
		h.open(spec)
	}
	coarse, err := NewRegistry(testLimits(), mustProfile(t, 2), h.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	fine := header(1, 10, 0)
	fine.Quantum = "0.001"
	wantErr(t, second(coarse.Open(fine)), ErrInvalid, "a quantum finer than the profile reads")
	h = newHarness(t, nil)
	edge := header(999999, 10, 0)
	edge.PeriodEnd = from.Add(40 * day)
	h.open(edge)
	spec := header(1, 10, 0)
	spec.PeriodStart = from.In(time.FixedZone("kst", 9*3600))
	h.open(spec)
	h.clock.advance(time.Hour)
	before := h.r.dump()
	h.open(header(1, 10, 0))
	if view := h.view(1); view.Spec.PeriodStart.Location() != time.UTC || !view.Opened.Equal(start) || !view.Changed.Equal(start) || view.State != Draft || view.Recorded || view.Superseded {
		t.Fatalf("%+v", view)
	}
	for name, change := range map[string]func(*Spec){
		"currency": func(s *Spec) { s.Currency = "USD" }, "start": func(s *Spec) { s.PeriodStart = from.Add(time.Second) }, "end": func(s *Spec) { s.PeriodEnd = to.Add(-time.Second) },
		"owner": func(s *Spec) { s.Owner = "owner-2" }, "quantum": func(s *Spec) { s.Quantum = "0.1" },
	} {
		other := header(1, 10, 0)
		change(&other)
		wantErr(t, second(h.r.Open(other)), ErrConflict, name)
	}
	h.unchanged(before, "repeats and conflicts")
	for name, spec := range map[string]Spec{
		"a version that does not exist":         header(3, 10, 2),
		"a version without a recorded approval": header(3, 10, 1),
	} {
		wantErr(t, second(h.r.Open(spec)), ErrRejected, name)
	}
	h.put(1, cost("fee", Facility, "1200000"))
	h.basis(1, shares("60", "20", "20"))
	h.record(1)
	if state, err := h.r.Open(header(1, 10, 0)); err != nil || state != ApprovalRecorded {
		t.Fatal(state, err)
	}
	for name, change := range map[string]func(*Spec){
		"another period":   func(s *Spec) { s.PeriodStart, s.PeriodEnd = month(11) },
		"a shorter period": func(s *Spec) { s.PeriodEnd = s.PeriodEnd.Add(-time.Second) },
		"a later start":    func(s *Spec) { s.PeriodStart = s.PeriodStart.Add(time.Second) },
		"another currency": func(s *Spec) { s.Currency = "USD" },
	} {
		spec := header(2, 10, 1)
		change(&spec)
		before := h.r.dump()
		wantErr(t, second(h.r.Open(spec)), ErrRejected, name)
		h.unchanged(before, name)
	}
	h.open(header(2, 10, 1))
	h.open(header(3, 10, 1))
	wantErr(t, second(h.r.Open(header(2, 10, 0))), ErrConflict, "the same version naming nothing")
	if view := h.view(1); view.Superseded || h.standing(1) != "current[]" {
		t.Fatal("naming a version changes nothing until the approval is recorded")
	}
	for _, number := range []int{2, 3} {
		h.put(number, cost("fee", Facility, "900000"))
		h.basis(number, shares("1", "1"))
	}
	h.clock.advance(time.Hour)
	h.record(2)
	if view := h.view(1); !view.Changed.Equal(start.Add(2 * time.Hour)) {
		t.Fatalf("being superseded is a change: %s", view.Changed)
	}
	if view := h.view(1); !view.Superseded || view.SupersededBy != 2 || h.standing(1) != "not_current[superseded]" || h.standing(2) != "current[]" {
		t.Fatalf("%+v", view)
	}
	before = h.r.dump()
	wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 3, approved)), ErrRejected, "the second of two versions that name the same one")
	wantErr(t, second(h.r.Open(header(4, 10, 1))), ErrRejected, "naming a version that is superseded")
	h.unchanged(before, "a version superseded meanwhile")
	if blockers := h.view(3).Blockers; !blockers.NamedSuperseded || blockers.Overlapping.Total != 1 || blockers.Overlapping.Items[0] != 2 {
		t.Fatalf("%+v", blockers)
	}
	if err := h.r.Withdraw("ws1", "dc-1", 2, EnteredInError); err != nil {
		t.Fatal(err)
	}
	if h.standing(1) != "not_current[superseded]" || h.standing(2) != "not_current[withdrawn]" {
		t.Fatal("being superseded is not undone by a withdrawal")
	}
	if number, _ := h.counting(10); number != 0 {
		t.Fatal(number)
	}
	h.whole(header(5, 10, 2), "700000")
	if view := h.view(2); !view.Superseded || view.SupersededBy != 5 || view.State != Withdrawn {
		t.Fatalf("a withdrawn version that had its approval recorded can be named: %+v", view)
	}
	if err := h.r.Withdraw("ws1", "dc-1", 5, ByOperator); err != nil {
		t.Fatal(err)
	}
	h.whole(header(6, 10, 0), "600000")
	if number, _ := h.counting(10); number != 6 || h.standing(6) != "current[]" {
		t.Fatal("the period is free again for a version that names nothing")
	}
	if err := h.r.Withdraw("ws1", "dc-1", 1, ByOperator); err != nil {
		t.Fatal(err)
	}
	if view := h.view(1); !view.Superseded || view.State != Withdrawn || h.standing(1) != "not_current[withdrawn]" {
		t.Fatalf("a superseded version can be withdrawn and stays superseded: %+v", view)
	}
}

func mustProfile(t testing.TB, scale int) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "coarse", Revision: "1", InputIntegerDigits: 12, InputScale: scale, ResultIntegerDigits: 14, ResultScale: 6, IntermediateDigits: 40})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func TestLimitsOfPoolsAndVersions(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Pools, l.Versions = 2, 2 })
	h.open(header(1, 10, 0))
	h.open(header(2, 11, 0))
	wantErr(t, second(h.r.Open(header(3, 12, 0))), ErrLimit, "a third version")
	if err := h.r.Withdraw("ws1", "dc-1", 1, ByOperator); err != nil {
		t.Fatal(err)
	}
	wantErr(t, second(h.r.Open(header(3, 12, 0))), ErrLimit, "a withdrawn version still counts")
	h.open(header(2, 11, 0))
	other := header(1, 10, 0)
	other.Pool = "dc-2"
	h.open(other)
	third := header(1, 10, 0)
	third.Pool = "dc-3"
	before := h.r.dump()
	wantErr(t, second(h.r.Open(third)), ErrLimit, "a third pool")
	h.unchanged(before, "limits")
	far := header(1, 10, 0)
	far.WorkspaceID, far.Pool = "ws2", "dc-3"
	if _, err := h.r.Open(far); err != nil {
		t.Fatalf("limits are per workspace: %v", err)
	}
	invalid := header(3, 12, 0)
	invalid.Currency = ""
	wantErr(t, second(h.r.Open(invalid)), ErrInvalid, "malformed before no room")
	conflict := header(2, 11, 0)
	conflict.Owner = "owner-2"
	wantErr(t, second(h.r.Open(conflict)), ErrConflict, "a conflict before no room")
	wantErr(t, second(h.r.Open(header(3, 12, 9))), ErrInvalid, "a later version named")
	wantErr(t, second(h.r.Open(header(3, 12, 1))), ErrRejected, "rejected before no room")
}
