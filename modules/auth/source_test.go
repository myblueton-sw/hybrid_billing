package auth

import (
	"errors"
	"testing"
	"time"
)

type fakeMember struct {
	joined uint64
	active bool
}

type fakeSource struct {
	subjects   map[string]Subject
	members    map[[2]string]fakeMember
	generation uint64
	reads      int
	changeAt   int
	calls      int
	tearAt     int
}

func (s *fakeSource) read() {
	s.calls++
	if s.calls == s.tearAt {
		s.generation++
	}
}

func (s *fakeSource) Subject(id string) (Subject, bool) {
	s.read()
	found, known := s.subjects[id]
	return found, known
}

func (s *fakeSource) Member(workspaceID, id string) (uint64, bool) {
	s.read()
	found := s.members[[2]string{workspaceID, id}]
	return found.joined, found.active
}

func (s *fakeSource) Generation() uint64 {
	s.reads++
	if s.reads == s.changeAt {
		s.generation++
	}
	return s.generation
}

func (s *fakeSource) join(workspaceID, id string, joined uint64, active bool) {
	s.members[[2]string{workspaceID, id}] = fakeMember{joined: joined, active: active}
	s.generation++
}

func (s *fakeSource) set(id string, state State, epoch uint64) {
	subject := s.subjects[id]
	subject.State, subject.Epoch = state, epoch
	s.subjects[id] = subject
	s.generation++
}

type sourced struct {
	fixture
	source *fakeSource
}

