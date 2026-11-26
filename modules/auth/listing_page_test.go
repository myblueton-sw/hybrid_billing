package auth

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestPermissionPagesWalkTheWholeListInIdentifierOrder(t *testing.T) {
	f := setup(t)
	for _, n := range []int{7, 2, 9, 0, 5, 3, 8, 1, 6, 4} {
		f.grant(fmt.Sprintf("g-%02d", n), "alice", terms(Issue, One("c1")))
	}
	f.grant("home-bob", "bob", terms(RequestApproval, One("c9")))
	for _, n := range []int{3, 0, 2, 1} {
		if err := f.delegate(fmt.Sprintf("d-%02d", n), "alice", "bob", "", false, terms(Issue, One("c1")), always); err != nil {
			t.Fatal(err)
		}
	}
	whole, wholeDelegations := f.Grants(), f.Delegations()
	if grants, delegations := f.Counts(); grants != len(whole) || delegations != len(wholeDelegations) || grants != 11 || delegations != 4 {
		t.Fatalf("counts %d %d", grants, delegations)
	}
	for _, size := range []int{1, 3, 11, 50} {
		var walked []GrantView
		after, pages := "", 0
		for {
			page, more := f.GrantPage(after, size, *f.now)
			walked = append(walked, page...)
			pages++
			if !more {
				break
			}
			if len(page) != size || pages > 20 {
				t.Fatalf("page size %d: a page of %d said more follow", size, len(page))
			}
			after = page[len(page)-1].ID
		}
		if !reflect.DeepEqual(walked, whole) {
			t.Errorf("page size %d: the pages differ from the whole list", size)
		}
	}
	var walked []DelegationView
	for after := ""; ; {
		page, more := f.DelegationPage(after, 3, *f.now)
		walked = append(walked, page...)
		if !more {
			break
		}
		after = page[len(page)-1].ID
	}
	if !reflect.DeepEqual(walked, wholeDelegations) {
		t.Error("the delegation pages differ from the whole list")
	}
	if page, more := f.GrantPage("g-04x", 2, *f.now); len(page) != 2 || page[0].ID != "g-05" || !more {
		t.Errorf("a cursor between two identifiers: %+v %v", page, more)
	}
	if page, more := f.GrantPage("zzz", 5, *f.now); len(page) != 0 || more {
		t.Errorf("a cursor past the end: %+v %v", page, more)
	}
	for _, limit := range []int{0, -1} {
		if page, more := f.GrantPage("", limit, *f.now); page != nil || more {
			t.Errorf("limit %d: %+v %v", limit, page, more)
		}
		if page, more := f.DelegationPage("", limit, *f.now); page != nil || more {
			t.Errorf("limit %d: %+v %v", limit, page, more)
		}
	}
	if _, known := f.GrantAt("d-00", *f.now); known {
		t.Error("a delegation's identifier answered as a grant")
	}
	if _, known := f.DelegationAt("g-00", *f.now); known {
		t.Error("a grant's identifier answered as a delegation")
	}
	if got, known := f.GrantAt("g-03", *f.now); !known || !reflect.DeepEqual(got, whole[3]) {
		t.Errorf("one grant: %+v %v", got, known)
	}
	got, _ := f.GrantPage("", 1, *f.now)
	got[0].Terms.Fields = append(got[0].Terms.Fields, "tampered")
	if again, _ := f.GrantPage("", 1, *f.now); len(again[0].Terms.Fields) != len(whole[0].Terms.Fields) {
		t.Error("a page shares its terms with the registry")
	}
}

