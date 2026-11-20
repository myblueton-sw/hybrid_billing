package organizations

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
)

func TestEligiblePartyRechecksWorkspacePartyAndParent(t *testing.T) {
	r := seeded(t, ProfileMSP)
	party, version, err := r.EligibleParty("w1", "unit-1")
	if err != nil || party.ID != "unit-1" || version != 1 {
		t.Fatalf("eligible internal organization: %+v %d %v", party, version, err)
	}
	parent, parentVersion, _ := r.Party("w1", "customer-a")
	parent.State = Suspended
	if _, err := r.PutParty(parent, parentVersion); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.EligibleParty("w1", "unit-1"); !errors.Is(err, ErrSuspended) {
		t.Fatal("parent suspension did not make child organization ineligible", err)
	}
	historical, unchanged, err := r.Party("w1", "unit-1")
	if err != nil || historical.State != Active || unchanged != version {
		t.Fatal("eligibility was confused with historical party state/version")
	}
	parent.State = Active
	if _, err := r.PutParty(parent, parentVersion+1); err != nil {
		t.Fatal(err)
	}
	if _, ownVersion, err := r.EligibleParty("w1", "unit-1"); err != nil || ownVersion != version {
		t.Fatal("restoration failed or own version became a parent eligibility snapshot", err)
	}
	party.State = Suspended
	if _, err := r.PutParty(party, version); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.EligibleParty("w1", "unit-1"); !errors.Is(err, ErrSuspended) {
		t.Fatal("suspended party remained eligible", err)
	}
	workspace, workspaceVersion, _ := r.Workspace("w1")
	workspace.State = Suspended
	if _, err := r.PutWorkspace(workspace, workspaceVersion); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.EligibleParty("w1", "seller"); !errors.Is(err, ErrSuspended) {
		t.Fatal("suspended workspace remained eligible", err)
	}
	rows, err := r.PartyPage("w1", "", 3)
	if err != nil || len(rows) != 3 {
		t.Fatal("workspace suspension destroyed historical paging", err)
	}
}

func TestEligiblePartyRejectsForeignMissingAndInvalidReferences(t *testing.T) {
	r := seeded(t, ProfileMSP)
	for _, args := range []struct{ workspace, id string }{{"w1", "foreign"}, {"w2", "seller"}, {"missing", "seller"}, {"w1", "missing"}} {
		if _, _, err := r.EligibleParty(args.workspace, args.id); !errors.Is(err, ErrNotFound) {
			t.Fatal("foreign/missing organization disclosed or admitted", args, err)
		}
	}
	for _, args := range []struct{ workspace, id string }{{"", "seller"}, {"w1", ""}, {"bad workspace", "seller"}, {"w1", "bad id"}} {
		if _, _, err := r.EligibleParty(args.workspace, args.id); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid organization reference admitted", args, err)
		}
	}
	if party, version, err := r.EligibleParty("w1", "seller"); err != nil || party.Kind != LegalEntity || version != 1 {
		t.Fatal("eligible legal entity refused", party, version, err)
	}
}

func assertPartyPageIndex(t *testing.T, r *Registry) {
	t.Helper()
	r.mu.RLock()
	defer r.mu.RUnlock()
	oracle := map[string][]string{}
	for key := range r.parties {
		oracle[key.workspace] = append(oracle[key.workspace], key.id)
	}
	for key := range oracle {
		sort.Strings(oracle[key])
	}
	if !reflect.DeepEqual(oracle, r.partyIndex) {
		t.Fatalf("party index differs from actual workspace records: %v / %v", r.partyIndex, oracle)
	}
}

