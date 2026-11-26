package server

import (
	"slices"

	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
)

func draftContextCommonAction(f map[string]string, permission string, submit bool) string {
	switch f[permission] {
	case "denied":
		return "permission_denied"
	case "unavailable":
		return "authority_unavailable"
	}
	if f["readiness"] == "installation_held" {
		return "installation_held"
	}
	if f["step_up"] != "satisfied" {
		return f["step_up"]
	}
	switch f["readiness"] {
	case "unavailable":
		return "evidence_unavailable"
	case "source_ineligible":
		return "source_ineligible"
	case "configuration_missing":
		return "setup_missing"
	case "configuration_ambiguous":
		return "setup_ambiguous"
	case "terms_missing":
		if !submit {
			return "setup_missing"
		}
	}
	return ""
}

func (a *financialIssuance) draftContextActions(workspace, actor string, p *financialDraftContext) {
	f, existing := p.Evidence.Fields, p.Evidence.Existing
	submit, prepare, approve := "ready", "draft_required", "review_required"
	if existing != nil {
		submit = "existing_draft"
		if existing.Draft.Application == nil {
			submit, prepare, approve = "legacy_read_only", "legacy_read_only", "legacy_read_only"
		} else if existing.Draft.State == "submitted" {
			source, err := draftOriginSource(existing.Draft)
			if err != nil {
				prepare, approve = "evidence_unavailable", "evidence_unavailable"
			} else {
				prepare = "ready"
				if actor == existing.Draft.Submitter || actor == existing.Draft.FinalRequester || actor == source.Basis.Subject.Author || slices.Contains(source.Basis.Contributors, actor) {
					prepare = "independent_reviewer_required"
				}
				if existing.HasReview {
					r := existing.Review
					switch r.State {
					case "approved":
						prepare, approve = "completed", "completed"
					case "prepared", "approval_unknown":
						prepare, approve = "owned_by_other", "owned_by_other"
						if r.Actor == actor {
							prepare, approve = "unresolved_operation", "ready"
							if p.Drift {
								approve = "recheck_required"
							}
							if r.State == "approval_unknown" && r.Module.Digest != "" {
								approve = "unresolved_operation"
							}
						}
					}
				}
				if (prepare == "ready" || approve == "ready" || approve == "recheck_required") && a.draftActor(workspace, existing.Draft.Submitter, draftauthority.Submit) != nil {
					prepare, approve = "evidence_unavailable", "evidence_unavailable"
				}
			}
		}
	}
	f["submit_action"], f["prepare_action"], f["approve_action"] = submit, prepare, approve
	for _, operation := range []string{"submit", "prepare", "approve"} {
		permission := "approve_permission"
		if operation == "submit" {
			permission = "submit_permission"
		}
		if action := draftContextCommonAction(f, permission, operation == "submit"); action != "" {
			f[operation+"_action"] = action
		}
	}
}
