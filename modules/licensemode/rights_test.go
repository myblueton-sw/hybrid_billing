package licensemode

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var issued int64

func reg(t testing.TB, f world, c Claims) string {
	t.Helper()
	if c.IssuedAt == start.Add(-time.Hour).Unix() {
		issued++
		c.IssuedAt = start.Add(-time.Hour).Unix() + issued
	}
	out, err := f.r.Register("i1", f.sign(c), ann)
	f.must(err)
	return out
}

func TestRightsDB14(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	day := 24 * time.Hour
	a := commercial("a", "agr-a", start, start.Add(30*day))
	if out := reg(t, f, a); out != OutcomeAccepted {
		t.Fatalf("a %s", out)
	}
	m := commercial("m", "agr-m", start.Add(day), start.Add(60*day))
	m.Channel = ChannelMarketplace
	f.advance(day)
	if out := reg(t, f, m); out != OutcomeDuplicateRight {
		t.Errorf("marketplace overlap %s", out)
	}
	same := commercial("d2", "agr-d2", start.Add(day), start.Add(60*day))
	if out := reg(t, f, same); out != OutcomeDuplicateRight {
		t.Errorf("direct overlap %s", out)
	}
	bad := m
	bad.ID, bad.SubjectID, bad.Supersedes = "b", "acme", "agr-zzz"
	if out := reg(t, f, bad); out != OutcomeBadSupersedes {
		t.Errorf("bad supersedes %s", out)
	}
	foreign := m
	foreign.ID, foreign.SubjectID, foreign.Supersedes = "f", "gamma", "agr-a"
	if out := reg(t, f, foreign); out != OutcomeBadSupersedes {
		t.Errorf("foreign supersedes %s", out)
	}
	renew := m
	renew.ID, renew.Supersedes = "r", "agr-a"
	if out := reg(t, f, renew); out != OutcomeAccepted {
		t.Errorf("migration %s", out)
	}
	if v := status(t, f, "i1"); v.Entitlement == nil || v.Entitlement.ID != "r" {
		t.Errorf("current %+v", v.Entitlement)
	}
	f.advance(day)
	again := commercial("r2", "agr-a2", start.Add(2*day), start.Add(90*day))
	if out := reg(t, f, again); out != OutcomeDuplicateRight {
		t.Errorf("superseded effective end, overlap with r %s", out)
	}
	g := standard(t)
	g.must(g.r.SelectMode("i1", ModeCommercial, ann))
	p := commercial("p", "agr-p", start, start)
	p.End, p.Perpetual = 0, true
	if out := reg(t, g, p); out != OutcomeAccepted {
		t.Fatalf("perpetual %s", out)
	}
	g.advance(500 * day)
	later := commercial("q", "agr-q", start.Add(400*day), start.Add(2000*day))
	later.IssuedAt = start.Add(300 * day).Unix()
	if out := reg(t, g, later); out != OutcomeDuplicateRight {
		t.Errorf("perpetual overlaps later %s", out)
	}
	h := standard(t)
	h.must(h.r.SelectMode("i1", ModeCommercial, ann))
	first := commercial("x1", "agr-x", start.Add(-day), start.Add(time.Hour))
	if out := reg(t, h, first); out != OutcomeAccepted {
		t.Fatal(out)
	}
	h.advance(time.Hour)
	adjacent := commercial("x2", "agr-y", start.Add(time.Hour), start.Add(10*day))
	if out := reg(t, h, adjacent); out != OutcomeAccepted {
		t.Errorf("adjacent half-open %s", out)
	}
	sameAgreement := commercial("x3", "agr-y", start.Add(2*time.Hour), start.Add(20*day))
	sameAgreement.Supersedes, sameAgreement.IssuedAt = "agr-y", start.Unix()
	if out := reg(t, h, sameAgreement); out != OutcomeNotYetValid {
		t.Errorf("not yet valid %s", out)
	}
	h.advance(time.Hour)
	if out := reg(t, h, sameAgreement); out != OutcomeAccepted {
		t.Errorf("same agreement renewal %s", out)
	}
	older := commercial("x4", "agr-z", start.Add(-time.Hour), start.Add(30*day))
	older.Supersedes, older.IssuedAt = "agr-y", start.Add(-3*time.Hour).Unix()
	if out := reg(t, h, older); out != OutcomeSuperseded {
		t.Errorf("replay of older %s", out)
	}
	first2 := commercial("own", "agr-own", start, start.Add(day))
	first2.SubjectID, first2.Supersedes, first2.IssuedAt = "delta", "agr-own", start.Add(2*time.Hour).Unix()
	if out := reg(t, h, first2); out != OutcomeBadSupersedes {
		t.Errorf("first names own agreement %s", out)
	}
}

