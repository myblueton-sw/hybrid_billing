package server

import (
	"encoding/json"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func financialFields(revision string) map[string]string {
	return map[string]string{"policy_revision": revision, "approvals": "1", "validity_hours": "24", "required": "input,result"}
}
func financialRead(f world, a actor, kind, id string) reply {
	body, _ := json.Marshal(map[string]any{"request_id": "financial-read", "target_id": id, "body": map[string]string{}})
	return f.call("financial_policy."+kind, string(body), a.cookie, a.csrf)
}
func TestFinancialPolicyActualGovernanceAndHistory(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	before := len(f.s.admin.Grants())
	current := financialRead(f, root, "current", approvals.StageCharge)
	if current.status != 200 || field(current, "policy_revision") != "1" {
		t.Fatal(current.raw)
	}
	p := f.run(root, "financial_policy.propose", "financial-two", financialFields("2"))
	f.ok(t, "propose", p, approvals.StatePending, "")
	digest := map[string]string{"digest": field(p, "digest")}
	f.code(t, "self confirmation", f.run(root, "financial_policy.confirm", "financial-two", digest), 403, "self_approval_denied")
	f.code(t, "wrong digest", f.run(second, "financial_policy.confirm", "financial-two", map[string]string{"digest": "wrong"}), 409, "evidence_stale")
	confirmed := f.run(second, "financial_policy.confirm", "financial-two", digest)
	f.ok(t, "independent confirm", confirmed, approvals.StateConfirmed, "")
	if field(confirmed, "proposed_by") != SampleAdmin || field(confirmed, "confirmed_by") != SampleAdmin2 || field(confirmed, "receipt") != "historical_decision" {
		t.Fatal(confirmed.raw)
	}
	f.ok(t, "fresh-key replay", f.run(second, "financial_policy.confirm", "financial-two", digest), approvals.StateConfirmed, "")
	if len(f.s.admin.Grants()) != before {
		t.Fatal("policy created authority grants")
	}
	current = financialRead(f, second, "current", approvals.StageCharge)
	if field(current, "policy_revision") != "2" {
		t.Fatal(current.raw)
	}
	if got := f.approvalRead(root, "view=history"); got.status != 200 {
		t.Fatal("policy replacement lost history", got.raw)
	}
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin && g.Terms.Action == gate.AdministerInstallation {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "revoked proposer replay", f.run(second, "financial_policy.confirm", "financial-two", digest), 403, "permission_denied")
}

func TestFinancialPolicyPreflightAndCollision(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, name := range []string{"policy_revision", "approvals", "validity_hours", "required"} {
		fields := financialFields("2")
		fields[name] = "01"
		f.code(t, name, f.run(root, "financial_policy.propose", "bad-"+name, fields), 422, "invalid_request")
	}
	f.code(t, "stale revision", f.run(root, "financial_policy.propose", "stale", financialFields("1")), 409, "evidence_stale")
	f.code(t, "demo collision", f.run(root, "financial_policy.propose", "sample-"+approvals.StageCharge, financialFields("2")), 409, "evidence_stale")
	limits := f.s.cfg.Commands
	for _, adjust := range []func(){func() { f.s.cfg.Commands.Fields = 15 }, func() { f.s.cfg.Commands.Value = 127 }, func() { f.s.cfg.Commands.Rationale = f.s.cfg.Commands.Value + 1 }} {
		adjust()
		got := f.run(root, "financial_policy.propose", "too-small", financialFields("2"))
		if got.status == 200 {
			t.Fatal("response preflight accepted", got.raw)
		}
		f.s.cfg.Commands = limits
	}
	if len(f.s.financialPolicies) != 0 {
		t.Fatal("failed requests created policy evidence")
	}
}
