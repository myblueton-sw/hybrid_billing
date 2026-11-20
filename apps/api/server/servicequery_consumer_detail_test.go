package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestServiceQueryKnownAudienceIndependentOfSearch(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	page := f.serviceQuery(root, "/api/v1/identity/service-audiences?q=absent-audience")
	if page.status != 200 || len(page.body["rows"].([]any)) != 0 {
		t.Fatal("search premise", page.status)
	}
	got := f.serviceQuery(root, "/api/v1/identity/service-audiences/test-service-api")
	if got.status != 200 {
		t.Fatal("known configured audience must resolve independently", got.status)
	}
	row := got.body["row"].(map[string]any)
	if row["id"] != "test-service-api" || row["resource"] != "https://service.example.invalid/api" || got.body["catalog_version"] != f.s.serviceCatalog.Version() || got.body["workspace"] != SampleWorkspace {
		t.Fatal("wrong configured metadata")
	}
	if got.body["eligibility"] != nil {
		t.Fatal("configured metadata must not claim effective eligibility")
	}
	for _, id := range []string{"missing", "bad!"} {
		refused := f.serviceQuery(root, "/api/v1/identity/service-audiences/"+id)
		f.failed(t, "uniform unknown reference", refused, commands.PermissionDenied)
		if refused.body["row"] != nil || refused.body["count"] != nil {
			t.Fatal("refused metadata leak")
		}
	}
}

func TestServiceQueryCommandByteLimits(t *testing.T) {
	for _, rationale := range []int{128, 512} {
		t.Run(strconv.Itoa(rationale), func(t *testing.T) {
			value := 8192
			if strconv.IntSize == 64 {
				parsed, err := strconv.ParseInt("9007199254740993", 10, 64)
				if err != nil {
					t.Fatal(err)
				}
				value = int(parsed)
			}
			f := setupWith(t, func(c *Config) {
				c.Commands = CommandLimits()
				c.Commands.Value = value
				c.Commands.Rationale = rationale
			})
			root := f.as(AdminLogin)
			service := f.newService(root, "byte-limits")
			for _, path := range []string{"/api/v1/identity/services", "/api/v1/identity/services/" + service} {
				got := f.serviceQuery(root, path)
				if got.status != 200 {
					t.Fatal("limits response", got.status)
				}
				data, err := json.Marshal(got.body["limits"])
				if err != nil {
					t.Fatal(err)
				}
				var limits struct {
					Value     string `json:"command_value_bytes"`
					Rationale int    `json:"rationale_bytes"`
					Body      int    `json:"body_bytes"`
				}
				if err := json.Unmarshal(data, &limits); err != nil {
					t.Fatal(err)
				}
				if limits.Value != strconv.Itoa(value) || limits.Rationale != min(256, rationale) || limits.Body != 1048576 {
					t.Fatalf("actual byte limits missing or rounded: %+v", limits)
				}
			}
		})
	}
}

func TestServiceQueryAudienceAdmissionAndForeignMetadata(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		var catalog struct {
			Audiences []map[string]any `json:"audiences"`
		}
		if err := json.Unmarshal(c.ServiceCatalog, &catalog); err != nil {
			t.Fatal(err)
		}
		catalog.Audiences = append(catalog.Audiences, map[string]any{"workspace": EmptyWorkspace, "id": "foreign-audience", "resource": "urn:synthetic:foreign", "scopes": []any{}})
		data, err := json.Marshal(catalog)
		if err != nil {
			t.Fatal(err)
		}
		c.ServiceCatalog = data
	})
	root := f.as(AdminLogin)
	member := f.as(MemberLogin)
	for _, path := range []string{"/api/v1/identity/service-audiences/foreign-audience", "/api/v1/identity/service-audiences/missing"} {
		got := f.serviceQuery(root, path)
		f.failed(t, "uniform foreign and unknown audience", got, commands.PermissionDenied)
		if got.body["row"] != nil || got.body["count"] != nil {
			t.Fatal("foreign metadata disclosed")
		}
	}
	f.failed(t, "non-administrator audience detail", f.serviceQuery(member, "/api/v1/identity/service-audiences/test-service-api"), "unauthenticated")
	got := f.do("GET", "/api/v1/identity/service-audiences/test-service-api", "", func(r *http.Request) { r.AddCookie(root.cookie) })
	if got.status != 403 {
		t.Fatal("audience detail must enforce CSRF", got.status)
	}
	wrong := f.serviceQuery(root, "/api/v1/identity/service-audiences/test-service-api?workspace="+EmptyWorkspace)
	if wrong.status == 200 || wrong.body["row"] != nil {
		t.Fatal("foreign requested workspace metadata disclosed")
	}
}
