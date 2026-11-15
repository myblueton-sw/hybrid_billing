package overview

import (
	"strconv"
	"testing"
	"time"
)

func TestAsOfKeepsListDetailAndTotalsTogether(t *testing.T) {
	f := open(t, testLimits())
	a := subjectOf("acme", "k1")
	f.ingest(t, fact("f1", a, "billing-a", 1, StageRated, map[string]string{MeaningSales: "100", MeaningCost: "70"}, start.Add(-time.Hour)))
	first := must(f.r.Query("ws", Query{Limit: 10}, human("admin")))
	f.clock.advance(time.Minute)
	f.ingest(t, fact("f2", a, "billing-a", 2, StageIssued, map[string]string{MeaningSales: "120", MeaningCost: "70"}, start))
	later := must(f.r.Query("ws", Query{Limit: 10}, human("admin")))
	old := must(f.r.Query("ws", Query{AsOf: first.AsOf, Limit: 10}, human("admin")))
	if old.Rows[0].Stage != StageRated || old.Rows[0].Amounts[MeaningSales] != "100" || later.Rows[0].Stage != StageIssued || !old.Evaluated.Equal(first.Evaluated) {
		t.Fatalf("as-of: old %+v later %+v", old.Rows, later.Rows)
	}
	if old.Totals[1].Meaning != MeaningSales || old.Totals[1].Current != "100" || later.Totals[1].Current != "120" || old.Totals[0].Meaning != MeaningCost {
		t.Fatalf("totals: %+v %+v", old.Totals, later.Totals)
	}
	d := must(f.r.Detail("ws", a, first.AsOf, human("admin")))
	if d.Row.Stage != StageRated || len(d.Timeline) != 1 || d.AsOf != first.AsOf {
		t.Fatalf("detail at the old as-of: %+v", d)
	}
	d = must(f.r.Detail("ws", a, "", human("admin")))
	if len(d.Timeline) != 2 || !d.Timeline[0].Superseded || d.Timeline[1].Superseded {
		t.Fatalf("detail now: %+v", d.Timeline)
	}
}

func TestLateLowerRevisionsNeverRollBack(t *testing.T) {
	f := open(t, testLimits())
	a := subjectOf("acme", "k1")
	f.ingest(t, fact("f2", a, "src", 2, StageApproved, map[string]string{MeaningSales: "50"}, start))
	f.ingest(t, fact("f1", a, "src", 1, StageCollected, map[string]string{MeaningSales: "10"}, start))
	rows := f.all(t, "admin", Query{})
	if rows[0].Stage != StageApproved || rows[0].Amounts[MeaningSales] != "50" {
		t.Fatal(rows)
	}
	d := must(f.r.Detail("ws", a, "", human("admin")))
	if !d.Timeline[1].Superseded || d.Timeline[0].Superseded {
		t.Fatal(d.Timeline)
	}
}

