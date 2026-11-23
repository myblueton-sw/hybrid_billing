package server

import (
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func issuanceStepRow(t *testing.T, f world, a actor, bulk, store string) (int, any) {
	t.Helper()
	for _, raw := range f.bulkRead(a, "/"+bulk).body["impact"].(map[string]any)["pending"].([]any) {
		if row := raw.(map[string]any); row["store"] == store {
			return whole(row["count"]), row["rechecked"]
		}
	}
	t.Fatal(store + " omitted from impact")
	return 0, nil
}

func stepTally(t *testing.T, f world, tally func(*Server) (map[string]int, error), name string, want map[string]int) {
	t.Helper()
	got, err := tally(f.s)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal(name, got, err)
	}
}

func stepImpactWorld(t *testing.T) configuredSetupWorld {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	return configuredSetupIn(t, f)
}

func stepImpactFails(t *testing.T, f world, root actor, staged reply, name string) {
	t.Helper()
	f.code(t, name, f.prepare(root, field(staged, "draft"), "suspend_membership"), 503, string(commands.DependencyUnavailable))
}

func TestIssuanceProtectedStepsCountInBulkImpact(t *testing.T) {
	w := stepImpactWorld(t)
	f, a := w.f, w.f.s.financialIssuance
	none := map[string]int{}
	stepTally(t, f, issuanceProtectedStepTally, "confirmed rules and accepted channels", none)

	spec := w.change
	spec.ID, spec.Field = "impact-change", protected.FieldPaymentAccount
	spec.Channel, spec.ChannelDigest = w.channels["entity"].Spec.ID, w.channels["entity"].Digest
	p, err := requestChangeAs(f, w.requester, spec)
	f.must(err)
	stepTally(t, f, issuanceProtectedStepTally, "requested change", map[string]int{"setup-requester": 1})
	p, err = verifyChangeAs(f, w.verifier, p)
	f.must(err)
	stepTally(t, f, issuanceProtectedStepTally, "verified change awaiting a decision", map[string]int{"setup-requester": 1})

	root := f.as(AdminLogin)
	staged := f.stage(root, "", "setup-requester")
	f.ok(t, "stage", staged, "draft", "")
	before := f.s.Revision()
	f.s.fault = func(at string) bool { return at == "issuance-protected-step-impact" }
	stepImpactFails(t, f, root, staged, "impact fails closed")
	f.s.fault = nil
	saved := a.changes[spec.ID]
	for name, edit := range map[string]func(*financialChangeRecord){
		"unknown phase":     func(c *financialChangeRecord) { c.Phase = "unrecognized" },
		"unknown state":     func(c *financialChangeRecord) { c.Module.State = "unrecognized" },
		"foreign workspace": func(c *financialChangeRecord) { c.Workspace = "another-workspace" },
	} {
		corrupt := saved
		edit(&corrupt)
		a.changes[spec.ID] = corrupt
		stepImpactFails(t, f, root, staged, name+" fails closed")
	}
	a.changes[spec.ID] = saved
	channel := a.channels[spec.Channel]
	broken := channel
	broken.State = "unrecognized"
	a.channels[spec.Channel] = broken
	stepImpactFails(t, f, root, staged, "unknown channel state fails closed")
	waiting := channel
	waiting.State = "registered"
	a.channels[spec.Channel] = waiting
	stepTally(t, f, issuanceProtectedStepTally, "channel awaiting acceptance", map[string]int{"setup-requester": 1, channel.Registrar: 1})
	waiting.State = "admission_unknown"
	a.channels[spec.Channel] = waiting
	stepTally(t, f, issuanceProtectedStepTally, "channel acceptance with an unknown outcome", map[string]int{"setup-requester": 1, channel.Registrar: 1})
	a.channels[spec.Channel] = channel
	rules := *a.protectedRules
	proposed := rules
	proposed.State = "proposed"
	a.protectedRules = &proposed
	stepTally(t, f, issuanceProtectedStepTally, "rules awaiting confirmation", map[string]int{"setup-requester": 1, rules.Proposer: 1})
	proposed.State = "unrecognized"
	stepImpactFails(t, f, root, staged, "unknown rules state fails closed")
	a.protectedRules = &rules
	if f.s.Revision() != before {
		t.Fatal("impact failure mutated revision")
	}
	list := f.prepare(root, field(staged, "draft"), "suspend_membership")
	f.ok(t, "prepare impact", list, "prepared", "")
	if count, rechecked := issuanceStepRow(t, f, root, field(list, "bulk"), "issuance_protected_steps"); count != 1 || rechecked != true {
		t.Fatal("bulk impact row", count, rechecked)
	}
	if count, _ := issuanceStepRow(t, f, root, field(list, "bulk"), "issuance_term_steps"); count != 0 {
		t.Fatal("term steps counted for a change requester", count)
	}

	_, err = decideChangeAs(f, w.approver, p)
	f.must(err)
	stepTally(t, f, issuanceProtectedStepTally, "decided change", none)
}

