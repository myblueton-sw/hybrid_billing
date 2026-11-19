package auth

import (
	"errors"
	"math/rand/v2"
	"sync"
	"testing"
	"time"
)

func setupInstallation(t testing.TB, source bool) fixture {
	t.Helper()
	now := start
	catalog := Catalog{AuthorClient: "app", Kinds: []string{InstallationKind, "usage_statement"}, Fields: map[string]FieldClass{"quantity": General}}
	var r *Registry
	var err error
	if source {
		s := &fakeSource{subjects: map[string]Subject{}, members: map[[2]string]fakeMember{}, generation: 1}
		for _, id := range []string{"alice", "bob", "carol"} {
			s.subjects[id] = Subject{ID: id, Kind: Human, State: Active, Epoch: 1}
			s.members[[2]string{"w1", id}] = fakeMember{joined: 1, active: true}
		}
		s.subjects["robot"] = Subject{ID: "robot", Kind: Service, State: Active, Epoch: 1}
		s.members[[2]string{"w1", "robot"}] = fakeMember{joined: 1, active: true}
		s.members[[2]string{"w2", "alice"}] = fakeMember{joined: 1, active: true}
		r, err = NewRegistryWithSubjects("synthetic-installation", catalog, func() time.Time { return now }, s)
	} else {
		r, err = NewRegistry("synthetic-installation", catalog, func() time.Time { return now })
	}
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{r, &now, t}
	if !source {
		for _, id := range []string{"alice", "bob", "carol"} {
			f.subject(id, Human)
		}
	}
	if _, err := r.PutClient(Client{ID: "app", State: Active}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PutClient(Client{ID: "spare", State: Active}, 0); err != nil {
		t.Fatal(err)
	}
	return f
}

func installationTerms(workspace string) Terms {
	return Terms{Action: AdministerInstallation, Kind: InstallationKind, Period: AnyPeriod(),
		Scope: Scope{WorkspaceID: workspace, Customer: All(), LegalEntity: All(), Contract: All()}}
}

func administer(workspace string) Request {
	return Request{Action: AdministerInstallation, Kind: InstallationKind, Resource: Resource{WorkspaceID: workspace}}
}

func viaApp(subject string) Principal {
	p := who(subject)
	p.ClientID = "app"
	return p
}

func TestInstallationWorkspaceIsSetOnce(t *testing.T) {
	f := setupInstallation(t, true)
	if err := f.SetInstallationWorkspace("bad id"); !errors.Is(err, ErrInvalid) {
		t.Errorf("an invalid workspace: %v", err)
	}
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Errorf("setting the same workspace again: %v", err)
	}
	if err := f.SetInstallationWorkspace("w2"); !errors.Is(err, ErrConflict) {
		t.Errorf("a different workspace: %v", err)
	}
	if f.installationWorkspace != "w1" {
		t.Errorf("workspace %q", f.installationWorkspace)
	}
}

func TestInstallationActionIsDescribedOnlyForItsKind(t *testing.T) {
	f := setupInstallation(t, true)
	if !f.Describes(AdministerInstallation, InstallationKind, nil) {
		t.Error("the action with its kind")
	}
	if f.Describes(AdministerInstallation, "usage_statement", nil) {
		t.Error("the action with another kind")
	}
	if f.Describes(AdministerInstallation, InstallationKind, []string{"quantity"}) {
		t.Error("the action with fields")
	}
	if !f.Describes(View, InstallationKind, []string{"quantity"}) {
		t.Error("another action with the installation kind")
	}
}

