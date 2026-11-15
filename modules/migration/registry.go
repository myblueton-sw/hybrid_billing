package migration

import (
	"sync"
	"time"
)

const (
	maxSources       = 10000
	maxWorkspaces    = 64
	maxItems         = 100000
	maxBatches       = 1000
	maxHistory       = 1000000
	maxHistoryTotal  = 10000000
	maxQuarantine    = 100000
	maxSeries        = 10000
	maxSeriesTotal   = 100000
	maxPrinted       = 1000000
	maxPrintedTotal  = 10000000
	maxDelivery      = 2000000
	maxDeliveryTotal = 20000000
	maxKeyRefs       = 10000
	maxProposals     = 1000
	maxNumber        = 1000000000000000
	maxDER           = 16 << 10
	registerScope    = ""
)

type batch struct {
	spec       BatchSpec
	state      string
	items      map[string]Item
	codes      map[string]string
	computed   string
	summary    map[string]int
	effect     string
	rejectedBy string
	reason     string
}

type entry struct {
	code     string
	document *Document
	key      *KeyView
}

type source struct {
	view       Source
	mappings   map[[2]string]string
	batches    map[string]*batch
	order      []string
	open       string
	history    map[string]*entry
	quarantine []Classified
	seen       map[string]bool
}

type tenant struct {
	history  int
	keys     int
	high     map[[2]string]int64
	printed  map[[2]string]bool
	delivery map[[2]string]bool
	numbers  map[[3]string]string
	displays map[[2]string]string
	external map[[2]string]string
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
	mu         sync.Mutex
	config     Config
	clock      func() time.Time
	now        time.Time
	counter    uint64
	sources    map[string]*source
	tenants    map[string]*tenant
	claims     map[string]string
	identities map[string]string
	history    int
	series     int
	printed    int
	delivery   int
	proposals  map[uint64]*proposal
	scopes     map[string]*scope
}

func NewRegistry(config Config, clock func() time.Time) (*Registry, error) {
	if config.Evidence == nil || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{config: config, clock: clock, sources: map[string]*source{}, tenants: map[string]*tenant{}, claims: map[string]string{}, identities: map[string]string{},
		proposals: map[uint64]*proposal{}, scopes: map[string]*scope{}}, nil
}

func (r *Registry) tenant(id string) *tenant {
	t := r.tenants[id]
	if t == nil {
		t = &tenant{high: map[[2]string]int64{}, printed: map[[2]string]bool{}, delivery: map[[2]string]bool{}, numbers: map[[3]string]string{},
			displays: map[[2]string]string{}, external: map[[2]string]string{}}
		r.tenants[id] = t
	}
	return t
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
	if name != registerScope && r.sources[name] == nil {
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
	if len(s.pending) >= maxProposals || mine >= (maxProposals+3)/4 {
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
	p.view.State, p.view.Approver = ProposalApplied, actor.ID
	s := r.scopes[p.scope]
	s.pending = drop(s.pending, number)
	r.apply(p.view.Change, now)
	s.done = append(s.done, number)
	if len(s.done) > maxProposals {
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
