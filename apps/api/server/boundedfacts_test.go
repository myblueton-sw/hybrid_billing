package server

import (
	"fmt"
	"sort"
	"testing"
	"time"
)

func compareSources(t *testing.T, f world, stage string) {
	t.Helper()
	actor := f.as(AdminLogin).verifier(f)
	whole, err := f.s.accessFacts(actor)
	if err != nil {
		t.Fatalf("%s: %v", stage, err)
	}
	bounded, failure := f.s.boundedFacts(actor)
	people := []string{"nobody", "not a valid id"}
	for user := range whole.members {
		people = append(people, user)
	}
	for user := range whole.users {
		if _, member := whole.members[user]; !member {
			people = append(people, user)
		}
	}
	for _, user := range people {
		wantMember, wantOK := whole.src.member(user)
		gotMember, gotOK := bounded.src.member(user)
		if gotOK != wantOK || fmt.Sprint(gotMember) != fmt.Sprint(wantMember) {
			t.Errorf("%s: member %s: %+v %v want %+v %v", stage, user, gotMember, gotOK, wantMember, wantOK)
		}
		if wantOK && fmt.Sprint(bounded.src.user(user)) != fmt.Sprint(whole.src.user(user)) {
			t.Errorf("%s: user %s: %+v want %+v", stage, user, bounded.src.user(user), whole.src.user(user))
		}
		if whole.person(user) != bounded.person(user) && wantOK {
			t.Errorf("%s: person %s: %s want %s", stage, user, bounded.person(user), whole.person(user))
		}
		if whole.eligible(user) != bounded.eligible(user) || whole.holdsAdmin(user) != bounded.holdsAdmin(user) {
			t.Errorf("%s: %s: eligibility or administrator grant differs", stage, user)
		}
		if fmt.Sprint(bounded.src.grantsOf(user)) != fmt.Sprint(whole.src.grantsOf(user)) {
			t.Errorf("%s: grants of %s differ", stage, user)
		}
		if fmt.Sprint(bounded.src.delegationsOf(user)) != fmt.Sprint(whole.src.delegationsOf(user)) {
			t.Errorf("%s: delegations of %s differ", stage, user)
		}
	}
	if fmt.Sprint(bounded.src.clientGrants()) != fmt.Sprint(whole.src.clientGrants()) || len(whole.src.clientGrants()) == 0 {
		t.Errorf("%s: the console's grants differ or are empty", stage)
	}
	for _, g := range whole.grants {
		want, _ := whole.src.grant(g.ID)
		if got, ok := bounded.src.grant(g.ID); !ok || fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s: grant %s differs", stage, g.ID)
		}
	}
	if _, ok := bounded.src.grant("no-such-grant"); ok {
		t.Errorf("%s: an unknown grant was found", stage)
	}
	holders := map[string]bool{}
	for _, user := range bounded.src.approvers() {
		if holders[user] {
			t.Errorf("%s: %s is a candidate twice", stage, user)
		}
		holders[user] = true
	}
	eligible := []string{}
	for _, user := range people {
		if whole.eligible(user) {
			eligible = append(eligible, user)
			if !holders[user] {
				t.Errorf("%s: %s is eligible and not among the candidates", stage, user)
			}
		}
	}
	if len(eligible) == 0 {
		t.Fatalf("%s: nobody is eligible", stage)
	}
	if *failure != nil {
		t.Errorf("%s: the bounded source failed: %v", stage, *failure)
	}
}

func compareRequestRows(t *testing.T, f world, stage string, callers ...actor) {
	t.Helper()
	for n, a := range callers {
		whole := f.accessRead(a, "")
		want := map[string]map[string]any{}
		for _, raw := range whole.body["requests"].([]any) {
			want[fmt.Sprint(raw.(map[string]any)["id"])] = raw.(map[string]any)
		}
		got := f.readAs(a, requestsPath+"?limit=50")
		if got.status != 200 || len(got.body["rows"].([]any)) != len(want) || len(want) == 0 {
			t.Fatalf("%s, caller %d: %d %d rows, want %d", stage, n, got.status, len(listIDs(got)), len(want))
		}
		for _, raw := range got.body["rows"].([]any) {
			row := raw.(map[string]any)
			keys := []string{}
			for key := range want[fmt.Sprint(row["id"])] {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if fmt.Sprint(row[key]) != fmt.Sprint(want[fmt.Sprint(row["id"])][key]) {
					t.Errorf("%s, caller %d, %v %s: %v want %v", stage, n, row["id"], key, row[key], want[fmt.Sprint(row["id"])][key])
				}
			}
		}
	}
}

