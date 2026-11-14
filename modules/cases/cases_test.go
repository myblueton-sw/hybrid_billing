package cases

import (
	"testing"
	"time"
)

func TestInquiryConversationAndDueHistory(t *testing.T) {
	r, clock := openWith(t, testLimits())
	me := cust("u1", "c1")
	v := must(r.OpenCase("ws", inquiry("q1", "c1"), me))
	if v.State != CaseOpen || v.OpenedBy != "u1" || v.OpenedSide != SideCustomer || !v.Opened.Equal(start) || v.Digest != "" || v.ResponseOverdue {
		t.Fatalf("opened: %+v", v)
	}
	if again := must(r.OpenCase("ws", inquiry("q1", "c1"), me)); again.Opened != v.Opened {
		t.Error("an identical open is not a repeat")
	}
	other := inquiry("q1", "c1")
	other.Topic = "tax"
	expect(t, "conflict", second(r.OpenCase("ws", other, me)), "other content")
	expect(t, "conflict", second(r.OpenCase("ws", inquiry("q1", "c1"), cust("u2", "c1"))), "another opener")
	expect(t, "conflict", second(r.OpenCase("ws", inquiry("q1", "c1"), op("u1"))), "the same id from the other side")

	staff := op("agent")
	must(r.AssignCase("ws", "c1", "q1", staff, CaseStep{Step: "a1", Owner: "ana"}))
	due := start.Add(48 * time.Hour)
	expect(t, "rejected", second(r.SetDue("ws", "c1", "q1", staff, CaseStep{Step: "d0", Due: due, Reason: "escalation"})), "a first due time that is not initial")
	expect(t, "rejected", second(r.SetDue("ws", "c1", "q1", staff, CaseStep{Step: "d0", Due: start, Reason: "initial"})), "due now")
	expect(t, "rejected", second(r.SetDue("ws", "c1", "q1", staff, CaseStep{Step: "d0", Due: start.Add(366*24*time.Hour + time.Second), Reason: "initial"})), "due too far")
	must(r.SetDue("ws", "c1", "q1", staff, CaseStep{Step: "d1", Due: start.Add(366 * 24 * time.Hour), Reason: "initial"}))
	expect(t, "rejected", second(r.SetDue("ws", "c1", "q1", staff, CaseStep{Step: "d2", Due: due, Reason: "initial"})), "initial twice")
	in := time.FixedZone("x", 9*3600)
	v = must(r.SetDue("ws", "c1", "q1", staff, CaseStep{Step: "d2", Due: due.In(in), Reason: "customer_request"}))
	if !v.Due.Equal(due) || v.Due.Location() != time.UTC || !v.OriginalDue.Equal(start.Add(366*24*time.Hour)) || len(v.DueHistory) != 2 || v.Owner != "ana" {
		t.Fatalf("due: %+v", v)
	}
	if again := must(r.SetDue("ws", "c1", "q1", staff, CaseStep{Step: "d2", Due: due, Reason: "customer_request"})); len(again.DueHistory) != 2 {
		t.Error("a repeated step moved again")
	}
	expect(t, "conflict", second(r.SetDue("ws", "c1", "q1", staff, CaseStep{Step: "d2", Due: due.Add(time.Hour), Reason: "customer_request"})), "the same step with other content")
	expect(t, "conflict", second(r.AssignCase("ws", "c1", "q1", staff, CaseStep{Step: "d2", Owner: "x"})), "the same step of another kind")
	expect(t, "conflict", second(r.SetDue("ws", "c1", "q1", op("other"), CaseStep{Step: "d2", Due: due, Reason: "customer_request"})), "the same step by another actor")

	clock.set(due)
	v = must(r.Case("ws", "c1", "q1", me))
	if !v.ResponseOverdue || v.OriginalDue != (time.Time{}) || v.DueHistory != nil || v.Owner != "" {
		t.Fatalf("customer view at the due time: %+v", v)
	}
	must(r.Note("ws", "c1", "q1", staff, Entry{ID: "n1", Body: digest('1')}))
	v = must(r.Answer("ws", "c1", "q1", staff, Entry{ID: "e1", Body: digest('2')}))
	if v.State != CaseAnswered || v.ResponseOverdue {
		t.Fatalf("answered: %+v", v)
	}
	seen := must(r.Case("ws", "c1", "q1", me))
	if len(seen.Entries) != 1 || seen.Entries[0].Body != digest('2') || seen.Entries[0].Actor != "" || seen.Entries[0].Digest != "" || seen.Entries[0].Side != SideOperator {
		t.Fatalf("the customer reads the answer: %+v", seen.Entries)
	}
	staffView := must(r.Case("ws", "c1", "q1", staff))
	if len(staffView.Entries) != 2 || staffView.Entries[0].Kind != EntryNote || len(staffView.Entries[1].Digest) != 64 || staffView.Digest == "" {
		t.Fatalf("operator view: %+v", staffView)
	}
	clock.advance(time.Hour)
	v = must(r.Reply("ws", "c1", "q1", me, Entry{ID: "e2", Body: digest('3')}))
	if v.State != CaseOpen || !v.Due.IsZero() || v.ResponseOverdue {
		t.Fatalf("a message after an answer: %+v", v)
	}
	must(r.Answer("ws", "c1", "q1", staff, Entry{ID: "e3", Body: digest('4')}))
	v = must(r.Reply("ws", "c1", "q1", me, Entry{ID: "e2", Body: digest('3')}))
	if v.State != CaseAnswered {
		t.Fatal("a retried reply moved the case")
	}
	expect(t, "conflict", second(r.Reply("ws", "c1", "q1", me, Entry{ID: "e2", Body: digest('5')})), "an entry with other content")
	expect(t, "conflict", second(r.Reply("ws", "c1", "q1", me, Entry{ID: "e1", Body: digest('2')})), "a customer reuses an answer id")
	expect(t, "ok", second(r.Reply("ws", "c1", "q1", me, Entry{ID: "n1", Body: digest('6')})), "a customer cannot collide with a note")
	expect(t, "rejected", second(r.Note("ws", "c1", "q1", me, Entry{ID: "n9", Body: digest('7')})), "a customer note")
	expect(t, "rejected", second(r.Answer("ws", "c1", "q1", me, Entry{ID: "e9", Body: digest('7')})), "a customer answer")
	expect(t, "rejected", second(r.Reply("ws", "c1", "q1", staff, Entry{ID: "e9", Body: digest('7')})), "an operator message")
	staffView = must(r.Case("ws", "c1", "q1", staff))
	full := staffView.DueHistory
	if len(full) != 3 || full[2].Reason != "cleared_on_message" || !full[2].Due.IsZero() || staffView.Due != (time.Time{}) {
		t.Fatalf("due history: %+v", full)
	}
	must(r.SetDue("ws", "c1", "q1", staff, CaseStep{Step: "d3", Due: clock.read().Add(time.Hour), Reason: "follow_up"}))
}

