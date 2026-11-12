package auth

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func (f fixture) homes(ids ...string) {
	f.t.Helper()
	for _, id := range ids {
		f.grant("home-"+id, id, terms(RequestApproval, One("c9")))
	}
}

func (f fixture) refuse(name string, want error, err error) {
	f.t.Helper()
	if !errors.Is(err, want) {
		f.t.Errorf("%s: error %v; want %v", name, err, want)
	}
}

func TestDelegationKindRules(t *testing.T) {
	f := setup(t)
	view := terms(View, One("c1"), "quantity")
	pool := terms(View, One("c1"), "quantity")
	pool.Kind = "allocation_pool"
	selling := terms(View, One("c1"), "quantity")
	selling.Kind = "selling_amount"
	f.grant("root", "alice", view)
	f.grant("root-pool", "alice", pool)
	f.homes("bob", "carol")
	if err := f.delegate("d", "alice", "bob", "", true, view, always); err != nil {
		t.Fatal(err)
	}
	f.expect("the delegated kind", who("bob"), ask(View, "c1", "quantity"), true, ReasonAllowed)
	other := ask(View, "c1", "quantity")
	other.Kind = "allocation_pool"
	f.expect("another kind that the root holds directly", who("bob"), other, false, ReasonNoGrant)
	f.refuse("a parentless delegation of a kind the delegator does not hold", ErrNotBacked, f.delegate("x1", "alice", "carol", "", false, selling, always))
	f.refuse("a child of another kind than its parent", ErrNotBacked, f.delegate("x2", "bob", "carol", "d", false, pool, always))
}

func TestDelegationPeriodRules(t *testing.T) {
	f := setup(t)
	f.grant("root", "alice", terms(View, One("c1"), "quantity"))
	f.homes("erin", "dave")
	bounded := terms(View, One("c1"), "quantity")
	bounded.Period = Months(month(t, 2026, 1), month(t, 2026, 4))
	if err := f.delegate("d", "alice", "erin", "", true, bounded, always); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		at      Month
		has     bool
		allowed bool
	}{
		"a month inside the delegated period":      {month(t, 2026, 2), true, true},
		"a month the root holds but did not share": {month(t, 2026, 6), true, false},
		"a resource without month":                 {0, false, false},
	} {
		request := ask(View, "c1", "quantity")
		request.Resource.Month, request.Resource.HasMonth = c.at, c.has
		reason := ReasonAllowed
		if !c.allowed {
			reason = ReasonNoGrant
		}
		f.expect(name, who("erin"), request, c.allowed, reason)
	}
	child := func(from, to Month) Terms {
		terms := terms(View, One("c1"), "quantity")
		terms.Period = Months(from, to)
		return terms
	}
	f.refuse("a child period ending after the parent's", ErrNotBacked, f.delegate("x1", "erin", "dave", "d", false, child(month(t, 2026, 2), month(t, 2026, 5)), always))
	f.refuse("a child period starting before the parent's", ErrNotBacked, f.delegate("x2", "erin", "dave", "d", false, child(month(t, 2025, 12), month(t, 2026, 3)), always))
	if err := f.delegate("ok", "erin", "dave", "d", false, child(month(t, 2026, 2), month(t, 2026, 3)), always); err != nil {
		t.Errorf("a child period inside the parent's: %v", err)
	}
}

func TestDelegationScopeContainment(t *testing.T) {
	f := setup(t)
	narrow := terms(Issue, One("c1"))
	narrow.Scope.Contract, narrow.Scope.LegalEntity = One("k1"), One("le1")
	f.grant("root", "alice", terms(Issue, One("c1")))
	f.grant("contract-only", "dave", narrow)
	f.homes("bob", "carol", "erin")
	if err := f.delegate("d", "alice", "bob", "", true, narrow, always); err != nil {
		t.Fatal(err)
	}
	widened := map[string]func(*Terms){
		"every contract":       func(t *Terms) { t.Scope.Contract = All() },
		"every legal entity":   func(t *Terms) { t.Scope.LegalEntity = All() },
		"another contract":     func(t *Terms) { t.Scope.Contract = One("k2") },
		"another legal entity": func(t *Terms) { t.Scope.LegalEntity = One("le2") },
	}
	for name, change := range widened {
		changed := narrow
		change(&changed)
		f.refuse("a child with "+name, ErrNotBacked, f.delegate("x-child", "bob", "carol", "d", false, changed, always))
		f.refuse("a parentless delegation with "+name, ErrNotBacked, f.delegate("x-root", "dave", "erin", "", false, changed, always))
	}
	if err := f.delegate("child", "bob", "carol", "d", false, narrow, always); err != nil {
		t.Fatal(err)
	}
	if err := f.delegate("exact", "dave", "erin", "", false, narrow, always); err != nil {
		t.Fatal(err)
	}
	inside := ask(Issue, "c1")
	inside.Resource.ContractID, inside.Resource.LegalEntityID = "k1", "le1"
	outside := inside
	outside.Resource.ContractID = "k2"
	f.expect("the delegated contract", who("carol"), inside, true, ReasonAllowed)
	f.expect("a contract the root holds but did not share", who("carol"), outside, false, ReasonNoGrant)
}

