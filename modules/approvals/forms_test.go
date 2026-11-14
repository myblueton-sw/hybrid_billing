package approvals

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOpenAndProposalForms(t *testing.T) {
	clock := &testClock{now: start}
	r, _ := NewRegistry(testProfile(t), testLimits(), clock.read)
	expect(t, "invalid", second(r.Open("ws", []string{"a"}, human("i"))), "one administrator")
	expect(t, "invalid", second(r.Open("ws", []string{"a", "a"}, human("i"))), "a repeated administrator")
	expect(t, "invalid", second(r.Open("ws", []string{"a", "b", "c", "d", "e"}, human("i"))), "too many administrators")
	expect(t, "invalid", second(r.Open("ws", []string{"a", ""}, human("i"))), "a malformed administrator")
	expect(t, "invalid", second(r.Open("ws", []string{"a", "b"}, Actor{ID: "i", Kind: "robot"})), "a malformed actor")
	expect(t, ReasonAINotAllowed, second(r.Open("ws", []string{"a", "b"}, Actor{ID: "i", Kind: KindSystem})), "a system opening")
	if v := must(r.Open("ws", []string{"b", "a"}, human("i"))); len(v) != 2 || v[0] != "a" {
		t.Error(v)
	}
	expect(t, "conflict", second(r.Open("ws", []string{"a", "b"}, human("i"))), "a second opening")
	must(r.Open("w2", []string{"a", "b"}, human("i")))
	must(r.Open("w3", []string{"a", "b"}, human("i")))
	expect(t, "limit", second(r.Open("w4", []string{"a", "b"}, human("i"))), "workspaces")
	expect(t, "not_found", second(r.Admins("w9")), "unknown workspace")
	expect(t, "not_found", second(r.Propose("w9", Proposal{ID: "x", Kind: ProposeAddAdmin, Admin: "c"}, human("a"))), "unknown workspace")
	good := Policy{Stage: StageCharge, Revision: 1, Approvals: 2, Required: []string{"result", "input"}, Validity: time.Hour}
	for name, change := range map[string]func(*Policy){
		"stage": func(p *Policy) { p.Stage = "pay" }, "revision": func(p *Policy) { p.Revision = 0 }, "approvals": func(p *Policy) { p.Approvals = 4 }, "no approvals": func(p *Policy) { p.Approvals = 0 },
		"short": func(p *Policy) { p.Validity = time.Hour - 1 }, "long": func(p *Policy) { p.Validity = 90*24*time.Hour + 1 }, "empty": func(p *Policy) { p.Required = nil },
		"no result": func(p *Policy) { p.Required = []string{"input"} }, "kind": func(p *Policy) { p.Required = []string{"result", "tax"} }, "repeat": func(p *Policy) { p.Required = []string{"result", "result"} },
	} {
		p := good
		change(&p)
		expect(t, "invalid", second(r.Propose("ws", Proposal{ID: "p", Kind: ProposePolicy, Policy: p}, human("a"))), name)
	}
	v := must(r.Propose("ws", Proposal{ID: "p", Kind: ProposePolicy, Policy: good}, human("a")))
	if v.Proposal.Policy.Required[0] != "input" || len(v.Digest) != 64 || v.State != StatePending {
		t.Errorf("sorted kinds: %+v", v)
	}
	expect(t, "ok", second(r.Propose("ws", Proposal{ID: "p", Kind: ProposePolicy, Policy: Policy{Stage: StageCharge, Revision: 1, Approvals: 2, Required: []string{"input", "result"}, Validity: time.Hour}}, human("a"))), "a repeat in another order")
	expect(t, "conflict", second(r.Propose("ws", Proposal{ID: "p", Kind: ProposeAddAdmin, Admin: "c"}, human("a"))), "a proposal conflict")
	expect(t, "conflict", second(r.Propose("ws", Proposal{ID: "p", Kind: ProposePolicy, Policy: good}, human("b"))), "a proposal by another")
	must(r.Confirm("ws", "p", human("b")))
	if p := must(r.PolicyOf("ws", StageCharge)); p.Approvals != 2 || len(p.Required) != 2 {
		t.Error(p)
	}
	expect(t, ReasonNoPolicy, second(r.PolicyOf("ws", StageIssue)), "no policy")
	expect(t, "invalid", second(r.PolicyOf("ws", "pay")), "stage")
	expect(t, "not_found", second(r.PolicyOf("w9", StageIssue)), "workspace")
	ga := Authority{ID: "g", Approver: "c", Stage: StageCharge, Currency: "USD", Limit: "10.50", From: "2026-01-01", To: "2026-12-31"}
	for name, change := range map[string]func(*Authority){
		"id": func(a *Authority) { a.ID = "" }, "approver": func(a *Authority) { a.Approver = "a b" }, "stage": func(a *Authority) { a.Stage = "x" }, "currency": func(a *Authority) { a.Currency = "usd" },
		"limit": func(a *Authority) { a.Limit = "-1" }, "from": func(a *Authority) { a.From = "2026-13-01" }, "to": func(a *Authority) { a.To = "2025-12-31" }, "to form": func(a *Authority) { a.To = "x" },
	} {
		a := ga
		change(&a)
		expect(t, "invalid", second(r.Propose("ws", Proposal{ID: "q", Kind: ProposeGrant, Authority: a}, human("a"))), name)
	}
	for name, p := range map[string]Proposal{
		"kind": {ID: "q", Kind: "promote"}, "id": {Kind: ProposeAddAdmin, Admin: "c"}, "admin": {ID: "q", Kind: ProposeAddAdmin},
		"grant with admin": {ID: "q", Kind: ProposeGrant, Authority: ga, Admin: "c"}, "policy with grant": {ID: "q", Kind: ProposePolicy, Policy: good, Authority: ga},
		"admin with policy": {ID: "q", Kind: ProposeRemoveAdmin, Admin: "c", Policy: good},
		"grant with policy": {ID: "q", Kind: ProposeGrant, Authority: ga, Policy: good}, "admin with grant": {ID: "q", Kind: ProposeAddAdmin, Admin: "c", Authority: ga},
	} {
		expect(t, "invalid", second(r.Propose("ws", p, human("a"))), name)
	}
	must(r.Propose("ws", Proposal{ID: "g", Kind: ProposeGrant, Authority: ga}, human("a")))
	must(r.Confirm("ws", "g", human("b")))
	if a := must(r.AuthorityOf("ws", "g")); a.Authority.Limit != "10.5" || a.Proposer != "a" || a.Confirmer != "b" {
		t.Error(a)
	}
	must(r.Propose("ws", Proposal{ID: "g2", Kind: ProposeGrant, Authority: ga}, human("a")))
	expect(t, "conflict", second(r.Confirm("ws", "g2", human("b"))), "a grant id used twice")
	expect(t, "not_found", second(r.AuthorityOf("ws", "zz")), "unknown authority")
	expect(t, "not_found", second(r.Confirm("ws", "zz", human("b"))), "unknown proposal")
	expect(t, "not_found", second(r.Revoke("ws", "zz", human("b"))), "unknown authority revoke")
	must(r.Propose("ws", Proposal{ID: "dup", Kind: ProposeAddAdmin, Admin: "a"}, human("b")))
	expect(t, ReasonStaleProposal, second(r.Confirm("ws", "dup", human("a"))), "adding an administrator twice")
	must(r.Propose("ws", Proposal{ID: "gone", Kind: ProposeRemoveAdmin, Admin: "zz"}, human("b")))
	expect(t, ReasonStaleProposal, second(r.Confirm("ws", "gone", human("a"))), "removing a stranger")
	expect(t, ReasonAINotAllowed, second(r.Confirm("ws", "gone", Actor{ID: "a", Kind: KindAssistant})), "an assistant confirming")
}

