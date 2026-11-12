package identity

import (
	"testing"
	"time"
)

func TestTheSubjectViewAndItsGeneration(t *testing.T) {
	f := empty(t, testLimits)
	view := f.r.Subjects()
	generation := func() uint64 { return view.Generation() }
	steady := func(name string, run func()) {
		t.Helper()
		before := generation()
		run()
		if after := generation(); after != before {
			t.Errorf("%s changed the generation from %d to %d", name, before, after)
		}
	}
	moves := func(name string, run func()) {
		t.Helper()
		before := generation()
		run()
		if after := generation(); after != before+1 {
			t.Errorf("%s moved the generation from %d to %d", name, before, after)
		}
	}
	hour := Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}
	if f.r.InstallationID() != "synthetic-installation" || !f.r.Clock()().Equal(start) {
		t.Errorf("installation %q and clock %v", f.r.InstallationID(), f.r.Clock()())
	}
	steady("arming the bootstrap", func() { f.must(f.in.ArmBootstrap(v("boot"), hour, "w1")) })
	steady("a refused bootstrap", func() {
		f.refusal("wrong verifier", f.b.Bootstrap(v("guess"), "root", triple("root"), ""), ReasonVerifier)
	})
	moves("the bootstrap", func() { f.must(f.b.Bootstrap(v("boot"), "root", triple("root"), "")) })
	if joined, active := view.Member("w1", "root"); !active || joined != 1 {
		t.Errorf("the first administrator: joined %d active %v", joined, active)
	}
	moves("creating a user", func() { f.user("ann") })
	steady("a refused creation", func() { f.want("existing", f.in.CreateUser(operator, "ann", triple("ann-x"), ""), ErrConflict) })
	if kind, state, epoch, found := view.Subject("ann"); !found || kind != Human || state != Active || epoch != 1 {
		t.Errorf("a new user: %s %s %d %v", kind, state, epoch, found)
	}
	if _, _, _, found := view.Subject("nobody"); found {
		t.Errorf("an unknown user is found")
	}
	if joined, active := view.Member("w1", "ann"); active || joined != 0 {
		t.Errorf("a user without membership: joined %d active %v", joined, active)
	}
	admin := func() Verifier { return f.login("root") }
	steady("a login", func() { admin() })
	steady("an invitation", func() { f.must(f.w.Invite(admin(), "w1", "i1", Member, v("i1"), f.window(), "ann")) })
	steady("an invitation and its revocation", func() {
		f.must(f.w.Invite(admin(), "w1", "i0", Member, v("i0"), f.window(), "ann"))
		f.must(f.w.RevokeInvitation(admin(), "w1", "i0"))
	})
	ann := f.login("ann")
	steady("a refused acceptance", func() { f.refusal("wrong verifier", f.w.Accept("w1", "i1", v("guess"), ann), ReasonVerifier) })
	moves("an acceptance", func() { f.must(f.w.Accept("w1", "i1", v("i1"), ann)) })
	if joined, active := view.Member("w1", "ann"); !active || joined != 1 {
		t.Errorf("a first membership: joined %d active %v", joined, active)
	}
	if _, active := view.Member("w2", "ann"); active {
		t.Errorf("a member of one workspace is a member of another")
	}
	steady("an access and a recheck", func() {
		answer, err := f.b.Access(ann, "w1")
		f.must(err)
		f.must(f.b.Recheck(answer))
	})
	steady("ending a session", func() { f.must(f.b.EndSession(f.login("ann"))) })
	moves("a role change", func() { f.must(f.w.ChangeRole(admin(), "w1", "ann", Administrator, 1)) })
	steady("a refused role change", func() { f.want("stale", f.w.ChangeRole(admin(), "w1", "ann", Member, 1), ErrConflict) })
	moves("a membership suspension", func() { f.must(f.w.SuspendMembership(admin(), "w1", "ann", 2)) })
	if joined, active := view.Member("w1", "ann"); active || joined != 0 {
		t.Errorf("a suspended membership: joined %d active %v", joined, active)
	}
	moves("a membership reinstatement", func() { f.must(f.w.ReinstateMembership(admin(), "w1", "ann", 3)) })
	if joined, active := view.Member("w1", "ann"); !active || joined != 1 {
		t.Errorf("a reinstated membership: joined %d active %v", joined, active)
	}
	moves("a removal", func() { f.must(f.w.RemoveMembership(admin(), "w1", "ann", 4)) })
	if _, active := view.Member("w1", "ann"); active {
		t.Errorf("a removed member is active")
	}
	f.must(f.w.Invite(admin(), "w1", "i2", Member, v("i2"), f.window(), "ann"))
	moves("a second acceptance", func() { f.must(f.w.Accept("w1", "i2", v("i2"), f.login("ann"))) })
	if joined, active := view.Member("w1", "ann"); !active || joined != 2 {
		t.Errorf("a membership created again: joined %d active %v", joined, active)
	}
	moves("linking", func() { f.must(f.b.LinkIdentity(f.login("ann"), triple("ann-2"), "", "proof")) })
	moves("unlinking", func() { f.must(f.b.UnlinkIdentity(f.login("ann"), triple("ann-2"))) })
	steady("a refused unlinking", func() { f.refusal("last", f.b.UnlinkIdentity(f.login("ann"), triple("ann")), ReasonLastIdentity) })
	moves("a suspension", func() { f.must(f.in.SuspendUser(operator, "ann", 3)) })
	if kind, state, epoch, _ := view.Subject("ann"); kind != Human || state != Suspended || epoch != 4 {
		t.Errorf("a suspended user: %s %s %d", kind, state, epoch)
	}
	if _, active := view.Member("w1", "ann"); active {
		t.Errorf("a suspended user is an active member")
	}
	steady("a refused suspension", func() { f.want("twice", f.in.SuspendUser(operator, "ann", 4), ErrConflict) })
	moves("a reinstatement", func() { f.must(f.in.ReinstateUser(operator, "ann", 4)) })

	moves("creating a service user", func() { f.must(f.w.CreateServiceUser(admin(), "w1", "svc", "ann")) })
	if kind, state, epoch, found := view.Subject("svc"); !found || kind != Service || state != Active || epoch != 1 {
		t.Errorf("a service user: %s %s %d %v", kind, state, epoch, found)
	}
	if joined, active := view.Member("w1", "svc"); !active || joined != 1 {
		t.Errorf("a service user in its workspace: joined %d active %v", joined, active)
	}
	if _, active := view.Member("w2", "svc"); active {
		t.Errorf("a service user is a member of another workspace")
	}
	if creator, owner, found := view.Service("svc"); !found || creator != "root" || owner != "ann" {
		t.Errorf("service record %q %q %v", creator, owner, found)
	}
	if _, _, found := view.Service("ann"); found {
		t.Errorf("a human has a service record")
	}
	steady("a credential and its revocation", func() {
		f.must(f.w.AddCredential(admin(), "w1", "svc", "c1", v("c1"), hour))
		f.must(second(f.b.AccessByCredential("c1", v("c1"))))
		f.must(f.w.RevokeCredential(admin(), "w1", "c1"))
	})
	moves("suspending a service user", func() { f.must(f.w.SuspendServiceUser(admin(), "w1", "svc", 1)) })
	if _, active := view.Member("w1", "svc"); active {
		t.Errorf("a suspended service user is active in its workspace")
	}
	moves("reinstating a service user", func() { f.must(f.w.ReinstateServiceUser(admin(), "w1", "svc", 2)) })
	moves("suspending it again", func() { f.must(f.w.SuspendServiceUser(admin(), "w1", "svc", 3)) })
	moves("the operator's block", func() { f.must(f.in.SuspendUser(operator, "svc", 4)) })
	steady("a refused reinstatement by an administrator", func() { f.refusal("blocked", f.w.ReinstateServiceUser(admin(), "w1", "svc", 5), ReasonOperatorBlock) })
	moves("the operator's reinstatement", func() { f.must(f.in.ReinstateUser(operator, "svc", 5)) })
	moves("retiring a service user", func() { f.must(f.w.RetireServiceUser(admin(), "w1", "svc", 6)) })
	if _, active := view.Member("w1", "svc"); active {
		t.Errorf("a retired service user is active in its workspace")
	}
	steady("a refused departure", func() { f.want("last administrator", f.in.DepartUser(operator, "root", 1), ErrLastAdministrator) })
	moves("a departure", func() { f.must(f.in.DepartUser(operator, "ann", 5)) })
	if kind, state, _, found := view.Subject("ann"); !found || kind != Human || state != Departed {
		t.Errorf("a departed user: %s %s %v", kind, state, found)
	}
	if _, active := view.Member("w1", "ann"); active {
		t.Errorf("a departed user is an active member")
	}

	var unbuilt Registry
	var none *Subjects
	for _, handle := range []*Subjects{unbuilt.Subjects(), none, {}} {
		if _, _, _, found := handle.Subject("ann"); found {
			t.Errorf("an unconstructed view finds a subject")
		}
		if _, active := handle.Member("w1", "ann"); active {
			t.Errorf("an unconstructed view finds a member")
		}
		if _, _, found := handle.Service("svc"); found || handle.Generation() != 0 {
			t.Errorf("an unconstructed view answers")
		}
	}
	var missing *Registry
	if missing.InstallationID() != "" || missing.Clock() != nil {
		t.Errorf("a nil registry has an installation or a clock")
	}
}
