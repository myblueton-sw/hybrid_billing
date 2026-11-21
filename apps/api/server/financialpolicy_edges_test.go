package server

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFinancialPolicyCurrentGovernorRefusals(t *testing.T) {
	for _, mode := range []string{"proposer_suspended", "proposer_removed", "confirmer_suspended", "confirmer_removed", "session_ended", "console_revoked", "confirmer_revoked"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root, second := f.as(AdminLogin), f.as(Admin2Login)
			p := f.run(root, "financial_policy.propose", "pending", financialFields("2"))
			f.ok(t, "propose", p, approvals.StatePending, "")
			switch mode {
			case "proposer_suspended":
				f.must(f.s.identity.Workspace().SuspendMembership(second.verifier(f), SampleWorkspace, SampleAdmin, f.membership(SampleAdmin).Version))
			case "proposer_removed":
				f.must(f.s.identity.Workspace().RemoveMembership(second.verifier(f), SampleWorkspace, SampleAdmin, f.membership(SampleAdmin).Version))
			case "confirmer_suspended":
				f.must(f.s.identity.Workspace().SuspendMembership(root.verifier(f), SampleWorkspace, SampleAdmin2, f.membership(SampleAdmin2).Version))
			case "confirmer_removed":
				f.must(f.s.identity.Workspace().RemoveMembership(root.verifier(f), SampleWorkspace, SampleAdmin2, f.membership(SampleAdmin2).Version))
			case "session_ended":
				f.must(f.s.identity.Boundary().EndSession(second.verifier(f)))
			default:
				for _, grant := range f.s.admin.Grants() {
					if grant.Terms.Action == gate.AdministerInstallation && (mode == "console_revoked" && grant.ClientID == ConsoleClient || mode == "confirmer_revoked" && grant.SubjectID == SampleAdmin2) {
						f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
					}
				}
			}
			got := f.runWith(second, f.s.Revision(), "financial_policy.confirm", "pending", map[string]string{"digest": field(p, "digest")})
			if got.status == 200 {
				t.Fatal("lost authority confirmed", got.raw)
			}
			record, err := f.s.approvals.ProposalOf(SampleWorkspace, "pending")
			if err != nil || record.State != approvals.StatePending || f.s.financialPolicies["pending"].ConfirmationReason != "" {
				t.Fatal("refusal mutated policy", record, err)
			}
		})
	}
}

func TestFinancialPolicyRationaleCapacityAndConcurrentConfirmation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p := f.run(root, "financial_policy.propose", "first", financialFields("2"))
	f.ok(t, "propose", p, approvals.StatePending, "")
	reason := f.s.financialPolicies["first"].ProposalReason
	err := inputAuthorityAs(f, root, func() error {
		_, _, err := f.s.proposeFinancialPolicy(commands.Context{Workspace: SampleWorkspace, User: SampleAdmin, Target: "first", Fields: financialFields("2"), Rationale: reason + " changed"})
		return err
	})
	if !errors.Is(err, approvals.ErrConflict) {
		t.Fatal("proposal rationale changed", err)
	}
	q := f.run(root, "financial_policy.propose", "other", financialFields("2"))
	f.ok(t, "second proposal", q, approvals.StatePending, "")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, record := range []reply{p, q} {
		id := "first"
		if field(record, "proposal") == "other" {
			id = "other"
		}
		digest := field(record, "digest")
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- inputAuthorityAs(f, second, func() error {
				_, _, err := f.s.confirmFinancialPolicy(commands.Context{Workspace: SampleWorkspace, User: SampleAdmin2, Target: id, Fields: map[string]string{"digest": digest}, Rationale: "Independent confirmation"})
				return err
			})
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("competing revisions confirmed", success)
	}
	var confirmed approvals.ProposalView
	for _, id := range []string{"first", "other"} {
		record, _ := f.s.approvals.ProposalOf(SampleWorkspace, id)
		if record.State == approvals.StateConfirmed {
			confirmed = record
		}
	}
	err = inputAuthorityAs(f, second, func() error {
		_, _, err := f.s.confirmFinancialPolicy(commands.Context{Workspace: SampleWorkspace, User: SampleAdmin2, Target: confirmed.Proposal.ID, Fields: map[string]string{"digest": confirmed.Digest}, Rationale: "Changed confirmation"})
		return err
	})
	if !errors.Is(err, approvals.ErrConflict) {
		t.Fatal("confirmation rationale changed", err)
	}
	for len(f.s.financialPolicies) < 100 {
		f.s.financialPolicies[fmt.Sprintf("capacity-%d", len(f.s.financialPolicies))] = financialPolicyBinding{}
	}
	err = inputAuthorityAs(f, root, func() error {
		_, _, err := f.s.proposeFinancialPolicy(commands.Context{Workspace: SampleWorkspace, User: SampleAdmin, Target: "overflow", Fields: financialFields("3"), Rationale: reason})
		return err
	})
	if !errors.Is(err, approvals.ErrLimit) {
		t.Fatal("unbounded sidecar", err)
	}
	if _, err := f.s.approvals.ProposalOf(SampleWorkspace, "overflow"); !errors.Is(err, approvals.ErrNotFound) {
		t.Fatal("capacity refusal mutated module", err)
	}
}

func TestFinancialPolicyImpactFailureAndHold(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p := f.run(second, "financial_policy.propose", "pending", financialFields("2"))
	f.ok(t, "propose", p, approvals.StatePending, "")
	tally, err := financialPolicyTally(f.s)
	if err != nil || tally[SampleAdmin2] != 1 || len(tally) != 1 {
		t.Fatal(tally, err)
	}
	staged := f.stage(root, "", SampleAdmin2)
	f.s.fault = func(at string) bool { return at == "financial-policy-impact" }
	f.code(t, "unavailable impact", f.prepare(root, field(staged, "draft"), "suspend_membership"), 503, "dependency_unavailable")
	f.s.fault = nil
	f.s.enterHold("synthetic review")
	if got := f.run(root, "financial_policy.confirm", "pending", map[string]string{"digest": field(p, "digest")}); got.status == 200 {
		t.Fatal("held confirmation", got.raw)
	}
	if got := financialRead(f, root, "inspect", "pending"); got.status != 200 {
		t.Fatal("held inspect", got.raw)
	}
	f.s.hold = nil
	f.ok(t, "confirm", f.run(root, "financial_policy.confirm", "pending", map[string]string{"digest": field(p, "digest")}), approvals.StateConfirmed, "")
	tally, err = financialPolicyTally(f.s)
	if err != nil || len(tally) != 0 {
		t.Fatal("confirmed counted pending", tally, err)
	}
}
