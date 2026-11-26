package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
)

func (s *chargeCandidates) financialCandidate(workspace, id string) (chargeCandidate, error) {
	if !validRequestID(workspace) || !validRequestID(id) {
		return chargeCandidate{}, errCandidateInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, found := s.records[[2]string{workspace, id}]
	if !found {
		return chargeCandidate{}, errCandidateMissing
	}
	if len(record.envelope) > candidateBytes || len(record.input) > candidateBytes {
		return chargeCandidate{}, errCandidateLimit
	}
	sum := sha256.Sum256(record.envelope)
	if hex.EncodeToString(sum[:]) != record.digest {
		return chargeCandidate{}, errCandidateConflict
	}
	value, err := decodeCandidate(record)
	if err != nil {
		return chargeCandidate{}, err
	}
	input, err := json.Marshal(value.Envelope.Input)
	if err != nil {
		return chargeCandidate{}, err
	}
	if !bytes.Equal(input, record.input) {
		return chargeCandidate{}, errCandidateConflict
	}
	return value, nil
}

func financialEvidenceDigest(kind string, value any) (string, error) {
	return normalizedDigest(struct {
		Schema, Kind string
		Value        any
	}{"hb_financial_evidence_v1", kind, value})
}

func (s *Server) financialSubjectLocked(workspace, request, candidateID, digest string) (approvals.Subject, error) {
	if s.cfg.Seed != SeedSample || s.chargeProducer == nil || s.approvals == nil {
		return approvals.Subject{}, commands.Fail(commands.UnsupportedCapability)
	}
	if workspace != s.bootWorkspace || !validRequestID(request) {
		return approvals.Subject{}, errCandidateInvalid
	}
	p := s.chargeProducer
	value, err := p.candidates.financialCandidate(workspace, candidateID)
	if err != nil {
		return approvals.Subject{}, err
	}
	in, origin := value.Envelope.Input, value.Envelope.Input.Normalized
	if value.Envelope.Schema != "hb_charge_candidate_v7" || in.Workspace != workspace || in.ID != candidateID || value.Digest != digest || !candidateForm(in) || !validCandidateInput(in) || !validRequestID(origin.CaptureAuthor) || origin.InputEligibility == nil || origin.Pricing == nil || !value.Envelope.Result.NoRuleHeld || len(value.Envelope.Result.Excluded) != 0 {
		return approvals.Subject{}, errCandidateInvalid
	}
	actual, err := normalizedDigest(value.Envelope)
	if err != nil {
		return approvals.Subject{}, err
	}
	if actual != digest {
		return approvals.Subject{}, errCandidateConflict
	}
	owned, err := p.captureBinding(workspace, origin.Binding.ID)
	if err != nil {
		return approvals.Subject{}, err
	}
	if owned.SourceRun != origin.Run || p.pricingPolicies == nil {
		return approvals.Subject{}, errCandidateConflict
	}
	from, e1 := time.Parse(time.DateOnly, in.From)
	last, e2 := time.Parse(time.DateOnly, in.To)
	if e1 != nil || e2 != nil || last.AddDate(0, 0, 1).Year() > 9999 {
		return approvals.Subject{}, errCandidateInvalid
	}
	policy, err := p.pricingPolicies.CheckEvidence(owned.Scope, origin.Pricing.Proposal.ID, origin.Pricing.Digest, from, last.AddDate(0, 0, 1))
	if err != nil {
		return approvals.Subject{}, err
	}
	if !reflect.DeepEqual(policy, *origin.Pricing) {
		return approvals.Subject{}, errCandidateConflict
	}
	input, err := s.currentCaptureInputLocked(workspace, origin.InputEligibility.Review.ID, origin.InputEligibility.Review.Evidence.Digest, origin.InputEligibility.Publication.Revision, owned, policy)
	if err != nil {
		return approvals.Subject{}, err
	}
	b := owned.Binding
	b.Policy, b.Profile, b.Limits = policy.Proposal.Spec.Policy, policy.Proposal.Spec.Profile, policy.Proposal.Spec.Limits
	rebuilt, err := p.prepareEvidenceInputLocked(workspace, owned.SourceRun, candidateID, b, &policy, input, origin.CaptureAuthor)
	if err != nil {
		return approvals.Subject{}, err
	}
	if !reflect.DeepEqual(rebuilt, in) {
		return approvals.Subject{}, errCandidateConflict
	}
	result, err := rateCandidateInput(in)
	if err != nil {
		return approvals.Subject{}, err
	}
	if !reflect.DeepEqual(result, value.Envelope.Result) {
		return approvals.Subject{}, errCandidateConflict
	}
	financialPolicy, err := s.currentOwnedFinancialPolicyLocked(workspace)
	if err != nil {
		return approvals.Subject{}, err
	}
	target, err := financialSubjectTarget(in)
	if err != nil {
		return approvals.Subject{}, err
	}
	subject := approvals.Subject{ID: request, Stage: approvals.StageCharge, Target: "financial-target-" + target, Author: origin.CaptureAuthor, Amount: result.Total, Currency: result.Currency, Digests: map[string]string{}}
	for _, kind := range financialPolicy.Required {
		var evidence any
		switch kind {
		case "input":
			evidence = input
		case "rule":
			evidence = policy
		case "mapping":
			evidence = struct {
				Target, Mapping string
				Key             normalize.MappingKey
				Revision        uint64
			}{origin.TargetDigest, origin.MappingDigest, origin.Mapping, origin.MappingRevision}
		case "attribution":
			evidence = struct {
				Contract  organizations.ContractRevision
				Profile   organizations.BillingProfile
				Document  organizations.DocumentKey
				Lifecycle candidateLifecycle
				Target    pricingpolicy.Target
			}{origin.Contract, origin.BillingProfile, origin.Document, origin.Lifecycle, policy.Proposal.Target}
		case "result":
			subject.Digests[kind] = digest
			continue
		default:
			return approvals.Subject{}, commands.Fail(commands.PrerequisiteFailed)
		}
		subject.Digests[kind], err = financialEvidenceDigest(kind, evidence)
		if err != nil {
			return approvals.Subject{}, err
		}
	}
	prepared, err := s.approvals.PrepareSubject(subject)
	if err != nil {
		return approvals.Subject{}, err
	}
	if prepared.Amount != subject.Amount {
		return approvals.Subject{}, errCandidateInvalid
	}
	return prepared, nil
}

func financialSubjectTarget(in chargeInput) (string, error) {
	b := in.Normalized.Binding
	return normalizedDigest(struct {
		Schema, Workspace, Customer, Entity, Contract, BillingProfile, Provider, Account, Period, From, To string
	}{"hb_financial_target_v1", in.Workspace, in.Customer, in.Entity, b.Contract, b.BillingProfile, b.Provider, b.Scope, b.Period, in.From, in.To})
}
