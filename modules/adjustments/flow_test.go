package adjustments

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
)

type flow struct {
	t       *testing.T
	adjust  *Registry
	approve *approvals.Registry
}

func ahuman(id string) approvals.Actor { return approvals.Actor{ID: id, Kind: approvals.KindHuman} }

func newFlow(t *testing.T) *flow {
	clock := &testClock{now: start}
	ap, err := approvals.NewRegistry(testProfile(t), approvals.Limits{Workspaces: 1, Admins: 2, Proposals: 10, Authorities: 4, Requests: 10, Items: 1, Executions: 10}, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	must(ap.Open("ws", []string{"adm1", "adm2"}, ahuman("installer")))
	steps := []approvals.Proposal{
		{ID: "p1", Kind: approvals.ProposePolicy, Policy: approvals.Policy{Stage: approvals.StageCharge, Revision: 1, Approvals: 1, Required: []string{"result"}, Validity: 24 * time.Hour}},
		{ID: "p2", Kind: approvals.ProposeGrant, Authority: approvals.Authority{ID: "g-ann", Approver: "ann", Stage: approvals.StageCharge, Currency: "USD", Limit: "50", From: "2026-01-01"}},
	}
	for _, p := range steps {
		must(ap.Propose("ws", p, ahuman("adm1")))
		must(ap.Confirm("ws", p.ID, ahuman("adm2")))
	}
	adj, err := NewRegistry(testProfile(t), testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return &flow{t: t, adjust: adj, approve: ap}
}

func (f *flow) request(id string, v View) approvals.RequestView {
	return must(f.approve.Request("ws", approvals.Subject{ID: id, Stage: approvals.StageCharge, Target: target(v), Author: "clerk",
		Digests: map[string]string{"result": v.Subject}, Amount: v.Gross, Currency: v.Draft.Base.Currency}, ahuman("clerk")))
}

func execution(id, request string, v View) approvals.Execution {
	return approvals.Execution{ID: id, Request: request, Stage: approvals.StageCharge, Target: target(v),
		Digests: map[string]string{"result": v.Subject}, Amount: v.Gross, Currency: v.Draft.Base.Currency}
}

func target(v View) string { return "adjustment:" + v.Draft.ID + ":" + strconv.Itoa(v.Revision) }

func deciders(view approvals.RequestView) []Actor {
	var out []Actor
	for _, d := range view.Decisions {
		out = append(out, human(d.Actor))
	}
	return out
}

func refusal(err error) string {
	var a approvals.Refusal
	if errors.As(err, &a) {
		return a.Reason
	}
	return kind(err)
}

func TestFlowThroughApprovals(t *testing.T) {
	f := newFlow(t)
	v := ready(t, f.adjust, "d1", item("e1", EntryTargetTotal, "", "120", ""))
	f.request("q1", v)
	decided := must(f.approve.Decide("ws", "q1", approvals.Decision{Verdict: approvals.VerdictApprove}, ahuman("ann")))
	q := Release{Request: "rel-1", Revision: v.Revision, Subject: v.Subject, Base: v.Draft.Base, Approval: "x1", Approvers: deciders(decided)}
	if len(q.Approvers) != 1 || q.Approvers[0] != human("ann") {
		t.Fatalf("approvers %+v", q.Approvers)
	}
	expect(t, "ok", f.adjust.Ready("ws", "d1", q, system("runner")), "ready")
	must(f.approve.Execute("ws", execution("x1", "q1", v), approvals.Actor{ID: "runner", Kind: approvals.KindSystem}))
	out := must(f.adjust.Release("ws", "d1", q, system("runner")))
	if out.Total != "120" || out.Disposition != DispositionNewRun || out.Release.Approval != "x1" {
		t.Fatalf("outcome %+v", out)
	}
}

func TestFlowChangedDraftCannotUseOldApproval(t *testing.T) {
	f := newFlow(t)
	v := ready(t, f.adjust, "d1", item("e1", EntryTargetTotal, "", "120", ""))
	f.request("q1", v)
	decided := must(f.approve.Decide("ws", "q1", approvals.Decision{Verdict: approvals.VerdictApprove}, ahuman("ann")))
	changed := must(f.adjust.Add("ws", "d1", v.Revision, item("e2", EntryFixed, "l2", "", "3"), human("clerk")))
	changed = must(f.adjust.Validate("ws", "d1", check(changed, "0", changed.Total), system("tax-engine")))
	if got := refusal(second(f.approve.Execute("ws", execution("x1", "q1", changed), approvals.Actor{ID: "runner", Kind: approvals.KindSystem}))); got != approvals.ReasonSubjectChanged {
		t.Fatalf("execute with new subject under old request: %s", got)
	}
	old := Release{Request: "rel-1", Revision: v.Revision, Subject: v.Subject, Base: v.Draft.Base, Approval: "x1", Approvers: deciders(decided)}
	expect(t, RefuseStaleRevision, f.adjust.Ready("ws", "d1", old, system("runner")), "old release")
	old.Revision = changed.Revision
	expect(t, RefuseDigestChanged, f.adjust.Ready("ws", "d1", old, system("runner")), "old subject")
	f.request("q2", changed)
	if len(must(f.approve.RequestOf("ws", "q1")).Executed) != 0 {
		t.Fatal("old request executed")
	}
	decided = must(f.approve.Decide("ws", "q2", approvals.Decision{Verdict: approvals.VerdictApprove}, ahuman("ann")))
	q := Release{Request: "rel-2", Revision: changed.Revision, Subject: changed.Subject, Base: changed.Draft.Base, Approval: "x2", Approvers: deciders(decided)}
	expect(t, "ok", f.adjust.Ready("ws", "d1", q, system("runner")), "ready")
	must(f.approve.Execute("ws", execution("x2", "q2", changed), approvals.Actor{ID: "runner", Kind: approvals.KindSystem}))
	expect(t, "ok", second(f.adjust.Release("ws", "d1", q, system("runner"))), "release")
}

func TestFlowReadyRefusesBeforeConsuming(t *testing.T) {
	f := newFlow(t)
	v := ready(t, f.adjust, "d1", item("e1", EntryFixed, "", "", "5"))
	f.request("q1", v)
	must(f.approve.Decide("ws", "q1", approvals.Decision{Verdict: approvals.VerdictApprove}, ahuman("ann")))
	q := Release{Request: "rel-1", Revision: v.Revision, Subject: v.Subject, Base: v.Draft.Base, Approval: "x1", Approvers: []Actor{human("clerk")}}
	expect(t, RefuseSelfApproval, f.adjust.Ready("ws", "d1", q, system("runner")), "self approval found before execute")
	if len(must(f.approve.RequestOf("ws", "q1")).Executed) != 0 {
		t.Fatal("approval consumed")
	}
}

func TestFlowLateRefusalIsReported(t *testing.T) {
	f := newFlow(t)
	a := ready(t, f.adjust, "d1", item("e1", EntryFixed, "", "", "5"))
	b := ready(t, f.adjust, "d2", item("e1", EntryFixed, "", "", "6"))
	f.request("q2", b)
	decided := must(f.approve.Decide("ws", "q2", approvals.Decision{Verdict: approvals.VerdictApprove}, ahuman("ann")))
	q := Release{Request: "rel-2", Revision: b.Revision, Subject: b.Subject, Base: b.Draft.Base, Approval: "x2", Approvers: deciders(decided)}
	expect(t, "ok", f.adjust.Ready("ws", "d2", q, system("runner")), "ready")
	runner := approvals.Actor{ID: "runner", Kind: approvals.KindSystem}
	must(f.approve.Execute("ws", execution("x2", "q2", b), runner))
	must(f.adjust.Release("ws", "d1", release("rel-1", a, human("bob")), system("other")))
	err := second(f.adjust.Release("ws", "d2", q, system("runner")))
	expect(t, RefuseBaseConsumed, err, "late refusal")
	reported := must(f.approve.Report("ws", "x2", approvals.Outcome{Result: approvals.ResultFailed, Reason: "refused_downstream"}, runner))
	if reported.Outcome.Result != approvals.ResultFailed || reported.Outcome.Reason != "refused_downstream" {
		t.Fatalf("report %+v", reported)
	}
}

func TestFlowRecoversAfterLateStaleRefusal(t *testing.T) {
	f := newFlow(t)
	runner := approvals.Actor{ID: "runner", Kind: approvals.KindSystem}
	v := ready(t, f.adjust, "d1", item("e1", EntryFixed, "", "", "5"))
	f.request("q1", v)
	decided := must(f.approve.Decide("ws", "q1", approvals.Decision{Verdict: approvals.VerdictApprove}, ahuman("ann")))
	q := Release{Request: "rel-1", Revision: v.Revision, Subject: v.Subject, Base: v.Draft.Base, Approval: "x1", Approvers: deciders(decided)}
	expect(t, "ok", f.adjust.Ready("ws", "d1", q, system("runner")), "ready")
	must(f.approve.Execute("ws", execution("x1", "q1", v), runner))
	changed := must(f.adjust.Add("ws", "d1", v.Revision, item("e2", EntryFixed, "", "", "1"), assistant("bot")))
	expect(t, RefuseStaleRevision, second(f.adjust.Release("ws", "d1", q, system("runner"))), "late stale refusal")
	must(f.approve.Report("ws", "x1", approvals.Outcome{Result: approvals.ResultFailed, Reason: "refused_downstream"}, runner))
	changed = must(f.adjust.Validate("ws", "d1", check(changed, "0", changed.Total), system("tax-engine")))
	f.request("q2", changed)
	decided = must(f.approve.Decide("ws", "q2", approvals.Decision{Verdict: approvals.VerdictApprove}, ahuman("ann")))
	q = Release{Request: "rel-2", Revision: changed.Revision, Subject: changed.Subject, Base: changed.Draft.Base, Approval: "x2", Approvers: deciders(decided)}
	expect(t, "ok", f.adjust.Ready("ws", "d1", q, system("runner")), "ready again")
	must(f.approve.Execute("ws", execution("x2", "q2", changed), runner))
	out := must(f.adjust.Release("ws", "d1", q, system("runner")))
	if out.Total != "106" || out.Release.Approval != "x2" {
		t.Fatalf("outcome %+v", out)
	}
	if got := refusal(second(f.approve.Execute("ws", execution("x3", "q2", changed), runner))); got != approvals.ReasonAlreadyExecuted {
		t.Fatalf("second execution: %s", got)
	}
}
