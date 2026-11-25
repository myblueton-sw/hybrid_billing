package retention

import (
	"sync"
	"time"
)

const (
	maxArtifacts     = 10000000
	maxReferences    = 10000000
	maxTenants       = 64
	maxStores        = 16
	maxHolds         = 10000
	maxHoldArtifacts = 10000
	maxUnfinished    = 100
	maxRunArtifacts  = 100000
	maxChunk         = 1000
	maxPage          = 10000
	maxOutcomePage   = 1000
	maxTombstonePage = 1000
	maxPending       = 1000
	maxMine          = 250
	maxDone          = 1000
	maxPeriodDays    = 36500
	maxGraceDays     = 365
	maxLease         = 24 * time.Hour
	minLease         = time.Minute
	impactWindow     = time.Hour
	day              = 24 * time.Hour
)

type people struct {
	proposer string
	approver string
}

type artifact struct {
	view    Artifact
	deleted bool
	damaged bool
	stores  map[string]string
	storeAt map[string]time.Time
	run     string
}

type hold struct {
	view     Hold
	placer   string
	released bool
}

type chunk struct {
	token    uint64
	tokens   map[uint64]bool
	until    time.Time
	claimed  bool
	doneOnce bool
}

type run struct {
	id       string
	plan     DeletionPlan
	revision uint64
	outcomes map[string]string
	chunks   []chunk
	stopped  bool
	stopper  string
	by       people
}

type proposal struct {
	view ProposalView
	code string
}

type Registry struct {
	mu         sync.Mutex
	clock      func() time.Time
	now        time.Time
	counter    uint64
	tokens     uint64
	artifacts  map[string]*artifact
	byClass    map[string]map[string]bool
	refs       map[[3]string]bool
	requiredBy map[string]int
	requires   map[string][]string
	guards     map[string][]string
	holds      map[string]*hold
	live       map[string]*hold
	policies   map[string][]PolicyView
	runs       map[string]*run
	runSeq     uint64
	unfinished int
	tombstones []Tombstone
	backups    map[string]time.Time
	proposals  map[uint64]*proposal
	pending    []uint64
	done       []uint64
}

func NewRegistry(clock func() time.Time) (*Registry, error) {
	if clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{clock: clock, artifacts: map[string]*artifact{}, byClass: map[string]map[string]bool{}, refs: map[[3]string]bool{}, requiredBy: map[string]int{},
		requires: map[string][]string{}, guards: map[string][]string{}, holds: map[string]*hold{}, live: map[string]*hold{}, policies: map[string][]PolicyView{}, runs: map[string]*run{}, backups: map[string]time.Time{},
		proposals: map[uint64]*proposal{}}, nil
}

func (r *Registry) enter() time.Time {
	now := r.clock().UTC().Round(0)
	r.mu.Lock()
	if now.After(r.now) {
		r.now = now
	}
	return r.now
}

func (r *Registry) leave() { r.mu.Unlock() }

func refuse(reason string) error { return Refusal{Reason: reason} }

func (a Actor) valid() bool {
	return validID(a.ID) && oneOf(a.Kind, KindHuman, KindSystem, KindAssistant)
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

func (a Actor) system() error {
	switch {
	case !a.valid():
		return ErrInvalid
	case a.Kind == KindAssistant:
		return refuse(ReasonAINotAllowed)
	case a.Kind == KindHuman:
		return refuse(ReasonNotSystem)
	}
	return nil
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
	c, err := clean(c)
	if err != nil {
		return ProposalView{}, err
	}
	code := encodeChange(c)
	mine := 0
	for _, number := range r.pending {
		p := r.proposals[number]
		if p.view.Proposer != actor.ID {
			continue
		}
		if p.code == code {
			return copyProposal(p.view), nil
		}
		mine++
	}
	if err := r.check(c, actor.ID, "", Impact{}, now); err != nil {
		return ProposalView{}, err
	}
	if len(r.pending) >= maxPending || mine >= maxMine {
		return ProposalView{}, ErrLimit
	}
	if r.shortening(c) {
		c.Impact = r.impact(c.Policy, now).Impact
	}
	r.counter++
	p := &proposal{view: ProposalView{Number: r.counter, Change: copyChange(c), State: ProposalPending, Proposer: actor.ID}, code: code}
	r.proposals[p.view.Number] = p
	r.pending = append(r.pending, p.view.Number)
	return copyProposal(p.view), nil
}

func (r *Registry) Approve(number uint64, actor Actor, reviewed Impact) (ProposalView, error) {
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
	if err := r.check(p.view.Change, p.view.Proposer, actor.ID, reviewed, now); err != nil {
		return ProposalView{}, err
	}
	p.view.State, p.view.Approver = ProposalApplied, actor.ID
	r.pending = drop(r.pending, number)
	r.apply(p.view.Change, people{proposer: p.view.Proposer, approver: actor.ID}, now)
	r.done = append(r.done, number)
	if len(r.done) > maxDone {
		delete(r.proposals, r.done[0])
		r.done = r.done[1:]
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
	r.pending = drop(r.pending, number)
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

func copyProposal(v ProposalView) ProposalView {
	v.Change = copyChange(v.Change)
	return v
}
