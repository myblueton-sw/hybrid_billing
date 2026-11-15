package installer

import (
	"crypto/sha256"
	"sync"
	"time"
)

const (
	maxKeys            = 16
	namesPerKey        = 64
	minKeyLife         = time.Hour
	maxKeyLife         = 3650 * 24 * time.Hour
	maxReleases        = 10000
	maxSites           = 100000
	maxDestinations    = 64
	maxReports         = 1000
	maxRuns            = 100
	minObservationLife = time.Minute
	maxObservationLife = 24 * time.Hour
	minBootstrapLife   = time.Minute
	maxBootstrapLife   = 24 * time.Hour
	maxProposals       = 1000
	probesPerSite      = 16
	registerScope      = ""
)

type key struct {
	version    string
	public     []byte
	notBefore  time.Time
	notAfter   time.Time
	retired    bool
	compromise *Mark
}

type release struct {
	release    Release
	signature  []byte
	digest     string
	revocation *Mark
}

type run struct {
	view   RunView
	proofs []string
	paused uint64
}

func (r *run) evidence(index int) string { return r.proofs[index] }

func (r *run) record(digest string) { r.proofs = append(r.proofs, digest) }

type code struct {
	site   string
	run    string
	expiry time.Time
}

type site struct {
	view         Site
	probes       int
	revision     uint64
	config       SiteConfig
	observation  *Observation
	observed     time.Time
	sequence     uint64
	destinations map[string]Destination
	reports      []Result
	runs         map[string]*run
	order        []string
	counter      int
	admin        string
	installed    bool
	code         *[sha256.Size]byte
}

type proposal struct {
	view    ProposalView
	scope   string
	code    string
	used    bool
	retired bool
}

type scope struct {
	pending []uint64
	done    []uint64
}

type Registry struct {
	mu        sync.Mutex
	config    Config
	limits    Limits
	clock     func() time.Time
	now       time.Time
	counter   uint64
	keys      map[string]*key
	names     map[string]bool
	releases  map[string]*release
	sites     map[string]*site
	probes    map[string]string
	codes     map[[sha256.Size]byte]*code
	proposals map[uint64]*proposal
	scopes    map[string]*scope
}

func NewRegistry(config Config, limits Limits, clock func() time.Time) (*Registry, error) {
	if config.Random == nil || clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{config: config, limits: limits, clock: clock, keys: map[string]*key{}, names: map[string]bool{}, releases: map[string]*release{},
		sites: map[string]*site{}, probes: map[string]string{}, codes: map[[sha256.Size]byte]*code{}, proposals: map[uint64]*proposal{}, scopes: map[string]*scope{}}, nil
}

func (l Limits) valid() bool {
	pairs := [][2]int{{l.Keys, maxKeys}, {l.Releases, maxReleases}, {l.Sites, maxSites}, {l.Destinations, maxDestinations}, {l.Reports, maxReports},
		{l.Runs, maxRuns}, {l.Proposals, maxProposals}}
	for _, p := range pairs {
		if p[0] < 1 || p[0] > p[1] {
			return false
		}
	}
	return l.KeyLife >= minKeyLife && l.KeyLife <= maxKeyLife && l.ObservationLife >= minObservationLife && l.ObservationLife <= maxObservationLife &&
		l.BootstrapLife >= minBootstrapLife && l.BootstrapLife <= maxBootstrapLife
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
	if len(s.pending) >= r.limits.Proposals || mine >= (r.limits.Proposals+3)/4 {
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
	r.apply(p.view.Change)
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

func (r *Registry) retire(scopes []*scope, match func(Acceptance) bool) {
	for _, s := range scopes {
		for _, number := range s.done {
			p := r.proposals[number]
			if p.view.Change.Kind == ChangePlan && match(p.view.Change.Acceptance) {
				p.retired = true
			}
		}
	}
}
