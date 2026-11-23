package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestERPFinalPolicyOwnedStageIsolation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	first, second := f.as(AdminLogin), f.as(Admin2Login)
	beforeGrants := len(f.s.admin.Grants())
	_, err := f.s.currentOwnedFinancialPolicyForStage(SampleWorkspace, erpFinalPolicyStage)
	documentFailure(t, err, commands.PrerequisiteFailed)
	p := f.run(first, "erp_final_policy.propose", "final-policy-two", financialFields("2"))
	f.ok(t, "final proposal", p, approvals.StatePending, "")
	if field(p, "stage") != approvals.StageERPFinal {
		t.Fatal("wrong final stage")
	}
	fields := map[string]string{"digest": field(p, "digest")}
	f.code(t, "self confirm", f.run(first, "erp_final_policy.confirm", "final-policy-two", fields), 403, "self_approval_denied")
	f.code(t, "charge cannot inspect final", documentRead(f, first, "financial_policy.inspect", "final-policy-two", map[string]string{}), 403, "permission_denied")
	f.code(t, "shared namespace collision", f.run(first, "financial_policy.propose", "final-policy-two", financialFields("2")), 409, "evidence_stale")
	tally, err := erpFinalPolicyTally(f.s)
	f.must(err)
	if tally[SampleAdmin] != 1 {
		t.Fatal("missing final proposal impact")
	}
	old, err := financialPolicyTally(f.s)
	f.must(err)
	if len(old) != 0 {
		t.Fatal("charge impact included final proposal")
	}
	approved := f.run(second, "erp_final_policy.confirm", "final-policy-two", fields)
	f.ok(t, "independent final policy confirmation", approved, approvals.StateConfirmed, "")
	if field(approved, "stage") != approvals.StageERPFinal || field(approved, "receipt") != "historical_decision" {
		t.Fatal("wrong receipt")
	}
	policy, err := f.s.currentOwnedFinancialPolicyForStage(SampleWorkspace, erpFinalPolicyStage)
	f.must(err)
	if policy.Stage != approvals.StageERPFinal || policy.Revision != 2 {
		t.Fatal(policy)
	}
	charge, err := f.s.approvals.PolicyOf(SampleWorkspace, approvals.StageCharge)
	f.must(err)
	if charge.Revision != 1 || len(f.s.financialPolicies) != 0 || len(f.s.erpFinalPolicies) != 1 || len(f.s.admin.Grants()) != beforeGrants {
		t.Fatal("final policy altered charge or grants")
	}
	tally, err = erpFinalPolicyTally(f.s)
	f.must(err)
	if len(tally) != 0 {
		t.Fatal("confirmed policy counted pending")
	}
	f.code(t, "wrong current target", documentRead(f, first, "erp_final_policy.current", approvals.StageCharge, map[string]string{}), 403, "permission_denied")
	_, err = f.s.financialPolicyDomain(financialPolicyStage(99))
	documentFailure(t, err, commands.UnsupportedCapability)
	if _, err = financialPolicySpecForStage(financialFields("2"), financialPolicyStage(99)); err == nil {
		t.Fatal("unknown stage accepted")
	}
}

func TestFinancialPolicyCachedConfirmationRechecksProposer(t *testing.T) {
	for _, prefix := range []string{"financial_policy", "erp_final_policy"} {
		t.Run(prefix, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			first, second := f.as(AdminLogin), f.as(Admin2Login)
			p := f.run(first, prefix+".propose", "cached-policy", financialFields("2"))
			f.ok(t, "proposal", p, approvals.StatePending, "")
			body := chargeBody(f, "policy-confirmation-key", "cached-policy", map[string]string{"digest": field(p, "digest")})
			f.ok(t, "confirm", f.call(prefix+".confirm", body, second.cookie, second.csrf), approvals.StateConfirmed, "")
			for _, g := range f.s.admin.Grants() {
				if g.SubjectID == SampleAdmin && g.Terms.Action == gate.AdministerInstallation {
					f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
				}
			}
			f.code(t, "proposer before cached receipt", f.call(prefix+".confirm", body, second.cookie, second.csrf), 403, "permission_denied")
		})
	}
}