func TestCompromise(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeKey, Key: keySpec(f, "leak", "owner")})
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	day := 24 * time.Hour
	genuine := commercial("g", "agr-g", start, start.Add(100*day))
	if out := reg(t, f, genuine); out != OutcomeAccepted {
		t.Fatal(out)
	}
	f.advance(day)
	attack := commercial("bad", "agr-bad", start.Add(day), start.Add(1000*day))
	attack.KeyID, attack.Supersedes, attack.IssuedAt = "leak", "agr-g", start.Add(day).Unix()
	if out := reg(t, f, attack); out != OutcomeAccepted {
		t.Fatal(out)
	}
	if v := status(t, f, "i1"); v.Entitlement.ID != "bad" {
		t.Fatalf("attack current %+v", v.Entitlement)
	}
	f.advance(day)
	f.approve(Change{Kind: ChangeCompromise, Compromise: "leak"})
	if v := status(t, f, "i1"); v.State != StateCommercialActive || v.Entitlement.ID != "g" {
		t.Errorf("genuine restored %+v %s", v.Entitlement, v.State)
	}
	if out := reg(t, f, attack); out != OutcomeUnverifiable {
		t.Errorf("compromised doc %s", out)
	}
	k := standard(t)
	k.approve(Change{Kind: ChangeKey, Key: keySpec(k, "leak", "owner")})
	k.must(k.r.SelectMode("i1", ModeCommercial, ann))
	only := commercial("only", "agr-o", start, start.Add(10*day))
	only.KeyID = "leak"
	reg(t, k, only)
	k.approve(Change{Kind: ChangeCompromise, Compromise: "leak"})
	if v := status(t, k, "i1"); v.State != StateUnverified || v.Reason != ReasonKeyCompromised || !v.Warning {
		t.Errorf("only compromised %+v", v)
	}
	mix := standard(t)
	mix.approve(Change{Kind: ChangeKey, Key: keySpec(mix, "leak", "owner")})
	mix.must(mix.r.SelectMode("i1", ModeCommercial, ann))
	if out := reg(t, mix, commercial("old", "agr-old", start.Add(-2*day), start.Add(30*time.Minute))); out != OutcomeAccepted {
		t.Fatal(out)
	}
	mix.advance(time.Hour)
	cur := commercial("cur", "agr-cur", start.Add(30*time.Minute), start.Add(10*day))
	cur.KeyID, cur.IssuedAt = "leak", start.Add(time.Hour).Unix()
	if out := reg(t, mix, cur); out != OutcomeAccepted {
		t.Fatal(out)
	}
	mix.approve(Change{Kind: ChangeCompromise, Compromise: "leak"})
	if v := status(t, mix, "i1"); v.State != StateUnverified || v.Reason != ReasonKeyCompromised {
		t.Errorf("compromised would be current %+v", v)
	}
	e := standard(t)
	e.approve(Change{Kind: ChangeKey, Key: keySpec(e, "leak", "owner")})
	e.must(e.r.SelectMode("i1", ModeEvaluation, ann))
	grant := evaluation("g1")
	grant.KeyID = "leak"
	e.must(second(e.r.Activate("i1", e.sign(grant), Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)))
	e.approve(Change{Kind: ChangeCompromise, Compromise: "leak"})
	if v := status(t, e, "i1"); v.State != StateUnverified || v.Reason != ReasonKeyCompromised {
		t.Errorf("evaluation compromised %+v", v)
	}
}

