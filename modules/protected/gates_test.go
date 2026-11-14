package protected

import (
	"testing"
	"time"
)

func TestNewRegistryForms(t *testing.T) {
	clock := &testClock{now: start}
	for _, l := range []Limits{{0, 1, 1}, {1, 0, 1}, {1, 1, 0}, {maxWorkspaces + 1, 1, 1}, {1, maxChannels + 1, 1}, {1, 1, maxChanges + 1}} {
		if _, err := NewRegistry(l, clock.read); err != ErrInvalid {
			t.Fatalf("%+v: %v", l, err)
		}
	}
	if _, err := NewRegistry(Limits{maxWorkspaces, maxChannels, maxChanges}, nil); err != ErrInvalid {
		t.Fatal(err)
	}
	must(NewRegistry(Limits{maxWorkspaces, maxChannels, maxChanges}, clock.read))
}

func TestRulesForms(t *testing.T) {
	f := newFixture(t)
	good := testRules()
	bad := []func(*Rules){
		func(r *Rules) { r.Approvals = 0 },
		func(r *Rules) { r.Approvals = maxApprovals + 1 },
		func(r *Rules) { r.Validity = minValidity - 1 },
		func(r *Rules) { r.Validity = maxValidity + 1 },
		func(r *Rules) { r.MaxLead = minLead - 1 },
		func(r *Rules) { r.MaxLead = maxLead + 1 },
		func(r *Rules) { r.ChannelAge = minChannelAge - 1 },
		func(r *Rules) { r.ChannelAge = maxChannelAge + 1 },
	}
	for _, change := range bad {
		r := good
		change(&r)
		expect(t, "rules form", second(f.r.Propose("w1", r, human("a"))), "invalid")
	}
	edge := Rules{Approvals: maxApprovals, Validity: maxValidity, MaxLead: maxLead, ChannelAge: maxChannelAge}
	must(f.r.Propose("w1", edge, human("a")))
	low := Rules{Approvals: 1, Validity: minValidity, MaxLead: minLead, ChannelAge: minChannelAge}
	must(f.r.Propose("w2", low, human("a")))
	expect(t, "workspace id", second(f.r.Propose("bad id", good, human("a"))), "invalid")
	expect(t, "actor id", second(f.r.Propose("w3", good, human(""))), "invalid")
	expect(t, "actor kind", second(f.r.Propose("w3", good, Actor{ID: "a", Kind: "robot"})), "invalid")
}

