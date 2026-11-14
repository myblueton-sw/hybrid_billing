package approvals

import (
	"testing"
	"time"
)

func TestApprovedSubjectExecutesOnce(t *testing.T) {
	r, clock := setup(t, 1)
	req := must(r.Request("ws", subject("q1", "inv-1", "800.50"), Actor{ID: "bot", Kind: KindAssistant}))
	if req.State != StatePending || req.Policy != 1 || len(req.Digest) != 64 || req.Subject.Amount != "800.5" {
		t.Fatalf("request: %+v", req)
	}
	expect(t, ReasonNotApproved, second(r.Execute("ws", runOf("e0", "q1", "inv-1", "800.5"), human("ops"))), "execution before approval")
	expect(t, ReasonNoAuthority, second(r.Decide("ws", "q1", approve, human("bob"))), "a limit below the amount")
	clock.advance(time.Minute)
	v := must(r.Decide("ws", "q1", approve, human("ann")))
	if v.State != StateApproved || !v.ApprovedAt.Equal(start.Add(time.Minute)) || len(v.Decisions) != 1 {
		t.Fatalf("approved: %+v", v)
	}
	x := must(r.Execute("ws", runOf("e1", "q1", "inv-1", "800.5"), human("ops")))
	if x.Actor != "ops" || len(x.Digest) != 64 {
		t.Fatal(x)
	}
	expect(t, ReasonAlreadyExecuted, second(r.Execute("ws", runOf("e2", "q1", "inv-1", "800.5"), human("ops"))), "a second execution")
	expect(t, "ok", second(r.Execute("ws", runOf("e1", "q1", "inv-1", "800.5"), human("ops"))), "a replay")
	expect(t, "conflict", second(r.Execute("ws", runOf("e1", "q1", "inv-1", "800.5"), human("other"))), "a replay by another actor")
	must(r.Request("ws", subject("q2", "inv-1", "800.5"), human("writer2")))
	must(r.Decide("ws", "q2", approve, human("ann")))
	expect(t, ReasonAlreadyExecuted, second(r.Execute("ws", runOf("e3", "q2", "inv-1", "800.5"), human("ops"))), "a new request for an executed target")
}

func TestBlockedRightAfterApproval(t *testing.T) {
	cases := map[string]func(r *Registry, clock *testClock) (Execution, string){
		"a concurrent new request": func(r *Registry, _ *testClock) (Execution, string) {
			must(r.Request("ws", subject("later", "inv", "10"), human("writer")))
			return runOf("e", "q", "inv", "10"), ReasonSuperseded
		},
		"a withdrawal": func(r *Registry, _ *testClock) (Execution, string) {
			must(r.Withdraw("ws", "q", human("req")))
			return runOf("e", "q", "inv", "10"), ReasonWithdrawn
		},
		"an ERP draft change with no new request": func(r *Registry, _ *testClock) (Execution, string) {
			e := runOf("e", "q", "inv", "10")
			e.Digests["erp_draft"] = digest('c')
			return e, ReasonSubjectChanged
		},
		"another amount": func(r *Registry, _ *testClock) (Execution, string) {
			return runOf("e", "q", "inv", "10.01"), ReasonSubjectChanged
		},
		"another target": func(r *Registry, _ *testClock) (Execution, string) {
			return runOf("e", "q", "inv-2", "10"), ReasonSubjectChanged
		},
		"another stage": func(r *Registry, _ *testClock) (Execution, string) {
			e := runOf("e", "q", "inv", "10")
			e.Stage = StageDeliver
			return e, ReasonSubjectChanged
		},
		"a missing digest": func(r *Registry, _ *testClock) (Execution, string) {
			e := runOf("e", "q", "inv", "10")
			delete(e.Digests, "erp_draft")
			return e, ReasonSubjectChanged
		},
		"a revoked authority": func(r *Registry, _ *testClock) (Execution, string) {
			must(r.Revoke("ws", "g-ann", human("ann")))
			return runOf("e", "q", "inv", "10"), ReasonAuthorityLost
		},
		"a policy change": func(r *Registry, _ *testClock) (Execution, string) {
			administer(t, r, policy(StageIssue, 2, 1, "result", "erp_draft", "fx"))
			return runOf("e", "q", "inv", "10"), ReasonSuperseded
		},
		"expiry": func(r *Registry, clock *testClock) (Execution, string) {
			clock.advance(24*time.Hour + time.Nanosecond)
			return runOf("e", "q", "inv", "10"), ReasonExpired
		},
	}
	for name, change := range cases {
		r, clock := setup(t, 1)
		must(r.Request("ws", subject("q", "inv", "10"), human("req")))
		must(r.Decide("ws", "q", approve, human("ann")))
		e, want := change(r, clock)
		expect(t, want, second(r.Execute("ws", e, human("ops"))), name)
	}
	r, clock := setup(t, 1)
	must(r.Request("ws", subject("q", "inv", "10"), human("req")))
	must(r.Decide("ws", "q", approve, human("ann")))
	clock.advance(24 * time.Hour)
	expect(t, "ok", second(r.Execute("ws", runOf("e", "q", "inv", "10"), human("ops"))), "at the end of the validity")
}