func TestCaseStateTable(t *testing.T) {
	type call func(r *Registry, step string) error
	staff, me := op("agent"), cust("u", "c1")
	entry := func(id string) Entry { return Entry{ID: id, Body: digest('e')} }
	calls := map[string]call{
		"reply":  func(r *Registry, s string) error { return second(r.Reply("ws", "c1", "q", me, entry(s))) },
		"answer": func(r *Registry, s string) error { return second(r.Answer("ws", "c1", "q", staff, entry(s))) },
		"note":   func(r *Registry, s string) error { return second(r.Note("ws", "c1", "q", staff, entry(s))) },
		"assign": func(r *Registry, s string) error {
			return second(r.AssignCase("ws", "c1", "q", staff, CaseStep{Step: s, Owner: "o"}))
		},
		"due": func(r *Registry, s string) error {
			return second(r.SetDue("ws", "c1", "q", staff, CaseStep{Step: s, Due: start.Add(time.Hour), Reason: "initial"}))
		},
		"close": func(r *Registry, s string) error {
			return second(r.Close("ws", "c1", "q", staff, CaseStep{Step: s, Closing: Closing{Outcome: OutcomeAnswered}}))
		},
		"reopen":   func(r *Registry, s string) error { return second(r.ReopenCase("ws", "c1", "q", me, CaseStep{Step: s})) },
		"withdraw": func(r *Registry, s string) error { return second(r.Withdraw("ws", "c1", "q", me, CaseStep{Step: s})) },
	}
	reach := map[string]func(r *Registry){
		CaseOpen:     func(r *Registry) {},
		CaseAnswered: func(r *Registry) { must(r.Answer("ws", "c1", "q", staff, entry("pre"))) },
		CaseClosed: func(r *Registry) {
			must(r.Close("ws", "c1", "q", staff, CaseStep{Step: "pre", Closing: Closing{Outcome: OutcomeAnswered}}))
		},
		CaseWithdrawn: func(r *Registry) { must(r.Withdraw("ws", "c1", "q", me, CaseStep{Step: "pre"})) },
	}
	table := map[string]map[string]string{
		"reply":    {CaseOpen: CaseOpen, CaseAnswered: CaseOpen},
		"answer":   {CaseOpen: CaseAnswered, CaseAnswered: CaseAnswered},
		"note":     {CaseOpen: CaseOpen, CaseAnswered: CaseAnswered, CaseClosed: CaseClosed, CaseWithdrawn: CaseWithdrawn},
		"assign":   {CaseOpen: CaseOpen, CaseAnswered: CaseAnswered},
		"due":      {CaseOpen: CaseOpen, CaseAnswered: CaseAnswered},
		"close":    {CaseOpen: CaseClosed, CaseAnswered: CaseClosed},
		"reopen":   {CaseClosed: CaseOpen},
		"withdraw": {CaseOpen: CaseWithdrawn, CaseAnswered: CaseWithdrawn},
	}
	for name, c := range calls {
		for state, prepare := range reach {
			r, _ := openWith(t, testLimits())
			must(r.OpenCase("ws", inquiry("q", "c1"), me))
			prepare(r)
			err := c(r, "x")
			want, allowed := table[name][state]
			if !allowed {
				expect(t, "rejected", err, name+" in "+state)
				continue
			}
			expect(t, "ok", err, name+" in "+state)
			if got := must(r.Case("ws", "c1", "q", staff)).State; got != want {
				t.Errorf("%s in %s: %s, want %s", name, state, got, want)
			}
		}
	}
}