func TestActivationRules(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeEvaluation, ann))
	stale := evaluation("g1")
	stale.ActivateBy = start.Unix()
	reason(t, "stale", second(f.r.Activate("i1", f.sign(stale), Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)), ReasonStaleGrant)
	fresh := evaluation("g2")
	fresh.ActivateBy = start.Add(time.Second).Unix()
	f.must(second(f.r.Activate("i1", f.sign(fresh), Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)))
	g := Acceptance{Actor: "ann", Terms: "eval-terms-v1"}
	reason(t, "assistant", second(f.r.Activate("i1", f.sign(fresh), g, bot)), ReasonAINotAllowed)
	expect(t, "acceptance actor", second(f.r.Activate("i1", f.sign(fresh), g, cat)), ErrInvalid)
	expect(t, "terms form", second(f.r.Activate("i1", f.sign(fresh), Acceptance{Actor: "ann"}, ann)), ErrInvalid)
	expect(t, "installation form", second(f.r.Activate("", f.sign(fresh), g, ann)), ErrInvalid)
	expect(t, "installation missing", second(f.r.Activate("i9", f.sign(fresh), g, ann)), ErrNotFound)
	reason(t, "commercial doc", second(f.r.Activate("i1", f.sign(commercial("e", "a", start, start.Add(time.Hour))), g, ann)), OutcomeWrongKind)
	f.must(f.r.Install("i2", "hybrid-billing", ann))
	f.must(f.r.SelectMode("i2", ModeEvaluation, ann))
	moved := fresh
	moved.ID, moved.Installation = "g3", "i2"
	other, err := f.r.Activate("i2", f.sign(moved), g, ann)
	f.must(err)
	if again, _ := f.r.Activate("i2", f.sign(moved), g, ann); !again.Start.Equal(other.Start) {
		t.Error("repeat on other installation")
	}
	for len(f.r.grants) < maxRecords {
		f.r.grants["x"+itoa(len(f.r.grants))] = &activation{}
	}
	f.must(f.r.Install("i3", "hybrid-billing", ann))
	f.must(f.r.SelectMode("i3", ModeEvaluation, ann))
	cap := fresh
	cap.ID, cap.Installation = "g4", "i3"
	expect(t, "grants cap", second(f.r.Activate("i3", f.sign(cap), g, ann)), ErrLimit)
}

