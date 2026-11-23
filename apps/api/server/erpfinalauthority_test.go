package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"strings"
	"testing"
)

func TestERPFinalAuthorityStageIsolation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	first, second := f.as(AdminLogin), f.as(Admin2Login)
	grants := len(f.s.admin.Grants())
	p := f.run(first, "erp_final_authority.propose", "final-limit", financialAuthorityFields(f))
	f.ok(t, "final proposal", p, approvals.StatePending, "")
	if field(p, "stage") != approvals.StageERPFinal || field(p, "authority") != "erp-final-limit-final-limit" {
		t.Fatal("wrong stage or authority")
	}
	fields := map[string]string{"digest": field(p, "digest")}
	f.code(t, "self confirmation", f.run(first, "erp_final_authority.confirm", "final-limit", fields), 403, "self_approval_denied")
	f.code(t, "charge cannot inspect", documentRead(f, first, "financial_authority.inspect", "final-limit", map[string]string{}), 403, "permission_denied")
	f.code(t, "charge cannot adopt", f.run(first, "financial_authority.propose", "final-limit", financialAuthorityFields(f)), 409, "evidence_stale")
	q := f.run(first, "financial_authority.propose", "charge-limit", financialAuthorityFields(f))
	f.ok(t, "charge proposal", q, approvals.StatePending, "")
	f.code(t, "final cannot adopt", f.run(first, "erp_final_authority.propose", "charge-limit", financialAuthorityFields(f)), 409, "evidence_stale")
	tally, err := erpFinalAuthorityTally(f.s)
	f.must(err)
	if tally[SampleAdmin] != 1 {
		t.Fatal("wrong pending impact")
	}
	c := f.run(second, "erp_final_authority.confirm", "final-limit", fields)
	f.ok(t, "independent confirmation", c, approvals.StateConfirmed, "")
	if field(c, "module_amount_authority") != "active" || len(f.s.admin.Grants()) != grants {
		t.Fatal("wrong authority or gate effect")
	}
	tally, err = erpFinalAuthorityTally(f.s)
	f.must(err)
	if len(tally) != 0 {
		t.Fatal("confirmed counted pending")
	}
	f.s.enterHold("final limit verification")
	f.code(t, "held increase", f.run(first, "erp_final_authority.propose", "held-limit", financialAuthorityFields(f)), 503, "dependency_unavailable")
	f.ok(t, "held reduction", f.run(second, "erp_final_authority.revoke", "final-limit", fields), "revoked", "")
	r := documentRead(f, first, "erp_final_authority.inspect", "final-limit", map[string]string{})
	if r.status != 200 || field(r, "module_amount_authority") != "revoked" {
		t.Fatal("revocation not retained")
	}
	_, err = f.s.financialAuthorityDomain(financialPolicyStage(99))
	documentFailure(t, err, commands.UnsupportedCapability)
	f.s.erpFinalAuthorities["final-limit"] = financialAuthorityBinding{}
	if _, err := erpFinalAuthorityTally(f.s); err == nil {
		t.Fatal("corrupt impact accepted")
	}
}

func TestFinancialAuthorityCachedConfirmationAndReduction(t *testing.T) {
	for _, prefix := range []string{"financial_authority", "erp_final_authority"} {
		for _, loss := range []string{"proposer", "recipient"} {
			t.Run(prefix+"/"+loss, func(t *testing.T) {
				f := setup(t, SeedSample).ready()
				first, second := f.as(AdminLogin), f.as(Admin2Login)
				p := f.run(first, prefix+".propose", "cached-limit", financialAuthorityFields(f))
				f.ok(t, "proposal", p, approvals.StatePending, "")
				fields := map[string]string{"digest": field(p, "digest")}
				body := chargeBody(f, "limit-confirmation-key", "cached-limit", fields)
				f.ok(t, "confirm", f.call(prefix+".confirm", body, second.cookie, second.csrf), approvals.StateConfirmed, "")
				if loss == "recipient" {
					f.must(f.s.identity.Workspace().SuspendMembership(second.verifier(f), SampleWorkspace, SampleAdmin3, f.membership(SampleAdmin3).Version))
				} else {
					for _, g := range f.s.admin.Grants() {
						if g.SubjectID == SampleAdmin && g.Terms.Action == gate.AdministerInstallation {
							f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						}
					}
				}
				f.code(t, "cached confirmation rechecks", f.call(prefix+".confirm", body, second.cookie, second.csrf), 403, "permission_denied")
				f.ok(t, "revoke despite eligibility loss", f.run(second, prefix+".revoke", "cached-limit", fields), "revoked", "")
			})
		}
	}
}

func TestERPFinalAuthorityDerivedIDCollisionAndLength(t *testing.T) {
	f := setup(t, SeedSample).ready()
	first := f.as(AdminLogin)
	spec, err := f.s.financialAuthoritySpecForStage(commands.Context{Target: "collision", Fields: financialAuthorityFields(f)}, erpFinalPolicyStage)
	f.must(err)
	_, err = f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "module-only-limit", Kind: approvals.ProposeGrant, Authority: spec}, approvals.Actor{ID: SampleAdmin, Kind: approvals.KindHuman})
	f.must(err)
	_, err = f.s.approvals.Confirm(SampleWorkspace, "module-only-limit", approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	f.must(err)
	f.code(t, "derived authority cannot be adopted", f.run(first, "erp_final_authority.propose", "collision", financialAuthorityFields(f)), 409, "evidence_stale")
	f.code(t, "derived ID bounded", f.run(first, "erp_final_authority.propose", strings.Repeat("x", 113), financialAuthorityFields(f)), 422, "invalid_request")
	if len(f.s.erpFinalAuthorities) != 0 {
		t.Fatal("collision or invalid ID changed ownership")
	}
}
