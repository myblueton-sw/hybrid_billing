package authpolicy

import (
	"testing"
	"time"
)

func TestActors(t *testing.T) {
	f := setup(t)
	c := planOf("x")
	reason(t, "plan before policy", second(f.r.Propose(c, ann, f.strong())), ReasonNoPolicy)
	reason(t, "lift before policy", second(f.r.Propose(Change{Kind: ChangeLift, Lift: Lift{Plan: "w1/1", Reason: "r"}}, ann, f.strong())), ReasonNoPolicy)
	f.revise(open())
	reason(t, "assistant", second(f.r.Propose(c, Principal{User: "x", Workspace: "w1", Role: RoleAdministrator, Kind: KindAssistant}, f.strong())), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, Principal{User: "x", Workspace: "w1", Role: RoleAdministrator, Kind: KindSystem}, f.strong())), ReasonNotHuman)
	expect(t, "kind", second(f.r.Propose(c, Principal{User: "x", Workspace: "w1", Role: RoleAdministrator, Kind: "robot"}, f.strong())), ErrInvalid)
	expect(t, "user", second(f.r.Propose(c, Principal{Workspace: "w1", Role: RoleAdministrator, Kind: KindHuman}, f.strong())), ErrInvalid)
	expect(t, "change form", second(f.r.Propose(Change{Kind: "x"}, ann, f.strong())), ErrInvalid)
	expect(t, "a revision cannot be proposed", second(f.r.Propose(Change{Kind: ChangeRevision, Revision: open()}, ann, f.strong())), ErrInvalid)
	expect(t, "auth form", second(f.r.Propose(c, ann, Authentication{Method: MethodPassword})), ErrInvalid)
	reason(t, "other workspace", second(f.r.Propose(c, Principal{User: "x", Workspace: "w2", Role: RoleAdministrator, Kind: KindHuman}, f.strong())), ReasonOtherWorkspace)
	reason(t, "member", second(f.r.Propose(c, member("m1"), f.strong())), ReasonNotAdministrator)
	f.r.workspaces["w2"] = &workspace{revisions: []RevisionView{{Revision: open(), Number: 1}}}
	reason(t, "lift other workspace", second(f.r.Propose(Change{Kind: ChangeLift, Lift: Lift{Plan: "w1/7", Reason: "r"}}, Principal{User: "zed", Workspace: "w2", Role: RoleAdministrator, Kind: KindHuman}, f.strong())), ReasonOtherWorkspace)
	delete(f.r.workspaces, "w2")
	p, err := f.r.Propose(c, ann, f.weak())
	f.must(err)
	if again, _ := f.r.Propose(c, ann, f.strong()); again.Number != p.Number {
		t.Error("repeat")
	}
	reason(t, "self", second(f.r.Approve(p.Number, ann, f.strong())), ReasonSelfApproval)
	expect(t, "approve missing", second(f.r.Approve(99, bob, f.strong())), ErrNotFound)
	reason(t, "approve member", second(f.r.Approve(p.Number, member("m1"), f.strong())), ReasonNotAdministrator)
	stale := f.weak()
	stale.At = f.at().Add(-25 * time.Hour)
	reason(t, "approver stale", second(f.r.Approve(p.Number, bob, stale)), ReasonStale)
	v, err := f.r.Approve(p.Number, bob, f.weak())
	if err != nil || v.State != ProposalApplied || v.Approver != "bob" || v.Workspace != "w1" {
		t.Fatalf("applied %+v %v", v, err)
	}
	if again, err := f.r.Approve(p.Number, bob, f.weak()); err != nil || again.State != ProposalApplied {
		t.Error("repeat approve")
	}
	reason(t, "applied", second(f.r.Approve(p.Number, cat, f.weak())), ReasonNotPending)
	reason(t, "stale proposer", second(f.r.Propose(planOf("y"), ann, stale)), ReasonStale)
	tight := Revision{Workspace: "w1", Methods: []string{MethodOIDC}, Issuers: []string{"idp"}, MFA: MFAEveryone, MaxAge: time.Hour}
	f.revise(tight)
	reason(t, "a weak sign-in under a tight policy", second(f.r.Propose(planOf("y"), ann, f.weak())), ReasonMethodNotAllowed)
	r, err := f.r.Propose(planOf("z"), ann, f.strong())
	f.must(err)
	reason(t, "withdraw other", second(f.r.Withdraw(r.Number, bob, f.strong())), ReasonNotProposer)
	reason(t, "withdraw weak", second(f.r.Withdraw(r.Number, ann, f.weak())), ReasonMethodNotAllowed)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, ann, f.strong())), ErrNotFound)
	if w, err := f.r.Withdraw(r.Number, ann, f.strong()); err != nil || w.State != ProposalWithdrawn {
		t.Error("withdraw")
	}
	expect(t, "withdrawn", second(f.r.Proposal(r.Number)), ErrNotFound)
	if got, err := f.r.Proposal(p.Number); err != nil || got.State != ProposalApplied {
		t.Error("proposal read")
	}
	s, _ := f.r.Propose(planOf("s1", "s2"), ann, f.strong())
	f.must(second(f.r.Approve(s.Number, bob, f.strong())))
	reason(t, "withdraw applied", second(f.r.Withdraw(s.Number, ann, f.strong())), ReasonNotPending)
	got, _ := f.r.Proposal(s.Number)
	got.Change.Plan.Users[0] = "zzz"
	if again, _ := f.r.Proposal(s.Number); again.Change.Plan.Users[0] != "s1" {
		t.Error("proposal shares memory")
	}
}

