package server

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func TestServiceQueryNamedPeriodAndSelectorProjectionRemainExact(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		var data map[string]any
		if err := json.Unmarshal(namedServiceCatalog("sample-le-north", "sample-le-south", "sample-contract-01"), &data); err != nil {
			t.Fatal(err)
		}
		scope := data["audiences"].([]any)[0].(map[string]any)["scopes"].([]any)[0].(map[string]any)
		scope["period"] = map[string]any{"from": "2026-11", "until": "2027-01"}
		encoded, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		c.ServiceCatalog = encoded
	})
	root := f.as(AdminLogin)
	path := "/api/v1/identity/service-audiences/test-service-api/scopes"
	listed := f.serviceQuery(root, path)
	known := f.serviceQuery(root, path+"/read")
	if listed.status != 200 || known.status != 200 || len(listed.body["rows"].([]any)) != 1 || known.body["eligibility"] != "eligible" {
		t.Fatal("named configured scope was not available as metadata", listed.status, known.status)
	}
	for _, raw := range []any{listed.body["rows"].([]any)[0], known.body["row"]} {
		row := raw.(map[string]any)
		restrictions := row["restrictions"].(map[string]any)
		period := restrictions["period"].(map[string]any)
		if period["from"] != "2026-11" || period["until"] != "2027-01" || period["any"] != nil {
			t.Fatal("named exclusive month interval was widened or reformatted")
		}
		for key, want := range map[string]string{"customer": "sample-le-north", "legal_entity": "sample-le-south", "contract": "sample-contract-01"} {
			selector := restrictions[key].(map[string]any)
			if selector["id"] != want || selector["any"] != nil {
				t.Fatal("named selector became a wildcard", key)
			}
		}
	}
	for _, uri := range []string{path + "?q=not-a-configured-scope", "/api/v1/identity/service-organizations?q=not-a-configured-organization"} {
		got := f.serviceQuery(root, uri)
		if got.status != 200 || len(got.body["rows"].([]any)) != 0 || got.body["count"].(map[string]any)["value"] != "0" {
			t.Fatal("nonmatching choice search fabricated results", got.status)
		}
	}
}

func TestServiceQueryIneligibleScopeIsKnownMetadataButNotAnEligibleChoice(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.ServiceCatalog = namedServiceCatalog("missing-customer", "", "") })
	root := f.as(AdminLogin)
	path := "/api/v1/identity/service-audiences/test-service-api/scopes"
	listed, known := f.serviceQuery(root, path), f.serviceQuery(root, path+"/read")
	if listed.status != 200 || len(listed.body["rows"].([]any)) != 0 || listed.body["count"].(map[string]any)["value"] != "0" {
		t.Fatal("ineligible configured scope became an eligible choice", listed.status)
	}
	if known.status != 200 || known.body["eligibility"] != "ineligible" || known.body["row"].(map[string]any)["id"] != "read" {
		t.Fatal("configured ineligible scope metadata disappeared", known.status)
	}
	for _, uri := range []string{path + "/missing", "/api/v1/identity/service-audiences/missing/scopes/read"} {
		got := f.serviceQuery(root, uri)
		f.failed(t, "uniform missing scope metadata", got, commands.PermissionDenied)
		if got.body["row"] != nil || got.body["eligibility"] != nil {
			t.Fatal("missing scope disclosed metadata")
		}
	}
}

func TestServiceQueryContractDependencyLossDoesNotInventScopeEligibility(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.ServiceCatalog = namedServiceCatalog("sample-le-north", "sample-le-south", "sample-contract-01")
	})
	root := f.as(AdminLogin)
	f.s.fault = func(at string) bool { return at == "contract-read" }
	path := "/api/v1/identity/service-audiences/test-service-api/scopes"
	listed := f.serviceQuery(root, path)
	f.failed(t, "scope choices cannot evaluate current contract", listed, commands.DependencyUnavailable)
	if listed.body["rows"] != nil || listed.body["count"] != nil {
		t.Fatal("unavailable scope choices published a partial set")
	}
	known := f.serviceQuery(root, path+"/read")
	if known.status != 200 || known.body["eligibility"] != "unavailable" || known.body["row"].(map[string]any)["id"] != "read" {
		t.Fatal("unavailable contract was confused with unknown configuration", known.status)
	}
}

func TestServiceQuerySuspendedOrganizationRemainsKnownButIneligible(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	party, version, err := f.s.orgs.Party(SampleWorkspace, "sample-le-seller")
	f.must(err)
	party.State = organizations.Suspended
	_, err = f.s.orgs.PutParty(party, version)
	f.must(err)
	got := f.serviceQuery(root, "/api/v1/identity/service-organizations/sample-le-seller")
	if got.status != 200 || got.body["eligibility"] != "ineligible" || got.body["row"].(map[string]any)["id"] != party.ID {
		t.Fatal("suspended known organization was hidden or falsely eligible", got.status)
	}
	for _, id := range []string{"missing", "bad!"} {
		got := f.serviceQuery(root, "/api/v1/identity/services/"+url.PathEscape(id)+"/declarations")
		f.failed(t, "uniform invalid or missing history target", got, commands.PermissionDenied)
		if got.body["rows"] != nil || got.body["availability"] != nil {
			t.Fatal("unknown history target disclosed metadata")
		}
	}
}
