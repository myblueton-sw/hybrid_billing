package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/documentauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func TestDocumentPermissionFixedWireAndCapacity(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, name := range []string{"recipient", "ceiling", "action", "kind", "workspace"} {
		fields := documentPermissionTimes(f)
		fields[name] = "approve"
		f.code(t, name, f.run(root, "document_authority.propose_console", "forged-"+name, fields), 422, string(commands.InvalidRequest))
	}
	saved := f.s.cfg.Commands
	for _, change := range []func(){func() { f.s.cfg.Commands.Fields = 19 }, func() { f.s.cfg.Commands.Value = 1023 }} {
		change()
		f.code(t, "capacity", f.run(root, "document_authority.propose_console", "capacity", documentPermissionTimes(f)), 501, string(commands.UnsupportedCapability))
		f.s.cfg.Commands = saved
	}
	if len(f.s.documentAuthority.bindings) != 0 {
		t.Fatal("refusal mutated bindings")
	}
}

func TestDocumentPermissionHumanRecipientExclusions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	console := provisionDocumentCeiling(t, f)
	fields := documentPermissionTimes(f)
	fields["recipient"], fields["ceiling"] = SampleAdmin, field(console, "grant")
	f.code(t, "recipient proposes", f.run(root, "document_authority.propose_human", "self", fields), 403, string(commands.PermissionDenied))
	fields["recipient"] = SampleAdmin2
	p := f.run(root, "document_authority.propose_human", "recipient", fields)
	f.ok(t, "proposal", p, inputauthority.Pending, "")
	f.code(t, "recipient approves", documentPermissionApprove(f, second, "recipient", p), 403, string(commands.PermissionDenied))
	f.code(t, "recipient rejects", f.run(second, "document_authority.reject", "recipient", map[string]string{"digest": field(p, "digest")}), 403, string(commands.PermissionDenied))
}

func TestDocumentPermissionFinalGrantShapeIsClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	a := f.s.documentAuthority
	spec := inputauthority.Spec{Workspace: SampleWorkspace, Capability: documentauthority.Prepare, Kind: inputauthority.Console, Recipient: ConsoleClient, NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
	for _, change := range []func(*inputauthority.Spec){
		func(s *inputauthority.Spec) { s.Workspace = "foreign" }, func(s *inputauthority.Spec) { s.Capability = "approve" },
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
}

func TestDocumentPermissionUnseeded(t *testing.T) {
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
	for _, kind := range []string{"propose_console", "propose_human", "approve", "reject", "withdraw"} {
		fields := map[string]string{"digest": "none"}
		if kind == "propose_console" || kind == "propose_human" {
			fields = documentPermissionTimes(f)
		}
		if kind == "propose_human" {
			fields["recipient"] = "member"
			fields["ceiling"] = "none"
		}
		f.code(t, kind, f.run(root, "document_authority."+kind, "unseeded", fields), 501, string(commands.UnsupportedCapability))
	}
	f.code(t, "inspect", documentPermissionRead(f, root, "unseeded"), 501, string(commands.UnsupportedCapability))
	if f.s.documentAuthority != nil || len(f.s.admin.Grants()) != before {
		t.Fatal("unseeded effect")
	}
}
