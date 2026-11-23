package server

import (
	"errors"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

type financialAuthorityBinding struct {
	financialPolicyBinding
	RevocationReason string
}

type financialAuthorityRecord struct {
	proposal  approvals.ProposalView
	binding   financialAuthorityBinding
	authority *approvals.AuthorityView
	asOf      time.Time
}

func (s *Server) financialRecipient(actor string) error {
	facts, err := s.accessFacts(s.caller)
	if err != nil {
		return err
	}
	member, ok := facts.members[actor]
	user := facts.users[actor]
	if !ok || !member.Effective || member.UserState != identity.Active || user.Kind != identity.Human || user.State != identity.Active {
		return inputauthority.ErrDenied
	}
	return nil
}

func (s *Server) financialAuthorityBoundsForStage(ctx commands.Context, mutation bool, stage financialPolicyStage) error {
	domain, err := s.financialAuthorityDomain(stage)
	if err != nil {
		return err
	}
	if s.cfg.Seed != SeedSample || s.approvals == nil || domain.bindings == nil || s.cfg.Commands.Fields < 23 || s.cfg.Commands.Value < 128 || s.cfg.Commands.Value < s.cfg.Commands.Rationale {
		return commands.Fail(commands.UnsupportedCapability)
	}
	if mutation && (strings.TrimSpace(ctx.Rationale) == "" || len(ctx.Rationale) > s.cfg.Commands.Rationale) {
		return approvals.ErrInvalid
	}
	return nil
}

func (s *Server) financialAuthoritySpecForStage(ctx commands.Context, stage financialPolicyStage) (approvals.Authority, error) {
	domain, err := s.financialAuthorityDomain(stage)
	if err != nil {
		return approvals.Authority{}, err
	}
	a := approvals.Authority{ID: domain.idPrefix + ctx.Target, Approver: ctx.Fields["recipient"], Stage: domain.stage, Currency: ctx.Fields["currency"], Limit: ctx.Fields["limit"], From: ctx.Fields["from"], To: ctx.Fields["to"]}
	if a.To == "" {
		return a, approvals.ErrInvalid
	}
	canonical, err := s.approvals.PrepareAuthority(a)
	if err != nil {
		return a, err
	}
	if canonical != a {
		return a, approvals.ErrInvalid
	}
	return a, nil
}

func (s *Server) financialAuthorityRecordForStage(workspace, id string, stage financialPolicyStage) (financialAuthorityRecord, error) {
	out := financialAuthorityRecord{}
	domain, err := s.financialAuthorityDomain(stage)
	if err != nil {
		return out, err
	}
	binding, ok := domain.bindings[id]
	if !ok {
		return out, approvals.ErrNotFound
	}
	record, err := s.approvals.ProposalOf(workspace, id)
	if err != nil {
		return out, err
	}
	out.proposal, out.binding = record, binding
	if record.Proposal.Kind != approvals.ProposeGrant || record.Proposal.Authority.Stage != domain.stage || record.Proposal.Authority.ID != domain.idPrefix+id || record.Digest != binding.Digest || record.Proposer != binding.Proposer ||
		record.State != approvals.StatePending && record.State != approvals.StateConfirmed || (record.State == approvals.StateConfirmed) != (binding.ConfirmationReason != "") {
		return out, approvals.ErrConflict
	}
	if record.State == approvals.StatePending {
		if binding.RevocationReason != "" {
			return out, approvals.ErrConflict
		}
		return out, nil
	}
	authority, at, err := s.approvals.AuthorityAt(workspace, record.Proposal.Authority.ID)
	if err != nil {
		return out, err
	}
	if authority.Authority != record.Proposal.Authority || authority.Proposer != record.Proposer || authority.Confirmer != record.Confirmer || !authority.Granted.Equal(record.Confirmed) || (!authority.Revoked.IsZero()) != (binding.RevocationReason != "") {
		return out, approvals.ErrConflict
	}
	out.authority = &authority
	out.asOf = at
	return out, nil
}

func (s *Server) proposeFinancialAuthorityForStage(ctx commands.Context, stage financialPolicyStage) (financialAuthorityRecord, error) {
	out := financialAuthorityRecord{}
	domain, err := s.financialAuthorityDomain(stage)
	if err != nil {
		return out, err
	}
	if err := s.financialAuthorityBoundsForStage(ctx, true, stage); err != nil {
		return out, err
	}
	if err := s.governorCaller(ctx.Workspace, ctx.User); err != nil {
		return out, err
	}
	if err := s.financialGovernor(ctx.Workspace, ctx.User); err != nil {
		return out, err
	}
	authority, err := s.financialAuthoritySpecForStage(ctx, stage)
	if err != nil {
		return out, err
	}
	if authority.Approver == ctx.User {
		return out, commands.Fail(commands.SelfApprovalDenied)
	}
	if err := s.financialRecipient(authority.Approver); err != nil {
		return out, err
	}
	if _, exists := domain.bindings[ctx.Target]; exists {
		out, err = s.financialAuthorityRecordForStage(ctx.Workspace, ctx.Target, stage)
		if err != nil {
			return out, err
		}
		if out.proposal.Proposer != ctx.User || out.binding.ProposalReason != ctx.Rationale || out.proposal.Proposal.Authority != authority {
			return out, approvals.ErrConflict
		}
		return out, nil
	}
	if _, err := s.approvals.ProposalOf(ctx.Workspace, ctx.Target); !errors.Is(err, approvals.ErrNotFound) {
		if err == nil {
			err = approvals.ErrConflict
		}
		return out, err
	}
	if _, err := s.approvals.AuthorityOf(ctx.Workspace, authority.ID); !errors.Is(err, approvals.ErrNotFound) {
		if err == nil {
			err = approvals.ErrConflict
		}
		return out, err
	}
	if len(domain.bindings) >= 100 {
		return out, approvals.ErrLimit
	}
	proposal, err := s.approvals.Propose(ctx.Workspace, approvals.Proposal{ID: ctx.Target, Kind: approvals.ProposeGrant, Authority: authority}, approvals.Actor{ID: ctx.User, Kind: approvals.KindHuman})
	if err != nil {
		return out, err
	}
	binding := financialAuthorityBinding{financialPolicyBinding: financialPolicyBinding{Digest: proposal.Digest, Proposer: proposal.Proposer, ProposalReason: ctx.Rationale}}
	domain.bindings[ctx.Target] = binding
	return financialAuthorityRecord{proposal: proposal, binding: binding}, nil
}

func (s *Server) confirmFinancialAuthorityForStage(ctx commands.Context, stage financialPolicyStage) (financialAuthorityRecord, error) {
	out := financialAuthorityRecord{}
	domain, err := s.financialAuthorityDomain(stage)
	if err != nil {
		return out, err
	}
	if err := s.financialAuthorityBoundsForStage(ctx, true, stage); err != nil {
		return out, err
	}
	if err := s.governorCaller(ctx.Workspace, ctx.User); err != nil {
		return out, err
	}
	if err := s.financialGovernor(ctx.Workspace, ctx.User); err != nil {
		return out, err
	}
	out, err = s.financialAuthorityRecordForStage(ctx.Workspace, ctx.Target, stage)
	if err != nil {
		return out, err
	}
	if err := s.financialGovernor(ctx.Workspace, out.proposal.Proposer); err != nil {
		return out, err
	}
	if err := s.financialRecipient(out.proposal.Proposal.Authority.Approver); err != nil {
		return out, err
	}
	if ctx.User == out.proposal.Proposer || ctx.User == out.proposal.Proposal.Authority.Approver {
		return out, commands.Fail(commands.SelfApprovalDenied)
	}
	if ctx.Fields["digest"] != out.proposal.Digest || out.proposal.State == approvals.StateConfirmed && (out.proposal.Confirmer != ctx.User || out.binding.ConfirmationReason != ctx.Rationale) {
		return out, approvals.ErrConflict
	}
	record, err := s.approvals.Confirm(ctx.Workspace, ctx.Target, approvals.Actor{ID: ctx.User, Kind: approvals.KindHuman})
	if err != nil {
		return out, err
	}
	out.proposal = record
	out.binding.ConfirmationReason = ctx.Rationale
	if out.authority == nil {
		out.asOf = record.Confirmed
		out.authority = &approvals.AuthorityView{Authority: record.Proposal.Authority, Proposer: record.Proposer, Confirmer: record.Confirmer, Granted: record.Confirmed}
	}
	domain.bindings[ctx.Target] = out.binding
	return out, nil
}

func (s *Server) revokeFinancialAuthorityForStage(ctx commands.Context, stage financialPolicyStage) (financialAuthorityRecord, error) {
	out := financialAuthorityRecord{}
	domain, err := s.financialAuthorityDomain(stage)
	if err != nil {
		return out, err
	}
	if err := s.financialAuthorityBoundsForStage(ctx, true, stage); err != nil {
		return out, err
	}
	if err := s.governorCaller(ctx.Workspace, ctx.User); err != nil {
		return out, err
	}
	if err := s.financialGovernor(ctx.Workspace, ctx.User); err != nil {
		return out, err
	}
	out, err = s.financialAuthorityRecordForStage(ctx.Workspace, ctx.Target, stage)
	if err != nil {
		return out, err
	}
	if out.authority == nil {
		return out, approvals.ErrRejected
	}
	if ctx.Fields["digest"] != out.proposal.Digest || !out.authority.Revoked.IsZero() && (out.authority.RevokedBy != ctx.User || out.binding.RevocationReason != ctx.Rationale) {
		return out, approvals.ErrConflict
	}
	authority, err := s.approvals.Revoke(ctx.Workspace, out.authority.Authority.ID, approvals.Actor{ID: ctx.User, Kind: approvals.KindHuman})
	if err != nil {
		return out, err
	}
	out.authority = &authority
	if out.asOf.Before(authority.Revoked) {
		out.asOf = authority.Revoked
	}
	out.binding.RevocationReason = ctx.Rationale
	domain.bindings[ctx.Target] = out.binding
	return out, nil
}

func financialAmountStatus(record financialAuthorityRecord, now time.Time) string {
	if record.authority == nil {
		return "not_granted"
	}
	a := record.authority
	if !a.Revoked.IsZero() {
		return "revoked"
	}
	today := now.UTC().Format(time.DateOnly)
	if today < a.Authority.From {
		return "not_yet_valid"
	}
	if a.Authority.To != "" && today > a.Authority.To {
		return "expired"
	}
	return "active"
}
