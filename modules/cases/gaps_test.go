package cases

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDisputeAxisFilters(t *testing.T) {
	l := testLimits()
	l.Page, l.CustomerCases = 2, 8
	r, _ := openWith(t, l)
	staff, me := op("agent"), cust("u", "c1")
	must(r.OpenCase("ws", dispute("d1", "c1", "inv"), me))
	must(r.OpenCase("ws", dispute("d2", "c1", "inv"), me))
	must(r.OpenCase("ws", dispute("d3", "c1", "inv"), me))
	must(r.OpenCase("ws", dispute("d4", "c1", "other"), me))
	must(r.OpenCase("ws", dispute("d5", "c1", "inv"), me))
	q := inquiry("q", "c1")
	q.Subjects = []Subject{{Document: "inv"}}
	must(r.OpenCase("ws", q, me))
	must(r.Answer("ws", "c1", "d2", staff, Entry{ID: "a", Body: digest('1')}))
	must(r.Withdraw("ws", "c1", "d3", me, CaseStep{Step: "w"}))
	must(r.Close("ws", "c1", "d5", staff, CaseStep{Step: "c", Closing: Closing{Outcome: OutcomeNotUpheld}}))
	ids := func(after string) string {
		out := ""
		for _, v := range must(r.Disputes("ws", "c1", "inv", after, staff)) {
			out += v.Case.ID
		}
		return out
	}
	if got := ids(""); got != "d1d2" {
		t.Errorf("disputes: %q", got)
	}
	if got := ids("d1"); got != "d2" {
		t.Errorf("after d1: %q", got)
	}
	must(r.OpenCase("ws", dispute("d6", "c1", "inv"), me))
	must(r.OpenCase("ws", dispute("d0", "c1", "inv"), me))
	if got := ids(""); got != "d0d1" {
		t.Errorf("page: %q", got)
	}
	if got := ids("d1"); got != "d2d6" {
		t.Errorf("next page: %q", got)
	}
	expect(t, "invalid", second(r.Disputes("ws", "c1", "inv", "a b", staff)), "cursor")
	expect(t, "invalid", second(r.Disputes("ws", "c1", "inv", "", Actor{ID: "x"})), "actor")
	if got := len(must(r.Cases("ws", staff, CaseFilter{}))); got != 2 {
		t.Errorf("a list of more than a page: %d", got)
	}
}

func TestRefusedCallsChangeNothing(t *testing.T) {
	r, _ := openWith(t, testLimits())
	staff, me := op("agent"), cust("u", "c1")
	must(r.OpenCase("ws", inquiry("q", "c1"), me))
	must(r.AssignCase("ws", "c1", "q", staff, CaseStep{Step: "a", Owner: "ana"}))
	must(r.SetDue("ws", "c1", "q", staff, CaseStep{Step: "d", Due: start.Add(time.Hour), Reason: "initial"}))
	for _, state := range []string{CaseOpen, CaseAnswered, CaseClosed, CaseWithdrawn} {
		switch state {
		case CaseAnswered:
			must(r.Answer("ws", "c1", "q", staff, Entry{ID: "an", Body: digest('1')}))
		case CaseClosed:
			must(r.Close("ws", "c1", "q", staff, CaseStep{Step: "cl", Closing: Closing{Outcome: OutcomeAnswered}}))
		case CaseWithdrawn:
			must(r.ReopenCase("ws", "c1", "q", staff, CaseStep{Step: "ro"}))
			must(r.Withdraw("ws", "c1", "q", me, CaseStep{Step: "wd"}))
		}
		before := must(r.Case("ws", "c1", "q", staff))
		refused := []func() error{
			func() error { return second(r.ReopenCase("ws", "c1", "q", staff, CaseStep{Step: "x1"})) },
			func() error { return second(r.Withdraw("ws", "c1", "q", staff, CaseStep{Step: "x2"})) },
			func() error { return second(r.Answer("ws", "c1", "q", me, Entry{ID: "x3", Body: digest('2')})) },
			func() error { return second(r.Reply("ws", "c1", "q", staff, Entry{ID: "x4", Body: digest('2')})) },
			func() error {
				return second(r.SetDue("ws", "c1", "q", staff, CaseStep{Step: "x5", Due: start.Add(time.Hour), Reason: "initial"}))
			},
			func() error {
				return second(r.Close("ws", "c1", "q", staff, CaseStep{Step: "x6", Closing: Closing{Outcome: OutcomeNotUpheld}}))
			},
			func() error { return second(r.AssignCase("ws", "c1", "q", me, CaseStep{Step: "x7", Owner: "o"})) },
			func() error {
				return second(r.SetDue("ws", "c1", "q", me, CaseStep{Step: "x8", Due: start.Add(time.Hour), Reason: "follow_up"}))
			},
			func() error {
				return second(r.Close("ws", "c1", "q", me, CaseStep{Step: "x9", Closing: Closing{Outcome: OutcomeAnswered}}))
			},
		}
		for index, call := range refused {
			if state == CaseClosed && index == 0 {
				continue
			}
			expect(t, "rejected", call(), state)
		}
		after := must(r.Case("ws", "c1", "q", staff))
		if !reflect.DeepEqual(before, after) {
			t.Errorf("%s: a refused call changed the case\n%+v\n%+v", state, before, after)
		}
	}
}