func TestRulesDualControl(t *testing.T) {
	f := newFixture(t)
	r1 := testRules()
	expect(t, "assistant", second(f.r.Propose("w1", r1, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Propose("w1", r1, system("job"))), ReasonNotHuman)
	if _, err := f.r.RulesOf("w1"); err != ErrNotFound {
		t.Fatal(err)
	}
	p := must(f.r.Propose("w1", r1, human("admin")))
	if p.Revision != 1 || p.Proposer != "admin" || !p.Proposed.Equal(start) {
		t.Fatalf("proposal %+v", p)
	}
	if again := must(f.r.Propose("w1", r1, human("admin"))); again != p {
		t.Fatalf("repeat %+v", again)
	}
	state := must(f.r.RulesOf("w1"))
	if state.Active != nil || state.Pending == nil || *state.Pending != p {
		t.Fatalf("state %+v", state)
	}
	expect(t, "request before rules", second(f.r.Request("w1", account("c1", ref("v", 'a'), start.Add(day)), human("clerk"))), ReasonNoRules)
	expect(t, "register before rules allowed", second(f.r.Register("w1", Channel{ID: "ch", Subject: "s", Kind: ChannelEmail, Ref: ref("v", 'a'), Evidence: "e"}, human("x"))), "ok")
	expect(t, "other proposer", second(f.r.Propose("w1", Rules{Approvals: 2, Validity: day, MaxLead: day, ChannelAge: day}, human("admin2"))), ReasonProposalPending)
	r2 := r1
	r2.Approvals = 2
	p2 := must(f.r.Propose("w1", r2, human("admin")))
	if p2.Revision != 2 {
		t.Fatalf("new number %+v", p2)
	}
	expect(t, "replaced", second(f.r.Confirm("w1", 1, human("admin2"))), ReasonStaleProposal)
	expect(t, "self confirm", second(f.r.Confirm("w1", 2, human("admin"))), ReasonSelfApproval)
	expect(t, "assistant confirm", second(f.r.Confirm("w1", 2, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system confirm", second(f.r.Confirm("w1", 2, system("job"))), ReasonNotHuman)
	expect(t, "confirm missing", second(f.r.Confirm("w9", 2, human("admin2"))), "not_found")
	expect(t, "confirm form", second(f.r.Confirm("w1", 0, human("admin2"))), "invalid")
	f.clock.advance(time.Minute)
	active := must(f.r.Confirm("w1", 2, human("admin2")))
	if active.Confirmer != "admin2" || !active.Confirmed.Equal(start.Add(time.Minute)) || active.Rules != r2 {
		t.Fatalf("active %+v", active)
	}
	if again := must(f.r.Confirm("w1", 2, human("admin2"))); again != active {
		t.Fatalf("confirm repeat %+v", again)
	}
	expect(t, "confirm by other after", second(f.r.Confirm("w1", 2, human("admin3"))), ReasonStaleProposal)
	expect(t, "same as active", second(f.r.Propose("w1", r2, human("admin3"))), ReasonNoChange)
	p3 := must(f.r.Propose("w1", r1, human("admin3")))
	expect(t, "discard assistant", second(f.r.Discard("w1", 3, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "discard system", second(f.r.Discard("w1", 3, system("job"))), ReasonNotHuman)
	expect(t, "discard stale", second(f.r.Discard("w1", 2, human("admin"))), ReasonStaleProposal)
	expect(t, "discard missing", second(f.r.Discard("w9", 3, human("admin"))), "not_found")
	expect(t, "discard form", second(f.r.Discard("w1", 0, human("admin"))), "invalid")
	d := must(f.r.Discard("w1", p3.Revision, human("admin")))
	if d.Discarder != "admin" || d.Revision != 3 {
		t.Fatalf("discard %+v", d)
	}
	if again := must(f.r.Discard("w1", 3, human("admin"))); again != d {
		t.Fatalf("discard repeat %+v", again)
	}
	expect(t, "discard by other after", second(f.r.Discard("w1", 3, human("admin2"))), ReasonStaleProposal)
	state = must(f.r.RulesOf("w1"))
	if state.Pending != nil || *state.Active != active {
		t.Fatalf("after discard %+v", state)
	}
	if p4 := must(f.r.Propose("w1", r1, human("admin2"))); p4.Revision != 4 {
		t.Fatalf("after discard %+v", p4)
	}
	expect(t, "confirm discarded", second(f.r.Confirm("w1", 3, human("admin"))), ReasonStaleProposal)
	must(f.r.Confirm("w1", 4, human("admin")))
}

func TestWorkspaceLimitAndIsolation(t *testing.T) {
	f := ready(t)
	f.rules(t, "w2", testRules())
	f.rules(t, "w3", testRules())
	expect(t, "limit", second(f.r.Propose("w4", testRules(), human("admin"))), "limit")
	expect(t, "foreign channel", second(f.r.ChannelOf("w2", "ch-phone")), "not_found")
	v := must(f.r.Request("w1", account("c1", ref("v", 'a'), start.Add(5*day)), human("clerk")))
	expect(t, "foreign change", second(f.r.Of("w2", "c1")), "not_found")
	expect(t, "foreign verify", second(f.r.Verify("w2", "c1", confirm("c1", "ch-phone", v.Digest), human("checker"))), "not_found")
	expect(t, "foreign approve", second(f.r.Approve("w2", "c1", approve(v.Digest), human("boss"))), "not_found")
	expect(t, "foreign withdraw", second(f.r.Withdraw("w2", "c1", human("boss"))), "not_found")
	c := account("c1", ref("v", 'a'), start.Add(5*day))
	expect(t, "foreign instruction channel", second(f.r.Request("w2", c, human("clerk"))), ReasonUnknownChannel)
	f.channel(t, "w2", Channel{ID: "ch-phone", Subject: "acme", Kind: ChannelPhone, Ref: ref("w2:phone", 'c'), Evidence: "e"})
	f.channel(t, "w2", Channel{ID: "ch-mail", Subject: "acme", Kind: ChannelEmail, Ref: ref("w2:mail", 'b'), Evidence: "e"})
	other := must(f.r.Request("w2", c, human("clerk")))
	if other.Digest == v.Digest {
		t.Fatal("digest ignores workspace")
	}
	expect(t, "w2 trail", second(f.r.Trail("w2", "acme", FieldPaymentAccount)), "ok")
	expect(t, "missing trail", second(f.r.Trail("w9", "acme", FieldPaymentAccount)), "not_found")
	expect(t, "missing resolve", second(f.r.Resolve("w9", "acme", FieldPaymentAccount, start)), "not_found")
	expect(t, "missing register", second(f.r.Register("w9", Channel{ID: "c", Subject: "s", Kind: ChannelEmail, Ref: ref("v", 'a'), Evidence: "e"}, human("x"))), "not_found")
	expect(t, "missing channel of", second(f.r.ChannelOf("w9", "c")), "not_found")
}

func TestChannelLifecycle(t *testing.T) {
	f := newFixture(t)
	f.rules(t, "w1", testRules())
	c := Channel{ID: "ch", Subject: "acme", Kind: ChannelPhone, Ref: ref("vault:p", 'a'), Evidence: "contract"}
	for _, bad := range []Channel{
		{ID: "", Subject: "acme", Kind: ChannelPhone, Ref: c.Ref, Evidence: "e"},
		{ID: "ch", Subject: "", Kind: ChannelPhone, Ref: c.Ref, Evidence: "e"},
		{ID: "ch", Subject: "acme", Kind: ChannelInPerson, Ref: c.Ref, Evidence: "e"},
		{ID: "ch", Subject: "acme", Kind: ChannelPhone, Ref: Ref{ID: "v", Fingerprint: "AA"}, Evidence: "e"},
		{ID: "ch", Subject: "acme", Kind: ChannelPhone, Ref: Ref{ID: "", Fingerprint: fingerprint('a')}, Evidence: "e"},
		{ID: "ch", Subject: "acme", Kind: ChannelPhone, Ref: c.Ref, Evidence: ""},
	} {
		expect(t, "channel form", second(f.r.Register("w1", bad, human("reg"))), "invalid")
	}
	expect(t, "register assistant", second(f.r.Register("w1", c, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "register system", second(f.r.Register("w1", c, system("job"))), ReasonNotHuman)
	v := must(f.r.Register("w1", c, human("reg")))
	if v.Registrar != "reg" || !v.Registered.Equal(start) || v.Admitter != "" {
		t.Fatalf("registered %+v", v)
	}
	if again := must(f.r.Register("w1", c, human("reg"))); again != v {
		t.Fatalf("repeat %+v", again)
	}
	expect(t, "other registrar", second(f.r.Register("w1", c, human("x"))), "conflict")
	changed := c
	changed.Evidence = "other"
	expect(t, "changed", second(f.r.Register("w1", changed, human("reg"))), "conflict")
	expect(t, "admit form", second(f.r.Admit("w1", "ch", "", human("adm"))), "invalid")
	expect(t, "admit missing", second(f.r.Admit("w1", "nope", "e", human("adm"))), "not_found")
	expect(t, "admit assistant", second(f.r.Admit("w1", "ch", "e", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "admit system", second(f.r.Admit("w1", "ch", "e", system("job"))), ReasonNotHuman)
	expect(t, "admit self", second(f.r.Admit("w1", "ch", "e", human("reg"))), ReasonSelfApproval)
	f.clock.advance(time.Hour)
	a := must(f.r.Admit("w1", "ch", "e", human("adm")))
	if a.Admitter != "adm" || a.Admission != "e" || !a.Admitted.Equal(start.Add(time.Hour)) {
		t.Fatalf("admitted %+v", a)
	}
	if again := must(f.r.Admit("w1", "ch", "e", human("adm"))); again != a {
		t.Fatalf("admit repeat %+v", again)
	}
	expect(t, "admit other evidence", second(f.r.Admit("w1", "ch", "e2", human("adm"))), "conflict")
	expect(t, "admit other", second(f.r.Admit("w1", "ch", "e", human("adm2"))), "conflict")
	expect(t, "retire form", second(f.r.Retire("w1", "", human("ops"))), "invalid")
	expect(t, "retire missing", second(f.r.Retire("w1", "nope", human("ops"))), "not_found")
	expect(t, "retire assistant", second(f.r.Retire("w1", "ch", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "retire system", second(f.r.Retire("w1", "ch", system("job"))), ReasonNotHuman)
	r := must(f.r.Retire("w1", "ch", human("ops")))
	if r.RetiredBy != "ops" || !r.Retired.Equal(start.Add(time.Hour)) {
		t.Fatalf("retired %+v", r)
	}
	if again := must(f.r.Retire("w1", "ch", human("ops"))); again != r {
		t.Fatalf("retire repeat %+v", again)
	}
	expect(t, "retire other", second(f.r.Retire("w1", "ch", human("x"))), "conflict")
	if got := must(f.r.ChannelOf("w1", "ch")); got != r {
		t.Fatalf("channel of %+v", got)
	}
	p := Channel{ID: "ch2", Subject: "acme", Kind: ChannelEmail, Ref: ref("vault:m", 'b'), Evidence: "e"}
	must(f.r.Register("w1", p, human("reg")))
	must(f.r.Retire("w1", "ch2", human("ops")))
	expect(t, "admit retired", second(f.r.Admit("w1", "ch2", "e", human("adm"))), ReasonChannelRetired)
	for i := 3; i <= 6; i++ {
		must(f.r.Register("w1", Channel{ID: "ch" + string(rune('0'+i)), Subject: "acme", Kind: ChannelEmail, Ref: ref("v", 'c'), Evidence: "e"}, human("reg")))
	}
	expect(t, "channel limit", second(f.r.Register("w1", Channel{ID: "ch7", Subject: "acme", Kind: ChannelEmail, Ref: ref("v", 'c'), Evidence: "e"}, human("reg"))), "limit")
	expect(t, "repeat at limit", second(f.r.Register("w1", p, human("reg"))), "ok")
}

func TestChangeForms(t *testing.T) {
	f := ready(t)
	good := account("c1", ref("v", 'a'), start.Add(5*day))
	bad := []func(*Change){
		func(c *Change) { c.ID = "" },
		func(c *Change) { c.Subject = "a b" },
		func(c *Change) { c.Field = "bank" },
		func(c *Change) { c.After = Ref{ID: "v", Fingerprint: fingerprint('g')} },
		func(c *Change) { c.After = Ref{ID: "v", Fingerprint: fingerprint('a')[:63]} },
		func(c *Change) { c.Provenance = "guess" },
		func(c *Change) { c.Instruction.Kind = "fax" },
		func(c *Change) { c.Instruction.Channel = "a b" },
		func(c *Change) { c.Instruction.Sender = "" },
		func(c *Change) { c.Instruction.Kind = ChannelInPerson },
		func(c *Change) { c.Instruction.Evidence = "" },
		func(c *Change) { c.Effective = time.Time{} },
	}
	for i, change := range bad {
		c := good
		change(&c)
		if got := kind(second(f.r.Request("w1", c, human("clerk")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	expect(t, "workspace", second(f.r.Request("", good, human("clerk"))), "invalid")
	expect(t, "actor", second(f.r.Request("w1", good, Actor{ID: "x"})), "invalid")
	long := make([]byte, maxIDBytes+1)
	for i := range long {
		long[i] = 'a'
	}
	c := good
	c.ID = string(long)
	expect(t, "long id", second(f.r.Request("w1", c, human("clerk"))), "invalid")
	c.ID = string(long[:maxIDBytes])
	c.Instruction.Channel = ""
	c.Instruction.Sender = fingerprint('c')
	expect(t, "max id", second(f.r.Request("w1", c, human("clerk"))), "ok")
	for _, field := range []string{FieldDocumentRecipient, FieldTaxIdentifier, FieldDocumentTemplate} {
		c := good
		c.ID, c.Field = "f-"+field, field
		expect(t, field, second(f.r.Request("w1", c, human("clerk"))), "ok")
	}
	for _, p := range []string{ProvenanceStaff, ProvenancePortal} {
		c := good
		c.ID, c.Subject, c.Provenance = "p-"+p, "s-"+p, p
		c.Instruction = Instruction{Kind: ChannelPostal, Sender: fingerprint('c'), Evidence: "e"}
		expect(t, p, second(f.r.Request("w1", c, human("clerk"))), "ok")
	}
}

func TestVerifyAndApproveForms(t *testing.T) {
	f := ready(t)
	v := must(f.r.Request("w1", account("c1", ref("v", 'a'), start.Add(5*day)), human("clerk")))
	good := confirm("c1", "ch-phone", v.Digest)
	for i, b := range []Verification{
		{Channel: "", Evidence: "e", Outcome: OutcomeConfirmed, Digest: v.Digest},
		{Channel: "ch-phone", Evidence: "", Outcome: OutcomeConfirmed, Digest: v.Digest},
		{Channel: "ch-phone", Evidence: "e", Outcome: "maybe", Digest: v.Digest},
		{Channel: "ch-phone", Evidence: "e", Outcome: OutcomeConfirmed, Digest: "x"},
	} {
		if got := kind(second(f.r.Verify("w1", "c1", b, human("checker")))); got != "invalid" {
			t.Fatalf("verify %d: %s", i, got)
		}
	}
	expect(t, "verify ws", second(f.r.Verify("", "c1", good, human("checker"))), "invalid")
	expect(t, "verify id", second(f.r.Verify("w1", "", good, human("checker"))), "invalid")
	expect(t, "verify actor", second(f.r.Verify("w1", "c1", good, human(""))), "invalid")
	expect(t, "verify missing", second(f.r.Verify("w1", "c9", good, human("checker"))), "not_found")
	expect(t, "verify system", second(f.r.Verify("w1", "c1", good, system("job"))), ReasonNotHuman)
	v = must(f.r.Verify("w1", "c1", good, human("checker")))
	if again := must(f.r.Verify("w1", "c1", good, human("checker"))); again.Verification.Digest != v.Verification.Digest {
		t.Fatal("verify repeat")
	}
	expect(t, "verify other", second(f.r.Verify("w1", "c1", good, human("checker2"))), ReasonNotPending)
	d := v.Verification.Digest
	for i, a := range []Approval{
		{Verdict: "maybe", Digest: d},
		{Verdict: VerdictApprove, Reason: "other", Digest: d},
		{Verdict: VerdictReject, Digest: d},
		{Verdict: VerdictReject, Reason: "mood", Digest: d},
		{Verdict: VerdictApprove, Digest: ""},
	} {
		if got := kind(second(f.r.Approve("w1", "c1", a, human("boss")))); got != "invalid" {
			t.Fatalf("approve %d: %s", i, got)
		}
	}
	expect(t, "approve ws", second(f.r.Approve("", "c1", approve(d), human("boss"))), "invalid")
	expect(t, "approve id", second(f.r.Approve("w1", "", approve(d), human("boss"))), "invalid")
	expect(t, "approve actor", second(f.r.Approve("w1", "c1", approve(d), human(""))), "invalid")
	expect(t, "approve missing", second(f.r.Approve("w1", "c9", approve(d), human("boss"))), "not_found")
	for _, reason := range []string{"incorrect_details", "missing_evidence"} {
		g := ready(t)
		w := g.schedule(t, account("c1", ref("v", 'a'), start.Add(5*day)))
		_ = w
		x := must(g.r.Request("w1", account("c2", ref("v", 'c'), start.Add(6*day)), human("clerk")))
		x = must(g.r.Verify("w1", "c2", confirm("c2", "ch-phone", x.Digest), human("checker")))
		expect(t, reason, second(g.r.Approve("w1", "c2", Approval{Verdict: VerdictReject, Reason: reason, Digest: x.Verification.Digest}, human("boss"))), "ok")
	}
	expect(t, "withdraw form", second(f.r.Withdraw("w1", "", human("x"))), "invalid")
	expect(t, "withdraw ws", second(f.r.Withdraw("", "c1", human("x"))), "invalid")
	expect(t, "withdraw actor", second(f.r.Withdraw("w1", "c1", human(""))), "invalid")
	expect(t, "withdraw missing", second(f.r.Withdraw("w1", "c9", human("x"))), "not_found")
	expect(t, "of missing", second(f.r.Of("w1", "c9")), "not_found")
	expect(t, "trail form", second(f.r.Trail("w1", "acme", "bank")), "invalid")
	expect(t, "trail ws", second(f.r.Trail("", "acme", FieldPaymentAccount)), "invalid")
	expect(t, "trail subject", second(f.r.Trail("w1", "", FieldPaymentAccount)), "invalid")
	expect(t, "resolve zero", second(f.r.Resolve("w1", "acme", FieldPaymentAccount, time.Time{})), "invalid")
	expect(t, "resolve field", second(f.r.Resolve("w1", "acme", "bank", start)), "invalid")
	expect(t, "resolve ws", second(f.r.Resolve("", "acme", FieldPaymentAccount, start)), "invalid")
	expect(t, "resolve subject", second(f.r.Resolve("w1", "", FieldPaymentAccount, start)), "invalid")
}
