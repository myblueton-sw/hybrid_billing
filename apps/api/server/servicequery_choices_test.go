package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func queryCatalogScopes(t testing.TB, count int, named bool) []byte {
	t.Helper()
	raw := testServiceCatalog()
	if named {
		raw = namedServiceCatalog("sample-le-north", "sample-le-south", "sample-contract-01")
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	audience := data["audiences"].([]any)[0].(map[string]any)
	template := audience["scopes"].([]any)[0].(map[string]any)
	scopes := []any{}
	for i := 0; i < count; i++ {
		copy := map[string]any{}
		for k, v := range template {
			copy[k] = v
		}
		copy["id"] = fmt.Sprintf("scope-%03d", i)
		scopes = append(scopes, copy)
	}
	audience["scopes"] = scopes
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestServiceQueryScopeBudgetDoesNotSkipUnevaluatedEntries(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.ServiceCatalog = queryCatalogScopes(t, 20, true)
		c.Access = DefaultAccessSettings()
		c.Access.ServiceQueryScan = 100
	})
	root := f.as(AdminLogin)
	path := "/api/v1/identity/service-audiences/test-service-api/scopes?limit=100"
	first := f.serviceQuery(root, path)
	if first.status != 200 || len(first.body["rows"].([]any)) != 12 || fmt.Sprint(first.body["work_units"]) != "96" || first.body["scan_stopped"] != true || first.body["next"] == "" {
		t.Fatal("scope eligibility escaped/reserved beyond work budget", first.status, first.body["code"])
	}
	second := f.serviceQuery(root, path+"&after="+url.QueryEscape(first.body["next"].(string)))
	if second.status != 200 || len(second.body["rows"].([]any)) != 8 || second.body["next"] != "" || second.body["count"].(map[string]any)["value"] != "20" {
		t.Fatal("unevaluated scopes were skipped")
	}
	firstID := first.body["rows"].([]any)[0].(map[string]any)["id"]
	lastID := second.body["rows"].([]any)[7].(map[string]any)["id"]
	if firstID != "scope-000" || lastID != "scope-019" {
		t.Fatal("scope order/continuation changed")
	}
	if second.body["rows"].([]any)[0].(map[string]any)["workspace"] != SampleWorkspace {
		t.Fatal("scope restrictions lost their canonical workspace")
	}
	restrictions := second.body["rows"].([]any)[0].(map[string]any)["restrictions"].(map[string]any)
	if restrictions["customer"].(map[string]any)["id"] != "sample-le-north" || restrictions["legal_entity"].(map[string]any)["id"] != "sample-le-south" || restrictions["contract"].(map[string]any)["id"] != "sample-contract-01" || restrictions["customer"].(map[string]any)["any"] != nil || restrictions["period"].(map[string]any)["any"] != true {
		t.Fatal("named restrictions or explicit unrestricted period lost their meaning")
	}
}

func TestServiceQueryScopeReferenceFreezesScheduledContractTransitions(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.ServiceCatalog = queryCatalogScopes(t, 2, true) })
	seq, err := f.s.lifecycles.Sequence(SampleWorkspace, "sample-contract-01")
	f.must(err)
	day, err := contracts.NewDay(2026, 11, 17)
	f.must(err)
	_, err = f.s.lifecycles.Amend(SampleWorkspace, "sample-contract-01", seq, "query-scheduled-suspend", nil, []contracts.Event{{ID: "query-next-day", Kind: contracts.Suspend, From: day}}, "")
	f.must(err)
	f.advance(14*time.Hour + 59*time.Minute)
	root := f.as(AdminLogin)
	path := "/api/v1/identity/service-audiences/test-service-api/scopes?limit=1"
	first := f.serviceQuery(root, path)
	if first.status != 200 || len(first.body["rows"].([]any)) != 1 || first.body["next"] == "" {
		t.Fatal("eligible pre-transition scope premise", first.status, first.body["code"])
	}
	f.advance(2 * time.Minute)
	second := f.serviceQuery(root, path+"&after="+url.QueryEscape(first.body["next"].(string)))
	if second.status != 200 || len(second.body["rows"].([]any)) != 1 || second.body["reference_at"] != first.body["reference_at"] {
		t.Fatal("scheduled transition changed frozen scope membership", second.status, second.body["code"])
	}
	current := f.serviceQuery(root, "/api/v1/identity/service-audiences/test-service-api/scopes/scope-001")
	if current.status != 200 || current.body["eligibility"] != "ineligible" {
		t.Fatal("known scope detail did not recompute current contract eligibility", current.status, current.body["code"])
	}
}

func TestServiceQueryChoicesPreserveCurrentParentAndFrozenOwnerEligibility(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.setPassword(MemberLogin, pw)
	f.must(f.s.identity.Workspace().SetRolePeriod(root.verifier(f), SampleWorkspace, SampleMember, start.Add(time.Minute), 1, start))
	owners := f.serviceQuery(root, "/api/v1/identity/service-owners?limit=1")
	if owners.status != 200 || owners.body["next"] == "" {
		t.Fatal("owner paging premise")
	}
	f.advance(time.Minute)
	full := []any{}
	next := owners.body["next"].(string)
	for next != "" {
		page := f.serviceQuery(root, "/api/v1/identity/service-owners?limit=1&after="+url.QueryEscape(next))
		if page.status != 200 {
			t.Fatal("natural owner expiry changed reference membership", page.status, page.body["code"])
		}
		full = append(full, page.body["rows"].([]any)...)
		next = page.body["next"].(string)
	}
	found := false
	for _, raw := range full {
		if raw.(map[string]any)["id"] == SampleMember {
			found = true
		}
	}
	if !found {
		t.Fatal("owner was evaluated at current time instead of signed reference")
	}
	current := f.serviceQuery(root, "/api/v1/identity/service-owners/"+SampleMember)
	if current.status != 200 || current.body["row"].(map[string]any)["eligible_at_reference"] != false {
		t.Fatal("known owner detail did not recompute current eligibility")
	}
	parent, version, err := f.s.orgs.Party(SampleWorkspace, "sample-le-north")
	f.must(err)
	parent.State = organizations.Suspended
	_, err = f.s.orgs.PutParty(parent, version)
	f.must(err)
	choices := f.serviceQuery(root, "/api/v1/identity/service-organizations")
	if choices.status != 200 {
		t.Fatal("organization choices", choices.status)
	}
	for _, raw := range choices.body["rows"].([]any) {
		id := raw.(map[string]any)["id"]
		if id == "sample-le-north" || id == "sample-unit-north-ops" || id == "sample-unit-north-data" {
			t.Fatal("suspended parent left eligible organization choices", id)
		}
	}
	_, err = f.s.orgs.PutParty(organizations.Party{ID: "unlabelled", WorkspaceID: SampleWorkspace, Kind: organizations.LegalEntity, State: organizations.Active}, 0)
	f.must(err)
	fallback := f.serviceQuery(root, "/api/v1/identity/service-organizations/unlabelled")
	row := fallback.body["row"].(map[string]any)
	if fallback.status != 200 || row["name"] != "" || row["name_state"] != "unavailable" || row["id"] != "unlabelled" {
		t.Fatal("organization name was synthesized")
	}
}
