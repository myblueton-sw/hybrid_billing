package approvals

import (
	"testing"
	"time"
)

func bulk(id, target string) Subject {
	s := subject(id, target, "600")
	s.Entries = []Entry{{Item: "inv-a", Scope: "c1", Amount: "300", Digest: digest('1')}, {Item: "inv-b", Scope: "c2", Amount: "-200", Digest: digest('2')}, {Item: "inv-c", Scope: "c3", Amount: "500", Digest: digest('3')}}
	return s
}

func item(id, request, name, amount string, d byte) Execution {
	e := runOf(id, request, "batch", amount)
	e.Item, e.ItemDigest = name, digest(d)
	return e
}

func TestBulkExclusionsAndPartialExecution(t *testing.T) {
	r, _ := setup(t, 2)
	administer(t, r, grant("g-dan", "dan", StageIssue, "800"))
	bad := bulk("x", "batch")
	bad.Amount = "601"
	expect(t, "invalid", second(r.Request("ws", bad, human("req"))), "an amount that is not the sum")
	must(r.Request("ws", bulk("q", "batch"), human("req")))
	expect(t, ReasonNoAuthority, second(r.Decide("ws", "q", approve, human("dan"))), "the absolute sum 1000 above a limit of 800")
	expect(t, "not_found", second(r.Decide("ws", "q", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "inv-z", Reason: "not_eligible"}}}, human("ann"))), "an unknown item")
	expect(t, ReasonNoAuthority, second(r.Decide("ws", "q", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "inv-b", Reason: "customer_hold"}}}, human("bob"))), "bob's limit after one exclusion")
	must(r.Decide("ws", "q", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "inv-c", Reason: "blocked_error"}}}, human("ann")))
	expect(t, ReasonNothingLeft, second(r.Decide("ws", "q", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "inv-a", Reason: "data_missing"}, {Item: "inv-b", Reason: "data_missing"}}}, human("dan"))), "excluding what is left")
	v := must(r.Decide("ws", "q", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "inv-b", Reason: "customer_hold"}}}, human("dan")))
	if v.State != StateApproved || len(v.Excluded) != 2 || v.Excluded[0].Item != "inv-b" || v.Excluded[1].Item != "inv-c" {
		t.Fatalf("union of exclusions: %+v", v)
	}
	expect(t, ReasonItemIneligible, second(r.Execute("ws", item("e-c", "q", "inv-c", "500", '3'), human("ops"))), "an excluded item")
	expect(t, ReasonItemIneligible, second(r.Execute("ws", item("e-z", "q", "inv-z", "1", '9'), human("ops"))), "an item outside the manifest")
	expect(t, ReasonSubjectChanged, second(r.Execute("ws", item("e-a", "q", "inv-a", "301", '1'), human("ops"))), "another item amount")
	expect(t, ReasonSubjectChanged, second(r.Execute("ws", item("e-a", "q", "inv-a", "300", '9'), human("ops"))), "another item digest")
	expect(t, ReasonSubjectChanged, second(r.Execute("ws", runOf("e-all", "q", "batch", "600"), human("ops"))), "executing a bulk subject without an item")
	must(r.Execute("ws", item("e-a", "q", "inv-a", "300", '1'), human("ops")))
	must(r.Report("ws", "e-a", Outcome{Result: ResultFailed, Reason: "timeout"}, human("ops")))
	expect(t, ReasonAlreadyExecuted, second(r.Execute("ws", item("e-a2", "q", "inv-a", "300", '1'), human("ops"))), "a new execution after a failed act")
	expect(t, "ok", second(r.Execute("ws", item("e-a", "q", "inv-a", "300", '1'), human("ops"))), "a replay after a failed act")
	single := subject("s", "inv-a", "300")
	must(r.Request("ws", single, human("req")))
	must(r.Decide("ws", "s", approve, human("ann")))
	must(r.Decide("ws", "s", approve, human("dan")))
	expect(t, ReasonAlreadyExecuted, second(r.Execute("ws", runOf("e-s", "s", "inv-a", "300"), human("ops"))), "a single subject over an executed bulk item")
	must(r.Withdraw("ws", "q", human("req")))
	if v := must(r.RequestOf("ws", "q")); v.State != StateWithdrawn || len(v.Executed) != 1 || v.Executed[0] != "inv-a" {
		t.Fatalf("withdrawn after a partial run: %+v", v)
	}
	expect(t, ReasonWithdrawn, second(r.Execute("ws", item("e-a", "q", "inv-a", "300", '1'), human("ops"))), "a replay after withdrawal")
}

