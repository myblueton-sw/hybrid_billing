package server

import (
	"errors"
	"maps"
	"reflect"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
)

type financialChargeBinding struct {
	Workspace, Candidate, CandidateDigest, Requester, Digest string
	Subject                                                  approvals.Subject
	Policy                                                   int
	RequestReason, DecisionReason                            string
}

func (s *Server) financialChargeRecord(workspace, id string) (approvals.RequestView, financialChargeBinding, error) {
	b, ok := s.financialCharges[id]
	if !ok || workspace != s.bootWorkspace || b.Workspace != workspace || s.approvals == nil {
		return approvals.RequestView{}, financialChargeBinding{}, approvals.ErrNotFound
	}
	v, err := s.approvals.RequestOf(workspace, id)
	if err != nil {
		return v, b, err
	}
	if b.Subject.ID != id || b.Subject.Stage != approvals.StageCharge || b.Subject.Author == "" || b.Requester != v.Requester || b.Policy != v.Policy || b.Digest != v.Digest || b.CandidateDigest != v.Subject.Digests["result"] || !reflect.DeepEqual(b.Subject, v.Subject) {
		return approvals.RequestView{}, financialChargeBinding{}, approvals.ErrConflict
	}
	b.Subject.Digests = maps.Clone(b.Subject.Digests)
	b.Subject.Entries = append([]approvals.Entry(nil), b.Subject.Entries...)
	return v, b, nil
}

func financialChargeEvidenceError(err error) error {
	var failure *commands.Failure
	if errors.As(err, &failure) {
		switch failure.Code {
		case commands.DependencyUnavailable, commands.AuthorityUnavailable, commands.UnsupportedCapability, commands.PrerequisiteFailed:
			return err
		default:
			return commands.Fail(commands.EvidenceStale)
		}
	}
	for _, known := range []error{errCandidateInvalid, errCandidateMissing, errCandidateConflict, errCandidateLimit, approvals.ErrNotFound, approvals.ErrInvalid, approvals.ErrConflict, approvals.ErrRejected, pricingpolicy.ErrNotFound, pricingpolicy.ErrInvalid, pricingpolicy.ErrConflict, pricingpolicy.ErrRejected} {
		if errors.Is(err, known) {
			return commands.Fail(commands.EvidenceStale)
		}
	}
	return commands.Fail(commands.DependencyUnavailable)
}

func financialChargeReceipt(v approvals.RequestView) map[string]string {
	return map[string]string{"mode": "synthetic", "request": v.Subject.ID, "digest": v.Digest, "recorded_state": v.State, "receipt": "historical_decision"}
}

func (s *Server) requestFinancialCharge(ctx commands.Context) (approvals.RequestView, error) {
	old, exists := s.financialCharges[ctx.Target]
	if !exists && len(s.financialCharges) >= approvalCap {
		return approvals.RequestView{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if exists && (old.Workspace != ctx.Workspace || old.Requester != ctx.User || old.Candidate != ctx.Fields["candidate"] || old.CandidateDigest != ctx.Fields["digest"]) {
		return approvals.RequestView{}, commands.Fail(commands.EvidenceStale)
	}
	if !exists {
		if _, err := s.approvals.RequestOf(ctx.Workspace, ctx.Target); !errors.Is(err, approvals.ErrNotFound) {
			return approvals.RequestView{}, commands.Fail(commands.EvidenceStale)
		}
	}
	subject, err := s.financialSubjectLocked(ctx.Workspace, ctx.Target, ctx.Fields["candidate"], ctx.Fields["digest"])
	if err != nil {
		return approvals.RequestView{}, financialChargeEvidenceError(err)
	}
	policy, err := s.currentOwnedFinancialPolicyLocked(ctx.Workspace)
	if err != nil {
		return approvals.RequestView{}, financialChargeEvidenceError(err)
	}
	if policy.Approvals != 1 {
		return approvals.RequestView{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if exists {
		v, b, err := s.financialChargeRecord(ctx.Workspace, ctx.Target)
		if err != nil || v.State != approvals.StatePending || b.Policy != policy.Revision || !reflect.DeepEqual(subject, b.Subject) {
			return approvals.RequestView{}, commands.Fail(commands.EvidenceStale)
		}
		return v, nil
	}
	v, err := s.approvals.Request(ctx.Workspace, subject, approvals.Actor{ID: ctx.User, Kind: approvals.KindHuman})
	if err != nil {
		return v, financialPolicyError(err)
	}
	s.financialCharges[ctx.Target] = financialChargeBinding{Workspace: ctx.Workspace, Candidate: ctx.Fields["candidate"], CandidateDigest: ctx.Fields["digest"], Requester: ctx.User, Subject: subject, Policy: v.Policy, Digest: v.Digest, RequestReason: ctx.Rationale}
	return v, nil
}

func (s *Server) decideFinancialCharge(ctx commands.Context, kind string) (approvals.RequestView, error) {
	v, b, err := s.financialChargeRecord(ctx.Workspace, ctx.Target)
	if err != nil || v.Digest != ctx.Fields["digest"] || v.State != approvals.StatePending {
		return approvals.RequestView{}, commands.Fail(commands.EvidenceStale)
	}
	subject, err := s.financialSubjectLocked(ctx.Workspace, ctx.Target, b.Candidate, b.CandidateDigest)
	if err != nil {
		return approvals.RequestView{}, financialChargeEvidenceError(err)
	}
	if !reflect.DeepEqual(subject, b.Subject) {
		return approvals.RequestView{}, commands.Fail(commands.EvidenceStale)
	}
	policy, err := s.currentOwnedFinancialPolicyLocked(ctx.Workspace)
	if err != nil {
		return approvals.RequestView{}, financialChargeEvidenceError(err)
	}
	if policy.Revision != b.Policy || policy.Approvals != 1 {
		return approvals.RequestView{}, commands.Fail(commands.EvidenceStale)
	}
	v, err = s.approvals.Decide(ctx.Workspace, ctx.Target, approvals.Decision{Verdict: kind, Reason: ctx.Fields["reason"]}, approvals.Actor{ID: ctx.User, Kind: approvals.KindHuman})
	if err != nil {
		return v, financialPolicyError(err)
	}
	b.DecisionReason = ctx.Rationale
	s.financialCharges[ctx.Target] = b
	return v, nil
}

func financialChargeReview(v approvals.RequestView, b financialChargeBinding) map[string]string {
	fields := financialChargeReceipt(v)
	fields["receipt"] = "historical_evidence"
	fields["current_eligibility"] = "unchecked"
	fields["candidate"], fields["candidate_digest"] = b.Candidate, b.CandidateDigest
	fields["author"], fields["requester"] = v.Subject.Author, v.Requester
	fields["amount"], fields["currency"] = v.Subject.Amount, v.Subject.Currency
	fields["amount_basis"] = "rated_candidate_total"
	fields["policy_revision"] = strconv.Itoa(v.Policy)
	for kind, digest := range v.Subject.Digests {
		fields[kind+"_digest"] = digest
	}
	return fields
}
