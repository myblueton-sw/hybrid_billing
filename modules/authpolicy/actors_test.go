package authpolicy

import (
	"testing"
	"time"
)

func TestActors(t *testing.T) {
	f := setup(t)
	c := Change{Kind: ChangeRevision, Revision: open()}
	reason(t, "assistant", second(f.r.Propose(c, Principal{User: "x", Workspace: "w1", Role: RoleAdministrator, Kind: KindAssistant}, f.strong())), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, Principal{User: "x", Workspace: "w1", Role: RoleAdministrator, Kind: KindSystem}, f.strong())), ReasonNotHuman)
	expect(t, "kind", second(f.r.Propose(c, Principal{User: "x", Workspace: "w1", Role: RoleAdministrator, Kind: "robot"}, f.strong())), ErrInvalid)
	expect(t, "user", second(f.r.Propose(c, Principal{Workspace: "w1", Role: RoleAdministrator, Kind: KindHuman}, f.strong())), ErrInvalid)
	expect(t, "change form", second(f.r.Propose(Change{Kind: "x"}, ann, f.strong())), ErrInvalid)
	expect(t, "auth form", second(f.r.Propose(c, ann, Authentication{Method: MethodPassword})), ErrInvalid)
	reason(t, "other workspace", second(f.r.Propose(c, Principal{User: "x", Workspace: "w2", Role: RoleAdministrator, Kind: KindHuman}, f.strong())), ReasonOtherWorkspace)
	reason(t, "member", second(f.r.Propose(c, member("m1"), f.strong())), ReasonNotAdministrator)
	reason(t, "plan before policy", second(f.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: []string{"x"}, Chunk: 1, Reason: "r"}}, ann, f.strong())), ReasonNoPolicy)
	f.r.workspaces["w2"] = &workspace{revisions: []RevisionView{{Revision: open(), Number: 1}}}
	reason(t, "lift other workspace", second(f.r.Propose(Change{Kind: ChangeLift, Lift: Lift{Plan: "w1/7", Reason: "r"}}, Principal{User: "zed", Workspace: "w2", Role: RoleAdministrator, Kind: KindHuman}, f.strong())), ReasonOtherWorkspace)
	delete(f.r.workspaces, "w2")
	reason(t, "lift before policy", second(f.r.Propose(Change{Kind: ChangeLift, Lift: Lift{Plan: "w1/1", Reason: "r"}}, ann, f.strong())), ReasonNoPolicy)
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
	reason(t, "approver stale for new revision", second(f.r.Approve(p.Number, bob, stale)), ReasonApproverLockedOut)
	v, err := f.r.Approve(p.Number, bob, f.weak())
	if err != nil || v.State != ProposalApplied || v.Approver != "bob" || v.Workspace != "w1" {
		t.Fatalf("applied %+v %v", v, err)
	}
	if again, err := f.r.Approve(p.Number, bob, f.weak()); err != nil || again.State != ProposalApplied {
		t.Error("repeat approve")
	}
	reason(t, "applied", second(f.r.Approve(p.Number, cat, f.weak())), ReasonNotPending)
	reason(t, "stale proposer", second(f.r.Propose(c, ann, stale)), ReasonStale)
	tight := Revision{Workspace: "w1", Methods: []string{MethodOIDC}, Issuers: []string{"idp"}, MFA: MFAEveryone, MaxAge: time.Hour}
	q, _ := f.r.Propose(Change{Kind: ChangeRevision, Revision: tight}, ann, f.weak())
	reason(t, "weak approver locked out", second(f.r.Approve(q.Number, bob, f.weak())), ReasonApproverLockedOut)
	f.must(second(f.r.Approve(q.Number, bob, f.strong())))
	loose := open()
	loose.MFA = MFANone
	r, err := f.r.Propose(Change{Kind: ChangeRevision, Revision: loose}, ann, f.strong())
	f.must(err)
	reason(t, "relaxation with weak sign-in", second(f.r.Approve(r.Number, bob, f.weak())), ReasonMethodNotAllowed)
	reason(t, "withdraw other", second(f.r.Withdraw(r.Number, bob, f.strong())), ReasonNotProposer)
	reason(t, "withdraw weak", second(f.r.Withdraw(r.Number, ann, f.weak())), ReasonMethodNotAllowed)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, ann, f.strong())), ErrNotFound)
	if w, err := f.r.Withdraw(r.Number, ann, f.strong()); err != nil || w.State != ProposalWithdrawn {
		t.Error("withdraw")
	}
	expect(t, "withdrawn", second(f.r.Proposal(r.Number)), ErrNotFound)
	if got, err := f.r.Proposal(q.Number); err != nil || got.State != ProposalApplied {
		t.Error("proposal read")
	}
	s, _ := f.r.Propose(Change{Kind: ChangeRevision, Revision: tight}, ann, f.strong())
	f.must(second(f.r.Approve(s.Number, bob, f.strong())))
	reason(t, "withdraw applied", second(f.r.Withdraw(s.Number, ann, f.strong())), ReasonNotPending)
	got, _ := f.r.Proposal(s.Number)
	got.Change.Revision.Methods[0] = "zzz"
	if again, _ := f.r.Proposal(s.Number); again.Change.Revision.Methods[0] != MethodOIDC {
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
		p, err := f.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: []string{"u1", "u2"}, Chunk: 1, Reason: "r"}}, ann, f.strong())
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
	f.r.workspaces["w2"] = &workspace{revisions: []RevisionView{{Revision: open(), Number: 1}}, number: 1}
	other := Principal{User: "zed", Workspace: "w2", Role: RoleAdministrator, Kind: KindHuman}
	p, _ := f.r.Propose(Change{Kind: ChangeRevision, Revision: Revision{Workspace: "w2", Methods: []string{MethodOIDC}, MFA: MFANone, MaxAge: time.Hour}}, other, f.strong())
	f.revise(open())
	if _, err := f.r.Proposal(p.Number); err != nil {
		t.Error("other workspace proposal dropped")
	}
}

