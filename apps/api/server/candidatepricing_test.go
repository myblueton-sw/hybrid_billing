package server

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type pricingFixtureAuthority struct{ deny bool }

func (a *pricingFixtureAuthority) Check(scope pricingpolicy.Scope, actor, action string) error {
	if a.deny || scope.Workspace != "workspace" {
		return pricingpolicy.ErrRejected
	}
	if action == pricingpolicy.ProposeAction && actor == "maker" || action == pricingpolicy.ApproveAction && actor == "checker" || action == pricingpolicy.RevokeAction && actor == "revoker" || action == pricingpolicy.UseAction && actor == "operator" {
		return nil
	}
	return pricingpolicy.ErrRejected
}
func pricingFixture(t *testing.T) (*normalizedCharges, normalizedBinding, *pricingpolicy.Registry, *pricingFixtureAuthority, pricingpolicy.Record) {
	t.Helper()
	p, b, packet := normalizedFixtureSetup(t, false, nil)
	packet.Run.Expected["billed"]["USD"] = "15"
	for i, amount := range []string{"10.25", "4.75"} {
		packet.Artifacts[i].Rows[0].Cells["billed"] = normalize.Cell{Text: amount}
		packet.Artifacts[i].Sums["USD"] = amount
	}
	p.domain.Lock()
	defer p.domain.Unlock()
	if _, err := p.assembleSourceLocked(packet); err != nil {
		t.Fatal(err)
	}
	authority := &pricingFixtureAuthority{}
	registry, err := p.newPricingPolicies(authority, func() time.Time { return time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	spec := pricingpolicy.Spec{Mode: "synthetic", Scope: pricingpolicy.Scope{Workspace: "workspace", Customer: "customer", Entity: "seller", Contract: "contract", BillingProfile: "billing"}, From: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), Policy: b.Policy, Profile: b.Profile, Limits: b.Limits}
	proposal, err := registry.Propose("price-proposal", "maker", "Synthetic pass-through policy", spec)
	if err != nil {
		t.Fatal(err)
	}
	return p, b, registry, authority, proposal
}
func pricingApprove(t *testing.T, p *normalizedCharges, r *pricingpolicy.Registry, record pricingpolicy.Record) pricingpolicy.Record {
	t.Helper()
	p.domain.Lock()
	defer p.domain.Unlock()
	out, err := r.Approve("workspace", record.Proposal.ID, record.Digest, "checker", "Independent synthetic review")
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func pricingBinding(b normalizedBinding) normalizedBinding {
	b.Policy = rating.Policy{}
	b.Profile = decimal.ProfileSpec{}
	b.Limits = rating.Limits{}
	return b
}
func TestGovernedPricingActualSourceAndImmutableCandidate(t *testing.T) {
	p, b, r, _, pending := pricingFixture(t)
	if _, err := p.captureGoverned(r, pending.Proposal.Spec.Scope, pending.Proposal.ID, pending.Digest, "operator", "normalized", "governed", pricingBinding(b)); !errors.Is(err, pricingpolicy.ErrRejected) {
		t.Fatal("pending price used", err)
	}
	a := pricingApprove(t, p, r, pending)
	got, err := p.captureGoverned(r, a.Proposal.Spec.Scope, a.Proposal.ID, a.Digest, "operator", "normalized", "governed", pricingBinding(b))
	if err != nil {
		t.Fatal(err)
	}
	if got.Envelope.Result.Total != "15" || got.Envelope.Result.Currency != "USD" || got.Envelope.Schema != "hb_charge_candidate_v5" || !reflect.DeepEqual(got.Envelope.Input.Normalized.Pricing, &a) || !reflect.DeepEqual(got.Envelope.Input.Policy, a.Proposal.Spec.Policy) {
		t.Fatal("wrong governed calculation", got.Envelope.Result)
	}
	bytes, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	p.domain.Lock()
	_, err = r.Revoke("workspace", a.Proposal.ID, a.Digest, "revoker", "Stop new use")
	p.domain.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.captureGoverned(r, a.Proposal.Spec.Scope, a.Proposal.ID, a.Digest, "operator", "normalized", "governed", pricingBinding(b)); !errors.Is(err, pricingpolicy.ErrRejected) {
		t.Fatal("cached candidate bypassed revocation", err)
	}
	got.Envelope.Input.Normalized.Pricing.Approval.Actor = "changed-returned-copy"
	got.Envelope.Input.Normalized.Pricing.Proposal.Spec.Policy.Rules[0].ID = "changed-returned-rule"
	history, err := p.candidates.read("workspace", "governed")
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(history)
	if string(bytes) != string(after) {
		t.Fatal("historical candidate changed")
	}
	sample, err := p.capture("workspace", "normalized", "historical-example", b)
	if err != nil || sample.Envelope.Input.Normalized.Pricing != nil {
		t.Fatal("example inherited approval", err)
	}
}
func TestGovernedPricingRejectsReplacementTermsAndChangedTargets(t *testing.T) {
	for _, kind := range []string{"policy", "profile", "limits", "binding-revision", "scope", "target-revision", "party", "lifecycle", "permission"} {
		t.Run(kind, func(t *testing.T) {
			p, b, r, authority, pending := pricingFixture(t)
			a := pricingApprove(t, p, r, pending)
			supplied := pricingBinding(b)
			scope := a.Proposal.Spec.Scope
			switch kind {
			case "policy":
				supplied.Policy = b.Policy
			case "profile":
				supplied.Profile = b.Profile
			case "limits":
				supplied.Limits = b.Limits
			case "binding-revision":
				supplied.ContractRevision++
			case "scope":
				scope.Customer = "other"
			case "target-revision":
				p.domain.Lock()
				profile, rev, err := p.orgs.BillingProfile("workspace", "billing")
				if err != nil {
					t.Fatal(err)
				}

				_, err = p.orgs.PutBillingProfile(profile, rev)
				p.domain.Unlock()
				if err != nil {
					t.Fatal(err)
				}
			case "party":
				p.domain.Lock()
				party, rev, err := p.orgs.Party("workspace", "customer")
				if err != nil {
					t.Fatal(err)
				}
				party.State = organizations.Suspended
				_, err = p.orgs.PutParty(party, rev)
				p.domain.Unlock()
				if err != nil {
					t.Fatal(err)
				}
			case "lifecycle":
				p.domain.Lock()
				seq, err := p.lifecycles.Sequence("workspace", "contract")
				if err != nil {
					t.Fatal(err)
				}
				_, err = p.lifecycles.Amend("workspace", "contract", seq, "pause", nil, []contracts.Event{{ID: "suspend", Kind: contracts.Suspend, From: contractDay(2026, 10, 15)}}, "")
				p.domain.Unlock()
				if err != nil {
					t.Fatal(err)
				}
			case "permission":
				authority.deny = true
			}
			if _, err := p.captureGoverned(r, scope, a.Proposal.ID, a.Digest, "operator", "normalized", "refused", supplied); err == nil {
				t.Fatal("invalid governed capture")
			}
			if _, err := p.candidates.read("workspace", "refused"); !errors.Is(err, errCandidateMissing) {
				t.Fatal("failed capture reserved a candidate", err)
			}
			if kind == "policy" || kind == "profile" || kind == "limits" || kind == "binding-revision" || kind == "scope" {
				if _, err := p.captureGoverned(r, a.Proposal.Spec.Scope, a.Proposal.ID, a.Digest, "operator", "normalized", "refused", pricingBinding(b)); err != nil {
					t.Fatal("failed attempt reserved candidate ID", err)
				}
			}
		})
	}
}
func TestGovernedPricingDifferentBillingProfilesCompete(t *testing.T) {
	p, _, r, _, a := pricingFixture(t)
	p.domain.Lock()
	profile, _, err := p.orgs.BillingProfile("workspace", "billing")
	if err != nil {
		t.Fatal(err)
	}
	profile.ID = "billing-other"
	if _, err = p.orgs.PutBillingProfile(profile, 0); err != nil {
		t.Fatal(err)
	}
	spec := a.Proposal.Spec
	spec.Scope.BillingProfile = "billing-other"
	b, err := r.Propose("other-profile", "maker", "Same contract alternative profile", spec)
	p.domain.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make([]error, 2)
	for i, record := range []pricingpolicy.Record{a, b} {
		wg.Add(1)
		go func(i int, record pricingpolicy.Record) {
			defer wg.Done()
			p.domain.Lock()
			defer p.domain.Unlock()
			_, results[i] = r.Approve("workspace", record.Proposal.ID, record.Digest, "checker", "Independent synthetic review")
		}(i, record)
	}
	wg.Wait()
	wins := 0
	for _, err := range results {
		if err == nil {
			wins++
		} else if !errors.Is(err, pricingpolicy.ErrConflict) {
			t.Fatal(err)
		}
	}
	if wins != 1 {
		t.Fatal("same economics approved twice", wins)
	}
}
func TestGovernedPricingDoesNotWaiveHeldOrExcludedRows(t *testing.T) {
	for _, held := range []bool{false, true} {
		p, b, r, _, proposal := pricingFixture(t)
		spec := proposal.Proposal.Spec
		spec.Policy.Version = "2"
		if held {
			spec.Policy.Rules = []rating.Rule{{ID: "unmatched", Level: rating.SKU, Key: "missing", Type: rating.PassThrough}}
		} else {
			spec.Policy.Eligible = []rating.Class{rating.ServiceFee}
		}
		p.domain.Lock()
		proposal, err := r.Propose("blocked-price", "maker", "Unsupported partial coverage", spec)
		p.domain.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		a := pricingApprove(t, p, r, proposal)
		if _, err = p.captureGoverned(r, a.Proposal.Spec.Scope, a.Proposal.ID, a.Digest, "operator", "normalized", "blocked", pricingBinding(b)); !errors.Is(err, errCandidateInvalid) {
			t.Fatal("policy approval waived missing priced coverage", err)
		}
		if _, err = p.candidates.read("workspace", "blocked"); !errors.Is(err, errCandidateMissing) {
			t.Fatal("blocked candidate persisted")
		}
	}
}

func TestGovernedPricingRegistryIsBoundToProducer(t *testing.T) {
	p, b, _, _, pending := pricingFixture(t)
	other, _, foreign, _, otherPending := pricingFixture(t)
	pricingApprove(t, other, foreign, otherPending)
	if _, err := p.captureGoverned(foreign, pending.Proposal.Spec.Scope, pending.Proposal.ID, pending.Digest, "operator", "normalized", "foreign-policy", pricingBinding(b)); !errors.Is(err, errCandidateInvalid) {
		t.Fatal("foreign registry accepted", err)
	}
	p.domain.Lock()
	_, err := p.newPricingPolicies(&pricingFixtureAuthority{}, time.Now)
	p.domain.Unlock()
	if !errors.Is(err, errCandidateConflict) {
		t.Fatal("registry replaced", err)
	}
}

func TestGovernedPricingProvenancePreflightIncludesPointers(t *testing.T) {
	p, b, r, _, pending := pricingFixture(t)
	a := pricingApprove(t, p, r, pending)
	made, err := p.captureGoverned(r, a.Proposal.Spec.Scope, a.Proposal.ID, a.Digest, "operator", "normalized", "bounded", pricingBinding(b))
	if err != nil {
		t.Fatal(err)
	}
	if !candidateForm(made.Envelope.Input) {
		t.Fatal("valid governed input rejected")
	}
	for _, value := range []string{string([]byte{0xff}), string(make([]byte, candidateBytes+1))} {
		input := made.Envelope.Input
		record := *input.Normalized.Pricing
		decision := *record.Approval
		decision.Rationale = value
		record.Approval = &decision
		input.Normalized.Pricing = &record
		if candidateForm(input) {
			t.Fatal("pointer-held approval bypassed preflight")
		}
	}
}

func TestGovernedPricingPreflightAggregateAndNilPointers(t *testing.T) {
	p, b, r, _, pending := pricingFixture(t)
	a := pricingApprove(t, p, r, pending)
	made, err := p.captureGoverned(r, a.Proposal.Spec.Scope, a.Proposal.ID, a.Digest, "operator", "normalized", "preflight", pricingBinding(b))
	if err != nil {
		t.Fatal(err)
	}
	original := made.Envelope.Input
	for _, value := range []string{"bad\xff", strings.Repeat("x", candidateBytes+1)} {
		input := original
		record := *input.Normalized.Pricing
		record.Proposal.Spec.Policy.Rules = append([]rating.Rule(nil), record.Proposal.Spec.Policy.Rules...)
		record.Proposal.Spec.Policy.Rules[0].ID = value
		input.Normalized.Pricing = &record
		if candidateForm(input) {
			t.Fatal("pointer-held policy text bypassed preflight")
		}
	}
	input := original
	record := *input.Normalized.Pricing
	decision := *record.Approval
	record.Proposal.Rationale = strings.Repeat("x", candidateBytes/2)
	decision.Rationale = record.Proposal.Rationale
	record.Approval = &decision
	input.Normalized.Pricing = &record
	if candidateForm(input) {
		t.Fatal("aggregate pointer text exceeded budget")
	}
	record = *original.Normalized.Pricing
	record.Approval = nil
	record.Revocation = nil
	input.Normalized.Pricing = &record
	if !candidateForm(input) {
		t.Fatal("nil decisions failed byte preflight")
	}
	input.Normalized.Pricing = nil
	if !candidateForm(input) {
		t.Fatal("nil provenance failed byte preflight")
	}
}
