package protected

import (
	"testing"
	"time"
)

func TestRequestOrderingRefusals(t *testing.T) {
	f := ready(t)
	now := f.clock.read()
	c := account("c1", ref("v1", 'a'), now)
	expect(t, "effective now", second(f.r.Request("w1", c, human("clerk"))), ReasonEffectiveOutOfRange)
	c.Effective = now.Add(30*day + time.Nanosecond)
	expect(t, "beyond lead", second(f.r.Request("w1", c, human("clerk"))), ReasonEffectiveOutOfRange)
	c.Effective = now.Add(30 * day)
	expect(t, "at lead", second(f.r.Request("w1", c, human("clerk"))), "ok")
	if again := must(f.r.Request("w1", c, human("clerk"))); again.Change != c {
		t.Fatal("repeat")
	}
	expect(t, "repeat other actor", second(f.r.Request("w1", c, human("x"))), "conflict")
	changed := c
	changed.After = ref("v9", 'a')
	expect(t, "repeat changed", second(f.r.Request("w1", changed, human("clerk"))), "conflict")
	d := account("c2", ref("v2", 'a'), now.Add(day))
	expect(t, "pending", second(f.r.Request("w1", d, human("clerk"))), ReasonChangePending)
	d.Field = FieldTaxIdentifier
	expect(t, "other field", second(f.r.Request("w1", d, human("clerk"))), "ok")
	e := account("c3", ref("v3", 'a'), now.Add(day))
	e.Subject = "beta"
	e.Instruction = Instruction{Kind: ChannelPhone, Sender: fingerprint('c'), Evidence: "x"}
	expect(t, "other subject", second(f.r.Request("w1", e, human("clerk"))), "ok")
	must(f.r.Withdraw("w1", "c1", human("clerk")))

	g := ready(t)
	g.schedule(t, account("b1", ref("v1", 'a'), start.Add(10*day)))
	expect(t, "earlier", second(g.r.Request("w1", account("b2", ref("v2", 'a'), start.Add(9*day)), human("clerk"))), ReasonOutOfOrder)
	expect(t, "equal", second(g.r.Request("w1", account("b2", ref("v2", 'a'), start.Add(10*day)), human("clerk"))), ReasonOutOfOrder)
	expect(t, "same ref", second(g.r.Request("w1", account("b2", ref("v1", 'a'), start.Add(11*day)), human("clerk"))), ReasonNoChange)
	expect(t, "same id other fingerprint", second(g.r.Request("w1", account("b2", ref("v1", 'b'), start.Add(11*day)), human("clerk"))), "ok")
}

func TestChangeLimit(t *testing.T) {
	f := ready(t)
	for i := 0; i < 6; i++ {
		c := account("c"+string(rune('0'+i)), ref("v", 'a'), start.Add(5*day))
		c.Subject = "s" + string(rune('0'+i))
		c.Instruction = Instruction{Kind: ChannelPostal, Sender: fingerprint('c'), Evidence: "e"}
		must(f.r.Request("w1", c, human("clerk")))
	}
	c := account("c9", ref("v", 'a'), start.Add(5*day))
	expect(t, "limit", second(f.r.Request("w1", c, human("clerk"))), "limit")
}