func TestBoundedRequestFactsEqualTheWholeFacts(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.must(f.s.identity.Installation().CreateUser(seedOperator, "user-outside", LoginTriple("outside@example.invalid"), "outside@example.invalid"))
	if facts, err := f.s.accessFacts(root.verifier(f)); err != nil || facts.users["user-outside"].ID == "" || facts.members["user-outside"].User != "" {
		t.Fatalf("fixture: a user who is not a member: %v", err)
	}
	compareSources(t, f, "the sample")
	asked := f.requestGrant(root, SampleMember, nil)
	if asked.status != 200 {
		t.Fatalf("ask for a grant %s", asked.raw)
	}
	compareSources(t, f, "a pending grant request")
	compareRequestRows(t, f, "a pending grant request", root, second)
	f.manyPermissions()
	compareSources(t, f, "many grants and delegations")
	compareRequestRows(t, f, "many grants and delegations", root, second)
	approved := f.approveRequest(second, asked)
	if approved.status != 200 {
		t.Fatalf("approve %s", approved.raw)
	}
	if got := f.run(root, "access.request_revoke", fmt.Sprint(approved.body["fields"].(map[string]any)["grant"]), nil); got.status != 200 {
		t.Fatalf("ask to revoke %s", got.raw)
	}
	if got := f.run(root, "access.request_revoke", "q-00", nil); got.status != 200 {
		t.Fatalf("ask to revoke a grant of the second administrator %s", got.raw)
	}
	compareSources(t, f, "pending revocation requests")
	compareRequestRows(t, f, "pending revocation requests", root, second)
	f.must(f.s.admin.RevokeGrant("q-03", 1))
	f.must(f.s.admin.RevokeDelegation("qd-01", 1))
	compareSources(t, f, "a revoked grant and delegation")
	compareRequestRows(t, f, "a revoked grant and delegation", root, second)
	f.advance(3 * time.Hour)
	root, second = f.as(AdminLogin), f.as(Admin2Login)
	compareSources(t, f, "after some grants expired")
	compareRequestRows(t, f, "after some grants expired", root, second)
	if got := f.run(root, "identity.suspend_membership", SampleMember, nil); got.status != 200 {
		t.Fatalf("suspend %s", got.raw)
	}
	compareSources(t, f, "a suspended member")
	compareRequestRows(t, f, "a suspended member", root, second)
}

func TestRequestsReadDoesNotLoadThePopulation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	if got := f.requestGrant(root, SampleMember, nil); got.status != 200 {
		t.Fatalf("ask for a grant %s", got.raw)
	}
	f.manyPermissions()
	f.s.fault = func(at string) bool { return at == "roster" || at == "users" }
	if got := f.accessRead(root, ""); got.status != 503 {
		t.Fatalf("the whole read without the roster and the user pages: %d", got.status)
	}
	if got := f.readAs(root, requestsPath+"?limit=50"); got.status != 200 || len(listIDs(got)) != 1 {
		t.Errorf("the requests read without the roster and the user pages: %d %s", got.status, got.raw[:min(len(got.raw), 160)])
	}
	f.s.fault = func(at string) bool { return at == "people" }
	if got := f.readAs(root, requestsPath+"?limit=50"); got.status != 503 {
		t.Errorf("the requests read without the people lookup: %d", got.status)
	}
	f.s.fault = nil
	scan := f.s.cfg.Access.ServiceQueryScan
	f.s.cfg.Access.ServiceQueryScan = 2
	if got := f.readAs(root, requestsPath+"?limit=50"); got.status != 503 || got.body["code"] != "dependency_unavailable" {
		t.Errorf("a holder with more grants than the lookup limit: %d %s", got.status, got.raw[:min(len(got.raw), 160)])
	}
	f.s.cfg.Access.ServiceQueryScan = scan
	if got := f.readAs(root, requestsPath+"?limit=50"); got.status != 200 {
		t.Errorf("with the limit restored: %d", got.status)
	}
}

func TestEachBoundedLookupRefusesAboveItsLimit(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.manyPermissions()
	actor := f.as(AdminLogin).verifier(f)
	for name, lookup := range map[string]func(factSource){
		"a subject's grants":    func(src factSource) { src.grantsOf(SampleAdmin2) },
		"the console's grants":  func(src factSource) { src.clientGrants() },
		"a user's delegations":  func(src factSource) { src.delegationsOf(SampleAdmin2) },
		"the administrator set": func(src factSource) { src.approvers() },
	} {
		facts, failure := f.s.boundedFacts(actor)
		lookup(facts.src)
		if *failure != nil {
			t.Errorf("%s within the limit: %v", name, *failure)
		}
		facts, failure = f.s.boundedFacts(actor)
		facts.src.(*boundedFacts).limit = 1
		lookup(facts.src)
		if *failure != errFactsBeyondLimit {
			t.Errorf("%s above a limit of one: %v", name, *failure)
		}
	}
}
