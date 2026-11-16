package auth

import (
	"sync"
	"testing"
	"time"
)

func TestListingStates(t *testing.T) {
	f := setupSource(t)
	f.grant("a-live", "alice", terms(Issue, One("c1")))
	f.grant("b-revoked", "bob", terms(Issue, All()))
	if err := f.RevokeGrant("b-revoked", 1); err != nil {
		t.Fatal(err)
	}
	if err := f.AddGrant(Grant{ID: "c-later", SubjectID: "carol", Terms: terms(Issue, All()), Validity: Validity{NotBefore: start.Add(time.Hour), NotAfter: start.Add(2 * time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	f.clientGrant("d-client", terms(Issue, All()))
	f.grant("e-fields", "dave", terms(View, All(), "quantity"))
	f.grant("f-bob", "bob", terms(RequestApproval, One("c9")))
	f.grant("g-carol", "carol", terms(RequestApproval, One("c9")))
	if err := f.delegate("del-1", "alice", "bob", "", false, terms(Issue, One("c1")), always); err != nil {
		t.Fatal(err)
	}
	if err := f.delegate("del-0", "alice", "carol", "", false, terms(Issue, One("c1")), always); err != nil {
		t.Fatal(err)
	}
	if err := f.RevokeDelegation("del-0", 1); err != nil {
		t.Fatal(err)
	}
	got := f.Grants()
	want := map[string][4]bool{"a-live": {false, true, true}, "b-revoked": {true, false, false}, "c-later": {false, false, false}, "d-client": {false, true, true}, "e-fields": {false, true, true}, "f-bob": {false, true, true}, "g-carol": {false, true, true}}
	if len(got) != len(want) {
		t.Fatalf("grants %d", len(got))
	}
	for i, g := range got {
		w := want[g.ID]
		if g.Closed != w[0] || g.Active != w[1] || g.Effective != w[2] || g.Version == 0 {
			t.Errorf("%s: %+v", g.ID, g)
		}
		if i > 0 && got[i-1].ID >= g.ID {
			t.Error("not sorted")
		}
	}
	got[4].Terms.Fields[0] = "changed"
	if f.Grants()[4].Terms.Fields[0] != "quantity" {
		t.Error("grant fields alias the registry")
	}
	ds := f.Delegations()
	if len(ds) != 2 || ds[0].ID != "del-0" || !ds[0].Closed || ds[0].Active || ds[1].Closed || !ds[1].Active || ds[1].DelegateID != "bob" {
		t.Errorf("delegations %+v", ds)
	}
	ds[1].Terms.Fields = append(ds[1].Terms.Fields, "x")
	if len(f.Delegations()[1].Terms.Fields) != 0 {
		t.Error("delegation fields alias the registry")
	}
	f.source.join("w1", "alice", 2, true)
	for _, g := range f.Grants() {
		if g.ID == "a-live" && (!g.Active || g.Effective) {
			t.Errorf("grant after membership change %+v", g)
		}
	}
	if !All().Any() || One("x").Any() || One("x").ID() != "x" || !AnyPeriod().Any() {
		t.Error("selector accessors")
	}
	from, to := month(t, 2026, 1), month(t, 2026, 3)
	if p := Months(from, to); p.Any() || p.From() != from || p.To() != to {
		t.Error("period accessors")
	}
}

func TestExplainMatchesDecideWithoutChangingState(t *testing.T) {
	f := setupSource(t)
	f.grant("g", "alice", terms(Issue, One("c1")))
	f.grant("home", "bob", terms(RequestApproval, One("c9")))
	if err := f.delegate("d", "alice", "bob", "", false, terms(Issue, One("c1")), always); err != nil {
		t.Fatal(err)
	}
	generation := f.generation
	for _, c := range []struct {
		subject string
		req     Request
	}{{"alice", ask(Issue, "c1")}, {"alice", ask(Issue, "c2")}, {"bob", ask(Issue, "c1")}, {"carol", ask(Issue, "c1")}, {"nobody", ask(Issue, "c1")}} {
		got := f.Explain(c.subject, c.req)
		want := f.Decide(who(c.subject), c.req)
		if got.Allowed != want.Allowed || got.Reason != want.Reason || got.Code != want.Code || len(got.Delegations) != len(want.Delegations) {
			t.Errorf("%s: %+v vs %+v", c.subject, got, want)
		}
	}
	if f.Explain("bob", ask(Issue, "c1")).Delegations[0] != "d" {
		t.Error("delegation chain")
	}
	if f.generation != generation {
		t.Error("explain changed the generation")
	}
	for _, action := range []Action{AdministerInstallation, Approve, ApproveSupportSession} {
		if got := f.Explain("alice", Request{Action: action, Kind: "usage_statement", Resource: Resource{WorkspaceID: "w1", CustomerID: "c1"}}); got.Allowed || got.Reason != ReasonInvalidRequest {
			t.Errorf("%s: %+v", action, got)
		}
	}
	f.source.set("alice", Suspended, 1)
	if got := f.Explain("alice", ask(Issue, "c1")); got.Allowed || got.Reason != ReasonPrincipal {
		t.Errorf("suspended %+v", got)
	}
	f.source.changeAt = f.source.reads + 2
	if got := f.Explain("bob", ask(Issue, "c1")); got.Allowed || got.Reason != ReasonStamp {
		t.Errorf("generation change %+v", got)
	}
}

func TestExplainRacesChanges(t *testing.T) {
	f := setupInstallation(t, true)
	f.source = &lockedSource{inner: f.source.(*fakeSource)}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				f.Explain("alice", Request{Action: View, Kind: "usage_statement", Resource: Resource{WorkspaceID: "w1"}, Fields: []string{"quantity"}})
				f.Grants()
				f.Delegations()
			}
		}()
		go func(i int) {
			defer wg.Done()
			id := "race-" + string(rune('a'+i))
			if err := f.AddGrant(Grant{ID: id, SubjectID: "alice", Terms: Terms{Action: View, Kind: "usage_statement", Scope: Scope{WorkspaceID: "w1", Customer: All(), LegalEntity: All(), Contract: All()}, Fields: []string{"quantity"}, Period: AnyPeriod()}, Validity: always}); err == nil {
				_ = f.RevokeGrant(id, 1)
			}
		}(i)
	}
	wg.Wait()
}
