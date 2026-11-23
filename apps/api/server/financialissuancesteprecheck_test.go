package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func suspendThroughBulk(t *testing.T, f world, target, store string) {
	t.Helper()
	root := f.as(AdminLogin)
	bulk, digest := f.prepared(t, root, "suspend_membership", target)
	if count, rechecked := issuanceStepRow(t, f, root, bulk, store); count != 1 || rechecked != true {
		t.Fatal("impact before execution", count, rechecked)
	}
	if got := f.execute(root, bulk, digest); got.status != 202 {
		t.Fatal(got.raw)
	}
}

func TestIssuanceChannelAcceptanceRechecksSuspendedRegistrar(t *testing.T) {
	f, registrar, admitter, spec := configurationChannelWorld(t)
	p, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	suspendThroughBulk(t, f, SampleMember, "issuance_protected_steps")
	_, err = admitChannelAs(f, admitter, p)
	documentFailure(t, err, commands.PermissionDenied)
	if got := f.s.financialIssuance.channels[spec.ID].State; got != "registered" {
		t.Fatal("a channel of a suspended registrar was accepted: " + got)
	}
}

func TestIssuanceTermApplicationApprovalRechecksSuspendedAuthor(t *testing.T) {
	w := stepImpactWorld(t)
	f := w.f
	term, err := recordTermAs(f, w.requester, w.term)
	f.must(err)
	term, err = approveTermAs(f, w.approver, term)
	f.must(err)
	spec := bindingSpec(term)
	spec.Zone = w.configuration.Spec.Zone
	binding, err := bindTermAs(f, w.requester, spec)
	f.must(err)
	suspendThroughBulk(t, f, "setup-requester", "issuance_term_steps")
	_, err = approveBindingAs(f, w.approver, binding)
	documentFailure(t, err, commands.PermissionDenied)
	if got := f.s.financialIssuance.bindingIntents[spec.ID].State; got != "pending" {
		t.Fatal("an application by a suspended author was approved: " + got)
	}
}

func TestIssuanceRulesConfirmationRechecksSuspendedProposer(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p, err := proposeRulesAs(f, f.as(Admin2Login), "recheck-rules", issuanceRules())
	f.must(err)
	suspendThroughBulk(t, f, SampleAdmin2, "issuance_protected_steps")
	_, err = confirmRulesAs(f, f.as(Admin3Login), p)
	if err == nil || f.s.financialIssuance.protectedRules.State != "proposed" {
		t.Fatal("rules of a suspended proposer were confirmed", err)
	}
	f.must(inputAuthorityAs(f, f.as(Admin3Login), func() error {
		if _, standing := f.s.governor(SampleWorkspace, SampleAdmin3); standing != nil {
			t.Fatal("the confirmer lost governor standing, so the refusal does not show the proposer recheck", standing)
		}
		if _, standing := f.s.governor(SampleWorkspace, SampleAdmin2); standing == nil {
			t.Fatal("the suspended proposer kept governor standing")
		}
		return nil
	}))
}
