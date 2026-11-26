package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"reflect"
	"testing"
)

func TestFinancialConfiguredDraftApprovalExcludesActualContributors(t *testing.T) {
	f, member, _, spec, binding := configuredDraftWorld(t)
	draft, err := submitDraftAs(f, member, spec)
	f.must(err)
	identities := append([]string{draft.Submitter, draft.FinalRequester, draft.Application.Source.Basis.Subject.Author}, draft.Application.Source.Basis.Contributors...)
	logins := map[string]string{SampleMember: MemberLogin, SampleAdmin: AdminLogin, SampleAdmin2: Admin2Login, SampleAdmin3: Admin3Login}
	seen := map[string]bool{}
	if len(draft.Application.Source.Basis.Contributors) == 0 {
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

func TestFinancialConfiguredDraftApprovalCurrentAndHistoricalGovernor(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	a := f.s.financialIssuance
	draft, err := submitDraftAs(f, member, spec)
	f.must(err)
	prepared, err := prepareDraftReviewAs(f, reviewer, "governor-check", draft, binding)
	f.must(err)
	approved, err := approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	removed := 0
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin3 && g.Terms.Action == gate.AdministerInstallation {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
			removed++
		}
	}
	if removed == 0 {
		t.Fatal("configuration governor was not revoked")
	}
	f.must(inputAuthorityAs(f, reviewer, func() error {
		if err := a.draftOperation(SampleWorkspace, "configured-draft-reviewer", draftauthority.Approve); err != nil {
			return err
		}
		_, err := a.liveDraftApproval(draft, "configured-draft-reviewer")
		documentFailure(t, err, commands.PermissionDenied)
		return nil
	}))
	replay, err := approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	if !reflect.DeepEqual(replay, approved) || replay.Eligibility != "unchecked" {
		t.Fatal("historical receipt reopened current configuration")
	}
}