func TestRequestFormsAndRooms(t *testing.T) {
	r, _ := setup(t, 1)
	for name, change := range map[string]func(*Subject){
		"id": func(s *Subject) { s.ID = "" }, "stage": func(s *Subject) { s.Stage = "pay" }, "target": func(s *Subject) { s.Target = "" }, "author": func(s *Subject) { s.Author = "" },
		"currency": func(s *Subject) { s.Currency = "US" }, "no digests": func(s *Subject) { s.Digests = nil }, "digest kind": func(s *Subject) { s.Digests = map[string]string{"tax": digest('a')} },
		"digest": func(s *Subject) { s.Digests = map[string]string{"result": "x"} }, "amount": func(s *Subject) { s.Amount = "1e3" },
		"entry item":  func(s *Subject) { s.Amount, s.Entries = "1", []Entry{{Scope: "c", Amount: "1", Digest: digest('1')}} },
		"entry scope": func(s *Subject) { s.Amount, s.Entries = "1", []Entry{{Item: "i", Amount: "1", Digest: digest('1')}} },
		"entry digest": func(s *Subject) {
			s.Amount, s.Entries = "1", []Entry{{Item: "i", Scope: "c", Amount: "1", Digest: "x"}}
		},
		"entry amount": func(s *Subject) {
			s.Amount, s.Entries = "1", []Entry{{Item: "i", Scope: "c", Amount: "x", Digest: digest('1')}}
		},
		"entry repeat": func(s *Subject) {
			s.Amount, s.Entries = "2", []Entry{{Item: "i", Scope: "c", Amount: "1", Digest: digest('1')}, {Item: "i", Scope: "c", Amount: "1", Digest: digest('1')}}
		},
		"entries": func(s *Subject) {
			s.Amount = "5"
			for _, id := range []string{"a", "b", "c", "d", "e"} {
				s.Entries = append(s.Entries, Entry{Item: id, Scope: "c", Amount: "1", Digest: digest('1')})
			}
		},
	} {
		s := subject("q", "inv", "10")
		change(&s)
		expect(t, "invalid", second(r.Request("ws", s, human("req"))), name)
	}
	s := subject("q", "inv", "10")
	s.Digests = map[string]string{"result": digest('a')}
	expect(t, "rejected", second(r.Request("ws", s, human("req"))), "a missing required digest")
	s.Digests = map[string]string{"result": digest('a'), "fx": digest('b')}
	expect(t, "rejected", second(r.Request("ws", s, human("req"))), "another required digest")
	s.Digests = map[string]string{"result": digest('a'), "erp_draft": digest('b'), "fx": digest('c')}
	expect(t, "rejected", second(r.Request("ws", s, human("req"))), "an extra digest")
	d := subject("d", "inv", "10")
	d.Stage = StageDeliver
	expect(t, ReasonNoPolicy, second(r.Request("ws", d, human("req"))), "a stage without policy")
	expect(t, "not_found", second(r.Request("w9", subject("q", "inv", "10"), human("req"))), "unknown workspace")
	in := subject("q", "inv", "10.0")
	must(r.Request("ws", in, human("req")))
	in.Digests["result"] = digest('z' - 20)
	if v := must(r.RequestOf("ws", "q")); v.Subject.Digests["result"] != digest('a') {
		t.Error("the request shares the caller's digests")
	}
	expect(t, "ok", second(r.Request("ws", subject("q", "inv", "10"), human("req"))), "a repeat")
	expect(t, "conflict", second(r.Request("ws", subject("q", "inv", "11"), human("req"))), "another amount")
	expect(t, "conflict", second(r.Request("ws", subject("q", "inv", "10"), human("other"))), "another requester")
	for i := 1; i < 10; i++ {
		must(r.Request("ws", subject("r"+string(rune('0'+i)), "t"+string(rune('0'+i)), "1"), human("req")))
	}
	expect(t, "limit", second(r.Request("ws", subject("rx", "tx", "1"), human("req"))), "requests")
	expect(t, "not_found", second(r.RequestOf("ws", "zz")), "unknown request")
	expect(t, "invalid", second(r.RequestOf("", "q")), "malformed read")
	for name, d := range map[string]Decision{
		"verdict": {Verdict: "maybe"}, "approve reason": {Verdict: VerdictApprove, Reason: "other"}, "reject no reason": {Verdict: VerdictReject},
		"reject reason": {Verdict: VerdictReject, Reason: "dislike"}, "reject excludes": {Verdict: VerdictReject, Reason: "other", Exclude: []Exclusion{{Item: "a", Reason: "not_eligible"}}},
		"exclusion reason": {Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "a", Reason: "dislike"}}}, "exclusion repeat": {Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "a", Reason: "not_eligible"}, {Item: "a", Reason: "not_eligible"}}},
		"exclusion item": {Verdict: VerdictApprove, Exclude: []Exclusion{{Reason: "not_eligible"}}},
	} {
		expect(t, "invalid", second(r.Decide("ws", "q", d, human("ann"))), name)
	}
	expect(t, "not_found", second(r.Decide("ws", "zz", approve, human("ann"))), "unknown request decision")
	expect(t, "not_found", second(r.Withdraw("ws", "zz", human("req"))), "unknown request withdrawal")
	expect(t, "rejected", second(r.Withdraw("ws", "q", human("ann"))), "withdrawal by another")
	must(r.Withdraw("ws", "q", human("req")))
	expect(t, "ok", second(r.Withdraw("ws", "q", human("req"))), "a repeat withdrawal")
	expect(t, "rejected", second(r.Withdraw("ws", "r1", human("ann"))), "another withdrawal")
	must(r.Decide("ws", "r2", approve, human("ann")))
	must(r.Execute("ws", runOf("e", "r2", "t2", "1"), human("ops")))
	expect(t, "rejected", second(r.Withdraw("ws", "r2", human("req"))), "withdraw an executed single subject")
}

