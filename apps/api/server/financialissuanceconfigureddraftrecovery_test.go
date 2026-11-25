package server

import (
	"errors"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func grantConfiguredDraftCapability(f world, prefix, capability, recipient string) {
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	defer f.s.endProbe(root.verifier(f))
	defer f.s.endProbe(second.verifier(f))
	spec := draftPermissionSpec(f, capability)
	p, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: prefix + capability + "-ceiling", Spec: spec, Rationale: "Explicit configured draft ceiling"})
	f.must(err)
	c, err := draftPermissionDecision(f, second, p)
	f.must(err)
	spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, recipient, c.Decision.GrantID
	p, err = draftPermissionProposal(f, root, inputauthority.Proposal{ID: prefix + capability + "-human", Spec: spec, Rationale: "Explicit configured draft operator"})
	f.must(err)
	_, err = draftPermissionDecision(f, second, p)
	f.must(err)
}

func TestFinancialConfiguredDraftSubmissionRecovery(t *testing.T) {
	for _, point := range []string{"before-submit", "after-intent", "after-submit", "after-submission-publication"} {
		t.Run(point, func(t *testing.T) {
			f, _, c, digest := applicationWorld(t)
			a := f.s.financialIssuance
			login := activateProtectedMember(f, "configured-other-submitter")
			for _, capability := range []string{draftauthority.Submit, draftauthority.Inspect} {
				grantConfiguredDraftCapability(f, "other-submit-", capability, "configured-other-submitter")
			}
			member, other := f.as(MemberLogin), f.as(login)
			f.must(inputAuthorityAs(f, other, func() error {
				return a.draftOperation(SampleWorkspace, "configured-other-submitter", draftauthority.Submit)
			}))
			spec := financialDraftSpec{ID: "configured-submit", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(c), FinalRequest: "issuance-final", FinalDigest: digest}}
			f.s.fault = func(at string) bool { return at == "issuance-draft-"+point }
			_, err := submitDraftAs(f, member, spec)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			retained, exists := a.drafts[spec.ID]
			actual, moduleErr := a.documents.DraftOf(SampleWorkspace, spec.ID)
			if point == "before-submit" {
				if exists || len(a.draftScopes) != 0 || !errors.Is(moduleErr, issuance.ErrNotFound) {
					t.Fatal("prewrite submission reserved state")
				}
			} else {
				_, err = submitDraftAs(f, other, spec)
				documentFailure(t, err, commands.EvidenceStale)
				if !reflect.DeepEqual(a.drafts[spec.ID], retained) {
					t.Fatal("authorized other actor stole uncertainty")
				}
				if point == "after-intent" && !errors.Is(moduleErr, issuance.ErrNotFound) {
					t.Fatal("intent wrote module")
				}
				if point != "after-intent" && (moduleErr != nil || !reflect.DeepEqual(actual, retained.Module)) {
					t.Fatal("module snapshot not retained")
				}
			}
			equal := spec
			r := *spec.Application
			equal.Application = &r
			recovered, err := submitDraftAs(f, member, equal)
			f.must(err)
			actual, err = a.documents.DraftOf(SampleWorkspace, spec.ID)
			f.must(err)
			if recovered.State != "submitted" || !reflect.DeepEqual(recovered.Module, actual) || recovered.Submitter != SampleMember || len(a.drafts) != 1 || len(a.draftScopes) != 1 {
				t.Fatal("configured submission failed exact recovery")
			}
			original := copyFinancialDraft(recovered)
			recovered.Spec.Application.Configuration.ID = "changed"
			recovered.Application.Configuration.Kind.Languages[0] = "changed"
			recovered.Application.Source.Basis.Rows[0].Net = "999"
			recovered.Application.Source.Basis.Subject.Digests["result"] = "changed"
			spec.Application.FinalRequest = "changed"
			again, err := submitDraftAs(f, member, equal)
			f.must(err)
			if !reflect.DeepEqual(again, original) {
				t.Fatal("configured draft aliases caller evidence")
			}
		})
	}
}

func TestFinancialConfiguredDraftApprovalRecovery(t *testing.T) {
	for _, point := range []string{"before-intent", "after-intent", "after-module", "after-publication"} {
		t.Run(point, func(t *testing.T) {
			f, member, reviewer, spec, binding := configuredDraftWorld(t)
			a := f.s.financialIssuance
			login := activateProtectedMember(f, "configured-other-reviewer")
			grantDraftReviewer(f, "other-configured-review-", "configured-other-reviewer")
			other := f.as(login)
			f.must(inputAuthorityAs(f, other, func() error {
				return a.draftOperation(SampleWorkspace, "configured-other-reviewer", draftauthority.Approve)
			}))
			draft, err := submitDraftAs(f, member, spec)
			f.must(err)
			prepared, err := prepareDraftReviewAs(f, reviewer, "configured-recovery", draft, binding)
			f.must(err)
			f.s.fault = func(at string) bool { return at == "issuance-draft-review-"+point }
			_, err = approveDraftAs(f, reviewer, prepared.Review)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			retained := copyDraftReview(a.draftReviews[prepared.Review.ID])
			module, err := a.documents.DraftOf(SampleWorkspace, spec.ID)
			f.must(err)
			_, err = approveDraftAs(f, other, prepared.Review)
			documentFailure(t, err, commands.PermissionDenied)
			if !reflect.DeepEqual(a.draftReviews[prepared.Review.ID], retained) {
				t.Fatal("authorized other reviewer stole uncertainty")
			}
			replay, err := submitDraftAs(f, member, spec)
			f.must(err)
			if !reflect.DeepEqual(replay, draft) {
				t.Fatal("submission replay changed later review phase")
			}
			actual, err := a.documents.DraftOf(SampleWorkspace, spec.ID)
			f.must(err)
			if !reflect.DeepEqual(actual, module) {
				t.Fatal("submission replay replaced approval")
			}
			approved, err := approveDraftAs(f, reviewer, prepared.Review)
			f.must(err)
			actual, err = a.documents.DraftOf(SampleWorkspace, spec.ID)
			f.must(err)
			if approved.Review.State != "approved" || !reflect.DeepEqual(actual, approved.Review.Module) || approved.Eligibility != "unchecked" {
				t.Fatal("configured approval failed recovery")
			}
			replay, err = submitDraftAs(f, member, spec)
			f.must(err)
			again, err := approveDraftAs(f, reviewer, prepared.Review)
			f.must(err)
			if !reflect.DeepEqual(replay, draft) || !reflect.DeepEqual(again, approved) {
				t.Fatal("configured historical replay changed evidence")
			}
		})
	}
}
