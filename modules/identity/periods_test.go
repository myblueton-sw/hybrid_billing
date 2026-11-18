package identity

import (
	"errors"
	"testing"
	"time"
)

func (f fixture) info(id string) MemberInfo {
	f.t.Helper()
	admin := f.login("root")
	for after := ""; ; {
		members, err := f.w.Members(admin, "w1", after, 2)
		f.must(err)
		for _, found := range members {
			if found.User == id {
				return found
			}
		}
		if len(members) < 2 {
			f.t.Fatalf("member %s is not listed", id)
		}
		after = members[len(members)-1].User
	}
}

func (f fixture) bounded(id string, role Role, until time.Time) {
	f.t.Helper()
	f.member(id, role)
	f.must(f.w.SetRolePeriod(f.login("root"), "w1", id, until, f.version(id), *f.now))
}

func (f fixture) at(t time.Time) { *f.now = t }

func TestAccessEndsAtTheRolePeriodEnd(t *testing.T) {
	f := setup(t)
	until := start.Add(20 * time.Minute)
	f.bounded("ann", Member, until)
	session := f.login("ann")
	answer, err := f.b.Access(session, "w1")
	f.must(err)
	if !answer.NotAfter().Equal(until) {
		t.Errorf("not after %v", answer.NotAfter())
	}
	f.at(until.Add(-time.Nanosecond))
	f.must(f.b.Recheck(answer))
	if _, err := f.b.Access(session, "w1"); err != nil {
		t.Errorf("access one nanosecond before the end: %v", err)
	}
	f.at(until)
	f.refusal("recheck at the end", f.b.Recheck(answer), ReasonAnswer)
	f.refusal("access at the end", second(f.b.Access(session, "w1")), ReasonExpired)
	f.at(until.Add(time.Nanosecond))
	f.refusal("access after the end", second(f.b.Access(session, "w1")), ReasonExpired)
	f.at(start.Add(time.Minute))
	f.bounded("bob", Member, start.Add(20*time.Hour))
	long, err := f.b.Access(f.login("bob"), "w1")
	f.must(err)
	if !long.NotAfter().Equal(start.Add(8*time.Hour + time.Minute)) {
		t.Errorf("the session absolute end is the earlier: %v", long.NotAfter())
	}
	f.at(until)
	expired, err := f.in.ExpireRoles(operator, "w1", *f.now)
	if err != nil || len(expired) != 1 || expired[0].User != "ann" || expired[0].Blocked {
		t.Fatalf("expired %+v %v", expired, err)
	}
	f.refusal("access after the sweep", second(f.b.Access(session, "w1")), ReasonExpired)
	f.refusal("a new session after the sweep", second(f.b.Access(f.login("ann"), "w1")), ReasonExpired)
	f.must(f.w.RemoveMembership(f.login("root"), "w1", "bob", f.version("bob")))
	f.refusal("no membership", second(f.b.Access(f.login("bob"), "w1")), ReasonMembership)
}

