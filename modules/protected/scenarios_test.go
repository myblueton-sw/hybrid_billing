package protected

import (
	"errors"
	"testing"
	"time"
)

func newEmail(id string, effective time.Time) Change {
	c := account(id, ref("vault:new", '7'), effective)
	c.Instruction = Instruction{Kind: ChannelEmail, Sender: fingerprint('e'), Evidence: "mail:" + id}
	return c
}

func TestNewEmailAloneNeverApplies(t *testing.T) {
	f := ready(t)
	effective := start.Add(3 * day)
	v := must(f.r.Request("w1", newEmail("c1", effective), human("clerk")))
	if v.State != StateRequested || v.Before != (Ref{}) {
		t.Fatalf("view %+v", v)
	}
	expect(t, "approve before confirmation", second(f.r.Approve("w1", "c1", approve(v.Digest), human("boss"))), ReasonNotConfirmed)
	f.clock.advance(5 * day)
	expect(t, "resolve", second(f.r.Resolve("w1", "acme", FieldPaymentAccount, start.Add(10*day))), "not_found")
	got := must(f.r.Of("w1", "c1"))
	if got.State != StateLapsed || !got.Ended.Equal(effective) || got.EndedBy != "" {
		t.Fatalf("lapse %+v", got)
	}
}

func TestLLMGeneratedAccountRefused(t *testing.T) {
	f := ready(t)
	c := account("c1", ref("vault:new", '7'), start.Add(3*day))
	c.Provenance = ProvenanceAssistant
	expect(t, "llm provenance", second(f.r.Request("w1", c, human("clerk"))), ReasonLLMGenerated)
	c.Provenance = ProvenanceStaff
	expect(t, "assistant", second(f.r.Request("w1", c, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Request("w1", c, system("job"))), ReasonNotHuman)
	v := must(f.r.Request("w1", c, human("clerk")))
	expect(t, "assistant verifies", second(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), assistant("bot"))), ReasonAINotAllowed)
	v = must(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), human("checker")))
	expect(t, "assistant approves", second(f.r.Approve("w1", "c1", approve(v.Verification.Digest), assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system approves", second(f.r.Approve("w1", "c1", approve(v.Verification.Digest), system("job"))), ReasonNotHuman)
}

func TestConfirmedAndApprovedChangeAppliesFromEffectiveOnly(t *testing.T) {
	f := ready(t)
	first := f.schedule(t, account("c1", ref("vault:old", '8'), start.Add(3*day)))
	if first.State != StateScheduled || first.Before != (Ref{}) {
		t.Fatalf("baseline %+v", first)
	}
	f.clock.advance(2 * day)
	effective := start.Add(10 * day)
	v := must(f.r.Request("w1", newEmail("c2", effective), human("clerk")))
	if v.Before != ref("vault:old", '8') {
		t.Fatalf("before %+v", v.Before)
	}
	v = must(f.r.Verify("w1", "c2", confirm("c2", "ch-phone", v.Digest), human("checker")))
	if v.State != StateConfirmed || v.Verification.Digest == "" {
		t.Fatalf("confirmed %+v", v)
	}
	v = must(f.r.Approve("w1", "c2", approve(v.Verification.Digest), human("boss")))
	if v.State != StateScheduled || !v.Scheduled.Equal(f.clock.read()) {
		t.Fatalf("scheduled %+v", v)
	}
	at := func(when time.Time) Resolution {
		return must(f.r.Resolve("w1", "acme", FieldPaymentAccount, when))
	}
	if got := at(effective.Add(-time.Nanosecond)); got.Change != "c1" || got.Ref != ref("vault:old", '8') {
		t.Fatalf("before effective %+v", got)
	}
	if got := at(effective); got.Change != "c2" || got.Ref != ref("vault:new", '7') || !got.Effective.Equal(effective) {
		t.Fatalf("at effective %+v", got)
	}
	expect(t, "before baseline", second(f.r.Resolve("w1", "acme", FieldPaymentAccount, start.Add(3*day-time.Nanosecond))), "not_found")
	trail := must(f.r.Trail("w1", "acme", FieldPaymentAccount))
	if len(trail) != 2 || trail[0].Change.ID != "c1" || trail[1].Change.ID != "c2" {
		t.Fatalf("trail %+v", trail)
	}
}

func TestConfirmationChannelControls(t *testing.T) {
	f := ready(t)
	f.channel(t, "w1", Channel{ID: "ch-other", Subject: "beta", Kind: ChannelPhone, Ref: ref("vault:beta", 'c'), Evidence: "contract:2"})
	must(f.r.Register("w1", Channel{ID: "ch-pending", Subject: "acme", Kind: ChannelPortal, Ref: ref("vault:portal", 'd'), Evidence: "contract:3"}, human("onboard")))
	f.channel(t, "w1", Channel{ID: "ch-old", Subject: "acme", Kind: ChannelPostal, Ref: ref("vault:post", 'f'), Evidence: "contract:4"})
	must(f.r.Retire("w1", "ch-old", human("ops")))
	c := account("c1", ref("vault:new", '7'), start.Add(5*day))
	v := must(f.r.Request("w1", c, human("clerk")))
	f.channel(t, "w1", Channel{ID: "ch-new", Subject: "acme", Kind: ChannelPhone, Ref: ref("vault:phone2", '1'), Evidence: "contract:5"})
	verify := func(channel string, actor Actor) error {
		return second(f.r.Verify("w1", "c1", confirm("c1", channel, v.Digest), actor))
	}
	expect(t, "requester verifies", verify("ch-phone", human("clerk")), ReasonSelfVerification)
	expect(t, "missing channel", verify("ch-none", human("checker")), ReasonUnknownChannel)
	expect(t, "other subject", verify("ch-other", human("checker")), ReasonUnknownChannel)
	expect(t, "pending channel", verify("ch-pending", human("checker")), ReasonChannelNotAdmitted)
	expect(t, "instruction channel", verify("ch-mail", human("checker")), ReasonSameChannel)
	expect(t, "retired", verify("ch-old", human("checker")), ReasonChannelRetired)
	expect(t, "too new", verify("ch-new", human("checker")), ReasonChannelTooNew)
	expect(t, "custodian verifies", verify("ch-phone", human("onboard2")), ReasonSelfChannel)
	expect(t, "registrar verifies", verify("ch-phone", human("onboard")), ReasonSelfChannel)
	bad := confirm("c1", "ch-phone", fingerprint('0'))
	expect(t, "digest", second(f.r.Verify("w1", "c1", bad, human("checker"))), ReasonDigestChanged)
	if got := must(f.r.Of("w1", "c1")); got.State != StateRequested || got.Verification != nil {
		t.Fatalf("refusals changed state %+v", got)
	}
}

func TestSenderFingerprintMatchesChannel(t *testing.T) {
	f := ready(t)
	c := account("c1", ref("vault:new", '7'), start.Add(5*day))
	c.Instruction = Instruction{Kind: ChannelPhone, Sender: fingerprint('a'), Evidence: "call:1"}
	expect(t, "undeclared", second(f.r.Request("w1", c, human("clerk"))), ReasonUndeclaredChannel)
	c.Instruction.Channel = "ch-phone"
	c.Instruction.Sender = fingerprint('9')
	expect(t, "fingerprint mismatch", second(f.r.Request("w1", c, human("clerk"))), ReasonUnknownChannel)
	c.Instruction.Sender = fingerprint('a')
	c.Instruction.Kind = ChannelEmail
	expect(t, "kind mismatch", second(f.r.Request("w1", c, human("clerk"))), ReasonUnknownChannel)
	c.Instruction = Instruction{Kind: ChannelInPerson, Evidence: "visit:1"}
	v := must(f.r.Request("w1", c, human("clerk")))
	must(f.r.Verify("w1", "c1", confirm("c1", "ch-mail", v.Digest), human("checker")))
	f2 := ready(t)
	must(f2.r.Register("w1", Channel{ID: "ch-x", Subject: "acme", Kind: ChannelEmail, Ref: ref("vault:x", 'e'), Evidence: "x"}, human("onboard")))
	expect(t, "pending channel sender", second(f2.r.Request("w1", newEmail("c1", start.Add(5*day)), human("clerk"))), ReasonUndeclaredChannel)
	f3 := ready(t)
	f3.channel(t, "w1", Channel{ID: "ch-y", Subject: "acme", Kind: ChannelPhone, Ref: ref("vault:y", 'b'), Evidence: "y"})
	f3.clock.advance(2 * day)
	v = must(f3.r.Request("w1", account("c1", ref("vault:new", '7'), start.Add(9*day)), human("clerk")))
	expect(t, "same fingerprint other channel", second(f3.r.Verify("w1", "c1", confirm("c1", "ch-y", v.Digest), human("checker"))), ReasonSameChannel)
}

func TestApprovalSeparation(t *testing.T) {
	f := ready(t)
	rules := testRules()
	rules.Approvals = 2
	f.rules(t, "w1", rules)
	v := must(f.r.Request("w1", account("c1", ref("vault:new", '7'), start.Add(5*day)), human("clerk")))
	if v.Rules != rules || v.Revision != 2 {
		t.Fatalf("snapshot %+v", v)
	}
	v = must(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), human("checker")))
	d := v.Verification.Digest
	for _, who := range []string{"clerk", "checker", "onboard", "onboard2"} {
		expect(t, "self "+who, second(f.r.Approve("w1", "c1", approve(d), human(who))), ReasonSelfApproval)
	}
	expect(t, "request digest", second(f.r.Approve("w1", "c1", approve(v.Digest), human("boss"))), ReasonDigestChanged)
	v = must(f.r.Approve("w1", "c1", approve(d), human("boss")))
	if v.State != StateConfirmed || len(v.Decisions) != 1 {
		t.Fatalf("one of two %+v", v)
	}
	again := must(f.r.Approve("w1", "c1", approve(d), human("boss")))
	if len(again.Decisions) != 1 {
		t.Fatalf("repeat added %+v", again)
	}
	reject := Approval{Verdict: VerdictReject, Reason: "other", Digest: d}
	expect(t, "flip", second(f.r.Approve("w1", "c1", reject, human("boss"))), ReasonAlreadyDecided)
	v = must(f.r.Approve("w1", "c1", approve(d), human("cfo")))
	if v.State != StateScheduled || len(v.Decisions) != 2 {
		t.Fatalf("two of two %+v", v)
	}
	expect(t, "after scheduled", second(f.r.Approve("w1", "c1", approve(d), human("third"))), ReasonNotPending)
	must(f.r.Approve("w1", "c1", approve(d), human("cfo")))
}

