package protected

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestVerifyRefusalPairs(t *testing.T) {
	f := ready(t)
	must(f.r.Register("w1", Channel{ID: "ch-pend", Subject: "acme", Kind: ChannelPortal, Ref: ref("vault:pend", 'd'), Evidence: "e"}, human("onboard")))
	must(f.r.Register("w1", Channel{ID: "ch-beta-pend", Subject: "beta", Kind: ChannelPortal, Ref: ref("vault:bp", 'd'), Evidence: "e"}, human("onboard")))
	c := account("c1", ref("v", 'a'), start.Add(9*day))
	c.Instruction = Instruction{Kind: ChannelEmail, Sender: fingerprint('7'), Evidence: "e"}
	v := must(f.r.Request("w1", c, human("clerk")))
	f.channel(t, "w1", Channel{ID: "ch-new", Subject: "acme", Kind: ChannelPhone, Ref: ref("vault:new", '3'), Evidence: "e"})
	f.channel(t, "w1", Channel{ID: "ch-sender-new", Subject: "acme", Kind: ChannelEmail, Ref: ref("vault:s", '7'), Evidence: "e"})
	verify := func(channel, digest string, actor Actor) string {
		return kind(second(f.r.Verify("w1", "c1", confirm("c1", channel, digest), actor)))
	}
	bad := fingerprint('0')
	cases := []struct {
		label, channel, digest string
		actor                  Actor
		want                   string
	}{
		{"system before state", "ch-phone", v.Digest, system("job"), ReasonNotHuman},
		{"digest before self", "ch-phone", bad, human("clerk"), ReasonDigestChanged},
		{"self before unknown", "ch-none", v.Digest, human("clerk"), ReasonSelfVerification},
		{"unknown before not admitted", "ch-beta-pend", v.Digest, human("checker"), ReasonUnknownChannel},
		{"not admitted before custodian", "ch-pend", v.Digest, human("onboard"), ReasonChannelNotAdmitted},
		{"same sender before too new", "ch-sender-new", v.Digest, human("checker"), ReasonSameChannel},
		{"too new before custodian", "ch-new", v.Digest, human("onboard2"), ReasonChannelTooNew},
	}
	for _, tc := range cases {
		if got := verify(tc.channel, tc.digest, tc.actor); got != tc.want {
			t.Fatalf("%s: got %s want %s", tc.label, got, tc.want)
		}
	}
	g := ready(t)
	w := must(g.r.Request("w1", account("c1", ref("v", 'a'), start.Add(9*day)), human("clerk")))
	must(g.r.Retire("w1", "ch-mail", human("ops")))
	expect(t, "same before retired", second(g.r.Verify("w1", "c1", confirm("c1", "ch-mail", w.Digest), human("checker"))), ReasonSameChannel)
	must(g.r.Retire("w1", "ch-phone", human("ops")))
	expect(t, "retired before custodian", second(g.r.Verify("w1", "c1", confirm("c1", "ch-phone", w.Digest), human("onboard"))), ReasonChannelRetired)
}

func TestVerifyRefusalStateFirst(t *testing.T) {
	f := ready(t)
	must(f.r.Request("w1", account("c1", ref("v", 'a'), start.Add(9*day)), human("clerk")))
	must(f.r.Withdraw("w1", "c1", human("clerk")))
	expect(t, "state before digest", second(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", fingerprint('0')), human("clerk"))), ReasonNotPending)
}

func TestApproveRefusalPairs(t *testing.T) {
	f := ready(t)
	v := must(f.r.Request("w1", account("c1", ref("v", 'a'), start.Add(9*day)), human("clerk")))
	expect(t, "requested before digest", second(f.r.Approve("w1", "c1", approve(fingerprint('0')), human("clerk"))), ReasonNotConfirmed)
	v = must(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), human("checker")))
	d := v.Verification.Digest
	expect(t, "digest before retired", second(f.r.Approve("w1", "c1", approve(v.Digest), human("boss"))), ReasonDigestChanged)
	g := ready(t)
	w := must(g.r.Request("w1", account("c1", ref("v", 'a'), start.Add(9*day)), human("clerk")))
	w = must(g.r.Verify("w1", "c1", confirm("c1", "ch-phone", w.Digest), human("checker")))
	must(g.r.Retire("w1", "ch-phone", human("ops")))
	expect(t, "digest before retired 2", second(g.r.Approve("w1", "c1", approve(fingerprint('0')), human("clerk"))), ReasonDigestChanged)
	expect(t, "retired before self", second(g.r.Approve("w1", "c1", approve(w.Verification.Digest), human("clerk"))), ReasonChannelRetired)
	rules := testRules()
	rules.Approvals = 2
	h := ready(t)
	h.rules(t, "w1", rules)
	x := must(h.r.Request("w1", account("c1", ref("v", 'a'), start.Add(9*day)), human("clerk")))
	x = must(h.r.Verify("w1", "c1", confirm("c1", "ch-phone", x.Digest), human("checker")))
	must(h.r.Approve("w1", "c1", approve(x.Verification.Digest), human("boss")))
	expect(t, "self before decided", second(h.r.Approve("w1", "c1", approve(x.Verification.Digest), human("clerk"))), ReasonSelfApproval)
	expect(t, "system repeat", second(h.r.Approve("w1", "c1", approve(x.Verification.Digest), system("boss"))), ReasonNotHuman)
	expect(t, "missing change", second(h.r.Approve("w1", "c9", approve(d), human("boss"))), "not_found")
	h.clock.advance(3 * day)
	expect(t, "lapsed before digest", second(h.r.Approve("w1", "c1", approve(fingerprint('0')), human("cfo"))), ReasonLapsed)
}

