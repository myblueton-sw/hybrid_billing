package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"reflect"
	"testing"
	"time"
)

func TestFinancialDraftApprovalWrongAuthorizedRecoveryActor(t *testing.T) {
	f, _, reviewer, draft, binding := draftApprovalWorld(t)
	login := activateProtectedMember(f, "other-draft-reviewer")
	grantDraftReviewer(f, "other-review-", "other-draft-reviewer")
	other := f.as(login)
	f.must(inputAuthorityAs(f, other, func() error {
		return f.s.financialIssuance.draftOperation(SampleWorkspace, "other-draft-reviewer", draftauthority.Approve)
	}))
	prepared, err := prepareDraftReviewAs(f, reviewer, "pinned-review", draft, binding)
	f.must(err)
	f.s.fault = func(at string) bool { return at == "issuance-draft-review-after-module" }
	_, err = approveDraftAs(f, reviewer, prepared.Review)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	a := f.s.financialIssuance
	retained := copyDraftReview(a.draftReviews[prepared.Review.ID])
	_, err = approveDraftAs(f, other, prepared.Review)
	documentFailure(t, err, commands.PermissionDenied)
	if !reflect.DeepEqual(retained, a.draftReviews[prepared.Review.ID]) {
		t.Fatal("other authorized actor recovered pinned attempt")
	}
	recovered, err := approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	if recovered.Review.State != "approved" || !reflect.DeepEqual(retained.Module, recovered.Review.Module) {
		t.Fatal("original actor failed exact recovery")
	}
}

func TestFinancialDraftApprovalMidnightRequiresNewReview(t *testing.T) {
	f, _, reviewer, draft, binding := draftApprovalWorld(t, 38*time.Hour+58*time.Minute)
	a := f.s.financialIssuance
	if f.s.clock().Hour() != 23 || f.s.clock().Minute() != 59 || draft.Profile.Spec.Zone != "UTC" {
		t.Fatal("fixture not at actual billing midnight boundary")
	}
	prepared, err := prepareDraftReviewAs(f, reviewer, "before-midnight", draft, binding)
	f.must(err)
	f.advance(time.Minute)
	refused, err := approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	if refused.Review.State != "refused" || refused.Review.Refusal != issuance.ReasonCoverageMismatch {
		t.Fatal("midnight did not refuse pinned calendar coverage")
	}
	actual, err := a.documents.DraftOf(SampleWorkspace, draft.Spec.ID)
	f.must(err)
	if !reflect.DeepEqual(actual, draft.Module) {
		t.Fatal("midnight refusal wrote approval")
	}
	fresh, err := prepareDraftReviewAs(f, reviewer, "after-midnight", draft, binding)
	f.must(err)
	if fresh.Review.Coverage == prepared.Review.Coverage {
		t.Fatal("new review retained yesterday coverage")
	}
	approved, err := approveDraftAs(f, reviewer, fresh.Review)
	f.must(err)
	if approved.Review.State != "approved" || *approved.Review.Module.Coverage != fresh.Review.Coverage {
		t.Fatal("fresh midnight coverage not approved")
	}
}

func TestFinancialDraftApprovalExcludesActualContributors(t *testing.T) {
	f, _, _, draft, binding := draftApprovalWorld(t)
	identities := append([]string{draft.Submitter, draft.FinalRequester, draft.Profile.Source.Basis.Subject.Author}, draft.Profile.Source.Basis.Contributors...)
	logins := map[string]string{SampleMember: MemberLogin, SampleAdmin: AdminLogin, SampleAdmin2: Admin2Login, SampleAdmin3: Admin3Login}
	seen := map[string]bool{}
	if len(draft.Profile.Source.Basis.Contributors) == 0 {
		t.Fatal("fixture lacks actual contributors")
	}
	for _, id := range identities {
		if seen[id] {
			continue
		}
		seen[id] = true
		login, ok := logins[id]
		if !ok {
			t.Fatal("actual contributor has no fixture login", id)
		}
		grantDraftReviewer(f, "excluded-"+id+"-", id)
		who := f.as(login)
		f.must(inputAuthorityAs(f, who, func() error {
			return f.s.financialIssuance.draftOperation(SampleWorkspace, id, draftauthority.Approve)
		}))
		_, err := prepareDraftReviewAs(f, who, "excluded-review-"+id, draft, binding)
		documentFailure(t, err, commands.SelfApprovalDenied)
		if len(f.s.financialIssuance.draftReviews) != 0 {
			t.Fatal("excluded identity retained review")
		}
	}
	actual, err := f.s.financialIssuance.documents.DraftOf(SampleWorkspace, draft.Spec.ID)
	f.must(err)
	if !reflect.DeepEqual(actual, draft.Module) {
		t.Fatal("excluded contributor changed module")
	}
}

func TestFinancialDraftApprovalRechecksLiveAdmission(t *testing.T) {
	for _, mode := range []string{"submitter", "source"} {
		t.Run(mode, func(t *testing.T) {
			f, _, reviewer, draft, binding := draftApprovalWorld(t)
			prepared, err := prepareDraftReviewAs(f, reviewer, "live-admission", draft, binding)
			f.must(err)
			expected := commands.EvidenceStale
			if mode == "source" {
				key := [2]string{SampleWorkspace, "public-result"}
				application := f.s.documentERP.resultApplications[key]
				application.receipt = "changed-source"
				f.s.documentERP.resultApplications[key] = application
			} else {
				expected = commands.PermissionDenied
				revoked := 0
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == draft.Submitter && g.Terms.Kind == draftauthority.ResourceKind && g.Terms.Action == gate.SubmitIssuanceDraft {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						revoked++
					}
				}
				if revoked == 0 {
					t.Fatal("no actual submitter grant revoked")
				}
			}
			_, err = approveDraftAs(f, reviewer, prepared.Review)
			documentFailure(t, err, expected)
			a := f.s.financialIssuance
			if !reflect.DeepEqual(prepared.Review, a.draftReviews[prepared.Review.ID]) {
				t.Fatal("failed admission changed reviewed intent")
			}
			actual, err := a.documents.DraftOf(SampleWorkspace, draft.Spec.ID)
			f.must(err)
			if !reflect.DeepEqual(actual, draft.Module) {
				t.Fatal("failed admission approved module")
			}
		})
	}
}