func TestWithdrawal(t *testing.T) {
	f := ready(t)
	v := must(f.r.Request("w1", account("c1", ref("v1", 'a'), start.Add(5*day)), human("clerk")))
	expect(t, "assistant", second(f.r.Withdraw("w1", "c1", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Withdraw("w1", "c1", system("job"))), ReasonNotHuman)
	w := must(f.r.Withdraw("w1", "c1", human("other")))
	if w.State != StateWithdrawn || w.EndedBy != "other" || !w.Ended.Equal(f.clock.read()) {
		t.Fatalf("withdrawn %+v", w)
	}
	if again := must(f.r.Withdraw("w1", "c1", human("other"))); again.State != StateWithdrawn {
		t.Fatal("repeat")
	}
	expect(t, "other repeat", second(f.r.Withdraw("w1", "c1", human("clerk"))), ReasonNotPending)
	expect(t, "verify withdrawn", second(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), human("checker"))), ReasonNotPending)

	v = must(f.r.Request("w1", account("c2", ref("v1", 'a'), start.Add(5*day)), human("clerk")))
	v = must(f.r.Verify("w1", "c2", confirm("c2", "ch-phone", v.Digest), human("checker")))
	must(f.r.Withdraw("w1", "c2", human("clerk")))
	expect(t, "approve withdrawn", second(f.r.Approve("w1", "c2", approve(v.Verification.Digest), human("boss"))), ReasonNotPending)

	first := f.schedule(t, account("c3", ref("v1", 'a'), start.Add(5*day)))
	second3 := f.schedule(t, account("c4", ref("v2", 'a'), start.Add(6*day)))
	expect(t, "successor scheduled", second(f.r.Withdraw("w1", first.Change.ID, human("clerk"))), ReasonHasSuccessor)
	must(f.r.Withdraw("w1", second3.Change.ID, human("clerk")))
	must(f.r.Request("w1", account("c5", ref("v3", 'a'), start.Add(7*day)), human("clerk")))
	expect(t, "successor pending", second(f.r.Withdraw("w1", first.Change.ID, human("clerk"))), ReasonHasSuccessor)
	must(f.r.Withdraw("w1", "c5", human("clerk")))
	if got := must(f.r.Resolve("w1", "acme", FieldPaymentAccount, start.Add(8*day))); got.Change != "c3" {
		t.Fatalf("after withdrawals %+v", got)
	}
	f.clock.advance(5*day - (f.clock.read().Sub(start)))
	expect(t, "at effective", second(f.r.Withdraw("w1", first.Change.ID, human("clerk"))), ReasonAlreadyEffective)

	g := ready(t)
	s := g.schedule(t, account("c1", ref("v1", 'a'), start.Add(5*day)))
	g.clock.advance(time.Hour)
	w = must(g.r.Withdraw("w1", s.Change.ID, human("clerk")))
	if w.State != StateWithdrawn {
		t.Fatalf("scheduled withdrawn %+v", w)
	}
	expect(t, "resolve withdrawn", second(g.r.Resolve("w1", "acme", FieldPaymentAccount, start.Add(9*day))), "not_found")
	if v := must(g.r.Request("w1", account("c2", ref("v1", 'a'), start.Add(4*day)), human("clerk"))); v.Before != (Ref{}) {
		t.Fatalf("before after withdrawal %+v", v)
	}
}

func TestSuccessorLapsesThenWithdraw(t *testing.T) {
	f := ready(t)
	first := f.schedule(t, account("c1", ref("v1", 'a'), start.Add(10*day)))
	f.clock.advance(time.Hour)
	v := must(f.r.Request("w1", account("c2", ref("v2", 'a'), start.Add(10*day+time.Hour)), human("clerk")))
	must(f.r.Verify("w1", "c2", confirm("c2", "ch-phone", v.Digest), human("checker")))
	expect(t, "pending", second(f.r.Withdraw("w1", first.Change.ID, human("clerk"))), ReasonHasSuccessor)
	f.clock.advance(2*day + time.Nanosecond)
	w := must(f.r.Withdraw("w1", first.Change.ID, human("clerk")))
	if w.State != StateWithdrawn || must(f.r.Of("w1", "c2")).State != StateLapsed {
		t.Fatalf("withdraw after successor lapsed %+v", w)
	}
}

