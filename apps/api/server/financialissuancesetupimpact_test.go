package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func issuanceSetupPending(t *testing.T, f world, a actor, bulk string) (int, any) {
	t.Helper()
	for _, raw := range f.bulkRead(a, "/"+bulk).body["impact"].(map[string]any)["pending"].([]any) {
		if row := raw.(map[string]any); row["store"] == "issuance_setup_proposals" {
			return whole(row["count"]), row["rechecked"]
		}
	}
	t.Fatal("issuance setup proposals omitted from impact")
	return 0, nil
}

func issuanceSetupImpact(t *testing.T, f world, a actor, target string) (int, any) {
	t.Helper()
	staged := f.stage(a, "", target)
	f.ok(t, "stage", staged, "draft", "")
	p := f.prepare(a, field(staged, "draft"), "suspend_membership")
	f.ok(t, "prepare impact", p, "prepared", "")
	return issuanceSetupPending(t, f, a, field(p, "bulk"))
}

func TestIssuanceSetupProposalsCountInBulkImpact(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	_, err := proposeConfigurationAs(f, f.as(Admin2Login), configurationSpec())
	f.must(err)
	if count, rechecked := issuanceSetupImpact(t, f, root, SampleAdmin2); count != 1 || rechecked != true {
		t.Fatal("pending proposal not counted for its proposer", count, rechecked)
	}
	if count, _ := issuanceSetupImpact(t, f, root, SampleAdmin3); count != 0 {
		t.Fatal("pending proposal counted for another user", count)
	}

	g := setup(t, SeedSample).ready()
	root = g.as(AdminLogin)
	p, err := proposeConfigurationAs(g, g.as(Admin2Login), configurationSpec())
	g.must(err)
	staged := g.stage(root, "", SampleAdmin2)
	before := g.s.Revision()
	g.s.fault = func(at string) bool { return at == "issuance-setup-impact" }
	g.code(t, "impact fails closed", g.prepare(root, field(staged, "draft"), "suspend_membership"), 503, string(commands.DependencyUnavailable))
	g.s.fault = nil
	saved := g.s.financialIssuance.configurations[p.Spec.ID]
	corrupt := saved
	corrupt.State = "unrecognized"
	g.s.financialIssuance.configurations[p.Spec.ID] = corrupt
	g.code(t, "unknown state fails closed", g.prepare(root, field(staged, "draft"), "suspend_membership"), 503, string(commands.DependencyUnavailable))
	g.s.financialIssuance.configurations[p.Spec.ID] = saved
	if g.s.Revision() != before {
		t.Fatal("impact failure mutated revision")
	}
	_, err = confirmConfigurationAs(g, g.as(Admin3Login), p)
	g.must(err)
	prepared := g.prepare(root, field(staged, "draft"), "suspend_membership")
	g.ok(t, "prepare impact", prepared, "prepared", "")
	if count, _ := issuanceSetupPending(t, g, root, field(prepared, "bulk")); count != 0 {
		t.Fatal("confirmed configuration still counted", count)
	}
}

func TestIssuanceSetupUncertainProposalsCountInBulkImpact(t *testing.T) {
	for _, point := range []string{"propose-after-intent", "confirm-after-intent"} {
		t.Run(point, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			var p financialIssuanceConfiguration
			var err error
			if point == "confirm-after-intent" {
				p, err = proposeConfigurationAs(f, f.as(Admin2Login), configurationSpec())
				f.must(err)
			}
			f.s.fault = func(at string) bool { return at == "issuance-configuration-"+point }
			if point == "confirm-after-intent" {
				_, err = confirmConfigurationAs(f, f.as(Admin3Login), p)
			} else {
				_, err = proposeConfigurationAs(f, f.as(Admin2Login), configurationSpec())
			}
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			want := map[string]string{"propose-after-intent": "proposal_unknown", "confirm-after-intent": "confirmation_unknown"}[point]
			if got := f.s.financialIssuance.configurations[configurationSpec().ID].State; got != want {
				t.Fatalf("fixture state %s, want %s", got, want)
			}
			if count, _ := issuanceSetupImpact(t, f, root, SampleAdmin2); count != 1 {
				t.Fatal("uncertain proposal not counted for its proposer", count)
			}
		})
	}
}
