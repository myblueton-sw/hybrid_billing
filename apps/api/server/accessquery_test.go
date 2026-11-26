package server

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func permissionIDs(rows []any) []string {
	out := []string{}
	for _, raw := range rows {
		row := raw.(map[string]any)
		out = append(out, fmt.Sprint(row["source"], "/", row["id"]))
	}
	return out
}

func (f world) permissions(a actor, query string) reply {
	f.t.Helper()
	return f.readAs(a, "/api/v1/identity/access/permissions"+query)
}

func (f world) walkPermissions(a actor, query string) ([]string, reply) {
	f.t.Helper()
	ids := []string{}
	next := ""
	var last reply
	for pages := 0; ; pages++ {
		path := query
		if next != "" {
			path += map[bool]string{true: "?", false: "&"}[query == ""] + "after=" + url.QueryEscape(next)
		}
		last = f.permissions(a, path)
		if last.status != 200 || pages > 200 {
			f.t.Fatalf("page %d of %q: %d %s", pages, query, last.status, last.raw)
		}
		ids = append(ids, permissionIDs(last.body["rows"].([]any))...)
		next, _ = last.body["next"].(string)
		if next == "" {
			return ids, last
		}
	}
}

func permissionTerms(customer gate.Selector) gate.Terms {
	return gate.Terms{Action: gate.View, Kind: ReaderKind, Fields: []string{ReaderField}, Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: customer, LegalEntity: gate.All(), Contract: gate.All()}}
}

func (f world) manyPermissions() (int, int) {
	f.t.Helper()
	now := f.s.clock()
	for n := 0; n < 23; n++ {
		customer := gate.All()
		if n%3 == 0 {
			customer = gate.One("c1")
		}
		f.must(f.s.admin.AddGrant(gate.Grant{ID: fmt.Sprintf("q-%02d", n), SubjectID: SampleAdmin2, Terms: permissionTerms(customer),
			Validity: gate.Validity{NotBefore: now, NotAfter: now.Add(time.Duration(n+1) * time.Hour)}}))
	}
	for n := 0; n < 4; n++ {
		f.must(f.s.admin.AddDelegation(gate.Delegation{ID: fmt.Sprintf("qd-%02d", n), DelegatorID: SampleAdmin, DelegateID: SampleAdmin2, Terms: permissionTerms(gate.All()),
			Validity: gate.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}}))
	}
	shown := 0
	for _, g := range f.s.admin.Grants() {
		if !gate.InstallationAction(g.Terms.Action) {
			shown++
		}
	}
	return shown, len(f.s.admin.Delegations())
}

