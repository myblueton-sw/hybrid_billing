package identity

import (
	"crypto/sha256"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestInvitations(t *testing.T) {
	f := setup(t)
	f.user("ann")
	f.user("bob")
	admin := f.login("root")
	week := f.window()
	invalid := map[string]error{
		"an invalid workspace":               f.w.Invite(admin, "", "i", Member, v("i"), week, "ann"),
		"an invalid identifier":              f.w.Invite(admin, "w1", "bad id", Member, v("i"), week, "ann"),
		"an invalid invitee":                 f.w.Invite(admin, "w1", "i", Member, v("i"), week, ""),
		"the service role":                   f.w.Invite(admin, "w1", "i", ServiceRole, v("i"), week, "ann"),
		"an unknown role":                    f.w.Invite(admin, "w1", "i", "owner", v("i"), week, "ann"),
		"an invalid verifier":                f.w.Invite(admin, "w1", "i", Member, Verifier{}, week, "ann"),
		"the empty-secret verifier":          f.w.Invite(admin, "w1", "i", Member, Verifier(sha256.Sum256(nil)), week, "ann"),
		"a validity above the limit":         f.w.Invite(admin, "w1", "i", Member, v("i"), Validity{NotBefore: start, NotAfter: start.Add(72*time.Hour + 1)}, "ann"),
		"a validity already over":            f.w.Invite(admin, "w1", "i", Member, v("i"), Validity{NotBefore: start.Add(-time.Hour), NotAfter: start}, "ann"),
		"an inverted validity in the future": f.w.Invite(admin, "w1", "i", Member, v("i"), Validity{NotBefore: start.Add(2 * time.Hour), NotAfter: start.Add(time.Hour)}, "ann"),
	}
	for name, err := range invalid {
		f.want("inviting with "+name, err, ErrInvalid)
	}
	f.must(f.w.Invite(admin, "w1", "i1", Member, v("i1"), Validity{NotBefore: start.Add(time.Minute), NotAfter: start.Add(time.Hour)}, "ann"))
	f.want("an existing identifier", f.w.Invite(admin, "w1", "i1", Member, v("x"), week, "bob"), ErrConflict)
	ann, bob := f.login("ann"), f.login("bob")
	f.want("accepting with an invalid verifier", f.w.Accept("w1", "i1", Verifier{}, ann), ErrInvalid)
	f.refusal("before the validity starts", f.w.Accept("w1", "i1", v("i1"), ann), ReasonExpired)
	*f.now = start.Add(time.Minute)
	f.refusal("the right verifier in another workspace", f.w.Accept("w2", "i1", v("i1"), ann), ReasonUnknown)
	f.refusal("an unknown invitation", f.w.Accept("w1", "ghost", v("i1"), ann), ReasonUnknown)
	f.refusal("a dead session", f.w.Accept("w1", "i1", v("i1"), v("nothing")), ReasonSession)
	f.refusal("another user with the right verifier", f.w.Accept("w1", "i1", v("i1"), bob), ReasonNotInvitee)
	for attempt := 0; attempt < 5; attempt++ {
		f.refusal("another user guessing", f.w.Accept("w1", "i1", v("guess"), bob), ReasonNotInvitee)
		f.refusal("a guess in another workspace", f.w.Accept("w2", "i1", v("guess"), ann), ReasonUnknown)
		f.refusal("a guess with a dead session", f.w.Accept("w1", "i1", v("guess"), v("nothing")), ReasonSession)
	}
	f.refusal("the invitee's first wrong verifier", f.w.Accept("w1", "i1", v("guess"), ann), ReasonVerifier)
	f.refusal("the invitee's second wrong verifier", f.w.Accept("w1", "i1", v("guess"), ann), ReasonVerifier)
	f.must(f.w.Accept("w1", "i1", v("i1"), ann))
	f.refusal("accepting twice", f.w.Accept("w1", "i1", v("i1"), ann), ReasonState)
	if answer, err := f.b.Access(ann, "w1"); err != nil || answer.Role() != Member || answer.MembershipVersion() != 1 {
		t.Errorf("the new member %+v %v", answer, err)
	}
	f.refusal("revoking an accepted invitation", f.w.RevokeInvitation(f.login("root"), "w1", "i1"), ReasonState)

	f.must(f.w.Invite(f.login("root"), "w1", "i2", Member, v("i2"), f.window(), "ann"))
	f.refusal("a user who is already a member", f.w.Accept("w1", "i2", v("i2"), ann), ReasonMember)

	f.must(f.w.Invite(f.login("root"), "w1", "i3", Member, v("i3"), f.window(), "bob"))
	for attempt := 1; attempt <= 3; attempt++ {
		f.refusal("a wrong verifier", f.w.Accept("w1", "i3", v("guess"), bob), ReasonVerifier)
	}
	f.refusal("the right verifier after the lock", f.w.Accept("w1", "i3", v("i3"), bob), ReasonState)

	f.must(f.w.Invite(f.login("root"), "w1", "i4", Member, v("i4"), Validity{NotBefore: *f.now, NotAfter: f.now.Add(time.Hour)}, "bob"))
	*f.now = f.now.Add(time.Hour)
	f.refusal("exactly at the end of the validity", f.w.Accept("w1", "i4", v("i4"), f.login("bob")), ReasonExpired)
	*f.now = f.now.Add(-time.Nanosecond)
	f.must(f.w.Accept("w1", "i4", v("i4"), f.login("bob")))

	f.user("cat")
	f.must(f.w.Invite(f.login("root"), "w1", "i5", Member, v("i5"), f.window(), "cat"))
	f.refusal("revoking from another workspace", f.w.RevokeInvitation(f.login("root"), "w2", "i5"), ReasonNotAdministrator)
	f.refusal("revoking by a member", f.w.RevokeInvitation(f.login("ann"), "w1", "i5"), ReasonNotAdministrator)
	f.refusal("revoking an unknown invitation", f.w.RevokeInvitation(f.login("root"), "w1", "ghost"), ReasonUnknown)
	f.must(f.w.RevokeInvitation(f.login("root"), "w1", "i5"))
	f.refusal("a revoked invitation", f.w.Accept("w1", "i5", v("i5"), f.login("cat")), ReasonState)
	f.refusal("revoking twice", f.w.RevokeInvitation(f.login("root"), "w1", "i5"), ReasonState)

	f.must(f.w.Invite(f.login("root"), "w1", "i6", Member, v("i6"), f.window(), "ghost-user"))
	f.must(f.w.Invite(f.login("root"), "w1", "i7", Member, v("i7"), f.window(), "cat"))
	cat := f.login("cat")
	f.must(f.in.SuspendUser(operator, "cat", 1))
	f.refusal("a suspended invitee with the session it held", f.w.Accept("w1", "i7", v("i7"), cat), ReasonSession)
	f.must(f.in.ReinstateUser(operator, "cat", 2))
	f.refusal("a reinstated invitee with the session it held before", f.w.Accept("w1", "i7", v("i7"), cat), ReasonSession)
	ended := f.login("cat")
	f.must(f.b.EndSession(ended))
	f.refusal("an invitee with an ended session", f.w.Accept("w1", "i7", v("i7"), ended), ReasonSession)
	idle := f.login("cat")
	*f.now = f.now.Add(31 * time.Minute)
	f.refusal("an invitee with an idle session", f.w.Accept("w1", "i7", v("i7"), idle), ReasonSession)
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "robot", "root"))
	f.must(f.w.RevokeInvitation(f.login("root"), "w1", "i6"))
	f.must(f.w.Invite(f.login("root"), "w1", "i8", Member, v("i8"), f.window(), "robot"))
	f.must(f.w.AddCredential(f.login("root"), "w1", "robot", "robot-key", v("robot-key"), f.window()))
	f.refusal("a credential verifier is not a session verifier", f.w.Accept("w1", "i8", v("i8"), v("robot-key")), ReasonSession)
	f.must(f.w.Accept("w1", "i7", v("i7"), f.login("cat")))
}

