package server

import (
	"errors"
	"maps"
	"reflect"
	"slices"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type financialFinalBinding struct {
	Workspace, Requester, Digest, EvidenceDigest string
	Candidate                                    financialFinalCandidate
	RequestReason, DecisionReason                string
}

func cloneFinancialFinalCandidate(c financialFinalCandidate) financialFinalCandidate {
	c.Subject.Digests = maps.Clone(c.Subject.Digests)
	c.Subject.Entries = slices.Clone(c.Subject.Entries)
	c.Policy.Required = slices.Clone(c.Policy.Required)
	c.Contributors = slices.Clone(c.Contributors)
	return c
}

func (s *Server) financialFinalRecord(workspace, id string) (approvals.RequestView, financialFinalBinding, error) {
	b, ok := s.financialFinals[id]
	if !ok || workspace != s.bootWorkspace || b.Workspace != workspace || s.approvals == nil {
		return approvals.RequestView{}, financialFinalBinding{}, approvals.ErrNotFound
	}
	v, err := s.approvals.RequestOf(workspace, id)
	if err != nil {
		return v, financialFinalBinding{}, err
	}
	digest, err := financialEvidenceDigest("erp_final_binding", b.Candidate)
	if err != nil || digest != b.EvidenceDigest || b.Candidate.Subject.ID != id || b.Candidate.Subject.Stage != approvals.StageERPFinal || b.Candidate.Policy.Stage != approvals.StageERPFinal || b.Candidate.Policy.Revision != v.Policy || b.Requester != v.Requester || b.Digest != v.Digest || !reflect.DeepEqual(b.Candidate.Subject, v.Subject) {
		return approvals.RequestView{}, financialFinalBinding{}, approvals.ErrConflict
	}
	b.Candidate = cloneFinancialFinalCandidate(b.Candidate)
	return v, b, nil
}

func (s *Server) checkFinancialFinalEvidence(workspace, id, actor string, b financialFinalBinding) error {
	c, err := s.financialFinalSubjectLocked(workspace, id, b.Candidate.ResultID, b.Candidate.ResultDigest, actor)
	if err != nil {
		return financialChargeEvidenceError(err)
	}
	if !reflect.DeepEqual(c, b.Candidate) {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}

func (s *Server) requestFinancialFinal(ctx commands.Context) (approvals.RequestView, error) {
	if s.financialFinals == nil || s.approvals == nil || s.cfg.Seed != SeedSample {
		return approvals.RequestView{}, commands.Fail(commands.UnsupportedCapability)
	}
	if s.held() {
		return approvals.RequestView{}, commands.Fail(commands.PrerequisiteFailed)
	}
	old, exists := s.financialFinals[ctx.Target]
	if !exists && s.financialApprovalCount() >= approvalCap {
		return approvals.RequestView{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if exists && (old.Workspace != ctx.Workspace || old.Requester != ctx.User || old.RequestReason != ctx.Rationale || old.Candidate.ResultID != ctx.Fields["result"] || old.Candidate.ResultDigest != ctx.Fields["digest"]) {
		return approvals.RequestView{}, commands.Fail(commands.EvidenceStale)
	}
	if !exists {
		if _, err := s.approvals.RequestOf(ctx.Workspace, ctx.Target); !errors.Is(err, approvals.ErrNotFound) {
			return approvals.RequestView{}, commands.Fail(commands.EvidenceStale)
		}
	}
	candidate, err := s.financialFinalSubjectLocked(ctx.Workspace, ctx.Target, ctx.Fields["result"], ctx.Fields["digest"], ctx.User)
	if err != nil {
		return approvals.RequestView{}, financialChargeEvidenceError(err)
	}
	if exists {
		v, b, err := s.financialFinalRecord(ctx.Workspace, ctx.Target)
		if err != nil || v.State != approvals.StatePending || !reflect.DeepEqual(candidate, b.Candidate) {
			return approvals.RequestView{}, commands.Fail(commands.EvidenceStale)
		}
		return v, nil
	}
	evidence, err := financialEvidenceDigest("erp_final_binding", candidate)
	if err != nil {
		return approvals.RequestView{}, err
	}
	if s.cfg.Commands.Fields < 24 || s.cfg.Commands.Value < 1024 || len(ctx.Rationale) > s.cfg.Commands.Rationale {
		return approvals.RequestView{}, commands.Fail(commands.UnsupportedCapability)
	}
	b := financialFinalBinding{Workspace: ctx.Workspace, Requester: ctx.User, Candidate: cloneFinancialFinalCandidate(candidate), EvidenceDigest: evidence, RequestReason: ctx.Rationale}
	v, err := s.approvals.Request(ctx.Workspace, candidate.Subject, approvals.Actor{ID: ctx.User, Kind: approvals.KindHuman})
	if err != nil {
		return v, financialPolicyError(err)
	}
	b.Digest = v.Digest
	s.financialFinals[ctx.Target] = b
	return v, nil
}

func (s *Server) decideFinancialFinal(ctx commands.Context, verdict string) (approvals.RequestView, error) {
	if verdict != approvals.VerdictApprove && verdict != approvals.VerdictReject {
		return approvals.RequestView{}, commands.Fail(commands.InvalidRequest)
	}
	v, b, err := s.financialFinalRecord(ctx.Workspace, ctx.Target)
	if err != nil || v.Digest != ctx.Fields["digest"] || v.State != approvals.StatePending {
		return approvals.RequestView{}, commands.Fail(commands.EvidenceStale)
	}
	if err := financialFinalContributorCheck(b.Candidate, b.Requester, ctx.User); err != nil {
		return approvals.RequestView{}, err
	}
	if err := s.checkFinancialFinalEvidence(ctx.Workspace, ctx.Target, ctx.User, b); err != nil {
		return approvals.RequestView{}, err
	}
	if verdict == approvals.VerdictApprove {
		if s.held() {
			return approvals.RequestView{}, commands.Fail(commands.PrerequisiteFailed)
		}
		if err := s.currentOwnedFinancialAuthorityCoverage(ctx.Workspace, ctx.User, erpFinalPolicyStage, v.Subject.Currency, v.Subject.Amount); err != nil {
			return approvals.RequestView{}, err
		}
	}
	v, err = s.approvals.Decide(ctx.Workspace, ctx.Target, approvals.Decision{Verdict: verdict, Reason: ctx.Fields["reason"]}, approvals.Actor{ID: ctx.User, Kind: approvals.KindHuman})
	if err != nil {
		return v, financialPolicyError(err)
	}
	b.DecisionReason = ctx.Rationale
	s.financialFinals[ctx.Target] = b
	return v, nil
}

func financialFinalReceipt(v approvals.RequestView) map[string]string {
	return map[string]string{"mode": "synthetic", "request": v.Subject.ID, "digest": v.Digest, "recorded_state": v.State, "receipt": "historical_decision", "stage": approvals.StageERPFinal}
}

func financialFinalReview(v approvals.RequestView, b financialFinalBinding) map[string]string {
	fields := financialFinalReceipt(v)
	fields["receipt"] = "historical_evidence"
	fields["current_eligibility"] = "unchecked"
	fields["result"], fields["result_digest"] = b.Candidate.ResultID, b.Candidate.ResultDigest
	fields["author"], fields["requester"] = v.Subject.Author, v.Requester
	fields["amount"], fields["currency"] = v.Subject.Amount, v.Subject.Currency
	fields["net"], fields["tax"] = b.Candidate.Net, b.Candidate.Tax
	fields["amount_basis"] = b.Candidate.AmountProfile
	fields["policy_revision"] = strconv.Itoa(v.Policy)
	for kind, digest := range v.Subject.Digests {
		fields["subject_"+kind+"_digest"] = digest
	}
	return fields
}