func TestCaseHistoryDigests(t *testing.T) {
	roomy := testLimits()
	roomy.Steps = 10
	r, clock := openWith(t, roomy)
	staff, me := op("agent"), cust("u", "c1")
	must(r.OpenCase("ws", dispute("d", "c1", "inv"), me))
	must(r.AssignCase("ws", "c1", "d", staff, CaseStep{Step: "a1", Owner: "ana"}))
	must(r.AssignCase("ws", "c1", "d", staff, CaseStep{Step: "a2", Owner: "bo"}))
	must(r.SetDue("ws", "c1", "d", staff, CaseStep{Step: "d1", Due: start.Add(time.Hour), Reason: "initial"}))
	must(r.Close("ws", "c1", "d", staff, CaseStep{Step: "c1", Closing: Closing{Outcome: OutcomeNotUpheld}}))
	clock.advance(time.Second)
	must(r.ReopenCase("ws", "c1", "d", me, CaseStep{Step: "r1"}))
	v := must(r.SetDue("ws", "c1", "d", staff, CaseStep{Step: "d2", Due: clock.read().Add(time.Hour), Reason: "escalation"}))
	if len(v.Owners) != 2 || v.Owners[0].Owner != "ana" || v.Owners[1].Owner != "bo" || v.Owners[1].Step != "a2" || v.Owners[1].Actor != "agent" || v.Owner != "bo" {
		t.Fatalf("owners: %+v", v.Owners)
	}
	seen := map[string]bool{}
	for _, d := range []string{v.Owners[0].Digest, v.Owners[1].Digest, v.DueHistory[0].Digest, v.DueHistory[1].Digest, v.DueHistory[2].Digest, v.Closings[0].Digest, v.Closings[0].Reopen} {
		if len(d) != 64 || seen[d] {
			t.Errorf("digest %q", d)
		}
		seen[d] = true
	}
	if seen[v.Digest] {
		t.Error("a step digest equals the case digest")
	}
	if c := must(r.Case("ws", "c1", "d", me)); c.Owners != nil || c.Closings != nil {
		t.Error("a customer sees the history")
	}
	v.Owners[0].Owner = "mutated"
	if must(r.Case("ws", "c1", "d", staff)).Owners[0].Owner != "ana" {
		t.Error("a view shares the owner history")
	}
}

func TestPinnedDigest(t *testing.T) {
	r, _ := openWith(t, testLimits())
	v := must(r.RecordFinding("ws", spike("f", "c1"), op("rec")))
	sum := sha256.New()
	for _, field := range []string{findingLabel, "ws", "f", "cost_spike", "c1", "", "", "", "", "2026-10-01", "2026-10-31", "cost", "120.5", "USD", digest('a'), "high", "owner-1", "manual", "", "rec"} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		sum.Write(size[:])
		sum.Write([]byte(field))
	}
	if want := hex.EncodeToString(sum.Sum(nil)); v.Digest != want {
		t.Errorf("finding digest %s, want %s", v.Digest, want)
	}
}

