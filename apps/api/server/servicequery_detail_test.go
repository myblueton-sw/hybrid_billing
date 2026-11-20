package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestServiceQueryKnownDetailSourceChangeIsAnExplicitRefresh(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "known")
	actor := root.verifier(f)
	changed := false
	f.s.fault = func(at string) bool {
		if at == "services" && !changed {
			changed = true
			f.must(f.s.identity.Workspace().AddCredential(actor, SampleWorkspace, svc, "changed", identity.Verifier{7}, identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
		}
		return false
	}
	got := f.serviceQuery(root, "/api/v1/identity/services/"+svc)
	if !changed {
		t.Fatal("source mutation did not execute")
	}
	if got.status != 409 || got.body["code"] != string(commands.RevisionConflict) || got.body["row"] != nil {
		t.Fatal("changed detail returned internal error or mixed-source row", got.status, got.body["code"])
	}
}

func TestServiceQueryAdmissionCountIsExactAndIndependentOfFilters(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	first := f.newService(root, "alpha")
	second := f.newService(root, "bravo")
	if got := f.run(root, "service.retire", second, nil); got.status != 200 {
		t.Fatal("retirement premise", got.status)
	}
	got := f.serviceQuery(root, "/api/v1/identity/services?q=no-match")
	if got.status != 200 {
		t.Fatal("filtered admission query", got.status)
	}
	if got.body["count"].(map[string]any)["value"] != "0" {
		t.Fatal("filtered-count premise")
	}
	admission, ok := got.body["admission"].(map[string]any)
	if !ok || admission["recorded"] != "2" || admission["certainty"] != "exact" || admission["limit"] != "200" {
		t.Fatal("filtered result was substituted for recorded-account admission count")
	}
	detail := f.serviceQuery(root, "/api/v1/identity/services/"+first)
	if detail.status != 200 {
		t.Fatal("known admission detail", detail.status)
	}
	exact, ok := detail.body["admission"].(map[string]any)
	if !ok || exact["recorded"] != "2" || exact["certainty"] != "exact" {
		t.Fatal("detail omitted independent admission count")
	}
}

func TestServiceQueryCurrentAffordancesRespectRealCommandPrerequisites(t *testing.T) {
	for _, scenario := range []string{"operator", "capacity", "hold", "reauthentication", "stranded"} {
		t.Run(scenario, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			svc := f.newService(root, "control")
			switch scenario {
			case "operator":
				f.must(f.s.identity.Installation().SuspendUser(seedOperator, svc, 1))
			case "capacity":
				for i := 0; i < f.s.cfg.Identity.CredentialsPerServiceUser; i++ {
					f.issued(root, svc, "1")
				}
			case "hold":
				f.s.enterHold("synthetic-query-control")
			case "reauthentication":
				f.advance(stepUpDefault + time.Nanosecond)
			case "stranded":
				f.s.stranded = []uint64{1}
			}
			got := f.serviceQuery(root, "/api/v1/identity/services/"+svc)
			if got.status != 200 {
				t.Fatal("known control read", got.status, got.body["code"])
			}
			eligibility := got.body["eligibility"].(map[string]any)
			if scenario == "operator" {
				if eligibility["can_reinstate"] != false {
					t.Fatal("operator-blocked service advertised reinstatement")
				}
			} else if eligibility["can_issue"] != false {
				t.Fatal("blocked issuance advertised as available", scenario)
			}
			if scenario == "hold" || scenario == "reauthentication" || scenario == "stranded" {
				if eligibility["can_declare"] != false || eligibility["can_reinstate"] != false {
					t.Fatal("blocked expanding commands advertised as available", scenario)
				}
			}
			if scenario == "reauthentication" && eligibility["step_up_state"] != "reauthentication_required" {
				t.Fatal("reauthentication state was not distinguished")
			}
		})
	}
}