func TestSetRolePeriod(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.member("adm", Administrator)
	root := f.login("root")
	later, earlier := start.Add(48*time.Hour), start.Add(24*time.Hour)
	f.want("a wrong version", f.w.SetRolePeriod(root, "w1", "ann", later, 99, *f.now), ErrConflict)
	f.want("an end not after now", f.w.SetRolePeriod(root, "w1", "ann", *f.now, f.version("ann"), *f.now), ErrInvalid)
	f.want("an invalid workspace", f.w.SetRolePeriod(root, "", "ann", later, f.version("ann"), *f.now), ErrInvalid)
	f.refusal("a member as actor", f.w.SetRolePeriod(f.login("ann"), "w1", "adm", later, f.version("adm"), *f.now), ReasonNotAdministrator)
	f.refusal("an unknown target", f.w.SetRolePeriod(root, "w1", "nobody", later, 1, *f.now), ReasonUnknown)
	f.refusal("no end on an open period", f.w.SetRolePeriod(root, "w1", "ann", time.Time{}, f.version("ann"), *f.now), ReasonState)
	generation := f.r.Subjects().Generation()
	version := f.version("ann")
	f.must(f.w.SetRolePeriod(root, "w1", "ann", later, version, *f.now))
	if got := f.info("ann"); !got.Until.Equal(later) || got.Version != version+1 || f.r.Subjects().Generation() != generation+1 {
		t.Errorf("set %+v generation %d", got, f.r.Subjects().Generation())
	}
	f.refusal("an equal end", f.w.SetRolePeriod(root, "w1", "ann", later, f.version("ann"), *f.now), ReasonState)
	f.must(f.w.SetRolePeriod(root, "w1", "ann", earlier, f.version("ann"), *f.now))
	f.must(f.w.SetRolePeriod(root, "w1", "ann", later.Add(time.Hour), f.version("ann"), *f.now))
	f.must(f.w.SetRolePeriod(root, "w1", "ann", time.Time{}, f.version("ann"), *f.now))
	if got := f.info("ann"); !got.Until.IsZero() || !got.From.Equal(start) {
		t.Errorf("cleared %+v", got)
	}
	f.must(f.w.SetRolePeriod(root, "w1", "ann", earlier, f.version("ann"), *f.now))
	f.at(earlier.Add(time.Minute))
	root = f.login("root")
	if got := f.info("ann"); got.Effective {
		t.Errorf("an ended period is effective %+v", got)
	}
	f.must(f.w.SetRolePeriod(root, "w1", "ann", later, f.version("ann"), *f.now))
	if got := f.info("ann"); !got.Effective || !got.From.Equal(*f.now) {
		t.Errorf("restore before the sweep %+v", got)
	}
	f.must(f.w.SetRolePeriod(root, "w1", "ann", f.now.Add(time.Hour), f.version("ann"), *f.now))
	f.at(f.now.Add(2 * time.Hour))
	root = f.login("root")
	_, err := f.in.ExpireRoles(operator, "w1", *f.now)
	f.must(err)
	if got := f.info("ann"); got.State != MembershipSuspended || got.SuspendedFor != RolePeriodEnded {
		t.Fatalf("swept %+v", got)
	}
	f.want("a restore with a past end", f.w.SetRolePeriod(root, "w1", "ann", f.now.Add(-time.Minute), f.version("ann"), *f.now), ErrInvalid)
	f.must(f.w.SetRolePeriod(root, "w1", "ann", time.Time{}, f.version("ann"), *f.now))
	if got := f.info("ann"); got.State != MembershipActive || got.SuspendedFor != SuspendedByCommand || !got.Effective || !got.From.Equal(*f.now) {
		t.Errorf("restored %+v", got)
	}
	f.must(f.w.SuspendMembership(root, "w1", "ann", f.version("ann")))
	f.must(f.w.SetRolePeriod(root, "w1", "ann", f.now.Add(time.Hour), f.version("ann"), *f.now))
	if got := f.info("ann"); got.State != MembershipSuspended {
		t.Errorf("a period on a suspended membership does not reinstate it %+v", got)
	}
}

func TestTheLastAdministratorKeepsAPeriodButCannotBeSweptAway(t *testing.T) {
	f := setup(t)
	root := f.login("root")
	f.must(f.w.SetRolePeriod(root, "w1", "root", start.Add(time.Hour), f.version("root"), *f.now))
	f.member("ann", Member)
	f.at(start.Add(2 * time.Hour))
	expired, err := f.in.ExpireRoles(operator, "w1", *f.now)
	if err != nil || len(expired) != 1 || !expired[0].Blocked || expired[0].User != "root" || expired[0].Role != Administrator {
		t.Fatalf("blocked %+v %v", expired, err)
	}
	if got := f.r.memberships["w1"]["root"]; got.state != MembershipActive {
		t.Errorf("a blocked end changed the membership %+v", got)
	}
	generation := f.r.Subjects().Generation()
	if _, err := f.in.ExpireRoles(operator, "w1", *f.now); err != nil || f.r.Subjects().Generation() != generation {
		t.Errorf("a sweep that changes nothing bumps the generation: %v", err)
	}
}