func TestSourcesCombineWithBlockedPrecedenceAndOwnedMeanings(t *testing.T) {
	f := open(t, testLimits())
	a := subjectOf("acme", "k1")
	f.ingest(t, fact("b1", a, "billing", 1, StageBlocked, map[string]string{MeaningSales: "100"}, start.Add(-2*time.Hour)))
	f.ingest(t, fact("r1", a, "receivables", 1, StageIssued, map[string]string{MeaningOutstanding: "100"}, start.Add(-time.Hour)))
	rows := f.all(t, "admin", Query{})
	if rows[0].Stage != StageBlocked || rows[0].Amounts[MeaningSales] != "100" || rows[0].Amounts[MeaningOutstanding] != "100" {
		t.Fatalf("combined: %+v", rows[0])
	}
	expect(t, "owned", second(f.r.Ingest("ws", fact("r2", a, "receivables", 2, StageIssued, map[string]string{MeaningSales: "90"}, start), system("feed"))), ReasonMeaningOwned)
	f.ingest(t, fact("b2", a, "billing", 2, StageApproved, map[string]string{MeaningSales: "100"}, start.Add(-30*time.Minute)))
	rows = f.all(t, "admin", Query{})
	if rows[0].Stage != StageApproved {
		t.Fatalf("latest occurrence leads once nothing blocks: %+v", rows[0])
	}
	b := subjectOf("acme", "k2")
	f.ingest(t, fact("b3", b, "billing", 3, StageApproved, map[string]string{MeaningSales: "100"}, start))
	rows = f.all(t, "admin", Query{})
	if len(rows) != 2 || rows[0].Amounts[MeaningSales] != "" && rows[0].Subject == a {
		t.Fatalf("a source correction moves out of the old subject: %+v", rows)
	}
	for _, row := range rows {
		if row.Subject == a {
			if _, ok := row.Amounts[MeaningSales]; ok {
				t.Fatal("the moved source still reports in the old subject")
			}
		}
	}
	f.ingest(t, fact("r3", a, "receivables", 3, StageIssued, map[string]string{MeaningSales: "1", MeaningOutstanding: "100"}, start))
	f.ingest(t, fact("l2", subjectOf("acme", "k3"), "late", 2, StageRated, map[string]string{MeaningAllocation: "1"}, start))
	f.ingest(t, fact("l1", a, "late", 1, StageBlocked, map[string]string{MeaningSales: "999"}, start))
	if row := f.all(t, "admin", Query{Filter: Filter{Contracts: []string{"k1"}}})[0]; row.Amounts[MeaningSales] != "1" || row.Stage == StageBlocked {
		t.Fatalf("a late lower revision keeps its history without becoming current: %+v", row)
	}
}

func TestStaleUnknownAndZeroStayApart(t *testing.T) {
	f := open(t, testLimits())
	f.ingest(t, fact("a", subjectOf("acme", "k1"), "s1", 1, StageRated, map[string]string{MeaningSales: "0"}, start))
	f.ingest(t, fact("b", subjectOf("beta", "k1"), "s2", 1, StageRated, map[string]string{}, start))
	f.clock.advance(2 * time.Hour)
	f.ingest(t, fact("c", subjectOf("gamma", "k1"), "s3", 1, StageRated, map[string]string{MeaningSales: "5"}, f.clock.read()))
	p := must(f.r.Query("ws", Query{Limit: 10, Sort: Sort{Key: SortCustomer}}, human("admin")))
	if p.Rows[0].Status != StatusStale || p.Rows[1].Status != StatusStale || p.Rows[2].Status != StatusCurrent || p.Stale != 2 {
		t.Fatalf("stale: %+v", p.Rows)
	}
	var sales Total
	for _, total := range p.Totals {
		if total.Meaning == MeaningSales {
			sales = total
		}
	}
	if sales.Current != "5" || sales.Stale != "0" || sales.Unknown != 1 || sales.Rows != 3 || sales.StaleRows != 2 {
		t.Fatalf("sales total: %+v", sales)
	}
	for _, total := range p.Totals {
		if total.Meaning == MeaningAllocation && (total.Current != "" || total.Stale != "" || total.Unknown != 3) {
			t.Fatalf("unknown is not zero: %+v", total)
		}
	}
	f.clock.advance(2 * time.Hour)
	if q := must(f.r.Query("ws", Query{Limit: 10}, human("admin"))); q.Stale != 3 {
		t.Fatal("staleness follows the clock when nothing arrives")
	}
	if q := must(f.r.Query("ws", Query{AsOf: p.AsOf, Limit: 10}, human("admin"))); q.Stale != 2 {
		t.Fatal("an as-of keeps its evaluation instant")
	}
}