func TestRequestRefusalPairs(t *testing.T) {
	f := ready(t)
	c := account("c1", ref("v", 'a'), start.Add(9*day))
	c.Provenance = ProvenanceAssistant
	expect(t, "actor before provenance", second(f.r.Request("w1", c, system("job"))), ReasonNotHuman)
	expect(t, "provenance before rules", second(f.r.Request("w9", c, human("clerk"))), ReasonLLMGenerated)
	c.Provenance = ProvenanceDocument
	c.Instruction.Channel = "ch-nope"
	expect(t, "rules before channel", second(f.r.Request("w9", c, human("clerk"))), ReasonNoRules)
	must(f.r.Request("w1", account("c0", ref("v", 'a'), start.Add(9*day)), human("clerk")))
	expect(t, "channel before pending", second(f.r.Request("w1", c, human("clerk"))), ReasonUnknownChannel)
	c.Instruction = Instruction{Kind: ChannelPhone, Sender: fingerprint('a'), Evidence: "e"}
	expect(t, "undeclared before pending", second(f.r.Request("w1", c, human("clerk"))), ReasonUndeclaredChannel)
	c = account("c1", ref("v", 'a'), start)
	expect(t, "pending before range", second(f.r.Request("w1", c, human("clerk"))), ReasonChangePending)
	g := ready(t)
	g.schedule(t, account("b1", ref("v", 'a'), start.Add(9*day)))
	expect(t, "range before order", second(g.r.Request("w1", account("b2", ref("v", 'a'), start), human("clerk"))), ReasonEffectiveOutOfRange)
	expect(t, "order before no change", second(g.r.Request("w1", account("b2", ref("v", 'a'), start.Add(8*day)), human("clerk"))), ReasonOutOfOrder)
	h := ready(t)
	for i := 0; i < 6; i++ {
		x := account("x"+strconv.Itoa(i), ref("v", 'a'), start.Add(9*day))
		x.Subject = "s" + strconv.Itoa(i)
		x.Instruction = Instruction{Kind: ChannelPostal, Sender: fingerprint('c'), Evidence: "e"}
		must(h.r.Request("w1", x, human("clerk")))
	}
	x := account("x9", ref("v", 'a'), start.Add(9*day))
	x.Subject = "s0"
	x.Instruction = Instruction{Kind: ChannelPostal, Sender: fingerprint('c'), Evidence: "e"}
	expect(t, "pending before limit", second(h.r.Request("w1", x, human("clerk"))), ReasonChangePending)
}

func TestInstructionChannelChecks(t *testing.T) {
	f := ready(t)
	f.channel(t, "w1", Channel{ID: "ch-beta", Subject: "beta", Kind: ChannelEmail, Ref: ref("vault:b", 'b'), Evidence: "e"})
	c := account("c1", ref("v", 'a'), start.Add(9*day))
	c.Instruction.Channel = "ch-beta"
	expect(t, "other subject instruction", second(f.r.Request("w1", c, human("clerk"))), ReasonUnknownChannel)
	must(f.r.Retire("w1", "ch-mail", human("ops")))
	expect(t, "retired sender undeclared", second(f.r.Request("w1", newEmailFrom("c2", 'b'), human("clerk"))), ReasonUndeclaredChannel)
	expect(t, "other subject sender", second(f.r.Request("w1", otherSubject("c3", 'b'), human("clerk"))), ReasonUndeclaredChannel)
	expect(t, "unrelated sender", second(f.r.Request("w1", otherSubject("c4", 'e'), human("clerk"))), "ok")
}

func newEmailFrom(id string, sender byte) Change {
	c := account(id, ref("v", 'a'), start.Add(9*day))
	c.Instruction = Instruction{Kind: ChannelEmail, Sender: fingerprint(sender), Evidence: "e"}
	return c
}

func otherSubject(id string, sender byte) Change {
	c := newEmailFrom(id, sender)
	c.Subject = "beta"
	return c
}