func TestExpireRoles(t *testing.T) {
	f := setup(t)
	f.bounded("ann", Member, start.Add(time.Hour))
	f.bounded("bob", Administrator, start.Add(time.Hour))
	f.bounded("cat", Member, start.Add(3*time.Hour))
	f.member("dan", Member)
	if _, err := f.in.ExpireRoles(Operator{}, "w1", *f.now); !errors.Is(err, ErrInvalid) {
		t.Errorf("an invalid operator %v", err)
	}
	if _, err := f.in.ExpireRoles(operator, "", *f.now); !errors.Is(err, ErrInvalid) {
		t.Errorf("an invalid workspace %v", err)
	}
	f.at(start.Add(2 * time.Hour))
	generation := f.r.Subjects().Generation()
	versions := map[string]uint64{"ann": f.version("ann"), "bob": f.version("bob")}
	expired, err := f.in.ExpireRoles(operator, "w1", *f.now)
	f.must(err)
	if len(expired) != 2 || expired[0].User != "ann" || expired[1].User != "bob" || expired[0].Version != versions["ann"] || expired[1].Role != Administrator ||
		!expired[1].Until.Equal(start.Add(time.Hour)) {
		t.Fatalf("expired %+v", expired)
	}
	if f.r.Subjects().Generation() != generation+1 {
		t.Errorf("generation %d", f.r.Subjects().Generation())
	}
	for _, id := range []string{"ann", "bob"} {
		if got := f.info(id); got.State != MembershipSuspended || got.SuspendedFor != RolePeriodEnded || got.Version != versions[id]+1 {
			t.Errorf("%s %+v", id, got)
		}
	}
	for _, id := range []string{"cat", "dan"} {
		if got := f.info(id); got.State != MembershipActive {
			t.Errorf("%s changed %+v", id, got)
		}
	}
	root := f.login("root")
	f.refusal("reinstating an ended period", f.w.ReinstateMembership(root, "w1", "ann", f.version("ann")), ReasonRolePeriod)
	f.refusal("changing the role of an ended period", f.w.ChangeRoleUntil(root, "w1", "bob", Member, time.Time{}, f.version("bob"), *f.now), ReasonRolePeriod)
	f.refusal("the existing change of role on an ended period", f.w.ChangeRole(root, "w1", "bob", Member, f.version("bob")), ReasonRolePeriod)
	f.must(f.w.RemoveMembership(root, "w1", "ann", f.version("ann")))
	f.at(start.Add(4 * time.Hour))
	root = f.login("root")
	f.must(f.w.SuspendMembership(root, "w1", "cat", f.version("cat")))
	f.refusal("reinstating a suspension whose end passed", f.w.ReinstateMembership(root, "w1", "cat", f.version("cat")), ReasonRolePeriod)
	f.refusal("changing the role of a suspension whose end passed", f.w.ChangeRoleUntil(root, "w1", "cat", Administrator, time.Time{}, f.version("cat"), *f.now), ReasonRolePeriod)
	if expired, _ := f.in.ExpireRoles(operator, "w1", *f.now); len(expired) != 0 {
		t.Errorf("a suspended membership is not swept again %+v", expired)
	}
}

func TestChangeRoleUntil(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	root := f.login("root")
	f.want("an invalid role", f.w.ChangeRoleUntil(root, "w1", "ann", ServiceRole, time.Time{}, f.version("ann"), *f.now), ErrInvalid)
	f.want("a past end", f.w.ChangeRoleUntil(root, "w1", "ann", Administrator, *f.now, f.version("ann"), *f.now), ErrInvalid)
	f.want("an invalid role in the existing change", f.w.ChangeRole(root, "w1", "ann", ServiceRole, f.version("ann")), ErrInvalid)
	f.refusal("the same role and end", f.w.ChangeRoleUntil(root, "w1", "ann", Member, time.Time{}, f.version("ann"), *f.now), ReasonState)
	f.at(start.Add(time.Minute))
	f.must(f.w.ChangeRoleUntil(root, "w1", "ann", Administrator, start.Add(time.Hour), f.version("ann"), *f.now))
	if got := f.info("ann"); got.Role != Administrator || !got.Until.Equal(start.Add(time.Hour)) || !got.From.Equal(*f.now) {
		t.Errorf("changed %+v", got)
	}
	f.must(f.w.ChangeRole(root, "w1", "ann", Member, f.version("ann")))
	if got := f.info("ann"); got.Role != Member || !got.Until.Equal(start.Add(time.Hour)) {
		t.Errorf("the existing change keeps the end %+v", got)
	}
}