func TestExecutionFormsAndRooms(t *testing.T) {
	l := testLimits()
	l.Executions = 1
	clock := &testClock{now: start}
	r, _ := NewRegistry(testProfile(t), l, clock.read)
	must(r.Open("ws", []string{"adm1", "adm2"}, human("installer")))
	administer(t, r, policy(StageIssue, 1, 1, "result", "erp_draft"))
	administer(t, r, grant("g-ann", "ann", StageIssue, "1000"))
	must(r.Request("ws", subject("q", "inv", "10"), human("req")))
	must(r.Request("ws", subject("q2", "inv2", "10"), human("req")))
	must(r.Decide("ws", "q", approve, human("ann")))
	must(r.Decide("ws", "q2", approve, human("ann")))
	for name, change := range map[string]func(*Execution){
		"id": func(e *Execution) { e.ID = "" }, "request": func(e *Execution) { e.Request = "" }, "stage": func(e *Execution) { e.Stage = "x" }, "target": func(e *Execution) { e.Target = "" },
		"currency": func(e *Execution) { e.Currency = "" }, "digests": func(e *Execution) { e.Digests = nil }, "digest": func(e *Execution) { e.Digests["result"] = "x" },
		"kind": func(e *Execution) { e.Digests["tax"] = digest('a') }, "item digest alone": func(e *Execution) { e.ItemDigest = digest('a') },
		"item without digest": func(e *Execution) { e.Item = "i" }, "item form": func(e *Execution) { e.Item, e.ItemDigest = "a b", digest('a') }, "amount": func(e *Execution) { e.Amount = "x" },
	} {
		e := runOf("e", "q", "inv", "10")
		change(&e)
		expect(t, "invalid", second(r.Execute("ws", e, human("ops"))), name)
	}
	expect(t, "invalid", second(r.Execute("ws", runOf("e", "q", "inv", "10"), Actor{ID: "ops"})), "actor")
	expect(t, "not_found", second(r.Execute("ws", runOf("e", "zz", "inv", "10"), human("ops"))), "unknown request")
	expect(t, "not_found", second(r.Execute("w9", runOf("e", "q", "inv", "10"), human("ops"))), "unknown workspace")
	must(r.Execute("ws", runOf("e", "q", "inv", "10.00"), Actor{ID: "ops", Kind: KindSystem}))
	expect(t, "limit", second(r.Execute("ws", runOf("e2", "q2", "inv2", "10"), human("ops"))), "executions")
	expect(t, "not_found", second(r.ExecutionOf("ws", "zz")), "unknown execution")
	expect(t, "invalid", second(r.ExecutionOf("ws", "")), "malformed execution")
	v := must(r.ExecutionOf("ws", "e"))
	v.Execution.Digests["result"] = "mutated"
	if must(r.ExecutionOf("ws", "e")).Execution.Digests["result"] != digest('a') {
		t.Error("a view shares the digests")
	}
}

