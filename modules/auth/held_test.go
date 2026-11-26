package auth

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestLookupsByHolderEqualAFilterOfTheWholeLists(t *testing.T) {
	for seed := int64(1); seed <= 60; seed++ {
		r := rand.New(rand.NewSource(seed))
		w := randomWorld(t, r)
		for _, g := range w.Grants() {
			if r.Intn(4) == 0 {
				if err := w.RevokeGrant(g.ID, g.Version); err != nil {
					t.Fatalf("seed %d: revoke %s: %v", seed, g.ID, err)
				}
			}
		}
		for _, d := range w.Delegations() {
			if r.Intn(4) == 0 && !d.Closed {
				if err := w.RevokeDelegation(d.ID, d.Version); err != nil {
					t.Fatalf("seed %d: revoke %s: %v", seed, d.ID, err)
				}
			}
		}
		at := *w.now
		grants, delegations := w.GrantPage("", 1000, at)
		_ = delegations
		all, _ := w.DelegationPage("", 1000, at)
		subjects, clients, actions := map[string]bool{"nobody": true}, map[string]bool{"no-client": true}, map[Action]bool{AdministerInstallation: true}
		for _, g := range grants {
			subjects[g.SubjectID], clients[g.ClientID] = true, true
			if InstallationAction(g.Terms.Action) {
				actions[g.Terms.Action] = true
			}
		}
		for _, d := range all {
			subjects[d.DelegatorID], subjects[d.DelegateID] = true, true
		}
		keep := func(pass func(GrantView) bool) []GrantView {
			out := []GrantView{}
			for _, g := range grants {
				if pass(g) {
					out = append(out, g)
				}
			}
			return out
		}
		for subject := range subjects {
			if subject == "" {
				continue
			}
			want := keep(func(g GrantView) bool { return g.SubjectID == subject })
			if got, more := w.GrantsOfSubject(subject, 1000, at); more || !reflect.DeepEqual(got, want) {
				t.Fatalf("seed %d: grants of %s: %d rows, want %d", seed, subject, len(got), len(want))
			}
			if len(want) > 1 {
				if got, more := w.GrantsOfSubject(subject, len(want)-1, at); !more || !reflect.DeepEqual(got, want[:len(want)-1]) {
					t.Fatalf("seed %d: grants of %s under a limit", seed, subject)
				}
			}
			wanted := []DelegationView{}
			for _, d := range all {
				if d.DelegatorID == subject || d.DelegateID == subject {
					wanted = append(wanted, d)
				}
			}
			if got, more := w.DelegationsOfUser(subject, 1000, at); more || !reflect.DeepEqual(got, wanted) {
				t.Fatalf("seed %d: delegations of %s: %d rows, want %d", seed, subject, len(got), len(wanted))
			}
			if len(wanted) > 1 {
				if got, more := w.DelegationsOfUser(subject, len(wanted)-1, at); !more || !reflect.DeepEqual(got, wanted[:len(wanted)-1]) {
					t.Fatalf("seed %d: delegations of %s under a limit", seed, subject)
				}
			}
		}
		for client := range clients {
			if client == "" {
				continue
			}
			if got, more := w.GrantsOfClient(client, 1000, at); more || !reflect.DeepEqual(got, keep(func(g GrantView) bool { return g.ClientID == client })) {
				t.Fatalf("seed %d: grants of client %s", seed, client)
			}
		}
		for action := range actions {
			if got, more := w.InstallationGrants(action, 1000, at); more || !reflect.DeepEqual(got, keep(func(g GrantView) bool { return g.Terms.Action == action })) {
				t.Fatalf("seed %d: grants of action %s", seed, action)
			}
		}
		if got, more := w.GrantsOfSubject("nobody", 0, at); got != nil || more {
			t.Fatalf("seed %d: a limit of zero", seed)
		}
		if got, more := w.DelegationsOfUser("nobody", 0, at); got != nil || more {
			t.Fatalf("seed %d: a limit of zero", seed)
		}
	}
}

func TestLookupsByClientAndInstallationAction(t *testing.T) {
	f := setupInstallation(t, true)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"adm-b", "adm-a"} {
		subject := map[string]string{"adm-a": "alice", "adm-b": "bob"}[id]
		if err := f.AddGrant(Grant{ID: id, SubjectID: subject, Terms: installationTerms("w1"), Validity: always}); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	usage := terms(View, All(), "quantity")
	for _, id := range []string{"app-2", "app-1"} {
		if err := f.AddGrant(Grant{ID: id, ClientID: "app", Terms: usage, Validity: always}); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	if err := f.AddGrant(Grant{ID: "spare-1", ClientID: "spare", Terms: usage, Validity: always}); err != nil {
		t.Fatal(err)
	}
	if err := f.AddGrant(Grant{ID: "app-adm", ClientID: "app", Terms: installationTerms("w1"), Validity: always}); err != nil {
		t.Fatal(err)
	}
	ids := func(rows []GrantView) string {
		out := ""
		for _, g := range rows {
			out += g.ID + " "
		}
		return out
	}
	whole := f.Grants()
	filter := func(pass func(GrantView) bool) string {
		kept := []GrantView{}
		for _, g := range whole {
			if pass(g) {
				kept = append(kept, g)
			}
		}
		return ids(kept)
	}
	if got, more := f.GrantsOfClient("app", 10, *f.now); more || ids(got) != "app-1 app-2 app-adm " || ids(got) != filter(func(g GrantView) bool { return g.ClientID == "app" }) {
		t.Errorf("the application's grants: %s", ids(got))
	}
	if got, more := f.GrantsOfClient("app", 2, *f.now); !more || ids(got) != "app-1 app-2 " {
		t.Errorf("the application's grants under a limit: %s %v", ids(got), more)
	}
	if got, _ := f.GrantsOfClient("spare", 10, *f.now); ids(got) != "spare-1 " {
		t.Errorf("the other client's grants: %s", ids(got))
	}
	holders, more := f.InstallationGrants(AdministerInstallation, 10, *f.now)
	if more || ids(holders) != "adm-a adm-b app-adm " || ids(holders) != filter(func(g GrantView) bool { return g.Terms.Action == AdministerInstallation }) {
		t.Errorf("the administrator grants: %s", ids(holders))
	}
	for _, g := range holders {
		if g.SubjectID != "" && !g.Effective {
			t.Errorf("%s is not effective", g.ID)
		}
	}
	if got, _ := f.InstallationGrants(View, 10, *f.now); len(got) != 0 {
		t.Errorf("an ordinary action is not indexed: %s", ids(got))
	}
	if err := f.RevokeGrant("adm-a", 1); err != nil {
		t.Fatal(err)
	}
	after, _ := f.InstallationGrants(AdministerInstallation, 10, *f.now)
	if ids(after) != "adm-a adm-b app-adm " || !after[0].Closed || after[0].Effective {
		t.Errorf("a revoked grant stays listed as closed: %+v", after[0])
	}
}