func TestTheTimeAwarePredicateInTheWindowBeforeTheSweep(t *testing.T) {
	f := setup(t)
	f.member("adm", Administrator)
	f.user("ann")
	f.user("bob")
	inviter := f.login("adm")
	f.must(f.w.Invite(inviter, "w1", "from-adm", Member, v("from-adm"), f.window(), "ann"))
	f.must(f.w.SetRolePeriod(f.login("root"), "w1", "adm", start.Add(time.Hour), f.version("adm"), *f.now))
	f.must(f.w.Invite(f.login("adm"), "w1", "from-adm-2", Member, v("from-adm-2"), f.window(), "bob"))
	f.at(start.Add(time.Hour))
	f.refusal("acceptance from an inviter whose period ended", f.w.Accept("w1", "from-adm-2", v("from-adm-2"), f.login("bob")), ReasonInviter)
	f.refusal("acceptance from an invitation made under an older period", f.w.Accept("w1", "from-adm", v("from-adm"), f.login("ann")), ReasonInviter)
	if f.r.activeAdministrator("w1", "adm", *f.now) || f.r.administrators("w1", *f.now) != 1 {
		t.Error("the ended administrator is counted")
	}
	if _, member := f.r.Subjects().Member("w1", "adm"); member {
		t.Error("the subject view reports a membership")
	}
	if got := f.info("adm"); got.Effective || got.State != MembershipActive {
		t.Errorf("member info %+v", got)
	}
	ended := f.login("adm")
	f.refusal("a transition by the ended administrator", f.w.SuspendMembership(ended, "w1", "root", f.version("root")), ReasonNotAdministrator)
	f.refusal("the members list for the ended administrator", second(f.w.Members(ended, "w1", "", 2)), ReasonNotAdministrator)
	f.refusal("leaving by the ended administrator", f.w.RemoveMembership(ended, "w1", "adm", f.version("adm")), ReasonNotAdministrator)
	f.refusal("a service user owned by the ended administrator", f.w.CreateServiceUser(f.login("root"), "w1", "svc", "adm"), ReasonOwner)
	if got := f.r.withoutAdministrator(*f.now); len(got) != 0 {
		t.Errorf("without administrator %v", got)
	}
	f.must(f.w.SetRolePeriod(f.login("root"), "w1", "root", f.now.Add(time.Hour), f.version("root"), *f.now))
	f.at(f.now.Add(time.Hour))
	if got := f.r.withoutAdministrator(*f.now); len(got) != 1 || got[0] != "w1" {
		t.Errorf("without administrator after every end %v", got)
	}
}

func TestInvitationsCarryTheRoleEnd(t *testing.T) {
	f := setup(t)
	f.user("ann")
	root := f.login("root")
	f.want("an end before the invitation starts", f.w.InviteUntil(root, "w1", "i1", Member, v("i1"), f.window(), "ann", start), ErrInvalid)
	f.want("an end already over", f.w.InviteUntil(root, "w1", "i1", Member, v("i1"), Validity{NotBefore: start.Add(-time.Hour), NotAfter: start.Add(time.Hour)}, "ann", start.Add(-time.Minute)), ErrInvalid)
	f.must(f.w.InviteUntil(root, "w1", "i1", Member, v("i1"), f.window(), "ann", start.Add(time.Hour)))
	if got := f.r.invitations["i1"].validity.NotAfter; !got.Equal(start.Add(time.Hour)) {
		t.Errorf("the invitation is not capped: %v", got)
	}
	f.must(f.w.InviteUntil(root, "w1", "i2", Member, v("i2"), f.window(), "ann", start.Add(72*time.Hour)))
	if got := f.r.invitations["i2"].validity.NotAfter; !got.Equal(start.Add(48 * time.Hour)) {
		t.Errorf("a later end changed the invitation: %v", got)
	}
	f.at(start.Add(time.Hour))
	f.refusal("acceptance at the role end", f.w.Accept("w1", "i1", v("i1"), f.login("ann")), ReasonExpired)
	f.must(f.w.Accept("w1", "i2", v("i2"), f.login("ann")))
	if got := f.info("ann"); !got.Until.Equal(start.Add(72*time.Hour)) || !got.From.Equal(*f.now) {
		t.Errorf("accepted %+v", got)
	}
}

