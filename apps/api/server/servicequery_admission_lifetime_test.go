package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestServiceQueryLegacyAccountCanRepairDeclarationWithoutInventingIssuanceEligibility(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.must(f.s.identity.Workspace().CreateServiceUser(root.verifier(f), SampleWorkspace, "legacy-known", SampleAdmin))
	got := f.serviceQuery(root, "/api/v1/identity/services/legacy-known")
	if got.status != 200 {
		t.Fatal("known legacy account observation", got.status)
	}
	row := got.body["row"].(map[string]any)
	declaration := row["declaration"].(map[string]any)
	flags := got.body["eligibility"].(map[string]any)
	if declaration["state"] != "missing" || declaration["purpose"] != "" || len(declaration["scopes"].([]any)) != 0 || flags["state"] != "ineligible" || flags["can_declare"] != true || flags["can_issue"] != false {
		t.Fatal("legacy account fabricated metadata or lost declaration repair")
	}
	refused := f.run(root, "service.issue_credential", "legacy-known", map[string]string{"days": "1"})
	if refused.status != 422 || refused.body["code"] != string(commands.PrerequisiteFailed) {
		t.Fatal("missing declaration allowed issuance", refused.status)
	}
	f.assertServiceQueryIDs(root, "/api/v1/identity/service-credentials?service=legacy-known", nil)
}

func TestServiceQueryMFARequirementsRemainDistinctFromPasswordFreshness(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	service := f.newService(root, "mfa-current")
	credential, _, _ := f.issued(root, service, "1")
	f.setPassword(AdminLogin, pw)
	f.directPolicy(authpolicy.Revision{MaxAge: time.Hour, StepUp: []authpolicy.StepUp{{Action: ActionSensitive, MaxAge: time.Hour, MFA: true}}})
	root = f.signedIn(AdminLogin, pw)
	before := f.s.Revision()
	for _, tc := range []struct {
		path string
		keys []string
	}{
		{"/api/v1/identity/services/" + service, []string{"can_issue", "can_reinstate", "can_declare", "can_suspend", "can_retire"}},
		{"/api/v1/identity/service-credentials/" + credential, []string{"can_revoke"}},
	} {
		got := f.serviceQuery(root, tc.path)
		if got.status != 200 {
			t.Fatal("MFA prerequisite erased safe metadata", got.status)
		}
		flags := got.body["eligibility"].(map[string]any)
		if flags["step_up_state"] != "prerequisite_failed" || flags["execution_state"] != "ready" {
			t.Fatal("unsupported factor was confused with stale password or dependency failure")
		}
		for _, key := range tc.keys {
			if value, present := flags[key]; !present || value != false {
				t.Fatal("MFA prerequisite omitted or enabled sensitive action", key)
			}
		}
	}
	if f.s.Revision() != before {
		t.Fatal("MFA observations changed revision")
	}
}

func TestServiceQueryAdmittedReadCannotPublishAfterActualLogout(t *testing.T) {
	for _, detail := range []bool{false, true} {
		t.Run(map[bool]string{false: "list", true: "detail"}[detail], func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			service := f.newService(root, "logout-reader")
			verifier := root.verifier(f)
			r := httptest.NewRequest(http.MethodGet, "/api/v1/identity/services", nil)
			r.AddCookie(root.cookie)
			r.Header.Set(csrfHeader, root.csrf)
			admission := httptest.NewRecorder()
			published := httptest.NewRecorder()
			if detail {
				read, ok := f.s.beginServiceDetail(admission, r, "service_detail", service)
				if !ok {
					t.Fatal("detail admission premise", admission.Code)
				}
				if f.logout(root).status != http.StatusNoContent {
					t.Fatal("actual logout failed")
				}
				read.write(published, r, map[string]any{"row": map[string]any{"id": service}})
				if published.Code != http.StatusUnauthorized {
					t.Fatal("ended session published a projected detail", published.Code)
				}
			} else {
				walk, ok := f.s.beginServiceQuery(admission, r, "services", "")
				if !ok {
					t.Fatal("list admission premise", admission.Code)
				}
				if f.logout(root).status != http.StatusNoContent {
					t.Fatal("actual logout failed")
				}
				writeServiceQueryPage(published, r, walk, []serviceAccountRow{{ID: service}}, false, "service_id_asc")
				if published.Code != http.StatusUnauthorized {
					t.Fatal("ended session published a projected page", published.Code)
				}
			}
			decoder := json.NewDecoder(bytes.NewReader(published.Body.Bytes()))
			var body map[string]any
			if decoder.Decode(&body) != nil || body["code"] != string(commands.Unauthenticated) {
				t.Fatal("ended session did not receive one authentication refusal")
			}
			var extra any
			if decoder.Decode(&extra) != io.EOF {
				t.Fatal("projected data was appended after the authentication refusal")
			}
			for _, key := range []string{"row", "rows", "count", "admission", "eligibility", "catalog_version", "limits"} {
				if _, present := body[key]; present {
					t.Fatal("ended session received projected metadata", key)
				}
			}
			if ready, state := f.s.serviceStepUpState(verifier, SampleAdmin, "service.issue_credential", service); ready || state != "unavailable" {
				t.Fatal("ended original session retained step-up readiness")
			}
			if _, err := f.s.serviceBasis(verifier, "services"); err == nil {
				t.Fatal("ended original session retained source authority")
			}
			if _, err := f.s.serviceAdmission(verifier); err == nil {
				t.Fatal("ended original session retained admission authority")
			}
		})
	}
}
