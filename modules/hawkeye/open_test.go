package hawkeye

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestConstructionAndRulesForStrings(t *testing.T) {
	clock := &testClock{at: start}
	for index, change := range []func(*Limits){func(l *Limits) { l.Receptions = 0 }, func(l *Limits) { l.Lines = 0 }, func(l *Limits) { l.Currencies = 0 }, func(l *Limits) { l.Page = 0 }, func(l *Limits) { l.Listed = 0 }, func(l *Limits) { l.MaxPeriod = 0 }} {
		limits := testLimits()
		change(&limits)
		if _, err := NewRegistry(limits, 9500, testProfile(t), clock.read); err != ErrInvalid {
			t.Fatalf("limit %d: %v", index, err)
		}
	}
	for _, coverage := range []int{-1, 10001} {
		if _, err := NewRegistry(testLimits(), coverage, testProfile(t), clock.read); err != ErrInvalid {
			t.Fatal(coverage, err)
		}
	}
	for _, coverage := range []int{0, 10000} {
		if _, err := NewRegistry(testLimits(), coverage, testProfile(t), clock.read); err != nil {
			t.Fatal(coverage, err)
		}
	}
	if _, err := NewRegistry(testLimits(), 9500, decimal.Profile{}, clock.read); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := NewRegistry(testLimits(), 9500, testProfile(t), nil); err != ErrInvalid {
		t.Fatal(err)
	}
	var none *Registry
	from, to := month(10)
	for _, r := range []*Registry{none, {}} {
		for index, err := range []error{third(r.Open(stmt("r", "s", 10))), second(r.Post("ws1", "r", team("a", "1"))), second(r.Close("ws1", "r")), r.Void("ws1", "r", VoidedBy), second(r.Eligibility("ws1", "r")),
			second(r.Counting("ws1", "hawk-1", from, to)), second(r.Reception("ws1", "r")), second(r.Lines("ws1", "r", 0, 1)), second(r.Receptions("ws1", "hawk-1", "", 1))} {
			if err != ErrInvalid {
				t.Fatalf("operation %d on a registry that was not built: %v", index, err)
			}
		}
	}
	long := ""
	for len(long) < 128 {
		long += "a"
	}
	for text, want := range map[string]bool{"a": true, "A.z_0:9-": true, long: true, long + "a": false, "": false, "a b": false, "a/b": false, "팀": false, "a@b": false, "a[b": false, "a`b": false, "a{b": false} {
		if validID(text) != want {
			t.Errorf("identifier %q", text)
		}
	}
	for text, want := range map[string]bool{"KRW": true, "AZZ": true, "ZAA": true, "KR": false, "KRWW": false, "kRW": false, "KrW": false, "KRw": false, "@RW": false, "K[W": false, "KR1": false, "": false} {
		if validCurrency(text) != want {
			t.Errorf("currency %q", text)
		}
	}
	h := newHarness(t, nil)
	h.clock.at = start.Add(time.Hour).In(time.FixedZone("kst", 9*3600))
	h.open(stmt("r1", "s1", 10))
	h.clock.advance(-30 * time.Minute)
	h.open(stmt("r2", "s2", 11))
	for _, id := range []string{"r1", "r2"} {
		if view := h.view(id); !view.Opened.Equal(start.Add(time.Hour)) || view.Opened.Location() != time.UTC || !view.Changed.Equal(view.Opened) {
			t.Fatalf("%s: %+v", id, view)
		}
	}
}

