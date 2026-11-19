package auth

import (
	"errors"
	"testing"
	"time"
)

func TestIntegrationRightsAreSeparateFromEveryOtherRight(t *testing.T) {
	f := setupInstallation(t, true)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	all := append(append([]Action{AdministerInstallation, RevokeAccessInBulk}, trustRights...), IntegrationRights()...)
	for _, right := range all {
		f.clientGrant(string(right)+"-app", termsFor(right, "w1"))
	}
	for _, held := range all {
		g := setupInstallation(t, true)
		if err := g.SetInstallationWorkspace("w1"); err != nil {
			t.Fatal(err)
		}
		for _, right := range all {
			g.clientGrant(string(right)+"-app", termsFor(right, "w1"))
		}
		g.grant("only-"+string(held), "bob", termsFor(held, "w1"))
		for _, asked := range all {
			got := g.Decide(viaApp("bob"), requestFor(asked, "w1"))
			if got.Allowed != (asked == held) {
				t.Errorf("holding %s, asking %s: %+v", held, asked, got)
			}
		}
	}
	for _, right := range IntegrationRights() {
		if err := f.AddGrant(Grant{ID: "robot-" + string(right), SubjectID: "robot", Terms: termsFor(right, "w1"), Validity: always}); err == nil {
			t.Errorf("a service subject received %s", right)
		}
		if err := f.AddDelegation(Delegation{ID: "del-" + string(right), DelegatorID: "alice", DelegateID: "carol", Terms: termsFor(right, "w1"), Validity: always}); err == nil {
			t.Errorf("%s was delegated", right)
		}
		bad := termsFor(right, "w1")
		bad.Kind = "usage_statement"
		if err := f.AddGrant(Grant{ID: "kind-" + string(right), SubjectID: "carol", Terms: bad, Validity: always}); err == nil {
			t.Errorf("%s on another kind", right)
		}
	}
}

func authorFixture(t *testing.T) fixture {
	f := setupInstallation(t, true)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	f.clientGrant("modules-app", termsFor(AdministerModules, "w1"))
	return f
}

func TestCheckAuthorRules(t *testing.T) {
	f := authorFixture(t)
	ask := requestFor(AdministerModules, "w1")
	if got := f.CheckAuthor(viaApp("bob"), "bob", ask); !got.Allowed {
		t.Fatalf("the author without a grant %+v", got)
	}
	cases := map[string]struct {
		principal Principal
		author    string
		request   Request
		reason    Reason
	}{
		"empty author":     {viaApp("bob"), "", ask, ReasonNotAuthor},
		"another author":   {viaApp("bob"), "alice", ask, ReasonNotAuthor},
		"no client":        {who("bob"), "bob", ask, ReasonPrincipal},
		"spare client":     {Principal{Installation: "synthetic-installation", SubjectID: "bob", Epoch: 1, ClientID: "spare"}, "bob", ask, ReasonClient},
		"support session":  {Principal{Installation: "synthetic-installation", SubjectID: "bob", Epoch: 1, ClientID: "app", SessionID: "s1"}, "bob", ask, ReasonPrincipal},
		"service subject":  {viaApp("robot"), "robot", ask, ReasonNotHuman},
		"stale epoch":      {Principal{Installation: "synthetic-installation", SubjectID: "bob", Epoch: 9, ClientID: "app"}, "bob", ask, ReasonPrincipal},
		"other workspace":  {viaApp("bob"), "bob", requestFor(AdministerModules, "w2"), ReasonInvalidRequest},
		"ordinary action":  {viaApp("bob"), "bob", ask0(), ReasonInvalidRequest},
		"valid ordinary":   {viaApp("bob"), "bob", Request{Action: Issue, Kind: "usage_statement", Resource: Resource{WorkspaceID: "w1"}}, ReasonInvalidRequest},
		"right not on app": {viaApp("bob"), "bob", requestFor(ApproveModules, "w1"), ReasonClient},
	}
	for name, c := range cases {
		got := f.CheckAuthor(c.principal, c.author, c.request)
		if got.Allowed || got.Reason != c.reason {
			t.Errorf("%s: %+v", name, got)
		}
	}
	f.SetAvailable(false)
	if got := f.CheckAuthor(viaApp("bob"), "bob", ask); got.Code != AuthorityUnavailable {
		t.Errorf("unavailable %+v", got)
	}
}

func ask0() Request {
	return Request{Action: View, Kind: "usage_statement", Resource: Resource{WorkspaceID: "w1"}}
}

