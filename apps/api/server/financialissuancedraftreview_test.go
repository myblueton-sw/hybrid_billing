package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
	"reflect"
	"testing"
	"time"
)

func draftApprovalWorld(t *testing.T, ages ...time.Duration) (world, actor, actor, financialDraftRecord, financialTermBinding) {
	t.Helper()
	f, _, _, _, profile, _ := referenceConsumptionWorld(t, ages...)
	for _, right := range []struct{ capability, recipient string }{{termsauthority.RecordTerm, SampleMember}, {termsauthority.ApproveTerm, SampleAdmin3}, {termsauthority.BindTerm, SampleMember}, {termsauthority.ApproveBinding, SampleAdmin3}} {
		grantTermCapability(t, f, right.capability, right.recipient)
	}
	member, decider := f.as(MemberLogin), f.as(Admin3Login)
	term, err := recordTermAs(f, member, financialTermSpec{ID: "review-term", Profile: profile.Spec.ID, ProfileDigest: profile.Digest, Mode: receivables.ModeCalendar, Revision: 1, NetDays: 30, Reason: "Draft approval calendar"})
	f.must(err)
	term, err = approveTermAs(f, decider, term)
	f.must(err)
	bs := bindingSpec(term)
	bs.Zone = profile.Spec.Zone
	binding, err := bindTermAs(f, member, bs)
	f.must(err)
	binding, err = approveBindingAs(f, decider, binding)
	f.must(err)
	grantDraftCapability(f, draftauthority.Submit)
	grantDraftCapability(f, draftauthority.Inspect)
	login := activateProtectedMember(f, "draft-reviewer")
	grantDraftReviewer(f, "review-", "draft-reviewer")
	member = f.as(MemberLogin)
	draft, err := submitDraftAs(f, member, financialDraftSpec{ID: "review-draft", Profile: profile.Spec.ID, ProfileDigest: profile.Digest, Language: "en"})
	f.must(err)
	return f, member, f.as(login), draft, binding
}
func prepareDraftReviewAs(f world, who actor, id string, p financialDraftRecord, b financialTermBinding) (financialDraftReviewResult, error) {
	var out financialDraftReviewResult
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.prepareDraftApproval(SampleWorkspace, f.s.answer.User(), id, p.Spec.ID, p.Digest, b.Spec.ID, b.Digest)
		return err
	})
	return out, err
}
func approveDraftAs(f world, who actor, r financialDraftReview) (financialDraftReviewResult, error) {
	var out financialDraftReviewResult
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.approveDraft(SampleWorkspace, f.s.answer.User(), r.ID, r.Digest)
		return err
	})
	return out, err
}
func TestFinancialDraftApprovalActualAndRecovery(t *testing.T) {
	for _, point := range []string{"", "before-intent", "after-intent", "after-module", "after-publication"} {
		t.Run(point, func(t *testing.T) {
			f, member, reviewer, draft, binding := draftApprovalWorld(t)
			a := f.s.financialIssuance
			prepared, err := prepareDraftReviewAs(f, reviewer, "review-1", draft, binding)
			f.must(err)
			if prepared.Review.State != "prepared" || len(prepared.Review.References) != 3 || prepared.Review.Coverage.Binding == "" {
				t.Fatal("missing actual review evidence")
			}
			if point != "" {
				f.s.fault = func(at string) bool { return at == "issuance-draft-review-"+point }
				_, err = approveDraftAs(f, reviewer, prepared.Review)
				documentFailure(t, err, commands.DependencyUnavailable)
				f.s.fault = nil
				if point == "after-intent" {
					retained := a.draftReviews[prepared.Review.ID]
					actual, err := a.documents.DraftOf(SampleWorkspace, draft.Spec.ID)
					f.must(err)
					if retained.State != "approval_unknown" || retained.Module.Digest != "" || !reflect.DeepEqual(actual, draft.Module) {
						t.Fatal("intent boundary wrote module approval")
					}
				}
				replay, err := submitDraftAs(f, member, draft.Spec)
				f.must(err)
				if !reflect.DeepEqual(replay, draft) {
					t.Fatal("approval interruption changed submission receipt")
				}
			}
			approved, err := approveDraftAs(f, reviewer, prepared.Review)
			f.must(err)
			if approved.Review.State != "approved" || approved.Receipt != "historical_approval" || approved.Eligibility != "unchecked" || approved.Review.Module.Approver != "draft-reviewer" || approved.Review.Module.Number != "" {
				t.Fatal("missing actual approval receipt")
			}
			actual, err := a.documents.DraftOf(SampleWorkspace, draft.Spec.ID)
			f.must(err)
			if !reflect.DeepEqual(actual, approved.Review.Module) || !reflect.DeepEqual(a.drafts[draft.Spec.ID], draft) {
				t.Fatal("module approval or immutable submission mismatch")
			}
			replay, err := approveDraftAs(f, reviewer, prepared.Review)
			f.must(err)
			if !reflect.DeepEqual(replay, approved) {
				t.Fatal("approval replay changed receipt")
			}
			_, err = prepareDraftReviewAs(f, reviewer, "review-2", draft, binding)
			documentFailure(t, err, commands.PrerequisiteFailed)
		})
	}
}