func TestPermissionPagesEvaluateValidityAtTheGivenTime(t *testing.T) {
	f := setup(t)
	from, until := start.Add(time.Hour), start.Add(3*time.Hour)
	window := Validity{NotBefore: from, NotAfter: until}
	if err := f.AddGrant(Grant{ID: "root", SubjectID: "alice", Terms: terms(Issue, One("c1")), Validity: window}); err != nil {
		t.Fatal(err)
	}
	f.grant("home-bob", "bob", terms(RequestApproval, One("c9")))
	*f.now = from
	if err := f.delegate("del", "alice", "bob", "", false, terms(Issue, One("c1")), Validity{NotBefore: from, NotAfter: start.Add(2 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	*f.now = start.Add(10 * time.Hour)
	for name, c := range map[string]struct {
		at                     time.Time
		grant, delegation, eff bool
	}{
		"before both begin":                  {start, false, false, false},
		"at the start":                       {from, true, true, true},
		"just before the delegation ends":    {start.Add(2*time.Hour - time.Nanosecond), true, true, true},
		"when the delegation has ended":      {start.Add(2 * time.Hour), true, false, false},
		"when the grant has ended":           {until, false, false, false},
		"at the registry's own current time": {start.Add(10 * time.Hour), false, false, false},
	} {
		g, known := f.GrantAt("root", c.at)
		d, also := f.DelegationAt("del", c.at)
		if !known || !also || g.Active != c.grant || g.Effective != c.grant || d.Active != c.delegation || d.Effective != c.eff {
			t.Errorf("%s: grant %v/%v delegation %v/%v", name, g.Active, g.Effective, d.Active, d.Effective)
		}
		page, _ := f.GrantPage("home-bob", 5, c.at)
		delegations, _ := f.DelegationPage("", 5, c.at)
		if len(page) != 1 || !reflect.DeepEqual(page[0], g) || len(delegations) != 1 || !reflect.DeepEqual(delegations[0], d) {
			t.Errorf("%s: the page and the single read differ", name)
		}
	}
	if g := f.Grants(); g[1].Active {
		t.Error("the whole list is evaluated at the registry's time, where the grant has ended")
	}
	generation := f.Generation()
	if _, err := f.PutSubject(Subject{ID: "bob", Kind: Human, State: Suspended, Epoch: 1}, 1); err != nil {
		t.Fatal(err)
	}
	if f.Generation() == generation {
		t.Fatal("a change of a person's standing did not advance the generation")
	}
	if d, _ := f.DelegationAt("del", from); d.Effective {
		t.Error("a delegation to a suspended person is effective at an earlier reference time; standing is read as stored")
	}
}

func TestPermissionPagesKeepClosedRowsAndFollowAChainToItsRoot(t *testing.T) {
	f := setup(t)
	f.grant("root", "alice", terms(Issue, One("c1")))
	f.grant("gone", "alice", terms(Issue, One("c2")))
	for _, id := range []string{"bob", "carol"} {
		f.grant("home-"+id, id, terms(RequestApproval, One("c9")))
	}
	if err := f.RevokeGrant("gone", 1); err != nil {
		t.Fatal(err)
	}
	if err := f.delegate("first", "alice", "bob", "", true, terms(Issue, One("c1")), always); err != nil {
		t.Fatal(err)
	}
	if err := f.delegate("second", "bob", "carol", "first", false, terms(Issue, One("c1")), always); err != nil {
		t.Fatal(err)
	}
	page, _ := f.GrantPage("", 10, *f.now)
	var closed GrantView
	for _, g := range page {
		if g.ID == "gone" {
			closed = g
		}
	}
	if len(page) != 4 || !closed.Closed || closed.Active || closed.Effective || closed.Version != 2 {
		t.Fatalf("a revoked grant stays in the pages as closed: %+v", closed)
	}
	if child, known := f.DelegationAt("second", *f.now); !known || !child.Effective || child.ParentID != "first" {
		t.Fatalf("a delegation through a parent: %+v", child)
	}
	if err := f.RevokeDelegation("first", 1); err != nil {
		t.Fatal(err)
	}
	chain, _ := f.DelegationPage("", 10, *f.now)
	if len(chain) != 2 || !chain[0].Closed || chain[0].Effective || chain[1].Closed || !chain[1].Active || chain[1].Effective {
		t.Fatalf("closing the parent leaves the child listed, active by its own dates and not effective: %+v", chain)
	}
	if !reflect.DeepEqual(chain, f.Delegations()) {
		t.Error("the page differs from the whole list after a revocation")
	}
}
