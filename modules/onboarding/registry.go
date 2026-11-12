package onboarding

import (
	"reflect"
	"sync"
	"time"
)

type key struct {
	workspace string
	project   string
}

type grant struct {
	number    uint64
	role      Role
	holder    string
	at        time.Time
	expiry    time.Time
	condition Condition
	ended     bool
}

func (g *grant) active(at time.Time) bool {
	return g != nil && !g.ended && (g.expiry.IsZero() || at.Before(g.expiry))
}

type run struct {
	id         string
	given      Run
	sequence   uint64
	generation uint64
	grant      *grant
	underSetup bool
	at         time.Time
	until      time.Time
}

type record struct {
	run     *run
	invalid Reason
}

type opening struct {
	invalid Reason
}

type written struct {
	kind     string
	content  any
	sequence uint64
}

type project struct {
	workspace  string
	id         string
	source     string
	path       Path
	hash       string
	created    string
	required   []Stage
	sequence   uint64
	generation uint64
	runs       map[string]*run
	listed     []*run
	stages     map[Stage]*record
	floors     map[Stage]uint64
	grants     []*grant
	active     map[Role]*grant
	opening    *opening
	writes     map[string]written
}

type Registry struct {
	mu       sync.Mutex
	built    bool
	limits   Limits
	version  string
	clock    func() time.Time
	last     time.Time
	projects map[key]*project
	perSpace map[string]int
}

func NewRegistry(limits Limits, version string, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || !validID(version) || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, version: version, clock: clock, projects: map[key]*project{}, perSpace: map[string]int{}}, nil
}

func (r *Registry) enter() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	return r.built
}

func (r *Registry) leave() {
	if r != nil {
		r.mu.Unlock()
	}
}

func (r *Registry) now() time.Time {
	if read := r.clock(); read.After(r.last) {
		r.last = read
	}
	return r.last
}

func (r *Registry) Create(workspace, id, source string, path Path, hash string) (bool, error) {
	defer r.leave()
	if !r.enter() || !validID(workspace) || !validID(id) || !validID(source) || !path.known() || !validID(hash) {
		return false, ErrInvalid
	}
	r.now()
	if stored := r.projects[key{workspace, id}]; stored != nil {
		if stored.source == source && stored.path == path && stored.created == hash {
			return false, nil
		}
		return false, ErrConflict
	}
	if r.perSpace[workspace] >= r.limits.Projects {
		return false, ErrLimit
	}
	r.projects[key{workspace, id}] = &project{workspace: workspace, id: id, source: source, path: path, hash: hash, created: hash, required: Required(path), generation: 1,
		runs: map[string]*run{}, stages: map[Stage]*record{}, floors: map[Stage]uint64{}, active: map[Role]*grant{}, writes: map[string]written{}}
	r.perSpace[workspace]++
	return true, nil
}

func (p *project) status(stage Stage, at time.Time) (Status, Reason) {
	held := p.stages[stage]
	switch {
	case held == nil:
		return Missing, ""
	case held.invalid != "":
		return Stale, held.invalid
	case p.path.connected() && !held.run.grant.active(at):
		return Stale, RoleEnded
	case !at.Before(held.run.until):
		return Stale, Expired
	}
	return Holding, ""
}

func (p *project) holds(stage Stage, at time.Time) bool {
	status, _ := p.status(stage, at)
	return status == Holding
}

func (p *project) empty(stage Stage) bool {
	held := p.stages[stage]
	return held != nil && held.run.given.Outcome == PassedNoUsage
}

func (p *project) blockers(at time.Time, used bool) []Blocker {
	found := []Blocker{}
	for _, stage := range p.required {
		if stage == BillingOpen {
			continue
		}
		switch view := p.view(stage, at); {
		case view.Status == Missing:
			found = append(found, Blocker{Stage: stage, Reason: NotReached})
		case view.Status == Stale:
			found = append(found, Blocker{Stage: stage, Reason: view.Reason})
		case view.Empty:
			found = append(found, Blocker{Stage: stage, Reason: Empty})
		}
	}
	if p.path.connected() {
		if p.active[Setup].active(at) {
			found = append(found, Blocker{Stage: BillingOpen, Reason: SetupActive})
		}
		if read := p.stages[OperationalRead]; read != nil && read.run.underSetup {
			found = append(found, Blocker{Stage: OperationalRead, Reason: ReadUnderSetup})
		}
	}
	if used {
		found = append(found, Blocker{Stage: BillingOpen, Reason: CapacityExhausted})
	}
	return found
}

