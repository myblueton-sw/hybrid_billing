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

func pricingTimes(f world) map[string]string {
	return map[string]string{"capability": pricingauthority.Approve, "not_before": f.s.clock().Format(time.RFC3339Nano), "not_after": f.s.clock().Add(12 * time.Hour).Format(time.RFC3339Nano)}
}
func pricingApproveHTTP(f world, a actor, id string, proposal reply) reply {
	return f.run(a, "pricing_authority.approve", id, map[string]string{"digest": field(proposal, "digest")})
}

func TestPricingAuthorityHTTPSeparateProvisioning(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	console := f.run(root, "pricing_authority.propose_console", "console-http", pricingTimes(f))
	f.ok(t, "console proposal", console, pricingauthority.Pending, "")
	made := pricingApproveHTTP(f, second, "console-http", console)
	f.ok(t, "console approval", made, pricingauthority.Applied, "")
	humanFields := pricingTimes(f)
	humanFields["recipient"] = SampleMember
	humanFields["ceiling"] = field(made, "grant")
	human := f.run(root, "pricing_authority.propose_human", "human-http", humanFields)
	f.ok(t, "human proposal", human, pricingauthority.Pending, "")
	granted := pricingApproveHTTP(f, second, "human-http", human)
	f.ok(t, "human approval", granted, pricingauthority.Applied, "")
	req := gate.Request{Action: gate.ApprovePricingPolicy, Kind: pricingauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
	for _, test := range []struct {
		actor   actor
		allowed bool
	}{{root, false}, {second, false}, {member, true}} {
		if err := inputAuthorityAs(f, test.actor, func() error {
			if got := f.s.gate.Decide(f.s.answer, ConsoleClient, req); got.Allowed != test.allowed {
				t.Fatalf("confirmation allowed %v want %v", got.Allowed, test.allowed)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	before := f.s.Revision()
	inspect := pricingInspectHTTP(f, root, "human-http")
	if inspect.status != 200 || field(inspect, "state") != pricingauthority.Applied || field(inspect, "recorded_grant_status") != pricingauthority.ActiveGrant || f.s.Revision() != before {
		t.Fatalf("inspect %s", inspect.raw)
	}
	denied := pricingInspectHTTP(f, member, "human-http")
	f.code(t, "member inspect", denied, 403, string(commands.PermissionDenied))
	f.code(t, "unknown inspect", pricingInspectHTTP(f, root, "missing-http"), 403, string(commands.PermissionDenied))
}

func TestPricingAuthorityHTTPUnknownRequiresExplicitNewRequest(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	proposed := f.run(root, "pricing_authority.propose_console", "console-http", pricingTimes(f))
	f.ok(t, "proposal", proposed, pricingauthority.Pending, "")
	before := f.s.counter
	body, _ := json.Marshal(map[string]any{"request_id": "input-unknown", "target_id": "console-http", "idempotency_key": "input-unknown-key", "expected_revision": f.s.Revision(), "rationale": "sample reason", "body": map[string]string{"digest": field(proposed, "digest")}})
	f.s.fault = func(at string) bool { return at == "pricing-authority-after-grant" }
	failed := f.call("pricing_authority.approve", string(body), second.cookie, second.csrf)
	f.code(t, "unknown", failed, 500, string(commands.InternalError))
	if !strings.Contains(failed.raw, "inspect_operation") || f.s.counter != before+1 {
		t.Fatal("unknown protocol/revision missing")
	}
	f.s.fault = nil
	grants := len(f.s.admin.Grants())
	replay := f.call("pricing_authority.approve", string(body), second.cookie, second.csrf)
	f.code(t, "retained unknown", replay, 500, string(commands.InternalError))
	if f.s.counter != before+1 || len(f.s.admin.Grants()) != grants {
		t.Fatal("same-key unknown reexecuted")
	}
	inspect := pricingInspectHTTP(f, root, "console-http")
	if inspect.status != 200 || field(inspect, "state") != pricingauthority.Unknown {
		t.Fatalf("unknown inspection %s", inspect.raw)
	}
	f.code(t, "unknown reject", f.run(second, "pricing_authority.reject", "console-http", map[string]string{"digest": field(proposed, "digest")}), 422, string(commands.InvalidStage))
	resolved := pricingApproveHTTP(f, second, "console-http", proposed)
	f.ok(t, "resolved", resolved, pricingauthority.Applied, "")
	if f.s.counter != before+2 || len(f.s.admin.Grants()) != grants {
		t.Fatal("resolution duplicated effect or revision")
	}
}

func TestPricingAuthorityHTTPRefusalRevisions(t *testing.T) {
	for _, cause := range []string{"expired", "superseded"} {
		t.Run(cause, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root, second := f.as(AdminLogin), f.as(Admin2Login)
			proposed := f.run(root, "pricing_authority.propose_console", "proposal", pricingTimes(f))
			if cause == "expired" {
				f.advance(time.Hour)
				second = f.as(Admin2Login)
			} else {
				if _, err := f.s.admin.PutClient(gate.Client{ID: "unrelated", State: gate.Active}, 0); err != nil {
					t.Fatal(err)
				}
			}
			before := f.s.counter
			refused := pricingApproveHTTP(f, second, "proposal", proposed)
			f.code(t, "first refusal", refused, 422, string(commands.InvalidStage))
			if f.s.counter != before+1 {
				t.Fatal("changed refusal did not advance revision")
			}
			f.code(t, "repeat refusal", pricingApproveHTTP(f, second, "proposal", proposed), 422, string(commands.InvalidStage))
			if f.s.counter != before+1 {
				t.Fatal("unchanged refusal advanced revision")
			}
		})
	}
}

func pricingInspectHTTP(f world, a actor, id string) reply {
	body, _ := json.Marshal(map[string]any{"request_id": "input-inspect", "target_id": id, "body": map[string]string{}})
	return f.call("pricing_authority.inspect", string(body), a.cookie, a.csrf)
}