func TestPermissionsArePagedInOneStableOrderAndEqualTheWholeRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	grants, delegations := f.manyPermissions()
	whole := f.accessRead(root, "")
	want := []string{}
	for _, raw := range whole.body["grants"].([]any) {
		want = append(want, "grant/"+fmt.Sprint(raw.(map[string]any)["id"]))
	}
	for _, raw := range whole.body["delegations"].([]any) {
		want = append(want, "delegation/"+fmt.Sprint(raw.(map[string]any)["id"]))
	}
	if len(want) != grants+delegations || delegations < 4 {
		t.Fatalf("the whole read has %d rows, the registry %d and %d", len(want), grants, delegations)
	}
	for _, limit := range []string{"1", "7", "50"} {
		got, last := f.walkPermissions(root, "?limit="+limit)
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("limit %s: pages %v\nwhole %v", limit, got, want)
		}
		count := last.body["count"].(map[string]any)
		if count["certainty"] != "exact" || count["value"] != fmt.Sprint(len(want)) || last.body["order"] != permissionsOrder {
			t.Errorf("limit %s: the last page's count %+v", limit, count)
		}
	}
	first := f.permissions(root, "?limit=5")
	if count := first.body["count"].(map[string]any); count["certainty"] != "lower_bound" || count["value"] != "5" || first.body["next"] == "" || first.body["reference_at"] == "" {
		t.Errorf("a page that is not the last: %s", first.raw[:min(len(first.raw), 300)])
	}
	rowOf := func(id string, rows []any) map[string]any {
		for _, raw := range rows {
			if row := raw.(map[string]any); row["id"] == id {
				return row
			}
		}
		return nil
	}
	all := f.permissions(root, "?q=q&limit=50")
	grant, old := rowOf("q-03", all.body["rows"].([]any)), rowOf("q-03", whole.body["grants"].([]any))
	for _, key := range []string{"id", "subject", "client", "action", "kind", "not_before", "not_after", "state", "effective", "version"} {
		if fmt.Sprint(grant[key]) != fmt.Sprint(old[key]) {
			t.Errorf("grant field %s: %v in the page, %v in the whole read", key, grant[key], old[key])
		}
	}
	delegation, before := rowOf("qd-01", all.body["rows"].([]any)), rowOf("qd-01", whole.body["delegations"].([]any))
	if delegation["subject"] != before["delegate"] || delegation["delegator"] != before["delegator"] || delegation["state"] != before["state"] || delegation["source"] != sourceDelegation {
		t.Errorf("delegation row %+v against %+v", delegation, before)
	}
	for name, c := range map[string]struct {
		query string
		keep  func(string) bool
	}{
		"source grant":      {"?source=grant", func(id string) bool { return strings.HasPrefix(id, "grant/") }},
		"source delegation": {"?source=delegation", func(id string) bool { return strings.HasPrefix(id, "delegation/") }},
		"search":            {"?q=Q-1", func(id string) bool { return strings.Contains(id, "q-1") }},
		"search and source": {"?q=qd&source=delegation", func(id string) bool { return strings.Contains(id, "qd-") }},
	} {
		expect := []string{}
		for _, id := range want {
			if c.keep(id) {
				expect = append(expect, id)
			}
		}
		if got, _ := f.walkPermissions(root, c.query+"&limit=4"); strings.Join(got, " ") != strings.Join(expect, " ") || len(expect) == 0 {
			t.Errorf("%s: %v want %v", name, got, expect)
		}
	}
	for name, c := range map[string]struct {
		query string
		count int
	}{
		"customer c1": {"?customer=c1", 8}, "the second administrator's delegations": {"?q=qd&subject=" + SampleAdmin2 + "&source=delegation", 4},
		"ending within five hours and a half": {"?q=q-&expires_before=" + url.QueryEscape(f.s.clock().Add(5*time.Hour+30*time.Minute).UTC().Format(time.RFC3339)), 5},
		"no such subject":                     {"?subject=nobody", 0}, "no such action": {"?action=nothing", 0}, "state expired": {"?q=q-&state=expired", 0}, "state active": {"?q=q-&state=active&source=grant", 23},
	} {
		got, last := f.walkPermissions(root, c.query)
		if len(got) != c.count || last.body["count"].(map[string]any)["value"] != fmt.Sprint(c.count) {
			t.Errorf("%s: %d rows %v", name, len(got), got)
		}
	}
}

func TestPermissionsQueryRefusesWhatItDoesNotUnderstandAndWhoMayNotRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	f.manyPermissions()
	long := strings.Repeat("a", permissionSearchBytes+1)
	for _, query := range []string{"?unknown=1", "?q=", "?q=" + long, "?q=%01", "?limit=0", "?limit=01", "?limit=x", fmt.Sprint("?limit=", f.s.cfg.Access.ServiceQueryPage+1), "?source=other", "?state=closed",
		"?subject=a%20b", "?action=", "?customer=a/b", "?legal_entity=%00", "?expires_before=tomorrow", "?expires_before=2026-11-26T09:00:00%2B09:00", "?source=grant&source=delegation", "?after="} {
		if got := f.permissions(root, query); got.status != 422 {
			t.Errorf("%q: %d %s", query, got.status, got.raw[:min(len(got.raw), 120)])
		}
	}
	f.code(t, "a member", f.permissions(f.as(MemberLogin), ""), 403, "permission_denied")
	f.code(t, "no session", f.permissions(actor{}, ""), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "access-query" }
	if got := f.permissions(root, ""); got.status != 503 {
		t.Errorf("a fault: %d", got.status)
	}
	if got := f.readAs(root, "/api/v1/identity/access/permissions/grant/q-00"); got.status != 503 {
		t.Errorf("a fault on the detail: %d", got.status)
	}
	f.s.fault = nil
}

