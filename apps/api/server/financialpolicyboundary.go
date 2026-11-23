package server

import (
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (s *Server) financialPolicyBoundary(command string, call commands.Call) commands.Code {
	stage, kind := chargePolicyStage, ""
	switch command {
	case "financial_policy.propose":
		kind = "propose"
	case "financial_policy.confirm":
		kind = "confirm"
	case "financial_policy.inspect":
		kind = "inspect"
	case "financial_policy.current":
		kind = "current"
	case "erp_final_policy.propose":
		stage, kind = erpFinalPolicyStage, "propose"
	case "erp_final_policy.confirm":
		stage, kind = erpFinalPolicyStage, "confirm"
	case "erp_final_policy.inspect":
		stage, kind = erpFinalPolicyStage, "inspect"
	case "erp_final_policy.current":
		stage, kind = erpFinalPolicyStage, "current"
	default:
		return ""
	}
	domain, err := s.financialPolicyDomain(stage)
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
		record, _, err := s.financialPolicyRecordForStage(call.Answer.Workspace(), call.Target, stage)
		if err != nil {
			return code(err)
		}
		if err := s.financialGovernor(call.Answer.Workspace(), record.Proposer); err != nil {
			return code(err)
		}
		if call.Answer.User() == record.Proposer {
			return commands.SelfApprovalDenied
		}
	}
	return ""
}
