package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func TestFinancialPermissionUnseeded(t *testing.T) {
	f := setup(t, SeedNone).ready()
	defer f.s.Close()
	login := "first@example.invalid"
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		t.Fatal(got.status)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	f.must(err)
	root := actor{cookie, csrf}
	before := len(f.s.admin.Grants())
	for _, kind := range []string{"propose", "approve", "reject", "withdraw"} {
		fields := map[string]string{"digest": "absent"}
		if kind == "propose" {
			fields = financialPermissionFields(f)
		}
		f.code(t, kind, f.run(root, "financial_permission."+kind, "unseeded", fields), 501, string(commands.UnsupportedCapability))
	}
	f.code(t, "inspect", financialPermissionRead(f, root, "unseeded"), 501, string(commands.UnsupportedCapability))
	if f.s.financialPermission != nil || len(f.s.admin.Grants()) != before {
		t.Fatal("unseeded permission effect")
	}
}

func TestFinancialPermissionFinalConstructionIsClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	a := f.s.financialPermission
	spec := inputauthority.Spec{Capability: financialpermission.Approve, Workspace: SampleWorkspace, Kind: financialpermission.DirectHuman, Recipient: SampleMember, NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
	for _, mutate := range []func(*inputauthority.Spec){
		func(s *inputauthority.Spec) { s.Kind = inputauthority.Console },
		func(s *inputauthority.Spec) { s.Kind = inputauthority.Human },
		func(s *inputauthority.Spec) { s.CeilingID = "ceiling" },
		func(s *inputauthority.Spec) { s.Capability = "inspect" },
		func(s *inputauthority.Spec) { s.Workspace = "foreign" },
	} {
		changed := spec
		mutate(&changed)
		if _, err := a.grant("invalid", changed); err == nil {
			t.Fatal("accepted forged shape", changed)
		}
	}
	for _, domain := range []grantAuthorityDomain{inputGrantAuthority, pricingGrantAuthority} {
		if _, err := f.s.grantAuthority(domain).grant("invalid", spec); err == nil {
			t.Fatal("legacy domain accepted direct human")
		}
	}
}

func TestFinancialPermissionRecipientAndChangedModuleGovernors(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	fields := financialPermissionFields(f)
	fields["recipient"] = SampleAdmin2
	p := f.run(root, "financial_permission.propose", "recipient", fields)
	f.ok(t, "proposal", p, inputauthority.Pending, "")
	for _, command := range []string{"approve", "reject"} {
		f.code(t, command, f.run(second, "financial_permission."+command, "recipient", map[string]string{"digest": field(p, "digest")}), 403, string(commands.PermissionDenied))
	}
	p = f.run(root, "financial_permission.propose", "changed-admins", financialPermissionFields(f))
	f.ok(t, "proposal", p, inputauthority.Pending, "")
	approvalAdminister(t, f.s, approvals.Proposal{ID: "new-governor", Kind: approvals.ProposeAddAdmin, Admin: SampleAdmin3})
	f.code(t, "changed governor set", f.run(second, "financial_permission.approve", "changed-admins", map[string]string{"digest": field(p, "digest")}), 422, string(commands.InvalidStage))
	if _, found := f.s.grantView("financial-permission-" + field(p, "digest")); found {
		t.Fatal("stale governor set created grant")
	}
}