func TestDisputeOutcomesAndReopen(t *testing.T) {
	r, clock := openWith(t, testLimits())
	staff, me := op("agent"), cust("u", "c1")
	must(r.OpenCase("ws", dispute("d", "c1", "inv-7"), me))
	expect(t, "rejected", second(r.Close("ws", "c1", "d", staff, CaseStep{Step: "s", Closing: Closing{Outcome: OutcomeAnswered}})), "a dispute closed as answered")
	expect(t, "invalid", second(r.Close("ws", "c1", "d", staff, CaseStep{Step: "s", Closing: Closing{Outcome: OutcomeUpheld}})), "upheld without remedy")
	expect(t, "invalid", second(r.Close("ws", "c1", "d", staff, CaseStep{Step: "s", Closing: Closing{Outcome: OutcomeNotUpheld, Remedy: Remedy{Kind: "credit_document", Reference: "x"}}})), "not upheld with remedy")
	expect(t, "invalid", second(r.Close("ws", "c1", "d", staff, CaseStep{Step: "s", Closing: Closing{Outcome: "won"}})), "outcome")
	if v := must(r.Disputes("ws", "c1", "inv-7", "", staff)); len(v) != 1 || v[0].Case.ID != "d" {
		t.Fatalf("disputes: %+v", v)
	}
	must(r.SetDue("ws", "c1", "d", staff, CaseStep{Step: "due", Due: start.Add(time.Hour), Reason: "initial"}))
	draft := Closing{Outcome: OutcomeUpheld, Remedy: Remedy{Kind: "adjustment_draft", Reference: "adj-1"}}
	v := must(r.Close("ws", "c1", "d", staff, CaseStep{Step: "c1", Closing: draft}))
	if v.State != CaseClosed || v.Remedy != draft.Remedy || v.ClosedBy != "agent" {
		t.Fatalf("closed: %+v", v)
	}
	if again := must(r.Close("ws", "c1", "d", staff, CaseStep{Step: "c1", Closing: draft})); again.State != CaseClosed {
		t.Fatal("repeat")
	}
	seen := must(r.Case("ws", "c1", "d", me))
	if seen.Outcome != OutcomeUpheld || seen.Remedy != (Remedy{}) || seen.ClosedBy != "" || !seen.Closed.IsZero() || seen.Closings != nil {
		t.Fatalf("the customer sees an internal remedy: %+v", seen)
	}
	if len(must(r.Disputes("ws", "c1", "inv-7", "", staff))) != 0 {
		t.Error("a closed dispute is still listed")
	}
	clock.advance(time.Minute)
	v = must(r.ReopenCase("ws", "c1", "d", me, CaseStep{Step: "r1"}))
	if v.State != CaseOpen || v.Outcome != "" || v.Due != (time.Time{}) {
		t.Fatalf("reopened: %+v", v)
	}
	if again := must(r.ReopenCase("ws", "c1", "d", me, CaseStep{Step: "r1"})); again.State != CaseOpen {
		t.Fatal("a retried reopen")
	}
	if len(must(r.Disputes("ws", "c1", "inv-7", "", staff))) != 1 {
		t.Error("a reopened dispute is not listed")
	}
	credit := Closing{Outcome: OutcomeUpheld, Remedy: Remedy{Kind: "credit_document", Reference: "cd-1"}}
	must(r.Close("ws", "c1", "d", staff, CaseStep{Step: "c2", Closing: credit}))
	if seen := must(r.Case("ws", "c1", "d", me)); seen.Remedy != credit.Remedy {
		t.Errorf("an issued credit document is hidden: %+v", seen.Remedy)
	}
	full := must(r.Case("ws", "c1", "d", staff))
	if len(full.Closings) != 2 || full.Closings[0].Remedy != draft.Remedy || full.Closings[0].Reopener != "u" || !full.Closings[0].Reopened.Equal(start.Add(time.Minute)) || !full.Closings[1].Reopened.IsZero() {
		t.Fatalf("closing history: %+v", full.Closings)
	}
	if h := full.DueHistory; len(h) != 2 || h[1].Reason != "cleared_on_reopen" || h[1].Actor != "u" {
		t.Fatalf("due history: %+v", h)
	}
	must(r.ReopenCase("ws", "c1", "d", staff, CaseStep{Step: "r1"}))
	must(r.Close("ws", "c1", "d", staff, CaseStep{Step: "c3", Closing: Closing{Outcome: OutcomeNotUpheld}}))
	if seen := must(r.Case("ws", "c1", "d", me)); seen.Outcome != OutcomeNotUpheld || seen.Remedy != (Remedy{}) {
		t.Fatalf("not upheld: %+v", seen)
	}
}