func TestRegistryLimits(t *testing.T) {
	if _, err := NewRegistry(testProfile(t), testLimits(), nil); kind(err) != "invalid" {
		t.Error("no clock")
	}
	caps := Limits{Workspaces: 100000, Admins: 100, Proposals: 100000, Authorities: 100000, Requests: 1000000, Items: 100000, Executions: 10000000}
	if _, err := NewRegistry(testProfile(t), caps, time.Now); err != nil {
		t.Error(err)
	}
	for name, change := range map[string]func(*Limits){
		"w": func(l *Limits) { l.Workspaces++ }, "a": func(l *Limits) { l.Admins++ }, "one admin": func(l *Limits) { l.Admins = 1 }, "p": func(l *Limits) { l.Proposals++ },
		"au": func(l *Limits) { l.Authorities++ }, "r": func(l *Limits) { l.Requests++ }, "i": func(l *Limits) { l.Items++ }, "e": func(l *Limits) { l.Executions++ },
	} {
		l := caps
		change(&l)
		if _, err := NewRegistry(testProfile(t), l, time.Now); kind(err) != "invalid" {
			t.Errorf("%s accepted", name)
		}
	}
	l := testLimits()
	l.Proposals, l.Authorities, l.Admins = 3, 1, 2
	clock := &testClock{now: start}
	r, _ := NewRegistry(testProfile(t), l, clock.read)
	must(r.Open("ws", []string{"a", "b"}, human("i")))
	must(r.Propose("ws", Proposal{ID: "g1", Kind: ProposeGrant, Authority: Authority{ID: "x", Approver: "c", Stage: StageIssue, Currency: "USD", Limit: "1", From: "2026-01-01"}}, human("a")))
	must(r.Propose("ws", Proposal{ID: "g2", Kind: ProposeGrant, Authority: Authority{ID: "y", Approver: "c", Stage: StageIssue, Currency: "USD", Limit: "1", From: "2026-01-01"}}, human("a")))
	must(r.Propose("ws", Proposal{ID: "add", Kind: ProposeAddAdmin, Admin: "c"}, human("a")))
	expect(t, "limit", second(r.Propose("ws", Proposal{ID: "x", Kind: ProposeAddAdmin, Admin: "d"}, human("a"))), "proposals")
	must(r.Confirm("ws", "g1", human("b")))
	expect(t, "limit", second(r.Confirm("ws", "g2", human("b"))), "authorities")
	expect(t, "limit", second(r.Confirm("ws", "add", human("b"))), "administrators")
}

