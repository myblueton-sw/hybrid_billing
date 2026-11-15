package support

import (
	"net/netip"
	"sync"
	"time"
)

const (
	maxSites         = 100000
	maxDestinations  = 64
	maxBundles       = 100
	maxSiteBytes     = 64 << 20
	maxBytes         = 4 << 30
	minBundleLife    = time.Hour
	maxBundleLife    = 30 * 24 * time.Hour
	maxProposals     = 1000
	maxSchemaPending = 16
	maxHostBytes     = 253
	maxLabel         = 63
	registerScope    = ""
)

type bundle struct {
	view      BundleView
	sources   map[string]bool
	previewed map[string]bool
	uploads   []UploadRecord
}

type site struct {
	view        Site
	handle      string
	counter     uint64
	destination string
	bundles     map[string]*bundle
	bytes       int64
}

type destination struct {
	view     Destination
	disabled bool
}

type proposal struct {
	view  ProposalView
	scope string
	code  string
}

type scope struct {
	pending []uint64
	done    []uint64
}

type Registry struct {
	mu           sync.Mutex
	limits       Limits
	clock        func() time.Time
	now          time.Time
	counter      uint64
	handles      uint64
	bytes        int64
	sites        map[string]*site
	schemas      map[string]map[string]FieldSpec
	destinations map[string]*destination
	proposals    map[uint64]*proposal
	scopes       map[string]*scope
}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{limits: limits, clock: clock, sites: map[string]*site{}, schemas: map[string]map[string]FieldSpec{}, destinations: map[string]*destination{},
		proposals: map[uint64]*proposal{}, scopes: map[string]*scope{}}, nil
}

func (l Limits) valid() bool {
	pairs := [][2]int64{{int64(l.Sites), maxSites}, {int64(l.Destinations), maxDestinations}, {int64(l.Bundles), maxBundles}, {l.SiteBytes, maxSiteBytes},
		{l.Bytes, maxBytes}, {int64(l.Proposals), maxProposals}}
	for _, p := range pairs {
		if p[0] < 1 || p[0] > p[1] {
			return false
		}
	}
	return l.SiteBytes <= l.Bytes && l.BundleLife >= minBundleLife && l.BundleLife <= maxBundleLife
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

func refuse(reason string) error { return Refusal{Reason: reason} }

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) human() error {
	switch {
	case !a.valid():
		return ErrInvalid
	case a.Kind == KindAssistant:
		return refuse(ReasonAINotAllowed)
	case a.Kind == KindSystem:
		return refuse(ReasonNotHuman)
	}
	return nil
}

func (r *Registry) scope(name string) *scope {
	s := r.scopes[name]
	if s == nil {
		s = &scope{}
		r.scopes[name] = s
	}
	return s
}

func drop(list []uint64, number uint64) []uint64 {
	out := list[:0]
	for _, value := range list {
		if value != number {
			out = append(out, value)
		}
	}
	return out
}

func (r *Registry) Propose(c Change, actor Actor) (ProposalView, error) {
	now := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return ProposalView{}, err
	}
	c, name, ok := clean(c)
	if !ok {
		return ProposalView{}, ErrInvalid
	}
	if name != registerScope && r.sites[name] == nil {
		return ProposalView{}, ErrNotFound
	}
	code := string(encodeChange(c))
	s := r.scope(name)
	for _, number := range s.pending {
		if p := r.proposals[number]; p.code == code && p.view.Proposer == actor.ID {
			return copyProposal(p.view), nil
		}
	}
	if err := r.check(c, now); err != nil {
		return ProposalView{}, err
	}
	mine := 0
	for _, number := range s.pending {
		if r.proposals[number].view.Proposer == actor.ID {
			mine++
		}
	}
	schemas := 0
	for _, number := range s.pending {
		if r.proposals[number].view.Change.Kind == ChangeSchema {
			schemas++
		}
	}
	if len(s.pending) >= r.limits.Proposals || mine >= (r.limits.Proposals+3)/4 || c.Kind == ChangeSchema && schemas >= maxSchemaPending {
		return ProposalView{}, ErrLimit
	}
	r.counter++
	p := &proposal{view: ProposalView{Number: r.counter, Change: copyChange(c), State: ProposalPending, Proposer: actor.ID}, scope: name, code: code}
	r.proposals[p.view.Number] = p
	s.pending = append(s.pending, p.view.Number)
	return copyProposal(p.view), nil
}

func (r *Registry) Approve(number uint64, actor Actor) (ProposalView, error) {
	now := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return ProposalView{}, err
	}
	p := r.proposals[number]
	if p == nil {
		return ProposalView{}, ErrNotFound
	}
	switch {
	case p.view.State == ProposalApplied && p.view.Approver == actor.ID:
		return copyProposal(p.view), nil
	case p.view.State != ProposalPending:
		return ProposalView{}, refuse(ReasonNotPending)
	case p.view.Proposer == actor.ID:
		return ProposalView{}, refuse(ReasonSelfApproval)
	}
	if err := r.check(p.view.Change, now); err != nil {
		return ProposalView{}, err
	}
	r.apply(p.view.Change, now)
	p.view.State, p.view.Approver = ProposalApplied, actor.ID
	s := r.scopes[p.scope]
	s.pending = drop(s.pending, number)
	s.done = append(s.done, number)
	if len(s.done) > r.limits.Proposals {
		delete(r.proposals, s.done[0])
		s.done = s.done[1:]
	}
	return copyProposal(p.view), nil
}

func (r *Registry) Withdraw(number uint64, actor Actor) (ProposalView, error) {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return ProposalView{}, err
	}
	p := r.proposals[number]
	switch {
	case p == nil:
		return ProposalView{}, ErrNotFound
	case p.view.Proposer != actor.ID:
		return ProposalView{}, refuse(ReasonNotProposer)
	case p.view.State != ProposalPending:
		return ProposalView{}, refuse(ReasonNotPending)
	}
	delete(r.proposals, number)
	s := r.scopes[p.scope]
	s.pending = drop(s.pending, number)
	p.view.State = ProposalWithdrawn
	return copyProposal(p.view), nil
}

func (r *Registry) Proposal(number uint64) (ProposalView, error) {
	r.enter()
	defer r.leave()
	p := r.proposals[number]
	if p == nil {
		return ProposalView{}, ErrNotFound
	}
	return copyProposal(p.view), nil
}

func validHost(host string) bool {
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Zone() == "" && !addr.Is4In6() && addr.String() == host
	}
	if host == "" || len(host) > maxHostBytes {
		return false
	}
	label, letter := 0, false
	for index := 0; index < len(host); index++ {
		c := host[index]
		switch {
		case c == '.':
			if label == 0 || host[index-1] == '-' {
				return false
			}
			label, letter = 0, false
		case c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-':
			if c == '-' && label == 0 || label == maxLabel {
				return false
			}
			letter = letter || c >= 'a' && c <= 'z'
			label++
		default:
			return false
		}
	}
	return letter && host[len(host)-1] != '-'
}
