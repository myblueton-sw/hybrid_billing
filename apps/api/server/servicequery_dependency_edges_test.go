package server

import (
	"math"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestServiceQueryLateDeclarationEvidenceLossCannotInventExpandingEligibility(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id := f.newService(root, "late-evidence")
	verifier := root.verifier(f)
	service, err := f.s.identity.Workspace().ServiceUser(verifier, SampleWorkspace, id)
	f.must(err)
	if before := f.s.currentServiceEligibility(verifier, SampleAdmin, service, f.s.clock()); before.State != "eligible" || !before.CanIssue {
		t.Fatal("current declared eligibility premise")
	}
	f.s.fault = func(at string) bool { return at == "organizations" }
	got := f.s.currentServiceEligibility(verifier, SampleAdmin, service, f.s.clock())
	if got.State != "unavailable" || got.CanIssue || got.CanReinstate || !got.CanSuspend || !got.CanRetire {
		t.Fatal("late missing declaration evidence changed safe reduction or expansion meaning")
	}
}

func TestServiceQueryExhaustedDeliveryGenerationIsUnavailableOnlyForCredentialReads(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	service := f.newService(root, "delivery-boundary")
	credential, _, _ := f.issued(root, service, "1")
	f.s.dir.deliveryVersion = math.MaxUint64
	for _, path := range []string{"/api/v1/identity/service-credentials", "/api/v1/identity/service-credentials/" + credential} {
		assertServiceQueryFailurePayload(t, f.serviceQuery(root, path), 503, commands.DependencyUnavailable)
	}
	if f.s.dir.deliveryVersion != math.MaxUint64 {
		t.Fatal("read wrapped or repaired an exhausted delivery generation")
	}
	if f.serviceQuery(root, "/api/v1/identity/services/"+service).status != 200 {
		t.Fatal("credential generation exhaustion incorrectly erased account metadata")
	}
}

func TestServiceQueryInstallerSettingListKeepsIndependentRowsAndWorkingSetters(t *testing.T) {
	rows := ServiceQuerySettingList()
	if len(rows) == 0 {
		t.Fatal("installer query settings are missing")
	}
	first := rows[0]
	names, flags := map[string]bool{}, map[string]bool{}
	for _, row := range rows {
		if row.Name == "" || row.Flag == "" || row.Unit == "" || names[row.Name] || flags[row.Flag] || row.Min <= 0 || row.Default < row.Min || row.Default > row.Max {
			t.Fatal("installer setting contract is incomplete or inconsistent")
		}
		names[row.Name], flags[row.Flag] = true, true
		settings := DefaultAccessSettings()
		row.Set(&settings, row.Min)
		if row.Value(settings) != row.Min {
			t.Fatal("installer setter does not select its configured field", row.Name)
		}
		row.Set(&settings, row.Default)
		if row.Value(settings) != row.Default {
			t.Fatal("installer default did not restore its field", row.Name)
		}
	}
	rows[0].Name = "caller-edited-copy"
	rows[0].Min = 0
	fresh := ServiceQuerySettingList()
	if fresh[0].Name != first.Name || fresh[0].Min != first.Min {
		t.Fatal("caller mutation changed the global installer setting registry")
	}
}