func TestInvitersAndTheirAuthority(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.member("adm", Administrator)
	f.user("bob")
	f.refusal("an invitation by a member", f.w.Invite(f.login("ann"), "w1", "x1", Member, v("x1"), f.window(), "bob"), ReasonNotAdministrator)
	f.refusal("an invitation by an administrator of another workspace", f.w.Invite(f.login("adm"), "w2", "x2", Member, v("x2"), f.window(), "bob"), ReasonNotAdministrator)
	f.refusal("an invitation with an unknown session", f.w.Invite(v("nothing"), "w1", "x3", Member, v("x3"), f.window(), "bob"), ReasonSession)
	held := f.login("adm")
	f.must(f.w.SuspendMembership(f.login("root"), "w1", "adm", 1))
	f.refusal("an invitation by an administrator with a suspended membership", f.w.Invite(held, "w1", "x4", Member, v("x4"), f.window(), "bob"), ReasonNotAdministrator)
	f.must(f.w.ReinstateMembership(f.login("root"), "w1", "adm", 2))
	f.must(f.in.SuspendUser(operator, "adm", 1))
	f.refusal("an invitation by a suspended user", f.w.Invite(held, "w1", "x5", Member, v("x5"), f.window(), "bob"), ReasonSession)
	changes := map[string]func(fixture){
		"suspended as a user": func(f fixture) { f.must(f.in.SuspendUser(operator, "adm", 1)) },
		"suspended and reinstated as a user": func(f fixture) {
			f.must(f.in.SuspendUser(operator, "adm", 1))
			f.must(f.in.ReinstateUser(operator, "adm", 2))
		},
		"demoted": func(f fixture) { f.must(f.w.ChangeRole(f.login("root"), "w1", "adm", Member, 1)) },
		"demoted and promoted again": func(f fixture) {
			f.must(f.w.ChangeRole(f.login("root"), "w1", "adm", Member, 1))
			f.must(f.w.ChangeRole(f.login("root"), "w1", "adm", Administrator, 2))
		},
		"suspended as a member": func(f fixture) { f.must(f.w.SuspendMembership(f.login("root"), "w1", "adm", 1)) },
		"suspended and reinstated as a member": func(f fixture) {
			f.must(f.w.SuspendMembership(f.login("root"), "w1", "adm", 1))
			f.must(f.w.ReinstateMembership(f.login("root"), "w1", "adm", 2))
		},
		"removed": func(f fixture) { f.must(f.w.RemoveMembership(f.login("root"), "w1", "adm", 1)) },
		"removed and added again": func(f fixture) {
			f.must(f.w.RemoveMembership(f.login("root"), "w1", "adm", 1))
			f.join("adm", Administrator)
		},
		"given another identity": func(f fixture) { f.must(f.b.LinkIdentity(f.login("adm"), triple("adm-2"), "", "proof")) },
	}
	for name, change := range changes {
		f := setup(t)
		f.member("adm", Administrator)
		f.user("bob")
		f.must(f.w.Invite(f.login("adm"), "w1", "inv", Member, v("inv"), f.window(), "bob"))
		change(f)
		f.refusal("an invitation whose inviter was "+name, f.w.Accept("w1", "inv", v("inv"), f.login("bob")), ReasonInviter)
	}
	limited := setup(t)
	limited.member("adm", Administrator)
	limited.user("bob")
	for _, id := range []string{"a", "b"} {
		limited.must(limited.w.Invite(limited.login("adm"), "w1", id, Member, v(id), limited.window(), "bob"))
	}
	limited.must(limited.w.Invite(limited.login("root"), "w1", "c", Member, v("c"), Validity{NotBefore: start.Add(time.Hour), NotAfter: start.Add(2 * time.Hour)}, "bob"))
	limited.want("a fourth pending invitation, one of them not started yet", limited.w.Invite(limited.login("root"), "w1", "d", Member, v("d"), limited.window(), "bob"), ErrLimit)
	limited.must(limited.w.RevokeInvitation(limited.login("root"), "w1", "a"))
	limited.must(limited.w.Invite(limited.login("root"), "w1", "d", Member, v("d"), limited.window(), "bob"))
	limited.want("the limit again", limited.w.Invite(limited.login("root"), "w1", "e", Member, v("e"), limited.window(), "bob"), ErrLimit)
	limited.must(limited.w.ChangeRole(limited.login("root"), "w1", "adm", Member, 1))
	limited.must(limited.w.Invite(limited.login("root"), "w1", "e", Member, v("e"), limited.window(), "bob"))
	limited.want("the limit without the invitation of the demoted inviter", limited.w.Invite(limited.login("root"), "w1", "g", Member, v("g"), limited.window(), "bob"), ErrLimit)
	*limited.now = start.Add(48 * time.Hour)
	limited.must(limited.w.Invite(limited.login("root"), "w1", "h", Member, v("h"), limited.window(), "bob"))
}