func setupSource(t testing.TB) sourced {
	t.Helper()
	now := start
	source := &fakeSource{subjects: map[string]Subject{}, members: map[[2]string]fakeMember{}, generation: 1}
	for _, id := range []string{"alice", "bob", "carol", "dave", "erin"} {
		source.subjects[id] = Subject{ID: id, Kind: Human, State: Active, Epoch: 1}
		source.members[[2]string{"w1", id}] = fakeMember{joined: 1, active: true}
	}
	source.subjects["robot"] = Subject{ID: "robot", Kind: Service, State: Active, Epoch: 1}
	source.members[[2]string{"w1", "robot"}] = fakeMember{joined: 1, active: true}
	r, err := NewRegistryWithSubjects("synthetic-installation", Catalog{
		Kinds:  []string{"usage_statement", "allocation_pool", "selling_amount"},
		Fields: map[string]FieldClass{"quantity": General, "service": General, "unit_cost": Cost, "margin": Cost},
	}, func() time.Time { return now }, source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.PutClient(Client{ID: "app", State: Active}, 0); err != nil {
		t.Fatal(err)
	}
	return sourced{fixture{r, &now, t}, source}
}

func TestASourceIsRequiredAndIsTheOnlyWriter(t *testing.T) {
	catalog := Catalog{Kinds: []string{"usage_statement"}}
	if _, err := NewRegistryWithSubjects("i", catalog, time.Now, nil); !errors.Is(err, ErrInvalid) {
		t.Errorf("a nil source: %v", err)
	}
	if _, err := NewRegistryWithSubjects("bad id", catalog, time.Now, &fakeSource{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("an invalid installation with a source: %v", err)
	}
	f := setupSource(t)
	f.grant("g", "alice", terms(Issue, One("c1")))
	first := f.expect("a decision from the source's subject", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	if _, err := f.PutSubject(Subject{ID: "zed", Kind: Human, State: Active, Epoch: 1}, 0); !errors.Is(err, ErrInvalid) {
		t.Errorf("writing a subject into a registry with a source: %v", err)
	}
	if _, err := f.PutSubject(Subject{ID: "alice", Kind: Human, State: Suspended, Epoch: 1}, 0); !errors.Is(err, ErrInvalid) {
		t.Errorf("suspending a subject in a registry with a source: %v", err)
	}
	if got := f.Recheck(first.Stamp, who("alice"), ask(Issue, "c1")); !got.Allowed {
		t.Errorf("a refused subject write changed the registry: %+v", got)
	}
	if len(f.subjects) != 0 {
		t.Errorf("a refused subject write stored %d records", len(f.subjects))
	}
}

func TestEverySubjectReadGoesThroughTheSource(t *testing.T) {
	f := setupSource(t)
	f.grant("g", "alice", terms(Issue, One("c1")))
	f.grant("approve", "bob", terms(Approve, One("c1")))
	f.grant("support", "carol", terms(ApproveSupportSession, One("c1")))
	f.grant("home-bob", "bob", terms(RequestApproval, One("c9")))
	if err := f.delegate("d", "alice", "bob", "", false, terms(Issue, One("c1")), always); err != nil {
		t.Fatal(err)
	}
	f.expect("the principal", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	f.expect("the delegate", who("bob"), ask(Issue, "c1"), true, ReasonAllowed)
	f.source.set("alice", Suspended, 1)
	f.expect("a principal the source suspended", who("alice"), ask(Issue, "c1"), false, ReasonPrincipal)
	f.expect("a delegate of a delegator the source suspended", who("bob"), ask(Issue, "c1"), false, ReasonNoGrant)
	f.source.set("alice", Active, 2)
	f.expect("the old epoch", who("alice"), ask(Issue, "c1"), false, ReasonPrincipal)
	renewed := who("alice")
	renewed.Epoch = 2
	f.expect("the epoch the source reports", renewed, ask(Issue, "c1"), true, ReasonAllowed)
	f.expect("a subject unknown to the source", who("nobody"), ask(Issue, "c1"), false, ReasonPrincipal)

	if err := f.AddGrant(Grant{ID: "x1", SubjectID: "nobody", Terms: terms(Issue, One("c1")), Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a grant for a subject unknown to the source: %v", err)
	}
	if err := f.AddGrant(Grant{ID: "x2", SubjectID: "robot", Terms: terms(Approve, One("c1")), Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Errorf("an approval grant for a subject the source calls a service: %v", err)
	}
	f.source.set("dave", Suspended, 1)
	if err := f.AddGrant(Grant{ID: "x3", SubjectID: "dave", Terms: terms(Issue, One("c1")), Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a grant for a subject the source suspended: %v", err)
	}
	if got := f.CheckApproval(who("bob"), "nobody", ask(Approve, "c1")); got.Reason != ReasonAuthor {
		t.Errorf("an author unknown to the source: %+v", got)
	}
	if got := f.CheckApproval(who("bob"), "erin", ask(Approve, "c1")); !got.Allowed {
		t.Errorf("an author known to the source: %+v", got)
	}
	f.source.set("bob", Suspended, 1)
	if got := f.CheckApproval(who("bob"), "erin", ask(Approve, "c1")); got.Reason != ReasonPrincipal {
		t.Errorf("an approver the source suspended: %+v", got)
	}
	session := SupportSession{ID: "s", ActorID: "erin", WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket", Kind: "usage_statement", Fields: []string{"quantity"}, Validity: always}
	stale := who("carol")
	stale.Epoch = 7
	if err := f.AddSupportSession(session, stale); !errors.Is(err, ErrDenied) {
		t.Errorf("a support approver whose epoch differs from the source's: %v", err)
	}
	f.source.set("erin", Suspended, 1)
	if err := f.AddSupportSession(session, who("carol")); !errors.Is(err, ErrInvalid) {
		t.Errorf("a support actor the source suspended: %v", err)
	}
	f.source.set("erin", Active, 2)
	if err := f.AddSupportSession(session, who("carol")); err != nil {
		t.Fatal(err)
	}
	in := who("erin")
	in.Epoch, in.SessionID = 2, "s"
	f.expect("the support actor", in, ask(View, "c1", "quantity"), true, ReasonAllowed)
	f.source.set("erin", Suspended, 2)
	f.expect("a support actor the source suspended", in, ask(View, "c1", "quantity"), false, ReasonPrincipal)
}

func TestGrantsLiveAndDieWithTheMembership(t *testing.T) {
	f := setupSource(t)
	f.source.join("w2", "alice", 1, true)
	f.source.join("w1", "dave", 0, false)
	other := terms(Issue, One("c1"))
	other.Scope.WorkspaceID = "w3"
	for name, grant := range map[string]Grant{
		"a subject who is not a member":           {ID: "x1", SubjectID: "dave", Terms: terms(Issue, One("c1")), Validity: always},
		"a member of other workspaces only":       {ID: "x2", SubjectID: "alice", Terms: other, Validity: always},
		"a subject without any membership record": {ID: "x3", SubjectID: "erin", Terms: other, Validity: always},
	} {
		if err := f.AddGrant(grant); !errors.Is(err, ErrInvalid) {
			t.Errorf("a grant for %s: %v", name, err)
		}
	}
	f.grant("g", "alice", terms(Issue, One("c1")))
	f.clientGrant("client", terms(Issue, One("c1")))
	f.expect("an active member", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	f.source.join("w1", "alice", 1, false)
	f.expect("a suspended membership", who("alice"), ask(Issue, "c1"), false, ReasonNoGrant)
	f.source.join("w1", "alice", 1, true)
	f.expect("the same membership reinstated", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	f.source.join("w1", "alice", 0, false)
	f.expect("a removed membership", who("alice"), ask(Issue, "c1"), false, ReasonNoGrant)
	f.source.join("w1", "alice", 2, true)
	f.expect("a membership created again", who("alice"), ask(Issue, "c1"), false, ReasonNoGrant)
	f.grant("g2", "alice", terms(Issue, One("c1")))
	f.expect("a new grant for the new membership", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	with := who("alice")
	with.ClientID = "app"
	f.expect("a client grant is not bound to a membership", with, ask(Issue, "c1"), true, ReasonAllowed)
}

func TestEveryUseOfDirectGrantsAppliesTheMembership(t *testing.T) {
	stale := func(t *testing.T) sourced {
		f := setupSource(t)
		f.grant("root", "alice", terms(Issue, One("c1")))
		f.grant("approve", "alice", terms(Approve, One("c1")))
		f.grant("support", "alice", terms(ApproveSupportSession, One("c1")))
		f.grant("home", "alice", terms(RequestApproval, One("c9")))
		f.grant("root-bob", "bob", terms(Issue, One("c1")))
		f.grant("home-carol", "carol", terms(RequestApproval, One("c9")))
		return f
	}
	f := stale(t)
	if err := f.delegate("d", "alice", "carol", "", false, terms(Issue, One("c1")), always); err != nil {
		t.Fatal(err)
	}
	f.expect("a delegate while the root is a member", who("carol"), ask(Issue, "c1"), true, ReasonAllowed)
	f.source.join("w1", "alice", 2, true)
	f.expect("a delegate whose root joined again", who("carol"), ask(Issue, "c1"), false, ReasonNoGrant)
	if err := f.delegate("x1", "alice", "carol", "", false, terms(Issue, One("c1")), always); !errors.Is(err, ErrNotBacked) {
		t.Errorf("a delegation backed only by a grant of an earlier membership: %v", err)
	}
	if err := f.delegate("x2", "bob", "alice", "", false, terms(Issue, One("c1")), always); !errors.Is(err, ErrInvalid) {
		t.Errorf("a delegate whose only grant in the workspace belongs to an earlier membership: %v", err)
	}
	if got := f.CheckApproval(who("alice"), "bob", ask(Approve, "c1")); got.Reason != ReasonNoGrant {
		t.Errorf("an approver whose grant belongs to an earlier membership: %+v", got)
	}
	session := SupportSession{ID: "s", ActorID: "erin", WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket", Kind: "usage_statement", Fields: []string{"quantity"}, Validity: always}
	if err := f.AddSupportSession(session, who("alice")); !errors.Is(err, ErrDenied) {
		t.Errorf("a support approver whose grant belongs to an earlier membership: %v", err)
	}
	f = stale(t)
	if got := f.CheckApproval(who("alice"), "bob", ask(Approve, "c1")); !got.Allowed {
		t.Errorf("the approver before the change: %+v", got)
	}
	if err := f.AddSupportSession(session, who("alice")); err != nil {
		t.Errorf("the support approver before the change: %v", err)
	}
	if err := f.delegate("d", "bob", "carol", "", false, terms(Issue, One("c1")), always); err != nil {
		t.Fatal(err)
	}
	f.source.join("w1", "carol", 1, false)
	f.expect("a delegate whose own membership is suspended", who("carol"), ask(Issue, "c1"), false, ReasonNoGrant)
	f.source.join("w1", "carol", 1, true)
	f.expect("the delegate reinstated", who("carol"), ask(Issue, "c1"), true, ReasonAllowed)
}

func TestDelegationsNeedMembersAndDoNotComeBack(t *testing.T) {
	build := func(t *testing.T) sourced {
		f := setupSource(t)
		f.grant("root", "alice", terms(Issue, One("c1")))
		f.grant("home-bob", "bob", terms(RequestApproval, One("c9")))
		return f
	}
	f := build(t)
	f.source.join("w1", "bob", 1, false)
	if err := f.delegate("x1", "alice", "bob", "", false, terms(Issue, One("c1")), always); !errors.Is(err, ErrInvalid) {
		t.Errorf("a delegate who is not an active member: %v", err)
	}
	if err := f.delegate("x1b", "dave", "bob", "", false, terms(Issue, One("c1")), always); !errors.Is(err, ErrInvalid) {
		t.Errorf("a delegate who is not an active member, from a delegator without backing: %v", err)
	}
	f = build(t)
	f.source.join("w1", "alice", 1, false)
	if err := f.delegate("x2", "alice", "bob", "", false, terms(Issue, One("c1")), always); !errors.Is(err, ErrInvalid) {
		t.Errorf("a delegator who is not an active member: %v", err)
	}
	for name, c := range map[string]struct {
		who    string
		joined uint64
		active bool
		regain func(sourced)
	}{
		"the delegator's membership is suspended":             {"alice", 1, false, nil},
		"the delegator is removed":                            {"alice", 0, false, nil},
		"the delegate's membership is suspended":              {"bob", 1, false, nil},
		"the delegate is removed":                             {"bob", 0, false, nil},
		"the delegator is removed, invited and granted again": {"alice", 2, true, func(f sourced) { f.grant("root-2", "alice", terms(Issue, One("c1"))) }},
		"the delegate is removed, invited and granted again":  {"bob", 2, true, func(f sourced) { f.grant("home-bob-2", "bob", terms(RequestApproval, One("c9"))) }},
	} {
		f := build(t)
		if err := f.delegate("d", "alice", "bob", "", false, terms(Issue, One("c1")), always); err != nil {
			t.Fatal(err)
		}
		f.expect("before", who("bob"), ask(Issue, "c1"), true, ReasonAllowed)
		f.source.join("w1", c.who, c.joined, c.active)
		if c.regain != nil {
			c.regain(f)
		}
		f.expect(name, who("bob"), ask(Issue, "c1"), false, ReasonNoGrant)
	}
	f = setupSource(t)
	f.source.join("w1", "bob", 3, true)
	f.grant("root", "alice", terms(Issue, One("c1")))
	f.grant("home-bob", "bob", terms(RequestApproval, One("c9")))
	if err := f.delegate("d", "alice", "bob", "", false, terms(Issue, One("c1")), always); err != nil {
		t.Fatal(err)
	}
	f.expect("a delegate who joined later than the delegator", who("bob"), ask(Issue, "c1"), true, ReasonAllowed)
	f = build(t)
	if err := f.delegate("d", "alice", "bob", "", false, terms(Issue, One("c1")), always); err != nil {
		t.Fatal(err)
	}
	f.source.join("w1", "alice", 1, false)
	f.source.join("w1", "alice", 1, true)
	f.expect("the delegator's membership suspended and reinstated", who("bob"), ask(Issue, "c1"), true, ReasonAllowed)
}

func TestSupportSessionsNeedAMemberActorAndDoNotComeBack(t *testing.T) {
	f := setupSource(t)
	f.grant("support", "carol", terms(ApproveSupportSession, One("c1")))
	session := SupportSession{ID: "s", ActorID: "erin", WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket", Kind: "usage_statement", Fields: []string{"quantity"}, Validity: always}
	f.source.join("w1", "erin", 1, false)
	if err := f.AddSupportSession(session, who("carol")); !errors.Is(err, ErrInvalid) {
		t.Errorf("an actor who is not an active member: %v", err)
	}
	f.source.join("w1", "erin", 1, true)
	if err := f.AddSupportSession(session, who("carol")); err != nil {
		t.Fatal(err)
	}
	in := who("erin")
	in.SessionID = "s"
	f.expect("the actor", in, ask(View, "c1", "quantity"), true, ReasonAllowed)
	f.source.join("w1", "erin", 1, false)
	f.expect("the actor's membership suspended", in, ask(View, "c1", "quantity"), false, ReasonSupportSession)
	f.source.join("w1", "erin", 1, true)
	f.expect("the same membership reinstated", in, ask(View, "c1", "quantity"), true, ReasonAllowed)
	f.source.join("w1", "erin", 2, true)
	f.expect("the actor removed and invited again", in, ask(View, "c1", "quantity"), false, ReasonSupportSession)
}

func TestStampsAndRegistrationsNoticeAChangeOfTheSource(t *testing.T) {
	f := setupSource(t)
	f.grant("g", "alice", terms(Issue, One("c1")))
	f.grant("approve", "bob", terms(Approve, One("c1")))
	first := f.expect("a decision", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	approved := f.CheckApproval(who("bob"), "alice", ask(Approve, "c1"))
	if got := f.Recheck(first.Stamp, who("alice"), ask(Issue, "c1")); !got.Allowed {
		t.Fatalf("recheck on an unchanged source: %+v", got)
	}
	f.source.generation++
	if got := f.Recheck(first.Stamp, who("alice"), ask(Issue, "c1")); got.Allowed || got.Reason != ReasonStamp {
		t.Errorf("recheck after the source's generation changed with the same values: %+v", got)
	}
	if got := f.RecheckApproval(approved.Stamp, who("bob"), "alice", ask(Approve, "c1")); got.Allowed || got.Reason != ReasonStamp {
		t.Errorf("approval recheck after the source's generation changed: %+v", got)
	}
	during := func(run func()) {
		f.source.changeAt = f.source.reads + 2
		run()
		f.source.changeAt = 0
	}
	during(func() {
		if got := f.Decide(who("alice"), ask(Issue, "c1")); got.Allowed || got.Reason != ReasonStamp || got.Stamp.issued {
			t.Errorf("a decision during which the source changed: %+v", got)
		}
	})
	during(func() {
		if got := f.CheckApproval(who("bob"), "alice", ask(Approve, "c1")); got.Allowed || got.Reason != ReasonStamp {
			t.Errorf("an approval check during which the source changed: %+v", got)
		}
	})
	current := f.expect("a decision on the changed source", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	if got := f.Recheck(current.Stamp, who("alice"), ask(Issue, "c1")); !got.Allowed {
		t.Fatalf("a recheck of the current stamp: %+v", got)
	}
	during(func() {
		if got := f.Recheck(current.Stamp, who("alice"), ask(Issue, "c1")); got.Allowed || got.Reason != ReasonStamp {
			t.Errorf("a recheck during which the source changed: %+v", got)
		}
	})
	f.SetAvailable(false)
	during(func() {
		if got := f.Decide(who("alice"), ask(Issue, "c1")); got.Code != AuthorityUnavailable {
			t.Errorf("an unavailable registry during a change of the source: %+v", got)
		}
	})
	f.SetAvailable(true)
	grant := Grant{ID: "late", SubjectID: "carol", Terms: terms(ApproveSupportSession, One("c1")), Validity: always}
	during(func() {
		if err := f.AddGrant(grant); !errors.Is(err, ErrChanged) {
			t.Errorf("a grant during which the source changed: %v", err)
		}
	})
	if err := f.AddGrant(grant); err != nil {
		t.Errorf("the same grant tried again: %v", err)
	}
	f.grant("home-bob", "bob", terms(RequestApproval, One("c9")))
	delegation := Delegation{ID: "d", DelegatorID: "alice", DelegateID: "bob", Terms: terms(Issue, One("c1")), Validity: always}
	during(func() {
		if err := f.AddDelegation(delegation); !errors.Is(err, ErrChanged) {
			t.Errorf("a delegation during which the source changed: %v", err)
		}
	})
	if err := f.AddDelegation(delegation); err != nil {
		t.Errorf("the same delegation tried again: %v", err)
	}
	session := SupportSession{ID: "s", ActorID: "erin", WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket", Kind: "usage_statement", Fields: []string{"quantity"}, Validity: always}
	during(func() {
		if err := f.AddSupportSession(session, who("carol")); !errors.Is(err, ErrChanged) {
			t.Errorf("a support session during which the source changed: %v", err)
		}
	})
	if err := f.AddSupportSession(session, who("carol")); err != nil {
		t.Errorf("the same support session tried again: %v", err)
	}
	fresh := f.expect("a decision after the changes", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	if fresh.Stamp == first.Stamp {
		t.Errorf("the stamp does not carry the source's generation")
	}
}

func TestAChangeAtAnySourceReadIsNoticed(t *testing.T) {
	session := SupportSession{ID: "s", ActorID: "erin", WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket", Kind: "usage_statement", Fields: []string{"quantity"}, Validity: always}
	build := func(t *testing.T) sourced {
		f := setupSource(t)
		f.grant("root", "alice", terms(Issue, One("c1")))
		f.grant("home-bob", "bob", terms(RequestApproval, One("c9")))
		f.grant("home-carol", "carol", terms(RequestApproval, One("c9")))
		f.grant("approve", "dave", terms(Approve, One("c1")))
		f.grant("support", "dave", terms(ApproveSupportSession, One("c1")))
		if err := f.delegate("d1", "alice", "bob", "", true, terms(Issue, One("c1")), always); err != nil {
			t.Fatal(err)
		}
		return f
	}
	operations := map[string]func(sourced) (bool, bool){
		"a delegated decision": func(f sourced) (bool, bool) {
			got := f.Decide(who("bob"), ask(Issue, "c1"))
			return got.Allowed, got.Reason == ReasonStamp
		},
		"an approval check": func(f sourced) (bool, bool) {
			got := f.CheckApproval(who("dave"), "alice", ask(Approve, "c1"))
			return got.Allowed, got.Reason == ReasonStamp
		},
		"a grant": func(f sourced) (bool, bool) {
			err := f.AddGrant(Grant{ID: "new", SubjectID: "erin", Terms: terms(Issue, One("c1")), Validity: always})
			return err == nil, errors.Is(err, ErrChanged)
		},
		"a delegation with a parent": func(f sourced) (bool, bool) {
			err := f.delegate("d2", "bob", "carol", "d1", false, terms(Issue, One("c1")), always)
			return err == nil, errors.Is(err, ErrChanged)
		},
		"a support session": func(f sourced) (bool, bool) {
			err := f.AddSupportSession(session, who("dave"))
			return err == nil, errors.Is(err, ErrChanged)
		},
	}
	for name, run := range operations {
		f := build(t)
		before := f.source.calls
		if done, _ := run(f); !done {
			t.Fatalf("%s does not succeed on a quiet source", name)
		}
		reads := f.source.calls - before
		if reads < 2 {
			t.Fatalf("%s reads the source only %d times", name, reads)
		}
		for at := 1; at <= reads; at++ {
			f := build(t)
			f.source.tearAt = f.source.calls + at
			if done, noticed := run(f); done || !noticed {
				t.Errorf("%s with a change at source read %d of %d: succeeded %v, noticed %v", name, at, reads, done, noticed)
			}
		}
	}
}

func TestRegistrationOrderOfRefusals(t *testing.T) {
	f := setupSource(t)
	f.grant("root", "alice", terms(Issue, One("c1")))
	f.grant("home-bob", "bob", terms(RequestApproval, One("c9")))
	f.grant("support", "dave", terms(ApproveSupportSession, One("c1")))
	session := SupportSession{ID: "s", ActorID: "erin", WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket", Kind: "usage_statement", Fields: []string{"quantity"}, Validity: always}
	delegation := Delegation{ID: "d", DelegatorID: "alice", DelegateID: "bob", Terms: terms(Issue, One("c1")), Validity: always}
	if err := f.AddDelegation(delegation); err != nil {
		t.Fatal(err)
	}
	if err := f.AddSupportSession(session, who("dave")); err != nil {
		t.Fatal(err)
	}
	repeated := map[string]func() error{
		"a grant": func() error {
			return f.AddGrant(Grant{ID: "root", SubjectID: "alice", Terms: terms(Issue, One("c1")), Validity: always})
		},
		"a delegation":      func() error { return f.AddDelegation(delegation) },
		"a support session": func() error { return f.AddSupportSession(session, who("dave")) },
	}
	for name, run := range repeated {
		if err := run(); !errors.Is(err, ErrConflict) {
			t.Errorf("%s with an existing identifier: %v", name, err)
		}
		f.source.changeAt = f.source.reads + 2
		if err := run(); !errors.Is(err, ErrChanged) {
			t.Errorf("%s with an existing identifier during a change of the source: %v", name, err)
		}
		f.source.changeAt = 0
	}
	for name, actor := range map[string]string{"an actor who is a member": "carol", "an actor who is not a member": "nobody", "a service actor": "robot"} {
		other := session
		other.ID, other.ActorID = "other", actor
		if err := f.AddSupportSession(other, who("alice")); !errors.Is(err, ErrDenied) {
			t.Errorf("an approver without the grant asking about %s: %v", name, err)
		}
	}
	other := session
	other.ID, other.ActorID = "other", "nobody"
	if err := f.AddSupportSession(other, who("dave")); !errors.Is(err, ErrInvalid) {
		t.Errorf("an approver with the grant naming an unknown actor: %v", err)
	}
}

func TestEveryLinkOfAChainNeedsItsMembers(t *testing.T) {
	build := func(t *testing.T) sourced {
		f := setupSource(t)
		f.grant("root", "alice", terms(Issue, One("c1")))
		for _, id := range []string{"bob", "carol"} {
			f.grant("home-"+id, id, terms(RequestApproval, One("c9")))
		}
		if err := f.delegate("d1", "alice", "bob", "", true, terms(Issue, One("c1")), always); err != nil {
			t.Fatal(err)
		}
		if err := f.delegate("d2", "bob", "carol", "d1", false, terms(Issue, One("c1")), always); err != nil {
			t.Fatal(err)
		}
		f.expect("the end of the chain", who("carol"), ask(Issue, "c1"), true, ReasonAllowed)
		return f
	}
	for name, change := range map[string]func(sourced){
		"the root delegator's membership is suspended":   func(f sourced) { f.source.join("w1", "alice", 1, false) },
		"the middle delegator's membership is suspended": func(f sourced) { f.source.join("w1", "bob", 1, false) },
		"the middle delegator joined again": func(f sourced) {
			f.source.join("w1", "bob", 2, true)
			f.grant("home-bob-2", "bob", terms(RequestApproval, One("c9")))
		},
		"the root delegator joined again": func(f sourced) {
			f.source.join("w1", "alice", 2, true)
			f.grant("root-2", "alice", terms(Issue, One("c1")))
		},
	} {
		f := build(t)
		change(f)
		f.expect(name, who("carol"), ask(Issue, "c1"), false, ReasonNoGrant)
	}
}