func TestHeaderRefusals(t *testing.T) {
	from, to := month(10)
	cases := []struct {
		name string
		set  func(*Spec)
		want error
	}{
		{"workspace", func(s *Spec) { s.WorkspaceID = "" }, ErrInvalid},
		{"id", func(s *Spec) { s.ID = "r 1" }, ErrInvalid},
		{"source", func(s *Spec) { s.Source = "" }, ErrInvalid},
		{"statement", func(s *Spec) { s.Statement = "" }, ErrInvalid},
		{"cost origin", func(s *Spec) { s.CostOrigin = "" }, ErrInvalid},
		{"schema", func(s *Spec) { s.Schema = "" }, ErrInvalid},
		{"connection", func(s *Spec) { s.Connection = "" }, ErrInvalid},
		{"a period that runs backwards", func(s *Spec) { s.PeriodStart, s.PeriodEnd = to, from }, ErrInvalid},
		{"a period of no length", func(s *Spec) { s.PeriodEnd = from }, ErrInvalid},
		{"a start that is not a whole second", func(s *Spec) { s.PeriodStart = from.Add(time.Millisecond) }, ErrInvalid},
		{"an end that is not a whole second", func(s *Spec) { s.PeriodEnd = to.Add(-time.Nanosecond) }, ErrInvalid},
		{"a period beyond the longest", func(s *Spec) { s.PeriodEnd = from.Add(40*day + time.Second) }, ErrInvalid},
		{"revision zero", func(s *Spec) { s.Revision = 0 }, ErrInvalid},
		{"revision too large", func(s *Spec) { s.Revision = 1000000 }, ErrInvalid},
		{"a supersession without a reason", func(s *Spec) { s.Supersedes = "s0" }, ErrInvalid},
		{"a reason without a supersession", func(s *Spec) { s.CorrectionReason = "recount" }, ErrInvalid},
		{"a malformed superseded statement", func(s *Spec) { s.Supersedes, s.CorrectionReason = "s 0", "recount" }, ErrInvalid},
		{"a reason that is free text", func(s *Spec) { s.Supersedes, s.CorrectionReason = "s0", "we counted again" }, ErrInvalid},
		{"superseding itself", func(s *Spec) { s.Supersedes, s.CorrectionReason = "s1", "recount" }, ErrInvalid},
		{"a mock with a supersession", func(s *Spec) { s.Mock, s.Supersedes, s.CorrectionReason = true, "s0", "recount" }, ErrInvalid},
		{"status", func(s *Spec) { s.Status = "draft" }, ErrInvalid},
		{"no status", func(s *Spec) { s.Status = "" }, ErrInvalid},
		{"tier", func(s *Spec) { s.Tier = "guessed" }, ErrInvalid},
		{"declaration", func(s *Spec) { s.Declaration = "" }, ErrInvalid},
		{"coverage negative", func(s *Spec) { s.Coverage = -1 }, ErrInvalid},
		{"coverage over the whole", func(s *Spec) { s.Coverage = 10001 }, ErrInvalid},
		{"no line declared", func(s *Spec) { s.Lines = 0 }, ErrInvalid},
		{"no total", func(s *Spec) { s.Totals = nil }, ErrInvalid},
		{"a malformed currency", func(s *Spec) { s.Totals = map[string]string{"krw": "1"} }, ErrInvalid},
		{"a total that is no number", func(s *Spec) { s.Totals = map[string]string{"KRW": "many"} }, ErrInvalid},
		{"a negative total", func(s *Spec) { s.Totals = map[string]string{"KRW": "-0.0001"} }, ErrInvalid},
		{"a total beyond the input bound", func(s *Spec) { s.Totals = map[string]string{"KRW": "1000000000000"} }, ErrInvalid},
		{"more lines than a reception may have", func(s *Spec) { s.Lines = 3001 }, ErrLimit},
		{"more currencies than a reception may have", func(s *Spec) { s.Totals = map[string]string{"KRW": "1", "USD": "1", "EUR": "1", "JPY": "1"} }, ErrLimit},
		{"an unknown superseded statement", func(s *Spec) { s.Supersedes, s.CorrectionReason = "s0", "recount" }, ErrRejected},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		spec := stmt("r1", "s1", 10)
		c.set(&spec)
		_, _, err := h.r.Open(spec)
		wantErr(t, err, c.want, c.name)
		if h.r.dump() != "" {
			t.Fatalf("%s wrote something", c.name)
		}
	}
	h := newHarness(t, func(l *Limits) { l.Receptions = 3 })
	edge := stmt("r1", "s1", 10)
	edge.PeriodEnd, edge.Lines, edge.Revision = from.Add(40*day), 3000, 999999
	edge.Coverage, edge.Totals = 0, map[string]string{"KRW": "0", "USD": "0", "EUR": "0"}
	h.open(edge)
	for index, variant := range []func(*Spec){func(s *Spec) { s.Status = Provisional }, func(s *Spec) { s.Tier = Estimated; s.Coverage = 10000; s.Declaration = Derived; s.Mock = true }} {
		spec := stmt([]string{"r2", "r3"}[index], []string{"s2", "s3"}[index], 3+index)
		variant(&spec)
		h.open(spec)
	}
	before := h.r.dump()
	_, _, err := h.r.Open(stmt("r4", "s4", 6))
	wantErr(t, err, ErrLimit, "reception limit")
	h.open(edge)
	h.unchanged(before, "the reception limit and a repeat at it")
}

