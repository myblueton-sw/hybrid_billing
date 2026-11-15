package recovery

import (
	"sync"
	"time"
)

const (
	maxInstallations = 100000
	maxWriters       = 16
	maxExternals     = 16
	maxLive          = 1000
	maxOpenPerWriter = 1000
	maxOpen          = 10000
	maxClosed        = 10000
	maxChunks        = 10000
	maxItems         = 10000
	maxPages         = 1000
	maxSeries        = 1000
	maxKeys          = 1000000
	maxKeysTotal     = 10000000
	maxEvents        = 1000000
	maxEventsTotal   = 10000000
	maxHistory       = 100
	maxProposals     = 1000
	maxBegins        = 1000
	maxSkew          = time.Hour
	minMaxCall       = time.Second
	maxMaxCall       = time.Hour
	maxMismatch      = 100
	registerScope    = ""
)

type writer struct {
	view    Writer
	removed bool
	live    []uint64
	open    int
}

type permit struct {
	writer     string
	generation uint64
	expiry     time.Time
}

type call struct {
	view   CallView
	at     time.Time
	closed time.Time
}

type event struct {
	view EventView
}

type pages struct {
	next    int
	digests []string
	from    time.Time
	through time.Time
	final   bool
}

type recovery struct {
	id         string
	point      *RestorePoint
	digest     string
	beganSeq   uint64
	domains    map[string]*DomainStatus
	externals  map[string]*pages
	applied    map[uint64]bool
	lost       []uint64
	reapplying bool
	unknown    []Target
	listed     map[string]bool
	skipped    int
}

type site struct {
	view       Installation
	generation uint64
	state      string
	stoppedAt  time.Time
	inflight   time.Time
	stopCall   time.Duration
	writers    map[string]*writer
	externals  map[string]External
	verifier   string
	sequence   uint64
	tokens     map[uint64]*permit
	calls      uint64
	open       map[string]*call
	closed     []*call
	closedByID map[string]*call
	high       map[[2]string]int64
	fenced     map[[2]string]bool
	events     []*event
	eventSeq   uint64
	pruned     time.Time
	current    *recovery
	recoveries int
	history    []History
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
	mu            sync.Mutex
	clock         func() time.Time
	now           time.Time
	counter       uint64
	begins        int
	keys          int
	eventsTotal   int
	installations map[string]*site
	identities    map[string]string
	proposals     map[uint64]*proposal
	scopes        map[string]*scope
}

func NewRegistry(clock func() time.Time) (*Registry, error) {
	if clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{clock: clock, installations: map[string]*site{}, identities: map[string]string{}, proposals: map[uint64]*proposal{}, scopes: map[string]*scope{}}, nil
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
	if name != registerScope && r.installations[name] == nil {
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
	if len(s.pending) >= maxProposals || mine >= (maxProposals+3)/4 || c.Kind == ChangeBegin && (r.pendingBegins(s) > 0 || r.begins >= maxBegins) {
		return ProposalView{}, ErrLimit
	}
	if c.Kind == ChangeBegin {
		r.begins++
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
	r.forgetBegin(p)
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
	r.forgetBegin(p)
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

func (r *Registry) pendingBegins(s *scope) int {
	count := 0
	for _, number := range s.pending {
		if r.proposals[number].view.Change.Kind == ChangeBegin {
			count++
		}
	}
	return count
}

func (r *Registry) forgetBegin(p *proposal) {
	if p.view.Change.Kind == ChangeBegin {
		r.begins--
	}
}

func (r *Registry) retireBegins(id string) {
	s := r.scopes[id]
	if s == nil {
		return
	}
	for _, number := range append([]uint64(nil), s.pending...) {
		p := r.proposals[number]
		if p.view.Change.Kind == ChangeBegin {
			delete(r.proposals, number)
			s.pending = drop(s.pending, number)
			r.forgetBegin(p)
		}
	}
}
