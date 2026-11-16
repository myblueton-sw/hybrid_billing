package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

func (f world) orgRead(a actor, query string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/organizations"+query, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func members(got reply) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, m := range got.body["members"].([]any) {
		row := m.(map[string]any)
		out[row["member"].(string)] = row
	}
	return out
}

func targetIDs(row map[string]any) string {
	ids := []string{}
	for _, t := range row["targets"].([]any) {
		ids = append(ids, t.(map[string]any)["id"].(string))
	}
	return strings.Join(ids, ",")
}

func TestOrganizationsReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	got := f.orgRead(root, "?on=2026-11-16")
	if got.status != 200 || got.body["on"] != "2026-11-16" || got.body["kind"] != "hierarchy" || got.body["workspace"].(map[string]any)["profile"] != "msp" ||
		fmt.Sprint(got.body["relationship_revision"]) != "1" || fmt.Sprint(got.body["active_revision"]) != "1" || got.body["next"] != "" {
		t.Fatalf("read %+v", got.body)
	}
	if ws := got.body["workspace"].(map[string]any); ws["id"] != SampleWorkspace {
		t.Errorf("workspace %v", ws)
	}
	parties := got.body["parties"].([]any)
	if len(parties) != 6 || parties[0].(map[string]any)["id"] != "sample-le-north" || parties[0].(map[string]any)["name"] != "North Holdings (sample)" {
		t.Errorf("parties %v", parties)
	}
	profiles := got.body["profiles"].([]any)
	if len(profiles) != 3 || profiles[1].(map[string]any)["path"] != "internal_chargeback" || profiles[1].(map[string]any)["payer"] != "" {
		t.Errorf("profiles %v", profiles)
	}
	rows := members(got)
	if len(rows) != 9 || targetIDs(rows["sample-acct-02"]) != "sample-unit-north-data" || targetIDs(rows["sample-unit-north-ops"]) != "sample-le-north" {
		t.Errorf("hierarchy %v", rows)
	}
	history := rows["sample-acct-02"]["history"].([]any)
	first, second := history[0].(map[string]any), history[1].(map[string]any)
	if first["from"] != "2026-01-01" || first["until"] != "2026-10-01" || second["from"] != "2026-10-01" || second["until"] != "" || second["name"] != "North data platform (sample)" {
		t.Errorf("history %v", history)
	}
	before := members(f.orgRead(root, "?on=2026-09-30"))
	at := members(f.orgRead(root, "?on=2026-10-01"))
	if targetIDs(before["sample-acct-02"]) != "sample-unit-north-ops" || targetIDs(at["sample-acct-02"]) != "sample-unit-north-data" {
		t.Errorf("boundary %v %v", before["sample-acct-02"], at["sample-acct-02"])
	}
	if early := members(f.orgRead(root, "?on=2025-12-31")); targetIDs(early["sample-acct-01"]) != "" {
		t.Errorf("before the start %v", early["sample-acct-01"])
	}
	views := members(f.orgRead(root, "?on=2026-11-16&kind=view_group"))
	if targetIDs(views["sample-acct-03"]) != "sample-view-north,sample-view-south" || targetIDs(views["sample-acct-04"]) != "sample-view-south" {
		t.Errorf("view groups %v", views)
	}
	for kind, n := range map[string]int{"payer": 6, "allocation_pool": 2, "contract_pool": 0} {
		if got := f.orgRead(root, "?kind="+kind); got.status != 200 || len(got.body["members"].([]any)) != n {
			t.Errorf("%s %d", kind, len(got.body["members"].([]any)))
		}
	}
	if today := f.orgRead(root, ""); today.body["on"] != "2026-11-16" {
		t.Errorf("default day %v", today.body["on"])
	}
	pinned := f.orgRead(root, "?revision=0")
	if pinned.status != 200 || len(pinned.body["members"].([]any)) != 0 || fmt.Sprint(pinned.body["relationship_revision"]) != "0" || fmt.Sprint(pinned.body["active_revision"]) != "1" {
		t.Errorf("pinned %+v", pinned.body)
	}
	f.code(t, "member", f.orgRead(f.as(MemberLogin), ""), 403, string(commands.PermissionDenied))
	f.code(t, "no session", f.orgRead(actor{}, ""), 401, string(commands.Unauthenticated))
}

