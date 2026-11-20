package grantauthority

import (
	"sort"
	"sync"
	"time"
)

type entry struct {
	record        Record
	proposerFacts string
	targetFacts   string
}

type Registry struct {
	mu           sync.Mutex
	built        bool
	version      string
	grantPrefix  string
	capabilities map[string]bool
	limits       Limits
	authority    Authority
	targets      Targets
	effects      Effects
	clock        func() time.Time
	now          time.Time
	records      map[[2]string]*entry
}

func NewRegistry(domain Domain, limits Limits, authority Authority, targets Targets, effects Effects, clock func() time.Time) (*Registry, error) {
	if limits.Capacity < 1 || limits.Capacity > 10000 || limits.Lifetime <= 0 || limits.MaxGrantValidity <= 0 || missing(authority) || missing(targets) || missing(effects) || clock == nil {
		return nil, ErrInvalid
	}
	if !validID(domain.Version) || !validID(domain.GrantPrefix) || len(domain.GrantPrefix) > 64 || len(domain.Capabilities) < 1 || len(domain.Capabilities) > 32 {
		return nil, ErrInvalid
	}
	capabilities := make(map[string]bool, len(domain.Capabilities))
	for _, c := range domain.Capabilities {
		if !validID(c) || capabilities[c] {
			return nil, ErrInvalid
		}
		capabilities[c] = true
	}
	return &Registry{built: true, version: domain.Version, grantPrefix: domain.GrantPrefix, capabilities: capabilities, limits: limits, authority: authority, targets: targets, effects: effects, clock: clock, records: make(map[[2]string]*entry)}, nil
}

func (r *Registry) instant() (time.Time, error) {
	at := canonical(r.clock())
	if !validTime(at) {
		return time.Time{}, ErrInvalid
	}
	if at.After(r.now) {
		r.now = at
	}
	return r.now, nil
}

func (r *Registry) authorized(workspace, actor string) (string, error) {
	facts, err := r.authority.Check(workspace, actor)
	if err != nil {
		return "", err
	}
	if !validText(facts, 4096) {
		return "", ErrInvalid
	}
	return facts, nil
}

func (r *Registry) target(spec Spec) (string, error) {
	facts, err := r.targets.Current(spec)
	if err != nil {
		return "", err
	}
	if !validText(facts, 4096) {
		return "", ErrInvalid
	}
	return facts, nil
}