func (p *project) view(stage Stage, at time.Time) StageView {
	shown := StageView{Stage: stage, Status: Missing, Prerequisites: true}
	for _, earlier := range p.required {
		if earlier == stage {
			break
		}
		if !p.holds(earlier, at) {
			shown.Prerequisites = false
		}
	}
	shown.Status, shown.Reason = p.status(stage, at)
	if held := p.stages[stage]; held != nil {
		shown.Run, shown.Empty = held.run.id, p.empty(stage)
	}
	return shown
}

func (r *Registry) room(reserved bool) uint64 {
	if reserved {
		return uint64(r.limits.Writes) + uint64(r.limits.Reserve)
	}
	return uint64(r.limits.Writes)
}

func (r *Registry) used(p *project) bool { return p.sequence >= r.room(true) }

func (r *Registry) billable(p *project, at time.Time) bool {
	return len(p.blockers(at, r.used(p))) == 0
}

func (r *Registry) guard(p *project, at time.Time, reason Reason) {
	if p.opening != nil && p.opening.invalid == "" && !r.billable(p, at) {
		p.opening.invalid = reason
	}
}

func (r *Registry) begin(workspace, id, write, kind string, content any, expected *uint64, reserved bool) (*project, time.Time, Result, bool, error) {
	if !r.enter() || !validID(workspace) || !validID(id) || !validID(write) {
		return nil, time.Time{}, Result{}, false, ErrInvalid
	}
	at := r.now()
	found := r.projects[key{workspace, id}]
	if found == nil {
		return nil, at, Result{}, false, ErrNotFound
	}
	if stored, repeated := found.writes[write]; repeated {
		if stored.kind == kind && reflect.DeepEqual(stored.content, content) {
			return found, at, Result{Sequence: stored.sequence, Replay: true}, true, nil
		}
		return nil, at, Result{}, false, ErrConflict
	}
	if expected != nil && *expected != found.sequence {
		return nil, at, Result{}, false, ErrConflict
	}
	if found.sequence >= r.room(reserved) {
		return nil, at, Result{}, false, ErrLimit
	}
	return found, at, Result{}, false, nil
}

func (r *Registry) commit(p *project, at time.Time, write, kind string, content any, after Reason) Result {
	p.sequence++
	p.writes[write] = written{kind: kind, content: content, sequence: p.sequence}
	if r.used(p) {
		after = CapacityExhausted
	}
	r.guard(p, at, after)
	return Result{Sequence: p.sequence}
}

func (r *Registry) RecordRun(workspace, id string, expected uint64, write string, given Run) (Result, error) {
	defer r.leave()
	found, at, result, replay, err := r.begin(workspace, id, write, "run", given, &expected, false)
	if err != nil || replay {
		return result, err
	}
	if position(found.required, given.Stage) < 0 || given.Stage == BillingOpen || !allowed(given.Outcome, given.Stage, found.path) ||
		given.Validity <= 0 || !validID(given.Detail) || !validID(given.Version) ||
		(given.Outcome == Failed) != (given.Code != "") || (given.Code != "" && !validID(given.Code)) {
		return Result{}, ErrInvalid
	}
	if given.Outcome == UnsupportedScope {
		if given.Alternative == found.path || (given.Alternative != NoAlternative && !given.Alternative.known()) {
			return Result{}, ErrInvalid
		}
	} else if given.Alternative != "" {
		return Result{}, ErrInvalid
	}
	if found.path.connected() != given.Role.known() || (!found.path.connected() && given.Role != "") {
		return Result{}, ErrInvalid
	}
	if given.Validity > r.limits.MaxValidity {
		return Result{}, ErrLimit
	}
	if given.Version != r.version {
		return Result{}, ErrRejected
	}
	made := &run{id: write, given: given, sequence: found.sequence + 1, generation: found.generation, at: at, until: at.Add(given.Validity)}
	if found.path.connected() {
		made.grant = found.active[given.Role]
		if !made.grant.active(at) {
			return Result{}, ErrRejected
		}
		made.underSetup = found.active[Setup].active(at)
	}
	r.guard(found, at, EvidenceLapsed)
	found.runs[write] = made
	found.listed = append(found.listed, made)
	return r.commit(found, at, write, "run", given, EvidenceLapsed), nil
}