func TestCustomerBoundary(t *testing.T) {
	r, _ := openWith(t, testLimits())
	staff, mine, theirs := op("agent"), cust("u1", "c1"), cust("u2", "c2")
	must(r.OpenCase("ws", dispute("d", "c1", "inv"), mine))
	must(r.Note("ws", "c1", "d", staff, Entry{ID: "secret", Body: digest('9')}))
	expect(t, "not_found", second(r.OpenCase("ws", inquiry("x", "c1"), theirs)), "open for another customer")
	expect(t, "not_found", second(r.OpenCase("ws", dispute("d", "c1", "inv"), theirs)), "open over another customer's case")
	expect(t, "ok", second(r.OpenCase("ws", dispute("d", "c2", "inv"), theirs)), "the same identifier for another customer")
	for name, err := range map[string]error{
		"read":     second(r.Case("ws", "c1", "d", theirs)),
		"reply":    second(r.Reply("ws", "c1", "d", theirs, Entry{ID: "e", Body: digest('1')})),
		"note":     second(r.Note("ws", "c1", "d", theirs, Entry{ID: "secret", Body: digest('9')})),
		"answer":   second(r.Answer("ws", "c1", "d", theirs, Entry{ID: "e", Body: digest('1')})),
		"assign":   second(r.AssignCase("ws", "c1", "d", theirs, CaseStep{Step: "s", Owner: "o"})),
		"due":      second(r.SetDue("ws", "c1", "d", theirs, CaseStep{Step: "s", Due: start.Add(time.Hour), Reason: "initial"})),
		"close":    second(r.Close("ws", "c1", "d", theirs, CaseStep{Step: "s", Closing: Closing{Outcome: OutcomeNotUpheld}})),
		"reopen":   second(r.ReopenCase("ws", "c1", "d", theirs, CaseStep{Step: "s"})),
		"withdraw": second(r.Withdraw("ws", "c1", "d", theirs, CaseStep{Step: "s"})),
		"unknown":  second(r.Case("ws", "c1", "zz", theirs)),
		"space":    second(r.Case("other", "c1", "d", mine)),
	} {
		expect(t, "not_found", err, name)
	}
	if v := must(r.Cases("ws", theirs, CaseFilter{})); len(v) != 1 || v[0].Case.Customer != "c2" {
		t.Fatalf("list: %+v", v)
	}
	if v := must(r.Cases("ws", theirs, CaseFilter{Customer: "c1"})); len(v) != 0 {
		t.Fatalf("filtered list: %+v", v)
	}
	if v := must(r.Disputes("ws", "c2", "inv", "", staff)); len(v) != 1 || v[0].Case.Customer != "c2" {
		t.Fatalf("disputes of one customer: %+v", v)
	}
	opened := must(r.OpenCase("ws", inquiry("staff-opened", "c1"), staff))
	if opened.OpenedBy != "agent" {
		t.Fatal(opened.OpenedBy)
	}
	if seen := must(r.Case("ws", "c1", "staff-opened", mine)); seen.OpenedBy != "" || seen.OpenedSide != SideOperator {
		t.Fatalf("the customer sees the operator: %+v", seen)
	}
}

