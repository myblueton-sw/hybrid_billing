package server

import (
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (s *Server) currentOwnedFinancialAuthorityCoverage(workspace, approver string, stage financialPolicyStage, currency, amount string) error {
	domain, err := s.financialAuthorityDomain(stage)
	if err != nil {
		return err
	}
	if s.cfg.Seed != SeedSample || s.approvals == nil || domain.bindings == nil {
		return commands.Fail(commands.UnsupportedCapability)
	}
	if len(domain.bindings) > 100 {
		return approvals.ErrLimit
	}
	ids := []string{}
	for id := range domain.bindings {
		record, err := s.financialAuthorityRecordForStage(workspace, id, stage)
		if err != nil {
			return err
		}
		if record.proposal.State == approvals.StateConfirmed && record.proposal.Proposal.Authority.Approver == approver {
			ids = append(ids, record.proposal.Proposal.Authority.ID)
		}
	}
	covered := false
	for _, id := range ids {
		err := s.approvals.CheckAuthorityCoverage(workspace, id, approver, domain.stage, currency, amount)
		if err == nil {
			covered = true
			continue
		}
		var refusal approvals.Refusal
		if !errors.As(err, &refusal) || refusal.Reason != approvals.ReasonNoAuthority {
			return err
		}
	}
	if !covered {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return nil
}