func TestDigestCoverage(t *testing.T) {
	base := View{
		Change:    account("c1", ref("v1", 'a'), start.Add(day)),
		Before:    ref("v0", 'b'),
		Requester: "clerk", Requested: start, Rules: testRules(), Revision: 1,
	}
	want := requestDigest("w1", base)
	mutate := []func(*View){
		func(v *View) { v.Change.ID = "c2" },
		func(v *View) { v.Change.Subject = "s" },
		func(v *View) { v.Change.Field = FieldTaxIdentifier },
		func(v *View) { v.Before.ID = "x" },
		func(v *View) { v.Before.Fingerprint = fingerprint('c') },
		func(v *View) { v.Change.After.ID = "x" },
		func(v *View) { v.Change.After.Fingerprint = fingerprint('c') },
		func(v *View) { v.Change.Provenance = ProvenanceStaff },
		func(v *View) { v.Change.Instruction.Kind = ChannelPhone },
		func(v *View) { v.Change.Instruction.Channel = "x" },
		func(v *View) { v.Change.Instruction.Sender = fingerprint('c') },
		func(v *View) { v.Change.Instruction.Evidence = "x" },
		func(v *View) { v.Change.Effective = v.Change.Effective.Add(1) },
		func(v *View) { v.Requester = "x" },
		func(v *View) { v.Requested = v.Requested.Add(1) },
		func(v *View) { v.Revision = 2 },
		func(v *View) { v.Rules.Approvals = 2 },
		func(v *View) { v.Rules.Validity++ },
		func(v *View) { v.Rules.MaxLead++ },
		func(v *View) { v.Rules.ChannelAge++ },
	}
	for i, m := range mutate {
		v := base
		m(&v)
		if requestDigest("w1", v) == want {
			t.Fatalf("request digest ignores field %d", i)
		}
	}
	if requestDigest("w2", base) == want {
		t.Fatal("workspace")
	}
	cv := VerificationView{Verification: confirm("c1", "ch", want), Verifier: "checker", At: start}
	cw := confirmDigest(want, cv)
	cmut := []func(*VerificationView){
		func(v *VerificationView) { v.Verifier = "x" },
		func(v *VerificationView) { v.Verification.Channel = "x" },
		func(v *VerificationView) { v.Verification.Evidence = "x" },
		func(v *VerificationView) { v.Verification.Outcome = OutcomeDenied },
		func(v *VerificationView) { v.At = v.At.Add(1) },
	}
	for i, m := range cmut {
		v := cv
		m(&v)
		if confirmDigest(want, v) == cw {
			t.Fatalf("confirm digest ignores field %d", i)
		}
	}
	if confirmDigest(fingerprint('0'), cv) == cw || cw == want {
		t.Fatal("confirm digest ignores request")
	}
	if hashOf("ab", "c") == hashOf("a", "bc") {
		t.Fatal("length prefix")
	}
}

func TestRetiredChannelBlocksApproval(t *testing.T) {
	f := ready(t)
	v := must(f.r.Request("w1", account("c1", ref("v1", 'a'), start.Add(5*day)), human("clerk")))
	v = must(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), human("checker")))
	must(f.r.Retire("w1", "ch-phone", human("ops")))
	expect(t, "retired", second(f.r.Approve("w1", "c1", approve(v.Verification.Digest), human("boss"))), ReasonChannelRetired)
	if got := must(f.r.Of("w1", "c1")); got.State != StateConfirmed {
		t.Fatalf("state %+v", got)
	}
}

func TestViewsAreCopies(t *testing.T) {
	f := ready(t)
	v := f.schedule(t, account("c1", ref("v1", 'a'), start.Add(5*day)))
	v.Verification.Verifier = "tampered"
	v.Decisions[0].Approver = "tampered"
	got := must(f.r.Of("w1", "c1"))
	if got.Verification.Verifier != "checker" || got.Decisions[0].Approver != "boss" {
		t.Fatalf("aliasing %+v", got)
	}
	trail := must(f.r.Trail("w1", "acme", FieldPaymentAccount))
	trail[0].Decisions[0].Approver = "tampered"
	state := must(f.r.RulesOf("w1"))
	state.Active.Rules.Approvals = 3
	if must(f.r.RulesOf("w1")).Active.Rules.Approvals != 1 {
		t.Fatal("rules aliasing")
	}
	if must(f.r.Of("w1", "c1")).Decisions[0].Approver != "boss" {
		t.Fatal("trail aliasing")
	}
}

func TestClockNeverMovesBack(t *testing.T) {
	f := ready(t)
	v := must(f.r.Request("w1", account("c1", ref("v1", 'a'), start.Add(5*day)), human("clerk")))
	f.clock.advance(-time.Hour)
	v = must(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), human("checker")))
	if v.Verification.At.Before(v.Requested) {
		t.Fatal("clock moved back")
	}
	local := time.FixedZone("x", 3600)
	c := account("c2", ref("v1", 'a'), start.Add(6*day).In(local))
	c.Field = FieldTaxIdentifier
	got := must(f.r.Request("w1", c, human("clerk")))
	if got.Change.Effective.Location() != time.UTC {
		t.Fatal("not normalised")
	}
	if again := must(f.r.Request("w1", c, human("clerk"))); again.Digest != got.Digest {
		t.Fatal("normalised repeat")
	}
}