func TestDelegationRegistrationNeedsALiveParentAndDelegator(t *testing.T) {
	f := setup(t)
	view := terms(View, One("c1"), "quantity")
	short := Validity{NotBefore: start.Add(-time.Hour), NotAfter: start.Add(time.Hour)}
	f.grant("root", "alice", view)
	f.grant("robot-view", "robot", view)
	f.homes("bob", "carol", "dave")
	for id, to := range map[string]string{"d-short": "bob", "d-revoked": "dave"} {
		if err := f.delegate(id, "alice", to, "", true, view, short); err != nil {
			t.Fatal(err)
		}
	}
	early := Validity{NotBefore: short.NotBefore.Add(-time.Minute), NotAfter: short.NotAfter}
	f.refuse("a child valid before its parent", ErrNotBacked, f.delegate("x1", "bob", "carol", "d-short", false, view, early))
	f.refuse("a service delegator", ErrInvalid, f.delegate("x2", "robot", "carol", "", false, view, always))
	if err := f.RevokeDelegation("d-revoked", 1); err != nil {
		t.Fatal(err)
	}
	f.refuse("a child of a revoked parent", ErrNotBacked, f.delegate("x3", "dave", "carol", "d-revoked", false, view, short))
	*f.now = short.NotAfter
	late := Validity{NotBefore: short.NotBefore, NotAfter: short.NotAfter}
	f.refuse("a child of an expired parent", ErrNotBacked, f.delegate("x4", "bob", "carol", "d-short", false, view, late))
	*f.now = start
	if _, err := f.PutSubject(Subject{ID: "alice", Kind: Human, State: Suspended, Epoch: 1}, 1); err != nil {
		t.Fatal(err)
	}
	f.refuse("a suspended delegator", ErrInvalid, f.delegate("x5", "alice", "carol", "", false, view, always))
}

func TestDecisionListsTheDelegationsUsedInOrder(t *testing.T) {
	f := setup(t)
	f.grant("root", "alice", terms(Issue, One("c1")))
	f.grant("root-view", "alice", terms(View, One("c1"), "quantity"))
	f.homes("bob", "carol", "dave")
	steps := []struct{ id, from, to, parent string }{{"z1", "alice", "bob", ""}, {"a2", "bob", "carol", "z1"}, {"m3", "carol", "dave", "a2"}}
	for _, step := range steps {
		if err := f.delegate(step.id, step.from, step.to, step.parent, true, terms(Issue, One("c1")), always); err != nil {
			t.Fatal(err)
		}
	}
	got := f.expect("the third delegate", who("dave"), ask(Issue, "c1"), true, ReasonAllowed)
	if want := []string{"a2", "m3", "z1"}; !reflect.DeepEqual(got.Delegations, want) {
		t.Errorf("delegations %v; want %v", got.Delegations, want)
	}
	for name, request := range map[string]Request{"an action": ask(Issue, "c1"), "a field action": ask(View, "c1", "quantity")} {
		if got := f.expect(name+" held directly", who("alice"), request, true, ReasonAllowed); got.Delegations != nil {
			t.Errorf("%s held directly lists delegations %v", name, got.Delegations)
		}
	}
}

