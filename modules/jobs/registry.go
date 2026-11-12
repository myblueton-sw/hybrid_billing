package jobs

import (
	"reflect"
	"sync"
	"time"
)

type key struct {
	workspace string
	job       string
}

type pair struct {
	workspace string
	kind      string
}

type closing struct {
	complete   bool
	checkpoint string
	code       string
	permanent  bool
}

type task struct {
	spec       TaskSpec
	state      TaskState
	attempts   int
	token      uint64
	open       bool
	code       string
	checkpoint string
	notBefore  time.Time
	granted    time.Time
	expiry     time.Time
	worker     string
	closing    *closing
	resolution Resolution
}

type job struct {
	workspace string
	id        string
	kind      string
	policy    Policy
	sequence  uint64
	tasks     []*task
	index     map[string]*task
	revision  uint64
	state     JobState
	reason    WaitReason
	attention int
	cancel    CancelReason
	granted   bool
	log       []Transition
}

type Registry struct {
	mu        sync.Mutex
	built     bool
	limits    Limits
	clock     func() time.Time
	last      time.Time
	tokens    uint64
	created   uint64
	grants    uint64
	jobs      map[key]*job
	order     []*job
	perSpace  map[string]int
	lastGrant map[pair]uint64
}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, clock: clock, jobs: map[key]*job{}, perSpace: map[string]int{}, lastGrant: map[pair]uint64{}}, nil
}

func (r *Registry) enter() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	return r.built
}

func (r *Registry) now() time.Time {
	read := r.clock()
	if read.Before(r.last) {
		return r.last
	}
	r.last = read
	return read
}

func (j *job) counts() map[TaskState]int {
	counts := map[TaskState]int{}
	for _, held := range j.tasks {
		counts[held.state]++
	}
	return counts
}

func (j *job) change(held *task, worker string, event Event, at time.Time) {
	counts := j.counts()
	j.state, j.reason = derive(counts, j.cancel != "", j.granted)
	j.attention = counts[Unknown]
	j.revision++
	name := ""
	if held != nil {
		name = held.spec.ID
	}
	j.log = append(j.log, Transition{Revision: j.revision, At: at, Task: name, Worker: worker, Event: event, State: j.state})
}

func (j *job) sweep(at time.Time) {
	for _, held := range j.tasks {
		if held.state != Leased || at.Before(held.expiry) {
			continue
		}
		if held.spec.Repeatable && held.attempts < j.policy.Attempts {
			held.state, held.notBefore = Pending, at.Add(j.policy.backoff(held.attempts))
		} else {
			held.state, held.code = Unknown, CodeLeaseExpired
		}
		j.change(held, held.worker, EventExpired, at)
	}
}

func (r *Registry) Create(workspace, id, kind string, policy Policy, tasks []TaskSpec) (bool, error) {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return false, ErrInvalid
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(id) || !validID(kind) || !policy.valid() || len(tasks) == 0 {
		return false, ErrInvalid
	}
	seen := map[string]bool{}
	for _, spec := range tasks {
		if !validID(spec.ID) || !validID(spec.Input) || seen[spec.ID] {
			return false, ErrInvalid
		}
		seen[spec.ID] = true
	}
	if stored := r.jobs[key{workspace, id}]; stored != nil {
		specs := make([]TaskSpec, len(stored.tasks))
		for index, held := range stored.tasks {
			specs[index] = held.spec
		}
		if stored.kind == kind && stored.policy == policy && reflect.DeepEqual(specs, tasks) {
			return false, nil
		}
		return false, ErrConflict
	}
	if policy.Attempts > r.limits.Attempts || len(tasks) > r.limits.Tasks || r.perSpace[workspace] >= r.limits.Jobs {
		return false, ErrLimit
	}
	at := r.now()
	r.created++
	made := &job{workspace: workspace, id: id, kind: kind, policy: policy, sequence: r.created, index: map[string]*task{}}
	for _, spec := range tasks {
		held := &task{spec: spec, state: Pending}
		made.tasks = append(made.tasks, held)
		made.index[spec.ID] = held
	}
	made.change(nil, "", EventCreated, at)
	r.jobs[key{workspace, id}] = made
	r.order = append(r.order, made)
	r.perSpace[workspace]++
	return true, nil
}

func (r *Registry) candidate(kinds map[string]bool, at time.Time) (*job, *task) {
	leasedKind, leasedPair := map[string]int{}, map[pair]int{}
	for _, stored := range r.order {
		for _, held := range stored.tasks {
			if held.state == Leased {
				leasedKind[stored.kind]++
				leasedPair[pair{stored.workspace, stored.kind}]++
			}
		}
	}
	var bestJob *job
	var bestTask *task
	for _, stored := range r.order {
		slot := pair{stored.workspace, stored.kind}
		if !kinds[stored.kind] || stored.state.Terminal() || stored.cancel != "" ||
			leasedKind[stored.kind] >= r.limits.PerType || leasedPair[slot] >= r.limits.PerWorkspace {
			continue
		}
		var eligible *task
		for _, held := range stored.tasks {
			if held.state == Pending && !at.Before(held.notBefore) && held.attempts < stored.policy.Attempts {
				eligible = held
				break
			}
		}
		if eligible == nil {
			continue
		}
		if bestJob != nil {
			chosen := pair{bestJob.workspace, bestJob.kind}
			if leasedPair[slot] > leasedPair[chosen] || (leasedPair[slot] == leasedPair[chosen] && r.lastGrant[slot] >= r.lastGrant[chosen]) {
				continue
			}
		}
		bestJob, bestTask = stored, eligible
	}
	return bestJob, bestTask
}