type reached struct {
	stage Stage
	run   string
}

func (r *Registry) Reach(workspace, id string, expected uint64, write string, stage Stage, name string) (Result, error) {
	defer r.leave()
	found, at, result, replay, err := r.begin(workspace, id, write, "reach", reached{stage, name}, &expected, false)
	if err != nil || replay {
		return result, err
	}
	place := position(found.required, stage)
	if place < 0 || stage == BillingOpen || !validID(name) {
		return Result{}, ErrInvalid
	}
	evidence := found.runs[name]
	if evidence == nil {
		return Result{}, ErrNotFound
	}
	current := found.stages[stage]
	if evidence.given.Stage != stage || !evidence.given.Outcome.reaches() || evidence.generation != found.generation || !at.Before(evidence.until) ||
		(found.path.connected() && (evidence.given.Role != CollectorReader || !evidence.grant.active(at))) ||
		(current != nil && evidence.sequence <= current.run.sequence) || evidence.sequence <= found.floors[stage] {
		return Result{}, ErrRejected
	}
	blank := evidence.given.Outcome == PassedNoUsage
	if stage == FullCoverage {
		if first := found.holds(FirstData, at); (first && found.empty(FirstData) != blank) || (!first && blank) {
			return Result{}, ErrRejected
		}
	}
	r.guard(found, at, EvidenceLapsed)
	renewal := found.holds(stage, at) && found.empty(stage) == blank
	found.stages[stage] = &record{run: evidence}
	if !renewal {
		for _, later := range found.required[place+1:] {
			found.floors[later] = found.sequence + 1
			if held := found.stages[later]; held != nil && held.invalid == "" {
				held.invalid = UpstreamRenewed
			}
		}
		if found.opening != nil && found.opening.invalid == "" {
			found.opening.invalid = UpstreamRenewed
		}
	}
	return r.commit(found, at, write, "reach", reached{stage, name}, EvidenceLapsed), nil
}

type changed struct {
	hash string
	from Stage
}

func (r *Registry) ChangeConfig(workspace, id, write, hash string, from Stage) (Result, error) {
	defer r.leave()
	found, at, result, replay, err := r.begin(workspace, id, write, "config", changed{hash, from}, nil, true)
	if err != nil || replay {
		return result, err
	}
	place := position(found.required, from)
	if place < 0 || !validID(hash) {
		return Result{}, ErrInvalid
	}
	r.guard(found, at, EvidenceLapsed)
	found.hash = hash
	found.generation++
	for _, stage := range found.required[place:] {
		found.floors[stage] = found.sequence + 1
		if held := found.stages[stage]; held != nil && held.invalid == "" {
			held.invalid = ConfigChanged
		}
	}
	if found.opening != nil && found.opening.invalid == "" {
		found.opening.invalid = ConfigChanged
	}
	return r.commit(found, at, write, "config", changed{hash, from}, EvidenceLapsed), nil
}

type granted struct {
	role      Role
	holder    string
	duration  time.Duration
	condition Condition
}

func (r *Registry) GrantRole(workspace, id string, expected uint64, write string, role Role, holder string, duration time.Duration, condition Condition) (Result, error) {
	defer r.leave()
	content := granted{role, holder, duration, condition}
	found, at, result, replay, err := r.begin(workspace, id, write, "grant", content, &expected, false)
	if err != nil || replay {
		return result, err
	}
	if !role.known() || !validID(holder) || (role == Setup) != (duration > 0) || (role == Setup) != condition.known() || (role != Setup && (duration != 0 || condition != "")) {
		return Result{}, ErrInvalid
	}
	if duration > r.limits.MaxSetup {
		return Result{}, ErrLimit
	}
	if !found.path.connected() || found.active[role].active(at) {
		return Result{}, ErrRejected
	}
	r.guard(found, at, EvidenceLapsed)
	made := &grant{number: uint64(len(found.grants)) + 1, role: role, holder: holder, at: at, condition: condition}
	if role == Setup {
		made.expiry = at.Add(duration)
	}
	found.grants = append(found.grants, made)
	found.active[role] = made
	after := EvidenceLapsed
	if role == Setup {
		after = SetupGranted
	}
	return r.commit(found, at, write, "grant", content, after), nil
}