func TestInstallsAndProposals(t *testing.T) {
	f := setup(t)
	expect(t, "authority", second(NewRegistry(Config{}, func() time.Time { return start })), ErrInvalid)
	expect(t, "clock", second(NewRegistry(Config{Authority: f.auth}, nil)), ErrInvalid)
	reason(t, "install assistant", f.r.Install("i1", "p", bot), ReasonAINotAllowed)
	reason(t, "install system", f.r.Install("i1", "p", sys), ReasonNotHuman)
	expect(t, "install actor", f.r.Install("i1", "p", Actor{Kind: KindHuman}), ErrInvalid)
	expect(t, "install kind", f.r.Install("i1", "p", Actor{ID: "x", Kind: "robot"}), ErrInvalid)
	expect(t, "install form", f.r.Install("", "p", ann), ErrInvalid)
	f.must(f.r.Install("i1", "p", ann))
	expect(t, "install repeat", f.r.Install("i1", "p", ann), ErrConflict)
	reason(t, "select system", f.r.SelectMode("i1", ModeCommercial, sys), ReasonNotHuman)
	expect(t, "select missing", f.r.SelectMode("i9", ModeCommercial, ann), ErrNotFound)
	for n := 0; n < maxHistory+5; n++ {
		f.must(f.r.SelectMode("i1", ModeNoncommercial, ann))
	}
	if modes, _, _ := f.r.History("i1"); len(modes) != 100 {
		t.Errorf("mode history %d", len(modes))
	}
	_, _, err := f.r.History("nope")
	expect(t, "history missing", err, ErrNotFound)
	for len(f.r.installations) < maxInstallations {
		f.r.installations["x"+itoa(len(f.r.installations))] = &installation{}
	}
	expect(t, "install cap", f.r.Install("i-new", "p", ann), ErrLimit)
	expect(t, "status form", second(f.r.Status("")), ErrInvalid)
	expect(t, "status missing", second(f.r.Status("nope")), ErrNotFound)
	g := setup(t)
	c := Change{Kind: ChangeKey, Key: keySpec(g, "k1", "owner")}
	reason(t, "assistant", second(g.r.Propose(c, bot)), ReasonAINotAllowed)
	p, err := g.r.Propose(c, ann)
	g.must(err)
	if again, _ := g.r.Propose(c, ann); again.Number != p.Number {
		t.Error("repeat")
	}
	q, _ := g.r.Propose(c, cat)
	reason(t, "self", second(g.r.Approve(p.Number, ann)), ReasonSelfApproval)
	reason(t, "approve system", second(g.r.Approve(p.Number, sys)), ReasonNotHuman)
	expect(t, "missing", second(g.r.Approve(99, bob)), ErrNotFound)
	g.must(second(g.r.Approve(p.Number, bob)))
	if again, err := g.r.Approve(p.Number, bob); err != nil || again.State != ProposalApplied {
		t.Error("repeat approve")
	}
	reason(t, "applied", second(g.r.Approve(p.Number, cat)), ReasonNotPending)
	expect(t, "recheck", second(g.r.Approve(q.Number, bob)), ErrConflict)
	reason(t, "withdraw other", second(g.r.Withdraw(q.Number, ann)), ReasonNotProposer)
	reason(t, "withdraw applied", second(g.r.Withdraw(p.Number, ann)), ReasonNotPending)
	reason(t, "withdraw assistant", second(g.r.Withdraw(q.Number, bot)), ReasonAINotAllowed)
	expect(t, "withdraw missing", second(g.r.Withdraw(99, ann)), ErrNotFound)
	if w, err := g.r.Withdraw(q.Number, cat); err != nil || w.State != ProposalWithdrawn {
		t.Error("withdraw")
	}
	expect(t, "withdrawn", second(g.r.Proposal(q.Number)), ErrNotFound)
	if got, err := g.r.Proposal(p.Number); err != nil || got.Change.Key.ID != "k1" {
		t.Error("read")
	}
	if !errors.Is(Refusal{Reason: "x"}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Error("refusal")
	}
	for len(g.r.keys) < maxKeys {
		id := "z" + itoa(len(g.r.keys))
		g.r.keys[id] = &key{}
		g.r.active++
	}
	expect(t, "keys cap", g.try(Change{Kind: ChangeKey, Key: keySpec(g, "k-new", "owner")}), ErrLimit)
	h := setup(t)
	people := []Actor{ann, bob, cat, {ID: "dan", Kind: KindHuman}}
	for _, who := range people {
		for n := 0; n < maxMine; n++ {
			h.must(second(h.r.Propose(Change{Kind: ChangeReset, Reset: Reset{At: int64(n + 1), Reason: who.ID}}, who)))
		}
		expect(t, "mine", second(h.r.Propose(Change{Kind: ChangeReset, Reset: Reset{At: 99999, Reason: who.ID}}, who)), ErrLimit)
	}
	expect(t, "total", second(h.r.Propose(Change{Kind: ChangeReset, Reset: Reset{At: 1, Reason: "eve"}}, Actor{ID: "eve", Kind: KindHuman})), ErrLimit)
	k := setup(t)
	first := k.approve(Change{Kind: ChangeReset, Reset: Reset{At: 1, Reason: "a"}})
	for n := 2; n < maxDone; n++ {
		k.r.done = append(k.r.done, uint64(100000+n))
	}
	kept := k.approve(Change{Kind: ChangeReset, Reset: Reset{At: 2, Reason: "b"}})
	if _, err := k.r.Proposal(first.Number); err != nil {
		t.Fatal("pruned early")
	}
	k.approve(Change{Kind: ChangeReset, Reset: Reset{At: 3, Reason: "c"}})
	expect(t, "pruned", second(k.r.Proposal(first.Number)), ErrNotFound)
	if len(k.r.done) != 1000 {
		t.Errorf("done %d", len(k.r.done))
	}
	if _, err := k.r.Proposal(kept.Number); err != nil {
		t.Error("pruned two")
	}
}