func TestSeparationOfDuties(t *testing.T) {
	r, _ := setup(t, 2)
	must(r.Request("ws", subject("q", "inv", "-400"), human("ann")))
	expect(t, ReasonSelfApproval, second(r.Decide("ws", "q", approve, human("ann"))), "the requester")
	expect(t, ReasonSelfApproval, second(r.Decide("ws", "q", approve, human("writer"))), "the author")
	expect(t, ReasonAINotAllowed, second(r.Decide("ws", "q", approve, Actor{ID: "bob", Kind: KindAssistant})), "an assistant")
	expect(t, ReasonAINotAllowed, second(r.Decide("ws", "q", approve, Actor{ID: "bob", Kind: KindSystem})), "a system")
	expect(t, ReasonNoAuthority, second(r.Decide("ws", "q", approve, human("carl"))), "no authority")
	v := must(r.Decide("ws", "q", approve, human("bob")))
	if v.State != StatePending {
		t.Fatal("one of two approvals approved")
	}
	expect(t, "ok", second(r.Decide("ws", "q", approve, human("bob"))), "a repeat decision")
	expect(t, "conflict", second(r.Decide("ws", "q", Decision{Verdict: VerdictReject, Reason: "other"}, human("bob"))), "another decision by the same approver")
	administer(t, r, grant("g-dan", "dan", StageIssue, "400"))
	v = must(r.Decide("ws", "q", approve, human("dan")))
	if v.State != StateApproved || len(v.Decisions) != 2 {
		t.Fatalf("two approvals: %+v", v)
	}
	expect(t, ReasonNotPending, second(r.Decide("ws", "q", Decision{Verdict: VerdictReject, Reason: "other"}, human("eve"))), "a late reject")
	must(r.Request("ws", subject("q2", "inv2", "1"), human("req")))
	must(r.Decide("ws", "q2", Decision{Verdict: VerdictReject, Reason: "missing_evidence"}, human("bob")))
	expect(t, ReasonNotPending, second(r.Decide("ws", "q2", approve, human("dan"))), "approve after a reject")
	expect(t, ReasonNotApproved, second(r.Execute("ws", runOf("e", "q2", "inv2", "1"), human("ops"))), "execute a rejected request")
}

func TestAdministrationNeedsTwo(t *testing.T) {
	r, _ := setup(t, 1)
	weaker := policy(StageIssue, 2, 1, "result")
	weaker.ID = "w"
	must(r.Propose("ws", weaker, human("adm1")))
	expect(t, ReasonNotAdmin, second(r.Confirm("ws", "w", human("adm1"))), "a proposer confirming alone")
	expect(t, ReasonNotAdmin, second(r.Confirm("ws", "w", human("ann"))), "a non-administrator confirming")
	expect(t, ReasonNotAdmin, second(r.Propose("ws", Proposal{ID: "x", Kind: ProposeGrant, Authority: Authority{ID: "g-x", Approver: "ann", Stage: StageIssue, Currency: "USD", Limit: "9", From: "2026-01-01"}}, human("ann"))), "a non-administrator proposing")
	self := grant("g-self", "adm1", StageIssue, "99")
	self.ID = "self"
	expect(t, ReasonGranteeActs, second(r.Propose("ws", self, human("adm1"))), "a grant to its proposer")
	self.ID = "self2"
	self.Authority.Approver = "adm2"
	must(r.Propose("ws", self, human("adm1")))
	expect(t, ReasonGranteeActs, second(r.Confirm("ws", "self2", human("adm2"))), "a grantee confirming its own grant")
	expect(t, ReasonAINotAllowed, second(r.Propose("ws", Proposal{ID: "ai", Kind: ProposeAddAdmin, Admin: "bot"}, Actor{ID: "adm1", Kind: KindAssistant})), "an assistant proposing")
	remove := Proposal{ID: "rm", Kind: ProposeRemoveAdmin, Admin: "adm1"}
	must(r.Propose("ws", remove, human("adm2")))
	expect(t, ReasonTooFewAdmins, second(r.Confirm("ws", "rm", human("adm1"))), "fewer than two administrators")
	must(r.Propose("ws", Proposal{ID: "add", Kind: ProposeAddAdmin, Admin: "adm3"}, human("adm2")))
	must(r.Confirm("ws", "add", human("adm1")))
	expect(t, "ok", second(r.Confirm("ws", "add", human("adm1"))), "a repeat confirmation")
	expect(t, "rejected", second(r.Confirm("ws", "add", human("adm3"))), "a second confirmer")
	must(r.Confirm("ws", "rm", human("adm3")))
	expect(t, ReasonStaleProposal, second(r.Confirm("ws", "w", human("adm3"))), "a removed proposer's proposal")
	if admins := must(r.Admins("ws")); len(admins) != 2 || admins[0] != "adm2" || admins[1] != "adm3" {
		t.Errorf("admins: %v", admins)
	}
	stale := policy(StageIssue, 3, 1, "result")
	stale.ID = "stale"
	must(r.Propose("ws", stale, human("adm2")))
	expect(t, ReasonStaleProposal, second(r.Confirm("ws", "stale", human("adm3"))), "a revision that skips one")
	expect(t, ReasonNotAdmin, second(r.Revoke("ws", "g-bob", human("ann"))), "another approver revoking")
	must(r.Revoke("ws", "g-bob", human("adm3")))
	expect(t, "ok", second(r.Revoke("ws", "g-bob", human("adm3"))), "a repeat revocation")
	expect(t, "rejected", second(r.Revoke("ws", "g-bob", human("adm2"))), "a second revoker")
	expect(t, ReasonNotAdmin, second(r.Propose("ws", Proposal{ID: "late", Kind: ProposeAddAdmin, Admin: "x"}, human("adm1"))), "a removed administrator")
}
