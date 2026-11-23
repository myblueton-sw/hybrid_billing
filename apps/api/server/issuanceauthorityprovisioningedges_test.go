package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func TestIssuanceAuthorityFinalGrantShapeIsClosed(t *testing.T) {
	for _, ns := range issuanceAuthorityNamespaces {
		t.Run(ns.name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			a := f.s.grantAuthority(ns.domain)
			spec := inputauthority.Spec{Workspace: SampleWorkspace, Capability: ns.capabilities[0], Kind: inputauthority.Console, Recipient: ConsoleClient, NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
			for _, change := range []func(*inputauthority.Spec){
				func(s *inputauthority.Spec) { s.Workspace = "foreign" }, func(s *inputauthority.Spec) { s.Capability = "administer" },
				func(s *inputauthority.Spec) { s.Kind = financialpermission.DirectHuman }, func(s *inputauthority.Spec) { s.Recipient = "foreign-client" },
				func(s *inputauthority.Spec) { s.CeilingID = "parent" }, func(s *inputauthority.Spec) {
					s.Kind = inputauthority.Human
					s.Recipient = SampleMember
					s.CeilingID = "foreign-ceiling"
				},
			} {
				bad := spec
				change(&bad)
				if _, err := a.grant("invalid", bad); err == nil {
					t.Fatal("forged final grant", bad)
				}
			}
			if _, err := a.grant("valid", spec); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestIssuanceAuthorityRevocationRejectionAndLostStanding(t *testing.T) {
	for _, ns := range issuanceAuthorityNamespaces {
		t.Run(ns.name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second, third := f.as(AdminLogin), f.as(Admin2Login), f.as(Admin3Login)
			capability := ns.capabilities[0]
			granted := provisionIssuanceCapability(t, f, ns.name, capability, SampleMember, "edge-")
			ceiling := issuanceAuthorityRead(f, third, ns.name, "edge-ceiling")
			human := issuanceAuthorityRead(f, third, ns.name, "edge-human")
			if got := f.run(root, "access.request_revoke", field(ceiling, "grant"), nil); got.status == 200 || got.status == 202 {
				t.Fatal("a console ceiling became publicly revocable", got.raw)
			}
			if got := f.revokeVia(root, second, field(granted, "grant")); got.status != 202 {
				t.Fatal(got.raw)
			}
			f.ok(t, "historical replay", f.run(second, ns.name+".approve", "edge-human", map[string]string{"digest": field(human, "digest")}), inputauthority.Applied, "")
			if issuanceGrantCount(f, ns.domain) != 2 {
				t.Fatal("a replay changed the recorded grants")
			}
			if g, ok := f.s.grantView(field(granted, "grant")); !ok || inputGrantStatus(g, f.s.clock()) != inputauthority.RevokedGrant {
				t.Fatal("a replay reactivated a revoked grant")
			}

			again := issuanceAuthorityTimes(f, capability)
			again["recipient"], again["ceiling"] = SampleMember, field(ceiling, "grant")
			again["not_after"] = f.s.clock().Add(11 * time.Hour).Format(time.RFC3339Nano)
			proposed := f.run(root, ns.name+".propose_human", "edge-rejected", again)
			f.ok(t, "second proposal", proposed, inputauthority.Pending, "")
			digest := map[string]string{"digest": field(proposed, "digest")}
			f.ok(t, "rejection", f.run(second, ns.name+".reject", "edge-rejected", digest), inputauthority.Rejected, "")
			f.code(t, "approve a rejected proposal", f.run(third, ns.name+".approve", "edge-rejected", digest), 422, string(commands.InvalidStage))

			again["not_after"] = f.s.clock().Add(10 * time.Hour).Format(time.RFC3339Nano)
			waiting := f.run(root, ns.name+".propose_human", "edge-waiting", again)
			f.ok(t, "third proposal", waiting, inputauthority.Pending, "")
			f.ok(t, "suspend a governor", f.run(root, "identity.suspend_user", SampleAdmin2, nil), "user_suspended", "")
			if got := f.runWith(second, f.s.Revision(), ns.name+".approve", "edge-waiting", map[string]string{"digest": field(waiting, "digest")}); got.status == 200 {
				t.Fatal("a suspended governor approved", got.raw)
			}
			if got := f.runWith(second, f.s.Revision(), ns.name+".propose_console", "edge-late", issuanceAuthorityTimes(f, capability)); got.status == 200 {
				t.Fatal("a suspended governor proposed", got.raw)
			}
			if got := issuanceAuthorityRead(f, second, ns.name, "edge-waiting"); got.status == 200 {
				t.Fatal("a suspended governor inspected", got.raw)
			}
			if got := issuanceAuthorityRead(f, third, ns.name, "edge-waiting"); field(got, "state") != inputauthority.Pending {
				t.Fatal("a refused approval changed the proposal", got.raw)
			}
		})
	}
}