func TestMembershipOperations(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.member("bob", Member)
	f.user("out")
	admin := func() Verifier { return f.login("root") }
	f.want("an invalid target", f.w.ChangeRole(admin(), "w1", "", Administrator, 1), ErrInvalid)
	f.want("an unknown role", f.w.ChangeRole(admin(), "w1", "ann", "owner", 1), ErrInvalid)
	f.want("the service role", f.w.ChangeRole(admin(), "w1", "ann", ServiceRole, 1), ErrInvalid)
	f.refusal("a member acting", f.w.ChangeRole(f.login("ann"), "w1", "bob", Administrator, 1), ReasonNotAdministrator)
	f.refusal("a member suspending", f.w.SuspendMembership(f.login("ann"), "w1", "bob", 1), ReasonNotAdministrator)
	f.refusal("a member removing another", f.w.RemoveMembership(f.login("ann"), "w1", "bob", 1), ReasonNotAdministrator)
	f.refusal("a member promoting itself", f.w.ChangeRole(f.login("ann"), "w1", "ann", Administrator, 1), ReasonNotAdministrator)
	f.refusal("an administrator in another workspace", f.w.ChangeRole(admin(), "w2", "ann", Administrator, 1), ReasonNotAdministrator)
	f.refusal("a user without membership as target", f.w.ChangeRole(admin(), "w1", "out", Administrator, 1), ReasonUnknown)
	f.refusal("an unknown target", f.w.SuspendMembership(admin(), "w1", "ghost", 1), ReasonUnknown)
	f.want("a stale version", f.w.ChangeRole(admin(), "w1", "ann", Administrator, 2), ErrConflict)
	f.refusal("the role already held", f.w.ChangeRole(admin(), "w1", "ann", Member, 1), ReasonState)
	f.refusal("reinstating an active membership", f.w.ReinstateMembership(admin(), "w1", "ann", 1), ReasonState)
	f.must(f.w.ChangeRole(admin(), "w1", "ann", Administrator, 1))
	f.must(f.w.SuspendMembership(f.login("ann"), "w1", "bob", 1))
	f.refusal("suspending twice", f.w.SuspendMembership(admin(), "w1", "bob", 2), ReasonState)
	f.refusal("access with a suspended membership", second(f.b.Access(f.login("bob"), "w1")), ReasonMembership)
	f.refusal("a suspended member leaving by itself", f.w.RemoveMembership(f.login("bob"), "w1", "bob", 2), ReasonNotAdministrator)
	f.must(f.w.ReinstateMembership(admin(), "w1", "bob", 2))
	if got := f.version("bob"); got != 3 {
		t.Errorf("version after suspension and reinstatement %d", got)
	}
	f.must(f.w.RemoveMembership(f.login("bob"), "w1", "bob", 3))
	f.refusal("a removed member as target", f.w.ReinstateMembership(admin(), "w1", "bob", 4), ReasonUnknown)
	f.refusal("access after leaving", second(f.b.Access(f.login("bob"), "w1")), ReasonMembership)
	f.join("bob", Administrator)
	if got := f.version("bob"); got != 5 {
		t.Errorf("version after removal and a new invitation %d", got)
	}
	f.must(f.in.SuspendUser(operator, "bob", 1))
	f.must(f.w.ChangeRole(admin(), "w1", "bob", Member, 5))
	members, err := f.w.Members(admin(), "w1", "", 2)
	want := []MemberInfo{{User: "ann", Role: Administrator, State: MembershipActive, Version: 2, UserState: Active}, {User: "bob", Role: Member, State: MembershipActive, Version: 6, UserState: Suspended}}
	if err != nil || !reflect.DeepEqual(members, want) {
		t.Errorf("first page %+v %v", members, err)
	}
	members, _ = f.w.Members(admin(), "w1", "bob", 2)
	if len(members) != 1 || members[0].User != "root" {
		t.Errorf("second page %+v", members)
	}
	for name, err := range map[string]error{"a zero page": second(f.w.Members(admin(), "w1", "", 0)), "a page above the bound": second(f.w.Members(admin(), "w1", "", 3)), "an invalid cursor": second(f.w.Members(admin(), "w1", "a b", 1))} {
		f.want(name, err, ErrInvalid)
	}
	f.refusal("a member listing", second(f.w.Members(f.login("out"), "w1", "", 1)), ReasonNotAdministrator)
	f.refusal("listing another workspace", second(f.w.Members(admin(), "w2", "", 1)), ReasonNotAdministrator)
}