func TestRejectionsEndChange(t *testing.T) {
	f := ready(t)
	v := must(f.r.Request("w1", account("c1", ref("vault:new", '7'), start.Add(5*day)), human("clerk")))
	deny := confirm("c1", "ch-phone", v.Digest)
	deny.Outcome = OutcomeDenied
	v = must(f.r.Verify("w1", "c1", deny, human("checker")))
	if v.State != StateRejected || v.EndedBy != "checker" {
		t.Fatalf("denied %+v", v)
	}
	must(f.r.Verify("w1", "c1", deny, human("checker")))
	expect(t, "verify again", second(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), human("checker"))), ReasonNotPending)
	v = must(f.r.Request("w1", account("c2", ref("vault:new", '7'), start.Add(5*day)), human("clerk")))
	v = must(f.r.Verify("w1", "c2", confirm("c2", "ch-phone", v.Digest), human("checker")))
	v = must(f.r.Approve("w1", "c2", Approval{Verdict: VerdictReject, Reason: "suspected_fraud", Digest: v.Verification.Digest}, human("boss")))
	if v.State != StateRejected || v.EndedBy != "boss" {
		t.Fatalf("rejected %+v", v)
	}
	expect(t, "withdraw rejected", second(f.r.Withdraw("w1", "c2", human("clerk"))), ReasonNotPending)
	expect(t, "resolve", second(f.r.Resolve("w1", "acme", FieldPaymentAccount, start.Add(20*day))), "not_found")
}