func TestBatchSuspensions(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.member("bob", Administrator)
	f.member("cat", Member)
	root := f.login("root")
	targets := func(ids ...string) []Target {
		out := []Target{}
		for _, id := range ids {
			out = append(out, Target{User: id, Version: f.version(id)})
		}
		return out
	}
	f.want("no targets", f.w.SuspendMemberships(root, "w1", nil, BulkRevocation), ErrInvalid)
	f.want("a manual reason", f.w.SuspendMemberships(root, "w1", targets("ann"), SuspendedByCommand), ErrInvalid)
	f.want("an invalid workspace", f.w.SuspendMemberships(root, "", targets("ann"), BulkRevocation), ErrInvalid)
	f.want("a repeated target", f.w.SuspendMemberships(root, "w1", append(targets("ann"), targets("ann")...), BulkRevocation), ErrInvalid)
	f.refusal("a member as actor", f.w.SuspendMemberships(f.login("ann"), "w1", targets("cat"), BulkRevocation), ReasonNotAdministrator)
	f.refusal("an unknown target", f.w.SuspendMemberships(root, "w1", []Target{{User: "nobody", Version: 1}}, BulkRevocation), ReasonUnknown)
	stale := targets("ann", "cat")
	stale[1].Version++
	generation := f.r.Subjects().Generation()
	f.want("one stale version", f.w.SuspendMemberships(root, "w1", stale, BulkRevocation), ErrConflict)
	if f.info("ann").State != MembershipActive || f.r.Subjects().Generation() != generation {
		t.Error("a refused batch changed something")
	}
	f.want("every administrator", f.w.SuspendMemberships(f.login("bob"), "w1", targets("root", "bob"), BulkRevocation), ErrLastAdministrator)
	f.must(f.w.SuspendMemberships(root, "w1", targets("ann", "cat", "bob"), BulkRevocation))
	if f.r.Subjects().Generation() != generation+1 {
		t.Errorf("generation %d", f.r.Subjects().Generation())
	}
	for _, id := range []string{"ann", "cat", "bob"} {
		if got := f.info(id); got.State != MembershipSuspended || got.SuspendedFor != BulkRevocation {
			t.Errorf("%s %+v", id, got)
		}
	}
	f.refusal("an already suspended target", f.w.SuspendMemberships(root, "w1", targets("ann"), BulkRevocation), ReasonState)
	f.must(f.w.ReinstateMembership(root, "w1", "ann", f.version("ann")))
	if got := f.info("ann"); got.SuspendedFor != SuspendedByCommand || got.State != MembershipActive {
		t.Errorf("reinstated %+v", got)
	}
}

func TestBatchUserSuspensions(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.member("bob", Administrator)
	f.user("cat")
	session := f.login("ann")
	users := func(ids ...string) []Target {
		out := []Target{}
		for _, id := range ids {
			out = append(out, Target{User: id, Version: f.epoch(id)})
		}
		return out
	}
	f.want("an invalid operator", f.in.SuspendUsers(Operator{}, users("ann")), ErrInvalid)
	f.want("no targets", f.in.SuspendUsers(operator, nil), ErrInvalid)
	f.want("a repeated target", f.in.SuspendUsers(operator, append(users("ann"), users("ann")...)), ErrInvalid)
	f.refusal("an unknown target", f.in.SuspendUsers(operator, []Target{{User: "nobody", Version: 1}}), ReasonUnknown)
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "svc", "ann"))
	f.refusal("a service user", f.in.SuspendUsers(operator, []Target{{User: "svc", Version: 1}}), ReasonUnknown)
	f.want("a stale epoch", f.in.SuspendUsers(operator, []Target{{User: "ann", Version: 9}}), ErrConflict)
	f.want("every administrator", f.in.SuspendUsers(operator, users("root", "bob")), ErrLastAdministrator)
	generation := f.r.Subjects().Generation()
	f.must(f.in.SuspendUsers(operator, users("ann", "cat", "bob")))
	if f.r.Subjects().Generation() != generation+1 {
		t.Errorf("generation %d", f.r.Subjects().Generation())
	}
	f.refusal("a session of a suspended user", second(f.b.Access(session, "w1")), ReasonSession)
	f.refusal("an already suspended user", f.in.SuspendUsers(operator, users("ann")), ReasonState)
	if _, state, epoch, _ := f.r.Subjects().Subject("ann"); state != Suspended || epoch != 2 {
		t.Errorf("ann %s %d", state, epoch)
	}
}

