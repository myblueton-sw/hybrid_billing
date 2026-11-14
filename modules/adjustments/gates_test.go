package adjustments

import (
	"testing"
)

func TestValidateOrder(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", false))
	expect(t, RefuseNothingToAdjust, second(r.Validate("ws", "d1", check(v, "0", "100"), system("tax"))), "nothing")
	v = must(r.Add("ws", "d1", v.Revision, item("e1", EntryFixed, "", "", "10"), human("clerk")))
	expect(t, RefuseAINotAllowed, second(r.Validate("ws", "d1", check(v, "0", "110"), assistant("bot"))), "assistant")
	stale := check(v, "0", "110")
	stale.Revision = 1
	expect(t, RefuseStaleRevision, second(r.Validate("ws", "d1", stale, system("tax"))), "stale")
	moved := check(v, "0", "110")
	moved.Digest = digest('9')
	expect(t, RefuseDigestChanged, second(r.Validate("ws", "d1", moved, system("tax"))), "digest")
	expect(t, RefuseFinalMismatch, second(r.Validate("ws", "d1", check(v, "0", "110.01"), system("tax"))), "final")
	expect(t, RefuseFinalMismatch, second(r.Validate("ws", "d1", check(v, "11", "110"), system("tax"))), "final without tax")
	expect(t, "invalid", second(r.Validate("ws", "d1", check(v, "0.001", "110.001"), system("tax"))), "off scale")
	expect(t, RefuseFinalMismatch, second(r.Validate("ws", "d1", check(v, "0", "-1"), system("tax"))), "negative final is a number")
	expect(t, "invalid", second(r.Validate("ws", "d1", check(v, "0.001", "110"), system("tax"))), "off scale tax alone")
	expect(t, "invalid", second(r.Validate("ws", "d1", check(v, "0", "110.001"), system("tax"))), "off scale final alone")
	expect(t, "not_found", second(r.Validate("ws", "zz", check(v, "0", "110"), system("tax"))), "unknown")
	expect(t, "not_found", second(r.Validate("other", "d1", check(v, "0", "110"), system("tax"))), "other workspace")
	ok := must(r.Validate("ws", "d1", check(v, "11", "121"), human("reviewer")))
	if ok.Check == nil || ok.Check.Check.TaxAmount != "11" || ok.Subject == "" || !ok.Check.Recorded.Equal(start) {
		t.Fatalf("check %+v", ok)
	}
	again := must(r.Validate("ws", "d1", check(v, "11", "121"), human("reviewer")))
	if again.Subject != ok.Subject {
		t.Fatal("replay subject")
	}
	expect(t, "conflict", second(r.Validate("ws", "d1", check(v, "11", "121"), human("other"))), "replay by another")
	other := check(v, "11", "121")
	other.Tax = digest('7')
	expect(t, "conflict", second(r.Validate("ws", "d1", other, human("reviewer"))), "different check")
}

func TestSubjectCoversCheckAndBase(t *testing.T) {
	subjects := map[string]bool{}
	for index, c := range []Check{
		{Tax: digest('c'), Allocation: digest('d'), TaxAmount: "0", Final: "105"},
		{Tax: digest('e'), Allocation: digest('d'), TaxAmount: "0", Final: "105"},
		{Tax: digest('c'), Allocation: digest('e'), TaxAmount: "0", Final: "105"},
		{Tax: digest('c'), Allocation: digest('d'), TaxAmount: "1", Final: "106"},
	} {
		r, _ := setup(t)
		v := opened(t, r, sample("d1", index == 0))
		v = must(r.Add("ws", "d1", v.Revision, item("e1", EntryFixed, "", "", "5"), human("clerk")))
		c.Revision, c.Digest = v.Revision, v.Digest
		subjects[must(r.Validate("ws", "d1", c, system("tax"))).Subject] = true
	}
	if len(subjects) != 4 {
		t.Fatalf("subjects %d", len(subjects))
	}
}

