package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
)

func (a *financialIssuance) boundedDraftFields(fields map[string]string) error {
	limits := a.server.cfg.Commands
	if limits.Fields < 64 || limits.Value < 4096 {
		return commands.Fail(commands.UnsupportedCapability)
	}
	if len(fields) > limits.Fields {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	for _, value := range fields {
		if len(value) > min(limits.Value, 4096) {
			return commands.Fail(commands.PrerequisiteFailed)
		}
	}
	raw, err := json.Marshal(fields)
	if err != nil || len(raw) > 65536 {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return nil
}

func draftJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", commands.Fail(commands.PrerequisiteFailed)
	}
	return string(raw), nil
}

func draftSummary(p financialDraftObservation) (map[string]string, error) {
	v := p.Draft
	source, err := draftOriginSource(v)
	if err != nil {
		return nil, err
	}
	fields := map[string]string{"draft": v.Spec.ID, "digest": v.Digest, "customer_id": source.Basis.Customer, "period": source.Basis.Period, "currency": v.Draft.Currency, "scale": strconv.Itoa(v.Draft.Scale), "total": v.Draft.Total, "submitter": v.Submitter, "submission_state": v.State, "review_state": "none", "module_observation": p.Module}
	if !v.Module.At.IsZero() {
		fields["submitted_at"] = v.Module.At.Format(time.RFC3339Nano)
	}
	if p.HasReview {
		fields["review_state"], fields["review_actor"] = p.Review.State, p.Review.Actor
	}
	return fields, nil
}

func (a *financialIssuance) draftRecoveryPermission(workspace, actor, owner, capability string) string {
	if actor != owner {
		return "denied"
	}
	err := a.draftCallerMode(workspace, actor, capability, false)
	if err == nil {
		return "allowed"
	}
	var failure *commands.Failure
	if errors.As(err, &failure) && (failure.Code == commands.PermissionDenied || failure.Code == commands.Unauthenticated) {
		return "denied"
	}
	return "unavailable"
}

func (a *financialIssuance) draftRecoveryFields(workspace, actor string, p financialDraftObservation, fields map[string]string) error {
	v := p.Draft
	if v.Application == nil {
		return nil
	}
	appendRecovery := func(key, operation, owner, id, state, capability string, args map[string]string) error {
		value, err := draftJSON(map[string]any{"operation": operation, "original_actor": owner, "business_id": id, "state": state, "permission": a.draftRecoveryPermission(workspace, actor, owner, capability), "arguments": args})
		if err != nil {
			return err
		}
		fields[key] = value
		return nil
	}
	if v.State == "submission_unknown" || v.State == "submission_refused" {
		r := v.Spec.Application
		if err := appendRecovery("recovery_submit", "submit", v.Submitter, v.Spec.ID, v.State, draftauthority.Submit, map[string]string{"configuration": r.Configuration.ID, "configuration_digest": r.Configuration.Digest, "final_request": r.FinalRequest, "final_digest": r.FinalDigest, "language": v.Spec.Language}); err != nil {
			return err
		}
	}
	if p.HasReview && (p.Review.State == "prepared" || p.Review.State == "approval_unknown") {
		r := p.Review
		if err := appendRecovery("recovery_prepare", "prepare_approval", r.Actor, r.ID, r.State, draftauthority.Approve, map[string]string{"draft": r.DraftID, "draft_digest": r.DraftDigest, "binding": r.BindingID, "binding_digest": r.BindingDigest}); err != nil {
			return err
		}
		return appendRecovery("recovery_approve", "approve", r.Actor, r.ID, r.State, draftauthority.Approve, map[string]string{"digest": r.Digest})
	}
	return nil
}

func (a *financialIssuance) draftInspectionFields(workspace, actor string, p financialDraftObservation) (map[string]string, error) {
	f, err := draftSummary(p)
	if err != nil {
		return nil, err
	}
	delete(f, "review_state")
	delete(f, "review_actor")
	v := p.Draft
	source, err := draftOriginSource(v)
	if err != nil {
		return nil, err
	}
	maps.Copy(f, map[string]string{"mode": "synthetic", "current_eligibility": "unchecked", "origin": "legacy", "final_request": source.Basis.FinalRequest, "final_digest": source.Basis.FinalDigest, "language": v.Spec.Language, "final_requester": v.FinalRequester, "line_count": strconv.Itoa(len(v.Draft.Lines)), "receipt": "submission_record"})
	if v.Application != nil {
		f["origin"], f["configuration"], f["configuration_digest"] = "configured", v.Spec.Application.Configuration.ID, v.Spec.Application.Configuration.Digest
	}
	if p.HasReview {
		r := p.Review
		review := map[string]string{"id": r.ID, "digest": r.Digest, "actor": r.Actor, "state": r.State, "reviewed_at": r.ReviewedAt.Format(time.RFC3339Nano)}
		if r.State == "refused" {
			review["refusal"] = r.Refusal
		}
		f["review"], err = draftJSON(review)
		if err != nil {
			return nil, err
		}
		f["receipt"] = draftReviewResult(r).Receipt
		if len(r.References) != 3 {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		for i, ref := range r.References {
			f[fmt.Sprintf("reference_%02d", i)], err = draftJSON(map[string]string{"field": ref.Field, "subject": ref.Subject, "change": ref.Change, "ref_id": ref.Ref.ID, "fingerprint": ref.Ref.Fingerprint, "effective": ref.Effective.Format(time.RFC3339Nano)})
			if err != nil {
				return nil, err
			}
		}
		c := r.Coverage
		f["coverage"], err = draftJSON(map[string]string{"contract": c.Contract, "binding": c.Binding, "term": c.Term, "revision": strconv.Itoa(c.Revision), "net_days": strconv.Itoa(c.NetDays), "calendar_mode": c.Mode, "zone": c.Zone, "issue_date": c.Date, "due_date": c.Due})
		if err != nil {
			return nil, err
		}
	}
	if err := a.draftRecoveryFields(workspace, actor, p, f); err != nil {
		return nil, err
	}
	return f, nil
}
