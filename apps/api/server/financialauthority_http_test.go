package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func financialAuthorityFields(f world) map[string]string {
	return map[string]string{"recipient": SampleAdmin3, "currency": "USD", "limit": "1234.56789012", "from": f.s.clock().UTC().Format(time.DateOnly), "to": f.s.clock().UTC().Add(24 * time.Hour).Format(time.DateOnly)}
}
func financialAuthorityRead(f world, a actor, id string) reply {
	body, _ := json.Marshal(map[string]any{"request_id": "limit-read", "target_id": id, "body": map[string]string{}})
	return f.call("financial_authority.inspect", string(body), a.cookie, a.csrf)
}
func TestFinancialAuthorityActualGovernanceAndRevocation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	grants := len(f.s.admin.Grants())
	p := f.run(root, "financial_authority.propose", "limit-one", financialAuthorityFields(f))
	f.ok(t, "propose", p, approvals.StatePending, "")
	digest := map[string]string{"digest": field(p, "digest")}
	f.code(t, "self confirmation", f.run(root, "financial_authority.confirm", "limit-one", digest), 403, string(commands.SelfApprovalDenied))
	f.code(t, "bad digest", f.run(second, "financial_authority.confirm", "limit-one", map[string]string{"digest": "bad"}), 409, string(commands.EvidenceStale))
	c := f.run(second, "financial_authority.confirm", "limit-one", digest)
	f.ok(t, "confirm", c, approvals.StateConfirmed, "")
	if field(c, "limit") != "1234.56789012" || field(c, "module_amount_authority") != "active" || field(c, "recipient") != SampleAdmin3 || field(c, "confirmed_by") != SampleAdmin2 {
		t.Fatal(c.raw)
	}
	f.ok(t, "confirmation receipt", f.run(second, "financial_authority.confirm", "limit-one", digest), approvals.StateConfirmed, "")
	f.s.enterHold("synthetic limit review")
	f.code(t, "held increase", f.run(root, "financial_authority.propose", "held", financialAuthorityFields(f)), 503, string(commands.DependencyUnavailable))
	r := f.run(root, "financial_authority.revoke", "limit-one", digest)
	f.ok(t, "held revocation", r, "revoked", "")
	if field(r, "module_amount_authority") != "revoked" || field(r, "revoked_by") != SampleAdmin {
		t.Fatal(r.raw)
	}
	if got := financialAuthorityRead(f, second, "limit-one"); got.status != 200 || field(got, "module_amount_authority") != "revoked" {
		t.Fatal(got.raw)
	}
	f.s.hold = nil
	c = f.run(second, "financial_authority.confirm", "limit-one", digest)
	f.ok(t, "historical confirm after revocation", c, approvals.StateConfirmed, "")
	if field(c, "module_amount_authority") != "revoked" {
		t.Fatal("replay revived authority", c.raw)
	}
	f.ok(t, "revocation receipt", f.run(root, "financial_authority.revoke", "limit-one", digest), "revoked", "")
	f.code(t, "different revoker", f.run(second, "financial_authority.revoke", "limit-one", digest), 409, string(commands.EvidenceStale))
	if len(f.s.admin.Grants()) != grants {
		t.Fatal("module amount authority created gate grants")
	}
}

func TestFinancialAuthorityCanonicalFieldsAndBounds(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for name, value := range map[string]string{"recipient": "unknown", "currency": "usd", "limit": "15.00", "from": "2026-02-30", "to": ""} {
		fields := financialAuthorityFields(f)
		fields[name] = value
		got := f.run(root, "financial_authority.propose", "invalid-"+name, fields)
		want := 422
		code := string(commands.InvalidRequest)
		if name == "recipient" {
			want = 403
			code = string(commands.PermissionDenied)
		}
		f.code(t, name, got, want, code)
	}
	f.code(t, "derived ID too long", f.run(root, "financial_authority.propose", strings.Repeat("x", 113), financialAuthorityFields(f)), 422, string(commands.InvalidRequest))
	fields := financialAuthorityFields(f)
	fields["recipient"] = SampleAdmin
	f.code(t, "recipient proposes", f.run(root, "financial_authority.propose", "recipient", fields), 403, string(commands.SelfApprovalDenied))
	saved := f.s.cfg.Commands
	for _, change := range []func(){func() { f.s.cfg.Commands.Fields = 22 }, func() { f.s.cfg.Commands.Value = 127 }, func() { f.s.cfg.Commands.Rationale = f.s.cfg.Commands.Value + 1 }} {
		change()
		f.code(t, "capacity", f.run(root, "financial_authority.propose", "capacity", financialAuthorityFields(f)), 501, string(commands.UnsupportedCapability))
		f.s.cfg.Commands = saved
	}
	if len(f.s.financialAuthorities) != 0 {
		t.Fatal("invalid requests mutated bindings")
	}
	fields = financialAuthorityFields(f)
	fields["limit"] = "0"
	f.ok(t, "explicit zero limit", f.run(root, "financial_authority.propose", "zero", fields), approvals.StatePending, "")
}

func TestFinancialAuthorityUTCInclusiveStatus(t *testing.T) {
	r := financialAuthorityRecord{authority: &approvals.AuthorityView{Authority: approvals.Authority{From: "2026-11-20", To: "2026-11-21"}}}
	for _, tc := range []struct{ at, want string }{
		{"2026-11-19T23:59:59.999999999Z", "not_yet_valid"}, {"2026-11-20T00:00:00Z", "active"}, {"2026-11-21T23:59:59.999999999Z", "active"}, {"2026-11-22T00:00:00Z", "expired"}, {"2026-11-20T08:00:00+09:00", "not_yet_valid"},
	} {
		at, err := time.Parse(time.RFC3339Nano, tc.at)
		if err != nil {
			t.Fatal(err)
		}
		if got := financialAmountStatus(r, at); got != tc.want {
			t.Fatal(tc, got)
		}
	}
	r.authority.Revoked = time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	if financialAmountStatus(r, time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)) != "revoked" {
		t.Fatal("revocation lost to expiry")
	}
	if financialAmountStatus(financialAuthorityRecord{}, time.Now()) != "not_granted" {
		t.Fatal("pending became granted")
	}
}
