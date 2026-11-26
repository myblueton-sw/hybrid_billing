package server

import (
	"maps"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func setupOptionsRead(f world, who actor, on, digest string) reply {
	return documentRead(f, who, "issuance_setup.options", "sample-contract-01", map[string]string{"on": on, "scope_digest": digest})
}

func TestFinancialSetupOptionsScopeAndHold(t *testing.T) {
	f, member := setupContextWorld(t)
	on := "2026-11-16"
	context := setupContextRead(f, member, "sample-contract-01", on)
	digest := field(context, "scope_digest")
	before := f.s.Revision()
	for _, held := range []bool{false, true} {
		if held {
			f.s.enterHold("Synthetic options read test")
		}
		got := setupOptionsRead(f, member, on, digest)
		if got.status != 200 {
			t.Fatal(got.raw)
		}
		for key, want := range map[string]string{"option": "synthetic_erp_sample_v1", "applicability": "synthetic_only", "current_eligibility": "unchecked", "region": "US", "tax": "synthetic-tax", "zone": "UTC", "languages": "ar,en,ko", "scope_digest": digest} {
			if field(got, key) != want {
				t.Fatal(key, got.raw)
			}
		}
		if f.s.Revision() != before || len(f.s.financialIssuance.configurations) != 0 || len(f.s.financialIssuance.configurationScopes) != 0 {
			t.Fatal("options mutated setup")
		}
	}
	f.code(t, "stale digest", setupOptionsRead(f, member, on, strings.Repeat("a", 64)), 409, string(commands.EvidenceStale))
	f.code(t, "invalid digest", setupOptionsRead(f, member, on, "bad"), 422, string(commands.InvalidRequest))
	f.code(t, "changed date", setupOptionsRead(f, member, "2026-06-30", digest), 409, string(commands.EvidenceStale))
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "revoked read", setupOptionsRead(f, member, on, digest), 403, string(commands.PermissionDenied))
}

func TestFinancialSetupOptionsNewIdentityAdmission(t *testing.T) {
	f, root, _, _, fields := setupMutationWorld(t)
	before := f.s.Revision()
	for key, value := range map[string]string{"region": "KR", "tax": "other-tax", "zone": "Asia/Seoul"} {
		changed := maps.Clone(fields)
		changed[key] = value
		got := draftRun(f, root, "issuance_setup.propose_configuration", "nonpreset-"+key, changed)
		f.code(t, key, got, 501, string(commands.UnsupportedCapability))
		if f.s.Revision() != before || len(f.s.financialIssuance.configurations) != 0 || len(f.s.financialIssuance.configurationScopes) != 0 {
			t.Fatal("unsupported option reserved state")
		}
	}
	f.ok(t, "advertised option", draftRun(f, root, "issuance_setup.propose_configuration", "preset", fields), "proposed", "")
}

func TestFinancialSetupOptionsAuthorityBeforeUnavailableSource(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.fault = func(at string) bool { return at == "contract-read" }
	f.code(t, "inspection required before source", setupOptionsRead(f, root, "2026-11-16", strings.Repeat("a", 64)), 403, string(commands.PermissionDenied))
	f, member := setupContextWorld(t)
	context := setupContextRead(f, member, "sample-contract-01", "2026-11-16")
	before := f.s.Revision()
	f.s.fault = func(at string) bool { return at == "contract-read" }
	got := setupOptionsRead(f, member, "2026-11-16", field(context, "scope_digest"))
	f.code(t, "unavailable source", got, 503, string(commands.DependencyUnavailable))
	if got.body["fields"] != nil || f.s.Revision() != before || len(f.s.financialIssuance.configurations) != 0 {
		t.Fatal("unavailable source disclosed options or changed state")
	}
}

func TestFinancialSetupOptionsRetainedNonpresetRecovery(t *testing.T) {
	f, root, second, _, fields := setupMutationWorld(t)
	fields["tax"] = "retained-tax"
	a := f.s.financialIssuance
	f.must(inputAuthorityAs(f, root, func() error {
		spec, setup, err := a.setupProposalSpec(SampleWorkspace, SampleAdmin, "retained-nonpreset", fields)
		if err != nil {
			return err
		}
		f.s.fault = func(at string) bool { return at == "issuance-configuration-propose-after-intent" }
		_, err = a.proposeConfigurationTracked(SampleWorkspace, SampleAdmin, "Synthetic financial review", spec, setup, nil, nil)
		if err == nil {
			t.Fatal("interruption did not occur")
		}
		return nil
	}))
	f.s.fault = nil
	f.code(t, "wrong actor", draftRun(f, second, "issuance_setup.propose_configuration", "retained-nonpreset", fields), 403, string(commands.PermissionDenied))
	got := draftRun(f, root, "issuance_setup.propose_configuration", "retained-nonpreset", fields)
	f.ok(t, "owned retained recovery", got, "proposed", "")
	if field(got, "tax") != "retained-tax" {
		t.Fatal("retained setting rewritten")
	}
	f.ok(t, "independent retained confirmation", draftRun(f, second, "issuance_setup.confirm_configuration", "retained-nonpreset", map[string]string{"digest": field(got, "digest")}), "confirmed", "")
}