func TestTheLastAdministrator(t *testing.T) {
	type world struct {
		fixture
	}
	alone := func(t *testing.T) world {
		f := setup(t)
		f.member("ann", Member)
		return world{f}
	}
	w := alone(t)
	root := func(w world) Verifier { return w.login("root") }
	w.want("demoting the last administrator", w.w.ChangeRole(root(w), "w1", "root", Member, 1), ErrLastAdministrator)
	w.want("suspending the last administrator's membership", w.w.SuspendMembership(root(w), "w1", "root", 1), ErrLastAdministrator)
	w.want("removing the last administrator", w.w.RemoveMembership(root(w), "w1", "root", 1), ErrLastAdministrator)
	w.want("the last administrator departing", w.in.DepartUser(operator, "root", 1), ErrLastAdministrator)
	if list, _ := w.in.WithoutAdministrator(); len(list) != 0 {
		t.Errorf("workspaces without administrator %v", list)
	}

	sole := setup(t)
	sole.want("the sole member demoting itself", sole.w.ChangeRole(sole.login("root"), "w1", "root", Member, 1), ErrLastAdministrator)
	sole.want("the sole member suspending its own membership", sole.w.SuspendMembership(sole.login("root"), "w1", "root", 1), ErrLastAdministrator)
	sole.must(sole.w.RemoveMembership(sole.login("root"), "w1", "root", 1))
	if list, _ := sole.in.WithoutAdministrator(); len(list) != 0 {
		t.Errorf("an empty workspace is reported: %v", list)
	}

	leaving := setup(t)
	leaving.must(leaving.in.DepartUser(operator, "root", 1))

	withService := setup(t)
	withService.must(withService.w.CreateServiceUser(withService.login("root"), "w1", "svc", "root"))
	withService.want("the last administrator leaving a service user behind", withService.w.RemoveMembership(withService.login("root"), "w1", "root", 1), ErrLastAdministrator)
	withService.want("the last administrator departing with a service user", withService.in.DepartUser(operator, "root", 1), ErrLastAdministrator)
	withService.must(withService.w.RetireServiceUser(withService.login("root"), "w1", "svc", 1))
	withService.must(withService.w.RemoveMembership(withService.login("root"), "w1", "root", 1))

	second := func(t *testing.T) world {
		w := alone(t)
		w.member("adm", Administrator)
		return w
	}
	for name, run := range map[string]func(world) error{
		"demotion":              func(w world) error { return w.w.ChangeRole(root(w), "w1", "root", Member, 1) },
		"membership suspension": func(w world) error { return w.w.SuspendMembership(root(w), "w1", "root", 1) },
		"removal":               func(w world) error { return w.w.RemoveMembership(root(w), "w1", "root", 1) },
		"departure":             func(w world) error { return w.in.DepartUser(operator, "root", 1) },
	} {
		w := second(t)
		w.want(name+" with a second administrator", run(w), nil)
		w = second(t)
		w.must(w.in.SuspendUser(operator, "adm", 1))
		w.want(name+" with a second administrator suspended as a user", run(w), ErrLastAdministrator)
		w = second(t)
		w.must(w.w.SuspendMembership(root(w), "w1", "adm", 1))
		w.want(name+" with a second administrator suspended as a member", run(w), ErrLastAdministrator)
	}
	both := second(t)
	both.must(both.w.ChangeRole(both.login("adm"), "w1", "root", Member, 1))
	both.want("the remaining administrator demoting itself", both.w.ChangeRole(both.login("adm"), "w1", "adm", Member, 1), ErrLastAdministrator)

	blocked := alone(t)
	blocked.must(blocked.in.SuspendUser(operator, "root", 1))
	if list, _ := blocked.in.WithoutAdministrator(); !reflect.DeepEqual(list, []string{"w1"}) {
		t.Errorf("after a security suspension of the last administrator: %v", list)
	}
	blocked.want("departure after a security suspension", blocked.in.DepartUser(operator, "root", 2), ErrLastAdministrator)
	blocked.must(blocked.w.RemoveMembership(blocked.login("ann"), "w1", "ann", 1))
	blocked.must(blocked.in.DepartUser(operator, "root", 2))
	if list, _ := blocked.in.WithoutAdministrator(); len(list) != 0 {
		t.Errorf("an emptied workspace is reported: %v", list)
	}

	recovered := alone(t)
	recovered.must(recovered.in.SuspendUser(operator, "root", 1))
	recovered.must(recovered.in.ReinstateUser(operator, "root", 2))
	if list, _ := recovered.in.WithoutAdministrator(); len(list) != 0 {
		t.Errorf("after reinstating the administrator: %v", list)
	}

	serviceOnly := setup(t)
	serviceOnly.must(serviceOnly.w.CreateServiceUser(serviceOnly.login("root"), "w1", "svc", "root"))
	serviceOnly.must(serviceOnly.in.SuspendUser(operator, "root", 1))
	if list, _ := serviceOnly.in.WithoutAdministrator(); !reflect.DeepEqual(list, []string{"w1"}) {
		t.Errorf("a workspace with a service user and a blocked administrator: %v", list)
	}
}

