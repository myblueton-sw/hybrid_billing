package server

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

type financialPolicyBinding struct {
	Digest             string
	Proposer           string
	ProposalReason     string
	ConfirmationReason string
}

func (s *Server) financialGovernor(workspace, actor string) error {
	if _, err := s.governor(workspace, actor); err != nil {
		return err
	}
	admins, err := s.approvals.Admins(workspace)
	if err != nil {
		return err
	}
	if !slices.Contains(admins, actor) {
		return inputauthority.ErrDenied
	}
	return nil
}

func financialPolicySpec(fields map[string]string) (approvals.Policy, error) {
	p := approvals.Policy{Stage: approvals.StageCharge}
	values := make([]int, 3)
	for index, name := range []string{"policy_revision", "approvals", "validity_hours"} {
		n, err := strconv.Atoi(fields[name])
		if err != nil || n < 1 || strconv.Itoa(n) != fields[name] {
			return p, approvals.ErrInvalid
		}
		values[index] = n
	}
	if values[1] > 3 || values[2] > 2160 {
		return p, approvals.ErrInvalid
	}
	p.Revision, p.Approvals, p.Validity = values[0], values[1], time.Duration(values[2])*time.Hour
	p.Required = strings.Split(fields["required"], ",")
	kinds := []string{"attribution", "erp_draft", "fx", "input", "mapping", "result", "rule"}
	for i, kind := range p.Required {
		if !slices.Contains(kinds, kind) || i > 0 && p.Required[i-1] >= kind {
			return p, approvals.ErrInvalid
		}
	}
	if !slices.Contains(p.Required, "result") {
		return p, approvals.ErrInvalid
	}
	return p, nil
}

func (s *Server) financialPolicyRecord(workspace, id string) (approvals.ProposalView, financialPolicyBinding, error) {
	binding, ok := s.financialPolicies[id]
	if !ok {
		return approvals.ProposalView{}, binding, approvals.ErrNotFound
	}
	record, err := s.approvals.ProposalOf(workspace, id)
	if err != nil {
		return record, binding, err
	}
	if record.Digest != binding.Digest || record.Proposer != binding.Proposer || record.Proposal.Kind != approvals.ProposePolicy || record.Proposal.Policy.Stage != approvals.StageCharge ||
		record.State != approvals.StatePending && record.State != approvals.StateConfirmed || (record.State == approvals.StateConfirmed) != (binding.ConfirmationReason != "") {
		return record, binding, approvals.ErrConflict
	}
	return record, binding, nil
}

func (s *Server) proposeFinancialPolicy(ctx commands.Context) (approvals.ProposalView, financialPolicyBinding, error) {
	if err := s.financialPolicyBounds(ctx, true); err != nil {
		return approvals.ProposalView{}, financialPolicyBinding{}, err
	}
	if err := s.governorCaller(ctx.Workspace, ctx.User); err != nil {
		return approvals.ProposalView{}, financialPolicyBinding{}, err
	}
	if err := s.financialGovernor(ctx.Workspace, ctx.User); err != nil {
		return approvals.ProposalView{}, financialPolicyBinding{}, err
	}
	policy, err := financialPolicySpec(ctx.Fields)
	if err != nil {
		return approvals.ProposalView{}, financialPolicyBinding{}, err
	}
	proposal := approvals.Proposal{ID: ctx.Target, Kind: approvals.ProposePolicy, Policy: policy}
	if _, exists := s.financialPolicies[ctx.Target]; exists {
		record, binding, err := s.financialPolicyRecord(ctx.Workspace, ctx.Target)
		if err != nil {
			return record, binding, err
		}
		if record.Proposer != ctx.User || binding.ProposalReason != ctx.Rationale || inputAuthorityFingerprint(record.Proposal) != inputAuthorityFingerprint(proposal) {
			return record, binding, approvals.ErrConflict
		}
		return record, binding, nil
	}
	if _, err := s.approvals.ProposalOf(ctx.Workspace, ctx.Target); !errors.Is(err, approvals.ErrNotFound) {
		if err == nil {
			err = approvals.ErrConflict
		}
		return approvals.ProposalView{}, financialPolicyBinding{}, err
	}
	if len(s.financialPolicies) >= 100 {
		return approvals.ProposalView{}, financialPolicyBinding{}, approvals.ErrLimit
	}
	current, err := s.approvals.PolicyOf(ctx.Workspace, approvals.StageCharge)
	if err != nil {
		return approvals.ProposalView{}, financialPolicyBinding{}, err
	}
	if policy.Revision <= current.Revision || policy.Revision-current.Revision != 1 {
		return approvals.ProposalView{}, financialPolicyBinding{}, approvals.ErrConflict
	}
	record, err := s.approvals.Propose(ctx.Workspace, proposal, approvals.Actor{ID: ctx.User, Kind: approvals.KindHuman})
	if err != nil {
		return record, financialPolicyBinding{}, err
	}
	binding := financialPolicyBinding{Digest: record.Digest, Proposer: record.Proposer, ProposalReason: ctx.Rationale}
	s.financialPolicies[ctx.Target] = binding
	return record, binding, nil
}

func (s *Server) confirmFinancialPolicy(ctx commands.Context) (approvals.ProposalView, financialPolicyBinding, error) {
	if err := s.financialPolicyBounds(ctx, true); err != nil {
		return approvals.ProposalView{}, financialPolicyBinding{}, err
	}
	if err := s.governorCaller(ctx.Workspace, ctx.User); err != nil {
		return approvals.ProposalView{}, financialPolicyBinding{}, err
	}
	if err := s.financialGovernor(ctx.Workspace, ctx.User); err != nil {
		return approvals.ProposalView{}, financialPolicyBinding{}, err
	}
	record, binding, err := s.financialPolicyRecord(ctx.Workspace, ctx.Target)
	if err != nil {
		return record, binding, err
	}
	if err := s.financialGovernor(ctx.Workspace, record.Proposer); err != nil {
		return record, binding, err
	}
	if ctx.User == record.Proposer {
		return record, binding, commands.Fail(commands.SelfApprovalDenied)
	}
	if ctx.Fields["digest"] != record.Digest || record.State == approvals.StateConfirmed && (binding.ConfirmationReason != ctx.Rationale || record.Confirmer != ctx.User) {
		return record, binding, approvals.ErrConflict
	}
	record, err = s.approvals.Confirm(ctx.Workspace, ctx.Target, approvals.Actor{ID: ctx.User, Kind: approvals.KindHuman})
	if err != nil {
		return record, binding, err
	}
	binding.ConfirmationReason = ctx.Rationale
	s.financialPolicies[ctx.Target] = binding
	return record, binding, nil
}
