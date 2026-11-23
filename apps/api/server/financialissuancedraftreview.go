package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"reflect"
	"slices"
	"time"
)

type financialDraftReview struct {
	Workspace, ID, DraftID, DraftDigest, Actor, BindingID, BindingDigest string
	Digest, State, Refusal                                               string
	ReviewedAt                                                           time.Time
	References                                                           []issuance.Reference
	Coverage                                                             receivables.Coverage
	Module                                                               issuance.DraftView
}
type financialDraftReviewResult struct {
	Review               financialDraftReview
	Receipt, Eligibility string
}

func copyDraftReview(r financialDraftReview) financialDraftReview {
	r.References = slices.Clone(r.References)
	r.Module = copyModuleDraftView(r.Module)
	return r
}
func draftReviewDigest(r financialDraftReview) (string, error) {
	r.Digest, r.State, r.Refusal = "", "", ""
	r.Module = issuance.DraftView{}
	return financialEvidenceDigest("issuance_draft_review", r)
}
func draftReviewResult(r financialDraftReview) financialDraftReviewResult {
	receipt := "review_attempt"
	if r.State == "approved" {
		receipt = "historical_approval"
	}
	return financialDraftReviewResult{Review: copyDraftReview(r), Receipt: receipt, Eligibility: "unchecked"}
}
func approvedDraftMatches(p financialDraftRecord, r financialDraftReview, v issuance.DraftView) bool {
	return reflect.DeepEqual(v.Draft, p.Module.Draft) && v.Revision == p.Module.Revision && v.Digest == p.Module.Digest && v.Submitter == p.Submitter && v.Approver == r.Actor && reflect.DeepEqual(v.References, r.References) && v.Coverage != nil && *v.Coverage == r.Coverage && v.Number == "" && !v.At.IsZero() && !v.At.Before(p.Module.At) && !v.At.Before(r.ReviewedAt)
}
func (a *financialIssuance) ownedDraft(workspace, id, digest string) (financialDraftRecord, error) {
	p, ok := a.drafts[id]
	expected, err := draftIntentDigest(p)
	if !ok || p.Workspace != workspace || p.Spec.ID != id || p.Digest != digest || err != nil || expected != digest || p.State != "submitted" || !unapprovedDraftMatches(p, p.Module) || !validRequestID(p.Submitter) || !validRequestID(p.FinalRequester) {
		return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if err := a.draftScopeIntegrity(p, true); err != nil {
		return financialDraftRecord{}, err
	}
	return copyFinancialDraft(p), nil
}
func (a *financialIssuance) draftReviewChain(p financialDraftRecord) (financialDraftReview, bool, error) {
	fail := func() (financialDraftReview, bool, error) {
		return financialDraftReview{}, false, commands.Fail(commands.EvidenceStale)
	}
	ids := a.draftReviewOrders[p.Spec.ID]
	if len(ids) > financialIssuanceLimit || len(a.draftReviews) > financialIssuanceLimit {
		return fail()
	}
	seen := map[string]bool{}
	var last financialDraftReview
	for i, id := range ids {
		r, ok := a.draftReviews[id]
		digest, err := draftReviewDigest(r)
		if !ok || seen[id] || r.ID != id || r.Workspace != p.Workspace || r.DraftID != p.Spec.ID || r.DraftDigest != p.Digest || err != nil || digest != r.Digest || !validRequestID(r.Actor) || r.ReviewedAt.IsZero() {
			return fail()
		}
		if r.State != "prepared" && r.State != "approval_unknown" && r.State != "approved" && r.State != "refused" {
			return fail()
		}
		if r.State == "refused" && r.Refusal != issuance.ReasonReferenceChanged && r.Refusal != issuance.ReasonCoverageMismatch || r.State != "refused" && r.Refusal != "" {
			return fail()
		}
		if i < len(ids)-1 && r.State != "refused" {
			return fail()
		}
		if r.Module.Digest != "" && (r.State != "approved" && r.State != "approval_unknown" || !approvedDraftMatches(p, r, r.Module)) {
			return fail()
		}
		if r.State == "approved" && r.Module.Digest == "" {
			return fail()
		}
		seen[id] = true
		last = r
	}
	for id, r := range a.draftReviews {
		if r.DraftID == p.Spec.ID && !seen[id] {
			return fail()
		}
	}
	return copyDraftReview(last), len(ids) > 0, nil
}
func (a *financialIssuance) draftPhaseReadback(p financialDraftRecord) (issuance.DraftView, error) {
	last, has, err := a.draftReviewChain(p)
	if err != nil {
		return issuance.DraftView{}, err
	}
	v, found, err := a.draftModule(p)
	if err != nil {
		return issuance.DraftView{}, err
	}
	if !found {
		return issuance.DraftView{}, commands.Fail(commands.EvidenceStale)
	}
	expected := p.Module
	if has && last.Module.Digest != "" {
		expected = last.Module
	}
	if !reflect.DeepEqual(v, expected) {
		return issuance.DraftView{}, commands.Fail(commands.EvidenceStale)
	}
	return v, nil
}
func (a *financialIssuance) liveDraftApproval(p financialDraftRecord, actor string) (financialDraftRecord, error) {
	source, err := draftOriginSource(p)
	if err != nil {
		return financialDraftRecord{}, err
	}
	current, err := a.currentDraftOrigin(p.Workspace, actor, draftauthority.Approve, p.Spec)
	if err != nil {
		return financialDraftRecord{}, err
	}
	if !reflect.DeepEqual(current.Profile, p.Profile) || !reflect.DeepEqual(current.Application, p.Application) {
		return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if err := a.draftActor(p.Workspace, p.Submitter, draftauthority.Submit); err != nil {
		return financialDraftRecord{}, err
	}
	identities := append([]string{p.Submitter, p.FinalRequester, source.Basis.Subject.Author}, source.Basis.Contributors...)
	for _, id := range identities {
		if !validRequestID(id) {
			return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
		}
		if id == actor {
			return financialDraftRecord{}, commands.Fail(commands.SelfApprovalDenied)
		}
	}
	return copyFinancialDraft(p), nil
}
