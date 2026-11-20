package server

import (
	"reflect"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
)

type candidatePricingTargets struct{ producer *normalizedCharges }
type candidatePricingParty struct {
	Party    organizations.Party
	Revision uint64
}

func pricingDays(from, to time.Time) (contracts.Interval, error) {
	if from.Location() != time.UTC || to.Location() != time.UTC || from.Hour() != 0 || from.Minute() != 0 || from.Second() != 0 || from.Nanosecond() != 0 || to.Hour() != 0 || to.Minute() != 0 || to.Second() != 0 || to.Nanosecond() != 0 {
		return contracts.Interval{}, errCandidateInvalid
	}
	start, err := contracts.NewDay(from.Year(), int(from.Month()), from.Day())
	if err != nil {
		return contracts.Interval{}, err
	}
	end, err := contracts.NewDay(to.Year(), int(to.Month()), to.Day())
	if err != nil {
		return contracts.Interval{}, err
	}
	return contracts.Between(start, end)
}
func (t candidatePricingTargets) Current(scope pricingpolicy.Scope, from, to time.Time) (pricingpolicy.Target, error) {
	p := t.producer
	period, err := pricingDays(from, to)
	if err != nil {
		return pricingpolicy.Target{}, err
	}
	sequence, err := p.lifecycles.Sequence(scope.Workspace, scope.Contract)
	if err != nil {
		return pricingpolicy.Target{}, err
	}
	segments, err := p.lifecycles.SegmentsAt(scope.Workspace, scope.Contract, sequence, period)
	if err != nil {
		return pricingpolicy.Target{}, err
	}
	if len(segments) == 0 {
		return pricingpolicy.Target{}, errCandidateInvalid
	}
	profile, billingRevision, err := p.orgs.BillingProfile(scope.Workspace, scope.BillingProfile)
	if err != nil {
		return pricingpolicy.Target{}, err
	}
	contract, err := p.orgs.ContractRevision(scope.Workspace, scope.Contract, segments[0].Revision)
	if err != nil {
		return pricingpolicy.Target{}, err
	}
	b := normalizedBinding{Contract: scope.Contract, ContractRevision: contract.Revision, BillingProfile: scope.BillingProfile, BillingRevision: billingRevision, From: from.Format(time.DateOnly), To: to.AddDate(0, 0, -1).Format(time.DateOnly), Policy: rating.Policy{Currency: profile.Currency, Path: rating.Path(contract.Path)}}
	contract, profile, document, err := p.attribution(scope.Workspace, b)
	if err != nil {
		return pricingpolicy.Target{}, err
	}
	if contract.CustomerID != scope.Customer || document.IssuerLegalEntityID != scope.Entity {
		return pricingpolicy.Target{}, errCandidateInvalid
	}
	life, err := p.lifecycleCoverage(scope.Workspace, b)
	if err != nil {
		return pricingpolicy.Target{}, err
	}
	workspace, workspaceRevision, err := p.orgs.Workspace(scope.Workspace)
	if err != nil {
		return pricingpolicy.Target{}, err
	}
	parties := []candidatePricingParty{}
	for _, id := range []string{profile.IssuerID, profile.BuyerID, profile.PayerID, profile.RecipientID} {
		if id == "" {
			continue
		}
		party, revision, e := p.orgs.Party(scope.Workspace, id)
		if e != nil {
			return pricingpolicy.Target{}, e
		}
		parties = append(parties, candidatePricingParty{party, revision})
	}
	fingerprint, err := normalizedDigest(struct {
		Workspace         organizations.Workspace
		WorkspaceRevision uint64
		Contract          organizations.ContractRevision
		Profile           organizations.BillingProfile
		ProfileRevision   uint64
		Document          organizations.DocumentKey
		Parties           []candidatePricingParty
		Lifecycle         candidateLifecycle
	}{workspace, workspaceRevision, contract, profile, billingRevision, document, parties, life})
	if err != nil {
		return pricingpolicy.Target{}, err
	}
	return pricingpolicy.Target{ContractRevision: contract.Revision, BillingRevision: billingRevision, Currency: profile.Currency, Path: rating.Path(contract.Path), Fingerprint: fingerprint}, nil
}
func (p *normalizedCharges) newPricingPolicies(authority pricingpolicy.Authority, clock func() time.Time) (*pricingpolicy.Registry, error) {
	if p.pricingPolicies != nil {
		return nil, errCandidateConflict
	}
	registry, err := pricingpolicy.New(1000, authority, candidatePricingTargets{p}, clock)
	if err != nil {
		return nil, err
	}
	p.pricingPolicies = registry
	return registry, nil
}
func (p *normalizedCharges) captureGoverned(policies *pricingpolicy.Registry, scope pricingpolicy.Scope, policyID, digest, actor, runID, candidateID string, b normalizedBinding) (chargeCandidate, error) {
	p.domain.Lock()
	defer p.domain.Unlock()
	return p.captureGovernedLocked(policies, scope, policyID, digest, actor, runID, candidateID, b)
}
func (p *normalizedCharges) captureGovernedLocked(policies *pricingpolicy.Registry, scope pricingpolicy.Scope, policyID, digest, actor, runID, candidateID string, b normalizedBinding) (chargeCandidate, error) {
	return p.captureGovernedInputLocked(policies, scope, policyID, digest, actor, runID, candidateID, b, nil)
}

func (p *normalizedCharges) captureGovernedInputLocked(policies *pricingpolicy.Registry, scope pricingpolicy.Scope, policyID, digest, actor, runID, candidateID string, b normalizedBinding, eligibility *candidateInputProvenance) (chargeCandidate, error) {
	if policies == nil || policies != p.pricingPolicies || !reflect.ValueOf(b.Policy).IsZero() || !reflect.ValueOf(b.Profile).IsZero() || !reflect.ValueOf(b.Limits).IsZero() || b.Contract != scope.Contract || b.BillingProfile != scope.BillingProfile {
		return chargeCandidate{}, errCandidateInvalid
	}
	from, e1 := time.Parse(time.DateOnly, b.From)
	last, e2 := time.Parse(time.DateOnly, b.To)
	if e1 != nil || e2 != nil || last.AddDate(0, 0, 1).Year() > 9999 {
		return chargeCandidate{}, errCandidateInvalid
	}
	approved, err := policies.Resolve(scope, policyID, digest, actor, from, last.AddDate(0, 0, 1))
	if err != nil {
		return chargeCandidate{}, err
	}
	if b.ContractRevision != approved.Proposal.Target.ContractRevision || b.BillingRevision != approved.Proposal.Target.BillingRevision {
		return chargeCandidate{}, errCandidateConflict
	}
	b.Policy = approved.Proposal.Spec.Policy
	b.Profile = approved.Proposal.Spec.Profile
	b.Limits = approved.Proposal.Spec.Limits
	return p.captureEvidenceLocked(scope.Workspace, runID, candidateID, b, &approved, eligibility)
}
