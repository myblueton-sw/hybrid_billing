package cases

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRegistryNeedsLimitsAndClock(t *testing.T) {
	if _, err := NewRegistry(testLimits(), nil); kind(err) != "invalid" {
		t.Error("no clock")
	}
	caps := Limits{Workspaces: 100000, Findings: 1000000, Cases: 1000000, CustomerCases: 1000000, Steps: 1000, Messages: 1000, Entries: 1000, Subjects: 100, Page: 1000}
	if _, err := NewRegistry(caps, time.Now); err != nil {
		t.Errorf("caps: %v", err)
	}
	for name, change := range map[string]func(*Limits){
		"workspaces": func(l *Limits) { l.Workspaces++ }, "findings": func(l *Limits) { l.Findings++ }, "cases": func(l *Limits) { l.Cases++ },
		"customer": func(l *Limits) { l.CustomerCases++ }, "steps": func(l *Limits) { l.Steps++ }, "messages": func(l *Limits) { l.Messages++ },
		"entries": func(l *Limits) { l.Entries++ }, "subjects": func(l *Limits) { l.Subjects++ }, "page": func(l *Limits) { l.Page++ },
		"zero": func(l *Limits) { l.Page = 0 }, "customer above cases": func(l *Limits) { l.Cases = 5; l.CustomerCases = 6 },
	} {
		l := caps
		change(&l)
		if _, err := NewRegistry(l, time.Now); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, field := range []func(*Limits){func(l *Limits) { l.Workspaces = 0 }, func(l *Limits) { l.Findings = 0 }, func(l *Limits) { l.Cases = 0 },
		func(l *Limits) { l.CustomerCases = 0 }, func(l *Limits) { l.Steps = 0 }, func(l *Limits) { l.Messages = 0 }, func(l *Limits) { l.Entries = 0 }, func(l *Limits) { l.Subjects = 0 }} {
		l := testLimits()
		field(&l)
		if _, err := NewRegistry(l, time.Now); kind(err) != "invalid" {
			t.Error("a zero limit")
		}
	}
}

func TestRooms(t *testing.T) {
	l := testLimits()
	l.Workspaces, l.Cases, l.CustomerCases, l.Findings = 2, 4, 2, 2
	r, _ := openWith(t, l)
	staff := op("agent")
	must(r.OpenCase("w1", inquiry("a", "c1"), staff))
	must(r.RecordFinding("w2", spike("f", "c1"), staff))
	expect(t, "limit", second(r.OpenCase("w3", inquiry("a", "c1"), staff)), "a third workspace by a case")
	expect(t, "limit", second(r.RecordFinding("w3", spike("f", "c1"), staff)), "a third workspace by a finding")
	must(r.RecordFinding("w2", spike("g", "c1"), staff))
	expect(t, "limit", second(r.RecordFinding("w2", spike("h", "c1"), staff)), "findings")
	expect(t, "ok", second(r.RecordFinding("w2", spike("g", "c1"), staff)), "a repeat with no room")
	must(r.OpenCase("w1", inquiry("b", "c1"), staff))
	expect(t, "limit", second(r.OpenCase("w1", inquiry("c", "c1"), staff)), "cases of one customer")
	expect(t, "ok", second(r.OpenCase("w1", inquiry("b", "c1"), staff)), "a repeat with no customer room")
	must(r.OpenCase("w1", inquiry("a", "c2"), staff))
	must(r.OpenCase("w1", inquiry("b", "c2"), staff))
	expect(t, "limit", second(r.OpenCase("w1", inquiry("a", "c3"), staff)), "cases of a workspace")

	q, _ := openWith(t, testLimits())
	me := cust("u", "c1")
	must(q.OpenCase("ws", inquiry("q", "c1"), me))
	for i := 0; i < 3; i++ {
		must(q.Reply("ws", "c1", "q", me, Entry{ID: "m" + string(rune('0'+i)), Body: digest('1')}))
	}
	expect(t, "limit", second(q.Reply("ws", "c1", "q", me, Entry{ID: "m9", Body: digest('1')})), "messages")
	for i := 0; i < 2; i++ {
		must(q.Note("ws", "c1", "q", staff, Entry{ID: "n" + string(rune('0'+i)), Body: digest('2')}))
		must(q.Answer("ws", "c1", "q", staff, Entry{ID: "a" + string(rune('0'+i)), Body: digest('2')}))
	}
	expect(t, "limit", second(q.Answer("ws", "c1", "q", staff, Entry{ID: "a9", Body: digest('2')})), "operator entries")
	expect(t, "limit", second(q.Note("ws", "c1", "q", staff, Entry{ID: "n9", Body: digest('2')})), "notes")
	expect(t, "ok", second(q.Answer("ws", "c1", "q", staff, Entry{ID: "a1", Body: digest('2')})), "a repeat with no room")
	for i := 0; i < 2; i++ {
		must(q.AssignCase("ws", "c1", "q", staff, CaseStep{Step: "s" + string(rune('0'+i)), Owner: "o"}))
	}
	must(q.SetDue("ws", "c1", "q", staff, CaseStep{Step: "d", Due: start.Add(time.Hour), Reason: "initial"}))
	must(q.Close("ws", "c1", "q", staff, CaseStep{Step: "c", Closing: Closing{Outcome: OutcomeAnswered}}))
	must(q.ReopenCase("ws", "c1", "q", me, CaseStep{Step: "r"}))
	expect(t, "limit", second(q.AssignCase("ws", "c1", "q", staff, CaseStep{Step: "s9", Owner: "o"})), "steps")
	expect(t, "ok", second(q.Close("ws", "c1", "q", staff, CaseStep{Step: "c2", Closing: Closing{Outcome: OutcomeAnswered}})), "close with no room")
	expect(t, "limit", second(q.ReopenCase("ws", "c1", "q", me, CaseStep{Step: "r2"})), "reopen with no room")
	must(q.OpenCase("ws", inquiry("w", "c1"), me))
	for i := 0; i < 4; i++ {
		must(q.AssignCase("ws", "c1", "w", staff, CaseStep{Step: "s" + string(rune('0'+i)), Owner: "o"}))
	}
	expect(t, "ok", second(q.Withdraw("ws", "c1", "w", me, CaseStep{Step: "x"})), "withdraw with no room")
	expect(t, "ok", second(q.Withdraw("ws", "c1", "w", me, CaseStep{Step: "x"})), "a repeated withdrawal")
	expect(t, "rejected", second(q.Withdraw("ws", "c1", "w", me, CaseStep{Step: "y"})), "a second withdrawal")
	expect(t, "rejected", second(q.Withdraw("ws", "c1", "w", staff, CaseStep{Step: "y"})), "an operator withdrawal")
}

func TestCaseForms(t *testing.T) {
	r, _ := openWith(t, testLimits())
	staff := op("agent")
	good := dispute("d", "c1", "inv")
	for name, change := range map[string]func(*Case){
		"id":            func(c *Case) { c.ID = "" },
		"kind":          func(c *Case) { c.Kind = "claim" },
		"customer":      func(c *Case) { c.Customer = "c 1" },
		"topic":         func(c *Case) { c.Topic = "price" },
		"body":          func(c *Case) { c.Body = digest('A') },
		"no subject":    func(c *Case) { c.Subjects = nil },
		"repeated":      func(c *Case) { c.Subjects = []Subject{{Document: "a"}, {Document: "a"}} },
		"document":      func(c *Case) { c.Subjects = []Subject{{Line: "l"}} },
		"line":          func(c *Case) { c.Subjects = []Subject{{Document: "a", Line: "l l"}} },
		"zero amount":   func(c *Case) { c.Amount = "0.00" },
		"negative":      func(c *Case) { c.Amount = "-4" },
		"no currency":   func(c *Case) { c.Currency = "" },
		"no amount":     func(c *Case) { c.Amount = "" },
		"inquiry money": func(c *Case) { c.Kind = KindInquiry },
		"fraction":      func(c *Case) { c.Amount = "1." },
		"long fraction": func(c *Case) { c.Amount = "1." + strings.Repeat("1", 19) },
	} {
		c := good
		change(&c)
		expect(t, "invalid", second(r.OpenCase("ws", c, staff)), name)
	}
	many := good
	many.Subjects = []Subject{{Document: "a"}, {Document: "b"}, {Document: "c"}, {Document: "d"}}
	expect(t, "limit", second(r.OpenCase("ws", many, staff)), "subjects beyond the room")
	general := Case{ID: "g", Kind: KindInquiry, Customer: "c1", Topic: "access", Body: digest('a')}
	expect(t, "ok", second(r.OpenCase("ws", general, staff)), "an inquiry with no subject")
	edge := good
	edge.Amount, edge.Subjects = "0.000000000000000001", []Subject{{Document: "a"}, {Document: "a", Line: "1"}, {Document: "b"}}
	expect(t, "ok", second(r.OpenCase("ws", edge, staff)), "the smallest amount and the subject room")
	edge.Subjects[0].Document = "mutated"
	if v := must(r.Case("ws", "c1", "d", staff)); v.Case.Subjects[0].Document != "a" {
		t.Error("the case shares the caller's subjects")
	}
	v := must(r.Case("ws", "c1", "d", staff))
	v.Case.Subjects[0].Document = "mutated"
	v.Entries = append(v.Entries, EntryView{})
	if again := must(r.Case("ws", "c1", "d", staff)); again.Case.Subjects[0].Document != "a" || len(again.Entries) != 0 {
		t.Error("a view shares the case")
	}
	for name, err := range map[string]error{
		"workspace":   second(r.OpenCase("", good, staff)),
		"actor":       second(r.OpenCase("ws", good, Actor{ID: "x"})),
		"read":        second(r.Case("ws", "", "d", staff)),
		"entry id":    second(r.Answer("ws", "c1", "d", staff, Entry{Body: digest('1')})),
		"entry body":  second(r.Answer("ws", "c1", "d", staff, Entry{ID: "e", Body: "x"})),
		"step":        second(r.AssignCase("ws", "c1", "d", staff, CaseStep{Owner: "o"})),
		"owner":       second(r.AssignCase("ws", "c1", "d", staff, CaseStep{Step: "s"})),
		"assign due":  second(r.AssignCase("ws", "c1", "d", staff, CaseStep{Step: "s", Owner: "o", Due: start})),
		"due zero":    second(r.SetDue("ws", "c1", "d", staff, CaseStep{Step: "s", Reason: "initial"})),
		"due reason":  second(r.SetDue("ws", "c1", "d", staff, CaseStep{Step: "s", Due: start.Add(time.Hour), Reason: "cleared_on_reopen"})),
		"due owner":   second(r.SetDue("ws", "c1", "d", staff, CaseStep{Step: "s", Due: start.Add(time.Hour), Reason: "initial", Owner: "o"})),
		"close extra": second(r.Close("ws", "c1", "d", staff, CaseStep{Step: "s", Reason: "initial", Closing: Closing{Outcome: OutcomeNotUpheld}})),
		"reopen":      second(r.ReopenCase("ws", "c1", "d", staff, CaseStep{Step: "s", Owner: "o"})),
		"withdraw":    second(r.Withdraw("ws", "c1", "d", staff, CaseStep{Step: "s", Closing: Closing{Outcome: OutcomeNotUpheld}})),
		"list":        second(r.Cases("", staff, CaseFilter{})),
		"disputes":    second(r.Disputes("ws", "", "inv", "", staff)),
	} {
		expect(t, "invalid", err, name)
	}
}

func TestNumbers(t *testing.T) {
	for text, want := range map[string]bool{"0": true, "0.5": true, "10": true, "-1": true, "-0": false, "-0.0": false, "-": false, "": false, "01": false, ".5": false, "1.": false,
		"1e3": false, "1,0": false, "+1": false, strings.Repeat("9", 64): true, strings.Repeat("9", 65): false, "1." + strings.Repeat("0", 18): true} {
		if validNumber(text, true) != want {
			t.Errorf("%q", text)
		}
	}
	if validNumber("-1", false) {
		t.Error("a sign where none is allowed")
	}
	for _, date := range []string{"1999-12-31", "2100-01-01", "2026-13-01", "2026-1-01", "2026-10-01T"} {
		if validDate(date) {
			t.Errorf("date %q", date)
		}
	}
	if !validDate("2000-01-01") || !validDate("2099-12-31") || !validDate("2024-02-29") || validDate("2026-02-29") {
		t.Error("date bounds")
	}
	if validID(strings.Repeat("a", 129)) || !validID(strings.Repeat("a", 128)) || validID("a/b") || !validID("Az09._:-") {
		t.Error("identifiers")
	}
}

func TestClockNeverGoesBack(t *testing.T) {
	r, clock := openWith(t, testLimits())
	staff := op("agent")
	clock.set(start.Add(time.Hour))
	must(r.OpenCase("ws", inquiry("q", "c1"), staff))
	clock.set(start)
	v := must(r.Answer("ws", "c1", "q", staff, Entry{ID: "e", Body: digest('1')}))
	if !v.Entries[0].At.Equal(start.Add(time.Hour)) {
		t.Errorf("time went back: %v", v.Entries[0].At)
	}
	clock.set(time.Date(2026, 11, 14, 13, 0, 0, 5, time.FixedZone("x", 3600)))
	v = must(r.Answer("ws", "c1", "q", staff, Entry{ID: "e2", Body: digest('1')}))
	if at := v.Entries[1].At; at.Location() != time.UTC || at.Nanosecond() != 5 {
		t.Errorf("at %v", at)
	}
}

func TestConcurrentEntriesStayWithinRoom(t *testing.T) {
	l := testLimits()
	l.Messages = 20
	r, _ := openWith(t, l)
	me := cust("u", "c1")
	must(r.OpenCase("ws", inquiry("q", "c1"), me))
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	for i := 0; i < 60; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := r.Reply("ws", "c1", "q", me, Entry{ID: "m" + string(rune('A'+i)), Body: digest('1')}); err == nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if accepted != 20 || len(must(r.Case("ws", "c1", "q", me)).Entries) != 20 {
		t.Errorf("accepted %d", accepted)
	}
}

func TestSourceRules(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Contains(text, "flo"+"at") || strings.Contains(text, "/"+"/ ") || strings.Contains(text, "/"+"*") {
			t.Errorf("%s uses a binary fraction type or comments", e.Name())
		}
		if strings.Count(text, "\n") >= 1500 {
			t.Errorf("%s is too long", e.Name())
		}
		if !strings.HasSuffix(e.Name(), "_test.go") && strings.Contains(text, "hybrid_bil"+"ling/") {
			t.Errorf("%s imports another module", e.Name())
		}
	}
}
