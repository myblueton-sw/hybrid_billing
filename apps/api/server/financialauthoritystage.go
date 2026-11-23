package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type financialAuthorityDomain struct {
	stage, prefix, idPrefix string
	bindings                map[string]financialAuthorityBinding
}

func (s *Server) financialAuthorityDomain(stage financialPolicyStage) (financialAuthorityDomain, error) {
	switch stage {
	case chargePolicyStage:
		return financialAuthorityDomain{approvals.StageCharge, "financial_authority", "financial-limit-", s.financialAuthorities}, nil
	case erpFinalPolicyStage:
		return financialAuthorityDomain{approvals.StageERPFinal, "erp_final_authority", "erp-final-limit-", s.erpFinalAuthorities}, nil
	default:
		return financialAuthorityDomain{}, commands.Fail(commands.UnsupportedCapability)
	}
}
func (s *Server) financialAuthorityBounds(ctx commands.Context, mutation bool) error {
	return s.financialAuthorityBoundsForStage(ctx, mutation, chargePolicyStage)
}
func (s *Server) financialAuthoritySpec(ctx commands.Context) (approvals.Authority, error) {
	return s.financialAuthoritySpecForStage(ctx, chargePolicyStage)
}
func (s *Server) financialAuthorityRecord(workspace, id string) (financialAuthorityRecord, error) {
	return s.financialAuthorityRecordForStage(workspace, id, chargePolicyStage)
}
func (s *Server) proposeFinancialAuthority(ctx commands.Context) (financialAuthorityRecord, error) {
	return s.proposeFinancialAuthorityForStage(ctx, chargePolicyStage)
}
func (s *Server) confirmFinancialAuthority(ctx commands.Context) (financialAuthorityRecord, error) {
	return s.confirmFinancialAuthorityForStage(ctx, chargePolicyStage)
}
func (s *Server) revokeFinancialAuthority(ctx commands.Context) (financialAuthorityRecord, error) {
	return s.revokeFinancialAuthorityForStage(ctx, chargePolicyStage)
}
