package server

import (
	"fmt"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"net/url"
	"strings"
	"testing"
	"time"
)

func (f world) subjects(a actor, query string) reply {
	f.t.Helper()
	return f.readAs(a, "/api/v1/identity/access/subjects"+query)
}

func subjectIDs(got reply) []string {
	out := []string{}
	for _, raw := range got.body["rows"].([]any) {
		out = append(out, fmt.Sprint(raw.(map[string]any)["user_id"]))
	}
	return out
}

func TestAccessSubjectsArePagedAndEqualTheWholeReadsSubjects(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	want := []string{}
	logins := map[string]string{}
	for _, raw := range f.accessRead(root, "").body["subjects"].([]any) {
		row := raw.(map[string]any)
		want = append(want, fmt.Sprint(row["user_id"]))
		logins[fmt.Sprint(row["user_id"])] = fmt.Sprint(row["login"])
	}
	if len(want) < 3 {
		t.Fatalf("the sample has %d subjects", len(want))
	}
	all := f.subjects(root, "?limit=50")
	if all.status != 200 || strings.Join(subjectIDs(all), " ") != strings.Join(want, " ") || all.body["order"] != subjectsOrder || all.body["next"] != "" ||
		all.body["count"].(map[string]any)["certainty"] != "exact" || all.body["count"].(map[string]any)["value"] != fmt.Sprint(len(want)) {
		t.Fatalf("one page: %v want %v", subjectIDs(all), want)
	}
	for _, raw := range all.body["rows"].([]any) {
		if row := raw.(map[string]any); row["login"] != logins[fmt.Sprint(row["user_id"])] || len(row) != 3 || row["revocation_unconfirmed"] != false {
			t.Errorf("row %+v", row)
		}
	}
	walked := []string{}
	next := ""
	for pages := 0; pages < 50; pages++ {
		path := "?limit=1"
		if next != "" {
			path += "&after=" + url.QueryEscape(next)
		}
		got := f.subjects(root, path)
		if got.status != 200 {
			t.Fatalf("page %d: %s", pages, got.raw)
		}
		walked = append(walked, subjectIDs(got)...)
		if next, _ = got.body["next"].(string); next == "" {
			break
		}
	}
	if strings.Join(walked, " ") != strings.Join(want, " ") {
		t.Errorf("one at a time: %v want %v", walked, want)
	}
	first := want[0]
	if got := f.subjects(root, "?id="+first); len(subjectIDs(got)) != 1 || subjectIDs(got)[0] != first || got.body["count"].(map[string]any)["value"] != "1" {
		t.Errorf("one person by identifier: %s", got.raw)
	}
	budget := f.s.cfg.Access.ServiceQueryScan
	f.s.cfg.Access.ServiceQueryScan = 1
	last := want[len(want)-1]
	if got := f.subjects(root, "?id="+last); len(subjectIDs(got)) != 1 || subjectIDs(got)[0] != last || got.body["next"] != "" || got.body["count"].(map[string]any)["certainty"] != "exact" {
		t.Errorf("one person by identifier does not depend on the scan budget or on where the person sorts: %s", got.raw)
	}
	if got := f.subjects(root, "?limit=50"); len(subjectIDs(got)) != 1 || got.body["next"] == "" || got.body["scan_stopped"] != true {
		t.Errorf("a list with a budget of one stops after one and can be continued: %s", got.raw)
	}
	f.s.cfg.Access.ServiceQueryScan = budget
	if got := f.subjects(root, "?id=nobody-here"); got.status != 200 || len(subjectIDs(got)) != 0 {
		t.Errorf("an identifier that names nobody: %s", got.raw)
	}
	term := strings.ToUpper(logins[first][:4])
	expect := []string{}
	for _, id := range want {
		if strings.Contains(strings.ToLower(logins[id]), strings.ToLower(term)) || strings.Contains(strings.ToLower(id), strings.ToLower(term)) {
			expect = append(expect, id)
		}
	}
	if got := f.subjects(root, "?limit=50&q="+term); strings.Join(subjectIDs(got), " ") != strings.Join(expect, " ") || got.body["search_input"] != term {
		t.Errorf("search %q: %v want %v", term, subjectIDs(got), expect)
	}
	if got := f.run(root, "identity.suspend_membership", SampleMember, nil); got.status != 200 {
		t.Fatalf("suspend %s", got.raw)
	}
	after := f.subjects(root, "?limit=50")
	for _, id := range subjectIDs(after) {
		if id == SampleMember {
			t.Error("a person whose membership is suspended is still offered")
		}
	}
	if strings.Join(subjectIDs(after), " ") != strings.Join(func() []string {
		out := []string{}
		for _, raw := range f.accessRead(root, "").body["subjects"].([]any) {
			out = append(out, fmt.Sprint(raw.(map[string]any)["user_id"]))
		}
		return out
	}(), " ") {
		t.Error("after a suspension the read and the whole read disagree")
	}
}