func TestChannelAgeBoundary(t *testing.T) {
	f := newFixture(t)
	f.rules(t, "w1", testRules())
	f.channel(t, "w1", Channel{ID: "ch-phone", Subject: "acme", Kind: ChannelPhone, Ref: ref("vault:phone", 'a'), Evidence: "e"})
	f.clock.advance(day)
	v := must(f.r.Request("w1", newEmailFrom("c1", 'e'), human("clerk")))
	expect(t, "exactly old enough", second(f.r.Verify("w1", "c1", confirm("c1", "ch-phone", v.Digest), human("checker"))), "ok")
	g := newFixture(t)
	g.rules(t, "w1", testRules())
	g.channel(t, "w1", Channel{ID: "ch-phone", Subject: "acme", Kind: ChannelPhone, Ref: ref("vault:phone", 'a'), Evidence: "e"})
	g.clock.advance(day - time.Nanosecond)
	w := must(g.r.Request("w1", newEmailFrom("c1", 'e'), human("clerk")))
	expect(t, "just too new", second(g.r.Verify("w1", "c1", confirm("c1", "ch-phone", w.Digest), human("checker"))), ReasonChannelTooNew)
}

func TestRepeatsNeedHumanKind(t *testing.T) {
	f := ready(t)
	c := account("c1", ref("v", 'a'), start.Add(9*day))
	v := must(f.r.Request("w1", c, human("clerk")))
	expect(t, "request repeat as assistant", second(f.r.Request("w1", c, assistant("clerk"))), "conflict")
	vf := confirm("c1", "ch-phone", v.Digest)
	v = must(f.r.Verify("w1", "c1", vf, human("checker")))
	expect(t, "verify repeat as system", second(f.r.Verify("w1", "c1", vf, system("checker"))), ReasonNotHuman)
	must(f.r.Approve("w1", "c1", approve(v.Verification.Digest), human("boss")))
	expect(t, "approve repeat as assistant", second(f.r.Approve("w1", "c1", approve(v.Verification.Digest), assistant("boss"))), ReasonAINotAllowed)
	must(f.r.Withdraw("w1", "c1", human("ops")))
	expect(t, "withdraw repeat as system", second(f.r.Withdraw("w1", "c1", system("ops"))), ReasonNotHuman)
	expect(t, "register repeat as system", second(f.r.Register("w1", Channel{ID: "ch-phone", Subject: "acme", Kind: ChannelPhone, Ref: ref("vault:phone", 'a'), Evidence: "contract:1"}, system("onboard"))), "conflict")
	expect(t, "admit repeat as system", second(f.r.Admit("w1", "ch-phone", "admission:ch-phone", system("onboard2"))), "conflict")
	must(f.r.Retire("w1", "ch-phone", human("ops")))
	expect(t, "retire repeat as system", second(f.r.Retire("w1", "ch-phone", system("ops"))), "conflict")
	expect(t, "confirm repeat as system", second(f.r.Confirm("w1", 1, system("admin2"))), ReasonNotHuman)
	expect(t, "propose repeat as system", second(f.r.Propose("w1", testRules(), system("admin"))), ReasonNotHuman)
}

func TestRulesHistoryAndRepeats(t *testing.T) {
	f := newFixture(t)
	r1 := testRules()
	p1 := must(f.r.Propose("w1", r1, human("a")))
	a1 := must(f.r.Confirm("w1", p1.Revision, human("b")))
	if again := must(f.r.Propose("w1", r1, human("a"))); again != a1 {
		t.Fatalf("proposer repeat after confirm %+v", again)
	}
	expect(t, "other same as active", second(f.r.Propose("w1", r1, human("c"))), ReasonNoChange)
	r2 := r1
	r2.Approvals = 2
	must(f.r.Propose("w1", r2, human("a")))
	must(f.r.Discard("w1", 2, human("c")))
	must(f.r.Propose("w1", r2, human("a")))
	must(f.r.Propose("w1", Rules{Approvals: 3, Validity: day, MaxLead: day, ChannelAge: day}, human("a")))
	a4 := must(f.r.Confirm("w1", 4, human("b")))
	state := must(f.r.RulesOf("w1"))
	if len(state.History) != 4 || state.History[0] != a1 || state.History[1].Discarder != "c" || state.History[2].Confirmer != "" || state.History[3] != a4 {
		t.Fatalf("history %+v", state.History)
	}
	state.History[0].Proposer = "x"
	if must(f.r.RulesOf("w1")).History[0].Proposer != "a" {
		t.Fatal("history aliasing")
	}
}

func TestConcurrentCalls(t *testing.T) {
	f := ready(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := newEmailFrom("c"+strconv.Itoa(i), 'e')
			c.Subject = "s" + strconv.Itoa(i%3)
			_, _ = f.r.Request("w1", c, human("clerk"))
			_, _ = f.r.Trail("w1", c.Subject, FieldPaymentAccount)
			_, _ = f.r.Resolve("w1", c.Subject, FieldPaymentAccount, start.Add(10*day))
			_, _ = f.r.RulesOf("w1")
			_, _ = f.r.ChannelOf("w1", "ch-phone")
		}(i)
	}
	wg.Wait()
	total := 0
	for i := 0; i < 3; i++ {
		total += len(must(f.r.Trail("w1", "s"+strconv.Itoa(i), FieldPaymentAccount)))
	}
	if total != 3 {
		t.Fatalf("one pending per subject, got %d", total)
	}
}
