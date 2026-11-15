package simulation

import (
	"sync"
	"time"
)

const (
	maxDatasets     = 10000
	maxScenarios    = 10000
	maxVersions     = 16
	maxRuns         = 1000
	maxRunsTotal    = 200000
	maxKeys         = 200000
	maxHarnesses    = 100
	maxProfiles     = 1000
	maxTargets      = 1000
	maxPending      = 1000
	maxMine         = 250
	maxDone         = 1000
	maxList         = 64
	maxChecks       = 256
	maxLinks        = 32
	maxPage         = 1000
	maxEventCount   = 10000000
	maxDecimalBytes = 40
	maxCountDigits  = 19
	maxZoneBytes    = 64
	maxHistory      = 16
	maxValues       = 8
)

type approved struct {
	view     Expectation
	author   string
	approver string
}

type scenario struct {
	view         Scenario
	fingerprint  string
	expectations []approved
	runs         []uint64
}

type key struct {
	output  string
	results string
	count   int
	marked  uint64
}

type run struct {
	view    RunView
	key     *key
	harness *harness
	checks  []Check
	codes   []byte
	values  []Outcome
}

type cell struct {
	view      Cell
	set       bool
	revisions int
	history   []CellChange
	failures  int
}

type target struct {
	view  Target
	proof *ProofView
}

type harness struct {
	identity string
	proposer string
	approver string
	revoked  bool
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
	datasets   map[string]*Dataset
	scenarios  map[string]*scenario
	runs       []*run
	keys       map[[3]string]*key
	outputs    map[string]bool
	evidences  map[string]bool
	profiles   map[string][]*cell
	targets    map[string]*target
	harnesses  map[string]*harness
	ran        map[string]bool
	identities map[string]string
	proposals  map[uint64]*proposal
	pending    []uint64
	done       []uint64
}

func NewRegistry(clock func() time.Time) (*Registry, error) {
	if clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{clock: clock, datasets: map[string]*Dataset{}, scenarios: map[string]*scenario{}, keys: map[[3]string]*key{}, outputs: map[string]bool{},
		evidences: map[string]bool{}, profiles: map[string][]*cell{}, targets: map[string]*target{}, harnesses: map[string]*harness{}, ran: map[string]bool{}, identities: map[string]string{},
		proposals: map[uint64]*proposal{}}, nil
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
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return ProposalView{}, err
	}
	c, ok := clean(c)
	if !ok {
		return ProposalView{}, ErrInvalid
	}
	if !r.scoped(c) {
		return ProposalView{}, ErrNotFound
	}
	code := string(encodeChange(c))
	for _, number := range r.pending {
		if p := r.proposals[number]; p.code == code && p.view.Proposer == actor.ID {
			return copyProposal(p.view), nil
		}
	}
	if err := r.check(c); err != nil {
		return ProposalView{}, err
	}
	mine := 0
	for _, number := range r.pending {
		if r.proposals[number].view.Proposer == actor.ID {
			mine++
		}
	}
	if len(r.pending) >= maxPending || mine >= maxMine {
		return ProposalView{}, ErrLimit
	}
	r.counter++
	p := &proposal{view: ProposalView{Number: r.counter, Change: copyChange(c), State: ProposalPending, Proposer: actor.ID}, code: code}
	r.proposals[p.view.Number] = p
	r.pending = append(r.pending, p.view.Number)
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
	if err := r.check(p.view.Change); err != nil {
		return ProposalView{}, err
	}
	p.view.State, p.view.Approver = ProposalApplied, actor.ID
	r.pending = drop(r.pending, number)
	r.apply(p.view.Change, p.view.Proposer, actor.ID, now)
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
