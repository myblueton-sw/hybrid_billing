package server

import (
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type financialPolicyStage uint8

const (
	chargePolicyStage financialPolicyStage = iota
	erpFinalPolicyStage
)

type financialPolicyDomain struct {
	stage, prefix string
	bindings      map[string]financialPolicyBinding
}

func (s *Server) financialPolicyDomain(stage financialPolicyStage) (financialPolicyDomain, error) {
	switch stage {
	case chargePolicyStage:
		return financialPolicyDomain{approvals.StageCharge, "financial_policy", s.financialPolicies}, nil
	case erpFinalPolicyStage:
		return financialPolicyDomain{approvals.StageERPFinal, "erp_final_policy", s.erpFinalPolicies}, nil
	default:
		return financialPolicyDomain{}, commands.Fail(commands.UnsupportedCapability)
	}
}

func financialPolicySpec(fields map[string]string) (approvals.Policy, error) {
	return financialPolicySpecForStage(fields, chargePolicyStage)
}
func (s *Server) financialPolicyRecord(workspace, id string) (approvals.ProposalView, financialPolicyBinding, error) {
	return s.financialPolicyRecordForStage(workspace, id, chargePolicyStage)
}
func (s *Server) proposeFinancialPolicy(ctx commands.Context) (approvals.ProposalView, financialPolicyBinding, error) {
	return s.proposeFinancialPolicyForStage(ctx, chargePolicyStage)
}
func (s *Server) confirmFinancialPolicy(ctx commands.Context) (approvals.ProposalView, financialPolicyBinding, error) {
	return s.confirmFinancialPolicyForStage(ctx, chargePolicyStage)
}
func (s *Server) financialPolicyBounds(ctx commands.Context, mutation bool) error {
	return s.financialPolicyBoundsForStage(ctx, mutation, chargePolicyStage)
}
func (s *Server) currentOwnedFinancialPolicyLocked(workspace string) (approvals.Policy, error) {
	return s.currentOwnedFinancialPolicyForStage(workspace, chargePolicyStage)
}

func (s *Server) currentOwnedFinancialPolicyForStage(workspace string, stage financialPolicyStage) (approvals.Policy, error) {
	d, err := s.financialPolicyDomain(stage)
	if err != nil {
		return approvals.Policy{}, err
	}
	if s.approvals == nil || d.bindings == nil {
		return approvals.Policy{}, commands.Fail(commands.UnsupportedCapability)
	}
	policy, err := s.approvals.PolicyOf(workspace, d.stage)
	if err != nil {
		return approvals.Policy{}, err
	}
	found := false
	for id := range d.bindings {
		record, _, err := s.financialPolicyRecordForStage(workspace, id, stage)
		if err != nil {
			return approvals.Policy{}, err
		}
		if record.State == approvals.StateConfirmed && record.Proposal.Policy.Revision == policy.Revision {
			if found || !reflect.DeepEqual(record.Proposal.Policy, policy) {
				return approvals.Policy{}, approvals.ErrConflict
			}
			found = true
		}
	}
	if !found {
		return approvals.Policy{}, commands.Fail(commands.PrerequisiteFailed)
	}
	return policy, nil
}
