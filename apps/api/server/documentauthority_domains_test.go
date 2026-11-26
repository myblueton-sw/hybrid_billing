package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/documentauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"reflect"
	"strings"
	"testing"
)

func TestDocumentAuthorityAllCapabilitiesAndNoDefaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	doc := string(f.mustOpenAPI())
	for _, kind := range []string{"propose_console", "propose_human", "approve", "reject", "withdraw", "inspect"} {
		if !strings.Contains(doc, "document_authority."+kind) {
			t.Fatal("missing OpenAPI command", kind)
		}
	}
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	for _, grant := range f.s.admin.Grants() {
		if documentGrantTerms(grant.Terms.Action, grant.Terms.Kind, grant.Terms.Fields) {
			t.Fatal("default document grant")
		}
	}
	expected := []struct {
		capability string
		action     gate.Action
		fields     []string
	}{
		{"propose", gate.ProposeDocumentEvidence, nil}, {"confirm", gate.ConfirmDocumentEvidence, nil}, {"revoke", gate.RevokeDocumentEvidence, nil}, {"prepare", gate.PrepareERPRecord, nil}, {"inspect", gate.ViewCost, []string{"document_evidence"}},
	}
	for _, want := range expected {
		capability := want.capability
		t.Run(capability, func(t *testing.T) {
			fields := documentPermissionTimes(f)
			fields["capability"] = capability
			console := f.run(root, "document_authority.propose_console", "console-"+capability, fields)
			f.ok(t, "console", console, documentauthority.Pending, "")
			made := documentPermissionApprove(f, second, "console-"+capability, console)
			f.ok(t, "ceiling", made, documentauthority.Applied, "")
			terms, err := documentGrantAuthority.terms(SampleWorkspace, capability)
			if err != nil {
				t.Fatal(err)
			}
			if terms.Action != want.action || terms.Kind != "synthetic_document_evidence" || !reflect.DeepEqual(terms.Fields, want.fields) {
				t.Fatal("wrong independently specified capability mapping", capability)
			}
			req := gate.Request{Action: want.action, Kind: "synthetic_document_evidence", Fields: want.fields, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
			err = inputAuthorityAs(f, member, func() error {
				if f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
					t.Fatal("ceiling without human granted")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			fields["recipient"] = SampleMember
			fields["ceiling"] = field(made, "grant")
			human := f.run(root, "document_authority.propose_human", "human-"+capability, fields)
			f.ok(t, "human", human, documentauthority.Pending, "")
			granted := documentPermissionApprove(f, second, "human-"+capability, human)
			f.ok(t, "grant", granted, documentauthority.Applied, "")
			err = inputAuthorityAs(f, member, func() error {
				if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
					t.Fatal("separate actual grants denied")
				}
				bad := req
				bad.Kind = inputauthority.ResourceKind
				if f.s.gate.Decide(f.s.answer, ConsoleClient, bad).Allowed {
					t.Fatal("cross-kind permission")
				}
				if capability == documentauthority.Inspect {
					bad = req
					bad.Fields = []string{inputauthority.EvidenceField}
					if f.s.gate.Decide(f.s.answer, ConsoleClient, bad).Allowed {
						t.Fatal("cross-evidence permission")
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	if Catalog(nil, map[string]gate.FieldClass{documentauthority.EvidenceField: gate.General}).Fields[documentauthority.EvidenceField] != gate.Cost {
		t.Fatal("document evidence downgraded")
	}
	if len(f.s.grantAuthorityDefinitions(0)) != 0 || f.s.grantAuthority(0) != nil {
		t.Fatal("unknown domain selected")
	}
}