func (r *Registry) WithdrawRole(workspace, id, write string, role Role) (Result, error) {
	defer r.leave()
	found, at, result, replay, err := r.begin(workspace, id, write, "withdraw", role, nil, true)
	if err != nil || replay {
		return result, err
	}
	if !role.known() {
		return Result{}, ErrInvalid
	}
	if !found.active[role].active(at) {
		return Result{}, ErrRejected
	}
	r.guard(found, at, EvidenceLapsed)
	found.active[role].ended = true
	return r.commit(found, at, write, "withdraw", role, EvidenceLapsed), nil
}

func (r *Registry) OpenBilling(workspace, id string, expected uint64, write string) (Result, []Blocker, error) {
	defer r.leave()
	found, at, result, replay, err := r.begin(workspace, id, write, "open", nil, &expected, false)
	if err != nil || replay {
		return result, nil, err
	}
	if blocked := found.blockers(at, r.used(found)); len(blocked) > 0 {
		return Result{}, blocked, ErrRejected
	}
	if found.opening != nil && found.opening.invalid == "" {
		return Result{}, nil, ErrRejected
	}
	found.opening = &opening{}
	return r.commit(found, at, write, "open", nil, EvidenceLapsed), nil, nil
}

func (r *Registry) find(workspace, id string) (*project, time.Time, error) {
	if !r.enter() || !validID(workspace) || !validID(id) {
		return nil, time.Time{}, ErrInvalid
	}
	at := r.now()
	found := r.projects[key{workspace, id}]
	if found == nil {
		return nil, at, ErrNotFound
	}
	return found, at, nil
}

func (r *Registry) Readiness(workspace, id string) (Readiness, error) {
	defer r.leave()
	found, at, err := r.find(workspace, id)
	if err != nil {
		return Readiness{}, err
	}
	view := Readiness{WorkspaceID: found.workspace, ProjectID: found.id, Path: found.path, Source: found.source, Hash: found.hash, Generation: found.generation,
		Sequence: found.sequence, Required: len(found.required), AsOf: at, Blockers: found.blockers(at, r.used(found)), Roles: []RoleView{}}
	view.BillingOpen = found.opening != nil && found.opening.invalid == "" && len(view.Blockers) == 0
	unbroken := true
	for _, stage := range found.required {
		shown := found.view(stage, at)
		if stage == BillingOpen {
			switch {
			case found.opening == nil:
			case view.BillingOpen:
				shown.Status = Holding
			case found.opening.invalid != "":
				shown.Status, shown.Reason = Stale, found.opening.invalid
			default:
				shown.Status, shown.Reason = Stale, EvidenceLapsed
			}
		}
		if unbroken && shown.Status == Holding {
			view.Progress++
		} else {
			unbroken = false
		}
		view.Stages = append(view.Stages, shown)
	}
	for _, made := range found.grants {
		view.Roles = append(view.Roles, RoleView{Role: made.role, Grant: made.number, Holder: made.holder, Active: made.active(at), At: made.at, Expiry: made.expiry, Condition: made.condition})
	}
	return view, nil
}

func (r *Registry) Runs(workspace, id string) ([]RunView, error) {
	defer r.leave()
	found, _, err := r.find(workspace, id)
	if err != nil {
		return nil, err
	}
	views := []RunView{}
	for _, made := range found.listed {
		shown := RunView{ID: made.id, Run: made.given, Sequence: made.sequence, Generation: made.generation, UnderSetup: made.underSetup, At: made.at, Until: made.until}
		if made.grant != nil {
			shown.Grant = made.grant.number
		}
		views = append(views, shown)
	}
	return views, nil
}