func TestStampsOfDelegatedAndSessionPrincipals(t *testing.T) {
	short := Validity{NotBefore: start.Add(-time.Hour), NotAfter: start.Add(time.Hour)}
	view := terms(View, One("c1"), "quantity")
	in := who("erin")
	in.SessionID = "s"
	build := func(t *testing.T) (fixture, Stamp, Stamp) {
		f := setup(t)
		f.grant("root", "alice", view)
		f.homes("bob", "carol")
		f.grant("support-approval", "dave", terms(ApproveSupportSession, One("c1")))
		if err := f.delegate("d1", "alice", "bob", "", true, view, short); err != nil {
			t.Fatal(err)
		}
		if err := f.delegate("d2", "bob", "carol", "d1", false, view, short); err != nil {
			t.Fatal(err)
		}
		if err := f.session("s", "erin", "dave", "quantity"); err != nil {
			t.Fatal(err)
		}
		delegated := f.expect("the delegate", who("carol"), ask(View, "c1", "quantity"), true, ReasonAllowed)
		session := f.expect("the session", in, ask(View, "c1", "quantity"), true, ReasonAllowed)
		for name, got := range map[string]Decision{
			"the delegate": f.Recheck(delegated.Stamp, who("carol"), ask(View, "c1", "quantity")),
			"the session":  f.Recheck(session.Stamp, in, ask(View, "c1", "quantity")),
		} {
			if !got.Allowed {
				t.Fatalf("recheck of %s on unchanged state: %+v", name, got)
			}
		}
		if got := f.Recheck(delegated.Stamp, in, ask(View, "c1", "quantity")); got.Allowed || got.Reason != ReasonStamp {
			t.Fatalf("the delegate's stamp presented by the session principal: %+v", got)
		}
		return f, delegated.Stamp, session.Stamp
	}
	for name, c := range map[string]struct {
		change func(fixture) error
		reason Reason
	}{
		"the delegation is revoked":       {func(f fixture) error { return f.RevokeDelegation("d2", 1) }, ReasonNoGrant},
		"the parent is revoked":           {func(f fixture) error { return f.RevokeDelegation("d1", 1) }, ReasonNoGrant},
		"the delegations expire by clock": {func(f fixture) error { *f.now = short.NotAfter; return nil }, ReasonNoGrant},
		"an unrelated session is ended":   {func(f fixture) error { return f.EndSupportSession("s", 1) }, ReasonStamp},
		"an unrelated grant is added": {func(f fixture) error {
			return f.AddGrant(Grant{ID: "more", SubjectID: "dave", Terms: view, Validity: always})
		}, ReasonStamp},
	} {
		f, delegated, _ := build(t)
		if err := c.change(f); err != nil {
			t.Fatal(err)
		}
		if got := f.Recheck(delegated, who("carol"), ask(View, "c1", "quantity")); got.Allowed || got.Reason != c.reason {
			t.Errorf("%s: recheck %+v; want denied with %s", name, got, c.reason)
		}
	}
	f, _, session := build(t)
	if err := f.EndSupportSession("s", 1); err != nil {
		t.Fatal(err)
	}
	if got := f.Recheck(session, in, ask(View, "c1", "quantity")); got.Allowed || got.Reason != ReasonSupportSession {
		t.Errorf("recheck after the session ended: %+v", got)
	}
	f, delegated, _ := build(t)
	if _, err := f.PutSubject(Subject{ID: "carol", Kind: Human, State: Active, Epoch: 2}, 1); err != nil {
		t.Fatal(err)
	}
	if got := f.Recheck(delegated, who("carol"), ask(View, "c1", "quantity")); got.Allowed || got.Reason != ReasonPrincipal {
		t.Errorf("recheck with the old epoch: %+v", got)
	}
	renewed := who("carol")
	renewed.Epoch = 2
	if got := f.Decide(renewed, ask(View, "c1", "quantity")); !got.Allowed {
		t.Fatalf("the new epoch decides on its own: %+v", got)
	}
	if got := f.Recheck(delegated, renewed, ask(View, "c1", "quantity")); got.Allowed || got.Reason != ReasonStamp {
		t.Errorf("an old stamp presented with the new epoch: %+v", got)
	}
}

