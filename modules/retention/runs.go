package retention

import (
	"sort"
	"time"
)

func (x *run) span(i int) []string {
	low := i * x.plan.Chunk
	return x.plan.Artifacts[low:min(low+x.plan.Chunk, len(x.plan.Artifacts))]
}

func (x *run) finished() bool {
	for _, c := range x.chunks {
		if !c.doneOnce {
			return false
		}
	}
	return true
}

func (r *Registry) locate(id string, i int, actor Actor) (*run, error) {
	if err := actor.system(); err != nil {
		return nil, err
	}
	x := r.runs[id]
	switch {
	case x == nil:
		return nil, ErrNotFound
	case i < 0 || i >= len(x.chunks):
		return nil, ErrInvalid
	}
	return x, nil
}

func (r *Registry) release(x *run, a *artifact, out string) {
	x.outcomes[a.view.ID] = out
	a.run = ""
}

func (r *Registry) lease(x *run, i int, now time.Time) Claim {
	c := &x.chunks[i]
	r.tokens++
	if c.tokens == nil {
		c.tokens = map[uint64]bool{}
	}
	c.token, c.until, c.claimed = r.tokens, now.Add(x.plan.Lease), true
	c.tokens[c.token] = true
	v := Claim{Run: x.id, Chunk: i, Token: c.token, Until: c.until, Deleting: []string{}}
	for _, id := range x.span(i) {
		if x.outcomes[id] == OutcomeDeleting {
			v.Deleting = append(v.Deleting, id)
		}
	}
	if len(v.Deleting) == 0 {
		r.finishChunk(x, i)
	}
	return v
}

func (x *run) deleting(i int) bool {
	for _, id := range x.span(i) {
		if x.outcomes[id] == OutcomeDeleting {
			return true
		}
	}
	return false
}

func (r *Registry) finishChunk(x *run, i int) {
	if x.chunks[i].doneOnce {
		return
	}
	x.chunks[i].doneOnce = true
	if x.finished() {
		r.unfinished--
	}
}

func (r *Registry) Claim(id string, i int, actor Actor) (Claim, error) {
	now := r.enter()
	defer r.leave()
	x, err := r.locate(id, i, actor)
	if err != nil {
		return Claim{}, err
	}
	c := x.chunks[i]
	switch {
	case c.claimed && x.deleting(i) && now.Before(c.until):
		return Claim{}, refuse(ReasonLeased)
	case c.claimed && x.deleting(i):
		return r.lease(x, i, now), nil
	case c.doneOnce:
		return Claim{}, refuse(ReasonChunkDone)
	case i > 0 && !x.chunks[i-1].doneOnce:
		return Claim{}, refuse(ReasonOutOfOrder)
	}
	for _, aid := range x.span(i) {
		if x.outcomes[aid] != OutcomePending {
			continue
		}
		a := r.artifacts[aid]
		if out := r.exclusion(a, now); out != "" {
			r.release(x, a, out)
			continue
		}
		x.outcomes[aid] = OutcomeDeleting
	}
	return r.lease(x, i, now), nil
}

func (r *Registry) Retry(id string, i int, actor Actor) (Claim, error) {
	now := r.enter()
	defer r.leave()
	x, err := r.locate(id, i, actor)
	if err != nil {
		return Claim{}, err
	}
	c := x.chunks[i]
	switch {
	case !c.doneOnce:
		return Claim{}, refuse(ReasonNotClaimed)
	case x.deleting(i) && now.Before(c.until):
		return Claim{}, refuse(ReasonLeased)
	case x.deleting(i):
		return r.lease(x, i, now), nil
	}
	found := false
	for _, aid := range x.span(i) {
		if x.outcomes[aid] != OutcomeFailed {
			continue
		}
		found = true
		a := r.artifacts[aid]
		if out := r.exclusion(a, now); out != "" {
			r.release(x, a, out)
			continue
		}
		x.outcomes[aid] = OutcomeDeleting
		a.clearFailed()
	}
	if !found {
		return Claim{}, refuse(ReasonNothingToRetry)
	}
	return r.lease(x, i, now), nil
}

func (r *Registry) Report(id string, i int, token uint64, store string, deleted, failed []string, actor Actor) error {
	now := r.enter()
	defer r.leave()
	x, err := r.locate(id, i, actor)
	if err != nil {
		return err
	}
	c := x.chunks[i]
	if !validID(store) || !ascending(deleted, 0, maxChunk, validID) || !ascending(failed, 0, maxChunk, validID) || len(deleted)+len(failed) == 0 {
		return ErrInvalid
	}
	stale := token != c.token
	switch {
	case !c.tokens[token]:
		return refuse(ReasonStaleToken)
	case stale && len(failed) > 0:
		return refuse(ReasonStaleToken)
	}
	in := map[string]bool{}
	for _, aid := range x.span(i) {
		in[aid] = true
	}
	for _, aid := range deleted {
		a := r.artifacts[aid]
		switch {
		case !in[aid] || contains(failed, aid) || !contains(a.view.Stores, store):
			return ErrInvalid
		}
	}
	for _, aid := range failed {
		a := r.artifacts[aid]
		switch {
		case !in[aid] || !contains(a.view.Stores, store):
			return ErrInvalid
		case a.stores[store] == StoreDeleted:
			return refuse(ReasonAlreadyDeleted)
		case x.outcomes[aid] != OutcomeDeleting:
			return refuse(ReasonNotInChunk)
		}
	}
	for _, aid := range deleted {
		a := r.artifacts[aid]
		if a.stores[store] != StoreDeleted {
			a.stores[store], a.storeAt[store], a.damaged = StoreDeleted, now, true
		}
	}
	for _, aid := range failed {
		r.artifacts[aid].stores[store] = StoreFailed
	}
	for _, aid := range append(copyStrings(deleted), failed...) {
		r.settleArtifact(x, r.artifacts[aid], now)
	}
	if !x.deleting(i) {
		r.finishChunk(x, i)
	}
	return nil
}