func TestAccessSubjectsRefusalsAdmissionAndCursor(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	for _, query := range []string{"?unknown=1", "?q=", "?q=" + strings.Repeat("a", permissionSearchBytes+1), "?q=%01", "?id=a%20b", "?id=", "?limit=0", "?limit=x", fmt.Sprint("?limit=", f.s.cfg.Access.ServiceQueryPage+1),
		"?id=someone&q=a", "?id=someone&after=a.b", "?after=", "?q=a&q=b"} {
		if got := f.subjects(root, query); got.status != 422 {
			t.Errorf("%q: %d", query, got.status)
		}
	}
	f.code(t, "a member", f.subjects(f.as(MemberLogin), ""), 403, "permission_denied")
	f.code(t, "no session", f.subjects(actor{}, ""), 401, "unauthenticated")
	first := f.subjects(root, "?limit=1")
	token := url.QueryEscape(first.body["next"].(string))
	conflict := func(name string, got reply) {
		t.Helper()
		if got.status != 409 || got.body["code"] != "revision_conflict" {
			t.Errorf("%s: %d %s", name, got.status, got.raw[:min(len(got.raw), 140)])
		}
	}
	if got := f.subjects(root, "?limit=1&after="+token); got.status != 200 {
		t.Fatalf("the cursor with its own query: %s", got.raw)
	}
	conflict("another search", f.subjects(root, "?limit=1&q=a&after="+token))
	conflict("another page size", f.subjects(root, "?limit=2&after="+token))
	conflict("another administrator", f.subjects(second, "?limit=1&after="+token))
	conflict("a changed token", f.subjects(root, "?limit=1&after="+token[:len(token)-2]+"xx"))
	permission := f.permissions(root, "?limit=1")
	conflict("a cursor of the permissions list", f.subjects(root, "?limit=1&after="+url.QueryEscape(permission.body["next"].(string))))
	if got := f.run(root, "identity.suspend_membership", SampleMember, nil); got.status != 200 {
		t.Fatalf("suspend %s", got.raw)
	}
	conflict("after a membership changed", f.subjects(root, "?limit=1&after="+token))
	fresh := f.subjects(root, "?limit=1")
	f.advance(time.Duration(f.s.cfg.Access.ServiceQuerySeconds+1) * time.Second)
	conflict("after the cursor's lifetime", f.subjects(f.as(AdminLogin), "?limit=1&after="+url.QueryEscape(fresh.body["next"].(string))))
	f.s.fault = func(at string) bool { return at == "roster" }
	if got := f.subjects(f.as(AdminLogin), ""); got.status != 503 {
		t.Errorf("a fault: %d", got.status)
	}
}

func browserUnconfirmed(whole reply) map[string]bool {
	holder := map[string]string{}
	for _, raw := range whole.body["grants"].([]any) {
		holder["grant/"+fmt.Sprint(raw.(map[string]any)["id"])] = fmt.Sprint(raw.(map[string]any)["subject"])
	}
	for _, raw := range whole.body["delegations"].([]any) {
		holder["delegation/"+fmt.Sprint(raw.(map[string]any)["id"])] = fmt.Sprint(raw.(map[string]any)["delegate"])
	}
	out := map[string]bool{}
	for _, raw := range whole.body["revocations"].([]any) {
		row := raw.(map[string]any)
		if who := holder[fmt.Sprint(row["target_kind"])+"/"+fmt.Sprint(row["target"])]; who != "" && (row["state"] == "received" || row["state"] == "decision_unconfirmed") {
			out[who] = true
		}
	}
	return out
}

func TestAccessSubjectsFlagAnUnconfirmedRevocationAsTheBrowserDid(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	confirmed := f.grantVia(root, second, SampleMember, nil)
	if got := f.revokeVia(root, second, confirmed); got.status != 202 {
		t.Fatalf("revoke %s", got.raw)
	}
	compare := func(stage string, want ...string) {
		t.Helper()
		whole := f.accessRead(root, "")
		rule := browserUnconfirmed(whole)
		flagged := []string{}
		for _, raw := range f.subjects(root, "?limit=50").body["rows"].([]any) {
			row := raw.(map[string]any)
			user := fmt.Sprint(row["user_id"])
			if row["revocation_unconfirmed"] != rule[user] {
				t.Errorf("%s: %s: %v, the browser's rule says %v", stage, user, row["revocation_unconfirmed"], rule[user])
			}
			if row["revocation_unconfirmed"] == true {
				flagged = append(flagged, user)
			}
			one := f.subjects(root, "?id="+user).body["rows"].([]any)[0].(map[string]any)
			if one["revocation_unconfirmed"] != row["revocation_unconfirmed"] {
				t.Errorf("%s: %s: the lookup by identifier disagrees with the list", stage, user)
			}
		}
		if strings.Join(flagged, " ") != strings.Join(want, " ") {
			t.Errorf("%s: flagged %v want %v", stage, flagged, want)
		}
	}
	compare("a confirmed revocation only")
	token := url.QueryEscape(f.subjects(root, "?limit=1").body["next"].(string))
	f.stoppedLedger()
	if got := f.revokeVia(root, second, "reader-"+SampleAdmin2); got.status != 202 {
		t.Fatalf("revoke on a stopped ledger %s", got.raw)
	}
	if got := f.subjects(root, "?limit=1&after="+token); got.status != 409 {
		t.Errorf("a cursor from before the revocation: %d", got.status)
	}
	compare("a grant's revocation left unconfirmed", SampleAdmin2)
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "member-holds", SubjectID: SampleMember, Terms: permissionTerms(gate.One("c1")),
		Validity: gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}}))
	addDelegation(f, "del-u", SampleAdmin, SampleMember, "")
	if got := f.run(root, "access.revoke_delegation", "del-u", nil); got.status != 202 {
		t.Fatalf("revoke a delegation on a stopped ledger %s", got.raw)
	}
	compare("a delegation's revocation left unconfirmed", SampleAdmin2, SampleMember)
	scan := f.s.cfg.Access.ServiceQueryScan
	f.s.cfg.Access.ServiceQueryScan = 1
	if got := f.subjects(root, "?id="+SampleAdmin); got.status != 503 {
		t.Errorf("more unconfirmed revocations than the limit: %d", got.status)
	}
	f.s.cfg.Access.ServiceQueryScan = scan
}
