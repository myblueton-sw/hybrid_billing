package marketplace

import (
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type revision struct {
	number    int
	facts     Facts
	state     string
	reasons   []string
	digest    string
	writtenAt time.Time
	judgedAt  time.Time
}

type statement struct {
	revisions []*revision
}

func (s *statement) highest() *revision { return s.revisions[len(s.revisions)-1] }

func (s *statement) lastSealed() *revision {
	for index := len(s.revisions) - 1; index >= 0; index-- {
		if s.revisions[index].state == StateSealed {
			return s.revisions[index]
		}
	}
	return nil
}

type settlementRevision struct {
	facts     SettlementFacts
	judged    string
	reasons   []string
	digest    string
	writtenAt time.Time
}

type settlement struct {
	revisions []*settlementRevision
}

type stored struct {
	t           Transaction
	statements  map[string]*statement
	settlements map[string]*settlement
}

type policyKey struct{ id, version string }

type storedPolicy struct {
	terms      PolicyTerms
	author     string
	approver   string
	last       string
	digest     string
	writtenAt  time.Time
	approvedAt time.Time
}

type space struct {
	transactions map[string]*stored
	policies     map[policyKey]*storedPolicy
}

type Registry struct {
	mu         sync.Mutex
	profile    decimal.Profile
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(profile decimal.Profile, limits Limits, clock func() time.Time) (*Registry, error) {
	if !profile.Valid() || !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{profile: profile, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (r *Registry) enter() time.Time {
	r.mu.Lock()
	now := r.clock().UTC().Round(0)
	if now.After(r.now) {
		r.now = now
	}
	return r.now
}

func (r *Registry) leave() { r.mu.Unlock() }

func (r *Registry) amount(text string) (string, bool) {
	if len(text) > maxNumberBytes {
		return "", false
	}
	value, err := decimal.Parse(r.profile, text)
	if err != nil || value.Sign() < 0 {
		return "", false
	}
	return value.String(), true
}

func (r *Registry) space(workspace string, create bool) *space {
	s := r.workspaces[workspace]
	if s == nil && create {
		s = &space{transactions: map[string]*stored{}, policies: map[policyKey]*storedPolicy{}}
		r.workspaces[workspace] = s
	}
	return s
}

func transactionForm(t Transaction) bool {
	p, ref, term := t.Participants, t.References, t.Term
	if !validID(t.ID) || !validID(t.Provider) || !validID(t.Agreement) || !roles[t.Role] || !channels[t.Channel] || !types[t.Type] || !collectors[t.Collector] {
		return false
	}
	if !validID(p.SoftwareProvider) || !validID(p.Seller) || !validID(p.Issuer) || !optionalID(p.Reseller) || !optionalID(p.Payee) {
		return false
	}
	if (t.Role == RoleBuyer) != (p.Payee == "") || (t.Role == RoleReseller) != (p.Reseller != "") {
		return false
	}
	if !validID(ref.Product) || !validID(ref.Offer) || !optionalID(ref.Subscription) || !optionalID(ref.Order) {
		return false
	}
	if !validMonth(term.Accepted) || !validMonth(term.Start) || !validMonth(term.End) || term.Start < term.Accepted || term.End < term.Start || !validCurrency(term.Currency) {
		return false
	}
	if t.Type == TypeChannelPrivate && (t.Role != RoleReseller || t.Channel == ChannelDirect) {
		return false
	}
	if t.Role == RoleBuyer && t.Collector != CollectorMSP {
		return false
	}
	if t.Collector == CollectorDistributor && t.Channel != ChannelDistributor || t.Collector == CollectorMarketplace && t.Channel != ChannelMarketplace {
		return false
	}
	return true
}

func (r *Registry) RecordTransaction(workspace string, t Transaction) (Transaction, error) {
	if !validID(workspace) || !transactionForm(t) {
		return Transaction{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.space(workspace, false)
	if s != nil && s.transactions[t.ID] != nil {
		if s.transactions[t.ID].t != t {
			return Transaction{}, ErrConflict
		}
		return t, nil
	}
	if s != nil && len(s.transactions) >= r.limits.Transactions {
		return Transaction{}, ErrLimit
	}
	s = r.space(workspace, true)
	s.transactions[t.ID] = &stored{t: t, statements: map[string]*statement{}, settlements: map[string]*settlement{}}
	return t, nil
}

func (r *Registry) find(workspace, id string) (*stored, error) {
	s := r.workspaces[workspace]
	if s == nil || s.transactions[id] == nil {
		return nil, ErrNotFound
	}
	return s.transactions[id], nil
}

func (r *Registry) Transaction(workspace, id string) (Transaction, error) {
	if !validID(workspace) || !validID(id) {
		return Transaction{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	st, err := r.find(workspace, id)
	if err != nil {
		return Transaction{}, err
	}
	return st.t, nil
}

func (r *Registry) sum(texts ...string) exact.Fraction {
	total := exact.Whole(0)
	for _, text := range texts {
		value, _ := decimal.Parse(r.profile, text)
		made, _ := exact.Of(value)
		total = total.Plus(made)
	}
	return total
}

func same(a, b exact.Fraction) bool { return !a.Less(b) && !b.Less(a) }