func TestHeaderRepeatsAndOneReceptionPerStatement(t *testing.T) {
	h := newHarness(t, nil)
	h.open(stmt("r0", "s0", 9))
	spec := superseding("r1", "s1", 10, "s0")
	h.open(spec)
	h.clock.advance(time.Hour)
	before := h.r.dump()
	again := superseding("r1", "s1", 10, "s0")
	again.Schema, again.Connection = "v4", "conn-2"
	again.Totals = map[string]string{"KRW": "300.00"}
	again.PeriodStart = again.PeriodStart.In(time.FixedZone("kst", 9*3600))
	if state, occupant, err := h.r.Open(again); err != nil || state != Open || occupant != "" {
		t.Fatalf("the same reception through another connection and schema version: %v %q %v", state, occupant, err)
	}
	if view := h.view("r1"); view.Spec.Schema != "v3" || view.Spec.Connection != "conn-1" || view.Spec.Totals["KRW"] != "300" || view.Spec.PeriodStart.Location() != time.UTC {
		t.Fatalf("%+v", view.Spec)
	}
	from, _ := month(10)
	for name, change := range map[string]func(*Spec){
		"source":                func(s *Spec) { s.Source = "hawk-2" },
		"statement":             func(s *Spec) { s.Statement = "s9" },
		"cost origin":           func(s *Spec) { s.CostOrigin = "cooling" },
		"start":                 func(s *Spec) { s.PeriodStart = from.Add(time.Second) },
		"end":                   func(s *Spec) { s.PeriodEnd = s.PeriodEnd.Add(-time.Second) },
		"revision":              func(s *Spec) { s.Revision = 3 },
		"superseded":            func(s *Spec) { s.Supersedes = "s7" },
		"reason":                func(s *Spec) { s.CorrectionReason = "late-meter" },
		"no longer superseding": func(s *Spec) { s.Supersedes, s.CorrectionReason = "", "" },
		"mock":                  func(s *Spec) { s.Mock, s.Supersedes, s.CorrectionReason = true, "", "" },
		"status":                func(s *Spec) { s.Status = Provisional },
		"tier":                  func(s *Spec) { s.Tier = Estimated },
		"coverage":              func(s *Spec) { s.Coverage = 9799 },
		"lines":                 func(s *Spec) { s.Lines = 4 },
		"total":                 func(s *Spec) { s.Totals = map[string]string{"KRW": "300.0001"} },
		"currency":              func(s *Spec) { s.Totals = map[string]string{"USD": "300"} },
		"a currency more":       func(s *Spec) { s.Totals = map[string]string{"KRW": "300", "USD": "0"} },
		"declaration":           func(s *Spec) { s.Declaration = Derived },
	} {
		other := superseding("r1", "s1", 10, "s0")
		change(&other)
		_, _, err := h.r.Open(other)
		wantErr(t, err, ErrConflict, name)
	}
	flagged := stmt("r0", "s0", 9)
	flagged.Mock = true
	_, _, err := h.r.Open(flagged)
	wantErr(t, err, ErrConflict, "the mock flag alone")
	state, occupant, err := h.r.Open(stmt("r9", "s1", 10))
	if err != ErrConflict || state != "" || occupant != "r1" {
		t.Fatalf("a second reception of a statement: %v %q %v", state, occupant, err)
	}
	h.unchanged(before, "repeats and conflicts")
	far := stmt("r1", "s1", 10)
	far.WorkspaceID = "ws2"
	if state, _, err := h.r.Open(far); err != nil || state != Open {
		t.Fatalf("the same identifiers in another workspace: %v %v", state, err)
	}
	other := stmt("r5", "s1", 10)
	other.Source = "hawk-2"
	h.open(other)
	if err := h.r.Void("ws1", "r1", WrongReport); err != nil {
		t.Fatal(err)
	}
	corrected := superseding("r1b", "s1", 10, "s0")
	corrected.Coverage, corrected.Revision = 9900, 7
	h.open(corrected)
	before = h.r.dump()
	if state, occupant, err := h.r.Open(spec); err != nil || state != Voided || occupant != "" {
		t.Fatalf("an identical open of a voided reception whose statement another reception took: %v %q %v", state, occupant, err)
	}
	h.unchanged(before, "a repeat of a voided reception")
	if view := h.view("r1b"); view.Spec.Coverage != 9900 || !view.Opened.Equal(start.Add(time.Hour)) {
		t.Fatalf("%+v", view)
	}
	spelled := stmt("sp", "s-spelled", 3)
	spelled.Totals = map[string]string{"KRW": "12.50"}
	h.open(spelled)
	if got := h.view("sp").Spec.Totals["KRW"]; got != "12.5" {
		t.Fatalf("declared totals are kept in canonical form: %s", got)
	}
	mock := stmt("m1", "s8", 12)
	mock.Mock = true
	h.open(mock)
	if _, occupant, err := h.r.Open(stmt("real", "s8", 12)); err != ErrConflict || occupant != "m1" {
		t.Fatalf("a mock occupies its statement: %q %v", occupant, err)
	}
	if err := h.r.Void("ws1", "m1", WrongReport); err != nil {
		t.Fatal(err)
	}
	h.open(stmt("real", "s8", 12))
}

