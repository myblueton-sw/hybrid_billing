package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialAuthorityExpiryCannotReactivateOnClockRollback(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	fields := financialAuthorityFields(f)
	fields["from"] = f.s.clock().UTC().Add(24 * time.Hour).Format(time.DateOnly)
	p := f.run(root, "financial_authority.propose", "clock", fields)
	f.ok(t, "propose", p, approvals.StatePending, "")
	c := f.run(second, "financial_authority.confirm", "clock", map[string]string{"digest": field(p, "digest")})
	f.ok(t, "confirm", c, approvals.StateConfirmed, "")
	if field(c, "module_amount_authority") != "not_yet_valid" {
		t.Fatal(c.raw)
	}
	f.advance(24 * time.Hour)
	record, err := f.s.financialAuthorityRecord(SampleWorkspace, "clock")
	if err != nil || financialAuthorityReceipt(record)["module_amount_authority"] != "active" {
		t.Fatal(record, err)
	}
	f.advance(24 * time.Hour)
	expired, err := f.s.financialAuthorityRecord(SampleWorkspace, "clock")
	if err != nil || financialAuthorityReceipt(expired)["module_amount_authority"] != "expired" {
		t.Fatal(expired, err)
	}
	f.advance(-24 * time.Hour)
	rollback, err := f.s.financialAuthorityRecord(SampleWorkspace, "clock")
	if err != nil || financialAuthorityReceipt(rollback)["module_amount_authority"] != "expired" || rollback.asOf.Before(expired.asOf) {
		t.Fatal("rollback reactivated", rollback, err)
	}
}

func TestFinancialAuthorityUnseededAndStepUp(t *testing.T) {
	t.Run("unseeded", func(t *testing.T) {
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
		f.code(t, "propose", f.run(root, "financial_authority.propose", "unseeded", financialAuthorityFields(f)), 501, string(commands.UnsupportedCapability))
		for _, kind := range []string{"confirm", "revoke"} {
			f.code(t, kind, f.run(root, "financial_authority."+kind, "unseeded", map[string]string{"digest": "missing"}), 501, string(commands.UnsupportedCapability))
		}
		f.code(t, "inspect", financialAuthorityRead(f, root, "unseeded"), 501, string(commands.UnsupportedCapability))
		if f.s.counter != before || len(f.s.admin.Grants()) != grants || len(f.s.financialAuthorities) != 0 {
			t.Fatal("unseeded side effect")
		}
	})
	t.Run("stepup", func(t *testing.T) {
		f := setup(t, SeedSample).ready()
		root, second := f.as(AdminLogin), f.as(Admin2Login)
		p := f.run(root, "financial_authority.propose", "stepup", financialAuthorityFields(f))
		f.ok(t, "propose", p, approvals.StatePending, "")
		f.advance(16 * time.Minute)
		f.code(t, "confirm", f.run(second, "financial_authority.confirm", "stepup", map[string]string{"digest": field(p, "digest")}), 403, CodeReauthentication)
		if got := financialAuthorityRead(f, root, "stepup"); got.status != 200 {
			t.Fatal(got.raw)
		}
	})
}