func TestInstallationGrantsBeforeTheWorkspaceIsSet(t *testing.T) {
	f := setupInstallation(t, true)
	if err := f.AddGrant(Grant{ID: "g", SubjectID: "alice", Terms: installationTerms("w1"), Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a grant before the workspace is set: %v", err)
	}
	f.expect("a decision before the workspace is set", who("alice"), administer("w1"), false, ReasonInstallationScope)
}

func TestInstallationGrantValidation(t *testing.T) {
	f := setupInstallation(t, true)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	narrow := func(change func(*Terms)) Terms {
		t := installationTerms("w1")
		change(&t)
		return t
	}
	refused := map[string]Grant{
		"another workspace":   {ID: "a", SubjectID: "alice", Terms: installationTerms("w2")},
		"another kind":        {ID: "b", SubjectID: "alice", Terms: narrow(func(t *Terms) { t.Kind = "usage_statement" })},
		"a customer":          {ID: "c", SubjectID: "alice", Terms: narrow(func(t *Terms) { t.Scope.Customer = One("c1") })},
		"a legal entity":      {ID: "d", SubjectID: "alice", Terms: narrow(func(t *Terms) { t.Scope.LegalEntity = One("e1") })},
		"a contract":          {ID: "e", SubjectID: "alice", Terms: narrow(func(t *Terms) { t.Scope.Contract = One("k1") })},
		"a bounded period":    {ID: "f", SubjectID: "alice", Terms: narrow(func(t *Terms) { t.Period = Months(month(f.t, 2026, 1), month(f.t, 2026, 2)) })},
		"fields":              {ID: "g", SubjectID: "alice", Terms: narrow(func(t *Terms) { t.Fields = []string{"quantity"} })},
		"a service subject":   {ID: "h", SubjectID: "robot", Terms: installationTerms("w1")},
		"a non-member":        {ID: "i", SubjectID: "nobody", Terms: installationTerms("w1")},
		"a client and person": {ID: "j", SubjectID: "alice", ClientID: "app", Terms: installationTerms("w1")},
		"a suspended subject": {ID: "k", SubjectID: "carol", Terms: installationTerms("w1")},
	}
	f.source.(*fakeSource).set("carol", Suspended, 1)
	for name, grant := range refused {
		grant.Validity = always
		if err := f.AddGrant(grant); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	f.grant("admin", "alice", installationTerms("w1"))
	f.clientGrant("console", installationTerms("w1"))
	f.grant("home", "bob", terms(RequestApproval, One("c9")))
	if err := f.delegate("d", "alice", "bob", "", false, installationTerms("w1"), always); !errors.Is(err, ErrInvalid) {
		t.Errorf("a delegation of the action: %v", err)
	}
}

func TestInstallationNeedsASubjectSource(t *testing.T) {
	f := setupInstallation(t, false)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	if err := f.AddGrant(Grant{ID: "g", SubjectID: "alice", Terms: installationTerms("w1"), Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a grant without a source: %v", err)
	}
	f.expect("a decision without a source", who("alice"), administer("w1"), false, ReasonInstallationScope)
}

func TestInstallationDecisions(t *testing.T) {
	f := setupInstallation(t, true)
	source := f.source.(*fakeSource)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	f.grant("admin", "alice", installationTerms("w1"))
	f.grant("other", "carol", installationTerms("w1"))
	f.clientGrant("console", installationTerms("w1"))
	first := f.expect("the administrator through the console", viaApp("alice"), administer("w1"), true, ReasonAllowed)
	if len(first.Delegations) != 0 || first.ClientID != "app" || first.ActorID != "alice" {
		t.Errorf("allowed decision %+v", first)
	}
	f.expect("the administrator without a client", who("alice"), administer("w1"), true, ReasonAllowed)
	f.expect("another workspace", who("alice"), administer("w2"), false, ReasonInstallationScope)
	f.expect("no grant", who("bob"), administer("w1"), false, ReasonInstallationScope)
	spare := who("alice")
	spare.ClientID = "spare"
	f.expect("a client without a grant", spare, administer("w1"), false, ReasonClient)
	unknown := who("alice")
	unknown.ClientID = "ghost"
	f.expect("an unknown client", unknown, administer("w1"), false, ReasonClient)
	session := who("alice")
	session.SessionID = "s1"
	f.expect("a support session", session, administer("w1"), false, ReasonInstallationScope)
	other := administer("w1")
	other.Kind = "usage_statement"
	f.expect("another kind", who("alice"), other, false, ReasonInvalidRequest)
	fields := administer("w1")
	fields.Fields = []string{"quantity"}
	f.expect("fields", who("alice"), fields, false, ReasonInvalidRequest)
	robot := who("robot")
	f.expect("a service subject", robot, administer("w1"), false, ReasonInstallationScope)
	if got := f.Recheck(first.Stamp, viaApp("alice"), administer("w1")); !got.Allowed {
		t.Errorf("recheck %+v", got)
	}
	source.subjects["robot"] = Subject{ID: "robot", Kind: Human, State: Active, Epoch: 1}
	if err := f.AddGrant(Grant{ID: "robot", SubjectID: "robot", Terms: installationTerms("w1"), Validity: always}); err != nil {
		t.Fatal(err)
	}
	source.subjects["robot"] = Subject{ID: "robot", Kind: Service, State: Active, Epoch: 1}
	f.expect("a subject that became a service", robot, administer("w1"), false, ReasonInstallationScope)
	if _, err := f.PutClient(Client{ID: "app", State: Suspended}, 1); err != nil {
		t.Fatal(err)
	}
	f.expect("a suspended client", viaApp("alice"), administer("w1"), false, ReasonClient)
	source.set("carol", Suspended, 1)
	f.expect("a suspended subject", who("carol"), administer("w1"), false, ReasonPrincipal)
	source.join("w1", "alice", 1, false)
	f.expect("a departed member", who("alice"), administer("w1"), false, ReasonInstallationScope)
	if got := f.Recheck(first.Stamp, viaApp("alice"), administer("w1")); got.Allowed {
		t.Errorf("recheck after change %+v", got)
	}
}

func TestInstallationIgnoresDelegations(t *testing.T) {
	f := setupInstallation(t, true)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	f.grant("admin", "alice", installationTerms("w1"))
	f.delegations["forged"] = versioned[Delegation]{value: Delegation{ID: "forged", DelegatorID: "alice", DelegateID: "bob", Terms: installationTerms("w1"), Validity: always}, version: 1}
	f.expect("a delegate", who("bob"), administer("w1"), false, ReasonInstallationScope)
	f.expect("the delegator", who("alice"), administer("w1"), true, ReasonAllowed)
}

type lockedSource struct {
	mu    sync.Mutex
	inner *fakeSource
}

func (s *lockedSource) Subject(id string) (Subject, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Subject(id)
}

func (s *lockedSource) Member(workspaceID, id string) (uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Member(workspaceID, id)
}

func (s *lockedSource) Generation() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Generation()
}

func TestInstallationWorkspaceRace(t *testing.T) {
	f := setupInstallation(t, true)
	f.source = &lockedSource{inner: f.source.(*fakeSource)}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() { defer wg.Done(); f.SetInstallationWorkspace("w1") }()
		go func() { defer wg.Done(); f.Decide(who("alice"), administer("w1")) }()
	}
	wg.Wait()
	if f.installationWorkspace != "w1" {
		t.Errorf("workspace %q", f.installationWorkspace)
	}
}

func TestInstallationChecksTheWorkspaceBeyondTheGrant(t *testing.T) {
	f := setupInstallation(t, true)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	f.grants["forged"] = versioned[Grant]{value: Grant{ID: "forged", SubjectID: "alice", Terms: installationTerms("w2"), Validity: always}, version: 1}
	f.grantJoined["forged"] = 1
	f.expect("a grant in another workspace that bypassed validation", who("alice"), administer("w2"), false, ReasonInstallationScope)
}

func TestDelegationRegistrationRefusesEachDefect(t *testing.T) {
	f := setupInstallation(t, true)
	f.grant("backing", "alice", terms(Issue, One("c1")))
	f.grant("home", "bob", terms(RequestApproval, One("c9")))
	if err := f.delegate("bad id", "alice", "bob", "", false, terms(Issue, One("c1")), always); !errors.Is(err, ErrInvalid) {
		t.Errorf("an invalid delegation ID: %v", err)
	}
	if err := f.delegate("d", "alice", "bob", "", false, terms(Issue, One("c1")), Validity{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("an invalid validity: %v", err)
	}
	if err := f.delegate("d", "alice", "bob", "", false, terms(Issue, One("c1")), always); err != nil {
		t.Errorf("a valid delegation: %v", err)
	}
}

func TestInstallationDecisionsMatchAModel(t *testing.T) {
	rng := rand.New(rand.NewPCG(262, 1))
	pick := func(values ...string) string { return values[rng.IntN(len(values))] }
	members := map[string]map[string]bool{"w1": {"alice": true, "bob": true, "robot": true}, "w2": {"alice": true}}
	reasons := map[Reason]int{}
	for i := range 2000 {
		f := setupInstallation(t, true)
		source := f.source.(*fakeSource)
		install := pick("", "w1", "w2")
		if install != "" {
			if err := f.SetInstallationWorkspace(install); err != nil {
				t.Fatal(err)
			}
		}
		subject := pick("alice", "bob", "robot")
		human := subject != "robot"
		hasSubject, hasClient := false, false
		if rng.IntN(4) > 0 {
			ws := pick("w1", "w2")
			ok := install != "" && ws == install && human && members[ws][subject]
			err := f.AddGrant(Grant{ID: "s", SubjectID: subject, Terms: installationTerms(ws), Validity: always})
			if (err == nil) != ok {
				t.Fatalf("case %d: subject grant in %s with installation %q for %s: %v", i, ws, install, subject, err)
			}
			hasSubject = ok
		}
		if rng.IntN(4) > 0 {
			ws := pick("w1", "w2")
			ok := install != "" && ws == install
			if err := f.AddGrant(Grant{ID: "c", ClientID: "app", Terms: installationTerms(ws), Validity: always}); (err == nil) != ok {
				t.Fatalf("case %d: client grant in %s with installation %q: %v", i, ws, install, err)
			}
			hasClient = ok
		}
		member := install != "" && members[install][subject]
		if member && rng.IntN(5) == 0 {
			source.join(install, subject, 1, false)
			member = false
		}
		suspended := rng.IntN(6) == 0
		if suspended {
			source.set(subject, Suspended, 1)
		}
		principal := who(subject)
		principal.ClientID = pick("", "app", "spare")
		principal.SessionID = pick("", "", "", "s1")
		workspace := pick("w1", "w2")
		want := ReasonAllowed
		switch {
		case suspended || (principal.ClientID != "" && principal.SessionID != ""):
			want = ReasonPrincipal
		case install == "" || workspace != install || principal.SessionID != "" || !human || !member || !hasSubject:
			want = ReasonInstallationScope
		case principal.ClientID != "" && (principal.ClientID != "app" || !hasClient):
			want = ReasonClient
		}
		got := f.Decide(principal, administer(workspace))
		if got.Reason != want || got.Allowed != (want == ReasonAllowed) {
			t.Fatalf("case %d: install %q subject %s grant %v client %q grant %v member %v suspended %v session %q workspace %s: got %s want %s",
				i, install, subject, hasSubject, principal.ClientID, hasClient, member, suspended, principal.SessionID, workspace, got.Reason, want)
		}
		if got.Allowed {
			if again := f.Recheck(got.Stamp, principal, administer(workspace)); !again.Allowed {
				t.Fatalf("case %d: recheck %+v", i, again)
			}
		}
		reasons[want]++
	}
	for _, reason := range []Reason{ReasonAllowed, ReasonPrincipal, ReasonInstallationScope, ReasonClient} {
		if reasons[reason] < 20 {
			t.Errorf("model reached %s only %d times", reason, reasons[reason])
		}
	}
}

func TestExplainRefusesInstallationForAHolder(t *testing.T) {
	f := setupInstallation(t, true)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	f.grant("admin", "alice", installationTerms("w1"))
	if got := f.Explain("alice", administer("w1")); got.Allowed || got.Reason != ReasonInvalidRequest {
		t.Errorf("explain an installation action %+v", got)
	}
}