func (r *Registry) Lease(worker string, kinds []string, limit int) ([]Lease, error) {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return nil, ErrInvalid
	}
	defer r.mu.Unlock()
	if !validID(worker) || len(kinds) == 0 || limit < 1 {
		return nil, ErrInvalid
	}
	if limit > r.limits.Batch {
		return nil, ErrLimit
	}
	wanted := map[string]bool{}
	for _, kind := range kinds {
		if !validID(kind) || wanted[kind] {
			return nil, ErrInvalid
		}
		wanted[kind] = true
	}
	at := r.now()
	for _, stored := range r.order {
		stored.sweep(at)
	}
	leases := []Lease{}
	for len(leases) < limit {
		stored, held := r.candidate(wanted, at)
		if stored == nil {
			break
		}
		r.tokens++
		r.grants++
		r.lastGrant[pair{stored.workspace, stored.kind}] = r.grants
		held.state, held.token, held.open, held.worker, held.resolution = Leased, r.tokens, true, worker, ""
		held.attempts++
		held.granted, held.expiry = at, at.Add(stored.policy.Lease)
		stored.granted = true
		stored.change(held, worker, EventGranted, at)
		leases = append(leases, Lease{WorkspaceID: stored.workspace, JobID: stored.id, TaskID: held.spec.ID, Input: held.spec.Input, Checkpoint: held.checkpoint,
			Token: held.token, Attempt: held.attempts, Expiry: held.expiry})
	}
	return leases, nil
}

func (r *Registry) Sweep() error {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return ErrInvalid
	}
	defer r.mu.Unlock()
	at := r.now()
	for _, stored := range r.order {
		stored.sweep(at)
	}
	return nil
}

func (r *Registry) find(workspace, id, name string) (*job, *task, time.Time, error) {
	if !validID(workspace) || !validID(id) || !validID(name) {
		return nil, nil, time.Time{}, ErrInvalid
	}
	stored := r.jobs[key{workspace, id}]
	if stored == nil || stored.index[name] == nil {
		return nil, nil, time.Time{}, ErrNotFound
	}
	at := r.now()
	stored.sweep(at)
	return stored, stored.index[name], at, nil
}

func (r *Registry) held(ref Ref) (*job, *task, time.Time, error) {
	stored, held, at, err := r.find(ref.WorkspaceID, ref.JobID, ref.TaskID)
	if err != nil {
		return nil, nil, at, err
	}
	if ref.Token == 0 || held.token != ref.Token {
		return nil, nil, at, ErrStale
	}
	return stored, held, at, nil
}

func (r *Registry) Extend(ref Ref) (time.Time, error) {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return time.Time{}, ErrInvalid
	}
	defer r.mu.Unlock()
	stored, held, at, err := r.held(ref)
	if err != nil {
		return time.Time{}, err
	}
	if held.state != Leased {
		return time.Time{}, ErrStale
	}
	held.expiry = at.Add(stored.policy.Lease)
	if longest := held.granted.Add(stored.policy.Life); held.expiry.After(longest) {
		held.expiry = longest
	}
	return held.expiry, nil
}

func (r *Registry) Checkpoint(ref Ref, checkpoint string) error {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return ErrInvalid
	}
	defer r.mu.Unlock()
	if !validID(checkpoint) {
		return ErrInvalid
	}
	_, held, _, err := r.held(ref)
	if err != nil {
		return err
	}
	if held.state != Leased {
		return ErrStale
	}
	held.checkpoint = checkpoint
	return nil
}

func (r *Registry) Complete(ref Ref, checkpoint string) (Report, error) {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return Report{}, ErrInvalid
	}
	defer r.mu.Unlock()
	if !validID(checkpoint) {
		return Report{}, ErrInvalid
	}
	stored, held, at, err := r.held(ref)
	if err != nil {
		return Report{}, err
	}
	if !held.open {
		if done := held.closing; done != nil && done.complete && done.checkpoint == checkpoint {
			return Report{State: held.state, Replay: true}, nil
		}
		return Report{}, ErrStale
	}
	held.state, held.checkpoint, held.open = TaskSucceeded, checkpoint, false
	held.closing = &closing{complete: true, checkpoint: checkpoint}
	stored.change(held, held.worker, EventCompleted, at)
	return Report{State: TaskSucceeded}, nil
}

