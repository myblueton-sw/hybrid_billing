package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
	"reflect"
	"strings"
	"testing"
)

func TestPricingAuthorityAllCapabilitiesAndNoDefaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	for _, grant := range f.s.admin.Grants() {
		if pricingGrantTerms(grant.Terms.Action, grant.Terms.Kind, grant.Terms.Fields) {
			t.Fatal("default pricing grant")
		}
	}
	expected := []struct {
		capability string
		action     gate.Action
		fields     []string
	}{
		{"propose", gate.ProposePricingPolicy, nil}, {"approve", gate.ApprovePricingPolicy, nil}, {"revoke", gate.RevokePricingPolicy, nil}, {"use", gate.UsePricingPolicy, nil}, {"inspect", gate.ViewCost, []string{"pricing_policy_evidence"}},
	}
	for _, want := range expected {
		capability := want.capability
		t.Run(capability, func(t *testing.T) {
			fields := pricingTimes(f)
			fields["capability"] = capability
			console := f.run(root, "pricing_authority.propose_console", "console-"+capability, fields)
			f.ok(t, "console", console, pricingauthority.Pending, "")
			made := pricingApproveHTTP(f, second, "console-"+capability, console)
			f.ok(t, "ceiling", made, pricingauthority.Applied, "")
			terms, err := pricingGrantAuthority.terms(SampleWorkspace, capability)
			if err != nil {
				t.Fatal(err)
			}
			if terms.Action != want.action || terms.Kind != "synthetic_pricing_policy" || !reflect.DeepEqual(terms.Fields, want.fields) {
				t.Fatal("wrong independently specified capability mapping", capability)
			}
			req := gate.Request{Action: want.action, Kind: "synthetic_pricing_policy", Fields: want.fields, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
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
			human := f.run(root, "pricing_authority.propose_human", "human-"+capability, fields)
			f.ok(t, "human", human, pricingauthority.Pending, "")
			granted := pricingApproveHTTP(f, second, "human-"+capability, human)
			f.ok(t, "grant", granted, pricingauthority.Applied, "")
			err = inputAuthorityAs(f, member, func() error {
				if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
					t.Fatal("separate actual grants denied")
				}
				bad := req
				bad.Kind = inputauthority.ResourceKind
				if f.s.gate.Decide(f.s.answer, ConsoleClient, bad).Allowed {
					t.Fatal("cross-kind permission")
				}
				if capability == pricingauthority.Inspect {
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
	if Catalog(nil, map[string]gate.FieldClass{pricingauthority.EvidenceField: gate.General}).Fields[pricingauthority.EvidenceField] != gate.Cost {
		t.Fatal("pricing evidence downgraded")
	}
	if len(f.s.grantAuthorityDefinitions(0)) != 0 || f.s.grantAuthority(0) != nil {
		t.Fatal("unknown domain selected")
	}
}
func TestPricingAuthorityDomainsKeepIDsCeilingsAndFaultsSeparate(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	fields := pricingTimes(f)
	fields["capability"] = pricingauthority.Inspect
	input := f.run(root, "input_authority.propose_console", "same", fields)
	f.ok(t, "input", input, inputauthority.Pending, "")
	pricing := f.run(root, "pricing_authority.propose_console", "same", fields)
	f.ok(t, "pricing", pricing, pricingauthority.Pending, "")
	if field(input, "digest") == field(pricing, "digest") {
		t.Fatal("same ID crossed domains")
	}
	f.code(t, "cross decision", pricingApproveHTTP(f, second, "same", input), 422, string(commands.InvalidStage))
	f.s.fault = func(at string) bool { return at == "pricing-authority-after-grant" }
	inputDone := authorityApprove(f, second, "same", input)
	f.ok(t, "input isolated from pricing fault", inputDone, inputauthority.Applied, "")
	f.s.fault = nil
	pricing = f.run(root, "pricing_authority.propose_console", "fresh", fields)
	f.ok(t, "fresh pricing", pricing, pricingauthority.Pending, "")
	f.s.fault = func(at string) bool { return at == "pricing-authority-after-grant" }
	f.code(t, "pricing unknown", pricingApproveHTTP(f, second, "fresh", pricing), 500, string(commands.InternalError))
	f.s.fault = nil
	pricingDone := pricingApproveHTTP(f, second, "fresh", pricing)
	f.ok(t, "pricing reconciled", pricingDone, pricingauthority.Applied, "")
	for _, tc := range []struct{ family, ceiling string }{{"pricing_authority", field(inputDone, "grant")}, {"input_authority", field(pricingDone, "grant")}} {
		fields["recipient"] = SampleMember
		fields["ceiling"] = tc.ceiling
		f.code(t, "foreign ceiling", f.run(root, tc.family+".propose_human", "foreign", fields), 403, string(commands.PermissionDenied))
	}
	if field(inputDone, "grant") == field(pricingDone, "grant") || !strings.HasPrefix(field(pricingDone, "grant"), "pricing-grant-") {
		t.Fatal("effect IDs crossed")
	}
	err := inputAuthorityAs(f, root, func() error {
		if _, err := f.s.pricingAuthority.Lookup(field(inputDone, "grant")); !errors.Is(err, pricingauthority.ErrConflict) {
			t.Fatal("foreign binding", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPricingAuthorityPendingStoresAndRevokedUnknownStaySeparate(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	input := f.run(second, "input_authority.propose_console", "shared-id", authorityTimes(f))
	f.ok(t, "input pending", input, inputauthority.Pending, "")
	p := f.run(second, "pricing_authority.propose_console", "shared-id", pricingTimes(f))
	f.ok(t, "pricing pending", p, pricingauthority.Pending, "")
	other := f.run(second, "pricing_authority.propose_console", "second-id", pricingTimes(f))
	f.ok(t, "pricing pending two", other, pricingauthority.Pending, "")
	in, err := inputAuthorityTally(f.s)
	if err != nil {
		t.Fatal(err)
	}
	price, err := pricingAuthorityTally(f.s)
	if err != nil {
		t.Fatal(err)
	}
	if in[SampleAdmin2] != 1 || price[SampleAdmin2] != 2 {
		t.Fatal("proposal counts crossed", in, price)
	}
	f.s.fault = func(at string) bool { return at == "pricing-authority-impact" }
	if _, err := inputAuthorityTally(f.s); err != nil {
		t.Fatal("pricing fault broke input tally", err)
	}
	if _, err := pricingAuthorityTally(f.s); !errors.Is(err, gate.ErrUnavailable) {
		t.Fatal("pricing tally failure hidden", err)
	}
	f.s.fault = func(at string) bool { return at == "pricing-authority-after-grant" }
	f.code(t, "unknown effect", pricingApproveHTTP(f, root, "shared-id", p), 500, string(commands.InternalError))
	f.s.fault = nil
	id := "pricing-grant-" + field(p, "digest")
	grant, ok := f.s.grantView(id)
	if !ok {
		t.Fatal("unknown effect grant missing")
	}
	if err := f.s.admin.RevokeGrant(id, grant.Version); err != nil {
		t.Fatal(err)
	}
	before := len(f.s.admin.Grants())
	resolved := pricingApproveHTTP(f, root, "shared-id", p)
	f.ok(t, "historical resolved", resolved, pricingauthority.Applied, "")
	if field(resolved, "recorded_grant_status") != pricingauthority.RevokedGrant || len(f.s.admin.Grants()) != before {
		t.Fatal("revoked effect revived")
	}
	fields := pricingTimes(f)
	fields["recipient"] = SampleMember
	fields["ceiling"] = id
	f.code(t, "revoked ceiling cannot back human", f.run(root, "pricing_authority.propose_human", "revoked-ceiling", fields), 403, string(commands.PermissionDenied))
}