func TestSupportSessionApproverAndActorState(t *testing.T) {
	f := setup(t)
	f.subject("frank", Human)
	f.subject("grace", Human)
	f.grant("approval-carol", "carol", terms(ApproveSupportSession, One("c1")))
	f.grant("approval-bob", "bob", terms(ApproveSupportSession, One("c1")))
	entity := terms(ApproveSupportSession, One("c1"))
	entity.Scope.LegalEntity = One("le1")
	f.grant("approval-entity", "dave", entity)
	period := terms(ApproveSupportSession, One("c1"))
	period.Period = Months(month(t, 2026, 1), month(t, 2027, 1))
	f.grant("approval-period", "erin", period)
	f.grant("root", "alice", terms(View, One("c1"), "quantity"))
	f.homes("frank")
	if err := f.delegate("d1", "alice", "frank", "", true, terms(View, One("c1"), "quantity"), always); err != nil {
		t.Fatal(err)
	}
	if err := f.delegate("d2", "frank", "carol", "d1", false, terms(View, One("c1"), "quantity"), always); err != nil {
		t.Fatal(err)
	}
	base := SupportSession{ActorID: "grace", WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket", Kind: "usage_statement", Fields: []string{"quantity"}, Validity: always}
	try := func(id, actor string, approver Principal) error {
		session := base
		session.ID, session.ActorID = id, actor
		return f.AddSupportSession(session, approver)
	}
	elsewhere := who("carol")
	elsewhere.Installation = "other"
	inSession := who("carol")
	inSession.SessionID = "some"
	f.refuse("an approver whose grant names one legal entity", ErrDenied, try("n1", "grace", who("dave")))
	f.refuse("an approver whose grant has a bounded period", ErrDenied, try("n2", "grace", who("erin")))
	f.refuse("an approver from another installation", ErrDenied, try("n3", "grace", elsewhere))
	f.refuse("an approver acting inside a session", ErrDenied, try("n4", "grace", inSession))
	f.refuse("an approver who is the actor's delegate at the second level", ErrDenied, try("n5", "alice", who("carol")))
	if err := f.RevokeDelegation("d2", 1); err != nil {
		t.Fatal(err)
	}
	if err := try("after-revocation", "alice", who("carol")); err != nil {
		t.Errorf("the approver qualifies when the delegation from the actor is revoked: %v", err)
	}
	if _, err := f.PutSubject(Subject{ID: "bob", Kind: Human, State: Suspended, Epoch: 1}, 1); err != nil {
		t.Fatal(err)
	}
	f.refuse("a suspended approver", ErrDenied, try("n6", "grace", who("bob")))
	if _, err := f.PutSubject(Subject{ID: "grace", Kind: Human, State: Suspended, Epoch: 1}, 1); err != nil {
		t.Fatal(err)
	}
	f.refuse("a suspended actor", ErrInvalid, try("n7", "grace", who("carol")))
}

func TestClosedRecordsAndEpochsDoNotMoveBack(t *testing.T) {
	f := setup(t)
	f.grant("g", "alice", terms(View, One("c1"), "quantity"))
	f.grant("approval", "carol", terms(ApproveSupportSession, One("c1")))
	f.homes("bob")
	if err := f.delegate("d", "alice", "bob", "", false, terms(View, One("c1"), "quantity"), always); err != nil {
		t.Fatal(err)
	}
	if err := f.session("s", "dave", "carol", "quantity"); err != nil {
		t.Fatal(err)
	}
	for name, close := range map[string]func(uint64) error{
		"a grant":      func(v uint64) error { return f.RevokeGrant("g", v) },
		"a delegation": func(v uint64) error { return f.RevokeDelegation("d", v) },
		"a session":    func(v uint64) error { return f.EndSupportSession("s", v) },
	} {
		if err := close(1); err != nil {
			t.Fatal(err)
		}
		f.refuse("closing "+name+" twice", ErrConflict, close(2))
	}
	if _, err := f.PutSubject(Subject{ID: "erin", Kind: Human, State: Active, Epoch: 2}, 1); err != nil {
		t.Fatal(err)
	}
	_, err := f.PutSubject(Subject{ID: "erin", Kind: Human, State: Active, Epoch: 1}, 2)
	f.refuse("an epoch moved back", ErrInvalid, err)
}

func TestARegistryThatWasNotConstructedFailsClosed(t *testing.T) {
	var r Registry
	if got := r.Decide(who("alice"), ask(Issue, "c1")); got.Allowed || got.Code != AuthorityUnavailable {
		t.Errorf("decision: %+v", got)
	}
	if got := r.CheckApproval(who("alice"), "bob", ask(Approve, "c1")); got.Allowed || got.Code != AuthorityUnavailable {
		t.Errorf("approval check: %+v", got)
	}
	if got := r.Recheck(Stamp{}, who("alice"), ask(Issue, "c1")); got.Allowed || got.Code != AuthorityUnavailable {
		t.Errorf("recheck: %+v", got)
	}
	if got := r.RecheckApproval(Stamp{}, who("alice"), "bob", ask(Approve, "c1")); got.Allowed || got.Code != AuthorityUnavailable {
		t.Errorf("approval recheck: %+v", got)
	}
	if _, err := r.PutSubject(Subject{ID: "alice", Kind: Human, State: Active, Epoch: 1}, 0); !errors.Is(err, ErrUnavailable) {
		t.Errorf("subject write error = %v", err)
	}
	if _, err := r.PutClient(Client{ID: "app", State: Active}, 0); !errors.Is(err, ErrUnavailable) {
		t.Errorf("client write error = %v", err)
	}
	if err := r.AddGrant(Grant{ID: "g", SubjectID: "alice", Terms: terms(Issue, One("c1")), Validity: always}); err == nil {
		t.Errorf("grant write accepted")
	}
	if err := r.AddSupportSession(SupportSession{}, who("alice")); !errors.Is(err, ErrUnavailable) {
		t.Errorf("session write error = %v", err)
	}
	r.SetAvailable(true)
	if got := r.Decide(who("alice"), ask(Issue, "c1")); got.Allowed || got.Code != AuthorityUnavailable {
		t.Errorf("decision after an availability call: %+v", got)
	}
}

func TestRepeatingTheCurrentAvailabilityKeepsStamps(t *testing.T) {
	f := setup(t)
	f.grant("g", "alice", terms(Issue, One("c1")))
	first := f.expect("the decision", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	f.SetAvailable(true)
	if got := f.Recheck(first.Stamp, who("alice"), ask(Issue, "c1")); !got.Allowed {
		t.Errorf("recheck after availability was set to its current value: %+v", got)
	}
}
