package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"testing"
	"time"
)

func TestERPFinalPermissionUnseeded(t *testing.T) {
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
		f.code(t, kind, f.run(root, "erp_final_permission."+kind, "unseeded", fields), 501, string(commands.UnsupportedCapability))
	}
	f.code(t, "inspect", documentRead(f, root, "erp_final_permission.inspect", "unseeded", map[string]string{}), 501, string(commands.UnsupportedCapability))
	if f.s.erpFinalPermission != nil || len(f.s.admin.Grants()) != before {
		t.Fatal("unseeded permission effect")
	}
}

func TestERPFinalPermissionConstructionIsClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	a := f.s.erpFinalPermission
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
