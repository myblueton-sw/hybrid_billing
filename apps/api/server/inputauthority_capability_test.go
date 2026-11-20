package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"testing"
)

func provisionInputCapability(t *testing.T, f world, capability string) (actor, string, string) {
	t.Helper()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	fields := authorityTimes(f)
	fields["capability"] = capability
	id := "console-" + capability
	proposed := f.run(root, "input_authority.propose_console", id, fields)
	f.ok(t, "console proposal", proposed, inputauthority.Pending, "")
	ceiling := authorityApprove(f, second, id, proposed)
	f.ok(t, "console approval", ceiling, inputauthority.Applied, "")
	fields["recipient"] = SampleMember
	fields["ceiling"] = field(ceiling, "grant")
	human := f.run(root, "input_authority.propose_human", "human-"+capability, fields)
	f.ok(t, "human proposal", human, inputauthority.Pending, "")
	granted := authorityApprove(f, second, "human-"+capability, human)
	f.ok(t, "human approval", granted, inputauthority.Applied, "")
	return f.as(MemberLogin), field(ceiling, "grant"), field(granted, "grant")
}

func TestInputAuthorityCapabilitiesRemainIndependent(t *testing.T) {
	for _, capability := range []string{inputauthority.Confirm, inputauthority.Inspect} {
		t.Run(capability, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			member, ceiling, _ := provisionInputCapability(t, f, capability)
			requests := []gate.Request{
				{Action: gate.ConfirmInput, Kind: inputauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}},
				{Action: gate.ViewCost, Kind: inputauthority.ResourceKind, Fields: []string{inputauthority.EvidenceField}, Resource: gate.Resource{WorkspaceID: SampleWorkspace}},
				{Action: gate.View, Kind: inputauthority.ResourceKind, Fields: []string{inputauthority.EvidenceField}, Resource: gate.Resource{WorkspaceID: SampleWorkspace}},
				{Action: gate.ViewCost, Kind: inputauthority.ResourceKind, Fields: []string{candidateField}, Resource: gate.Resource{WorkspaceID: SampleWorkspace}},
			}
			if err := inputAuthorityAs(f, member, func() error {
				for n, req := range requests {
					want := n == 0 && capability == inputauthority.Confirm || n == 1 && capability == inputauthority.Inspect
					if got := f.s.gate.Decide(f.s.answer, ConsoleClient, req); got.Allowed != want {
						t.Fatalf("request %d allowed %v want %v", n, got.Allowed, want)
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			fields := authorityTimes(f)
			if capability == inputauthority.Confirm {
				fields["capability"] = inputauthority.Inspect
			}
			fields["recipient"] = SampleMember
			fields["ceiling"] = ceiling
			f.code(t, "wrong capability ceiling", f.run(f.as(AdminLogin), "input_authority.propose_human", "wrong-ceiling", fields), 403, string(commands.PermissionDenied))
		})
	}
}