func TestFindingEdges(t *testing.T) {
	r, _ := openWith(t, testLimits())
	f := spike("f", "c1")
	f.Meaning, f.Currency = MeaningNone, ""
	expect(t, "invalid", second(r.RecordFinding("ws", f, op("rec"))), "no meaning with an amount")
	f = spike("f", "c1")
	f.Meaning, f.Amount = MeaningNone, ""
	expect(t, "invalid", second(r.RecordFinding("ws", f, op("rec"))), "no meaning with a currency")
	must(r.RecordFinding("ws", spike("f", "c1"), op("rec")))
	bad := Actor{ID: "x", Side: SideCustomer}
	expect(t, "invalid", second(r.AssignFinding("ws", "f", bad, Transition{Step: "s", Owner: "o"})), "a malformed actor moves")
	expect(t, "invalid", second(r.AssignFinding("", "f", op("a"), Transition{Step: "s", Owner: "o"})), "a malformed workspace moves")
	expect(t, "invalid", second(r.AssignFinding("ws", "", op("a"), Transition{Step: "s", Owner: "o"})), "a malformed finding moves")
	expect(t, "invalid", second(r.Finding("ws", "f", bad)), "a malformed actor reads")
	expect(t, "invalid", second(r.Finding("", "f", op("a"))), "a malformed workspace reads")
	expect(t, "invalid", second(r.Finding("ws", "", op("a"))), "a malformed finding reads")
	expect(t, "not_found", second(r.Finding("ws", "g", op("a"))), "an unknown finding in a known workspace")
	expect(t, "not_found", second(r.Finding("nope", "f", op("a"))), "an unknown workspace")
	expect(t, "invalid", second(r.Findings("ws", bad, FindingFilter{})), "a malformed actor lists")
	for _, state := range []string{FindingConfirmedNormal, FindingAdjustmentNeeded, FindingResolved} {
		g := spike("g"+state, "c1")
		must(r.RecordFinding("ws", g, op("rec")))
		switch state {
		case FindingConfirmedNormal:
			must(r.ConfirmNormal("ws", g.ID, op("rev"), because("s", "duplicate")))
		default:
			must(r.RequireAdjustment("ws", g.ID, op("rev"), because("s", "rate_error")))
			if state == FindingResolved {
				must(r.Resolve("ws", g.ID, op("rev"), Transition{Step: "t", Evidence: digest('e'), Remedy: Remedy{Kind: "source_correction", Reference: "x"}}))
				expect(t, "rejected", second(r.ReopenFinding("ws", g.ID, op("rev"), because("u", "decision_revised"))), "decision_revised after a resolution")
			}
		}
		if v := must(r.AssignFinding("ws", g.ID, op("lead"), Transition{Step: "own", Owner: "z"})); v.State != state || v.Owner != "z" {
			t.Errorf("assign in %s: %+v", state, v)
		}
	}
	p, _ := openWith(t, Limits{Workspaces: 1, Findings: 3, Cases: 3, CustomerCases: 3, Steps: 1, Messages: 1, Entries: 1, Subjects: 1, Page: 2})
	must(p.RecordFinding("ws", spike("a", "c"), op("rec")))
	must(p.RecordFinding("ws", spike("b", "c"), op("rec")))
	if len(must(p.Findings("ws", op("x"), FindingFilter{}))) != 2 {
		t.Error("a full page of findings")
	}
	must(p.OpenCase("ws", inquiry("a", "c"), op("x")))
	must(p.OpenCase("ws", inquiry("b", "c"), op("x")))
	if len(must(p.Cases("ws", op("x"), CaseFilter{}))) != 2 {
		t.Error("a full page of cases")
	}
	must(p.OpenCase("ws", dispute("da", "c", "inv"), op("x")))
	if len(must(p.Disputes("ws", "c", "inv", "", op("x")))) != 1 {
		t.Error("disputes")
	}
}

