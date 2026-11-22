package server

import (
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

type financialDraftObservation struct {
	Draft     financialDraftRecord
	Review    financialDraftReview
	HasReview bool
	Module    string
}

func (a *financialIssuance) retainedDraft(workspace, id string) (financialDraftObservation, error) {
	bad := func() (financialDraftObservation, error) {
		return financialDraftObservation{}, commands.Fail(commands.EvidenceStale)
	}
	p, ok := a.drafts[id]
	digest, err := draftIntentDigest(p)
	if !ok || err != nil || digest != p.Digest || p.Workspace != workspace || p.Spec.ID != id || !validRequestID(p.Submitter) || !validRequestID(p.FinalRequester) {
		return bad()
	}
	if p.State != "submitted" && p.State != "submission_unknown" && p.State != "submission_refused" {
		return bad()
	}
	var derived issuance.Draft
	if p.Application != nil {
		derived, err = issuanceDraftFromApplication(*p.Application, p.Spec.ID, p.Spec.Language)
	} else {
		derived, err = issuanceDraftFromSource(p.Profile, p.Spec.ID, p.Spec.Language)
	}
	if err != nil || !reflect.DeepEqual(derived, p.Draft) {
		return bad()
	}
	if err := a.draftScopeIntegrity(p, true); err != nil {
		return financialDraftObservation{}, err
	}
	observed := p.Module.Digest != ""
	if p.State == "submission_refused" && !reflect.DeepEqual(p.Module, issuance.DraftView{}) {
		return bad()
	}
	if observed && !unapprovedDraftMatches(p, p.Module) || !observed && !reflect.DeepEqual(p.Module, issuance.DraftView{}) || p.State == "submitted" && !observed {
		return bad()
	}
	r, has, err := a.draftReviewChain(p)
	if err != nil {
		return financialDraftObservation{}, err
	}
	if has && p.State != "submitted" {
		return bad()
	}
	for _, reviewID := range a.draftReviewOrders[p.Spec.ID] {
		v := a.draftReviews[reviewID]
		if !validRequestID(v.BindingID) || !validPlanHash(v.BindingDigest) {
			return bad()
		}
		if v.Module.Digest == "" && !reflect.DeepEqual(v.Module, issuance.DraftView{}) {
			return bad()
		}
	}
	return financialDraftObservation{Draft: copyFinancialDraft(p), Review: copyDraftReview(r), HasReview: has}, nil
}

func (a *financialIssuance) observeDraft(p financialDraftObservation) financialDraftObservation {
	if a.documents == nil {
		p.Module = "unavailable"
		return p
	}
	actual, found, err := a.draftModule(p.Draft)
	if err != nil {
		p.Module = "unavailable"
		return p
	}
	expected := p.Draft.Module
	if p.HasReview && p.Review.Module.Digest != "" {
		expected = p.Review.Module
	}
	p.Module = "conflict"
	if !found {
		if expected.Digest == "" {
			p.Module = "missing"
		}
		return p
	}
	if expected.Digest == "" || !reflect.DeepEqual(expected, actual) {
		return p
	}
	p.Module = "matching_unapproved"
	if expected.Approver != "" {
		p.Module = "matching_approved"
	}
	return p
}

func (a *financialIssuance) inspectDraftRecord(workspace, actor, id string) (financialDraftObservation, error) {
	if err := a.draftReadCaller(workspace, actor); err != nil {
		return financialDraftObservation{}, err
	}
	p, err := a.retainedDraft(workspace, id)
	if err != nil {
		return financialDraftObservation{}, err
	}
	return a.observeDraft(p), nil
}