func TestServiceAccounts(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.user("out")
	admin := func() Verifier { return f.login("root") }
	day := Validity{NotBefore: start, NotAfter: start.Add(24 * time.Hour)}
	f.want("an invalid identifier", f.w.CreateServiceUser(admin(), "w1", "", "ann"), ErrInvalid)
	f.refusal("creation by a member", f.w.CreateServiceUser(f.login("ann"), "w1", "svc", "ann"), ReasonNotAdministrator)
	f.refusal("creation in another workspace", f.w.CreateServiceUser(admin(), "w2", "svc", "ann"), ReasonNotAdministrator)
	f.refusal("an owner who is not a member", f.w.CreateServiceUser(admin(), "w1", "svc", "out"), ReasonOwner)
	f.refusal("an unknown owner", f.w.CreateServiceUser(admin(), "w1", "svc", "ghost"), ReasonOwner)
	f.must(f.w.CreateServiceUser(admin(), "w1", "svc", "ann"))
	f.want("an existing identifier", f.w.CreateServiceUser(admin(), "w1", "svc", "ann"), ErrConflict)
	f.want("a human's identifier", f.w.CreateServiceUser(admin(), "w1", "ann", "ann"), ErrConflict)
	f.refusal("a service user as owner", f.w.CreateServiceUser(admin(), "w1", "svc2", "svc"), ReasonOwner)

	for name, err := range map[string]error{
		"an invalid credential identifier":   f.w.AddCredential(admin(), "w1", "svc", "", v("c"), day),
		"an invalid verifier":                f.w.AddCredential(admin(), "w1", "svc", "c", Verifier{}, day),
		"a validity above the limit":         f.w.AddCredential(admin(), "w1", "svc", "c", v("c"), Validity{NotBefore: start, NotAfter: start.Add(720*time.Hour + 1)}),
		"a validity without end":             f.w.AddCredential(admin(), "w1", "svc", "c", v("c"), Validity{NotBefore: start}),
		"a validity already over":            f.w.AddCredential(admin(), "w1", "svc", "c", v("c"), Validity{NotBefore: start.Add(-time.Hour), NotAfter: start}),
		"an inverted validity in the future": f.w.AddCredential(admin(), "w1", "svc", "c", v("c"), Validity{NotBefore: start.Add(2 * time.Hour), NotAfter: start.Add(time.Hour)}),
	} {
		f.want("a credential with "+name, err, ErrInvalid)
	}
	f.refusal("a credential by a member", f.w.AddCredential(f.login("ann"), "w1", "svc", "c1", v("c1"), day), ReasonNotAdministrator)
	f.refusal("a credential for a human", f.w.AddCredential(admin(), "w1", "ann", "c1", v("c1"), day), ReasonUnknown)
	f.refusal("a credential for an unknown service user", f.w.AddCredential(admin(), "w1", "ghost", "c1", v("c1"), day), ReasonUnknown)
	f.must(f.w.AddCredential(admin(), "w1", "svc", "c1", v("c1"), Validity{NotBefore: start.Add(time.Minute), NotAfter: start.Add(time.Hour)}))
	f.want("an existing credential identifier", f.w.AddCredential(admin(), "w1", "svc", "c1", v("x"), day), ErrConflict)
	f.refusal("before the credential starts", second(f.b.AccessByCredential("c1", v("c1"))), ReasonExpired)
	*f.now = start.Add(time.Minute)
	answer, err := f.b.AccessByCredential("c1", v("c1"))
	if err != nil || answer.Role() != ServiceRole || answer.Kind() != Service || answer.Workspace() != "w1" || answer.User() != "svc" || answer.Epoch() != 1 || !answer.NotAfter().Equal(start.Add(time.Hour)) {
		t.Fatalf("service answer %+v %v", answer, err)
	}
	f.must(f.b.Recheck(answer))
	for attempt := 0; attempt < 10; attempt++ {
		f.refusal("a wrong verifier", second(f.b.AccessByCredential("c1", v("guess"))), ReasonVerifier)
	}
	f.must(second(f.b.AccessByCredential("c1", v("c1"))))
	f.refusal("an unknown credential", second(f.b.AccessByCredential("ghost", v("c1"))), ReasonUnknown)
	f.want("an invalid credential identifier", second(f.b.AccessByCredential("", v("c1"))), ErrInvalid)
	*f.now = start.Add(time.Hour)
	f.refusal("exactly at the credential's end", second(f.b.AccessByCredential("c1", v("c1"))), ReasonExpired)
	f.refusal("recheck at the credential's end", f.b.Recheck(answer), ReasonAnswer)
	*f.now = start.Add(time.Minute)

	f.must(f.w.AddCredential(admin(), "w1", "svc", "c2", v("c2"), Validity{NotBefore: start.Add(2 * time.Hour), NotAfter: start.Add(3 * time.Hour)}))
	f.want("a third credential, one of them not started yet", f.w.AddCredential(admin(), "w1", "svc", "c3", v("c3"), day), ErrLimit)
	f.refusal("revoking by a member", f.w.RevokeCredential(f.login("ann"), "w1", "c1"), ReasonNotAdministrator)
	f.refusal("revoking an unknown credential", f.w.RevokeCredential(admin(), "w1", "ghost"), ReasonUnknown)
	f.must(f.w.RevokeCredential(admin(), "w1", "c1"))
	f.refusal("revoking twice", f.w.RevokeCredential(admin(), "w1", "c1"), ReasonState)
	f.refusal("a revoked credential", second(f.b.AccessByCredential("c1", v("c1"))), ReasonUnknown)
	f.refusal("recheck after revocation", f.b.Recheck(answer), ReasonAnswer)
	f.must(f.w.AddCredential(admin(), "w1", "svc", "c3", v("c3"), day))
	rotated, err := f.b.AccessByCredential("c3", v("c3"))
	f.must(err)

	f.refusal("a session for a service user", f.b.OpenSession(triple("svc"), v("s"), time.Minute, time.Hour), ReasonIdentity)
	f.must(f.w.Invite(admin(), "w1", "for-service", Administrator, v("for-service"), f.window(), "svc"))
	f.refusal("a membership operation on a service user", f.w.ChangeRole(admin(), "w1", "svc", Administrator, 1), ReasonUnknown)

	f.want("a stale epoch", f.w.SuspendServiceUser(admin(), "w1", "svc", 2), ErrConflict)
	f.refusal("suspension by a member", f.w.SuspendServiceUser(f.login("ann"), "w1", "svc", 1), ReasonNotAdministrator)
	f.refusal("suspension from another workspace", f.w.SuspendServiceUser(admin(), "w2", "svc", 1), ReasonNotAdministrator)
	f.refusal("a human as a service user", f.w.SuspendServiceUser(admin(), "w1", "ann", 1), ReasonUnknown)
	f.refusal("reinstating an active service user", f.w.ReinstateServiceUser(admin(), "w1", "svc", 1), ReasonState)
	f.must(f.w.SuspendServiceUser(admin(), "w1", "svc", 1))
	f.refusal("a suspended service user", second(f.b.AccessByCredential("c3", v("c3"))), ReasonUser)
	f.refusal("recheck during suspension", f.b.Recheck(rotated), ReasonAnswer)
	f.refusal("suspending twice", f.w.SuspendServiceUser(admin(), "w1", "svc", 2), ReasonState)
	f.want("a stale reinstatement", f.w.ReinstateServiceUser(admin(), "w1", "svc", 1), ErrConflict)
	f.must(f.w.ReinstateServiceUser(admin(), "w1", "svc", 2))
	f.refusal("the old answer after reinstatement", f.b.Recheck(rotated), ReasonAnswer)
	current, err := f.b.AccessByCredential("c3", v("c3"))
	if err != nil || current.Epoch() != 3 {
		t.Fatalf("answer after reinstatement %+v %v", current, err)
	}

	f.must(f.in.SuspendUser(operator, "svc", 3))
	f.refusal("after the operator's suspension", second(f.b.AccessByCredential("c3", v("c3"))), ReasonUser)
	f.refusal("an administrator lifting the operator's block", f.w.ReinstateServiceUser(admin(), "w1", "svc", 4), ReasonOperatorBlock)
	f.must(f.in.ReinstateUser(operator, "svc", 4))
	f.must(second(f.b.AccessByCredential("c3", v("c3"))))
	f.must(f.w.SuspendServiceUser(admin(), "w1", "svc", 5))
	f.must(f.w.ReinstateServiceUser(admin(), "w1", "svc", 6))

	services, err := f.w.ServiceUsers(admin(), "w1", "", 2)
	if err != nil || !reflect.DeepEqual(services, []ServiceInfo{{ID: "svc", State: Active, Epoch: 7, Creator: "root", Owner: "ann"}}) {
		t.Errorf("service users %+v %v", services, err)
	}
	f.refusal("a member listing service users", second(f.w.ServiceUsers(f.login("ann"), "w1", "", 1)), ReasonNotAdministrator)
	f.want("a page above the bound", second(f.w.ServiceUsers(admin(), "w1", "", 3)), ErrInvalid)
	if users, _ := f.in.Users("root", 2); len(users) != 1 || users[0] != (UserInfo{ID: "svc", Kind: Service, State: Active, Epoch: 7, Workspace: "w1"}) {
		t.Errorf("the service user in the installation listing %+v", users)
	}

	latest, err := f.b.AccessByCredential("c3", v("c3"))
	f.must(err)
	f.must(f.w.RetireServiceUser(admin(), "w1", "svc", 7))
	f.refusal("a retired service user", second(f.b.AccessByCredential("c3", v("c3"))), ReasonUser)
	f.refusal("recheck after retirement", f.b.Recheck(latest), ReasonAnswer)
	f.refusal("retiring twice", f.w.RetireServiceUser(admin(), "w1", "svc", 8), ReasonState)
	f.refusal("reinstating a retired service user", f.w.ReinstateServiceUser(admin(), "w1", "svc", 8), ReasonState)
	f.refusal("a credential for a retired service user", f.w.AddCredential(admin(), "w1", "svc", "c9", v("c9"), day), ReasonState)
	f.want("reusing a retired identifier", f.w.CreateServiceUser(admin(), "w1", "svc", "ann"), ErrConflict)
}