func TestRecheckAuthorBindsTheStamp(t *testing.T) {
	f := authorFixture(t)
	ask := requestFor(AdministerModules, "w1")
	first := f.CheckAuthor(viaApp("bob"), "bob", ask)
	if got := f.RecheckAuthor(first.Stamp, viaApp("bob"), "bob", ask); !got.Allowed {
		t.Fatalf("recheck %+v", got)
	}
	if got := f.RecheckAuthor(first.Stamp, viaApp("bob"), "alice", ask); got.Allowed {
		t.Errorf("recheck for another author %+v", got)
	}
	if got := f.RecheckAuthor(Stamp{}, viaApp("bob"), "bob", ask); got.Allowed || got.Reason != ReasonStamp {
		t.Errorf("an unissued stamp %+v", got)
	}
	f.grant("bob-modules", "bob", termsFor(AdministerModules, "w1"))
	decided := f.Decide(viaApp("bob"), ask)
	if !decided.Allowed {
		t.Fatalf("decide %+v", decided)
	}
	if got := f.RecheckAuthor(decided.Stamp, viaApp("bob"), "bob", ask); got.Allowed || got.Reason != ReasonStamp {
		t.Errorf("a decide stamp served an author recheck %+v", got)
	}
	if got := f.Recheck(first.Stamp, viaApp("bob"), ask); got.Allowed {
		t.Errorf("an author stamp served a decide recheck %+v", got)
	}
	source := f.source.(*fakeSource)
	subject := source.subjects["bob"]
	subject.State = Suspended
	source.subjects["bob"] = subject
	source.generation++
	if got := f.RecheckAuthor(first.Stamp, viaApp("bob"), "bob", ask); got.Allowed {
		t.Errorf("a suspended author %+v", got)
	}
}

func TestCheckAuthorRefusesAnInactiveClient(t *testing.T) {
	f := authorFixture(t)
	if _, err := f.PutClient(Client{ID: "app", State: Suspended}, 1); err != nil {
		t.Fatal(err)
	}
	if got := f.CheckAuthor(viaApp("bob"), "bob", requestFor(AdministerModules, "w1")); got.Allowed || got.Reason != ReasonClient {
		t.Errorf("an inactive client %+v", got)
	}
}

func TestAuthorRefusesAnotherClientWithTheSameGrant(t *testing.T) {
	f := authorFixture(t)
	ask := requestFor(AdministerModules, "w1")
	if _, err := f.PutClient(Client{ID: "other", State: Active}, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.AddGrant(Grant{ID: "other-modules", ClientID: "other", Terms: termsFor(AdministerModules, "w1"), Validity: always}); err != nil {
		t.Fatal(err)
	}
	first := f.CheckAuthor(viaApp("bob"), "bob", ask)
	if !first.Allowed {
		t.Fatalf("configured author client: %+v", first)
	}
	other := viaApp("bob")
	other.ClientID = "other"
	if got := f.CheckAuthor(other, "bob", ask); got.Allowed || got.Reason != ReasonClient {
		t.Errorf("another granted client: %+v", got)
	}
	if got := f.RecheckAuthor(first.Stamp, other, "bob", ask); got.Allowed || got.Reason != ReasonClient {
		t.Errorf("another granted client on recheck: %+v", got)
	}
}

func TestAuthorClientConfigurationIsImmutableAndFailsClosed(t *testing.T) {
	for _, client := range []string{"", "app"} {
		f := authorFixture(t)
		catalog := Catalog{AuthorClient: client, Kinds: []string{InstallationKind}}
		r, err := NewRegistryWithSubjects("synthetic-installation", catalog, f.clock, f.source)
		if err != nil {
			t.Fatal(err)
		}
		catalog.AuthorClient = "other"
		f.Registry = r
		if err := f.SetInstallationWorkspace("w1"); err != nil {
			t.Fatal(err)
		}
		if _, err := f.PutClient(Client{ID: "app", State: Active}, 0); err != nil {
			t.Fatal(err)
		}
		ask := requestFor(AdministerModules, "w1")
		f.clientGrant("app-modules", termsFor(AdministerModules, "w1"))
		f.grant("bob-modules", "bob", termsFor(AdministerModules, "w1"))
		if got := f.Decide(viaApp("bob"), ask); !got.Allowed {
			t.Errorf("ordinary decision changed with author client %q: %+v", client, got)
		}
		got := f.CheckAuthor(viaApp("bob"), "bob", ask)
		if got.Allowed != (client == "app") || (client == "" && got.Reason != ReasonClient) {
			t.Errorf("configured author client %q: %+v", client, got)
		}
		if got := f.RecheckAuthor(got.Stamp, viaApp("bob"), "bob", ask); got.Allowed != (client == "app") {
			t.Errorf("configured author client recheck %q: %+v", client, got)
		}
	}
	if _, err := NewRegistry("synthetic-installation", Catalog{AuthorClient: "bad client", Kinds: []string{InstallationKind}}, func() time.Time { return start }); !errors.Is(err, ErrInvalid) {
		t.Errorf("invalid author client: %v", err)
	}
}