func TestERPFinalPolicyHoldStepUpAndCorruptImpact(t *testing.T) {
	f := setup(t, SeedSample).ready()
	first, second := f.as(AdminLogin), f.as(Admin2Login)
	p := f.run(first, "erp_final_policy.propose", "guarded-policy", financialFields("2"))
	f.ok(t, "proposal", p, approvals.StatePending, "")
	digest := map[string]string{"digest": field(p, "digest")}
	f.s.enterHold("final-policy-test")
	f.code(t, "held confirmation", f.run(second, "erp_final_policy.confirm", "guarded-policy", digest), 503, "dependency_unavailable")
	got := documentRead(f, second, "erp_final_policy.inspect", "guarded-policy", map[string]string{})
	if got.status != 200 {
		t.Fatal("held historical inspection unavailable")
	}
	f.s.erpFinalPolicies["guarded-policy"] = financialPolicyBinding{}
	if _, err := erpFinalPolicyTally(f.s); err == nil {
		t.Fatal("corrupt final impact accepted")
	}
	g := setup(t, SeedSample).ready()
	a, b := g.as(AdminLogin), g.as(Admin2Login)
	q := g.run(a, "erp_final_policy.propose", "step-up-final", financialFields("2"))
	g.ok(t, "proposal", q, approvals.StatePending, "")
	g.advance(16 * time.Minute)
	g.code(t, "final step-up", g.run(b, "erp_final_policy.confirm", "step-up-final", map[string]string{"digest": field(q, "digest")}), 403, CodeReauthentication)
}

func TestERPFinalPolicyRejectsChargeOwnedProposalID(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	p := f.run(root, "financial_policy.propose", "charge-owned", financialFields("2"))
	f.ok(t, "charge proposal", p, approvals.StatePending, "")
	f.code(t, "final cannot adopt charge", f.run(root, "erp_final_policy.propose", "charge-owned", financialFields("2")), 409, "evidence_stale")
	if len(f.s.erpFinalPolicies) != 0 || len(f.s.financialPolicies) != 1 {
		t.Fatal("collision changed ownership")
	}
}

func TestFinancialPolicyCachedConfirmationRechecksModuleGovernor(t *testing.T) {
	for _, prefix := range []string{"financial_policy", "erp_final_policy"} {
		t.Run(prefix, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			first, second := f.as(AdminLogin), f.as(Admin2Login)
			_, err := f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "add-governor", Kind: approvals.ProposeAddAdmin, Admin: SampleAdmin3}, approvals.Actor{ID: SampleAdmin, Kind: approvals.KindHuman})
			f.must(err)
			_, err = f.s.approvals.Confirm(SampleWorkspace, "add-governor", approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
			f.must(err)
			p := f.run(first, prefix+".propose", "cached-module-policy", financialFields("2"))
			f.ok(t, "proposal", p, approvals.StatePending, "")
			body := chargeBody(f, "module-policy-key", "cached-module-policy", map[string]string{"digest": field(p, "digest")})
			f.ok(t, "confirm", f.call(prefix+".confirm", body, second.cookie, second.csrf), approvals.StateConfirmed, "")
			_, err = f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "remove-governor", Kind: approvals.ProposeRemoveAdmin, Admin: SampleAdmin}, approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
			f.must(err)
			_, err = f.s.approvals.Confirm(SampleWorkspace, "remove-governor", approvals.Actor{ID: SampleAdmin3, Kind: approvals.KindHuman})
			f.must(err)
			f.code(t, "same-key confirmation after proposer module removal", f.call(prefix+".confirm", body, second.cookie, second.csrf), 403, "permission_denied")
		})
	}
}
