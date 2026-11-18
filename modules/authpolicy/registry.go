package authpolicy

import (
	"strings"
	"sync"
	"time"
)

const (
	maxWorkspaces    = 10000
	maxKeptRevisions = 100
	maxRules         = 64
	maxIssuers       = 64
	maxActivePlans   = 100
	maxEntries       = 2000000
	maxLiftedKept    = 1000
	maxPlanUsers     = 100000
	maxChunk         = 1000
	maxPendingSpace  = 100
	maxPendingMine   = 25
	maxDone          = 1000
	maxPending       = 10000
	maxSpaceEntries  = 100000
	maxReserved      = 1000
)

type people struct {
	proposer string
	approver string
}

type workspace struct {
	revisions []RevisionView
	number    uint64
	version   uint64
	active    int
	plans     uint64
}

type plan struct {
	view      Plan
	by        people
	completed int
	lifted    bool
	liftBy    people
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
	workspaces map[string]*workspace
	plans      map[string]*plan
	blocked    map[[2]string]int
	entries    int
	lifted     []string
	usage      map[string]int
	proposals  map[uint64]*proposal
	pending    []uint64
	done       []uint64
}

func NewRegistry(clock func() time.Time) (*Registry, error) {
	if clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{clock: clock, workspaces: map[string]*workspace{}, plans: map[string]*plan{}, blocked: map[[2]string]int{}, usage: map[string]int{}, proposals: map[uint64]*proposal{}}, nil
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

func drop(list []uint64, number uint64) []uint64 {
	out := list[:0]
	for _, value := range list {
		if value != number {
			out = append(out, value)
		}
	}
	return out
}

func (r *Registry) current(id string) *RevisionView {
	w := r.workspaces[id]
	if w == nil {
		return nil
	}
	return &w.revisions[len(w.revisions)-1]
}

func (r *Registry) actor(p Principal, a Authentication, space string, now time.Time) error {
	switch {
	case !validID(p.User) || !oneOf(p.Kind, KindHuman, KindSystem, KindAssistant):
		return ErrInvalid
	case p.Kind == KindAssistant:
		return refuse(ReasonAINotAllowed)
	case p.Kind == KindSystem:
		return refuse(ReasonNotHuman)
	}
	if err := formOf(p, a, ActionPolicyChange); err != nil {
		return err
	}
	switch {
	case p.Workspace != space:
		return refuse(ReasonOtherWorkspace)
	case p.Role != RoleAdministrator:
		return refuse(ReasonNotAdministrator)
	}
	rev := r.current(space)
	if rev == nil {
		return refuse(ReasonNoPolicy)
	}
	return r.judge(p, a, ActionPolicyChange, rev.Revision, now)
}

func (r *Registry) workspaceOf(c Change) string {
	if c.Kind == ChangePlan {
		return c.Plan.Workspace
	}
	return c.Lift.Plan[:strings.LastIndexByte(c.Lift.Plan, '/')]
}

func (r *Registry) Propose(c Change, p Principal, a Authentication) (ProposalView, error) {
	now := r.enter()
	defer r.leave()
	c, err := clean(c)
	if err != nil || c.Kind == ChangeRevision {
		return ProposalView{}, ErrInvalid
	}
	space := r.workspaceOf(c)
	if err := r.actor(p, a, space, now); err != nil {
		return ProposalView{}, err
	}
	code := encodeChange(c)
	inSpace, mine := 0, 0
	for _, number := range r.pending {
		x := r.proposals[number]
		if x.view.Workspace != space {
			continue
		}
		if x.code == code && x.view.Proposer == p.User {
			return copyProposal(x.view), nil
		}
		inSpace++
		if x.view.Proposer == p.User {
			mine++
		}
	}
	if err := r.check(c, p.User, ""); err != nil {
		return ProposalView{}, err
	}
	if len(r.pending) >= maxPending || inSpace >= maxPendingSpace || mine >= maxPendingMine || !r.room(space, len(c.Plan.Users)) {
		return ProposalView{}, ErrLimit
	}
	r.counter++
	x := &proposal{view: ProposalView{Number: r.counter, Change: copyChange(c), Workspace: space, State: ProposalPending, Proposer: p.User}, code: code}
	r.proposals[x.view.Number] = x
	r.pending = append(r.pending, x.view.Number)
	r.reserve(space, len(c.Plan.Users))
	return copyProposal(x.view), nil
}

func (r *Registry) Approve(number uint64, p Principal, a Authentication) (ProposalView, error) {
	now := r.enter()
	defer r.leave()
	x := r.proposals[number]
	if x == nil {
		return ProposalView{}, ErrNotFound
	}
	if err := r.actor(p, a, x.view.Workspace, now); err != nil {
		return ProposalView{}, err
	}
	switch {
	case x.view.State == ProposalApplied && x.view.Approver == p.User:
		return copyProposal(x.view), nil
	case x.view.State != ProposalPending:
		return ProposalView{}, refuse(ReasonNotPending)
	case x.view.Proposer == p.User:
		return ProposalView{}, refuse(ReasonSelfApproval)
	}
	if err := r.check(x.view.Change, x.view.Proposer, p.User); err != nil {
		return ProposalView{}, err
	}
	x.view.State, x.view.Approver = ProposalApplied, p.User
	r.pending = drop(r.pending, number)
	r.apply(x.view.Change, people{proposer: x.view.Proposer, approver: p.User}, now)
	r.bump(x.view.Workspace)
	r.done = append(r.done, number)
	if len(r.done) > maxDone {
		delete(r.proposals, r.done[0])
		r.done = r.done[1:]
	}
	return copyProposal(x.view), nil
}

func (r *Registry) Withdraw(number uint64, p Principal, a Authentication) (ProposalView, error) {
	now := r.enter()
	defer r.leave()
	x := r.proposals[number]
	if x == nil {
		return ProposalView{}, ErrNotFound
	}
	if err := r.actor(p, a, x.view.Workspace, now); err != nil {
		return ProposalView{}, err
	}
	switch {
	case x.view.Proposer != p.User:
		return ProposalView{}, refuse(ReasonNotProposer)
	case x.view.State != ProposalPending:
		return ProposalView{}, refuse(ReasonNotPending)
	}
	delete(r.proposals, number)
	r.pending = drop(r.pending, number)
	r.reserve(x.view.Workspace, -len(x.view.Change.Plan.Users))
	x.view.State = ProposalWithdrawn
	return copyProposal(x.view), nil
}

func (r *Registry) Proposal(number uint64) (ProposalView, error) {
	r.enter()
	defer r.leave()
	x := r.proposals[number]
	if x == nil {
		return ProposalView{}, ErrNotFound
	}
	return copyProposal(x.view), nil
}

func copyProposal(v ProposalView) ProposalView {
	v.Change = copyChange(v.Change)
	return v
}

func (r *Registry) room(space string, n int) bool {
	if n == 0 {
		return true
	}
	mine := r.usage[space] + n
	return mine <= maxSpaceEntries && (r.entries+n <= maxEntries || mine <= maxReserved)
}

func (r *Registry) reserve(space string, n int) {
	r.usage[space] += n
	r.entries += n
	if r.usage[space] == 0 {
		delete(r.usage, space)
	}
}