func (r *Registry) Fail(ref Ref, code string, permanent bool) (Report, error) {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return Report{}, ErrInvalid
	}
	defer r.mu.Unlock()
	if !validID(code) {
		return Report{}, ErrInvalid
	}
	stored, held, at, err := r.held(ref)
	if err != nil {
		return Report{}, err
	}
	if !held.open {
		if done := held.closing; done != nil && done.code == code && done.permanent == permanent {
			return Report{State: held.state, Replay: true}, nil
		}
		return Report{}, ErrStale
	}
	switch {
	case permanent || held.attempts >= stored.policy.Attempts:
		held.state = TaskFailed
	case stored.cancel != "":
		held.state = TaskCancelled
	default:
		held.state, held.notBefore = Pending, at.Add(stored.policy.backoff(held.attempts))
	}
	held.code, held.open = code, false
	held.closing = &closing{code: code, permanent: permanent}
	stored.change(held, held.worker, EventFailed, at)
	return Report{State: held.state}, nil
}

func (r *Registry) Unknown(ref Ref, code string) (Report, error) {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return Report{}, ErrInvalid
	}
	defer r.mu.Unlock()
	if !validID(code) {
		return Report{}, ErrInvalid
	}
	stored, held, at, err := r.held(ref)
	if err != nil {
		return Report{}, err
	}
	if !held.open {
		return Report{}, ErrStale
	}
	if held.state == Unknown {
		return Report{State: Unknown, Replay: true}, nil
	}
	held.state, held.code = Unknown, code
	stored.change(held, held.worker, EventUnknown, at)
	return Report{State: Unknown}, nil
}

func (r *Registry) Resolve(workspace, id, name string, token uint64, resolution Resolution) (Report, error) {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return Report{}, ErrInvalid
	}
	defer r.mu.Unlock()
	if resolution != ResolveSucceeded && resolution != ResolveFailed && resolution != ResolveRetry {
		return Report{}, ErrInvalid
	}
	stored, held, at, err := r.held(Ref{WorkspaceID: workspace, JobID: id, TaskID: name, Token: token})
	if err != nil {
		return Report{}, err
	}
	if done := held.resolution; done != "" {
		if done == resolution {
			return Report{State: held.state, Replay: true}, nil
		}
		return Report{}, ErrConflict
	}
	if held.state != Unknown || (resolution == ResolveRetry && (stored.cancel != "" || held.attempts >= stored.policy.Attempts)) {
		return Report{}, ErrRejected
	}
	switch resolution {
	case ResolveSucceeded:
		held.state = TaskSucceeded
	case ResolveFailed:
		held.state = TaskFailed
	default:
		held.state, held.notBefore = Pending, at
	}
	held.open, held.closing, held.resolution = false, nil, resolution
	stored.change(held, "", EventResolved, at)
	return Report{State: held.state}, nil
}

func (r *Registry) Cancel(workspace, id string, reason CancelReason) error {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return ErrInvalid
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(id) || (reason != Requested && reason != PermissionWithdrawn && reason != Superseded) {
		return ErrInvalid
	}
	stored := r.jobs[key{workspace, id}]
	if stored == nil {
		return ErrNotFound
	}
	if stored.cancel != "" {
		if stored.cancel == reason {
			return nil
		}
		return ErrConflict
	}
	if stored.state.Terminal() {
		return ErrRejected
	}
	at := r.now()
	stored.sweep(at)
	stored.cancel = reason
	for _, held := range stored.tasks {
		switch {
		case held.state == Leased || (held.state == Pending && held.open):
			held.state, held.code = Unknown, CodeCancelledWhileOpen
		case held.state == Pending:
			held.state = TaskCancelled
		}
	}
	stored.change(nil, "", EventCancel, at)
	return nil
}

func (r *Registry) Job(workspace, id string) (JobView, error) {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return JobView{}, ErrInvalid
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(id) {
		return JobView{}, ErrInvalid
	}
	stored := r.jobs[key{workspace, id}]
	if stored == nil {
		return JobView{}, ErrNotFound
	}
	at := r.now()
	view := JobView{WorkspaceID: stored.workspace, ID: stored.id, Type: stored.kind, Policy: stored.policy, State: stored.state, Reason: stored.reason,
		Attention: stored.attention, Cancel: stored.cancel, Revision: stored.revision, Counts: stored.counts(), AsOf: at}
	for _, held := range stored.tasks {
		shown := TaskView{ID: held.spec.ID, Input: held.spec.Input, Repeatable: held.spec.Repeatable, State: held.state, Attempts: held.attempts, Token: held.token,
			Open: held.open, Code: held.code, Checkpoint: held.checkpoint}
		if held.state == Pending {
			shown.NotBefore = held.notBefore
		}
		if held.state == Leased {
			shown.Expiry = held.expiry
			if !at.Before(held.expiry) {
				view.Overdue++
			}
		}
		view.Tasks = append(view.Tasks, shown)
	}
	return view, nil
}

func (r *Registry) Transitions(workspace, id string) ([]Transition, error) {
	if !r.enter() {
		if r != nil {
			r.mu.Unlock()
		}
		return nil, ErrInvalid
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(id) {
		return nil, ErrInvalid
	}
	stored := r.jobs[key{workspace, id}]
	if stored == nil {
		return nil, ErrNotFound
	}
	return append([]Transition(nil), stored.log...), nil
}
