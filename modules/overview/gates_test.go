package overview

import (
	"errors"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRegistryNeedsDependenciesAndLimits(t *testing.T) {
	f := open(t, testLimits())
	for _, c := range []Config{{Random: f.random}, {Scope: f.scope}, {Scope: f.scope, Random: &counter{fail: true}}} {
		if _, err := NewRegistry(c, testLimits(), f.clock.read); !errors.Is(err, ErrInvalid) {
			t.Fatal("config")
		}
	}
	if _, err := NewRegistry(Config{Scope: f.scope, Random: f.random}, testLimits(), nil); !errors.Is(err, ErrInvalid) {
		t.Fatal("clock")
	}
	for _, change := range []func(*Limits){
		func(l *Limits) { l.Workspaces = 0 }, func(l *Limits) { l.Subjects = maxSubjects + 1 }, func(l *Limits) { l.Facts = 0 },
		func(l *Limits) { l.Filters = 0 }, func(l *Limits) { l.Page = maxPage + 1 }, func(l *Limits) { l.Stale = minStale - 1 }, func(l *Limits) { l.Stale = maxStale + 1 },
	} {
		l := testLimits()
		change(&l)
		if _, err := NewRegistry(Config{Scope: f.scope, Random: f.random}, l, f.clock.read); !errors.Is(err, ErrInvalid) {
			t.Fatalf("limits %+v", l)
		}
	}
	for _, l := range []Limits{{maxWorkspaces, maxSubjects, maxFacts, maxFilters, maxPage, maxStale}, {1, 1, 1, 1, 1, minStale}} {
		if _, err := NewRegistry(Config{Scope: f.scope, Random: f.random}, l, f.clock.read); err != nil {
			t.Fatal(err)
		}
	}
	if (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" || !errors.Is(Refusal{Reason: "x"}, ErrRejected) {
		t.Fatal("refusal")
	}
}

func TestFormHelpers(t *testing.T) {
	if !validID(strings.Repeat("Z", maxIDBytes)) || validID(strings.Repeat("z", maxIDBytes+1)) || validID("!a") || validID("a!") || validID("") || !validID("Az09._:-") {
		t.Fatal("identifiers")
	}
	for _, ok := range []string{"0", "1", "-1", "10.5", "-0.000001", strings.Repeat("9", maxDigits)} {
		if !signedCanonical(ok) {
			t.Fatal(ok)
		}
	}
	for _, bad := range []string{"", "-", "-0", "-0.0", "01", "1.", ".5", "1.50", "1.0000001", "+1", "1a", "--1", strings.Repeat("9", maxDigits+1), "-00", "9.:"} {
		if signedCanonical(bad) {
			t.Fatal(bad)
		}
	}
	for _, c := range []string{"AED", "ZAR"} {
		if !validCurrency(c) {
			t.Fatal(c)
		}
	}
	for _, c := range []string{"KR", "krw", "K1W", "@RW", "[RW", "KR[", "KRWW"} {
		if validCurrency(c) {
			t.Fatal(c)
		}
	}
	if validPeriod("2026-13") || validPeriod("2026-1") || !validPeriod("2026-01") || validDate("2026-02-30") || !validDate("2026-02-28") {
		t.Fatal("dates")
	}
	for value, want := range map[int64]string{0: "0", -1: "-0.000001", 1500000: "1.5", -2000000: "-2", 7: "0.000007"} {
		if got := decimal(big.NewInt(value)); got != want {
			t.Fatalf("%d: %s", value, got)
		}
	}
	if micro("-1.5").String() != "-1500000" || micro("2").String() != "2000000" {
		t.Fatal("micro")
	}
	base := fact("f", subjectOf("acme", "k1"), "s", 1, StageRated, map[string]string{MeaningSales: "1"}, start)
	for _, change := range []func(*Fact){
		func(x *Fact) { x.ID = "" }, func(x *Fact) { x.Subject.Customer = "" }, func(x *Fact) { x.Subject.Contract = "" }, func(x *Fact) { x.Subject.Entity = "" },
		func(x *Fact) { x.Subject.Period = "2026" }, func(x *Fact) { x.Subject.Currency = "krw" }, func(x *Fact) { x.Source.ID = "" },
		func(x *Fact) { x.Source.Revision = 0 }, func(x *Fact) { x.Stage = "done" }, func(x *Fact) { x.Due = "2026-02-30" },
		func(x *Fact) { x.Owner = "a b" }, func(x *Fact) { x.Next = "a b" }, func(x *Fact) { x.Occurred = time.Time{} },
		func(x *Fact) { x.Amounts = map[string]string{"profit": "1"} }, func(x *Fact) { x.Amounts = map[string]string{MeaningSales: "1.0"} },
		func(x *Fact) { x.Evidence = []string{"a b"} }, func(x *Fact) { x.Evidence = make([]string, maxEvidence+1) },
	} {
		x := base
		x.Amounts = map[string]string{MeaningSales: "1"}
		change(&x)
		if factForm(x) {
			t.Fatalf("form %+v", x)
		}
	}
	ok := base
	ok.Due, ok.Owner, ok.Next, ok.Evidence = "2026-12-01", "o", "n", []string{"doc-1"}
	if !factForm(ok) || !factForm(base) {
		t.Fatal("valid facts")
	}
	for _, f := range []Filter{{Customers: []string{"a b"}}, {Contracts: []string{""}}, {Entities: []string{"!"}}, {Owners: []string{"a b"}},
		{Periods: []string{"2026"}}, {Currencies: []string{"usd"}}, {Stages: []string{"x"}}, {DueBy: "x"}, {Status: StatusAbsent}} {
		if filterForm(f) {
			t.Fatalf("filter %+v", f)
		}
	}
	if !filterForm(Filter{Status: StatusStale, DueBy: "2026-12-01", Periods: []string{"2026-10"}, Currencies: []string{"KRW"}, Stages: []string{StageRated}}) {
		t.Fatal("valid filter")
	}
	for _, f := range []Filter{{Customers: []string{"a"}}, {Contracts: []string{"a"}}, {Entities: []string{"a"}}, {Periods: []string{"2026-10"}}, {Currencies: []string{"KRW"}},
		{Stages: []string{StageRated}}, {Owners: []string{"o"}}, {DueBy: "2026-10-01"}, {Status: StatusStale}} {
		if empty(f) {
			t.Fatalf("not empty %+v", f)
		}
	}
	if !empty(Filter{}) {
		t.Fatal("empty")
	}
	for _, f := range []Filter{{Customers: []string{"a"}}, {Contracts: []string{"a"}}, {Entities: []string{"a"}}, {Owners: []string{"a"}}} {
		if sharedForm(f) {
			t.Fatalf("shared %+v", f)
		}
	}
}

func TestIngestGates(t *testing.T) {
	f := open(t, testLimits())
	x := fact("f1", subjectOf("acme", "k1"), "s", 1, StageRated, map[string]string{MeaningSales: "1"}, start)
	expect(t, "workspace", second(f.r.Ingest("", x, system("feed"))), "invalid")
	expect(t, "actor", second(f.r.Ingest("ws", x, Actor{ID: "x"})), "invalid")
	expect(t, "form", second(f.r.Ingest("ws", Fact{}, system("feed"))), "invalid")
	expect(t, "assistant", second(f.r.Ingest("ws", x, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "human", second(f.r.Ingest("ws", x, human("clerk"))), ReasonNotHuman)
	future := x
	future.Occurred = start.Add(time.Second)
	expect(t, "future", second(f.r.Ingest("ws", future, system("feed"))), ReasonFutureOccurrence)
	e := f.ingest(t, x)
	if e.Received != start || e.Actor != "feed" || e.Superseded {
		t.Fatal(e)
	}
	if again := must(f.r.Ingest("ws", x, system("other"))); again.Actor != "feed" {
		t.Fatal("identical repeat returns the record")
	}
	for _, change := range []func(*Fact){
		func(y *Fact) { y.Stage = StageIssued }, func(y *Fact) { y.Due = "2026-12-01" }, func(y *Fact) { y.Owner = "o" }, func(y *Fact) { y.Next = "n" },
		func(y *Fact) { y.Amounts = map[string]string{MeaningSales: "2"} }, func(y *Fact) { y.Amounts = map[string]string{MeaningSales: "1", MeaningCost: "1"} },
		func(y *Fact) { y.Amounts = map[string]string{MeaningCost: "1"} },
		func(y *Fact) { y.Evidence = []string{"d"} }, func(y *Fact) { y.Occurred = start.Add(-time.Second) }, func(y *Fact) { y.Subject.Contract = "k9" },
		func(y *Fact) { y.Source.Revision = 2 },
	} {
		y := x
		y.Amounts = map[string]string{MeaningSales: "1"}
		change(&y)
		expect(t, "changed", second(f.r.Ingest("ws", y, system("feed"))), "conflict")
	}
	ev := x
	ev.ID, ev.Source.Revision, ev.Evidence = "f2", 2, []string{"a"}
	f.ingest(t, ev)
	ev2 := ev
	ev2.Evidence = []string{"b"}
	expect(t, "evidence change", second(f.r.Ingest("ws", ev2, system("feed"))), "conflict")
	y := x
	y.ID = "f3"
	expect(t, "revision", second(f.r.Ingest("ws", y, system("feed"))), ReasonRevisionConflict)
	x.Amounts[MeaningSales] = "999"
	if f.all(t, "admin", Query{})[0].Amounts[MeaningSales] != "1" {
		t.Fatal("the register shares the caller's map")
	}
	l := testLimits()
	l.Facts, l.Workspaces = 1, 1
	g := open(t, l)
	g.ingest(t, fact("a", subjectOf("acme", "k1"), "s", 1, StageRated, nil, start))
	expect(t, "facts", second(g.r.Ingest("ws", fact("b", subjectOf("acme", "k1"), "s", 2, StageRated, nil, start), system("feed"))), "limit")
	expect(t, "workspaces", second(g.r.Ingest("ws2", fact("b", subjectOf("acme", "k1"), "s", 1, StageRated, nil, start), system("feed"))), "limit")
	l = testLimits()
	l.Subjects = 1
	h := open(t, l)
	h.ingest(t, fact("a", subjectOf("acme", "k1"), "s", 1, StageRated, nil, start))
	h.ingest(t, fact("b", subjectOf("acme", "k1"), "s", 2, StageRated, nil, start))
	expect(t, "subjects", second(h.r.Ingest("ws", fact("c", subjectOf("acme", "k2"), "t", 1, StageRated, nil, start), system("feed"))), "limit")
}

func TestQueryGates(t *testing.T) {
	f := open(t, testLimits())
	f.ingest(t, fact("a", subjectOf("acme", "k1"), "s", 1, StageRated, map[string]string{MeaningSales: "1"}, start))
	for label, q := range map[string]Query{
		"limit":        {},
		"limit high":   {Limit: 101},
		"filter":       {Limit: 1, Filter: Filter{Stages: []string{"x"}}},
		"sort":         {Limit: 1, Sort: Sort{Key: "amount"}},
		"owner only":   {Limit: 1, FilterOwner: "admin"},
		"id only":      {Limit: 1, FilterID: "x"},
		"bad id":       {Limit: 1, FilterOwner: "admin", FilterID: "a b"},
		"bad owner":    {Limit: 1, FilterOwner: "a b", FilterID: "x"},
		"both filters": {Limit: 1, FilterOwner: "admin", FilterID: "x", Filter: Filter{Stages: []string{StageRated}}},
		"both sorts":   {Limit: 1, FilterOwner: "admin", FilterID: "x", Sort: Sort{Key: SortDue}},
		"token":        {Limit: 1, AsOf: "garbage"},
	} {
		got := second(f.r.Query("ws", q, human("admin")))
		if reason(got) != "invalid" {
			t.Errorf("%s: %v", label, got)
		}
	}
	expect(t, "workspace", second(f.r.Query("", Query{Limit: 1}, human("admin"))), "invalid")
	expect(t, "actor", second(f.r.Query("ws", Query{Limit: 1}, Actor{ID: "admin"})), "invalid")
	expect(t, "assistant", second(f.r.Query("ws", Query{Limit: 1}, assistant("admin"))), ReasonAINotAllowed)
	expect(t, "scope", second(f.r.Query("ws", Query{Limit: 1}, human("nobody"))), ReasonScopeUnavailable)
	expect(t, "saved in empty workspace", second(f.r.Query("ws9", Query{Limit: 1, FilterOwner: "admin", FilterID: "x"}, human("admin"))), "not_found")
	expect(t, "unknown saved", second(f.r.Query("ws", Query{Limit: 1, FilterOwner: "admin", FilterID: "x"}, human("admin"))), "not_found")
	other := must(f.r.Query("ws2", Query{Limit: 1}, human("admin")))
	if other.Count != 0 || len(other.Rows) != 0 || other.AsOf == "" {
		t.Fatal("empty workspace")
	}
	expect(t, "token of another workspace", second(f.r.Query("ws", Query{Limit: 1, AsOf: other.AsOf}, human("admin"))), "invalid")
	if p := must(f.r.Query("ws", Query{Limit: 1}, system("job"))); p.Count != 1 {
		t.Fatal("systems may read")
	}
	f.random.fail = true
	expect(t, "nonce", second(f.r.Query("ws", Query{Limit: 1}, human("admin"))), ReasonRandom)
	f.random.fail = false
	f.ingest(t, fact("b", subjectOf("beta", "k1"), "t", 1, StageRated, nil, start))
	p := must(f.r.Query("ws", Query{Limit: 1}, human("admin")))
	f.random.fail = true
	expect(t, "nonce for the cursor", second(f.r.Query("ws", Query{Limit: 1, AsOf: p.AsOf}, human("admin"))), ReasonRandom)
	f.random.fail = false
	expect(t, "garbage cursor", second(f.r.Query("ws", Query{Limit: 1, Cursor: "!!"}, human("admin"))), ReasonCursorMismatch)
	expect(t, "short cursor", second(f.r.Query("ws", Query{Limit: 1, Cursor: "AA"}, human("admin"))), ReasonCursorMismatch)
	asOfAsCursor := second(f.r.Query("ws", Query{Limit: 1, Cursor: p.AsOf}, human("admin")))
	expect(t, "as-of used as a cursor", asOfAsCursor, ReasonCursorMismatch)
	expect(t, "cursor of another workspace", second(f.r.Query("ws2", Query{Limit: 1, Cursor: p.Next}, human("admin"))), ReasonCursorMismatch)
	expect(t, "cursor of another filter", second(f.r.Query("ws", Query{Limit: 1, Cursor: p.Next, Filter: Filter{Stages: []string{StageRated}}}, human("admin"))), ReasonCursorMismatch)
	expect(t, "cursor of another kind", second(f.r.Query("ws", Query{Limit: 1, Cursor: p.Next}, system("admin"))), ReasonCursorMismatch)
	beyond := must(f.r.Query("ws", Query{Limit: 1, Cursor: p.Next}, human("admin")))
	if len(beyond.Rows) != 1 || beyond.Next != "" {
		t.Fatal(beyond)
	}
	if p := must(f.r.Query("ws", Query{Limit: 5, Filter: Filter{Customers: []string{"zeta"}}}, human("admin"))); len(p.Rows) != 0 || p.Totals != nil {
		t.Fatal("no rows, no totals")
	}
}

func TestFilterMatching(t *testing.T) {
	f := open(t, testLimits())
	x := fact("a", subjectOf("acme", "k1"), "s1", 1, StageRated, map[string]string{MeaningSales: "1"}, start)
	x.Due, x.Owner = "2026-11-20", "ana"
	f.ingest(t, x)
	y := fact("b", Subject{Customer: "beta", Contract: "k2", Entity: "us", Period: "2026-11", Currency: "USD"}, "s2", 1, StageIssued, nil, start)
	f.ingest(t, y)
	for label, c := range map[string]struct {
		filter Filter
		want   int
	}{
		"customers":  {Filter{Customers: []string{"acme"}}, 1},
		"contracts":  {Filter{Contracts: []string{"k2"}}, 1},
		"entities":   {Filter{Entities: []string{"kr", "us"}}, 2},
		"periods":    {Filter{Periods: []string{"2026-11"}}, 1},
		"currencies": {Filter{Currencies: []string{"USD"}}, 1},
		"stages":     {Filter{Stages: []string{StageRated}}, 1},
		"owners":     {Filter{Owners: []string{"ana"}}, 1},
		"due by":     {Filter{DueBy: "2026-11-20"}, 1},
		"due before": {Filter{DueBy: "2026-11-19"}, 0},
		"status":     {Filter{Status: StatusCurrent}, 2},
		"stale":      {Filter{Status: StatusStale}, 0},
		"none":       {Filter{Customers: []string{"zeta"}}, 0},
	} {
		if got := len(f.all(t, "admin", Query{Filter: c.filter})); got != c.want {
			t.Errorf("%s: %d", label, got)
		}
	}
	if !subjectLess(Subject{Customer: "a"}, Subject{Customer: "b"}) || subjectLess(Subject{Customer: "a"}, Subject{Customer: "a"}) ||
		!subjectLess(Subject{Customer: "a", Currency: "A"}, Subject{Customer: "a", Currency: "B"}) {
		t.Fatal("subject order")
	}
}

func TestDetailAndSavedGates(t *testing.T) {
	f := open(t, testLimits())
	a := subjectOf("acme", "k1")
	f.ingest(t, fact("a", a, "s1", 1, StageRated, nil, start))
	expect(t, "workspace", second(f.r.Detail("", a, "", human("admin"))), "invalid")
	expect(t, "subject", second(f.r.Detail("ws", Subject{}, "", human("admin"))), "invalid")
	expect(t, "actor", second(f.r.Detail("ws", a, "", Actor{ID: "x"})), "invalid")
	expect(t, "assistant", second(f.r.Detail("ws", a, "", assistant("admin"))), ReasonAINotAllowed)
	expect(t, "scope", second(f.r.Detail("ws", a, "", human("nobody"))), ReasonScopeUnavailable)
	expect(t, "token", second(f.r.Detail("ws", a, "garbage", human("admin"))), "invalid")
	expect(t, "workspace unknown", second(f.r.Detail("ws9", a, "", human("admin"))), "not_found")
	early := must(f.r.Query("ws", Query{Limit: 1}, human("admin")))
	b := subjectOf("acme", "k2")
	f.ingest(t, fact("b", b, "s2", 1, StageRated, nil, start))
	expect(t, "not yet at that as-of", second(f.r.Detail("ws", b, early.AsOf, human("admin"))), "not_found")
	f.random.fail = true
	expect(t, "nonce", second(f.r.Detail("ws", a, "", human("admin"))), ReasonRandom)
	f.random.fail = false

	good := Saved{ID: "s", Filter: Filter{Stages: []string{StageRated}}}
	expect(t, "save workspace", second(f.r.Save("", good, human("admin"))), "invalid")
	expect(t, "save id", second(f.r.Save("ws", Saved{}, human("admin"))), "invalid")
	expect(t, "save actor", second(f.r.Save("ws", good, Actor{ID: "x"})), "invalid")
	expect(t, "save filter", second(f.r.Save("ws", Saved{ID: "s", Filter: Filter{Stages: []string{"x"}}}, human("admin"))), "invalid")
	expect(t, "save sort", second(f.r.Save("ws", Saved{ID: "s", Sort: Sort{Key: "x"}}, human("admin"))), "invalid")
	expect(t, "save assistant", second(f.r.Save("ws", good, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "save system", second(f.r.Save("ws", good, system("job"))), ReasonNotHuman)
	v := must(f.r.Save("ws", good, human("admin")))
	v.Saved.Filter.Stages[0] = "mutated"
	if must(f.r.Filters("ws", human("admin")))[0].Saved.Filter.Stages[0] != StageRated {
		t.Fatal("a view shares the filter")
	}
	good.Filter.Stages = []string{StageRated}
	must(f.r.Save("ws", Saved{ID: "s", Filter: Filter{Stages: []string{StageIssued}}}, human("other")))
	if list := must(f.r.Filters("ws", human("admin"))); len(list) != 1 {
		t.Fatal("per-owner namespaces")
	}
	for index := 0; index < 2; index++ {
		must(f.r.Save("ws", Saved{ID: "x" + strconv.Itoa(index), Shared: true}, human("admin")))
	}
	if list := must(f.r.Filters("ws", human("third"))); len(list) != 2 || list[0].Saved.ID != "x0" {
		t.Fatalf("shared filters: %+v", list)
	}
	expect(t, "filters limit", second(f.r.Save("ws", Saved{ID: "y"}, human("admin"))), "limit")
	must(f.r.Save("ws", Saved{ID: "s", Filter: Filter{Stages: []string{StageIssued}}}, human("admin")))
	expect(t, "filters form", second(f.r.Filters("", human("admin"))), "invalid")
	expect(t, "filters actor", second(f.r.Filters("ws", Actor{ID: "x"})), "invalid")
	expect(t, "filters assistant", second(f.r.Filters("ws", assistant("bot"))), ReasonAINotAllowed)
	if list := must(f.r.Filters("ws9", human("admin"))); len(list) != 0 {
		t.Fatal("empty workspace filters")
	}
	l := testLimits()
	l.Workspaces = 1
	g := open(t, l)
	g.ingest(t, fact("a", a, "s1", 1, StageRated, nil, start))
	expect(t, "save workspace limit", second(g.r.Save("ws2", good, human("admin"))), "limit")
	list := []SavedView{{Owner: "b", Saved: Saved{ID: "a"}}, {Owner: "a", Saved: Saved{ID: "b"}}, {Owner: "a", Saved: Saved{ID: "a"}}}
	sortSaved(list)
	if list[0].Owner != "a" || list[0].Saved.ID != "a" || list[2].Owner != "b" {
		t.Fatal(list)
	}
}

func TestHundredThousandSubjects(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test")
	}
	l := testLimits()
	l.Page = maxPage
	f := open(t, l)
	expected := map[string]int64{}
	for index := 0; index < 100000; index++ {
		currency := []string{"KRW", "USD"}[index%2]
		s := Subject{Customer: "c" + strconv.Itoa(index%500), Contract: "k" + strconv.Itoa(index), Entity: "kr", Period: "2026-10", Currency: currency}
		amount := strconv.Itoa(index%997) + "." + strconv.Itoa(1+index%9)
		f.ingest(t, fact("f"+strconv.Itoa(index), s, "src"+strconv.Itoa(index), 1, StageRated, map[string]string{MeaningSales: amount}, start))
		expected[currency] += int64(index%997)*1000000 + int64(1+index%9)*100000
	}
	q := Query{Limit: 1000, Sort: Sort{Key: SortCustomer}}
	first := must(f.r.Query("ws", q, human("admin")))
	if first.Count != 100000 {
		t.Fatal(first.Count)
	}
	sum := map[string]*big.Int{"KRW": new(big.Int), "USD": new(big.Int)}
	seen, pages := 0, 0
	unique := map[Subject]bool{}
	page := first
	for {
		pages++
		for _, row := range page.Rows {
			if unique[row.Subject] {
				t.Fatalf("duplicate %+v", row.Subject)
			}
			unique[row.Subject] = true
			sum[row.Subject.Currency].Add(sum[row.Subject.Currency], micro(row.Amounts[MeaningSales]))
			seen++
		}
		if page.Next == "" {
			break
		}
		q.Cursor = page.Next
		page = must(f.r.Query("ws", q, human("admin")))
	}
	if seen != 100000 || pages != 100 {
		t.Fatalf("seen %d pages %d", seen, pages)
	}
	found := 0
	for _, total := range first.Totals {
		if total.Meaning != MeaningSales {
			continue
		}
		found++
		want := strconv.FormatInt(expected[total.Currency]/1000000, 10) + "." + strings.TrimRight(strconv.FormatInt(1000000+expected[total.Currency]%1000000, 10)[1:], "0")
		if total.Current != decimal(sum[total.Currency]) || total.Current != strings.TrimSuffix(want, ".") || total.Rows != 50000 {
			t.Fatalf("totals %+v want %s", total, want)
		}
	}
	if found != 2 {
		t.Fatal("sales totals for both currencies")
	}
}
