package server

import (
	"testing"
	"time"
)

func TestServiceQueryKnownCredentialAffordanceUsesCurrentRevokePreconditions(t *testing.T) {
	for _, scenario := range []string{"ready", "expired", "revoked", "held", "stranded", "reauthentication"} {
		t.Run(scenario, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			svc := f.newService(root, "revoke-control")
			id, _, _ := f.issued(root, svc, "1")
			switch scenario {
			case "expired":
				f.advance(24 * time.Hour)
				root = f.as(AdminLogin)
			case "revoked":
				got := f.run(root, "service.revoke_credential", id, nil)
				if got.status != 200 {
					t.Fatal("revocation premise", got.status)
				}
			case "held":
				f.s.enterHold("synthetic-credential-control")
			case "stranded":
				f.s.stranded = []uint64{1}
			case "reauthentication":
				f.advance(stepUpDefault + time.Nanosecond)
			}
			got := f.serviceQuery(root, "/api/v1/identity/service-credentials/"+id)
			if got.status != 200 {
				t.Fatal("canonical credential control read", got.status, got.body["code"])
			}
			eligibility, ok := got.body["eligibility"].(map[string]any)
			expected := scenario == "ready" || scenario == "expired" || scenario == "held"
			if !ok || eligibility["can_revoke"] != expected {
				t.Fatal("credential affordance omitted real current preconditions", scenario)
			}
			if scenario == "reauthentication" && eligibility["step_up_state"] != "reauthentication_required" {
				t.Fatal("credential reauthentication state missing")
			}
		})
	}
}
