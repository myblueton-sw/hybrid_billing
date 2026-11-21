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

func TestFinancialAuthorityCurrentActors(t *testing.T) {
	for _, mode := range []string{"recipient_removed", "recipient_suspended", "proposer_removed", "proposer_revoked", "confirmer_revoked", "console_revoked", "session_ended"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root, second := f.as(AdminLogin), f.as(Admin2Login)
			p := f.run(root, "financial_authority.propose", "actors", financialAuthorityFields(f))
			f.ok(t, "propose", p, approvals.StatePending, "")
			switch mode {
			case "recipient_removed":
				f.must(f.s.identity.Workspace().RemoveMembership(root.verifier(f), SampleWorkspace, SampleAdmin3, f.membership(SampleAdmin3).Version))
			case "recipient_suspended":
				f.must(f.s.identity.Workspace().SuspendMembership(root.verifier(f), SampleWorkspace, SampleAdmin3, f.membership(SampleAdmin3).Version))
			case "proposer_removed":
				f.must(f.s.identity.Workspace().RemoveMembership(second.verifier(f), SampleWorkspace, SampleAdmin, f.membership(SampleAdmin).Version))
			case "session_ended":
				f.must(f.s.identity.Boundary().EndSession(second.verifier(f)))
			default:
				for _, g := range f.s.admin.Grants() {
					if g.Terms.Action == gate.AdministerInstallation && (mode == "console_revoked" && g.ClientID == ConsoleClient || mode == "proposer_revoked" && g.SubjectID == SampleAdmin || mode == "confirmer_revoked" && g.SubjectID == SampleAdmin2) {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
			}
			status, code := 403, string(commands.PermissionDenied)
			if mode == "session_ended" {
				status, code = 401, string(commands.Unauthenticated)
			}
			f.code(t, mode, f.runWith(second, f.s.Revision(), "financial_authority.confirm", "actors", map[string]string{"digest": field(p, "digest")}), status, code)
			if _, err := f.s.approvals.AuthorityOf(SampleWorkspace, "financial-limit-actors"); !errors.Is(err, approvals.ErrNotFound) {
				t.Fatal("refusal created authority", err)
			}
		})
	}
}

func TestFinancialAuthorityRecipientCannotConfirmAndCompetingGovernors(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, third := f.as(AdminLogin), f.as(Admin2Login), f.as(Admin3Login)
	approvalAdminister(t, f.s, approvals.Proposal{ID: "third-governor", Kind: approvals.ProposeAddAdmin, Admin: SampleAdmin3})
	p := f.run(root, "financial_authority.propose", "recipient", financialAuthorityFields(f))
	f.ok(t, "propose", p, approvals.StatePending, "")
	f.code(t, "recipient confirms", f.run(third, "financial_authority.confirm", "recipient", map[string]string{"digest": field(p, "digest")}), 403, string(commands.SelfApprovalDenied))
	fields := financialAuthorityFields(f)
	fields["recipient"] = SampleMember
	p = f.run(root, "financial_authority.propose", "competing", fields)
	f.ok(t, "propose", p, approvals.StatePending, "")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, who := range []struct {
		a  actor
		id string
	}{{second, SampleAdmin2}, {third, SampleAdmin3}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- inputAuthorityAs(f, who.a, func() error {
				_, err := f.s.confirmFinancialAuthority(commands.Context{Workspace: SampleWorkspace, User: who.id, Target: "competing", Fields: map[string]string{"digest": field(p, "digest")}, Rationale: "Independent limit review"})
				return err
			})
		}()
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, approvals.ErrConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("competing confirmation", success, conflict)
	}
}

func TestFinancialAuthorityImmutableReasonsBindingsAndCapacity(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p := f.run(root, "financial_authority.propose", "immutable", financialAuthorityFields(f))
	f.ok(t, "propose", p, approvals.StatePending, "")
	ctx := commands.Context{Workspace: SampleWorkspace, User: SampleAdmin, Target: "immutable", Fields: financialAuthorityFields(f), Rationale: "changed"}
	err := inputAuthorityAs(f, root, func() error { _, err := f.s.proposeFinancialAuthority(ctx); return err })
	if !errors.Is(err, approvals.ErrConflict) {
		t.Fatal("proposal reason rewritten", err)
	}
	digest := map[string]string{"digest": field(p, "digest")}
	f.ok(t, "confirm", f.run(second, "financial_authority.confirm", "immutable", digest), approvals.StateConfirmed, "")
	ctx.User, ctx.Fields = SampleAdmin2, digest
	err = inputAuthorityAs(f, second, func() error { _, err := f.s.confirmFinancialAuthority(ctx); return err })
	if !errors.Is(err, approvals.ErrConflict) {
		t.Fatal("confirmation reason rewritten", err)
	}
	f.ok(t, "revoke", f.run(root, "financial_authority.revoke", "immutable", digest), "revoked", "")
	ctx.User = SampleAdmin
	err = inputAuthorityAs(f, root, func() error { _, err := f.s.revokeFinancialAuthority(ctx); return err })
	if !errors.Is(err, approvals.ErrConflict) {
		t.Fatal("revocation reason rewritten", err)
	}
	original := f.s.financialAuthorities["immutable"]
	broken := original
	broken.Digest = "changed"
	f.s.financialAuthorities["immutable"] = broken
	if _, err := f.s.financialAuthorityRecord(SampleWorkspace, "immutable"); !errors.Is(err, approvals.ErrConflict) {
		t.Fatal("corrupt binding", err)
	}
	f.s.financialAuthorities["immutable"] = original
	delete(f.s.financialAuthorities, "immutable")
	if _, err := f.s.financialAuthorityRecord(SampleWorkspace, "immutable"); !errors.Is(err, approvals.ErrNotFound) {
		t.Fatal("unknown adopted", err)
	}
	f.s.financialAuthorities["immutable"] = original
	for len(f.s.financialAuthorities) < 100 {
		f.s.financialAuthorities[fmt.Sprintf("full-%d", len(f.s.financialAuthorities))] = financialAuthorityBinding{}
	}
	ctx.Target, ctx.Fields, ctx.Rationale = "overflow", financialAuthorityFields(f), "Explicit limit"
	err = inputAuthorityAs(f, root, func() error { _, err := f.s.proposeFinancialAuthority(ctx); return err })
	if !errors.Is(err, approvals.ErrLimit) {
		t.Fatal("capacity", err)
	}
	if _, err := f.s.approvals.ProposalOf(SampleWorkspace, "overflow"); !errors.Is(err, approvals.ErrNotFound) {
		t.Fatal("capacity mutated module", err)
	}
}

func TestFinancialAuthorityPendingImpact(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p := f.run(second, "financial_authority.propose", "impact", financialAuthorityFields(f))
	f.ok(t, "propose", p, approvals.StatePending, "")
	tally, err := financialAuthorityTally(f.s)
	if err != nil || len(tally) != 1 || tally[SampleAdmin2] != 1 {
		t.Fatal(tally, err)
	}
	staged := f.stage(root, "", SampleAdmin2)
	f.s.fault = func(at string) bool { return at == "financial-authority-impact" }
	f.code(t, "unavailable", f.prepare(root, field(staged, "draft"), "suspend_membership"), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	f.ok(t, "confirm", f.run(root, "financial_authority.confirm", "impact", map[string]string{"digest": field(p, "digest")}), approvals.StateConfirmed, "")
	tally, err = financialAuthorityTally(f.s)
	if err != nil || len(tally) != 0 {
		t.Fatal("confirmed pending", tally, err)
	}
}