func (r *Registry) settleArtifact(x *run, a *artifact, now time.Time) {
	all, reported := true, true
	for _, s := range a.view.Stores {
		all = all && a.stores[s] == StoreDeleted
		reported = reported && a.stores[s] != ""
	}
	switch {
	case all && !a.deleted:
		a.deleted = true
		owner := r.runs[a.run]
		r.release(x, a, OutcomeDeleted)
		tomb := x.id
		if owner != nil && owner != x {
			owner.outcomes[a.view.ID] = OutcomeDeleted
			tomb = owner.id
			if i := sort.SearchStrings(owner.plan.Artifacts, a.view.ID) / owner.plan.Chunk; owner.chunks[i].claimed && !owner.deleting(i) {
				r.finishChunk(owner, i)
			}
		}
		for _, to := range r.requires[a.view.ID] {
			r.requiredBy[to]--
		}
		r.tombstones = append(r.tombstones, Tombstone{Sequence: uint64(len(r.tombstones) + 1), Artifact: a.view.ID, Class: a.view.Class, Run: tomb, At: now})
	case reported && x.outcomes[a.view.ID] == OutcomeDeleting:
		x.outcomes[a.view.ID] = OutcomeFailed
	}
}

func (r *Registry) Proceed(id string, i int, token uint64, aid string, actor Actor) (bool, error) {
	r.enter()
	defer r.leave()
	x, err := r.locate(id, i, actor)
	if err != nil {
		return false, err
	}
	switch {
	case token != x.chunks[i].token:
		return false, refuse(ReasonStaleToken)
	case !contains(x.span(i), aid) || x.outcomes[aid] != OutcomeDeleting:
		return false, refuse(ReasonNotInChunk)
	}
	return !r.held(r.artifacts[aid]) && !r.guarding(r.artifacts[aid]), nil
}

func (r *Registry) Stop(id string, actor Actor) (RunView, error) {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return RunView{}, err
	}
	x := r.runs[id]
	if x == nil {
		return RunView{}, ErrNotFound
	}
	if !x.stopped {
		x.stopped, x.stopper = true, actor.ID
		for i, c := range x.chunks {
			if c.claimed {
				continue
			}
			for _, aid := range x.span(i) {
				if x.outcomes[aid] == OutcomePending {
					r.release(x, r.artifacts[aid], OutcomeStopped)
				}
			}
			r.finishChunk(x, i)
		}
	}
	return r.runView(x), nil
}

func (r *Registry) BackupExpired(store string, before time.Time, actor Actor) error {
	now := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	before = before.UTC()
	switch {
	case !validID(store) || before.IsZero() || before.After(now):
		return ErrInvalid
	case before.Before(r.backups[store]):
		return refuse(ReasonBackward)
	}
	r.backups[store] = before
	return nil
}

func (r *Registry) complete(a *artifact) bool {
	for _, s := range a.view.Stores {
		if !a.storeAt[s].Before(r.backups[s]) {
			return false
		}
	}
	return true
}

func (r *Registry) runView(x *run) RunView {
	v := RunView{ID: x.id, Plan: copyChange(Change{Plan: x.plan}).Plan, Revision: x.revision, Chunks: len(x.chunks), Proposer: x.by.proposer, Approver: x.by.approver,
		Stopper: x.stopper, Counts: map[string]int{}, Stores: map[string]StoreCounts{}, Outcomes: map[string]string{}}
	for _, out := range []string{OutcomePending, OutcomeDeleting, OutcomeDeleted, OutcomeFailed, OutcomeHeld, OutcomeGuarding, OutcomeReferenced, OutcomeNotDue, OutcomeStopped} {
		v.Counts[out] = 0
	}
	completeAll := true
	for _, aid := range x.plan.Artifacts {
		a := r.artifacts[aid]
		out := x.outcomes[aid]
		v.Counts[out]++
		v.Outcomes[aid] = out
		if a.damaged && !a.deleted {
			v.Damaged++
		}
		if out == OutcomeDeleted && !r.complete(a) {
			completeAll = false
		}
		for _, s := range a.view.Stores {
			sc := v.Stores[s]
			sc.Target++
			switch a.stores[s] {
			case StoreDeleted:
				sc.Deleted++
			case StoreFailed:
				sc.Failed++
			default:
				sc.Pending++
			}
			v.Stores[s] = sc
		}
	}
	for v.Next < len(x.chunks) && x.chunks[v.Next].doneOnce {
		v.Next++
	}
	switch {
	case !x.finished() && x.stopped:
		v.State = RunStopped
	case !x.finished():
		v.State = RunRunning
	case v.Counts[OutcomeFailed] == 0 && v.Counts[OutcomeDeleting] == 0 && v.Damaged == 0 && completeAll:
		v.State = RunComplete
	default:
		v.State = RunDone
	}
	return v
}

func (r *Registry) Run(id string) (RunView, error) {
	r.enter()
	defer r.leave()
	x := r.runs[id]
	if x == nil {
		return RunView{}, ErrNotFound
	}
	return r.runView(x), nil
}

func (a *artifact) clearFailed() {
	for s, v := range a.stores {
		if v == StoreFailed {
			delete(a.stores, s)
		}
	}
}