func TestSupersessionIsAFactAboutStatements(t *testing.T) {
	h := newHarness(t, nil)
	h.open(stmt("a", "s-a", 10))
	elsewhere := stmt("x", "s-x", 10)
	elsewhere.WorkspaceID = "ws2"
	h.r.Open(elsewhere)
	foreign := stmt("y", "s-y", 10)
	foreign.Source = "hawk-2"
	h.open(foreign)
	for name, earlier := range map[string]string{"unknown": "s-none", "of another workspace": "s-x", "of another source": "s-y"} {
		before := h.r.dump()
		_, _, err := h.r.Open(superseding("b", "s-b", 10, earlier))
		wantErr(t, err, ErrRejected, "a named statement "+name)
		h.unchanged(before, name)
	}
	moved := superseding("b", "s-b", 11, "s-a")
	h.open(moved)
	if a := h.view("a"); a.SupersededBy != "s-b" || h.verdict("a") != "not_eligible[not_closed superseded]" {
		t.Fatalf("a correction may move the period, and supersedes from the moment it is opened: %+v %s", a, h.verdict("a"))
	}
	before := h.r.dump()
	_, _, err := h.r.Open(superseding("c", "s-c", 10, "s-a"))
	wantErr(t, err, ErrRejected, "a statement already superseded by another")
	for name, line := range map[string]func() error{
		"a new line":    func() error { _, err := h.r.Post("ws1", "a", team("t1", "1")); return err },
		"a first close": func() error { _, err := h.r.Close("ws1", "a"); return err },
		"its lines":     func() error { _, err := h.r.Lines("ws1", "a", 0, 1); return err },
	} {
		wantErr(t, line(), ErrRejected, name+" of a reception superseded while open")
	}
	if state, _, err := h.r.Open(stmt("a", "s-a", 10)); err != nil || state != Open {
		t.Fatal(state, err)
	}
	h.unchanged(before, "a superseded reception")
	if err := h.r.Void("ws1", "b", WrongReport); err != nil {
		t.Fatal(err)
	}
	if h.view("a").SupersededBy != "s-b" || h.verdict("a") != "not_eligible[not_closed superseded]" {
		t.Fatal("voiding the later reception brought the earlier statement back")
	}
	if err := h.r.Void("ws1", "a", ReissuedAtSource); err != nil {
		t.Fatal(err)
	}
	h.open(stmt("a2", "s-a", 10))
	if h.view("a2").SupersededBy != "s-b" || h.verdict("a2") != "not_eligible[not_closed superseded]" {
		t.Fatal("a later reception of a superseded statement is superseded too")
	}
	h.open(stmt("free", "s-free", 7))
	for name, spec := range map[string]Spec{
		"naming another statement": superseding("b2", "s-b", 11, "s-free"),
		"naming none":              stmt("b2", "s-b", 11),
		"as a mock":                func() Spec { s := stmt("b2", "s-b", 11); s.Mock = true; return s }(),
	} {
		before := h.r.dump()
		_, _, err := h.r.Open(spec)
		wantErr(t, err, ErrRejected, "the second reception of a superseding statement "+name)
		h.unchanged(before, name)
	}
	h.full(superseding("b2", "s-b", 11, "s-a"))
	if got := h.verdict("b2"); got != "eligible[]" || h.view("b2").SupersededBy != "" {
		t.Fatalf("the same reference again writes no new fact: %s", got)
	}
	h.open(superseding("c", "s-c", 12, "s-b"))
	if got := h.verdict("b2"); got != "not_eligible[superseded]" {
		t.Fatal(got)
	}
	_, _, err = h.r.Open(superseding("d", "s-d", 12, "s-a"))
	wantErr(t, err, ErrRejected, "the first of a chain is still superseded by the second")
	if err := h.r.Void("ws1", "a2", ReissuedAtSource); err != nil {
		t.Fatal(err)
	}
	before = h.r.dump()
	_, _, err = h.r.Open(superseding("a3", "s-a", 10, "s-c"))
	wantErr(t, err, ErrRejected, "a circle of three statements")
	_, _, err = h.r.Open(superseding("a3", "s-a", 10, "s-b"))
	wantErr(t, err, ErrRejected, "a statement already superseded named again by the one it superseded")
	h.unchanged(before, "circles")
	late := stmt("z", "s-z", 9)
	h.full(late)
	h.open(superseding("a3", "s-a", 10, "s-z"))
	if h.verdict("z") != "not_eligible[superseded]" || h.view("a3").SupersededBy != "s-b" {
		t.Fatal("a reception of a superseded statement may itself name a supersession")
	}
	only := stmt("m", "s-m", 8)
	only.Mock = true
	h.open(only)
	h.r.Void("ws1", "m", WrongReport)
	h.open(superseding("n", "s-n", 8, "s-m"))
	if !reflect.DeepEqual(h.view("n").Spec.Supersedes, "s-m") {
		t.Fatal("a statement whose only reception was a mock may be named")
	}
}

func TestTwoStatementsCannotSupersedeEachOther(t *testing.T) {
	h := newHarness(t, nil)
	h.open(stmt("a", "s-a", 10))
	h.open(superseding("b", "s-b", 11, "s-a"))
	if err := h.r.Void("ws1", "a", ReissuedAtSource); err != nil {
		t.Fatal(err)
	}
	before := h.r.dump()
	_, _, err := h.r.Open(superseding("a2", "s-a", 10, "s-b"))
	wantErr(t, err, ErrRejected, "the superseded statement naming the one that superseded it")
	h.unchanged(before, "a circle of two")
}