func TestScopeLeaksNothing(t *testing.T) {
	f := open(t, testLimits())
	f.ingest(t, fact("a", subjectOf("acme", "k1"), "s1", 1, StageRated, map[string]string{MeaningSales: "10", MeaningCost: "7"}, start))
	f.ingest(t, fact("b", Subject{Customer: "acme", Contract: "k2", Entity: "us", Period: "2026-11", Currency: "USD"}, "s2", 1, StageRated, map[string]string{MeaningSales: "20"}, start))
	f.ingest(t, fact("c", subjectOf("beta", "k1"), "s3", 1, StageRated, map[string]string{MeaningSales: "30"}, start))
	f.scope.set("clerk", Visibility{Selectors: []Selector{{Customer: "acme", Entity: "kr"}}})
	p := must(f.r.Query("ws", Query{Limit: 10}, human("clerk")))
	if p.Count != 1 || len(p.Rows) != 1 || p.Rows[0].Subject.Customer != "acme" || p.Rows[0].Subject.Entity != "kr" {
		t.Fatalf("scope: %+v", p)
	}
	if _, ok := p.Rows[0].Amounts[MeaningCost]; ok {
		t.Fatal("cost without permission")
	}
	for _, total := range p.Totals {
		if total.Meaning == MeaningCost || total.Currency != "KRW" {
			t.Fatalf("totals leak: %+v", total)
		}
	}
	expect(t, "hidden detail", second(f.r.Detail("ws", subjectOf("beta", "k1"), "", human("clerk"))), "not_found")
	expect(t, "missing detail", second(f.r.Detail("ws", subjectOf("zeta", "k1"), "", human("clerk"))), "not_found")
	d := must(f.r.Detail("ws", subjectOf("acme", "k1"), "", human("clerk")))
	if _, ok := d.Timeline[0].Fact.Amounts[MeaningCost]; ok {
		t.Fatal("cost in the timeline")
	}
	f.scope.set("period", Visibility{Selectors: []Selector{{From: "2026-11", To: "2026-11"}}})
	if rows := f.all(t, "period", Query{}); len(rows) != 1 || rows[0].Subject.Period != "2026-11" {
		t.Fatalf("period range: %+v", rows)
	}
	f.scope.set("range", Visibility{Selectors: []Selector{{From: "2026-10", To: "2026-10"}, {Contract: "zz"}}})
	if rows := f.all(t, "range", Query{}); len(rows) != 2 {
		t.Fatalf("both edge months included: %+v", rows)
	}
	f.scope.set("before", Visibility{Selectors: []Selector{{To: "2026-09"}}})
	if rows := f.all(t, "before", Query{}); len(rows) != 0 {
		t.Fatal(rows)
	}
	f.scope.set("wide", Visibility{All: true})
	q := must(f.r.Query("ws", Query{Limit: 1}, human("wide")))
	f.scope.set("wide", Visibility{Selectors: []Selector{{Customer: "acme"}}})
	expect(t, "scope changed between pages", second(f.r.Query("ws", Query{Limit: 1, Cursor: q.Next}, human("wide"))), ReasonCursorMismatch)
}

func TestCursorsPageEverythingOnce(t *testing.T) {
	f := open(t, testLimits())
	for index := 0; index < 250; index++ {
		s := subjectOf("c"+strconv.Itoa(index%7), "k"+strconv.Itoa(index))
		due := ""
		if index%3 != 0 {
			due = "2026-11-" + strconv.Itoa(10+index%20)
		}
		x := fact("f"+strconv.Itoa(index), s, "src"+strconv.Itoa(index), 1, StageRated, map[string]string{MeaningSales: strconv.Itoa(index)}, start)
		x.Due = due
		f.ingest(t, x)
		f.clock.advance(time.Second)
	}
	for _, o := range []Sort{{}, {Key: SortDue}, {Key: SortDue, Descending: true}, {Key: SortReceived, Descending: true}, {Key: SortPeriod}, {Key: SortCustomer, Descending: true}} {
		rows := f.all(t, "admin", Query{Sort: o})
		seen := map[Subject]bool{}
		for index, row := range rows {
			if seen[row.Subject] {
				t.Fatalf("%+v: duplicate", o)
			}
			seen[row.Subject] = true
			if o.Key == SortDue && index > 0 && rows[index-1].Due == "" && row.Due != "" {
				t.Fatalf("%+v: rows without a due date come last", o)
			}
		}
		if len(rows) != 250 {
			t.Fatalf("%+v: %d rows", o, len(rows))
		}
	}
	p := must(f.r.Query("ws", Query{Limit: 100, Sort: Sort{Key: SortDue}}, human("admin")))
	expect(t, "other sort", second(f.r.Query("ws", Query{Limit: 100, Cursor: p.Next}, human("admin"))), ReasonCursorMismatch)
	expect(t, "other viewer", second(f.r.Query("ws", Query{Limit: 100, Cursor: p.Next, Sort: Sort{Key: SortDue}}, human("job"))), ReasonCursorMismatch)
	expect(t, "other as-of", second(f.r.Query("ws", Query{Limit: 100, Cursor: p.Next, Sort: Sort{Key: SortDue}, AsOf: must(f.r.Query("ws", Query{Limit: 1}, human("admin"))).AsOf}, human("admin"))), ReasonCursorMismatch)
	if got := must(f.r.Query("ws", Query{Limit: 100, Cursor: p.Next, Sort: Sort{Key: SortDue}, AsOf: p.AsOf}, human("admin"))); len(got.Rows) != 100 {
		t.Fatal("matching as-of with a cursor")
	}
	recent := must(f.r.Query("ws", Query{Limit: 1, Sort: Sort{Key: SortReceived, Descending: true}}, human("admin")))
	if recent.Rows[0].Subject.Contract != "k249" {
		t.Fatal("recent work comes first")
	}
}

