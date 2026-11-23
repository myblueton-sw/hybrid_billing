package server

import (
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func draftLineFields(fields map[string]string, lines []issuance.Line, start int) error {
	for i := start; i < min(start+10, len(lines)); i++ {
		line := lines[i]
		value, err := draftJSON(struct {
			Index  string `json:"index"`
			Text   string `json:"text"`
			Amount string `json:"amount"`
			Tax    bool   `json:"tax"`
		}{strconv.Itoa(i), line.Text, line.Amount, line.Tax})
		if err != nil {
			return err
		}
		fields[fmt.Sprintf("line_%02d", i-start)] = value
	}
	return nil
}

func draftMutationReceipt(p financialDraftRecord, r *financialDraftReview) map[string]string {
	f := map[string]string{"mode": "synthetic", "draft": p.Spec.ID, "digest": p.Digest, "submission_state": p.State, "receipt": "submission_record", "current_eligibility": "unchecked"}
	if r != nil {
		f["review"], f["review_digest"], f["review_state"], f["receipt"] = r.ID, r.Digest, r.State, draftReviewResult(*r).Receipt
	}
	return f
}

func (a *financialIssuance) admitDraftProjection(p financialDraftRecord, review *financialDraftReview) error {
	if len(p.Draft.Lines) == 0 || len(p.Draft.Lines) > 200 {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	p = copyFinancialDraft(p)
	p.Submitter, p.FinalRequester = strings.Repeat("a", maxRequestID), strings.Repeat("a", maxRequestID)
	p.Module.At = time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
	states := []string{"submission_unknown", "submission_refused", "submitted"}
	if review != nil {
		states = []string{"prepared", "approval_unknown", "approved", "refused"}
	}
	for _, state := range states {
		observation := financialDraftObservation{Draft: p, Module: "matching_unapproved"}
		var r *financialDraftReview
		if review == nil {
			observation.Draft.State = state
		} else {
			v := copyDraftReview(*review)
			v.State, v.Actor, v.ReviewedAt = state, strings.Repeat("a", maxRequestID), p.Module.At
			v.Refusal = ""
			if state == "refused" {
				v.Refusal = issuance.ReasonCoverageMismatch
			}
			observation.Draft.State = "submitted"
			observation.Review, observation.HasReview = v, true
			r = &v
		}
		if err := a.boundedDraftFields(draftMutationReceipt(observation.Draft, r)); err != nil {
			return err
		}
		fields, err := a.draftInspectionFields(p.Workspace, a.server.answer.User(), observation)
		if err != nil {
			return err
		}
		for _, key := range []string{"recovery_submit", "recovery_prepare", "recovery_approve"} {
			if fields[key] == "" {
				continue
			}
			var descriptor map[string]any
			if json.Unmarshal([]byte(fields[key]), &descriptor) != nil {
				return commands.Fail(commands.PrerequisiteFailed)
			}
			descriptor["permission"] = "unavailable"
			fields[key], err = draftJSON(descriptor)
			if err != nil {
				return err
			}
		}
		fields["snapshot"], fields["next_line"] = strings.Repeat("d", 64), "199"
		for start := 0; start < len(p.Draft.Lines); start += 10 {
			page := maps.Clone(fields)
			if err := draftLineFields(page, p.Draft.Lines, start); err != nil {
				return err
			}
			if err := a.boundedDraftFields(page); err != nil {
				return err
			}
		}
	}
	return nil
}