func TestPermissionsCursorBelongsToItsQueryCallerAndSource(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.manyPermissions()
	next := func() string {
		got := f.permissions(root, "?limit=3")
		if got.status != 200 {
			t.Fatalf("first page %d %s", got.status, got.raw)
		}
		return url.QueryEscape(got.body["next"].(string))
	}
	conflict := func(name string, got reply) {
		t.Helper()
		if got.status != 409 || got.body["code"] != "revision_conflict" {
			t.Errorf("%s: %d %s", name, got.status, got.raw[:min(len(got.raw), 160)])
		}
	}
	token := next()
	if got := f.permissions(root, "?limit=3&after="+token); got.status != 200 || len(got.body["rows"].([]any)) != 3 {
		t.Fatalf("the cursor with its own query: %d", got.status)
	}
	conflict("another filter", f.permissions(root, "?limit=3&source=grant&after="+token))
	conflict("another page size", f.permissions(root, "?limit=4&after="+token))
	conflict("another administrator", f.permissions(second, "?limit=3&after="+token))
	conflict("a changed token", f.permissions(root, "?limit=3&after="+token[:len(token)-2]+"xx"))
	conflict("not a token", f.permissions(root, "?limit=3&after=abc"))
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "later", SubjectID: SampleAdmin2, Terms: permissionTerms(gate.All()), Validity: gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}}))
	conflict("after a permission was added", f.permissions(root, "?limit=3&after="+token))
	token = next()
	if got := f.run(root, "access.request_revoke", "q-00", nil); got.status != 200 {
		t.Fatalf("request %s", got.raw)
	}
	conflict("after a change request was made", f.permissions(root, "?limit=3&after="+token))
	token = next()
	f.advance(time.Duration(f.s.cfg.Access.ServiceQuerySeconds+1) * time.Second)
	conflict("after the cursor's lifetime", f.permissions(f.as(AdminLogin), "?limit=3&after="+token))
}

func TestPermissionsAreEvaluatedAtTheReferenceTimeOfTheirFirstPage(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	now := f.s.clock()
	for n := 0; n < 4; n++ {
		f.must(f.s.admin.AddGrant(gate.Grant{ID: fmt.Sprintf("short-%d", n), SubjectID: SampleAdmin2, Terms: permissionTerms(gate.All()), Validity: gate.Validity{NotBefore: now, NotAfter: now.Add(2 * time.Second)}}))
	}
	first := f.permissions(root, "?q=short&limit=2")
	if first.status != 200 || len(first.body["rows"].([]any)) != 2 {
		t.Fatalf("first %d %s", first.status, first.raw)
	}
	f.advance(3 * time.Second)
	rest := f.permissions(root, "?q=short&limit=2&after="+url.QueryEscape(first.body["next"].(string)))
	if rest.status != 200 || rest.body["reference_at"] != first.body["reference_at"] || rest.body["observed_at"] == first.body["observed_at"] {
		t.Fatalf("the second page: %d %s", rest.status, rest.raw[:min(len(rest.raw), 200)])
	}
	for _, raw := range rest.body["rows"].([]any) {
		if row := raw.(map[string]any); row["state"] != "active" {
			t.Errorf("%v is %v on a later page of a list whose reference time is before its end", row["id"], row["state"])
		}
	}
	fresh := f.permissions(root, "?q=short&limit=4")
	for _, raw := range fresh.body["rows"].([]any) {
		if row := raw.(map[string]any); row["state"] != "expired" || row["effective"] != false {
			t.Errorf("%v is %v in a fresh list", row["id"], row["state"])
		}
	}
	detail := f.readAs(root, "/api/v1/identity/access/permissions/grant/short-0")
	if detail.status != 200 || detail.body["row"].(map[string]any)["state"] != "expired" {
		t.Errorf("the detail is evaluated when it is read: %s", detail.raw)
	}
}