func TestAuthorityDates(t *testing.T) {
	r, clock := setup(t, 1)
	administer(t, r, Proposal{Kind: ProposeGrant, Authority: Authority{ID: "g-day", Approver: "dan", Stage: StageIssue, Currency: "USD", Limit: "10", From: "2026-11-14", To: "2026-11-14"}})
	administer(t, r, Proposal{Kind: ProposeGrant, Authority: Authority{ID: "g-eur", Approver: "eve", Stage: StageIssue, Currency: "EUR", Limit: "10", From: "2026-01-01"}})
	administer(t, r, Proposal{Kind: ProposeGrant, Authority: Authority{ID: "g-late", Approver: "fay", Stage: StageIssue, Currency: "USD", Limit: "10", From: "2026-11-15"}})
	administer(t, r, Proposal{Kind: ProposeGrant, Authority: Authority{ID: "g-stage", Approver: "gus", Stage: StageDeliver, Currency: "USD", Limit: "10", From: "2026-01-01"}})
	must(r.Request("ws", subject("q", "inv", "10"), human("req")))
	expect(t, ReasonNoAuthority, second(r.Decide("ws", "q", approve, human("eve"))), "another currency")
	expect(t, ReasonNoAuthority, second(r.Decide("ws", "q", approve, human("fay"))), "an authority not yet active")
	expect(t, ReasonNoAuthority, second(r.Decide("ws", "q", approve, human("gus"))), "another stage")
	must(r.Decide("ws", "q", approve, human("dan")))
	clock.advance(15 * time.Hour)
	expect(t, ReasonAuthorityLost, second(r.Execute("ws", runOf("e", "q", "inv", "10"), human("ops"))), "an authority that ended yesterday")
}