func TestIssuanceChangeDecisionRechecksSuspendedRequester(t *testing.T) {
	w := stepImpactWorld(t)
	f := w.f
	spec := w.change
	spec.ID, spec.Field = "recheck-change", protected.FieldPaymentAccount
	spec.Channel, spec.ChannelDigest = w.channels["entity"].Spec.ID, w.channels["entity"].Digest
	p, err := requestChangeAs(f, w.requester, spec)
	f.must(err)
	p, err = verifyChangeAs(f, w.verifier, p)
	f.must(err)
	root := f.as(AdminLogin)
	bulk, digest := f.prepared(t, root, "suspend_membership", "setup-requester")
	if count, rechecked := issuanceStepRow(t, f, root, bulk, "issuance_protected_steps"); count != 1 || rechecked != true {
		t.Fatal("impact before execution", count, rechecked)
	}
	if got := f.execute(root, bulk, digest); got.status != 202 {
		t.Fatal(got.raw)
	}
	_, err = decideChangeAs(f, w.approver, p)
	documentFailure(t, err, commands.PermissionDenied)
	if f.s.financialIssuance.changes[spec.ID].Module.State != protected.StateConfirmed {
		t.Fatal("a change of a suspended requester was decided")
	}
}

func TestIssuanceTermStepsCountInBulkImpactAndRecheckAuthor(t *testing.T) {
	w := stepImpactWorld(t)
	f, a := w.f, w.f.s.financialIssuance
	none := map[string]int{}
	stepTally(t, f, issuanceTermStepTally, "before any term", none)
	term, err := recordTermAs(f, w.requester, w.term)
	f.must(err)
	stepTally(t, f, issuanceTermStepTally, "term awaiting approval", map[string]int{"setup-requester": 1})

	root := f.as(AdminLogin)
	staged := f.stage(root, "", "setup-requester")
	f.ok(t, "stage", staged, "draft", "")
	before := f.s.Revision()
	f.s.fault = func(at string) bool { return at == "issuance-term-step-impact" }
	stepImpactFails(t, f, root, staged, "impact fails closed")
	f.s.fault = nil
	saved := a.termIntents[term.Spec.ID]
	corrupt := saved
	corrupt.State = "unrecognized"
	a.termIntents[term.Spec.ID] = corrupt
	stepImpactFails(t, f, root, staged, "unknown term state fails closed")
	approved := saved
	approved.State = "approved"
	a.termIntents[term.Spec.ID] = approved
	stepTally(t, f, issuanceTermStepTally, "approved term", none)
	a.bindingIntents["impact-binding"] = financialTermBinding{Spec: financialTermBindingSpec{ID: "impact-binding"}, Term: approved, Author: "setup-requester", State: "pending"}
	stepTally(t, f, issuanceTermStepTally, "application awaiting approval", map[string]int{"setup-requester": 1})
	a.bindingIntents["impact-binding"] = financialTermBinding{Spec: financialTermBindingSpec{ID: "impact-binding"}, Term: approved, Author: "setup-requester", State: "unrecognized"}
	stepImpactFails(t, f, root, staged, "unknown application state fails closed")
	delete(a.bindingIntents, "impact-binding")
	a.termIntents[term.Spec.ID] = saved
	if f.s.Revision() != before {
		t.Fatal("impact failure mutated revision")
	}
	list := f.prepare(root, field(staged, "draft"), "suspend_membership")
	f.ok(t, "prepare impact", list, "prepared", "")
	if count, rechecked := issuanceStepRow(t, f, root, field(list, "bulk"), "issuance_term_steps"); count != 1 || rechecked != true {
		t.Fatal("bulk impact row", count, rechecked)
	}
	if got := f.execute(root, field(list, "bulk"), field(list, "digest")); got.status != 202 {
		t.Fatal(got.raw)
	}
	_, err = approveTermAs(f, w.approver, term)
	documentFailure(t, err, commands.PermissionDenied)
	if a.termIntents[term.Spec.ID].State != "pending" {
		t.Fatal("a term of a suspended author was approved")
	}
}

func TestIssuanceChannelAcceptanceUnknownCountsInBulkImpact(t *testing.T) {
	f, registrar, admitter, spec := configurationChannelWorld(t)
	p, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	f.s.fault = func(at string) bool { return at == "issuance-channel-after-admit" }
	_, err = admitChannelAs(f, admitter, p)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	if got := f.s.financialIssuance.channels[spec.ID].State; got != "admission_unknown" {
		t.Fatal("fixture state " + got)
	}
	root := f.as(AdminLogin)
	bulk, _ := f.prepared(t, root, "suspend_membership", SampleMember)
	if count, rechecked := issuanceStepRow(t, f, root, bulk, "issuance_protected_steps"); count != 1 || rechecked != true {
		t.Fatal("channel with an unknown acceptance outcome", count, rechecked)
	}
}
