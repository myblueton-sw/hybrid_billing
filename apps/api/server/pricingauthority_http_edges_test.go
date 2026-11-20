package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
)

func TestPricingAuthorityHTTPImpactCountsUnknownAndFailsClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	unknown := f.run(second, "pricing_authority.propose_console", "uncertain", pricingTimes(f))
	f.ok(t, "proposal", unknown, pricingauthority.Pending, "")
	f.s.fault = func(at string) bool { return at == "pricing-authority-after-grant" }
	f.code(t, "unknown", pricingApproveHTTP(f, root, "uncertain", unknown), 500, string(commands.InternalError))
	f.s.fault = nil
	pending := f.run(second, "pricing_authority.propose_console", "pending", pricingTimes(f))
	f.ok(t, "pending", pending, pricingauthority.Pending, "")
	staged := f.stage(root, "", SampleAdmin2)
	f.ok(t, "stage", staged, "draft", "")
	draft := field(staged, "draft")
	before := f.s.counter
	f.s.fault = func(at string) bool { return at == "pricing-authority-impact" }
	f.code(t, "impact unavailable", f.prepare(root, draft, "suspend_membership"), 503, string(commands.DependencyUnavailable))
	if f.s.counter != before {
		t.Fatal("failed impact prepared a snapshot")
	}
	f.s.fault = nil
	prepared := f.prepare(root, draft, "suspend_membership")
	f.ok(t, "prepare", prepared, "prepared", "")
	if field(prepared, "bulk") != "bulk-revocation-1" {
		t.Fatal("failed attempt reserved snapshot")
	}
	detail := f.bulkRead(root, "/"+field(prepared, "bulk"))
	rows := detail.body["impact"].(map[string]any)["pending"].([]any)
	found := false
	for _, raw := range rows {
		row := raw.(map[string]any)
		if row["store"] == "pricing_authority_proposals" {
			found = true
			if row["count"] != 2.0 || row["rechecked"] != true {
				t.Fatalf("input impact %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("new proposal namespace omitted")
	}
}

func TestPricingAuthorityHTTPHoldAndStepUp(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	proposed := f.run(root, "pricing_authority.propose_console", "hold-proposal", pricingTimes(f))
	f.ok(t, "proposal", proposed, pricingauthority.Pending, "")
	f.advance(16 * time.Minute)
	f.code(t, "stepup required", pricingApproveHTTP(f, second, "hold-proposal", proposed), 403, CodeReauthentication)
	root, second = f.as(AdminLogin), f.as(Admin2Login)
	f.s.domain.Lock()
	f.s.enterHold("synthetic-test")
	f.s.domain.Unlock()
	f.code(t, "held approval", pricingApproveHTTP(f, second, "hold-proposal", proposed), 503, string(commands.DependencyUnavailable))
	if got := pricingInspectHTTP(f, root, "hold-proposal"); got.status != 200 {
		t.Fatalf("held inspect %s", got.raw)
	}
	rejected := f.run(second, "pricing_authority.reject", "hold-proposal", map[string]string{"digest": field(proposed, "digest")})
	f.ok(t, "held rejection", rejected, pricingauthority.Rejected, "")
}

func TestPricingAuthorityHTTPReplayRechecksGovernance(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	proposed := f.run(root, "pricing_authority.propose_console", "replay-proposal", pricingTimes(f))
	body, _ := json.Marshal(map[string]any{"request_id": "input-replay", "target_id": "replay-proposal", "idempotency_key": "input-replay-key", "expected_revision": f.s.Revision(), "rationale": "sample reason", "body": map[string]string{"digest": field(proposed, "digest")}})
	made := f.call("pricing_authority.approve", string(body), second.cookie, second.csrf)
	f.ok(t, "approved", made, pricingauthority.Applied, "")
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.AdministerInstallation {
			if err := f.s.admin.RevokeGrant(g.ID, g.Version); err != nil {
				t.Fatal(err)
			}
		}
	}
	before := f.s.counter
	f.code(t, "revoked replay", f.call("pricing_authority.approve", string(body), second.cookie, second.csrf), 403, string(commands.PermissionDenied))
	if f.s.counter != before {
		t.Fatal("denied replay changed revision")
	}
}

func TestPricingAuthorityHTTPCatalogAndReadEnvelope(t *testing.T) {
	f := setup(t, SeedSample).ready()
	data, err := f.s.Document()
	if err != nil {
		t.Fatal(err)
	}
	for _, verb := range []string{"propose_console", "propose_human", "approve", "reject", "withdraw", "inspect"} {
		if !strings.Contains(string(data), "/commands/pricing_authority."+verb) {
			t.Fatal("missing command", verb)
		}
	}
	root := f.as(AdminLogin)
	before := f.s.counter
	f.code(t, "mutation envelope on read", f.run(root, "pricing_authority.inspect", "missing", nil), 422, string(commands.InvalidRequest))
	if f.s.counter != before {
		t.Fatal("invalid read mutated state")
	}
	f.code(t, "unauthenticated inspect", pricingInspectHTTP(f, actor{}, "missing"), 401, string(commands.Unauthenticated))
}

func TestPricingAuthorityHTTPUnseededUnsupported(t *testing.T) {
	f := setup(t, SeedNone).ready()
	defer f.s.Close()
	login := "first@example.invalid"
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		t.Fatal("bootstrap", got.status)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	root := actor{cookie, csrf}
	before, grants := f.s.counter, len(f.s.admin.Grants())
	for _, verb := range []string{"propose_console", "propose_human", "approve", "reject", "withdraw"} {
		fields := map[string]string{"digest": strings.Repeat("a", 64)}
		if verb == "propose_console" || verb == "propose_human" {
			fields = pricingTimes(f)
		}
		if verb == "propose_human" {
			fields["recipient"] = EmptyAdmin
			fields["ceiling"] = "synthetic-ceiling"
		}
		f.code(t, verb, f.run(root, "pricing_authority."+verb, "unseeded", fields), 501, string(commands.UnsupportedCapability))
	}
	f.code(t, "inspect", pricingInspectHTTP(f, root, "unseeded"), 501, string(commands.UnsupportedCapability))
	if f.s.pricingAuthority != nil || f.s.counter != before || len(f.s.admin.Grants()) != grants {
		t.Fatal("unseeded authority changed")
	}
	for _, g := range f.s.admin.Grants() {
		if g.Terms.Action == gate.ApprovePricingPolicy {
			t.Fatal("unseeded confirmation grant")
		}
	}
}

func TestPricingAuthorityHTTPUnknownRechecksRevokedApprover(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	proposed := f.run(root, "pricing_authority.propose_console", "unknown-revoked", pricingTimes(f))
	body, _ := json.Marshal(map[string]any{"request_id": "unknown-revoked", "target_id": "unknown-revoked", "idempotency_key": "unknown-revoked-key", "expected_revision": f.s.Revision(), "rationale": "sample reason", "body": map[string]string{"digest": field(proposed, "digest")}})
	f.s.fault = func(at string) bool { return at == "pricing-authority-after-grant" }
	f.code(t, "unknown", f.call("pricing_authority.approve", string(body), second.cookie, second.csrf), 500, string(commands.InternalError))
	f.s.fault = nil
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.AdministerInstallation {
			if err := f.s.admin.RevokeGrant(g.ID, g.Version); err != nil {
				t.Fatal(err)
			}
		}
	}
	before, grants, ceilings := f.s.counter, len(f.s.admin.Grants()), len(f.s.pricingAuthority.ceilings)
	f.code(t, "same key", f.call("pricing_authority.approve", string(body), second.cookie, second.csrf), 403, string(commands.PermissionDenied))
	f.code(t, "new key", f.runWith(second, f.s.Revision(), "pricing_authority.approve", "unknown-revoked", map[string]string{"digest": field(proposed, "digest")}), 403, string(commands.PermissionDenied))
	inspected := pricingInspectHTTP(f, root, "unknown-revoked")
	if inspected.status != 200 || field(inspected, "state") != pricingauthority.Unknown || f.s.counter != before || len(f.s.admin.Grants()) != grants || len(f.s.pricingAuthority.ceilings) != ceilings {
		t.Fatal("revoked reconciliation changed unknown effect")
	}
}