func TestCaseListOrderAndCursor(t *testing.T) {
	small := testLimits()
	small.Page = 3
	r, _ := openWith(t, small)
	staff := op("agent")
	for _, k := range [][2]string{{"c2", "a"}, {"c1", "b"}, {"c1", "a"}, {"c3", "a"}} {
		must(r.OpenCase("ws", inquiry(k[1], k[0]), staff))
	}
	must(r.OpenCase("ws", dispute("z", "c1", "inv-x"), staff))
	keys := func(f CaseFilter) string {
		out := ""
		for _, v := range must(r.Cases("ws", staff, f)) {
			out += v.Case.Customer + "/" + v.Case.ID + " "
		}
		return out
	}
	for _, row := range []struct {
		want string
		f    CaseFilter
	}{
		{"c1/a c1/b c1/z ", CaseFilter{}},
		{"c1/z c2/a c3/a ", CaseFilter{AfterCustomer: "c1", After: "b"}},
		{"c3/a ", CaseFilter{AfterCustomer: "c2", After: "a"}},
		{"c1/a c1/b c1/z ", CaseFilter{AfterCustomer: "c1"}},
		{"c1/z ", CaseFilter{Kind: KindDispute}},
		{"c1/a c1/b c2/a ", CaseFilter{Document: "inv-1"}},
		{"c2/a ", CaseFilter{Customer: "c2"}},
		{"", CaseFilter{State: CaseClosed}},
		{"c1/a c1/b c1/z ", CaseFilter{State: CaseOpen}},
	} {
		if got := keys(row.f); got != row.want {
			t.Errorf("%+v: %q, want %q", row.f, got, row.want)
		}
	}
	expect(t, "invalid", second(r.Cases("ws", staff, CaseFilter{After: "a"})), "a cursor without its customer")
	expect(t, "invalid", second(r.Cases("ws", staff, CaseFilter{State: "x"})), "state")
	expect(t, "invalid", second(r.Cases("ws", staff, CaseFilter{Kind: "x"})), "kind")
	if v := must(r.Cases("none", staff, CaseFilter{})); v == nil || len(v) != 0 {
		t.Error("an unknown workspace")
	}
	if v := must(r.Disputes("none", "c1", "inv", "", staff)); v == nil || len(v) != 0 {
		t.Error("an unknown workspace")
	}
}