func TestCaseEdges(t *testing.T) {
	r, _ := openWith(t, testLimits())
	staff := op("agent")
	d := dispute("d", "c1", "inv")
	must(r.OpenCase("ws", d, staff))
	other := d
	other.Currency = "EUR"
	expect(t, "conflict", second(r.OpenCase("ws", other, staff)), "another currency")
	other = d
	other.Subjects = []Subject{{Document: "inv", Line: "l2"}}
	expect(t, "conflict", second(r.OpenCase("ws", other, staff)), "another subject")
	expect(t, "invalid", second(r.Close("ws", "c1", "d", staff, CaseStep{Step: "s", Closing: Closing{Outcome: OutcomeUpheld, Remedy: Remedy{Kind: "source_correction", Reference: "x"}}})), "a source correction closes a case")
	expect(t, "invalid", second(r.Close("ws", "c1", "d", staff, CaseStep{Step: "s", Closing: Closing{Outcome: OutcomeUpheld, Remedy: Remedy{Kind: "credit_document"}}})), "a remedy without reference")
	must(r.Close("ws", "c1", "d", staff, CaseStep{Step: "s", Closing: Closing{Outcome: OutcomeUpheld, Remedy: Remedy{Kind: "credit_document", Reference: "a"}}}))
	expect(t, "conflict", second(r.Close("ws", "c1", "d", staff, CaseStep{Step: "s", Closing: Closing{Outcome: OutcomeUpheld, Remedy: Remedy{Kind: "credit_document", Reference: "b"}}})), "the same step with another closing")
	expect(t, "invalid", second(r.Cases("ws", Actor{ID: "x"}, CaseFilter{})), "a malformed actor lists")
	for _, id := range []string{"Z", "z", "9", "A", "a", "0"} {
		if !validID(id) {
			t.Errorf("identifier %q", id)
		}
	}
	for _, id := range []string{"/a", "[", "{", "@", "`", " a"} {
		if validID(id) {
			t.Errorf("identifier %q", id)
		}
	}
	for _, code := range []string{"AAA", "ZZZ"} {
		if !validCurrency(code) {
			t.Errorf("currency %q", code)
		}
	}
	for _, code := range []string{"@AA", "A@A", "AA@", "[AA", "A[A", "AA["} {
		if validCurrency(code) {
			t.Errorf("currency %q", code)
		}
	}
	for _, text := range []string{strings.Repeat("0", 64), strings.Repeat("f", 64), strings.Repeat("9", 64), strings.Repeat("a", 64)} {
		if !validDigest(text) {
			t.Errorf("digest %q", text)
		}
	}
	for _, text := range []string{"g" + strings.Repeat("0", 63), "/" + strings.Repeat("0", 63), "`" + strings.Repeat("0", 63)} {
		if validDigest(text) {
			t.Errorf("digest %q", text)
		}
	}
}

func TestConcurrentMixedCalls(t *testing.T) {
	l := testLimits()
	l.Messages, l.Entries, l.Steps = 10, 10, 10
	r, _ := openWith(t, l)
	staff, me := op("agent"), cust("u", "c1")
	must(r.OpenCase("ws", inquiry("q", "c1"), me))
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(4)
		id := string(rune('A' + i))
		go func() { defer wg.Done(); r.Reply("ws", "c1", "q", me, Entry{ID: "m" + id, Body: digest('1')}) }()
		go func() { defer wg.Done(); r.Answer("ws", "c1", "q", staff, Entry{ID: "a" + id, Body: digest('2')}) }()
		go func() {
			defer wg.Done()
			r.AssignCase("ws", "c1", "q", staff, CaseStep{Step: "s" + id, Owner: "o" + id})
		}()
		go func() { defer wg.Done(); r.Cases("ws", me, CaseFilter{}) }()
	}
	wg.Wait()
	v := must(r.Case("ws", "c1", "q", staff))
	messages, answers := 0, 0
	for _, e := range v.Entries {
		if e.Kind == EntryMessage {
			messages++
		} else {
			answers++
		}
	}
	if messages != 10 || answers != 10 || len(v.Owners) != 10 || v.Owner != v.Owners[9].Owner {
		t.Errorf("messages %d answers %d owners %d", messages, answers, len(v.Owners))
	}
}
