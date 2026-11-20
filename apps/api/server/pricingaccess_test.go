package server

import (
	"fmt"
	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
	"strings"
	"testing"
	"time"
)

func TestPricingAccessRejectsGenericGrantBypass(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	scope := gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}
	cases := []gate.Terms{}
	for _, action := range []gate.Action{gate.ProposePricingPolicy, gate.ApprovePricingPolicy, gate.RevokePricingPolicy, gate.UsePricingPolicy} {
		cases = append(cases, gate.Terms{Action: action, Kind: ReaderKind, Scope: scope, Period: gate.AnyPeriod()})
	}
	cases = append(cases, gate.Terms{Action: gate.View, Kind: pricingauthority.ResourceKind, Fields: []string{ReaderField}, Scope: scope, Period: gate.AnyPeriod()}, gate.Terms{Action: gate.ViewCost, Kind: ReaderKind, Fields: []string{pricingauthority.EvidenceField}, Scope: scope, Period: gate.AnyPeriod()})
	for n, terms := range cases {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			before := len(f.s.admin.Grants())
			validity := gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(48 * time.Hour)}
			for _, subject := range []string{"", SampleAdmin, SampleAdmin2} {
				g := gate.Grant{ID: fmt.Sprintf("pricing-fixture-%d-%s", n, subject), SubjectID: subject, Terms: terms, Validity: validity}
				if subject == "" {
					g.ClientID = ConsoleClient
				}
				if err := f.s.admin.AddGrant(g); err != nil {
					t.Fatal(err)
				}
			}
			for _, row := range f.s.ceiling() {
				if pricingGrantTerms(gate.Action(row.Action), row.Kind, row.Fields) {
					t.Fatal("dedicated pricing exposed as generic ceiling")
				}
			}
			fields := map[string]string{"action": string(terms.Action), "kind": terms.Kind, "fields": strings.Join(terms.Fields, ","), "days": "1"}
			f.code(t, "crafted generic request", f.requestGrant(root, SampleMember, fields), 422, string(commands.InvalidRequest))
			change := accesschange.Change{Kind: accesschange.ChangeAdd, Subject: SampleMember, Action: string(terms.Action), ResourceKind: terms.Kind, Fields: terms.Fields, Customer: "all", Days: 1}
			var retained accesschange.View
			err := inputAuthorityAs(f, root, func() error {
				facts, err := f.s.accessFacts(f.s.caller)
				if err != nil {
					return err
				}
				if _, ok := facts.ceilingFor(change); ok {
					t.Fatal("pricing ceiling discovered")
				}
				retained, _, err = f.s.changes.Propose(accesschange.Proposal{Workspace: SampleWorkspace, Change: change, Basis: facts.basis(change, SampleAdmin), CeilingUntil: validity.NotAfter, Proposer: accesschange.Actor{User: SampleAdmin, Human: true}, Allowed: true, Rationale: "Retained request fixture"})
				if err != nil {
					return err
				}
				if facts.standing(retained, SampleAdmin2).Allowed {
					t.Fatal("retained pricing request authorized")
				}
				if id, err := f.s.applyAdd(retained, f.s.clock()); err == nil || id != "" {
					t.Fatal("final application bypass", err)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			refused := f.run(second, "access.approve_request", retained.ID, map[string]string{"digest": retained.Digest})
			f.code(t, "retained pricing request refused", refused, 422, string(commands.PrerequisiteFailed))
			if len(f.s.admin.Grants()) != before+3 {
				t.Fatal("generic request published grant")
			}
		})
	}
	id := "pricing-fixture-0-" + SampleAdmin
	got := f.revokeVia(second, root, id)
	if got.status != 202 || got.body["status"] != "accepted" {
		t.Fatal("explicit pricing grant revocation blocked", got.raw)
	}
	grant, ok := f.s.grantView(id)
	if !ok || !grant.Closed || grant.Active {
		t.Fatal("revocation did not close pricing grant")
	}
}