func TestPermissionsCrossFromGrantsToDelegationsAtAPageBoundary(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	grants, delegations := f.manyPermissions()
	onlyGrants, last := f.walkPermissions(root, fmt.Sprint("?source=grant&limit=", min(grants, f.s.cfg.Access.ServiceQueryPage)))
	if len(onlyGrants) != grants || last.body["next"] != "" {
		t.Fatalf("source=grant: %d of %d, next %v", len(onlyGrants), grants, last.body["next"])
	}
	if grants > f.s.cfg.Access.ServiceQueryPage {
		t.Skip("the sample has more grants than one page holds")
	}
	first := f.permissions(root, fmt.Sprint("?limit=", grants))
	rows := permissionIDs(first.body["rows"].([]any))
	if len(rows) != grants || !strings.HasPrefix(rows[len(rows)-1], "grant/") || first.body["next"] == "" {
		t.Fatalf("a page that ends exactly where the grants end: %d rows, next %v", len(rows), first.body["next"])
	}
	rest := f.permissions(root, fmt.Sprint("?limit=", grants, "&after=", url.QueryEscape(first.body["next"].(string))))
	got := permissionIDs(rest.body["rows"].([]any))
	if rest.status != 200 || len(got) != delegations || !strings.HasPrefix(got[0], "delegation/") || rest.body["next"] != "" {
		t.Fatalf("the page after it holds the delegations: %v next %v", got, rest.body["next"])
	}
	hidden := 0
	for _, g := range f.s.admin.Grants() {
		if gate.InstallationAction(g.Terms.Action) {
			hidden++
		}
	}
	whole := f.permissions(root, "?limit=50")
	if hidden == 0 || fmt.Sprint(whole.body["work_units"]) != fmt.Sprint(grants+delegations+hidden) {
		t.Errorf("work units %v for %d listed and %d hidden rows", whole.body["work_units"], grants+delegations, hidden)
	}
}

func TestPermissionsScanStopsAtItsBudgetAndContinues(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	grants, delegations := f.manyPermissions()
	f.s.cfg.Access.ServiceQueryScan = 6
	first := f.permissions(root, "?source=delegation&q=qd&limit=4")
	if first.status != 200 {
		t.Fatal(first.raw)
	}
	got, last := f.walkPermissions(root, "?q=qd&limit=4")
	if len(got) != 4 || last.body["count"].(map[string]any)["certainty"] != "exact" {
		t.Errorf("walking past %d grants with a budget of six a page: %v", grants, got)
	}
	stopped := f.permissions(root, "?q=qd&limit=4")
	if stopped.body["scan_stopped"] != true || fmt.Sprint(stopped.body["work_units"]) != "6" || len(stopped.body["rows"].([]any)) != 0 || stopped.body["next"] == "" || stopped.body["count"].(map[string]any)["certainty"] != "lower_bound" {
		t.Errorf("a page that spent its budget on rows that do not match: %s", stopped.raw[:min(len(stopped.raw), 300)])
	}
	_ = delegations
}

func TestPermissionDetailAnswersHiddenAndUnknownAlike(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	f.manyPermissions()
	hidden := ""
	for _, g := range f.s.admin.Grants() {
		if gate.InstallationAction(g.Terms.Action) {
			hidden = g.ID
		}
	}
	if hidden == "" {
		t.Fatal("the sample has no grant for an installation action")
	}
	base := "/api/v1/identity/access/permissions/"
	known := f.readAs(root, base+"grant/q-05")
	if row := known.body["row"].(map[string]any); known.status != 200 || row["id"] != "q-05" || row["source"] != sourceGrant || known.body["workspace"] != SampleWorkspace || known.body["observed_at"] == "" {
		t.Fatalf("a grant: %s", known.raw)
	}
	if got := f.readAs(root, base+"delegation/qd-02"); got.status != 200 || got.body["row"].(map[string]any)["delegator"] != SampleAdmin {
		t.Fatalf("a delegation: %s", got.raw)
	}
	unknown := f.readAs(root, base+"grant/no-such-grant")
	for name, path := range map[string]string{"an installation-action grant": "grant/" + hidden, "a delegation's identifier as a grant": "grant/qd-02", "a grant's identifier as a delegation": "delegation/q-05",
		"an unknown delegation": "delegation/none", "a query string": "grant/q-05?x=1"} {
		got := f.readAs(root, base+path)
		if got.status != unknown.status || got.body["code"] != unknown.body["code"] || unknown.status != 422 {
			t.Errorf("%s: %d %v, an unknown grant %d %v", name, got.status, got.body["code"], unknown.status, unknown.body["code"])
		}
	}
	for _, path := range []string{"grant/" + hidden, "grant/q-05"} {
		f.code(t, "a member reading "+path, f.readAs(f.as(MemberLogin), base+path), 403, "permission_denied")
	}
	ids, _ := f.walkPermissions(root, "?limit=50")
	for _, id := range ids {
		if id == "grant/"+hidden {
			t.Error("the installation-action grant is in the list")
		}
	}
}
