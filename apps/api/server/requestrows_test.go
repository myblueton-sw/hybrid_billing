package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func rowIDs(f world, a actor) string {
	out := []string{}
	for _, r := range f.accessRead(a, "").body["requests"].([]any) {
		out = append(out, r.(map[string]any)["id"].(string))
	}
	return strings.Join(out, ",")
}

func TestOnlyAdministratorsAreEligibleApprovers(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	terms := gate.Terms{Action: gate.AdministerInstallation, Kind: gate.InstallationKind, Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "installation-member", SubjectID: SampleMember, Terms: terms, Validity: gate.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}}))
	asked := f.run(root, "access.request_revoke", "reader-"+SampleAdmin2, nil)
	if row := f.requestRow(root, requestOf(asked)); fmt.Sprint(row["eligible_approvers"]) != "2" {
		t.Errorf("a member holding installation administration counted as an approver %+v", row["eligible_approvers"])
	}
	f.s.caller = root.verifier(f)
	facts, err := f.s.accessFacts(f.s.caller)
	f.s.caller = identity.Verifier{}
	if err != nil || facts.eligible(SampleMember) || !facts.eligible(SampleAdmin2) {
		t.Error("eligibility")
	}
}

func TestCustomerScopedRequests(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Kinds = []string{"usage"}
		c.Fields = map[string]gate.FieldClass{"state": gate.General}
	})
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	scope := func(customer gate.Selector) gate.Terms {
		return gate.Terms{Action: gate.View, Kind: "usage", Fields: []string{"state"}, Period: gate.AnyPeriod(),
			Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: customer, LegalEntity: gate.All(), Contract: gate.All()}}
	}
	long := gate.Validity{NotBefore: start, NotAfter: start.Add(60 * 24 * time.Hour)}
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "usage-console", ClientID: ConsoleClient, Terms: scope(gate.All()), Validity: long}))
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "usage-admin", SubjectID: SampleAdmin, Terms: scope(gate.One("c1")), Validity: long}))
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "usage-admin2", SubjectID: SampleAdmin2, Terms: scope(gate.All()), Validity: long}))
	id := f.grantVia(root, second, SampleMember, map[string]string{"kind": "usage", "fields": "state", "customer": "c1", "days": "3"})
	g, _ := f.s.grantView(id)
	if g.Terms.Scope.Customer.Any() || g.Terms.Scope.Customer.ID() != "c1" {
		t.Errorf("the grant lost its customer scope %+v", g.Terms.Scope)
	}
	f.code(t, "a customer the proposer does not hold", f.requestGrant(root, SampleMember, map[string]string{"kind": "usage", "fields": "state", "customer": "c2", "days": "3"}),
		422, "prerequisite_failed")
}

func TestRequestRowFlags(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	made := f.grantVia(root, second, SampleMember, nil)
	revokeMember := f.run(root, "access.request_revoke", made, nil)
	if row := f.requestRow(second, requestOf(revokeMember)); row["can_reject"] != true || row["can_approve"] != true || row["subject"] != SampleMember {
		t.Errorf("a non-holder can reject a revocation %+v", row)
	}
	f.run(root, "access.withdraw_request", requestOf(revokeMember), nil)
	f.must(f.s.admin.RevokeGrant(made, 1))
	pending := f.requestGrant(root, SampleMember, nil)
	f.stoppedLedger()
	if row := f.requestRow(second, requestOf(pending)); row["can_approve"] != false || row["current"] != true || row["can_reject"] != true {
		t.Errorf("an addition can't be approved while the ledger can't record %+v", row)
	}
	revoke := f.run(root, "access.request_revoke", "reader-"+SampleAdmin2, nil)
	if row := f.requestRow(second, requestOf(revoke)); row["can_approve"] != true {
		t.Errorf("a revocation can be approved while the ledger can't record %+v", row)
	}
}

func TestMayEndListsOnlyEffectiveDelegationsOfTheHolder(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.grantVia(root, second, SampleMember, nil)
	addDelegation(f, "del-in", SampleAdmin, SampleAdmin2, "")
	addDelegation(f, "del-out", SampleAdmin, SampleMember, "")
	addDelegation(f, "del-closed", SampleAdmin2, SampleAdmin, "")
	f.must(f.s.admin.RevokeDelegation("del-closed", 1))
	asked := f.run(root, "access.request_revoke", "reader-"+SampleAdmin2, nil)
	if got := fmt.Sprint(f.requestRow(second, requestOf(asked))["may_end"]); got != "[del-in]" {
		t.Errorf("may end %s", got)
	}
}

func TestRequestOrderAndShownLimit(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Access = DefaultAccessSettings()
		c.Access.EndedShown = 1
	})
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	first := requestOf(f.requestGrant(root, SampleMember, nil))
	ended := []string{}
	for i := 0; i < 2; i++ {
		id := requestOf(f.run(root, "access.request_revoke", "reader-"+SampleAdmin2, nil))
		f.run(root, "access.withdraw_request", id, nil)
		ended = append(ended, id)
	}
	if got := rowIDs(f, root); got != first+","+ended[1] {
		t.Errorf("rows %s", got)
	}
}