func TestApplyRecorded(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.member("bob", Member)
	session := f.login("ann")
	f.want("an invalid operator", f.in.ApplyRecorded(Operator{}, "w1", Recorded{User: "ann"}), ErrInvalid)
	f.want("an invalid user", f.in.ApplyRecorded(operator, "w1", Recorded{User: ""}), ErrInvalid)
	f.want("an invalid workspace", f.in.ApplyRecorded(operator, "", Recorded{User: "ann"}), ErrInvalid)
	f.refusal("an unknown user", f.in.ApplyRecorded(operator, "w1", Recorded{User: "nobody"}), ReasonUnknown)
	f.want("an unknown user state", f.in.ApplyRecorded(operator, "w1", Recorded{User: "ann", UserState: Departed}), ErrInvalid)
	f.want("an unknown membership state", f.in.ApplyRecorded(operator, "w1", Recorded{User: "ann", Membership: MembershipRemoved}), ErrInvalid)
	f.user("cat")
	f.refusal("a membership that is not there", f.in.ApplyRecorded(operator, "w1", Recorded{User: "cat", Membership: MembershipSuspended}), ReasonUnknown)
	f.refusal("a period without a membership", f.in.ApplyRecorded(operator, "w1", Recorded{User: "cat", SetUntil: true}), ReasonUnknown)
	f.must(f.in.ApplyRecorded(operator, "w1", Recorded{User: "ann", UserState: Suspended}))
	f.refusal("the session of a suspended user", second(f.b.Access(session, "w1")), ReasonSession)
	f.must(f.in.ApplyRecorded(operator, "w1", Recorded{User: "ann", UserState: Suspended}))
	if f.epoch("ann") != 2 {
		t.Errorf("a repeated suspension bumps the epoch again: %d", f.epoch("ann"))
	}
	f.must(f.in.ApplyRecorded(operator, "w1", Recorded{User: "ann", UserState: Active}))
	version := f.version("bob")
	until := start.Add(time.Hour)
	f.must(f.in.ApplyRecorded(operator, "w1", Recorded{User: "bob", Membership: MembershipSuspended, Reason: RolePeriodEnded, SetUntil: true, Until: until}))
	if got := f.info("bob"); got.State != MembershipSuspended || got.SuspendedFor != RolePeriodEnded || !got.Until.Equal(until) || got.Version != version+1 {
		t.Errorf("bob %+v", got)
	}
	f.at(start.Add(time.Minute))
	f.must(f.in.ApplyRecorded(operator, "w1", Recorded{User: "bob", Membership: MembershipActive, SetUntil: true}))
	if got := f.info("bob"); got.State != MembershipActive || !got.Until.IsZero() || !got.From.Equal(*f.now) || got.Version != version+2 {
		t.Errorf("bob restored %+v", got)
	}
	f.must(f.in.ApplyRecorded(operator, "w1", Recorded{User: "bob", Membership: MembershipActive, SetUntil: true}))
	if got := f.info("bob"); got.Version != version+2 {
		t.Errorf("an unchanged record bumps the version %+v", got)
	}
	f.must(f.in.DepartUser(operator, "cat", f.epoch("cat")))
	f.want("a departed user", f.in.ApplyRecorded(operator, "w1", Recorded{User: "cat", UserState: Suspended}), ErrConflict)
}

func TestNewOperationsOnAnUnconstructedRegistry(t *testing.T) {
	var r Registry
	in, w := r.Installation(), r.Workspace()
	calls := map[string]error{
		"SetRolePeriod":            w.SetRolePeriod(v("s"), "w1", "u", time.Time{}, 1, start),
		"ChangeRoleUntil":          w.ChangeRoleUntil(v("s"), "w1", "u", Member, time.Time{}, 1, start),
		"InviteUntil":              w.InviteUntil(v("s"), "w1", "i", Member, v("i"), Validity{}, "u", time.Time{}),
		"ReinstateMembershipAt":    w.ReinstateMembershipAt(v("s"), "w1", "u", 1, start),
		"CheckSetRolePeriod":       w.CheckSetRolePeriod(v("s"), "w1", "u", time.Time{}, 1, start),
		"CheckChangeRoleUntil":     w.CheckChangeRoleUntil(v("s"), "w1", "u", Member, time.Time{}, 1, start),
		"CheckReinstateMembership": w.CheckReinstateMembership(v("s"), "w1", "u", 1, start),
		"CheckReinstateUser":       in.CheckReinstateUser(operator, "u", 1),
		"SuspendMemberships":       w.SuspendMemberships(v("s"), "w1", nil, BulkRevocation),
		"SuspendUsers":             in.SuspendUsers(operator, nil),
		"ExpireRoles":              second(in.ExpireRoles(operator, "w1", start)),
		"ApplyRecorded":            in.ApplyRecorded(operator, "w1", Recorded{}),
	}
	for name, err := range calls {
		if !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s on an unconstructed registry: %v", name, err)
		}
	}
}

