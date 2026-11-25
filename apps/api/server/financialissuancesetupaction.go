package server

import (
	"errors"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func setupActionCode(err error) string {
	var failure *commands.Failure
	if errors.As(configurationPublicError(err, nil), &failure) {
		switch failure.Code {
		case commands.PermissionDenied, commands.SelfApprovalDenied, commands.EvidenceStale, commands.UnsupportedCapability:
			return string(failure.Code)
		}
	}
	return "dependency_unavailable"
}

func (a *financialIssuance) setupActionFields(workspace, actor, contract string, input map[string]string) (map[string]string, error) {
	if err := a.protectedReadCaller(workspace, actor); err != nil {
		return nil, err
	}
	operation, id := input["operation"], input["configuration"]
	if !validRequestID(contract) || !validRequestID(id) || !setupMutation(operation) {
		return nil, commands.Fail(commands.InvalidRequest)
	}
	if operation == "propose_configuration" {
		if _, ok := parseContractDay(input["on"]); !ok || !validPlanHash(input["scope_digest"]) || input["digest"] != "" {
			return nil, commands.Fail(commands.InvalidRequest)
		}
	} else if !validPlanHash(input["digest"]) || input["on"] != "" || input["scope_digest"] != "" {
		return nil, commands.Fail(commands.InvalidRequest)
	}
	f := map[string]string{"schema": "hb_issuance_setup_action_v1", "contract": contract, "configuration": id, "operation": operation, "permission": "allowed", "step_up": "unchecked", "action": "blocked", "code": "", "current_eligibility": "unchecked"}
	finish := func(code string) (map[string]string, error) {
		f["code"] = code
		if err := a.boundedDraftFields(f); err != nil {
			return nil, err
		}
		return f, nil
	}
	if _, err := a.server.governor(workspace, actor); err != nil {
		f["permission"] = "unavailable"
		code := "authority_unavailable"
		if setupActionCode(err) == "permission_denied" {
			f["permission"], code = "denied", "permission_denied"
		}
		return finish(code)
	}
	if a.server.held() {
		return finish("installation_held")
	}
	switch a.server.stepUpCheck(a.server.caller, actor, "issuance_setup."+operation, id) {
	case "":
		f["step_up"] = "satisfied"
	case CodeReauthentication:
		f["step_up"] = "reauthentication_required"
		return finish("reauthentication_required")
	case commands.PrerequisiteFailed:
		f["step_up"] = "authentication_policy_unsatisfied"
		return finish("authentication_policy_unsatisfied")
	default:
		return finish("authority_unavailable")
	}
	p, exists := a.configurations[id]
	if exists {
		if p.Workspace != workspace || p.Spec.Contract != contract || p.Spec.ID != id {
			return finish("permission_denied")
		}
		if p.Setup == nil {
			return finish("unsupported_capability")
		}
		if operation == "propose_configuration" {
			if p.Proposer != actor {
				return finish("permission_denied")
			}
			if p.Setup.On != input["on"] || p.Setup.ScopeDigest != input["scope_digest"] {
				return finish("evidence_stale")
			}
		} else {
			if p.Proposer == actor {
				return finish("self_approval_denied")
			}
			if p.Confirmer != "" && p.Confirmer != actor {
				return finish("permission_denied")
			}
			if p.Digest != input["digest"] {
				return finish("evidence_stale")
			}
		}
		if _, err := a.observeConfigurationHistory(p); err != nil {
			return finish(setupActionCode(err))
		}
		if err := a.currentConfiguration(workspace, p); err != nil {
			return finish(setupActionCode(err))
		}
		context, err := a.setupContextFields(workspace, actor, contract, p.Setup.On)
		if err != nil {
			return finish(setupActionCode(err))
		}
		if context["scope_digest"] != p.Setup.ScopeDigest {
			return finish("evidence_stale")
		}
		if operation == "confirm_configuration" && p.State == "proposal_unknown" {
			return finish("configuration_pending")
		}
		f["action"] = "recheck_required"
		if operation == "propose_configuration" && p.State != "proposal_unknown" || operation == "confirm_configuration" && p.State == "confirmed" {
			f["action"] = "recorded"
		}
		return finish("")
	}
	if operation == "confirm_configuration" {
		return finish("permission_denied")
	}
	context, err := a.setupContextFields(workspace, actor, contract, input["on"])
	if err != nil {
		return finish(setupActionCode(err))
	}
	if context["scope_digest"] != input["scope_digest"] {
		return finish("evidence_stale")
	}
	if err := a.configurationHistoryIndex(); err != nil {
		return finish(setupActionCode(err))
	}
	for _, other := range a.configurations {
		if other.Workspace == workspace && other.Spec.Contract == contract && strconv.FormatUint(other.Spec.ContractRevision, 10) == context["contract_revision"] && other.Spec.BillingProfile == context["billing_profile"] && strconv.FormatUint(other.Spec.BillingRevision, 10) == context["billing_revision"] {
			return finish("configuration_exists")
		}
	}
	f["action"] = "recheck_required"
	return finish("")
}
