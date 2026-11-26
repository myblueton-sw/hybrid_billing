package server

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (s *Server) previewFinancialFinal(ctx commands.Context) (commands.Outcome, error) {
	candidate, err := s.financialFinalSubjectLocked(ctx.Workspace, ctx.Target, ctx.Fields["result"], ctx.Fields["digest"], ctx.User)
	if err != nil {
		return commands.Outcome{}, financialChargeEvidenceError(err)
	}
	status := "evidence_ready"
	if _, err := s.approvals.RequestOf(ctx.Workspace, ctx.Target); err == nil {
		status = "target_unavailable"
	} else if !errors.Is(err, approvals.ErrNotFound) {
		return commands.Outcome{}, financialPolicyError(err)
	}
	if s.financialApprovalCount() >= approvalCap {
		status = "capacity_reached"
	}
	if len(s.stranded) > 0 {
		status = "dependency_unavailable"
	}
	if s.held() {
		status = "hold_active"
	}
	evidence, err := financialEvidenceDigest("erp_final_binding", candidate)
	if err != nil {
		return commands.Outcome{}, err
	}
	contributors, err := json.Marshal(candidate.Contributors)
	if err != nil {
		return commands.Outcome{}, err
	}
	fields := map[string]string{
		"mode": "synthetic", "receipt": "current_request_preview", "request": ctx.Target,
		"result": candidate.ResultID, "result_digest": candidate.ResultDigest, "candidate_digest": evidence,
		"author": candidate.Subject.Author, "requester": ctx.User, "contributors": string(contributors),
		"amount": candidate.Subject.Amount, "currency": candidate.Subject.Currency, "net": candidate.Net, "tax": candidate.Tax,
		"amount_basis": candidate.AmountProfile, "policy_revision": strconv.Itoa(candidate.Policy.Revision),
		"checked_at": s.clock().UTC().Format(time.RFC3339Nano), "revision": s.Revision(), "request_status": status,
	}
	if len(fields) > s.cfg.Commands.Fields {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	for _, value := range fields {
		if len(value) > s.cfg.Commands.Value {
			return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
		}
	}
	return commands.Outcome{Fields: fields}, nil
}
