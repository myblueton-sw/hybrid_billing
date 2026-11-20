package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestServiceQueryAdmittedComponentsPreserveModuleRefusalAfterActualExpiry(t *testing.T) {
	for _, component := range []string{"account-credentials", "next-service"} {
		t.Run(component, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			id := f.newService(root, "component-expiry")
			f.issued(root, id, "1")
			area, path := "accounts", "/api/v1/identity/services"
			if component == "next-service" {
				area, path = "credentials", "/api/v1/identity/service-credentials"
			}
			r := serviceWalkRequest(root, path)
			control, ok := f.s.beginServiceQuery(httptest.NewRecorder(), r, area, "")
			if !ok {
				t.Fatal("actual control admission premise")
			}
			if component == "account-credentials" {
				complete, err := control.accountCredentials(id)
				if err != nil || !complete || control.cursor.Aggregate.Total != 1 {
					t.Fatal("control component did not read the actual issued credential")
				}
			} else if row, err := control.nextService(); err != nil || row.ID != id {
				t.Fatal("control component did not read the actual account")
			}
			walk, ok := f.s.beginServiceQuery(httptest.NewRecorder(), r, area, "")
			if !ok {
				t.Fatal("actual pre-expiry component admission premise")
			}
			f.advance(48 * time.Hour)
			var failure error
			if component == "account-credentials" {
				complete, err := walk.accountCredentials(id)
				failure = err
				if complete || walk.cursor.Aggregate.Total != 0 {
					t.Fatal("expired component published an exact credential aggregate")
				}
			} else {
				row, err := walk.nextService()
				failure = err
				if row.ID != "" || walk.cursor.PendingService != "" {
					t.Fatal("expired component published or retained an account")
				}
			}
			if failure == nil || walk.scanned != 0 {
				t.Fatal("actual expired actor was not refused before component progress")
			}
			w := httptest.NewRecorder()
			serviceQueryFailure(w, r, failure)
			assertServiceQueryFailurePayload(t, serviceQueryRecordedFailure(t, w), 503, commands.DependencyUnavailable)
			assertServiceQueryFailurePayload(t, f.serviceQuery(root, path), 401, commands.Unauthenticated)
		})
	}
}

func TestServiceQueryCredentialModuleFaultsPublishOnlyDependencyRefusal(t *testing.T) {
	for _, dependency := range []string{"services", "credentials"} {
		t.Run(dependency, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			id := f.newService(root, "module-refusal")
			f.issued(root, id, "1")
			if before := f.serviceQuery(root, "/api/v1/identity/service-credentials"); before.status != 200 || len(before.body["rows"].([]any)) != 1 {
				t.Fatal("healthy actual credential query premise")
			}
			f.s.fault = func(at string) bool { return at == dependency }
			assertServiceQueryFailurePayload(t, f.serviceQuery(root, "/api/v1/identity/service-credentials"), 503, commands.DependencyUnavailable)
			f.s.fault = nil
			if f.serviceQuery(root, "/api/v1/identity/service-credentials").status != 200 {
				t.Fatal("supported module fault changed the actual caller's authority")
			}
		})
	}
}
