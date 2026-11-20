package server

import (
	"errors"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
)

func TestPricingPolicyPendingImpactCountsOnlySelectedProposers(t *testing.T) {
	f, maker, checker, spec, _ := policyHTTPWorld(t)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	ceiling := ""
	for _, g := range f.s.admin.Grants() {
		if g.ClientID == ConsoleClient && g.Terms.Action == gate.ProposePricingPolicy {
			ceiling = g.ID
		}
	}
	fields := pricingTimes(f)
	fields["capability"] = "propose"
	fields["recipient"] = SampleAdmin3
	fields["ceiling"] = ceiling
	p := f.run(root, "pricing_authority.propose_human", "other-policy-maker", fields)
	f.ok(t, "other maker proposal", p, pricingauthority.Pending, "")
	f.ok(t, "other maker granted", pricingApproveHTTP(f, second, "other-policy-maker", p), pricingauthority.Applied, "")
	var first reply
	for i, id := range []string{"pending-one", "pending-two", "other-maker-policy"} {
		prepared := policyRead(f, maker, "prepare", id, map[string]string{"spec": spec})
		a := maker
		if i == 2 {
			a = checker
		}
		result := policyMutate(f, a, "propose", id, policyProposalFields(prepared, spec))
		f.ok(t, "policy proposal", result, pricingpolicy.Pending, "")
		if i == 0 {
			first = result
		}
	}
	tally, err := pricingPolicyTally(f.s)
	if err != nil || tally[SampleMember] != 2 || tally[SampleAdmin3] != 1 {
		t.Fatal(tally, err)
	}
	job, _ := f.prepared(t, root, "suspend_membership", SampleMember)
	impact := f.bulkRead(root, "/"+job).body["impact"].(map[string]any)
	counts := map[string]int{}
	for _, value := range impact["pending"].([]any) {
		row := value.(map[string]any)
		counts[row["store"].(string)] = whole(row["count"])
		if row["store"] == "pricing_policy_proposals" && row["rechecked"] != true {
			t.Fatal("proposer recheck not exposed", row)
		}
	}
	if counts["pricing_policy_proposals"] != 2 || counts["pricing_authority_proposals"] != 0 {
		t.Fatal("selected proposal categories crossed", counts)
	}
	f.ok(t, "approve first", policyMutate(f, checker, "approve", "pending-one", map[string]string{"digest": field(first, "digest")}), pricingpolicy.Approved, "")
	tally, err = pricingPolicyTally(f.s)
	if err != nil || tally[SampleMember] != 1 {
		t.Fatal("approved counted pending", tally, err)
	}
	f.ok(t, "revoke first", policyMutate(f, checker, "revoke", "pending-one", map[string]string{"digest": field(first, "digest")}), pricingpolicy.Revoked, "")
	tally, err = pricingPolicyTally(f.s)
	if err != nil || tally[SampleMember] != 1 {
		t.Fatal("revoked counted pending", tally, err)
	}
	staged := f.stage(root, "", SampleMember)
	f.s.fault = func(at string) bool { return at == "pricing-policy-impact" }
	if _, err := pricingPolicyTally(f.s); !errors.Is(err, gate.ErrUnavailable) {
		t.Fatal(err)
	}
	f.code(t, "unavailable impact", f.prepare(root, field(staged, "draft"), "suspend_membership"), 503, string(commands.DependencyUnavailable))
}

func TestPricingPolicyInspectionOutputBounds(t *testing.T) {
	f, maker, checker, spec, _ := policyHTTPWorld(t)
	id := "bounded-output"
	p := policyRead(f, maker, "prepare", id, map[string]string{"spec": spec})
	proposal := policyMutate(f, maker, "propose", id, policyProposalFields(p, spec))
	f.ok(t, "proposal", proposal, pricingpolicy.Pending, "")
	for _, state := range []string{pricingpolicy.Pending, pricingpolicy.Approved, pricingpolicy.Revoked} {
		if state != pricingpolicy.Pending {
			kind := "approve"
			if state == pricingpolicy.Revoked {
				kind = "revoke"
			}
			f.ok(t, kind, policyMutate(f, checker, kind, id, map[string]string{"digest": field(proposal, "digest")}), state, "")
		}
		read := policyRead(f, maker, "inspect", id, nil)
		if read.status != 200 {
			t.Fatal(read.raw)
		}
		fields := read.body["fields"].(map[string]any)
		if len(fields) > 19 || state == pricingpolicy.Revoked && len(fields) != 19 {
			t.Fatal("inspection field contract", state, len(fields))
		}
		for name, value := range fields {
			if len(value.(string)) > f.s.cfg.Commands.Value {
				t.Fatal("oversized field", name)
			}
		}
	}
	before := f.s.counter
	for _, limits := range []struct {
		fields, value int
		allowed       bool
	}{{18, 4096, false}, {19, 1023, false}, {19, 1024, true}} {
		f.s.cfg.Commands.Fields, f.s.cfg.Commands.Value = limits.fields, limits.value
		for _, kind := range []string{"prepare", "propose", "approve", "revoke", "inspect"} {
			if limits.allowed && kind != "inspect" {
				continue
			}
			err := inputAuthorityAs(f, maker, func() error {
				_, err := (pricingPolicyCommand{s: f.s, kind: kind}).Apply(commands.Context{Workspace: SampleWorkspace, User: SampleMember, Target: id})
				return err
			})
			var failure *commands.Failure
			if limits.allowed {
				if err != nil {
					t.Fatal("exact supported threshold", err)
				}
			} else if !errors.As(err, &failure) || failure.Code != commands.UnsupportedCapability {
				t.Fatal("unsupported capacity admitted", kind, err)
			}
		}
	}
	if f.s.counter != before {
		t.Fatal("capacity checks mutated revision")
	}
}
