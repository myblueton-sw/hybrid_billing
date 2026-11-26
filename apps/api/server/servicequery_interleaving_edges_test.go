package server

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func serviceQueryRecordedFailure(t testing.TB, w *httptest.ResponseRecorder) reply {
	t.Helper()
	var body map[string]any
	if json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatal("query failure response encoding")
	}
	return reply{status: w.Code, header: w.Header(), body: body, raw: w.Body.String()}
}

func TestServiceQueryEarlierPublicInsertionInvalidatesRemainingOrdering(t *testing.T) {
	for _, area := range []string{"accounts", "credentials"} {
		t.Run(area, func(t *testing.T) {
			f := setupWith(t, func(c *Config) {
				c.Identity.PageSize = 1
				c.Access = DefaultAccessSettings()
				c.Access.ServiceQueryScan = 100
				c.Identity.CredentialsPerServiceUser = 200
			})
			root := f.as(AdminLogin)
			first := f.newService(root, "middle")
			last := f.newService(root, "zeta")
			addServiceQueryBoundaryCredentials(f, root, first, 49, 0)
			addServiceQueryBoundaryCredentials(f, root, last, 49, 49)
			path := "/api/v1/identity/services?limit=100"
			if area == "credentials" {
				path = "/api/v1/identity/service-credentials?limit=100"
			}
			before := f.serviceQuery(root, path)
			if before.status != 200 || before.body["next"] == "" {
				t.Fatal("actual unfinished producer page premise", before.status)
			}
			r := serviceWalkRequest(root, path+"&after="+url.QueryEscape(before.body["next"].(string)))
			walk, ok := f.s.beginServiceQuery(httptest.NewRecorder(), r, area, "")
			if !ok || walk.cursor.PendingService != last || walk.cursor.Completed != 1 || walk.cursor.AfterCredential == "" {
				t.Fatal("continuation did not resume the last real service")
			}
			f.must(f.s.identity.Workspace().CreateServiceUser(walk.actor, SampleWorkspace, "svc-aa-before", SampleAdmin))
			var err error
			if area == "accounts" {
				rows, more, failure := walk.accounts()
				err = failure
				if rows != nil || more {
					t.Fatal("changed ordering published partial account rows")
				}
			} else {
				rows, more, failure := walk.credentials()
				err = failure
				if rows != nil || more {
					t.Fatal("changed ordering published partial credential rows")
				}
			}
			if !errors.Is(err, errServiceViewChanged) || walk.total != 3 || walk.cursor.Completed != 2 {
				t.Fatal("earlier insertion was confused with exact completion")
			}
			w := httptest.NewRecorder()
			serviceQueryFailure(w, r, err)
			assertServiceQueryFailurePayload(t, serviceQueryRecordedFailure(t, w), 409, commands.RevisionConflict)
			known := f.serviceQuery(root, "/api/v1/identity/services/svc-aa-before")
			if known.status != 200 || known.body["row"].(map[string]any)["declaration"].(map[string]any)["state"] != "missing" {
				t.Fatal("insertion was not a real public-module account")
			}
		})
	}
}

func TestServiceQueryAuthorizedSourceMutationRefusesStalePublication(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.newService(root, "observed")
	r := serviceWalkRequest(root, "/api/v1/identity/services?limit=100")
	walk, ok := f.s.beginServiceQuery(httptest.NewRecorder(), r, "accounts", "")
	if !ok {
		t.Fatal("admitted account projection premise")
	}
	rows, more, err := walk.accounts()
	if err != nil || more || len(rows) != 1 {
		t.Fatal("complete account projection premise")
	}
	f.must(f.s.identity.Workspace().CreateServiceUser(walk.actor, SampleWorkspace, "svc-new-source", SampleAdmin))
	w := httptest.NewRecorder()
	writeServiceQueryPage(w, r, walk, rows, false, "service_id_asc")
	assertServiceQueryFailurePayload(t, serviceQueryRecordedFailure(t, w), 409, commands.RevisionConflict)
	current := f.serviceQuery(root, "/api/v1/identity/services?limit=100")
	if current.status != 200 || len(current.body["rows"].([]any)) != 2 {
		t.Fatal("authorized source change was mistaken for caller loss")
	}
}

func TestServiceQueryDemotedActualMemberCannotContinueAnAdmittedRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	manager := f.as(AdminLogin)
	admin := manager.verifier(f)
	reader := f.as(Admin2Login)
	f.newService(manager, "role-reader")
	if f.serviceQuery(reader, "/api/v1/identity/services").status != 200 {
		t.Fatal("actual administrator was not admitted")
	}
	people, err := f.s.identity.Workspace().People(admin, SampleWorkspace, []string{SampleAdmin2})
	f.must(err)
	if len(people) != 1 || people[0].Member == nil {
		t.Fatal("actual reader membership premise")
	}
	demoted := false
	f.s.fault = func(at string) bool {
		if at == "services" && !demoted {
			demoted = true
			f.must(f.s.identity.Workspace().ChangeRole(admin, SampleWorkspace, SampleAdmin2, identity.Member, people[0].Member.Version))
		}
		return false
	}
	assertServiceQueryFailurePayload(t, f.serviceQuery(reader, "/api/v1/identity/services"), 503, commands.DependencyUnavailable)
	if !demoted {
		t.Fatal("role transition was not reached")
	}
	f.s.fault = nil
	assertServiceQueryFailurePayload(t, f.serviceQuery(reader, "/api/v1/identity/services"), 403, commands.PermissionDenied)
	if f.serviceQuery(manager, "/api/v1/identity/services/svc-role-reader").status != 200 {
		t.Fatal("reader demotion changed another administrator's access")
	}
}