func TestConcurrentExecutionsConsumeOnce(t *testing.T) {
	r, _ := setup(t, 1)
	must(r.Request("ws", subject("q", "inv", "10"), human("req")))
	must(r.Decide("ws", "q", approve, human("ann")))
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := r.Execute("ws", runOf("e"+string(rune('a'+i)), "q", "inv", "10"), human("ops")); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if ok != 1 {
		t.Errorf("%d executions", ok)
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
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, "hybrid_bil"+"ling/") && !strings.Contains(line, "packages/primitives/") {
				t.Errorf("%s imports %s", e.Name(), strings.TrimSpace(line))
			}
		}
	}
}

func TestSmallHelpers(t *testing.T) {
	err := refuse(ReasonExpired)
	if !strings.HasSuffix(err.Error(), ": expired") || !strings.HasPrefix(err.Error(), ErrRejected.Error()) {
		t.Error(err)
	}
	for _, text := range []string{strings.Repeat("0", 64), strings.Repeat("f", 64)} {
		if !validDigest(text) {
			t.Error(text)
		}
	}
	for _, text := range []string{"g" + strings.Repeat("0", 63), "/" + strings.Repeat("0", 63), strings.Repeat("0", 63)} {
		if validDigest(text) {
			t.Error(text)
		}
	}
	r, _ := setup(t, 1)
	expect(t, "invalid", second(r.Admins("")), "malformed admins")
	expect(t, "invalid", second(r.AuthorityOf("", "g")), "malformed authority")
	expect(t, "invalid", second(r.Revoke("ws", "", human("a"))), "malformed revoke")
	expect(t, "invalid", second(r.Confirm("ws", "", human("a"))), "malformed confirm")
	expect(t, "invalid", second(r.Propose("", Proposal{ID: "x", Kind: ProposeAddAdmin, Admin: "c"}, human("adm1"))), "malformed propose")
	expect(t, "invalid", second(r.Decide("ws", "", approve, human("ann"))), "malformed decide")
	expect(t, "invalid", second(r.Withdraw("ws", "", human("ann"))), "malformed withdraw")
	expect(t, "invalid", second(r.Report("ws", "", Outcome{Result: ResultSucceeded}, human("ann"))), "malformed report")
	expect(t, "invalid", second(r.Request("", subject("q", "inv", "1"), human("req"))), "malformed request")
	expect(t, "invalid", second(r.Execute("", runOf("e", "q", "inv", "1"), human("ops"))), "malformed execute")
	expect(t, "invalid", second(r.Request("ws", subject("q", "inv", "1."+strings.Repeat("0", 70)), human("req"))), "a long amount")
	expect(t, "not_found", second(r.Revoke("w9", "g", human("a"))), "unknown workspace revoke")
	expect(t, "not_found", second(r.AuthorityOf("w9", "g")), "unknown workspace authority")
	expect(t, "not_found", second(r.Confirm("w9", "p", human("a"))), "unknown workspace confirm")
	expect(t, "not_found", second(r.Decide("w9", "q", approve, human("ann"))), "unknown workspace decide")
	expect(t, "not_found", second(r.Withdraw("w9", "q", human("ann"))), "unknown workspace withdraw")
	expect(t, "not_found", second(r.Report("w9", "e", Outcome{Result: ResultSucceeded}, human("ann"))), "unknown workspace report")
	expect(t, "not_found", second(r.RequestOf("w9", "q")), "unknown workspace request")
	expect(t, "not_found", second(r.ExecutionOf("w9", "e")), "unknown workspace execution")
	expect(t, "ok", second(r.Revoke("ws", "g-ann", human("ann"))), "a grantee revoking its own authority")
	if v := must(r.AuthorityOf("ws", "g-ann")); v.RevokedBy != "ann" || v.Revoked.IsZero() {
		t.Error(v)
	}
	p := policy(StageIssue, 2, 2, "result")
	p.ID = "pp"
	must(r.Propose("ws", p, human("adm1")))
	for name, other := range map[string]Policy{
		"approvals": {Stage: StageIssue, Revision: 2, Approvals: 1, Required: []string{"result"}, Validity: 24 * time.Hour},
		"validity":  {Stage: StageIssue, Revision: 2, Approvals: 2, Required: []string{"result"}, Validity: 25 * time.Hour},
		"required":  {Stage: StageIssue, Revision: 2, Approvals: 2, Required: []string{"result", "fx"}, Validity: 24 * time.Hour},
		"kinds":     {Stage: StageIssue, Revision: 2, Approvals: 2, Required: []string{"fx", "result"}, Validity: 24 * time.Hour},
		"revision":  {Stage: StageIssue, Revision: 3, Approvals: 2, Required: []string{"result"}, Validity: 24 * time.Hour},
		"stage":     {Stage: StageCharge, Revision: 2, Approvals: 2, Required: []string{"result"}, Validity: 24 * time.Hour},
	} {
		expect(t, "conflict", second(r.Propose("ws", Proposal{ID: "pp", Kind: ProposePolicy, Policy: other}, human("adm1"))), name)
	}
	q := subject("q", "inv", "10")
	q.Entries = []Entry{{Item: "a", Scope: "c", Amount: "10", Digest: digest('1')}}
	must(r.Request("ws", q, human("req")))
	for name, change := range map[string]func(*Subject){
		"target": func(s *Subject) { s.Target = "inv2" }, "author": func(s *Subject) { s.Author = "w2" }, "currency": func(s *Subject) { s.Currency = "EUR" },
		"digests": func(s *Subject) { s.Digests = map[string]string{"result": digest('a'), "erp_draft": digest('c')} }, "stage": func(s *Subject) { s.Stage = StageCharge },
		"entry": func(s *Subject) { s.Entries = []Entry{{Item: "a", Scope: "c2", Amount: "10", Digest: digest('1')}} },
		"entries": func(s *Subject) {
			s.Entries = []Entry{{Item: "a", Scope: "c", Amount: "5", Digest: digest('1')}, {Item: "b", Scope: "c", Amount: "5", Digest: digest('1')}}
		},
	} {
		s := q
		s.Digests = digests()
		change(&s)
		expect(t, "conflict", second(r.Request("ws", s, human("req"))), name)
	}
	administer(t, r, grant("g-x", "xen", StageIssue, "100"))
	must(r.Decide("ws", "q", Decision{Verdict: VerdictApprove}, human("xen")))
	for name, d := range map[string]Decision{
		"reject":  {Verdict: VerdictReject, Reason: "other"},
		"exclude": {Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "a", Reason: "not_eligible"}}},
	} {
		expect(t, "conflict", second(r.Decide("ws", "q", d, human("xen"))), name)
	}
}