func TestOrganizationsPagingAndQueries(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	page := f.orgRead(root, "?limit=4")
	ids := []string{}
	for page.status == 200 {
		for _, m := range page.body["members"].([]any) {
			ids = append(ids, m.(map[string]any)["member"].(string))
		}
		if page.body["next"] == "" {
			break
		}
		page = f.orgRead(root, "?limit=4&after="+page.body["next"].(string))
	}
	if len(ids) != 9 || ids[0] != "sample-acct-01" || ids[8] != "sample-unit-south-web" {
		t.Errorf("paged %v", ids)
	}
	if last := f.orgRead(root, "?limit=9"); last.body["next"] != "" || len(last.body["members"].([]any)) != 9 {
		t.Errorf("exact page %+v", last.body["next"])
	}
	if one := f.orgRead(root, "?limit=1"); one.status != 200 || len(one.body["members"].([]any)) != 1 || one.body["next"] != "sample-acct-01" {
		t.Errorf("limit 1 %+v", one.body["next"])
	}
	if top := f.orgRead(root, "?limit=99"); top.status != 200 {
		t.Errorf("limit 99 %d", top.status)
	}
	for _, q := range []string{"?on=2026-13-01", "?on=2026-2-01", "?on=x", "?kind=other", "?limit=0", "?limit=100", "?limit=05", "?limit=a",
		"?after=bad%20cursor", "?revision=2", "?revision=-1", "?revision=01", "?revision=x"} {
		f.code(t, q, f.orgRead(root, q), 422, string(commands.InvalidRequest))
	}
	f.s.fault = func(at string) bool { return at == "organizations" }
	f.code(t, "fault", f.orgRead(root, ""), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
}

func TestOrganizationsSeedAgreesWithParties(t *testing.T) {
	f := setup(t, SeedSample).ready()
	parties, _ := f.s.orgs.Parties(SampleWorkspace)
	day := sampleDay(2026, 11, 16)
	for _, p := range parties {
		if p.Kind != organizations.InternalUnit {
			continue
		}
		target, ok, err := f.s.rels.TargetOn(SampleWorkspace, 1, relationships.Hierarchy, p.ID, day)
		if err != nil || !ok || target != p.ParentID {
			t.Errorf("%s: hierarchy %s, parent %s", p.ID, target, p.ParentID)
		}
	}
	profiles, _ := f.s.orgs.BillingProfiles(SampleWorkspace)
	for _, p := range profiles {
		if p.Path != organizations.ExternalInvoice {
			continue
		}
		accounts, _ := f.s.rels.MembersOn(SampleWorkspace, 1, relationships.Payer, p.PayerID, day)
		if len(accounts) == 0 {
			t.Errorf("%s: no account paid by %s", p.ID, p.PayerID)
		}
	}
}

func TestOrganizationsWithoutSampleAndAcrossWorkspaces(t *testing.T) {
	f := setup(t, SeedNone).ready()
	if parties, err := f.s.orgs.Parties(EmptyWorkspace); err != nil || len(parties) != 0 {
		t.Errorf("empty install %v %v", parties, err)
	}
	page, err := f.s.readOrganizations(EmptyWorkspace, orgQuery{on: sampleDay(2026, 11, 16), kind: relationships.Hierarchy, limit: 50})
	if err != nil || len(page.Parties) != 0 || len(page.Members) != 0 || page.Relationship != 0 {
		t.Errorf("empty read %+v %v", page, err)
	}
	g := setup(t, SeedSample).ready()
	if _, err := g.s.orgs.PutWorkspace(organizations.Workspace{ID: "other-workspace", Profile: organizations.ProfileGroup, State: organizations.Active}, 0); err != nil {
		t.Fatal(err)
	}
	other, err := g.s.readOrganizations("other-workspace", orgQuery{on: sampleDay(2026, 11, 16), kind: relationships.Hierarchy, limit: 50})
	if err != nil || len(other.Parties) != 0 || len(other.Profiles) != 0 || len(other.Members) != 0 || other.Workspace.Profile != "group" {
		t.Errorf("other workspace %+v %v", other, err)
	}
	if _, err := g.s.readOrganizations("unknown-workspace", orgQuery{kind: relationships.Hierarchy, limit: 50}); err == nil {
		t.Error("unknown workspace read")
	}
	for i := 0; i <= orgCap; i++ {
		if _, err := g.s.orgs.PutParty(organizations.Party{ID: fmt.Sprintf("cap-%04d", i), WorkspaceID: "other-workspace", Kind: organizations.LegalEntity, State: organizations.Active}, 0); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := g.s.readOrganizations("other-workspace", orgQuery{kind: relationships.Hierarchy, limit: 50}); err == nil {
		t.Error("above the cap")
	}
	if _, err := g.s.readOrganizations("bad workspace", orgQuery{kind: relationships.Hierarchy, limit: 50}); err == nil {
		t.Error("an invalid workspace")
	}
	if _, err := g.s.readOrganizations("bad workspace", orgQuery{kind: relationships.Hierarchy, limit: 50, revision: 1, pinned: true}); err == nil || errors.Is(err, errInvalidQuery) {
		t.Errorf("a registry failure with a pinned revision %v", err)
	}
	if RelationshipLimits().PageSize < 2 {
		t.Error("the page size leaves no room for the look-ahead row")
	}
}
