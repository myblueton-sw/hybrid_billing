package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func assertServiceQueryFailurePayload(t testing.TB, got reply, status int, code commands.Code) {
	t.Helper()
	if got.status != status || got.body["code"] != string(code) {
		t.Fatal("query refusal classification", got.status, got.body["code"])
	}
	for _, key := range []string{"row", "rows", "count", "admission", "query", "next", "settings", "limits", "reference_at", "observed_at", "catalog_version", "availability", "revision", "order", "scanned", "work_units", "scan_stopped", "search_input", "eligibility", "workspace"} {
		if _, found := got.body[key]; found {
			t.Fatal("query refusal published partial metadata", key)
		}
	}
	decoder := json.NewDecoder(strings.NewReader(got.raw))
	var first, second any
	if decoder.Decode(&first) != nil || decoder.Decode(&second) != io.EOF {
		t.Fatal("query refusal was not one complete JSON response")
	}
}

func serviceWalkRequest(root actor, path string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.AddCookie(root.cookie)
	r.Header.Set(csrfHeader, root.csrf)
	r.Header.Set("X-Request-Id", "query-lifetime-fixture")
	return r
}

func TestServiceQueryHttpExpiryBeforeModuleReadPublishesOnlyRefusal(t *testing.T) {
	for _, tc := range []struct {
		name, path, seam string
		status           int
		code             commands.Code
	}{
		{"account-count", "/api/v1/identity/services", "services", 503, commands.DependencyUnavailable},
		{"credential-count", "/api/v1/identity/service-credentials", "services", 503, commands.DependencyUnavailable},
		{"filtered-credential-service", "/api/v1/identity/service-credentials?service=svc-lifetime", "services", 503, commands.DependencyUnavailable},
		{"owner-page", "/api/v1/identity/service-owners", "roster", 503, commands.DependencyUnavailable},
		{"known-service-lookup", "/api/v1/identity/services/svc-lifetime", "services", 503, commands.DependencyUnavailable},
		{"known-credential-lookup", "/api/v1/identity/service-credentials/", "credentials", 503, commands.DependencyUnavailable},
		{"known-owner-lookup", "/api/v1/identity/service-owners/" + SampleAdmin, "roster", 503, commands.DependencyUnavailable},
		{"known-service-publication", "/api/v1/identity/services/svc-lifetime", "service-declaration", 401, commands.Unauthenticated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			service := f.newService(root, "lifetime")
			credential, _, _ := f.issued(root, service, "1")
			path := tc.path
			if tc.name == "known-credential-lookup" {
				path += credential
			}
			expired := false
			f.s.fault = func(at string) bool {
				if at == tc.seam && !expired {
					expired = true
					f.advance(48 * time.Hour)
				}
				return false
			}
			got := f.serviceQuery(root, path)
			if !expired {
				t.Fatal("named module boundary was not reached")
			}
			assertServiceQueryFailurePayload(t, got, tc.status, tc.code)
			f.s.fault = nil
			assertServiceQueryFailurePayload(t, f.serviceQuery(root, "/api/v1/identity/services"), 401, commands.Unauthenticated)
		})
	}
}

func TestServiceQueryCredentialPageExpiryDiscardsAlreadyProjectedRows(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.Identity.PageSize = 1 })
	root := f.as(AdminLogin)
	service := f.newService(root, "mid-page")
	f.issued(root, service, "1")
	f.issued(root, service, "1")
	r := serviceWalkRequest(root, "/api/v1/identity/service-credentials?service="+service+"&limit=10")
	walk, ok := f.s.beginServiceQuery(httptest.NewRecorder(), r, "credentials", "")
	if !ok {
		t.Fatal("admitted credential walk premise")
	}
	original := f.s.clock
	projected := false
	f.s.clock = func() time.Time {
		if !projected {
			projected = true
			f.advance(48 * time.Hour)
		}
		return original()
	}
	defer func() { f.s.clock = original }()
	rows, more, err := walk.credentials()
	f.s.clock = original
	if !projected || err == nil || rows != nil || more || walk.cursor.Matched != 1 {
		t.Fatal("mid-page expiry leaked a partially projected credential page")
	}
	w := httptest.NewRecorder()
	serviceQueryFailure(w, r, err)
	var body map[string]any
	if json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatal("component failure encoding")
	}
	assertServiceQueryFailurePayload(t, reply{status: w.Code, body: body, raw: w.Body.String()}, 503, commands.DependencyUnavailable)
	assertServiceQueryFailurePayload(t, f.serviceQuery(root, "/api/v1/identity/services"), 401, commands.Unauthenticated)
}
