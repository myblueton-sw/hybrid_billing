package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"reflect"
)

func (a *financialIssuance) prepareDraftApproval(workspace, actor, reviewID, draftID, draftDigest, bindingID, bindingDigest string) (financialDraftReviewResult, error) {
	if err := a.draftOperation(workspace, actor, draftauthority.Approve); err != nil {
		return financialDraftReviewResult{}, err
	}
	if !validRequestID(reviewID) || !validRequestID(bindingID) || !validPlanHash(bindingDigest) {
		return financialDraftReviewResult{}, commands.Fail(commands.InvalidRequest)
	}
	p, err := a.ownedDraft(workspace, draftID, draftDigest)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	last, has, err := a.draftReviewChain(p)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	if _, err := a.draftPhaseReadback(p); err != nil {
		return financialDraftReviewResult{}, err
	}
	if old, ok := a.draftReviews[reviewID]; ok {
		if old.Workspace != workspace || old.Actor != actor || old.DraftID != draftID || old.DraftDigest != draftDigest || old.BindingID != bindingID || old.BindingDigest != bindingDigest {
			return financialDraftReviewResult{}, commands.Fail(commands.EvidenceStale)
		}
		return draftReviewResult(old), nil
	}
	if has && last.State != "refused" || len(a.draftReviews) >= financialIssuanceLimit || len(a.draftReviewOrders[draftID]) >= financialIssuanceLimit {
		return financialDraftReviewResult{}, commands.Fail(commands.PrerequisiteFailed)
	}
	profile, err := a.liveDraftApproval(p, actor)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	at := a.server.clock().UTC().Round(0)
	refs, err := a.draftReferences(profile, at)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	coverage, err := a.draftCoverage(profile, bindingID, bindingDigest, at)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	r := financialDraftReview{Workspace: workspace, ID: reviewID, DraftID: draftID, DraftDigest: draftDigest, Actor: actor, BindingID: bindingID, BindingDigest: bindingDigest, ReviewedAt: at, References: refs, Coverage: coverage, State: "prepared"}
	r.Digest, err = draftReviewDigest(r)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	a.draftReviews[reviewID] = copyDraftReview(r)
	a.draftReviewOrders[draftID] = append(a.draftReviewOrders[draftID], reviewID)
	return draftReviewResult(r), nil
}

func (a *financialIssuance) refuseDraftReview(p financialDraftRecord, r financialDraftReview, code string) (financialDraftReviewResult, error) {
	actual, found, err := a.draftModule(p)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	if !found || !reflect.DeepEqual(actual, p.Module) {
		return financialDraftReviewResult{}, commands.Fail(commands.EvidenceStale)
	}
	r.State, r.Refusal = "refused", code
	a.draftReviews[r.ID] = copyDraftReview(r)
	return draftReviewResult(r), nil
}

func (a *financialIssuance) approveDraft(workspace, actor, reviewID, reviewDigest string) (financialDraftReviewResult, error) {
	if err := a.draftOperation(workspace, actor, draftauthority.Approve); err != nil {
		return financialDraftReviewResult{}, err
	}
	r, ok := a.draftReviews[reviewID]
	if !ok || r.Workspace != workspace || r.ID != reviewID || r.Digest != reviewDigest {
		return financialDraftReviewResult{}, commands.Fail(commands.EvidenceStale)
	}
	if r.Actor != actor {
		return financialDraftReviewResult{}, commands.Fail(commands.PermissionDenied)
	}
	p, err := a.ownedDraft(workspace, r.DraftID, r.DraftDigest)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	if _, _, err := a.draftReviewChain(p); err != nil {
		return financialDraftReviewResult{}, err
	}
	actual, err := a.draftPhaseReadback(p)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	if r.State == "approved" || r.State == "refused" {
		return draftReviewResult(r), nil
	}
	if r.State == "approval_unknown" && r.Module.Digest != "" {
		if !approvedDraftMatches(p, r, actual) || !reflect.DeepEqual(actual, r.Module) {
			return financialDraftReviewResult{}, commands.Fail(commands.EvidenceStale)
		}
		r.State = "approved"
		a.draftReviews[r.ID] = copyDraftReview(r)
		return draftReviewResult(r), nil
	}
	profile, err := a.liveDraftApproval(p, actor)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	now := a.server.clock().UTC().Round(0)
	refs, err := a.draftReferences(profile, now)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	coverage, err := a.draftCoverage(profile, r.BindingID, r.BindingDigest, now)
	if err != nil {
		return financialDraftReviewResult{}, err
	}
	if !reflect.DeepEqual(refs, r.References) {
		return a.refuseDraftReview(p, r, issuance.ReasonReferenceChanged)
	}
	if coverage != r.Coverage {
		return a.refuseDraftReview(p, r, issuance.ReasonCoverageMismatch)
	}
	if a.server.faulty("issuance-draft-review-before-intent") {
		return financialDraftReviewResult{}, commands.Fail(commands.DependencyUnavailable)
	}
	r.State = "approval_unknown"
	a.draftReviews[r.ID] = copyDraftReview(r)
	if a.server.faulty("issuance-draft-review-after-intent") {
		return financialDraftReviewResult{}, commands.Fail(commands.DependencyUnavailable)
	}
	actual, err = a.documents.ApproveReviewed(workspace, p.Spec.ID, p.Module.Digest, r.References, &r.Coverage, issuance.Actor{ID: actor, Kind: issuance.KindHuman})
	if err != nil {
		var refusal issuance.Refusal
		if errors.As(err, &refusal) && (refusal.Reason == issuance.ReasonReferenceChanged || refusal.Reason == issuance.ReasonCoverageMismatch) {
			return a.refuseDraftReview(p, r, refusal.Reason)
		}
		return financialDraftReviewResult{}, err
	}
	r.Module = copyModuleDraftView(actual)
	a.draftReviews[r.ID] = copyDraftReview(r)
	if !approvedDraftMatches(p, r, actual) {
		return financialDraftReviewResult{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-draft-review-after-module") {
		return financialDraftReviewResult{}, commands.Fail(commands.DependencyUnavailable)
	}
	r.State = "approved"
	a.draftReviews[r.ID] = copyDraftReview(r)
	if a.server.faulty("issuance-draft-review-after-publication") {
		return financialDraftReviewResult{}, commands.Fail(commands.DependencyUnavailable)
	}
	return draftReviewResult(r), nil
}
