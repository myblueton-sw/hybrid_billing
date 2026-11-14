package marketplace

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

const maxRateScale = 8

type PolicyState struct {
	Terms         PolicyTerms
	EffectiveLast string
	Author        string
	Approver      string
	Digest        string
	WrittenAt     time.Time
	ApprovedAt    time.Time
}

type Policy struct {
	made      bool
	workspace string
	terms     PolicyTerms
	rate      exact.Fraction
	scope     map[string]bool
	scale     int
	digest    string
}

func (r *Registry) policyForm(terms PolicyTerms) (PolicyTerms, bool) {
	out := terms
	rate, ok := r.amount(terms.Rate)
	_, known := quanta[terms.Quantum]
	if !validID(terms.ID) || !validID(terms.Version) || !validID(terms.Customer) || !validID(terms.Transaction) || !ok || !known || len(terms.Scope) == 0 {
		return PolicyTerms{}, false
	}
	if !validMonth(terms.First) || !validMonth(terms.Last) || terms.Last < terms.First {
		return PolicyTerms{}, false
	}
	value, _ := decimal.Parse(r.profile, rate)
	if value.Scale() > maxRateScale || !r.sum(rate).Less(exact.Whole(1)) {
		return PolicyTerms{}, false
	}
	discounted := value.Sign() > 0
	switch terms.SourceDiscounted {
	case Excluded:
		if terms.SourceBasis != "" {
			return PolicyTerms{}, false
		}
	case Included:
		if !discounted || !validDigest(terms.SourceBasis) {
			return PolicyTerms{}, false
		}
	default:
		return PolicyTerms{}, false
	}
	if discounted != validDigest(terms.Evidence) || !discounted && terms.Evidence != "" {
		return PolicyTerms{}, false
	}
	scope := append([]string{}, terms.Scope...)
	sort.Strings(scope)
	for index, kind := range scope {
		if !lineKinds[kind] || index > 0 && scope[index-1] == kind {
			return PolicyTerms{}, false
		}
	}
	out.Rate, out.Scope = rate, scope
	return out, true
}

func samePolicy(a, b PolicyTerms) bool {
	if a.ID != b.ID || a.Version != b.Version || a.Customer != b.Customer || a.Transaction != b.Transaction || a.First != b.First || a.Last != b.Last ||
		a.Rate != b.Rate || a.SourceDiscounted != b.SourceDiscounted || a.Evidence != b.Evidence || a.SourceBasis != b.SourceBasis || a.Quantum != b.Quantum ||
		len(a.Scope) != len(b.Scope) {
		return false
	}
	for index := range a.Scope {
		if a.Scope[index] != b.Scope[index] {
			return false
		}
	}
	return true
}

func viewPolicy(p *storedPolicy) PolicyState {
	terms := p.terms
	terms.Scope = append([]string{}, p.terms.Scope...)
	return PolicyState{Terms: terms, EffectiveLast: p.last, Author: p.author, Approver: p.approver, Digest: p.digest, WrittenAt: p.writtenAt, ApprovedAt: p.approvedAt}
}

func (r *Registry) RecordPolicy(workspace string, terms PolicyTerms, author string) (PolicyState, error) {
	canonical, formed := r.policyForm(terms)
	if !validID(workspace) || !validID(author) || !formed {
		return PolicyState{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.space(workspace, false)
	if s == nil || s.transactions[canonical.Transaction] == nil {
		return PolicyState{}, ErrNotFound
	}
	key := policyKey{canonical.ID, canonical.Version}
	if existing := s.policies[key]; existing != nil {
		if !samePolicy(existing.terms, canonical) || existing.author != author {
			return PolicyState{}, ErrConflict
		}
		return viewPolicy(existing), nil
	}
	if t := s.transactions[canonical.Transaction].t; canonical.First < t.Term.Start || canonical.Last > t.Term.End {
		return PolicyState{}, ErrRejected
	}
	if len(s.policies) >= r.limits.Policies {
		return PolicyState{}, ErrLimit
	}
	stored := &storedPolicy{terms: canonical, last: canonical.Last, author: author, digest: policyDigest(workspace, canonical), writtenAt: now}
	s.policies[key] = stored
	return viewPolicy(stored), nil
}

func (r *Registry) ApprovePolicy(workspace, id, version, approver string) (PolicyState, error) {
	if !validID(workspace) || !validID(id) || !validID(version) || !validID(approver) {
		return PolicyState{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.space(workspace, false)
	if s == nil || s.policies[policyKey{id, version}] == nil {
		return PolicyState{}, ErrNotFound
	}
	p := s.policies[policyKey{id, version}]
	if p.approver != "" {
		if p.approver != approver {
			return PolicyState{}, ErrConflict
		}
		return viewPolicy(p), nil
	}
	if approver == p.author {
		return PolicyState{}, ErrRejected
	}
	superseded := []*storedPolicy{}
	for _, other := range s.policies {
		if other.approver == "" || other.terms.Transaction != p.terms.Transaction || other.terms.First > p.terms.Last || p.terms.First > other.last {
			continue
		}
		if other.terms.ID != p.terms.ID || other.terms.First >= p.terms.First {
			return PolicyState{}, ErrRejected
		}
		superseded = append(superseded, other)
	}
	for _, other := range superseded {
		other.last = monthBefore(p.terms.First)
	}
	p.approver, p.approvedAt = approver, now
	return viewPolicy(p), nil
}

func monthBefore(month string) string {
	year, _ := strconv.Atoi(month[:4])
	number, _ := strconv.Atoi(month[5:])
	if number == 1 {
		return fmt.Sprintf("%04d-12", year-1)
	}
	return fmt.Sprintf("%04d-%02d", year, number-1)
}

func (r *Registry) PolicyFor(workspace, transaction, month string) (Policy, error) {
	if !validID(workspace) || !validID(transaction) || !validMonth(month) {
		return Policy{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.space(workspace, false)
	if s == nil || s.transactions[transaction] == nil {
		return Policy{}, ErrNotFound
	}
	for _, p := range s.policies {
		if p.approver == "" || p.terms.Transaction != transaction || month < p.terms.First || month > p.last {
			continue
		}
		terms := p.terms
		terms.Scope, terms.Last = append([]string{}, p.terms.Scope...), p.last
		scope := map[string]bool{}
		for _, kind := range terms.Scope {
			scope[kind] = true
		}
		return Policy{made: true, workspace: workspace, terms: terms, rate: r.sum(terms.Rate), scope: scope, scale: quanta[terms.Quantum], digest: p.digest}, nil
	}
	return Policy{}, ErrNotFound
}

func (r *Registry) PolicyState(workspace, id, version string) (PolicyState, error) {
	if !validID(workspace) || !validID(id) || !validID(version) {
		return PolicyState{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.space(workspace, false)
	if s == nil || s.policies[policyKey{id, version}] == nil {
		return PolicyState{}, ErrNotFound
	}
	return viewPolicy(s.policies[policyKey{id, version}]), nil
}