func TestReplaysAndReports(t *testing.T) {
	r, clock := setup(t, 1)
	must(r.Request("ws", subject("q", "inv", "10"), human("req")))
	must(r.Decide("ws", "q", approve, human("ann")))
	must(r.Execute("ws", runOf("e", "q", "inv", "10"), human("ops")))
	other := runOf("e", "q", "inv", "10")
	other.Digests["result"] = digest('9')
	expect(t, "conflict", second(r.Execute("ws", other, human("ops"))), "a replay with other content")
	clock.advance(25 * time.Hour)
	expect(t, ReasonExpired, second(r.Execute("ws", runOf("e", "q", "inv", "10"), human("ops"))), "a replay after expiry")
	expect(t, "rejected", second(r.Report("ws", "e", Outcome{Result: ResultSucceeded}, human("other"))), "a report by another actor")
	must(r.Report("ws", "e", Outcome{Result: ResultSucceeded}, human("ops")))
	expect(t, "ok", second(r.Report("ws", "e", Outcome{Result: ResultSucceeded}, human("ops"))), "a repeat report")
	expect(t, "conflict", second(r.Report("ws", "e", Outcome{Result: ResultFailed, Reason: "timeout"}, human("ops"))), "a failure after success")
	if v := must(r.Execute("ws", runOf("e", "q", "inv", "10"), human("ops"))); v.Outcome.Result != ResultSucceeded {
		t.Error("a replay after success")
	}
	for name, o := range map[string]Outcome{"reason on success": {Result: ResultSucceeded, Reason: "timeout"}, "no reason": {Result: ResultFailed}, "result": {Result: "partial"}, "reason": {Result: ResultFailed, Reason: "sad"}} {
		expect(t, "invalid", second(r.Report("ws", "e", o, human("ops"))), name)
	}
	expect(t, "not_found", second(r.Report("ws", "zz", Outcome{Result: ResultSucceeded}, human("ops"))), "unknown execution")
	if v := must(r.ExecutionOf("ws", "e")); v.Outcome.Result != ResultSucceeded || v.Reported.IsZero() {
		t.Error(v)
	}
}

func TestExecutionCheckOrder(t *testing.T) {
	r, clock := setup(t, 1)
	must(r.Request("ws", bulk("q", "batch"), human("req")))
	must(r.Decide("ws", "q", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "inv-c", Reason: "not_eligible"}, {Item: "inv-b", Reason: "not_eligible"}}}, human("bob")))
	must(r.Execute("ws", item("done", "q", "inv-a", "300", '1'), human("ops")))
	bad := item("x", "q", "inv-c", "999", '9')
	bad.Digests["result"] = digest('8')
	must(r.Revoke("ws", "g-bob", human("adm1")))
	expect(t, ReasonSubjectChanged, second(r.Execute("ws", bad, human("ops"))), "subject before authority")
	bad = item("x", "q", "inv-c", "500", '3')
	expect(t, ReasonAuthorityLost, second(r.Execute("ws", bad, human("ops"))), "authority before eligibility")
	clock.advance(25 * time.Hour)
	expect(t, ReasonExpired, second(r.Execute("ws", bad, human("ops"))), "expiry before the subject")
	administer(t, r, policy(StageIssue, 2, 1, "result", "erp_draft"))
	expect(t, ReasonSuperseded, second(r.Execute("ws", bad, human("ops"))), "a superseding policy")
	r2, _ := setup(t, 1)
	must(r2.Request("ws", subject("q", "inv", "10"), human("req")))
	must(r2.Decide("ws", "q", approve, human("ann")))
	must(r2.Execute("ws", runOf("e", "q", "inv", "10"), human("ops")))
	must(r2.Request("ws", subject("q2", "inv", "10"), human("req")))
	expect(t, ReasonSuperseded, second(r2.Execute("ws", runOf("e9", "q", "inv", "10"), human("ops"))), "superseded before already executed")
	expect(t, ReasonNotApproved, second(r2.Execute("ws", runOf("e9", "q2", "inv", "10"), human("ops"))), "not approved before already executed")
	must(r2.Decide("ws", "q2", approve, human("ann")))
	expect(t, ReasonAlreadyExecuted, second(r2.Execute("ws", runOf("e9", "q2", "inv", "10"), human("ops"))), "already executed last")
}