func (r *Registry) Propose(p Proposal) (Record, error) {
	if r == nil || !r.built {
		return Record{}, ErrInvalid
	}
	p.Spec = normalizeSpec(p.Spec)
	if !validID(p.ID) || !validID(p.Proposer) || !validText(p.Rationale, 1024) || !r.validSpec(p.Spec, r.limits.MaxGrantValidity) {
		return Record{}, ErrInvalid
	}
	if p.Spec.Kind == Human && p.Spec.Recipient == p.Proposer {
		return Record{}, ErrDenied
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	pf, err := r.authorized(p.Spec.Workspace, p.Proposer)
	if err != nil {
		return Record{}, err
	}
	tf, err := r.target(p.Spec)
	if err != nil {
		return Record{}, err
	}
	if _, err := r.instant(); err != nil {
		return Record{}, err
	}
	if prior := r.records[[2]string{p.Spec.Workspace, p.ID}]; prior != nil {
		if prior.record.Proposal != p || prior.proposerFacts != pf || prior.targetFacts != tf {
			return Record{}, ErrConflict
		}
		return r.shown(prior), nil
	}
	at, err := r.instant()
	if err != nil {
		return Record{}, err
	}
	if !at.Before(p.Spec.NotAfter) {
		return Record{}, ErrConflict
	}
	if len(r.records) >= r.limits.Capacity {
		return Record{}, ErrLimit
	}
	expires := at.Add(r.limits.Lifetime)
	if expires.After(p.Spec.NotAfter) {
		expires = p.Spec.NotAfter
	}
	if !validTime(expires) {
		return Record{}, ErrInvalid
	}
	digest := hash(struct {
		Version                    string
		Proposal                   Proposal
		ProposerFacts, TargetFacts string
		At, Expires                time.Time
	}{r.version, p, pf, tf, at, expires})
	record := Record{Proposal: p, Digest: digest, State: Pending, ProposedAt: at, ExpiresAt: expires}
	r.records[[2]string{p.Spec.Workspace, p.ID}] = &entry{record: record, proposerFacts: pf, targetFacts: tf}
	return clone(record), nil
}

func (r *Registry) shown(q *entry) Record {
	record := clone(q.record)
	if record.State == Pending && !r.now.Before(record.ExpiresAt) {
		record.State = Expired
	}
	return record
}

func (r *Registry) Record(workspace, id string) (Record, error) {
	if r == nil || !r.built || !validID(workspace) || !validID(id) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.instant(); err != nil {
		return Record{}, err
	}
	q := r.records[[2]string{workspace, id}]
	if q == nil {
		return Record{}, ErrNotFound
	}
	return r.shown(q), nil
}

func (r *Registry) current(q *entry) (bool, error) {
	pf, err := r.authorized(q.record.Proposal.Spec.Workspace, q.record.Proposal.Proposer)
	if err != nil {
		return false, err
	}
	tf, err := r.target(q.record.Proposal.Spec)
	if err != nil {
		return false, err
	}
	return pf == q.proposerFacts && tf == q.targetFacts, nil
}

func (r *Registry) Approve(i Intent) (Record, error) {
	if r == nil || !r.built || !validIntent(i) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.authorized(i.Workspace, i.Actor); err != nil {
		return Record{}, err
	}
	q := r.records[[2]string{i.Workspace, i.ID}]
	if q == nil {
		return Record{}, ErrNotFound
	}
	p := q.record.Proposal
	if i.Actor == p.Proposer || (p.Spec.Kind == Human && i.Actor == p.Spec.Recipient) {
		return Record{}, ErrDenied
	}
	if q.record.Digest != i.Digest {
		return Record{}, ErrConflict
	}
	at, err := r.instant()
	if err != nil {
		return Record{}, err
	}
	retry := q.record.State == Unknown || q.record.State == Applied
	if retry {
		if !sameDecision(q.record, i) {
			return Record{}, ErrConflict
		}
		if _, err := r.authorized(i.Workspace, p.Proposer); err != nil {
			return Record{}, err
		}
		effect, err := r.effects.Lookup(q.record.Decision.GrantID)
		if err != nil {
			return Record{}, ErrUnknown
		}
		if effect.Found {
			if !exactEffect(effect, p.Spec) {
				return Record{}, ErrUnknown
			}
			if q.record.State == Applied {
				return clone(q.record), nil
			}
			q.record.State = Applied
			q.record.Decision.GrantStatus = effect.Status
			return clone(q.record), nil
		}
		if q.record.State == Applied {
			return Record{}, ErrUnknown
		}
	} else {
		if q.record.State != Pending {
			return Record{}, ErrConflict
		}
		if !at.Before(q.record.ExpiresAt) {
			q.record.State = Expired
			return clone(q.record), ErrConflict
		}
	}
	if at.Before(p.Spec.NotBefore) {
		return Record{}, ErrConflict
	}
	current, err := r.current(q)
	if err != nil {
		return Record{}, err
	}
	if !current || !at.Before(p.Spec.NotAfter) {
		if !retry {
			q.record.State = Superseded
			return clone(q.record), ErrConflict
		}
		return Record{}, ErrConflict
	}
	grantID := r.grantPrefix + q.record.Digest
	if !retry {
		effect, err := r.effects.Lookup(grantID)
		if err != nil {
			return Record{}, err
		}
		if effect.Found {
			return Record{}, ErrConflict
		}
		q.record.Decision = &Decision{Actor: i.Actor, Rationale: i.Rationale, At: at, GrantID: grantID}
	}
	q.record.State = Applying
	if err := r.effects.Apply(grantID, p.Spec); err != nil {
		q.record.State = Unknown
		return Record{}, ErrUnknown
	}
	effect, err := r.effects.Lookup(grantID)
	if err != nil || !exactEffect(effect, p.Spec) {
		q.record.State = Unknown
		return Record{}, ErrUnknown
	}
	q.record.State = Applied
	q.record.Decision.GrantStatus = effect.Status
	return clone(q.record), nil
}

func (r *Registry) finish(i Intent, withdraw bool) (Record, error) {
	if r == nil || !r.built || !validIntent(i) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.authorized(i.Workspace, i.Actor); err != nil {
		return Record{}, err
	}
	q := r.records[[2]string{i.Workspace, i.ID}]
	if q == nil {
		return Record{}, ErrNotFound
	}
	p := q.record.Proposal
	if (withdraw && i.Actor != p.Proposer) || (!withdraw && (i.Actor == p.Proposer || p.Spec.Kind == Human && i.Actor == p.Spec.Recipient)) {
		return Record{}, ErrDenied
	}
	if i.Digest != q.record.Digest || q.record.State != Pending {
		return Record{}, ErrConflict
	}
	at, err := r.instant()
	if err != nil {
		return Record{}, err
	}
	if !at.Before(q.record.ExpiresAt) {
		q.record.State = Expired
		return clone(q.record), ErrConflict
	}
	state := Rejected
	if withdraw {
		state = Withdrawn
	}
	q.record.State = state
	q.record.Decision = &Decision{Actor: i.Actor, Rationale: i.Rationale, At: at}
	return clone(q.record), nil
}

func (r *Registry) Reject(i Intent) (Record, error)   { return r.finish(i, false) }
func (r *Registry) Withdraw(i Intent) (Record, error) { return r.finish(i, true) }

func (r *Registry) Records(workspace string) ([]Record, error) {
	if r == nil || !r.built || !validID(workspace) {
		return nil, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.instant(); err != nil {
		return nil, err
	}
	out := []Record{}
	for key, q := range r.records {
		if key[0] == workspace {
			out = append(out, r.shown(q))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Proposal.ID < out[j].Proposal.ID })
	return out, nil
}
