package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (s *Server) financialAuthorityBoundary(command string, call commands.Call) commands.Code {
	stage, kind := chargePolicyStage, ""
	switch command {
	case "financial_authority.propose":
		kind = "propose"
	case "financial_authority.confirm":
		kind = "confirm"
	case "financial_authority.revoke":
		kind = "revoke"
	case "financial_authority.inspect":
		kind = "inspect"
	case "erp_final_authority.propose":
		stage, kind = erpFinalPolicyStage, "propose"
	case "erp_final_authority.confirm":
		stage, kind = erpFinalPolicyStage, "confirm"
	case "erp_final_authority.revoke":
		stage, kind = erpFinalPolicyStage, "revoke"
	case "erp_final_authority.inspect":
		stage, kind = erpFinalPolicyStage, "inspect"
	default:
		return ""
	}
	domain, err := s.financialAuthorityDomain(stage)
	if err != nil {
		return commands.UnsupportedCapability
	}
	if s.cfg.Seed != SeedSample || s.approvals == nil || domain.bindings == nil {
		return ""
	}
	code := func(err error) commands.Code {
		if err == nil {
			return ""
		}
		var failure *commands.Failure
		if errors.As(financialPolicyError(err), &failure) {
			return failure.Code
		}
		return commands.DependencyUnavailable
	}
	if err := s.governorCaller(call.Answer.Workspace(), call.Answer.User()); err != nil {
		return code(err)
	}
	if err := s.financialGovernor(call.Answer.Workspace(), call.Answer.User()); err != nil {
		return code(err)
	}
	if kind == "confirm" {
		record, err := s.financialAuthorityRecordForStage(call.Answer.Workspace(), call.Target, stage)
		if err != nil {
			return code(err)
		}
		if err := s.financialGovernor(call.Answer.Workspace(), record.proposal.Proposer); err != nil {
			return code(err)
		}
		if err := s.financialRecipient(record.proposal.Proposal.Authority.Approver); err != nil {
			return code(err)
		}
		if call.Answer.User() == record.proposal.Proposer || call.Answer.User() == record.proposal.Proposal.Authority.Approver {
			return commands.SelfApprovalDenied
		}
	}
	return ""
}
