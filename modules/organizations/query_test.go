package organizations

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

func TestContractPartiesAreExactHistoricalReferencesWithoutComponentPopulation(t *testing.T) {
	r := seeded(t, ProfileMSP)
	components := make([]Component, 513)
	for i := range components {
		components[i] = Component{ID: fmt.Sprintf("component-%04d", i), Kind: ProviderCharge, ProviderPaymentResponsibleID: "customer-a"}
	}
	revision := revision(Direct, "seller", components...)
	if err := r.AppendContractRevision(revision); err != nil {
		t.Fatal(err)
	}
	refs, err := r.ContractParties("w1", "c1", 1)
	if err != nil || refs.CustomerID != "customer-a" || refs.SellerID != "seller" {
		t.Fatal("narrow contract identity mismatch", refs, err)
	}
	full, err := r.ContractRevision("w1", "c1", 1)
	if err != nil || len(full.Components) != 513 {
		t.Fatal("narrow read changed original components", err)
	}
	party, version, err := r.Party("w1", "seller")
	if err != nil {
		t.Fatal(err)
	}
	party.State = Suspended
	if _, err = r.PutParty(party, version); err != nil {
		t.Fatal(err)
	}
	historical, err := r.ContractParties("w1", "c1", 1)
	if err != nil || historical != refs {
		t.Fatal("historical parties were confused with current eligibility", historical, err)
	}
	for _, args := range []struct {
		workspace, id string
		revision      uint64
	}{{"w2", "c1", 1}, {"w1", "missing", 1}, {"w1", "c1", 0}, {"w1", "c1", 2}} {
		if _, err := r.ContractParties(args.workspace, args.id, args.revision); !errors.Is(err, ErrNotFound) {
			t.Fatal("foreign/missing revision disclosed", args, err)
		}
	}
	var absent *Registry
	if _, err := absent.ContractParties("w1", "c1", 1); !errors.Is(err, ErrInvalid) {
		t.Fatal("nil contract parties", err)
	}
}

func TestOrganizationQueryVersionTracksEverySuccessfulWriterAndWorkspace(t *testing.T) {
	r := seeded(t, ProfileMSP)
	other, _ := r.QueryVersion("w2")
	requireAdvance := func(write func() error) {
		t.Helper()
		before, err := r.QueryVersion("w1")
		if err != nil {
			t.Fatal(err)
		}
		if err = write(); err != nil {
			t.Fatal(err)
		}
		after, err := r.QueryVersion("w1")
		if err != nil || after != before+1 {
			t.Fatal("successful organization writer omitted query fence", before, after, err)
		}
	}
	requireAdvance(func() error { w, version, _ := r.Workspace("w1"); _, err := r.PutWorkspace(w, version); return err })
	requireAdvance(func() error { p, version, _ := r.Party("w1", "agent"); _, err := r.PutParty(p, version); return err })
	requireAdvance(func() error {
		_, err := r.PutBillingProfile(external("invoice", "seller", "customer-a", "customer-a", "USD"), 0)
		return err
	})
	requireAdvance(func() error {
		profile, version, _ := r.BillingProfile("w1", "invoice")
		profile.State = Suspended
		_, err := r.PutBillingProfile(profile, version)
		return err
	})
	requireAdvance(func() error {
		return r.AppendContractRevision(revision(Direct, "seller", Component{ID: "usage", Kind: ProviderCharge, ProviderPaymentResponsibleID: "customer-a"}))
	})
	otherAfter, err := r.QueryVersion("w2")
	if err != nil || other != otherAfter {
		t.Fatal("foreign workspace fence advanced", err)
	}
	before, _ := r.QueryVersion("w1")
	p, _, _ := r.Party("w1", "seller")
	if _, err := r.PutParty(p, 0); !errors.Is(err, ErrConflict) {
		t.Fatal("failed-write premise", err)
	}
	after, _ := r.QueryVersion("w1")
	if before != after {
		t.Fatal("rejected organization write changed query source")
	}
	if _, err := r.QueryVersion("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal("unknown workspace query basis", err)
	}
	if _, err := r.QueryVersion("bad workspace"); !errors.Is(err, ErrInvalid) {
		t.Fatal("malformed workspace query basis", err)
	}
	var absent *Registry
	if _, err := absent.QueryVersion("w1"); !errors.Is(err, ErrInvalid) {
		t.Fatal("nil organization query basis", err)
	}
}

func TestOrganizationQueryVersionExhaustionDoesNotChangeDomainPolicy(t *testing.T) {
	r := seeded(t, ProfileMSP)
	r.mu.Lock()
	r.queryVersions["w1"] = math.MaxUint64 - 1
	r.mu.Unlock()
	for i := 0; i < 2; i++ {
		p, version, _ := r.Party("w1", "seller")
		if _, err := r.PutParty(p, version); err != nil {
			t.Fatal("query exhaustion blocked a valid domain write", err)
		}
		if _, err := r.QueryVersion("w1"); !errors.Is(err, ErrQueryUnavailable) {
			t.Fatal("exhausted organization fence wrapped/reused", err)
		}
	}
}