func TestCaps(t *testing.T) {
	f := standard(t)
	for n := 0; n < 25; n++ {
		x := open()
		x.MaxAge = time.Duration(n+2) * time.Hour
		f.must(second(f.r.Propose(Change{Kind: ChangeRevision, Revision: x}, ann, f.strong())))
	}
	extra := open()
	extra.MaxAge = 100 * time.Hour
	expect(t, "mine", second(f.r.Propose(Change{Kind: ChangeRevision, Revision: extra}, ann, f.strong())), ErrLimit)
	for _, who := range []string{"bob", "cat", "dan"} {
		for n := 0; n < 25; n++ {
			x := open()
			x.MaxAge = time.Duration(n+2) * time.Hour
			f.must(second(f.r.Propose(Change{Kind: ChangeRevision, Revision: x}, admin(who), f.strong())))
		}
	}
	expect(t, "workspace", second(f.r.Propose(Change{Kind: ChangeRevision, Revision: open()}, admin("eve"), f.strong())), ErrLimit)
	z := standard(t)
	for n := uint64(1000); len(z.r.pending) < 10000; n++ {
		z.r.proposals[n] = &proposal{view: ProposalView{Workspace: "zz", Proposer: "p"}}
		z.r.pending = append(z.r.pending, n)
	}
	expect(t, "global pending", second(z.r.Propose(Change{Kind: ChangeRevision, Revision: extra}, ann, z.strong())), ErrLimit)
	z.r.pending = z.r.pending[1:]
	z.must(second(z.r.Propose(Change{Kind: ChangeRevision, Revision: extra}, ann, z.strong())))
	g := setup(t)
	for len(g.r.workspaces) < 10000 {
		g.r.workspaces["z"+itoa(len(g.r.workspaces))] = &workspace{revisions: []RevisionView{{}}}
	}
	expect(t, "workspaces", g.try(Change{Kind: ChangeRevision, Revision: open()}), ErrLimit)
	delete(g.r.workspaces, "z0")
	g.must(g.try(Change{Kind: ChangeRevision, Revision: open()}))
	h := standard(t)
	for n := 0; n < 99; n++ {
		h.block([]string{"u" + itoa(n)}, 1)
	}
	h.must(second(h.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: []string{"last"}, Chunk: 1, Reason: "r"}}, cat, h.strong())))
	h.block([]string{"u100"}, 1)
	expect(t, "active plans", h.try(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: []string{"over"}, Chunk: 1, Reason: "r"}}), ErrLimit)
	users := func(prefix string, n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = prefix + itoa(1000000+i)
		}
		return out
	}
	q := standard(t)
	q.must(second(q.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: users("a", 100000), Chunk: 1000, Reason: "r"}}, ann, q.strong())))
	expect(t, "space entries", second(q.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: []string{"b"}, Chunk: 1, Reason: "r"}}, bob, q.strong())), ErrLimit)
	e := standard(t)
	e.r.entries = 2000000 - 5
	e.must(second(e.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: users("a", 5), Chunk: 1, Reason: "r"}}, ann, e.strong())))
	e.must(second(e.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: users("b", 995), Chunk: 1, Reason: "r"}}, bob, e.strong())))
	expect(t, "beyond reservation", second(e.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: users("c", 1), Chunk: 1, Reason: "r"}}, cat, e.strong())), ErrLimit)
	if e.r.usage["w1"] != 1000 || e.r.entries != 2000995 {
		t.Errorf("usage %d %d", e.r.usage["w1"], e.r.entries)
	}
	g2 := standard(t)
	g2.r.entries = 2000000 - 1001
	g2.must(second(g2.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: users("a", 1001), Chunk: 1, Reason: "r"}}, ann, g2.strong())))
	g3 := standard(t)
	g3.r.entries = 2000000 - 1000
	expect(t, "global over by one", second(g3.r.Propose(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: users("a", 1001), Chunk: 1, Reason: "r"}}, ann, g3.strong())), ErrLimit)
	d := setup(t)
	first := d.by(Change{Kind: ChangeRevision, Revision: open()}, ann, bob)
	for n := 2; n < 1000; n++ {
		d.r.done = append(d.r.done, uint64(100000+n))
	}
	d.revise(open())
	if _, err := d.r.Proposal(first.Number); err != nil {
		t.Fatal("pruned early")
	}
	d.revise(open())
	expect(t, "pruned", second(d.r.Proposal(first.Number)), ErrNotFound)
	if len(d.r.done) != 1000 {
		t.Error("done kept")
	}
}

func TestStepUpCopies(t *testing.T) {
	f := setup(t)
	x := open()
	x.StepUp = []StepUp{{Action: "export", MaxAge: time.Hour}}
	p, _ := f.r.Propose(Change{Kind: ChangeRevision, Revision: x}, ann, f.strong())
	x.StepUp[0].Action = "zz"
	p.Change.Revision.StepUp[0].Action = "yy"
	f.must(second(f.r.Approve(p.Number, bob, f.strong())))
	v, _ := f.r.Policy("w1")
	v.Revision.StepUp[0].Action = "xx"
	if w, _ := f.r.Policy("w1"); w.Revision.StepUp[0].Action != "export" {
		t.Errorf("step-up shared %+v", w.Revision.StepUp)
	}
}