func TestStaleness(t *testing.T) {
	kinds := map[string]func(f world, id string){
		"revision": func(f world, _ string) { f.revise(open()) },
		"plan":     func(f world, _ string) { f.block([]string{"zz"}, 1) },
		"lift":     func(f world, id string) { f.by(Change{Kind: ChangeLift, Lift: Lift{Plan: id, Reason: "x"}}, ann, bob) },
	}
	for name, change := range kinds {
		f := standard(t)
		id := f.block([]string{"y1"}, 1)
		p, err := f.r.Propose(planOf("u1", "u2"), ann, f.strong())
		f.must(err)
		before := f.r.entries
		if before != 3 {
			t.Fatalf("%s before %d", name, before)
		}
		change(f, id)
		expect(t, name+" approve", second(f.r.Approve(p.Number, bob, f.strong())), ErrNotFound)
		expect(t, name+" withdraw", second(f.r.Withdraw(p.Number, ann, f.strong())), ErrNotFound)
		if want := map[string]int{"revision": 1, "plan": 2, "lift": 0}[name]; f.r.entries != want || f.r.usage["w1"] != want {
			t.Errorf("%s entries %d want %d", name, f.r.entries, want)
		}
	}
	f := standard(t)
	other := Principal{User: "zed", Workspace: "w2", Role: RoleAdministrator, Kind: KindHuman}
	f.revise(Revision{Workspace: "w2", Methods: []string{MethodOIDC}, MFA: MFANone, MaxAge: time.Hour})
	p, err := f.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w2", Users: []string{"q"}, Chunk: 1, Reason: "r"}}, other, f.strong())
	f.must(err)
	f.revise(open())
	if _, err := f.r.Proposal(p.Number); err != nil {
		t.Error("other workspace proposal dropped")
	}
}