func TestFinancialDraftHistoricalApprovalAndDisclosure(t *testing.T) {
	f, _, reviewer, draft, binding := draftApprovalWorld(t)
	prepared, err := prepareDraftReviewAs(f, reviewer, "historical-review", draft, binding)
	f.must(err)
	approved, err := approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	key := [2]string{SampleWorkspace, "public-result"}
	application := f.s.documentERP.resultApplications[key]
	application.receipt = "changed-source"
	f.s.documentERP.resultApplications[key] = application
	replay, err := approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	if !reflect.DeepEqual(replay, approved) {
		t.Fatal("historical approval rebound current source")
	}
	f.must(inputAuthorityAs(f, reviewer, func() error {
		_, err := f.s.financialIssuance.liveDraftApproval(draft, "draft-reviewer")
		documentFailure(t, err, commands.EvidenceStale)
		return nil
	}))
	approved.Review.References[0].Change = "mutated"
	approved.Review.Module.Draft.Lines[0].Amount = "999"
	approved.Review.Module.Coverage.Zone = "changed"
	again, err := approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	if !reflect.DeepEqual(again, replay) {
		t.Fatal("returned approval aliases retained evidence")
	}
	revoked := 0
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == "draft-reviewer" && g.Terms.Kind == draftauthority.ResourceKind && g.Terms.Action == gate.ApproveIssuanceDraft {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
			revoked++
		}
	}
	if revoked == 0 {
		t.Fatal("no current approval authority revoked")
	}
	_, err = approveDraftAs(f, reviewer, prepared.Review)
	documentFailure(t, err, commands.PermissionDenied)
}
func TestFinancialDraftApprovalForeignAndBrokenOwnership(t *testing.T) {
	for _, mode := range []string{"foreign", "orphan", "missing-order", "wrong-draft", "malformed-module"} {
		t.Run(mode, func(t *testing.T) {
			f, _, reviewer, draft, binding := draftApprovalWorld(t)
			a := f.s.financialIssuance
			prepared, err := prepareDraftReviewAs(f, reviewer, "ownership-review", draft, binding)
			f.must(err)
			r := prepared.Review
			switch mode {
			case "foreign":
				_, err = a.documents.ApproveReviewed(SampleWorkspace, draft.Spec.ID, draft.Module.Digest, r.References, &r.Coverage, issuance.Actor{ID: "draft-reviewer", Kind: issuance.KindHuman})
				f.must(err)
			case "orphan":
				orphan := copyDraftReview(r)
				orphan.ID = "orphan-review"
				orphan.Digest, err = draftReviewDigest(orphan)
				f.must(err)
				a.draftReviews[orphan.ID] = orphan
			case "missing-order":
				delete(a.draftReviewOrders, draft.Spec.ID)
			case "wrong-draft":
				r.DraftID = "other-draft"
				a.draftReviews[r.ID] = r
			case "malformed-module":
				f.s.fault = func(at string) bool { return at == "issuance-draft-review-after-module" }
				_, err = approveDraftAs(f, reviewer, r)
				documentFailure(t, err, commands.DependencyUnavailable)
				f.s.fault = nil
				retained := a.draftReviews[r.ID]
				retained.Module.Submitter = "foreign-submitter"
				a.draftReviews[r.ID] = retained
			}
			before := copyDraftReview(a.draftReviews[r.ID])
			_, err = approveDraftAs(f, reviewer, prepared.Review)
			documentFailure(t, err, commands.EvidenceStale)
			if !reflect.DeepEqual(before, a.draftReviews[r.ID]) {
				t.Fatal("refusal adopted inconsistent approval")
			}
		})
	}
}