func TestSharedFiltersRunInTheViewersScope(t *testing.T) {
	f := open(t, testLimits())
	f.ingest(t, fact("a", subjectOf("acme", "k1"), "s1", 1, StageBlocked, map[string]string{MeaningSales: "10"}, start))
	f.ingest(t, fact("b", subjectOf("beta", "k1"), "s2", 1, StageBlocked, map[string]string{MeaningSales: "20"}, start))
	expect(t, "shared identifiers", second(f.r.Save("ws", Saved{ID: "mine", Filter: Filter{Customers: []string{"acme"}}, Shared: true}, human("admin"))), ReasonSharedIDs)
	must(f.r.Save("ws", Saved{ID: "mine", Filter: Filter{Customers: []string{"acme"}}}, human("admin")))
	must(f.r.Save("ws", Saved{ID: "blocked", Filter: Filter{Stages: []string{StageBlocked}}, Sort: Sort{Key: SortCustomer}, Shared: true}, human("admin")))
	f.scope.set("clerk", Visibility{Selectors: []Selector{{Customer: "beta"}}})
	list := must(f.r.Filters("ws", human("clerk")))
	if len(list) != 1 || list[0].Saved.ID != "blocked" {
		t.Fatalf("visible filters: %+v", list)
	}
	rows := f.all(t, "clerk", Query{FilterOwner: "admin", FilterID: "blocked"})
	if len(rows) != 1 || rows[0].Subject.Customer != "beta" {
		t.Fatalf("shared filter in the viewer scope: %+v", rows)
	}
	expect(t, "private filter", second(f.r.Query("ws", Query{FilterOwner: "admin", FilterID: "mine", Limit: 1}, human("clerk"))), "not_found")
	if len(f.all(t, "admin", Query{FilterOwner: "admin", FilterID: "mine"})) != 1 {
		t.Fatal("own private filter")
	}
	p := must(f.r.Query("ws", Query{FilterOwner: "admin", FilterID: "blocked", Limit: 1}, human("admin")))
	must(f.r.Save("ws", Saved{ID: "blocked", Filter: Filter{Stages: []string{StageRated}}, Shared: true}, human("admin")))
	expect(t, "edited filter", second(f.r.Query("ws", Query{FilterOwner: "admin", FilterID: "blocked", Limit: 1, Cursor: p.Next}, human("admin"))), ReasonCursorMismatch)
}

func TestAbsentSubjectsAfterAMove(t *testing.T) {
	f := open(t, testLimits())
	a, b := subjectOf("acme", "k1"), subjectOf("acme", "k2")
	f.ingest(t, fact("f1", a, "src", 1, StageRated, map[string]string{MeaningSales: "10"}, start))
	f.ingest(t, fact("f2", b, "src", 2, StageRated, map[string]string{MeaningSales: "10"}, start))
	rows := f.all(t, "admin", Query{})
	if len(rows) != 1 || rows[0].Subject != b {
		t.Fatalf("moved: %+v", rows)
	}
	d := must(f.r.Detail("ws", a, "", human("admin")))
	if d.Row.Status != StatusAbsent || len(d.Timeline) != 1 || !d.Timeline[0].Superseded {
		t.Fatalf("absent detail: %+v", d)
	}
	if bd := must(f.r.Detail("ws", b, "", human("admin"))); len(bd.Timeline) != 1 {
		t.Fatal("a timeline holds only its own facts")
	}
}

