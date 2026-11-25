package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"strconv"
	"time"
)

func (s *Server) prepareFinancialFinal(ctx commands.Context) (commands.Outcome, error) {
	v, b, err := s.financialFinalRecord(ctx.Workspace, ctx.Target)
	if err != nil || ctx.Fields["digest"] != v.Digest {
		return commands.Outcome{}, commands.Fail(commands.EvidenceStale)
	}
	request := gate.Request{Action: gate.Approve, Kind: "erp_final_approval", Resource: gate.Resource{WorkspaceID: ctx.Workspace}}
	code := financialGateCode(s.gate.CheckApproval(s.answer, v.Subject.Author, request))
	common := "eligible"
	if code != "" {
		common = string(code)
	} else if financialFinalContributorCheck(b.Candidate, v.Requester, ctx.User) != nil {
		common = string(commands.SelfApprovalDenied)
	} else if v.State != approvals.StatePending {
		common = "not_pending"
	} else if len(s.stranded) > 0 {
		common = string(commands.DependencyUnavailable)
	} else {
		common = preparationEvidenceCode(s.checkFinancialFinalEvidence(ctx.Workspace, ctx.Target, ctx.User, b))
	}
	outcomes := map[string]string{}
	for _, action := range []string{approvals.VerdictApprove, approvals.VerdictReject} {
		current := common
		if current == "eligible" && action == approvals.VerdictApprove && s.held() {
			current = "hold_active"
		}
		if current == "eligible" {
			switch step := s.stepUpCheck(s.caller, ctx.User, "erp_final."+action, ctx.Target); step {
			case "":
			case commands.Unauthenticated:
				return commands.Outcome{}, commands.Fail(commands.Unauthenticated)
			case CodeReauthentication, commands.PrerequisiteFailed:
				current = string(step)
			default:
				current = string(commands.DependencyUnavailable)
			}
		}
		if current == "eligible" && action == approvals.VerdictApprove {
			current = preparationEvidenceCode(s.currentOwnedFinancialAuthorityCoverage(ctx.Workspace, ctx.User, erpFinalPolicyStage, v.Subject.Currency, v.Subject.Amount))
		}
		if current == "eligible" {
			decision := approvals.Decision{Verdict: action}
			if action == approvals.VerdictReject {
				decision.Reason = "other"
			}
			current = preparationModuleCode(s.approvals.CheckDecision(ctx.Workspace, ctx.Target, decision, approvals.Actor{ID: ctx.User, Kind: approvals.KindHuman}))
		}
		outcomes[action] = current
	}
	return commands.Outcome{Fields: map[string]string{
		"mode": "synthetic", "receipt": "current_decision_preparation", "request": ctx.Target, "digest": v.Digest, "recorded_state": v.State,
		"author": v.Subject.Author, "requester": v.Requester, "amount": v.Subject.Amount, "currency": v.Subject.Currency, "amount_basis": financialFinalAmountProfile, "policy_revision": strconv.Itoa(v.Policy),
		"checked_at": s.clock().UTC().Format(time.RFC3339Nano), "revision": s.Revision(), "approve": outcomes["approve"], "reject": outcomes["reject"],
	}}, nil
}
