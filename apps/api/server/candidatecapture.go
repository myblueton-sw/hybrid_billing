package server

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
)

type candidateCaptureBinding struct {
	Scope     pricingpolicy.Scope
	SourceRun string
	Binding   normalizedBinding
}

type candidateInputProvenance struct {
	Review           inputreview.Record
	Publication      inputreview.Publication
	SourceRun        string
	Document         string
	DocumentRevision int
	DocumentDigest   string
	ComparisonDigest string
	BindingDigest    string
}

func (p *normalizedCharges) retainCaptureBindingLocked(workspace, run string, b normalizedBinding) error {
	if !validRequestID(workspace) || !validRequestID(run) || !validRequestID(b.ID) || !validRequestID(b.Version) || b.SourceMode != "synthetic" {
		return errCandidateInvalid
	}
	contract, _, document, err := p.attribution(workspace, b)
	if err != nil {
		return err
	}
	var empty normalizedBinding
	b.Policy, b.Profile, b.Limits = empty.Policy, empty.Profile, empty.Limits
	value := candidateCaptureBinding{Scope: pricingpolicy.Scope{Workspace: workspace, Customer: contract.CustomerID, Entity: document.IssuerLegalEntityID, Contract: b.Contract, BillingProfile: b.BillingProfile}, SourceRun: run, Binding: b}
	body, err := json.Marshal(value)
	if err != nil || len(body) > candidateBytes {
		return errCandidateInvalid
	}
	key := [2]string{workspace, b.ID}
	if prior, ok := p.captureBindings[key]; ok {
		if !bytes.Equal(prior, body) {
			return errCandidateConflict
		}
		return nil
	}
	if len(p.captureBindings) >= 1000 {
		return errCandidateLimit
	}
	if p.captureBindings == nil {
		p.captureBindings = make(map[[2]string][]byte)
	}
	p.captureBindings[key] = body
	return nil
}

func (p *normalizedCharges) captureBinding(workspace, id string) (candidateCaptureBinding, error) {
	var out candidateCaptureBinding
	body, ok := p.captureBindings[[2]string{workspace, id}]
	if !ok {
		return out, errCandidateMissing
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, err
	}
	if out.Scope.Workspace != workspace || out.Binding.ID != id {
		return out, errCandidateInvalid
	}
	return out, nil
}

func validCandidateInput(in chargeInput) bool {
	p := in.Normalized.InputEligibility
	if author := in.Normalized.CaptureAuthor; author != "" && (!validRequestID(author) || p == nil) {
		return false
	}
	if p == nil {
		return true
	}
	r, publication := p.Review, p.Publication
	scope := inputreview.Scope{Workspace: in.Workspace, Provider: in.Normalized.Binding.Provider, Account: in.Normalized.Binding.Scope, Period: in.Normalized.Binding.Period}
	return in.Normalized.Pricing != nil && in.Policy.CostBasis == rating.Billed && r.State == inputreview.Eligible && r.Evidence.Verdict == inputreview.Match && r.Decision != nil &&
		r.Evidence.Scope == scope && publication.Scope == scope && publication.Record == r.ID && publication.Digest == r.Evidence.Digest && r.Decision.Digest == r.Evidence.Digest &&
		publication.Revision > 0 && publication.Revision <= inputreview.MaxRevision && publication.Revision == r.Decision.ScopeRevision &&
		p.SourceRun == in.Normalized.Run && validRequestID(p.Document) && p.DocumentRevision > 0 && len(p.DocumentDigest) == 64 && len(p.ComparisonDigest) == 64 && len(p.BindingDigest) == 64
}

func (s *Server) eligibleCaptureInput(ctx commands.Context, owned candidateCaptureBinding, policy pricingpolicy.Record) (*candidateInputProvenance, error) {
	r := s.inputReviews
	if r == nil || r.producer != s.chargeProducer {
		return nil, commands.Fail(commands.UnsupportedCapability)
	}
	revision, err := strconv.ParseUint(ctx.Fields["scope_revision"], 10, 32)
	if err != nil || revision == 0 || strconv.FormatUint(revision, 10) != ctx.Fields["scope_revision"] {
		return nil, commands.Fail(commands.InvalidRequest, "scope_revision")
	}
	record, err := r.registry.Record(ctx.Workspace, ctx.Fields["input_review"])
	if err != nil {
		return nil, inputReviewError(err)
	}
	b := owned.Binding
	scope := inputreview.Scope{Workspace: ctx.Workspace, Provider: b.Provider, Account: b.Scope, Period: b.Period}
	if record.State != inputreview.Eligible || record.Decision == nil || record.Evidence.Verdict != inputreview.Match {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	if record.Evidence.Scope != scope || record.Evidence.Digest != ctx.Fields["input_digest"] || record.Decision.Digest != record.Evidence.Digest || record.Decision.ScopeRevision != revision {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	publication, err := r.registry.Current(scope)
	if err != nil {
		return nil, inputReviewError(err)
	}
	if publication.Record != record.ID || publication.Digest != record.Evidence.Digest || publication.Revision != revision {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	fresh, _, err := r.currentEvidenceLocked(ctx.Workspace, record.Evidence.Run)
	if err != nil {
		return nil, inputReviewError(err)
	}
	if !reflect.DeepEqual(fresh, record.Evidence) {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	held := r.bindings[[2]string{ctx.Workspace, record.Evidence.Run}]
	if held == nil || held.SourceRun != owned.SourceRun || held.Document.Scope != scope || held.Document.Basis != normalize.Billed || policy.Proposal.Spec.Policy.CostBasis != rating.Billed || held.Document.Currency != policy.Proposal.Spec.Policy.Currency {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	b.Policy = policy.Proposal.Spec.Policy
	cost, _, err := s.chargeProducer.fields(b)
	if err != nil || held.Document.Field != cost {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	return &candidateInputProvenance{Review: record, Publication: publication, SourceRun: owned.SourceRun, Document: held.Document.ID, DocumentRevision: held.Document.Revision, DocumentDigest: held.Digest, ComparisonDigest: held.Comparison, BindingDigest: held.Print}, nil
}

func (h pricingPolicyCommand) capture(ctx commands.Context) (commands.Outcome, error) {
	p := h.s.chargeProducer
	owned, err := p.captureBinding(ctx.Workspace, ctx.Fields["binding"])
	if err != nil {
		return commands.Outcome{}, commands.Fail(commands.PermissionDenied)
	}
	policy, err := p.pricingPolicies.Record(ctx.Workspace, ctx.Fields["policy"])
	if err != nil {
		return commands.Outcome{}, pricingPolicyError(err)
	}
	if policy.Proposal.Spec.Scope != owned.Scope {
		return commands.Outcome{}, commands.Fail(commands.EvidenceStale)
	}
	input, err := h.s.eligibleCaptureInput(ctx, owned, policy)
	if err != nil {
		return commands.Outcome{}, err
	}
	candidate, err := p.captureGovernedInputLocked(p.pricingPolicies, owned.Scope, ctx.Fields["policy"], ctx.Fields["policy_digest"], ctx.User, owned.SourceRun, ctx.Target, owned.Binding, input, ctx.User)
	if err != nil {
		return commands.Outcome{}, pricingPolicyError(err)
	}
	return commands.Outcome{ResultRevision: h.s.Next(), BusinessState: "captured", Fields: map[string]string{"mode": "synthetic", "candidate": ctx.Target, "digest": candidate.Digest, "receipt": "historical_capture"}}, nil
}