func TestPolicyRevisionCheckedAtExecution(t *testing.T) {
	r, _ := setup(t, 1)
	must(r.Request("ws", subject("q", "inv", "10"), human("req")))
	must(r.Decide("ws", "q", approve, human("ann")))
	s := r.workspaces["ws"]
	p := s.policies[StageIssue]
	p.Revision = 7
	s.policies[StageIssue] = p
	expect(t, ReasonPolicyChanged, second(r.Execute("ws", runOf("e", "q", "inv", "10"), human("ops"))), "a revision that moved")
}

func TestReviewGaps(t *testing.T) {
	r, clock := setup(t, 1)
	must(r.Request("ws", bulk("q", "batch"), human("req")))
	must(r.Decide("ws", "q", approve, human("ann")))
	x := must(r.Execute("ws", item("e-a", "q", "inv-a", "300", '1'), human("ops")))
	must(r.Report("ws", x.Execution.ID, Outcome{Result: ResultFailed, Reason: "external_error"}, human("ops")))
	v := must(r.Report("ws", "e-a", Outcome{Result: ResultSucceeded}, human("ops")))
	if v.Outcome.Result != ResultSucceeded || v.Earlier.Reason != "external_error" {
		t.Fatalf("a success after a failure: %+v", v)
	}
	expect(t, "conflict", second(r.Report("ws", "e-a", Outcome{Result: ResultFailed, Reason: "timeout"}, human("ops"))), "a third report")
	expect(t, "ok", second(r.Report("ws", "e-a", Outcome{Result: ResultSucceeded}, human("ops"))), "a repeat of the last report")
	must(r.Request("ws", bulk("q2", "batch"), human("req")))
	if v := must(r.RequestOf("ws", "q")); v.State != StateSuperseded || len(v.Executed) != 1 {
		t.Fatalf("a partly executed bulk request superseded: %+v", v)
	}
	expect(t, ReasonSuperseded, second(r.Execute("ws", item("e-c", "q", "inv-c", "500", '3'), human("ops"))), "a remaining item after supersede")
	expect(t, ReasonAINotAllowed, second(r.Revoke("ws", "g-bob", Actor{ID: "adm1", Kind: KindAssistant})), "an assistant revoking as an administrator")
	expect(t, ReasonAINotAllowed, second(r.Revoke("ws", "g-bob", Actor{ID: "bob", Kind: KindSystem})), "a system revoking as the grantee")
	must(r.Decide("ws", "q2", approve, human("ann")))
	s := r.workspaces["ws"]
	p := s.policies[StageIssue]
	p.Revision = 9
	s.policies[StageIssue] = p
	clock.advance(48 * time.Hour)
	expect(t, ReasonPolicyChanged, second(r.Execute("ws", item("e-b", "q2", "inv-b", "-200", '2'), human("ops"))), "a policy change before expiry")
}