func TestPartyPagesMatchIndependentMapAcrossUpdatesAndAbsentBoundaries(t *testing.T) {
	r := seeded(t, ProfileMSP)
	for i := 0; i < 401; i++ {
		id := fmt.Sprintf("p-%03d", (i*97)%401)
		if _, err := r.PutParty(Party{ID: id, WorkspaceID: "w1", Kind: LegalEntity, State: Active}, 0); err != nil {
			t.Fatal(err)
		}
	}
	party, version, _ := r.Party("w1", "p-123")
	party.State = Suspended
	if _, err := r.PutParty(party, version); err != nil {
		t.Fatal(err)
	}
	party.State = Active
	if _, err := r.PutParty(party, version+1); err != nil {
		t.Fatal(err)
	}
	for _, after := range []string{"", "p-123", "p-1235", "p-398", "zzz"} {
		ids := []string{}
		for key := range r.parties {
			if key.workspace == "w1" && key.id > after {
				ids = append(ids, key.id)
			}
		}
		sort.Strings(ids)
		want := []Party{}
		for _, id := range ids[:min(3, len(ids))] {
			want = append(want, r.parties[scoped{"w1", id}].value)
		}
		got, err := r.PartyPage("w1", after, 3)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("page after %q: got %+v want %+v, %v", after, got, want, err)
		}
	}
	rows, err := r.PartyPage("w1", "", 1000)
	if err != nil || len(rows) != 408 {
		t.Fatal("maximum page failed to exhaust own recorded parties", len(rows), err)
	}
	rows[0].State = Suspended
	stored, _, _ := r.Party("w1", rows[0].ID)
	if stored.State != Active {
		t.Fatal("caller modified stored party through returned page")
	}
	foreign, err := r.PartyPage("w2", "", 3)
	if err != nil || len(foreign) != 1 || foreign[0].ID != "foreign" {
		t.Fatal("workspace pages crossed records", foreign, err)
	}
	first, version, _ := r.Party("w1", "agent")
	first.State = Suspended
	if _, err := r.PutParty(first, version); err != nil {
		t.Fatal(err)
	}
	page, err := r.PartyPage("w1", "", 1)
	if err != nil || len(page) != 1 || page[0].ID != "agent" || page[0].State != Suspended {
		t.Fatal("historical page hid suspended party", page, err)
	}
	assertPartyPageIndex(t, r)
}

func TestPartyPagesInputFailuresAndFailedWritesPreserveIndex(t *testing.T) {
	r := seeded(t, ProfileMSP)
	for _, args := range []struct {
		workspace, after string
		limit            int
	}{{"w1", "", 0}, {"w1", "", 1001}, {"w1", "bad id", 1}, {"", "", 1}, {"bad workspace", "", 1}} {
		if _, err := r.PartyPage(args.workspace, args.after, args.limit); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid party page admitted", args, err)
		}
	}
	if _, err := r.PartyPage("missing", "", 1); !errors.Is(err, ErrNotFound) {
		t.Fatal("unknown workspace returned successful party page", err)
	}
	if _, err := r.PutWorkspace(Workspace{ID: "empty", Profile: ProfileMSP, State: Active}, 0); err != nil {
		t.Fatal(err)
	}
	if rows, err := r.PartyPage("empty", "", 1); err != nil || rows == nil || len(rows) != 0 {
		t.Fatal("known empty workspace page was not explicitly empty", rows, err)
	}
	for _, party := range []Party{{ID: "new-child", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "missing", State: Active}, {ID: "seller", WorkspaceID: "w1", Kind: LegalEntity, State: Active}, {ID: "ghost", WorkspaceID: "missing", Kind: LegalEntity, State: Active}} {
		if _, err := r.PutParty(party, 0); err == nil {
			t.Fatal("invalid new record or stale write succeeded", party.ID)
		}
	}
	assertPartyPageIndex(t, r)
}

func TestPartyPagesConcurrentUpdatesAndReadsKeepMembership(t *testing.T) {
	r := seeded(t, ProfileMSP)
	var readers sync.WaitGroup
	for i := 0; i < 3; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for n := 0; n < 50; n++ {
				_, _ = r.PartyPage("w1", "p-020", 2)
				_, _, _ = r.EligibleParty("w1", "seller")
			}
		}()
	}
	for i := 0; i < 50; i++ {
		if _, err := r.PutParty(Party{ID: fmt.Sprintf("p-%03d", i), WorkspaceID: "w1", Kind: LegalEntity, State: Active}, 0); err != nil {
			t.Fatal(err)
		}
	}
	readers.Wait()
	assertPartyPageIndex(t, r)
}
