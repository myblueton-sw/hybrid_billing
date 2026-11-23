package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialPolicyUnseededHTTP(t *testing.T) {
	f := setup(t, SeedNone).ready()
	defer f.s.Close()
	login := "first@example.invalid"
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		t.Fatal("bootstrap", got.status)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	f.must(err)
	root := actor{cookie, csrf}
	before, grants := f.s.counter, len(f.s.admin.Grants())
	f.code(t, "propose", f.run(root, "financial_policy.propose", "unseeded", financialFields("1")), 501, string(commands.UnsupportedCapability))
	f.code(t, "confirm", f.run(root, "financial_policy.confirm", "unseeded", map[string]string{"digest": "unavailable"}), 501, string(commands.UnsupportedCapability))
	for _, kind := range []string{"inspect", "current"} {
		f.code(t, kind, financialRead(f, root, kind, approvals.StageCharge), 501, string(commands.UnsupportedCapability))
	}
	if f.s.counter != before || len(f.s.admin.Grants()) != grants || len(f.s.financialPolicies) != 0 {
		t.Fatal("unseeded policy changed state")
	}
}

func TestFinancialPolicyStepUp(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p := f.run(root, "financial_policy.propose", "step-up", financialFields("2"))
	f.ok(t, "propose", p, approvals.StatePending, "")
	f.advance(16 * time.Minute)
	f.code(t, "confirm step-up", f.run(second, "financial_policy.confirm", "step-up", map[string]string{"digest": field(p, "digest")}), 403, CodeReauthentication)
	if got := financialRead(f, second, "inspect", "step-up"); got.status != 200 {
		t.Fatal("read required step-up", got.raw)
	}
	second = f.as(Admin2Login)
	f.ok(t, "fresh confirmation", f.run(second, "financial_policy.confirm", "step-up", map[string]string{"digest": field(p, "digest")}), approvals.StateConfirmed, "")
}