func TestLapseAfterValidity(t *testing.T) {
	f := ready(t)
	v := must(f.r.Request("w1", account("c1", ref("vault:new", '7'), start.Add(20*day)), human("clerk")))
	v = must(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), human("checker")))
	confirmed := v.Verification.At
	f.clock.advance(2 * day)
	if got := must(f.r.Of("w1", "c1")); got.State != StateConfirmed {
		t.Fatalf("at deadline %+v", got)
	}
	f.clock.advance(time.Nanosecond)
	expect(t, "lapsed", second(f.r.Approve("w1", "c1", approve(v.Verification.Digest), human("boss"))), ReasonLapsed)
	got := must(f.r.Of("w1", "c1"))
	if got.State != StateLapsed || !got.Ended.Equal(confirmed.Add(2*day)) {
		t.Fatalf("lapse %+v", got)
	}
	expect(t, "withdraw lapsed", second(f.r.Withdraw("w1", "c1", human("clerk"))), ReasonNotPending)
	expect(t, "verify lapsed", second(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), human("other"))), ReasonLapsed)
	must(f.r.Request("w1", account("c2", ref("vault:new", '7'), start.Add(20*day)), human("clerk")))
}

func TestErrorsUnwrap(t *testing.T) {
	err := refuse(ReasonLapsed)
	if !errors.Is(err, ErrRejected) || err.Error() != ErrRejected.Error()+": lapsed" {
		t.Fatal(err)
	}
}