func TestServiceAccountsOutliveTheirCreatorAndBlockedAdministrators(t *testing.T) {
	f := setup(t)
	f.member("adm", Administrator)
	f.must(f.w.CreateServiceUser(f.login("adm"), "w1", "svc", "adm"))
	f.must(f.w.AddCredential(f.login("adm"), "w1", "svc", "c1", v("c1"), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	held, err := f.b.Access(f.login("adm"), "w1")
	f.must(err)
	f.must(f.in.DepartUser(operator, "adm", 1))
	answer, err := f.b.AccessByCredential("c1", v("c1"))
	if err != nil || answer.User() != "svc" || answer.Role() != ServiceRole {
		t.Fatalf("the account after its creator departed: %+v %v", answer, err)
	}
	f.refusal("the departed human's answer", f.b.Recheck(held), ReasonAnswer)
	members, _ := f.w.Members(f.login("root"), "w1", "", 2)
	if len(members) != 1 || members[0].User != "root" {
		t.Errorf("members after the creator departed %+v", members)
	}
	f.refusal("the service user holding the departed human's membership", second(f.b.Access(v("c1"), "w1")), ReasonSession)
	f.must(f.in.SuspendUser(operator, "root", 1))
	if list, _ := f.in.WithoutAdministrator(); !reflect.DeepEqual(list, []string{"w1"}) {
		t.Fatalf("without administrator: %v", list)
	}
	f.must(second(f.b.AccessByCredential("c1", v("c1"))))
	f.must(f.b.Recheck(answer))
	f.must(f.in.SuspendUser(operator, "svc", 1))
	f.refusal("the service account after the operator's block", second(f.b.AccessByCredential("c1", v("c1"))), ReasonUser)
	f.refusal("its earlier answer", f.b.Recheck(answer), ReasonAnswer)
	f.must(f.in.DepartUser(operator, "svc", 2))
	f.must(f.in.DepartUser(operator, "root", 2))
}

func TestNoSequenceRecoversWhatWasClosed(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.user("bob")
	hour := Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}
	f.must(f.in.DepartUser(operator, "ann", 1))
	f.refusal("re-arming the bootstrap", f.in.ArmBootstrap(v("again"), hour, "w9"), ReasonBootstrap)
	f.refusal("a second bootstrap", f.b.Bootstrap(v("boot"), "again", triple("again"), ""), ReasonBootstrap)
	f.want("relinking a departed user's triple through creation", f.in.CreateUser(operator, "ann-again", triple("ann"), ""), ErrConflict)
	f.want("relinking a departed user's triple through linking", f.b.LinkIdentity(f.login("bob"), triple("ann"), "", "proof"), ErrConflict)
	f.want("reactivating a departed user", f.in.ReinstateUser(operator, "ann", 2), ErrConflict)
	f.refusal("a membership in a workspace without administrator", f.w.Invite(f.login("root"), "w2", "i", Administrator, v("i"), f.window(), "bob"), ReasonNotAdministrator)
	f.refusal("a membership operation creating a membership", f.w.ReinstateMembership(f.login("root"), "w1", "bob", 0), ReasonUnknown)
	f.refusal("access to a workspace nobody was invited to", second(f.b.Access(f.login("bob"), "w2")), ReasonMembership)
	if _, exists := f.r.memberships["w2"]; exists {
		t.Errorf("a refused operation created a workspace record")
	}
}

func (f fixture) seed(workspace, administrator string) {
	f.t.Helper()
	f.user(administrator)
	f.r.mu.Lock()
	f.r.memberships[workspace] = map[string]*membership{administrator: {role: Administrator, state: MembershipActive, version: 1}}
	f.r.mu.Unlock()
}

func TestWorkspacesAreSeparate(t *testing.T) {
	f := setup(t)
	f.user("ann")
	f.seed("w2", "adm2")
	hour := Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}
	other := func() Verifier { return f.login("adm2") }
	own := func() Verifier { return f.login("root") }
	for _, id := range []string{"j1", "j2", "j3"} {
		f.must(f.w.Invite(other(), "w2", id, Member, v(id), f.window(), "ann"))
	}
	f.want("the pending limit of the other workspace", f.w.Invite(other(), "w2", "j4", Member, v("j4"), f.window(), "ann"), ErrLimit)
	f.must(f.w.Invite(own(), "w1", "k1", Member, v("k1"), f.window(), "ann"))
	ann := f.login("ann")
	f.refusal("an invitation of the other workspace accepted here", f.w.Accept("w1", "j1", v("j1"), ann), ReasonUnknown)
	f.refusal("revoking an invitation of the other workspace", f.w.RevokeInvitation(own(), "w1", "j2"), ReasonUnknown)
	f.must(f.w.Accept("w2", "j1", v("j1"), ann))
	there, err := f.b.Access(ann, "w2")
	if err != nil || there.Workspace() != "w2" || there.Role() != Member {
		t.Fatalf("access in the other workspace %+v %v", there, err)
	}
	f.refusal("access here with a membership there", second(f.b.Access(ann, "w1")), ReasonMembership)
	f.refusal("an administrator here acting on a member there", f.w.ChangeRole(own(), "w1", "ann", Administrator, 1), ReasonUnknown)
	f.refusal("an administrator here acting in the other workspace", f.w.ChangeRole(own(), "w2", "ann", Administrator, 1), ReasonNotAdministrator)
	f.refusal("an administrator there acting here", f.w.ChangeRole(other(), "w1", "root", Member, 1), ReasonNotAdministrator)

	f.must(f.w.CreateServiceUser(other(), "w2", "svc2", "adm2"))
	f.must(f.w.AddCredential(other(), "w2", "svc2", "c2", v("c2"), hour))
	f.refusal("suspending a service user of the other workspace", f.w.SuspendServiceUser(own(), "w1", "svc2", 1), ReasonUnknown)
	f.refusal("retiring a service user of the other workspace", f.w.RetireServiceUser(own(), "w1", "svc2", 1), ReasonUnknown)
	f.refusal("a credential for a service user of the other workspace", f.w.AddCredential(own(), "w1", "svc2", "c9", v("c9"), hour), ReasonUnknown)
	f.refusal("revoking a credential of the other workspace", f.w.RevokeCredential(own(), "w1", "c2"), ReasonUnknown)
	if listed, err := f.w.ServiceUsers(own(), "w1", "", 2); err != nil || len(listed) != 0 {
		t.Errorf("service users of the other workspace listed here: %+v %v", listed, err)
	}
	if listed, _ := f.w.ServiceUsers(other(), "w2", "", 2); len(listed) != 1 || listed[0].ID != "svc2" {
		t.Errorf("service users there: %+v", listed)
	}
	if members, _ := f.w.Members(other(), "w2", "", 2); len(members) != 2 || members[0].User != "adm2" || members[1].User != "ann" {
		t.Errorf("members there: %+v", members)
	}
	answer, err := f.b.AccessByCredential("c2", v("c2"))
	if err != nil || answer.Workspace() != "w2" {
		t.Fatalf("service answer %+v %v", answer, err)
	}
	f.want("the last administrator there leaving a service user and a member", f.w.RemoveMembership(other(), "w2", "adm2", 1), ErrLastAdministrator)
	f.must(f.w.RemoveMembership(own(), "w1", "root", 1))
	f.must(f.b.Recheck(there))
	f.must(f.b.Recheck(answer))
	if list, _ := f.in.WithoutAdministrator(); len(list) != 0 {
		t.Errorf("without administrator: %v", list)
	}
	f.r.mu.Lock()
	f.r.users["svc3"] = &user{kind: Service, state: Active, epoch: 1, workspace: "w3"}
	f.r.mu.Unlock()
	if list, _ := f.in.WithoutAdministrator(); !reflect.DeepEqual(list, []string{"w3"}) {
		t.Errorf("a workspace with only a service user: %v", list)
	}
}