func TestCaps(t *testing.T) {
	f := standard(t)
	for n := 0; n < 25; n++ {
		f.must(second(f.r.Propose(planOf("a"+itoa(n)), ann, f.strong())))
	}
	expect(t, "mine", second(f.r.Propose(planOf("extra"), ann, f.strong())), ErrLimit)
	for _, who := range []string{"bob", "cat", "dan"} {
		for n := 0; n < 25; n++ {
			f.must(second(f.r.Propose(planOf(who+itoa(n)), admin(who), f.strong())))
		}
	}
	expect(t, "workspace", second(f.r.Propose(planOf("eve"), admin("eve"), f.strong())), ErrLimit)
	z := standard(t)
	for n := uint64(1000); len(z.r.pending) < 10000; n++ {
		z.r.proposals[n] = &proposal{view: ProposalView{Workspace: "zz", Proposer: "p"}}
		z.r.pending = append(z.r.pending, n)
	}
	expect(t, "global pending", second(z.r.Propose(planOf("extra"), ann, z.strong())), ErrLimit)
	z.r.pending = z.r.pending[1:]
	z.must(second(z.r.Propose(planOf("extra"), ann, z.strong())))
	g := setup(t)
	for len(g.r.workspaces) < 10000 {
		g.r.workspaces["z"+itoa(len(g.r.workspaces))] = &workspace{revisions: []RevisionView{{}}}
	}
	expect(t, "workspaces", second(g.r.Publish(Publication{Revision: open(), Epoch: 1, Number: 1})), ErrLimit)
	expect(t, "workspaces can publish", g.r.CanPublish(Publication{Revision: open(), Epoch: 1, Number: 1}), ErrLimit)
	delete(g.r.workspaces, "z0")
	g.must(second(g.r.Publish(Publication{Revision: open(), Epoch: 1, Number: 1})))
	h := standard(t)
	for n := 0; n < 99; n++ {
		h.block([]string{"u" + itoa(n)}, 1)
	}
	h.must(second(h.r.Propose(planOf("last"), cat, h.strong())))
	h.block([]string{"u100"}, 1)
	expect(t, "active plans", h.try(planOf("over")), ErrLimit)
	users := func(prefix string, n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = prefix + itoa(1000000+i)
		}
		return out
	}
	q := standard(t)
	q.must(second(q.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: users("a", 100000), Chunk: 1000, Reason: "r"}}, ann, q.strong())))
	expect(t, "space entries", second(q.r.Propose(planOf("b"), bob, q.strong())), ErrLimit)
	e := standard(t)
	e.r.entries = 2000000 - 5
	e.must(second(e.r.Propose(planOf(users("a", 5)...), ann, e.strong())))
	e.must(second(e.r.Propose(planOf(users("b", 995)...), bob, e.strong())))
	expect(t, "beyond reservation", second(e.r.Propose(planOf(users("c", 1)...), cat, e.strong())), ErrLimit)
	if e.r.usage["w1"] != 1000 || e.r.entries != 2000995 {
		t.Errorf("usage %d %d", e.r.usage["w1"], e.r.entries)
	}
	g2 := standard(t)
	g2.r.entries = 2000000 - 1001
	g2.must(second(g2.r.Propose(planOf(users("a", 1001)...), ann, g2.strong())))
	g3 := standard(t)
	g3.r.entries = 2000000 - 1000
	expect(t, "global over by one", second(g3.r.Propose(planOf(users("a", 1001)...), ann, g3.strong())), ErrLimit)
	d := standard(t)
	first := d.by(planOf("p0"), ann, bob)
	for n := 2; n < 1000; n++ {
		d.r.done = append(d.r.done, uint64(100000+n))
	}
	d.by(planOf("p1"), ann, bob)
	if _, err := d.r.Proposal(first.Number); err != nil {
		t.Fatal("pruned early")
	}
	d.by(planOf("p2"), ann, bob)
	expect(t, "pruned", second(d.r.Proposal(first.Number)), ErrNotFound)
	if len(d.r.done) != 1000 {
		t.Error("done kept")
	}
	k := setup(t)
	for n := 0; n < 101; n++ {
		k.revise(open())
	}
	if len(k.r.workspaces["w1"].revisions) != 100 {
		t.Errorf("kept revisions %d", len(k.r.workspaces["w1"].revisions))
	}
	expect(t, "an early revision", second(k.r.Revision("w1", 1)), ErrNotFound)
}

func TestStepUpCopies(t *testing.T) {
	f := setup(t)
	x := open()
	x.StepUp = []StepUp{{Action: "export", MaxAge: time.Hour}}
	published, err := f.r.Publish(Publication{Revision: x, Epoch: 1, Number: 1})
	f.must(err)
	x.StepUp[0].Action = "zz"
	published.Revision.StepUp[0].Action = "yy"
	v, _ := f.r.Policy("w1")
	v.Revision.StepUp[0].Action = "xx"
	if w, _ := f.r.Policy("w1"); w.Revision.StepUp[0].Action != "export" {
		t.Errorf("step-up shared %+v", w.Revision.StepUp)
	}
}
