package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestOwnedFinalAuthorityMustCoverEntireAmount(t *testing.T) {
	f := setup(t, SeedSample).ready()
	first, second := f.as(AdminLogin), f.as(Admin2Login)
	check := func(amount string) error {
		return f.s.currentOwnedFinancialAuthorityCoverage(SampleWorkspace, SampleAdmin3, erpFinalPolicyStage, "USD", amount)
	}
	documentFailure(t, check("10"), commands.PrerequisiteFailed)
	for _, id := range []string{"small-one", "small-two"} {
		fields := financialAuthorityFields(f)
		fields["limit"] = "5"
		p := f.run(first, "erp_final_authority.propose", id, fields)
		f.ok(t, "small proposal", p, approvals.StatePending, "")
		f.ok(t, "small confirmation", f.run(second, "erp_final_authority.confirm", id, map[string]string{"digest": field(p, "digest")}), approvals.StateConfirmed, "")
	}
	spec, err := f.s.financialAuthoritySpecForStage(commands.Context{Target: "unowned", Fields: financialAuthorityFields(f)}, erpFinalPolicyStage)
	f.must(err)
	_, err = f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "unowned", Kind: approvals.ProposeGrant, Authority: spec}, approvals.Actor{ID: SampleAdmin, Kind: approvals.KindHuman})
	f.must(err)
	_, err = f.s.approvals.Confirm(SampleWorkspace, "unowned", approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	f.must(err)
	documentFailure(t, check("10"), commands.PrerequisiteFailed)
	f.must(check("5"))
	p := f.run(first, "erp_final_authority.propose", "sufficient", financialAuthorityFields(f))
	f.ok(t, "covering proposal", p, approvals.StatePending, "")
	digest := map[string]string{"digest": field(p, "digest")}
	f.ok(t, "covering confirmation", f.run(second, "erp_final_authority.confirm", "sufficient", digest), approvals.StateConfirmed, "")
	f.must(check("10"))
	f.s.erpFinalAuthorities["corrupt"] = financialAuthorityBinding{}
	if check("10") == nil {
		t.Fatal("covering authority hid corrupt binding")
	}
	delete(f.s.erpFinalAuthorities, "corrupt")
	f.ok(t, "revoke owned coverage", f.run(second, "erp_final_authority.revoke", "sufficient", digest), "revoked", "")
	documentFailure(t, check("10"), commands.PrerequisiteFailed)
}