func TestReleaseOrder(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", false))
	v = must(r.Add("ws", "d1", v.Revision, item("e1", EntryFixed, "", "", "5"), assistant("bot")))
	q := Release{Request: "r1", Revision: v.Revision, Subject: digest('5'), Base: v.Draft.Base, Approval: "ap1", Approvers: []Actor{human("ann")}}
	expect(t, RefuseAINotAllowed, r.Ready("ws", "d1", q, assistant("bot")), "assistant releases")
	expect(t, RefuseNotValidated, r.Ready("ws", "d1", q, system("runner")), "not validated")
	stale := q
	stale.Revision = 1
	expect(t, RefuseStaleRevision, r.Ready("ws", "d1", stale, system("runner")), "stale")
	v = must(r.Validate("ws", "d1", check(v, "0", "105"), human("checker")))
	expect(t, RefuseDigestChanged, r.Ready("ws", "d1", q, system("runner")), "digest")
	q.Subject = v.Subject
	expect(t, "ok", r.Ready("ws", "d1", q, system("runner")), "ready")
	for _, a := range []Actor{assistant("ann"), system("ann")} {
		bad := q
		bad.Approvers = []Actor{human("bob"), a}
		expect(t, RefuseAINotAllowed, r.Ready("ws", "d1", bad, system("runner")), "approver kind "+a.Kind)
	}
	for _, id := range []string{"writer", "bot", "checker"} {
		bad := q
		bad.Approvers = []Actor{human("bob"), human(id)}
		expect(t, RefuseSelfApproval, r.Ready("ws", "d1", bad, system("runner")), "self "+id)
	}
	expect(t, "not_found", r.Ready("ws", "zz", q, system("runner")), "unknown")
	expect(t, "not_found", r.Ready("other", "d1", q, system("runner")), "other workspace")
	if must(r.Of("ws", "d1")).State != StateOpen {
		t.Fatal("ready recorded something")
	}
	must(r.Release("ws", "d1", q, human("ann")))
}

func TestRemoverCannotApprove(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", false))
	v = must(r.Add("ws", "d1", v.Revision, item("e1", EntryFixed, "", "", "5"), human("clerk")))
	v = must(r.Add("ws", "d1", v.Revision, item("e2", EntryFixed, "", "", "6"), human("clerk")))
	v = must(r.Remove("ws", "d1", v.Revision, "e2", human("lead")))
	v = must(r.Validate("ws", "d1", check(v, "0", "105"), system("tax")))
	expect(t, RefuseSelfApproval, r.Ready("ws", "d1", release("r1", v, human("lead")), system("runner")), "remover")
	expect(t, RefuseSelfApproval, r.Ready("ws", "d1", release("r1", v, human("clerk")), system("runner")), "author")
	expect(t, "ok", r.Ready("ws", "d1", release("r1", v, human("ann")), system("runner")), "other")
}

func TestReferencesAreSingleUse(t *testing.T) {
	r, _ := setup(t)
	a := ready(t, r, "d1", item("e1", EntryFixed, "", "", "5"))
	d := sample("d2", false)
	d.Base.Run = "run-2"
	b := opened(t, r, d)
	b = must(r.Add("ws", "d2", b.Revision, item("e1", EntryFixed, "", "", "5"), human("clerk")))
	b = must(r.Validate("ws", "d2", check(b, "0", "105"), system("tax")))
	must(r.Release("ws", "d1", release("r1", a, human("ann")), system("runner")))
	q := release("r1", b, human("ann"))
	q.Approval = "fresh"
	expect(t, "conflict", r.Ready("ws", "d2", q, system("runner")), "request reused")
	q = release("r2", b, human("ann"))
	q.Approval = "ap-r1"
	expect(t, "conflict", r.Ready("ws", "d2", q, system("runner")), "approval reused")
	expect(t, "ok", second(r.Release("ws", "d2", release("r2", b, human("ann")), system("runner"))), "fresh references")
	expect(t, "ok", r.Ready("ws", "d2", release("r2", b, human("ann")), system("runner")), "ready on replay")
}

func TestReleaseRefusesArithmetic(t *testing.T) {
	r, _ := setup(t)
	v := ready(t, r, "d1", item("e1", EntryFixed, "", "", "5"))
	d := must(r.find("ws", "d1"))
	d.result.amounts[0] = d.result.total.Minus(d.result.total)
	expect(t, RefuseZeroDifference, r.Ready("ws", "d1", release("r1", v, human("ann")), system("runner")), "zero")
	d.result.amounts[0] = d.result.total
	d.result.total = d.result.total.Minus(d.result.total).Minus(d.result.amounts[0])
	expect(t, RefuseNegativeTotal, r.Ready("ws", "d1", release("r1", v, human("ann")), system("runner")), "negative")
	d.result.amounts = nil
	expect(t, RefuseNothingToAdjust, r.Ready("ws", "d1", release("r1", v, human("ann")), system("runner")), "nothing")
}