func TestFinancialDraftApprovalReferenceDriftRequiresNewReview(t *testing.T) {
	f, _, reviewer, draft, binding := draftApprovalWorld(t)
	a := f.s.financialIssuance
	prepared, err := prepareDraftReviewAs(f, reviewer, "before-drift", draft, binding)
	f.must(err)
	spec := a.changes["consume-kind"].Spec
	spec.ID, spec.After.ID = "review-future-template", "review-next-template"
	spec.Effective = f.s.clock().Add(time.Minute)
	change, err := requestChangeAs(f, f.as(Admin2Login), spec)
	f.must(err)
	change, err = verifyChangeAs(f, f.as(AdminLogin), change)
	f.must(err)
	_, err = decideChangeAs(f, f.as("protected-fifth@example.invalid"), change)
	f.must(err)
	f.advance(time.Minute)
	refused, err := approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	if refused.Review.State != "refused" || refused.Review.Refusal != issuance.ReasonReferenceChanged {
		t.Fatal("changed reference did not refuse reviewed facts")
	}
	actual, err := a.documents.DraftOf(SampleWorkspace, draft.Spec.ID)
	f.must(err)
	if !reflect.DeepEqual(actual, draft.Module) {
		t.Fatal("known refusal wrote module approval")
	}
	same, err := prepareDraftReviewAs(f, reviewer, "before-drift", draft, binding)
	f.must(err)
	if !reflect.DeepEqual(same, refused) {
		t.Fatal("same review refreshed reviewed facts")
	}
	fresh, err := prepareDraftReviewAs(f, reviewer, "after-drift", draft, binding)
	f.must(err)
	if fresh.Review.References[2].Change != spec.ID || fresh.Review.Digest == prepared.Review.Digest {
		t.Fatal("explicit new review lacks current reference")
	}
	approved, err := approveDraftAs(f, reviewer, fresh.Review)
	f.must(err)
	if approved.Review.State != "approved" || len(a.draftReviewOrders[draft.Spec.ID]) != 2 {
		t.Fatal("new explicit review did not approve")
	}
}

func grantDraftReviewer(f world, prefix, recipient string) {
	firstLogin, secondLogin := AdminLogin, Admin2Login
	if recipient == SampleAdmin {
		firstLogin = Admin3Login
	}
	if recipient == SampleAdmin2 {
		secondLogin = Admin3Login
	}
	root, second := f.as(firstLogin), f.as(secondLogin)
	for _, capability := range []string{draftauthority.Approve, draftauthority.Inspect} {
		spec := draftPermissionSpec(f, capability)
		p, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: prefix + capability + "-ceiling", Spec: spec, Rationale: "Independent draft review"})
		f.must(err)
		ceiling, err := draftPermissionDecision(f, second, p)
		f.must(err)
		spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, recipient, ceiling.Decision.GrantID
		p, err = draftPermissionProposal(f, root, inputauthority.Proposal{ID: prefix + capability + "-human", Spec: spec, Rationale: "Independent draft reviewer"})
		f.must(err)
		_, err = draftPermissionDecision(f, second, p)
		f.must(err)
	}
	f.s.endProbe(root.verifier(f))
	f.s.endProbe(second.verifier(f))
}
