package server

import (
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
)

func draftCommandCapability(kind string) string {
	switch kind {
	case "submit":
		return draftauthority.Submit
	case "prepare_approval", "approve":
		return draftauthority.Approve
	}
	return ""
}

func publicDraftSpec(id string, fields map[string]string) financialDraftSpec {
	return financialDraftSpec{ID: id, Language: fields["language"], Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: financialConfigurationReference{Schema: "hb_issuance_configuration_reference_v1", ID: fields["configuration"], Digest: fields["configuration_digest"]}, FinalRequest: fields["final_request"], FinalDigest: fields["final_digest"]}}
}

func (a *financialIssuance) draftMutationIdentity(workspace, actor, kind, target string, fields map[string]string) error {
	if err := a.draftOperation(workspace, actor, draftCommandCapability(kind)); err != nil {
		return err
	}
	if kind == "submit" {
		spec := publicDraftSpec(target, fields)
		if !validRequestID(target) || !validDraftOriginRequest(spec) {
			return commands.Fail(commands.InvalidRequest)
		}
		if _, exists := a.drafts[target]; !exists {
			return nil
		}
		p, err := a.retainedDraft(workspace, target)
		if err != nil {
			return err
		}
		if p.Draft.Submitter != actor {
			return commands.Fail(commands.PermissionDenied)
		}
		if !reflect.DeepEqual(p.Draft.Spec, spec) {
			return commands.Fail(commands.EvidenceStale)
		}
		return nil
	}
	var r financialDraftReview
	draftID, draftDigest := fields["draft"], fields["draft_digest"]
	if kind == "approve" {
		var exists bool
		r, exists = a.draftReviews[target]
		if !exists || r.Workspace != workspace || r.ID != target || r.Digest != fields["digest"] {
			return commands.Fail(commands.EvidenceStale)
		}
		if r.Actor != actor {
			return commands.Fail(commands.PermissionDenied)
		}
		draftID, draftDigest = r.DraftID, r.DraftDigest
	}
	p, err := a.retainedDraft(workspace, draftID)
	if err != nil {
		return err
	}
	if p.Draft.Digest != draftDigest {
		return commands.Fail(commands.EvidenceStale)
	}
	if p.Draft.Application == nil {
		return commands.Fail(commands.UnsupportedCapability)
	}
	if kind == "prepare_approval" {
		if old, exists := a.draftReviews[target]; exists {
			if old.Actor != actor {
				return commands.Fail(commands.PermissionDenied)
			}
			if old.Workspace != workspace || old.ID != target || old.DraftID != draftID || old.DraftDigest != draftDigest || old.BindingID != fields["binding"] || old.BindingDigest != fields["binding_digest"] {
				return commands.Fail(commands.EvidenceStale)
			}
		}
	}
	return nil
}

func (a *financialIssuance) submitPublicDraft(ctx commands.Context, effect *financialDraftEffect) (financialDraftRecord, error) {
	if p, exists := a.drafts[ctx.Target]; exists && p.State == "submitted" {
		owned, err := a.retainedDraft(ctx.Workspace, ctx.Target)
		if err != nil {
			return financialDraftRecord{}, err
		}
		if _, err := a.draftPhaseReadback(owned.Draft); err != nil {
			return financialDraftRecord{}, err
		}
		var review *financialDraftReview
		if owned.HasReview {
			review = &owned.Review
		}
		if err := a.admitDraftProjection(owned.Draft, review); err != nil {
			return financialDraftRecord{}, err
		}
		return owned.Draft, nil
	}
	return a.submitDraftTracked(ctx.Workspace, ctx.User, publicDraftSpec(ctx.Target, ctx.Fields), effect, func(p financialDraftRecord) error { return a.admitDraftProjection(p, nil) })
}

func (a *financialIssuance) applyDraftMutation(kind string, ctx commands.Context) (commands.Outcome, error) {
	if err := a.draftMutationIdentity(ctx.Workspace, ctx.User, kind, ctx.Target, ctx.Fields); err != nil {
		return commands.Outcome{}, err
	}
	if a.documents == nil {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	effect := &financialDraftEffect{}
	var p financialDraftRecord
	var review *financialDraftReview
	var err error
	admit := func(p financialDraftRecord, r financialDraftReview) error { return a.admitDraftProjection(p, &r) }
	if kind == "submit" {
		p, err = a.submitPublicDraft(ctx, effect)
	} else {
		var result financialDraftReviewResult
		if kind == "prepare_approval" {
			result, err = a.prepareDraftApprovalTracked(ctx.Workspace, ctx.User, ctx.Target, ctx.Fields["draft"], ctx.Fields["draft_digest"], ctx.Fields["binding"], ctx.Fields["binding_digest"], effect, admit)
		} else {
			result, err = a.approveDraftTracked(ctx.Workspace, ctx.User, ctx.Target, ctx.Fields["digest"], effect, admit)
		}
		if err == nil {
			review = &result.Review
			p = copyFinancialDraft(a.drafts[result.Review.DraftID])
		}
	}
	if err != nil {
		return commands.Outcome{}, draftPublicError(err, effect)
	}
	fields := draftMutationReceipt(p, review)
	if err := a.boundedDraftFields(fields); err != nil {
		return commands.Outcome{}, draftPublicError(err, effect)
	}
	state := p.State
	if review != nil {
		state = review.State
	}
	return commands.Outcome{Fields: fields, BusinessState: state, ResultRevision: a.server.Next()}, nil
}