func TestOwnershipFollowsCurrentFacts(t *testing.T) {
	f := open(t, testLimits())
	a, b := subjectOf("acme", "k1"), subjectOf("acme", "k2")
	f.ingest(t, fact("o1", a, "owner", 1, StageRated, map[string]string{MeaningSales: "10"}, start))
	f.ingest(t, fact("m1", b, "mover", 1, StageRated, map[string]string{MeaningSales: "20"}, start))
	expect(t, "move into an owned subject", second(f.r.Ingest("ws", fact("m2", a, "mover", 2, StageRated, map[string]string{MeaningSales: "20"}, start), system("feed"))), ReasonMeaningOwned)
	f.ingest(t, fact("o2", a, "owner", 2, StageRated, map[string]string{MeaningOutstanding: "10"}, start))
	f.ingest(t, fact("m3", a, "mover", 3, StageRated, map[string]string{MeaningSales: "20"}, start))
	f.ingest(t, fact("o3", b, "owner", 3, StageRated, map[string]string{MeaningCost: "1"}, start))
	expect(t, "owner left but mover reports sales", second(f.r.Ingest("ws", fact("o4", a, "owner", 4, StageRated, map[string]string{MeaningSales: "1"}, start), system("feed"))), ReasonMeaningOwned)
	f.ingest(t, fact("m4", b, "mover", 4, StageRated, map[string]string{MeaningAllocation: "1"}, start))
	f.ingest(t, fact("o5", a, "owner", 5, StageRated, map[string]string{MeaningSales: "1"}, start))
	if row := f.all(t, "admin", Query{Filter: Filter{Contracts: []string{"k1"}}})[0]; row.Amounts[MeaningSales] != "1" {
		t.Fatal(row)
	}
}

func TestTokensAndSelectorsLeakNothing(t *testing.T) {
	f := open(t, testLimits())
	empty := must(f.r.Query("ws", Query{Limit: 1}, human("admin")))
	for index := 0; index < 120; index++ {
		f.ingest(t, fact("f"+strconv.Itoa(index), subjectOf("acme", "k"+strconv.Itoa(index)), "s"+strconv.Itoa(index), 1, StageRated, nil, start))
	}
	f.clock.advance(1234567 * time.Nanosecond)
	full := must(f.r.Query("ws", Query{Limit: 1}, human("admin")))
	if len(empty.AsOf) != len(full.AsOf) {
		t.Fatal("token length reveals activity")
	}
	f.scope.set("blank", Visibility{Selectors: []Selector{{}}})
	if p := must(f.r.Query("ws", Query{Limit: 1}, human("blank"))); p.Count != 0 {
		t.Fatal("an empty selector must match nothing")
	}
	must(f.r.Save("ws", Saved{ID: "private"}, human("twin")))
	f.scope.set("twin", Visibility{All: true})
	if list := must(f.r.Filters("ws", system("twin"))); len(list) != 0 {
		t.Fatal("a system actor with the same identifier sees a human's filter")
	}
	expect(t, "system twin", second(f.r.Query("ws", Query{FilterOwner: "twin", FilterID: "private", Limit: 1}, system("twin"))), "not_found")
}

func TestEmptyWorkspaceTokenStaysEmpty(t *testing.T) {
	f := open(t, testLimits())
	empty := must(f.r.Query("ws", Query{Limit: 1}, human("admin")))
	x := fact("a", subjectOf("acme", "k1"), "s", 1, StageRated, nil, start)
	x.Evidence = make([]string, maxEvidence)
	for index := range x.Evidence {
		x.Evidence[index] = "doc-" + strconv.Itoa(index)
	}
	f.ingest(t, x)
	if p := must(f.r.Query("ws", Query{AsOf: empty.AsOf, Limit: 1}, human("admin"))); p.Count != 0 {
		t.Fatal("an as-of taken before any fact must stay empty")
	}
}
