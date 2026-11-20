package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"math"
	"net/url"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func addServiceQueryBoundaryCredentials(f world, root actor, service string, count, offset int) {
	f.t.Helper()
	verifier := root.verifier(f)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("cred-%016x", offset+i+1)
		f.must(f.s.identity.Workspace().AddCredential(verifier, SampleWorkspace, service, id, identity.Verifier{byte(offset + i + 1)}, identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	}
}

func serviceQueryBoundaryCatalog(t testing.TB) []byte {
	t.Helper()
	var data map[string]any
	if json.Unmarshal(testServiceCatalog(), &data) != nil {
		t.Fatal("catalog fixture decoding")
	}
	audiences := data["audiences"].([]any)
	first := audiences[0].(map[string]any)
	scopes := first["scopes"].([]any)
	extra := map[string]any{}
	for key, value := range scopes[0].(map[string]any) {
		extra[key] = value
	}
	extra["id"] = "read-extra"
	first["scopes"] = append(scopes, extra)
	second := map[string]any{}
	for key, value := range first {
		second[key] = value
	}
	second["id"] = "test-z-api"
	second["resource"] = "https://service.example.invalid/z"
	data["audiences"] = append(audiences, second)
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatal("catalog fixture encoding")
	}
	return encoded
}

func saturatedServiceContinuation(t testing.TB, f world, first reply, change func(*serviceCursor)) string {
	t.Helper()
	if first.status != 200 {
		t.Fatal("bounded first-page premise", first.status)
	}
	next, ok := first.body["next"].(string)
	if !ok || next == "" {
		t.Fatal("producer did not emit a continuation")
	}
	cursor, ok := f.s.openServiceCursor(next)
	if !ok {
		t.Fatal("producer continuation could not be decoded")
	}
	change(&cursor)
	sealed, ok := f.s.sealServiceCursor(cursor)
	if !ok {
		t.Fatal("authenticated boundary continuation could not be sealed")
	}
	return url.QueryEscape(sealed)
}

func TestServiceQuerySignedMatchExhaustionNeverPublishesAnOverflowedPage(t *testing.T) {
	for _, area := range []string{"accounts", "credentials", "owners", "organizations", "audiences", "scopes", "history"} {
		t.Run(area, func(t *testing.T) {
			f := setupWith(t, func(c *Config) { c.ServiceCatalog = serviceQueryBoundaryCatalog(t) })
			root := f.as(AdminLogin)
			service := f.newService(root, "alpha")
			other := f.newService(root, "zeta")
			f.issued(root, service, "1")
			f.issued(root, other, "1")
			changed := f.run(root, "service.declare", service, f.declaredFields(map[string]string{"declaration_version": "1", "purpose": "Second retained declaration"}))
			if changed.status != 200 {
				t.Fatal("retained history premise", changed.status)
			}
			paths := map[string]string{
				"accounts": "/api/v1/identity/services?limit=1", "credentials": "/api/v1/identity/service-credentials?limit=1",
				"owners": "/api/v1/identity/service-owners?limit=1", "organizations": "/api/v1/identity/service-organizations?limit=1",
				"audiences": "/api/v1/identity/service-audiences?limit=1", "scopes": "/api/v1/identity/service-audiences/test-service-api/scopes?limit=1",
				"history": "/api/v1/identity/services/" + service + "/declarations?limit=1",
			}
			path := paths[area]
			after := saturatedServiceContinuation(t, f, f.serviceQuery(root, path), func(cursor *serviceCursor) { cursor.Matched = math.MaxUint64 })
			assertServiceQueryFailurePayload(t, f.serviceQuery(root, path+"&after="+after), 503, commands.DependencyUnavailable)
		})
	}
}

func TestServiceQuerySignedCompletionBeyondSourceRefusesBothListAreas(t *testing.T) {
	for _, path := range []string{"/api/v1/identity/services?limit=1", "/api/v1/identity/service-credentials?limit=1"} {
		t.Run(path, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			first := f.newService(root, "alpha")
			second := f.newService(root, "zeta")
			f.issued(root, first, "1")
			f.issued(root, second, "1")
			after := saturatedServiceContinuation(t, f, f.serviceQuery(root, path), func(cursor *serviceCursor) { cursor.Completed = math.MaxUint64 })
			assertServiceQueryFailurePayload(t, f.serviceQuery(root, path+"&after="+after), 409, commands.RevisionConflict)
		})
	}
}

func TestServiceQuerySignedAggregateExhaustionRefusesRemainingCredential(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.PageSize = 1
		c.Access = DefaultAccessSettings()
		c.Access.ServiceQueryScan = 100
		c.Identity.CredentialsPerServiceUser = 200
	})
	root := f.as(AdminLogin)
	service := f.newService(root, "aggregate")
	addServiceQueryBoundaryCredentials(f, root, service, 100, 0)
	path := "/api/v1/identity/services?limit=100"
	after := saturatedServiceContinuation(t, f, f.serviceQuery(root, path), func(cursor *serviceCursor) {
		if cursor.PendingService != service || cursor.AfterCredential == "" || cursor.Aggregate.Total != 99 {
			t.Fatal("continuation did not retain a real unfinished aggregate")
		}
		cursor.Aggregate.Total = math.MaxUint64
		cursor.Aggregate.Active = math.MaxUint64
	})
	assertServiceQueryFailurePayload(t, f.serviceQuery(root, path+"&after="+after), 503, commands.DependencyUnavailable)
}

func TestServiceQueryAuthenticatedMalformedHistoryPositionNeverPublishesHistory(t *testing.T) {
	for _, position := range []string{"letters", "0", "01", "18446744073709551616"} {
		t.Run(position, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			service := f.newService(root, "position")
			changed := f.run(root, "service.declare", service, f.declaredFields(map[string]string{"declaration_version": "1", "purpose": "Second retained declaration"}))
			if changed.status != 200 {
				t.Fatal("retained version premise", changed.status)
			}
			path := "/api/v1/identity/services/" + service + "/declarations?limit=1"
			after := saturatedServiceContinuation(t, f, f.serviceQuery(root, path), func(cursor *serviceCursor) { cursor.After = position })
			assertServiceQueryFailurePayload(t, f.serviceQuery(root, path+"&after="+after), 409, commands.RevisionConflict)
		})
	}
}
