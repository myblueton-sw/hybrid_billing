package server

import (
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func setupContextRead(f world, who actor, id, on string) reply {
	return documentRead(f, who, "issuance_setup.context", id, map[string]string{"on": on})
}

func setupContextWorld(t *testing.T) (world, actor) {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	grantSetupInspectionForTest(f, true)
	return f, f.as(MemberLogin)
}

func TestFinancialSetupContextEffectiveContractAndCurrentProfile(t *testing.T) {
	f, member := setupContextWorld(t)
	revision := f.s.Revision()
	early := setupContextRead(f, member, "sample-contract-01", "2026-06-30")
	late := setupContextRead(f, member, "sample-contract-01", "2026-07-01")
	for _, got := range []reply{early, late} {
		if got.status != 200 {
			t.Fatal(got.raw)
		}
		fields, ok := got.body["fields"].(map[string]any)
		if !ok || len(fields) != 16 {
			t.Fatal("unexpected disclosure fields", got.raw)
		}
		for key, want := range map[string]string{"billing_profile": "sample-profile-north", "billing_revision": "1", "profile_basis": "current", "current_eligibility": "unchecked", "currency": "USD"} {
			if field(got, key) != want {
				t.Fatal("incorrect setup context", key, got.raw)
			}
		}
	}
	if field(early, "contract_revision") != "1" || field(late, "contract_revision") != "2" || field(early, "scope_digest") == field(late, "scope_digest") {
		t.Fatal("effective revision ignored")
	}
	again := setupContextRead(f, member, "sample-contract-01", "2026-06-30")
	if field(again, "scope_digest") != field(early, "scope_digest") {
		t.Fatal("unstable scope digest")
	}
	a := f.s.financialIssuance
	if f.s.Revision() != revision || len(a.configurations) != 0 || len(a.configurationScopes) != 0 {
		t.Fatal("read reserved setup")
	}
	profile, version, err := f.s.orgs.BillingProfile(SampleWorkspace, "sample-profile-north")
	f.must(err)
	profile.State = organizations.Suspended
	version, err = f.s.orgs.PutBillingProfile(profile, version)
	f.must(err)
	f.code(t, "current suspended profile", setupContextRead(f, member, "sample-contract-01", "2026-06-30"), 409, string(commands.EvidenceStale))
	profile.State = organizations.Active
	_, err = f.s.orgs.PutBillingProfile(profile, version)
	f.must(err)
	changed := setupContextRead(f, member, "sample-contract-01", "2026-06-30")
	if changed.status != 200 || field(changed, "billing_revision") != "3" || field(changed, "currency") != "USD" || field(changed, "scope_digest") == field(early, "scope_digest") {
		t.Fatal("current profile basis not bound", changed.raw)
	}
}

func TestFinancialSetupContextRefusals(t *testing.T) {
	for _, mode := range []string{"date", "missing", "metadata", "profile", "party", "lifecycle", "organizations", "store", "fault", "suspended", "ended", "not_started", "name_overflow", "budget"} {
		t.Run(mode, func(t *testing.T) {
			f, member := setupContextWorld(t)
			id, on := "sample-contract-01", "2026-11-16"
			status, code := 409, commands.EvidenceStale
			switch mode {
			case "date":
				on = "2026-02-30"
				status, code = 422, commands.InvalidRequest
			case "missing":
				id = "missing-contract"
				status, code = 403, commands.PermissionDenied
			case "metadata":
				f.s.contracts.get(SampleWorkspace, id).workspace = "foreign"
			case "profile":
				f.s.contracts.get(SampleWorkspace, id).profile = "sample-profile-south"
			case "party":
				p, v, err := f.s.orgs.Party(SampleWorkspace, "sample-le-north")
				f.must(err)
				p.State = organizations.Suspended
				_, err = f.s.orgs.PutParty(p, v)
				f.must(err)
			case "lifecycle":
				f.s.lifecycles = nil
				status, code = 503, commands.DependencyUnavailable
			case "organizations":
				f.s.orgs = nil
				status, code = 503, commands.DependencyUnavailable
			case "store":
				f.s.contracts = nil
				status, code = 503, commands.DependencyUnavailable
			case "fault":
				f.s.fault = func(at string) bool { return at == "contract-read" }
				status, code = 503, commands.DependencyUnavailable
			case "suspended":
				id = "sample-contract-02"
				status, code = 422, commands.PrerequisiteFailed
			case "ended":
				id = "sample-contract-03"
				status, code = 422, commands.PrerequisiteFailed
			case "not_started":
				on = "2025-12-31"
				status, code = 422, commands.PrerequisiteFailed
			case "name_overflow":
				f.s.contracts.get(SampleWorkspace, id).name = strings.Repeat("x", 4097)
				status, code = 422, commands.PrerequisiteFailed
			case "budget":
				f.s.cfg.Commands.Value = 4095
				status, code = 501, commands.UnsupportedCapability
			}
			before := f.s.Revision()
			got := setupContextRead(f, member, id, on)
			f.code(t, mode, got, status, string(code))
			if got.body["fields"] != nil || f.s.Revision() != before || len(f.s.financialIssuance.configurations) != 0 {
				t.Fatal("refusal leaked fields or changed setup")
			}
		})
	}
}

func TestFinancialSetupContextAuthorizationAndHold(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, member := f.as(AdminLogin), f.as(MemberLogin)
	f.s.fault = func(at string) bool { return at == "contract-read" }
	for _, who := range []actor{root, member} {
		for _, id := range []string{"sample-contract-01", "missing-contract"} {
			f.code(t, "permission before lookup", setupContextRead(f, who, id, "2026-11-16"), 403, string(commands.PermissionDenied))
		}
	}
	f.s.fault = nil
	grantSetupInspectionForTest(f, true)
	before := f.s.Revision()
	f.s.enterHold("Synthetic setup context hold")
	got := setupContextRead(f, member, "sample-contract-01", "2026-11-16")
	if got.status != 200 || f.s.Revision() != before {
		t.Fatal("held read", got.raw)
	}
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "revoked held read", setupContextRead(f, member, "sample-contract-01", "2026-11-16"), 403, string(commands.PermissionDenied))
	f.s.endProbe(member.verifier(f))
	f.code(t, "ended session", setupContextRead(f, member, "sample-contract-01", "2026-11-16"), 401, string(commands.Unauthenticated))
	f.s.cfg.Seed = SeedNone
	f.code(t, "non sample", setupContextRead(f, root, "sample-contract-01", "2026-11-16"), 501, string(commands.UnsupportedCapability))
}
