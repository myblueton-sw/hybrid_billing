package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/resultauthority"
	"reflect"
	"strings"
	"testing"
)

func TestResultAuthorityAllCapabilitiesAndNoDefaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	doc := string(f.mustOpenAPI())
	for _, kind := range []string{"propose_console", "propose_human", "approve", "reject", "withdraw", "inspect"} {
		if !strings.Contains(doc, "result_authority."+kind) {
			t.Fatal("missing OpenAPI command", kind)
		}
	}
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	for _, grant := range f.s.admin.Grants() {
		if resultGrantTerms(grant.Terms.Action, grant.Terms.Kind, grant.Terms.Fields) {
			t.Fatal("default document grant")
		}
	}
	expected := []struct {
		capability string
		action     gate.Action
		fields     []string
	}{
		{"propose", gate.ProposeERPResult, nil}, {"confirm", gate.ConfirmERPResult, nil}, {"revoke", gate.RevokeERPResult, nil}, {"apply", gate.ApplyERPResult, nil}, {"inspect", gate.ViewCost, []string{"erp_result_evidence"}},
	}
	for _, want := range expected {
		capability := want.capability
		t.Run(capability, func(t *testing.T) {
			fields := resultPermissionTimes(f)
			fields["capability"] = capability
			console := f.run(root, "result_authority.propose_console", "console-"+capability, fields)
			f.ok(t, "console", console, resultauthority.Pending, "")
			made := resultPermissionApprove(f, second, "console-"+capability, console)
			f.ok(t, "ceiling", made, resultauthority.Applied, "")
			terms, err := resultGrantAuthority.terms(SampleWorkspace, capability)
			if err != nil {
				t.Fatal(err)
			}
			if terms.Action != want.action || terms.Kind != "synthetic_erp_result_evidence" || !reflect.DeepEqual(terms.Fields, want.fields) {
				t.Fatal("wrong independently specified capability mapping", capability)
			}
			req := gate.Request{Action: want.action, Kind: "synthetic_erp_result_evidence", Fields: want.fields, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
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
			human := f.run(root, "result_authority.propose_human", "human-"+capability, fields)
			f.ok(t, "human", human, resultauthority.Pending, "")
			granted := resultPermissionApprove(f, second, "human-"+capability, human)
			f.ok(t, "grant", granted, resultauthority.Applied, "")
			err = inputAuthorityAs(f, member, func() error {
				if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
					t.Fatal("separate actual grants denied")
				}
				bad := req
				bad.Kind = inputauthority.ResourceKind
				if f.s.gate.Decide(f.s.answer, ConsoleClient, bad).Allowed {
					t.Fatal("cross-kind permission")
				}
				if capability == resultauthority.Inspect {
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
	if Catalog(nil, map[string]gate.FieldClass{resultauthority.EvidenceField: gate.General}).Fields[resultauthority.EvidenceField] != gate.Cost {
		t.Fatal("document evidence downgraded")
	}
	if len(f.s.grantAuthorityDefinitions(0)) != 0 || f.s.grantAuthority(0) != nil {
		t.Fatal("unknown domain selected")
	}
}

func TestResultAuthorityRejectsDocumentCeilingAndBindings(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	fields := documentPermissionTimes(f)
	fields["capability"] = "inspect"
	p := f.run(root, "document_authority.propose_console", "document-only", fields)
	f.ok(t, "document proposal", p, "pending", "")
	approved := documentPermissionApprove(f, second, "document-only", p)
	f.ok(t, "document ceiling", approved, "applied", "")
	fields["recipient"], fields["ceiling"] = SampleMember, field(approved, "grant")
	got := f.run(root, "result_authority.propose_human", "foreign-ceiling", fields)
	f.code(t, "document ceiling cannot grant result", got, 403, "permission_denied")
	err := inputAuthorityAs(f, root, func() error {
		if _, e := f.s.resultAuthority.Lookup(field(approved, "grant")); e == nil {
			t.Fatal("foreign binding adopted")
		}
		if len(f.s.resultAuthority.bindings) != 0 || len(f.s.resultAuthority.ceilings) != 0 {
			t.Fatal("foreign state retained")
		}
		return nil
	})
	f.must(err)
}