func TestRecordedPeriodsAndCounterLimits(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.must(f.in.ApplyRecorded(operator, "w1", Recorded{User: "ann", SetUntil: true, Until: start.Add(time.Hour)}))
	if got := f.info("ann"); !got.Until.Equal(start.Add(time.Hour)) || got.State != MembershipActive {
		t.Errorf("a recorded end alone %+v", got)
	}
	f.r.memberships["w1"]["ann"].version = ^uint64(0)
	f.want("a version at its maximum", f.in.ApplyRecorded(operator, "w1", Recorded{User: "ann", Membership: MembershipSuspended}), ErrConflict)
	f.r.putMembership("w2", "ann", &membership{role: Member, state: MembershipActive, version: 1, joined: 1})
	f.r.memberships["w1"]["ann"].version = 3
	f.must(f.in.SuspendUsers(operator, []Target{{User: "ann", Version: f.epoch("ann")}}))
}

func TestOperatorMembersRead(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.bounded("bob", Administrator, start.Add(time.Hour))
	if _, err := f.in.Members(Operator{}, "w1"); !errors.Is(err, ErrInvalid) {
		t.Error("an invalid operator")
	}
	if _, err := f.in.Members(operator, ""); !errors.Is(err, ErrInvalid) {
		t.Error("an invalid workspace")
	}
	f.must(f.w.RemoveMembership(f.login("root"), "w1", "ann", f.version("ann")))
	list, err := f.in.Members(operator, "w1")
	if err != nil || len(list) != 2 || list[0].User != "bob" || !list[0].Until.Equal(start.Add(time.Hour)) || !list[0].Effective || list[1].User != "root" {
		t.Errorf("members %+v %v", list, err)
	}
	var r Registry
	if _, err := r.Installation().Members(operator, "w1"); !errors.Is(err, ErrUnavailable) {
		t.Error("an unconstructed registry")
	}
}

func TestAMemberCannotSetTheirOwnPeriod(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.refusal("a member's own period", f.w.SetRolePeriod(f.login("ann"), "w1", "ann", start.Add(time.Hour), f.version("ann"), *f.now), ReasonNotAdministrator)
	f.refusal("a member's own role", f.w.ChangeRoleUntil(f.login("ann"), "w1", "ann", Administrator, time.Time{}, f.version("ann"), *f.now), ReasonNotAdministrator)
	f.refusal("a member's own reinstatement", f.w.ReinstateMembershipAt(f.login("ann"), "w1", "ann", f.version("ann"), *f.now), ReasonNotAdministrator)
}

func TestApplyRecordedKeepsTheStartAndBumpsTheGeneration(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	from := f.info("ann").From
	f.at(start.Add(time.Hour))
	generation := f.r.Subjects().Generation()
	f.must(f.in.ApplyRecorded(operator, "w1", Recorded{User: "ann", Membership: MembershipActive}))
	if got := f.info("ann"); !got.From.Equal(from) || f.r.Subjects().Generation() <= generation {
		t.Errorf("an active membership restated %+v generation %d", got, f.r.Subjects().Generation())
	}
}

func TestBatchAndTimedReinstatementsChangeState(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	root := f.login("root")
	version := f.version("ann")
	f.must(f.w.SuspendMemberships(root, "w1", []Target{{User: "ann", Version: version}}, BulkRevocation))
	if got := f.info("ann"); got.Version != version+1 {
		t.Errorf("a batch suspension keeps the version %+v", got)
	}
	f.must(f.w.ReinstateMembershipAt(root, "w1", "ann", version+1, *f.now))
	if got := f.info("ann"); got.State != MembershipActive || got.Version != version+2 {
		t.Errorf("a timed reinstatement %+v", got)
	}
}
