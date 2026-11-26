package server

import (
	"encoding/json"
	"net/url"
	"strconv"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func TestServiceQueryDeclarationHistoryKeepsRetainedVersionsAndUnavailableDistinct(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "history")
	for version := 1; version < 7; version++ {
		got := f.run(root, "service.declare", svc, f.declaredFields(map[string]string{"declaration_version": strconv.Itoa(version), "purpose": "Purpose " + strconv.Itoa(version+1)}))
		if got.status != 200 {
			t.Fatal("history update", got.status, got.body["code"])
		}
	}
	path := "/api/v1/identity/services/" + svc + "/declarations?limit=2"
	rows := []string{}
	next := ""
	var final reply
	for {
		query := path
		if next != "" {
			query += "&after=" + url.QueryEscape(next)
		}
		got := f.serviceQuery(root, query)
		if got.status != 200 {
			t.Fatal("history query", got.status, got.body["code"])
		}
		for _, raw := range got.body["rows"].([]any) {
			row := raw.(map[string]any)
			rows = append(rows, row["version"].(string))
			if row["state"] != "recorded" || row["recorded_at"] == "" {
				t.Fatal("history record lost provenance")
			}
		}
		next = got.body["next"].(string)
		final = got
		if next == "" {
			break
		}
	}
	if len(rows) != 5 || rows[0] != "7" || rows[1] != "6" || rows[4] != "3" || final.body["count"].(map[string]any)["value"] != "5" || final.body["count"].(map[string]any)["certainty"] != "exact" {
		t.Fatal("retained bounded history lost order/count", rows)
	}
	pruned := f.serviceQuery(root, "/api/v1/identity/services/"+svc+"/declarations?version=1")
	if pruned.status != 200 || pruned.body["availability"] != "version_unavailable" || len(pruned.body["rows"].([]any)) != 0 {
		t.Fatal("pruned version was fabricated or confused with no recorded history")
	}
	known := f.serviceQuery(root, "/api/v1/identity/services/"+svc+"/declarations?version=4")
	if known.status != 200 || known.body["availability"] != "recorded" || known.body["rows"].([]any)[0].(map[string]any)["version"] != "4" {
		t.Fatal("known retained version not resolved")
	}
	f.must(f.s.identity.Workspace().CreateServiceUser(root.verifier(f), SampleWorkspace, "legacy", SampleAdmin))
	absent := f.serviceQuery(root, "/api/v1/identity/services/legacy/declarations")
	if absent.status != 200 || absent.body["availability"] != "not_recorded" || len(absent.body["rows"].([]any)) != 0 {
		t.Fatal("legacy history was fabricated")
	}
}

func TestServiceQueryKnownSelectionIsIndependentOfPageAndHidesForeignReferences(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		var raw map[string]any
		if err := json.Unmarshal(testServiceCatalog(), &raw); err != nil {
			t.Fatal(err)
		}
		foreign := raw["audiences"].([]any)[1].(map[string]any)
		foreign["id"] = "foreign-audience"
		encoded, err := json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		c.ServiceCatalog = encoded
	})
	root := f.as(AdminLogin)
	svc := f.newService(root, "selection")
	credential, _, _ := f.issued(root, svc, "1")
	filtered := f.serviceQuery(root, "/api/v1/identity/services?q=not-matching")
	if len(filtered.body["rows"].([]any)) != 0 {
		t.Fatal("filter premise")
	}
	for _, path := range []string{"/api/v1/identity/services/" + svc, "/api/v1/identity/service-credentials/" + credential} {
		got := f.serviceQuery(root, path)
		if got.status != 200 || got.body["row"] == nil {
			t.Fatal("known selection incorrectly depended on current page", path, got.status)
		}
	}
	_, err := f.s.orgs.PutWorkspace(organizations.Workspace{ID: "foreign-workspace", Profile: organizations.ProfileMSP, State: organizations.Active}, 0)
	f.must(err)
	_, err = f.s.orgs.PutParty(organizations.Party{ID: "foreign-party", WorkspaceID: "foreign-workspace", Kind: organizations.LegalEntity, State: organizations.Active}, 0)
	f.must(err)
	f.must(f.s.identity.Installation().CreateUser(seedOperator, "foreign-human", identity.Triple{Connection: "foreign-connection", Issuer: "https://foreign.example.invalid", Subject: "synthetic-human"}, "foreign@example.invalid"))
	for _, path := range []string{"/api/v1/identity/service-organizations/foreign-party", "/api/v1/identity/service-organizations/missing", "/api/v1/identity/service-owners/foreign-human", "/api/v1/identity/service-owners/missing", "/api/v1/identity/service-audiences/foreign-audience/scopes", "/api/v1/identity/service-audiences/missing/scopes", "/api/v1/identity/service-credentials/missing", "/api/v1/identity/services/missing"} {
		got := f.serviceQuery(root, path)
		f.failed(t, "uniform unknown/foreign reference", got, commands.PermissionDenied)
		if got.body["row"] != nil || got.body["rows"] != nil || got.body["count"] != nil {
			t.Fatal("refused reference disclosed metadata", path)
		}
	}
}

func TestServiceQueryCredentialLimitRetainsExactLargeConfiguration(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("large int configuration requires a64-bit runtime")
	}
	maximum := uint64(1)<<54 + 3
	f := setupWith(t, func(c *Config) { c.Identity.CredentialsPerServiceUser = int(maximum) })
	root := f.as(AdminLogin)
	svc := f.newService(root, "precise")
	detail := f.serviceQuery(root, "/api/v1/identity/services/"+svc)
	if detail.status != 200 {
		t.Fatal("large limit detail", detail.status)
	}
	want := strconv.FormatUint(maximum, 10)
	if detail.body["eligibility"].(map[string]any)["credential_limit"] != want || detail.body["limits"].(map[string]any)["credentials_per_service"] != want {
		t.Fatal("arbitrary configured limit lost decimal precision")
	}
}