func TestASuspensionThenADepartureKeepsAWayBack(t *testing.T) {
	f := setup(t)
	f.member("adm", Administrator)
	f.member("ann", Member)
	f.must(f.w.SuspendMembership(f.login("root"), "w1", "adm", 1))
	f.must(f.in.SuspendUser(operator, "root", 1))
	f.want("the blocked administrator departing while the other administrator's membership is suspended", f.in.DepartUser(operator, "root", 2), ErrLastAdministrator)
	f.must(f.in.ReinstateUser(operator, "root", 2))
	f.must(f.w.ReinstateMembership(f.login("root"), "w1", "adm", 2))
	f.must(f.in.SuspendUser(operator, "root", 3))
	f.must(f.in.DepartUser(operator, "root", 4))
	if list, _ := f.in.WithoutAdministrator(); len(list) != 0 {
		t.Errorf("without administrator: %v", list)
	}
	f.must(f.in.SuspendUser(operator, "adm", 1))
	f.want("the last administrator membership of a blocked user departing", f.in.DepartUser(operator, "adm", 2), ErrLastAdministrator)
}

func TestTheOperatorBlocksAServiceUserAnAdministratorSuspended(t *testing.T) {
	f := setup(t)
	admin := func() Verifier { return f.login("root") }
	f.must(f.w.CreateServiceUser(admin(), "w1", "svc", "root"))
	f.must(f.w.AddCredential(admin(), "w1", "svc", "c1", v("c1"), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	f.must(f.w.SuspendServiceUser(admin(), "w1", "svc", 1))
	f.must(f.in.SuspendUser(operator, "svc", 2))
	f.want("blocking twice", f.in.SuspendUser(operator, "svc", 3), ErrConflict)
	f.refusal("an administrator lifting the block", f.w.ReinstateServiceUser(admin(), "w1", "svc", 3), ReasonOperatorBlock)
	f.refusal("the blocked account", second(f.b.AccessByCredential("c1", v("c1"))), ReasonUser)
	f.must(f.in.ReinstateUser(operator, "svc", 3))
	f.must(second(f.b.AccessByCredential("c1", v("c1"))))
	f.must(f.w.SuspendServiceUser(admin(), "w1", "svc", 4))
	f.must(f.in.ReinstateUser(operator, "svc", 5))
	f.must(second(f.b.AccessByCredential("c1", v("c1"))))
	f.member("ann", Member)
	f.must(f.in.SuspendUser(operator, "ann", 1))
	f.want("a human is not suspended twice", f.in.SuspendUser(operator, "ann", 2), ErrConflict)
}

func TestCredentialSlotsAreFreedByExpiry(t *testing.T) {
	f := setup(t)
	admin := func() Verifier { return f.login("root") }
	f.must(f.w.CreateServiceUser(admin(), "w1", "svc", "root"))
	for _, id := range []string{"c1", "c2"} {
		f.must(f.w.AddCredential(admin(), "w1", "svc", id, v(id), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	}
	later := Validity{NotBefore: start, NotAfter: start.Add(3 * time.Hour)}
	*f.now = start.Add(time.Hour - time.Nanosecond)
	f.want("a third credential just before the others end", f.w.AddCredential(admin(), "w1", "svc", "c3", v("c3"), later), ErrLimit)
	*f.now = start.Add(time.Hour)
	f.must(f.w.AddCredential(admin(), "w1", "svc", "c3", v("c3"), later))
	f.must(f.w.RevokeCredential(admin(), "w1", "c3"))
	f.must(f.w.AddCredential(admin(), "w1", "svc", "c4", v("c4"), later))
	f.must(f.w.AddCredential(admin(), "w1", "svc", "c5", v("c5"), later))
	f.want("the limit with a revoked and two current credentials", f.w.AddCredential(admin(), "w1", "svc", "c6", v("c6"), later), ErrLimit)
}

func TestAUserOfTwoWorkspaces(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.seed("w2", "adm2")
	f.member("both", Administrator)
	f.must(f.w.Invite(f.login("adm2"), "w2", "to-w2", Member, v("to-w2"), f.window(), "both"))
	f.must(f.w.Accept("w2", "to-w2", v("to-w2"), f.login("both")))
	f.refusal("an administrator here who is a member there, acting there", f.w.SuspendMembership(f.login("both"), "w2", "adm2", 1), ReasonNotAdministrator)
	f.refusal("the same user listing there", second(f.w.Members(f.login("both"), "w2", "", 1)), ReasonNotAdministrator)
	f.must(f.w.SuspendMembership(f.login("both"), "w1", "ann", 1))
	here, err := f.b.Access(f.login("both"), "w1")
	f.must(err)
	there, err := f.b.Access(f.login("both"), "w2")
	f.must(err)
	if here.Role() != Administrator || there.Role() != Member {
		t.Fatalf("roles %s and %s", here.Role(), there.Role())
	}
	f.must(f.w.RemoveMembership(f.login("adm2"), "w2", "both", 1))
	f.must(f.b.Recheck(here))
	f.refusal("the answer of the workspace the user was removed from", f.b.Recheck(there), ReasonAnswer)
	f.must(f.w.Invite(f.login("adm2"), "w2", "again", Administrator, v("again"), f.window(), "both"))
	f.must(f.w.Accept("w2", "again", v("again"), f.login("both")))
	f.must(f.w.RemoveMembership(f.login("both"), "w2", "adm2", 1))
	f.want("departing as the last administrator of the second workspace, which is otherwise empty", f.in.DepartUser(operator, "both", 1), nil)
	f.must(second(f.b.Access(f.login("root"), "w1")))
	members, _ := f.w.Members(f.login("root"), "w1", "", 2)
	if len(members) != 2 || members[0].User != "ann" || members[1].User != "root" {
		t.Errorf("members here after the departure %+v", members)
	}
	f.r.mu.Lock()
	remaining := f.r.member("w2", "both")
	f.r.mu.Unlock()
	if remaining != nil {
		t.Errorf("the departed user keeps a membership in the second workspace")
	}
}

func TestCountersRefuseAtTheirMaximum(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.r.mu.Lock()
	f.r.users["ann"].epoch = math.MaxUint64
	f.r.memberships["w1"]["ann"].version = math.MaxUint64
	f.r.mu.Unlock()
	f.want("suspension at the highest epoch", f.in.SuspendUser(operator, "ann", math.MaxUint64), ErrConflict)
	f.want("a role change at the highest version", f.w.ChangeRole(f.login("root"), "w1", "ann", Administrator, math.MaxUint64), ErrConflict)
	limits := testLimits
	limits.PageSize = math.MaxInt
	wide := setupWith(t, limits)
	if users, err := wide.in.Users("root", math.MaxInt); err != nil || len(users) != 0 {
		t.Errorf("a page as large as the largest integer: %+v %v", users, err)
	}
}
