package organizations

import (
	"errors"
	"testing"
)

func TestListingsStayInTheirWorkspace(t *testing.T) {
	r := seeded(t, ProfileMSP)
	if _, err := r.PutBillingProfile(external("p-b", "seller", "customer-b", "customer-b", "USD"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PutBillingProfile(external("p-a", "seller", "customer-a", "customer-a", "USD"), 0); err != nil {
		t.Fatal(err)
	}
	foreign := external("p-z", "foreign", "foreign", "foreign", "USD")
	foreign.WorkspaceID = "w2"
	foreign.PayerID = ""
	foreign.Path = InternalChargeback
	if _, err := r.PutParty(Party{ID: "foreign-unit", WorkspaceID: "w2", Kind: InternalUnit, ParentID: "foreign", State: Active}, 0); err != nil {
		t.Fatal(err)
	}
	foreign.BuyerID = "foreign-unit"
	if _, err := r.PutBillingProfile(foreign, 0); err != nil {
		t.Fatal(err)
	}
	parties, err := r.Parties("w1")
	ids := ""
	for _, p := range parties {
		ids += p.ID + ","
	}
	if err != nil || ids != "agent,customer-a,customer-b,seller,unit-1,unit-2,unit-b," {
		t.Errorf("parties %s %v", ids, err)
	}
	parties[0].ID = "changed"
	if again, _ := r.Parties("w1"); again[0].ID != "agent" {
		t.Error("parties are shared with the registry")
	}
	profiles, err := r.BillingProfiles("w1")
	if err != nil || len(profiles) != 2 || profiles[0].ID != "p-a" || profiles[1].ID != "p-b" {
		t.Errorf("profiles %+v %v", profiles, err)
	}
	other, _ := r.Parties("w2")
	otherProfiles, _ := r.BillingProfiles("w2")
	if len(other) != 2 || other[0].ID != "foreign" || len(otherProfiles) != 1 || otherProfiles[0].ID != "p-z" {
		t.Errorf("w2 %+v %+v", other, otherProfiles)
	}
	if _, err := r.Parties("w9"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown workspace parties %v", err)
	}
	if _, err := r.BillingProfiles("w9"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown workspace profiles %v", err)
	}
	empty, _ := NewRegistry("i")
	empty.PutWorkspace(Workspace{ID: "e", Profile: ProfileMSP, State: Active}, 0)
	if p, err := empty.Parties("e"); err != nil || p == nil || len(p) != 0 {
		t.Errorf("empty parties %v %v", p, err)
	}
	if p, err := empty.BillingProfiles("e"); err != nil || p == nil || len(p) != 0 {
		t.Errorf("empty profiles %v %v", p, err)
	}
}