func TestRegisterForms(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	doc := f.sign(commercial("e1", "agr", start, start.Add(time.Hour)))
	reason(t, "assistant", second(f.r.Register("i1", doc, bot)), ReasonAINotAllowed)
	expect(t, "form", second(f.r.Register("", doc, ann)), ErrInvalid)
	expect(t, "missing", second(f.r.Register("i9", doc, ann)), ErrNotFound)
	for n := 0; n < maxHistory+3; n++ {
		f.must(second(f.r.Register("i1", Signed{}, ann)))
	}
	if _, subs, _ := f.r.History("i1"); len(subs) != maxHistory || subs[0].Outcome != OutcomeInvalid {
		t.Errorf("submissions %d", len(subs))
	}
	for len(f.r.documents) < maxRecords {
		f.r.documents["x"+itoa(len(f.r.documents))] = &entitlement{}
	}
	expect(t, "documents cap", second(f.r.Register("i1", doc, ann)), ErrLimit)
}

func TestConcurrency(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for m := 0; m < 20; m++ {
				c := commercial("e"+itoa(n)+"-"+itoa(m), "agr-"+itoa(n)+"-"+itoa(m), start, start.Add(time.Hour))
				c.SubjectID = "s" + itoa(n) + "-" + itoa(m)
				f.r.Register("i1", f.sign(c), ann)
				f.r.Status("i1")
				f.r.History("i1")
			}
		}(n)
	}
	wg.Wait()
	_, subs, _ := f.r.History("i1")
	accepted := 0
	for _, sub := range subs {
		if sub.Outcome == OutcomeAccepted {
			accepted++
		}
	}
	if len(f.r.documents) < 1 || len(subs) != maxHistory {
		t.Errorf("documents %d accepted %d", len(f.r.documents), accepted)
	}
}

func TestSameAgreementChains(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	day := 24 * time.Hour
	e0 := commercial("e0", "agr-a", start, start.Add(100*day))
	if out := reg(t, f, e0); out != OutcomeAccepted {
		t.Fatal(out)
	}
	f.advance(90 * day)
	e1 := commercial("e1", "agr-a", start.Add(90*day), start.Add(200*day))
	e1.Supersedes, e1.IssuedAt = "agr-a", start.Add(90*day).Unix()
	if out := reg(t, f, e1); out != OutcomeAccepted {
		t.Fatalf("first renewal %s", out)
	}
	f.advance(105 * day)
	e2 := commercial("e2", "agr-a", start.Add(190*day), start.Add(300*day))
	e2.Supersedes, e2.IssuedAt = "agr-a", start.Add(195*day).Unix()-10
	if out := reg(t, f, e2); out != OutcomeAccepted {
		t.Fatalf("second renewal %s", out)
	}
	if v := status(t, f, "i1"); v.State != StateCommercialActive || v.Entitlement.ID != "e2" {
		t.Errorf("second renewal current %+v %s", v.Entitlement, v.State)
	}
	e3 := commercial("e3", "agr-c", start.Add(195*day), start.Add(400*day))
	e3.IssuedAt = start.Add(195 * day).Unix()
	if out := reg(t, f, e3); out != OutcomeDuplicateRight {
		t.Errorf("duplicate after chain %s", out)
	}
	g := standard(t)
	g.must(g.r.SelectMode("i1", ModeCommercial, ann))
	a := commercial("a", "agr-a", start, start.Add(100*day))
	reg(t, g, a)
	g.advance(10 * day)
	b := commercial("b", "agr-b", start.Add(10*day), start.Add(200*day))
	b.Supersedes, b.Channel, b.IssuedAt = "agr-a", ChannelMarketplace, start.Add(10*day).Unix()
	if out := reg(t, g, b); out != OutcomeAccepted {
		t.Fatal(out)
	}
	g.advance(10 * day)
	back := commercial("a2", "agr-a", start.Add(20*day), start.Add(300*day))
	back.Supersedes, back.IssuedAt = "agr-b", start.Add(20*day).Unix()-10
	if out := reg(t, g, back); out != OutcomeAccepted {
		t.Fatalf("migration back %s", out)
	}
	if v := status(t, g, "i1"); v.Entitlement.ID != "a2" || v.State != StateCommercialActive {
		t.Errorf("back current %+v", v.Entitlement)
	}
	dup := commercial("x", "agr-x", start.Add(20*day), start.Add(30*day))
	dup.IssuedAt = start.Add(20 * day).Unix()
	if out := reg(t, g, dup); out != OutcomeDuplicateRight {
		t.Errorf("duplicate after migration back %s", out)
	}
}
