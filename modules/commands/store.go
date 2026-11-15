package commands

import (
	"sort"
	"sync"
	"time"
)

type holder struct {
	workspace string
	user      string
}

type bucket struct {
	tokens int
	last   time.Time
}

type Throttle struct {
	User  string
	Count int
	First time.Time
	Last  time.Time
}

type Unrecorded struct {
	User  string
	Code  Code
	Count int
	First time.Time
	Last  time.Time
}

type unrecordedKey struct {
	workspace string
	user      string
	code      Code
}

type held struct {
	workspaces map[string]int
	members    map[string]int
	users      map[holder]int
}

func newHeld() held {
	return held{workspaces: map[string]int{}, members: map[string]int{}, users: map[holder]int{}}
}

func (h held) room(who holder, l Limits, user, workspace int) bool {
	switch {
	case h.workspaces[who.workspace] == 0 && len(h.workspaces) >= l.Workspaces:
		return false
	case h.users[who] == 0 && h.members[who.workspace] >= l.Users:
		return false
	}
	return h.users[who] < user && h.workspaces[who.workspace] < workspace
}

func (h held) add(who holder) {
	if h.users[who] == 0 {
		h.members[who.workspace]++
	}
	h.workspaces[who.workspace]++
	h.users[who]++
}

func (h held) remove(who holder) {
	if h.users[who]--; h.users[who] == 0 {
		delete(h.users, who)
		if h.members[who.workspace]--; h.members[who.workspace] == 0 {
			delete(h.members, who.workspace)
		}
	}
	if h.workspaces[who.workspace]--; h.workspaces[who.workspace] == 0 {
		delete(h.workspaces, who.workspace)
	}
}

func seen(first, last *time.Time, at time.Time) {
	if first.IsZero() {
		*first = at
	}
	*last = at
}

func (r *Registry) take(who holder) (time.Duration, bool) {
	now := r.instant()
	r.mu.Lock()
	defer r.mu.Unlock()
	b := r.buckets[who]
	if b == nil {
		if len(r.buckets) >= 2*r.cleaned {
			r.clean(now)
		}
		b = &bucket{tokens: r.limits.Burst, last: now}
		r.buckets[who] = b
	}
	switch {
	case now.IsZero():
	case now.Before(b.last) && b.last.Sub(now) < r.limits.Refill:
		now = b.last
	case now.Before(b.last):
		b.last = now
	default:
		gained := now.Sub(b.last) / r.limits.Refill
		if gained >= time.Duration(r.limits.Burst-b.tokens) {
			b.tokens = r.limits.Burst
		} else if gained > 0 {
			b.tokens += int(gained)
			b.last = b.last.Add(gained * r.limits.Refill)
		}
		if b.tokens == r.limits.Burst {
			b.last = now
		}
	}
	if b.tokens > 0 {
		b.tokens--
		return 0, true
	}
	wait := r.limits.Refill
	if !now.IsZero() && !now.Before(b.last) {
		wait -= now.Sub(b.last)
	}
	byWorkspace := r.throttled[who.workspace]
	if byWorkspace == nil {
		byWorkspace = map[string]*Throttle{}
		r.throttled[who.workspace] = byWorkspace
	}
	t := byWorkspace[who.user]
	if t == nil {
		t = &Throttle{User: who.user}
		byWorkspace[who.user] = t
	}
	t.Count++
	seen(&t.First, &t.Last, now)
	return wait, false
}

func (r *Registry) clean(now time.Time) {
	for who, b := range r.buckets {
		if b.tokens == r.limits.Burst || !now.IsZero() && !now.Before(b.last) && now.Sub(b.last)/r.limits.Refill >= time.Duration(r.limits.Burst-b.tokens) {
			delete(r.buckets, who)
		}
	}
	r.cleaned = max(len(r.buckets), 1)
}

func (r *Registry) Throttled(workspace string) []Throttle {
	if r == nil || !r.built {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Throttle{}
	for _, t := range r.throttled[workspace] {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].User < out[j].User })
	return out
}

func (r *Registry) unrecorded(who holder, code Code) {
	now := r.instant()
	r.mu.Lock()
	defer r.mu.Unlock()
	key := unrecordedKey{who.workspace, who.user, code}
	u := r.aggregated[key]
	if u == nil {
		u = &Unrecorded{User: who.user, Code: code}
		r.aggregated[key] = u
	}
	u.Count++
	seen(&u.First, &u.Last, now)
}

func (r *Registry) Unrecorded(workspace string) []Unrecorded {
	if r == nil || !r.built {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Unrecorded{}
	for key, u := range r.aggregated {
		if key.workspace == workspace {
			out = append(out, *u)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].User != out[j].User {
			return out[i].User < out[j].User
		}
		return out[i].Code < out[j].Code
	})
	return out
}

func (r *Registry) sweepJournal(now time.Time) {
	if now.IsZero() {
		return
	}
	for r.head < len(r.journal) {
		kept := r.kept[r.head]
		if kept.IsZero() || now.Before(kept.Add(r.limits.Retention)) {
			break
		}
		entry := r.journal[r.head]
		r.records.remove(holder{entry.Workspace, entry.User})
		r.journal[r.head], r.kept[r.head] = Record{}, time.Time{}
		r.head++
	}
	if r.head > 0 && r.head*2 >= len(r.journal) {
		r.journal = append([]Record(nil), r.journal[r.head:]...)
		r.kept = append([]time.Time(nil), r.kept[r.head:]...)
		r.head = 0
	}
}

func (r *Registry) reserve(who holder) (time.Time, bool) {
	now := r.instant()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sweepJournal(now)
	if len(r.journal)-r.head+r.reserved >= r.limits.Journal || !r.records.room(who, r.limits, r.limits.UserJournal, r.limits.WorkspaceJournal) {
		return now, false
	}
	r.reserved++
	r.records.add(who)
	return now, true
}

func (r *Registry) record(entry Record, reserved time.Time) {
	entry.At = r.instant()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reserved--
	r.sequence++
	entry.Sequence = r.sequence
	kept := entry.At
	switch {
	case !kept.IsZero():
	case !r.lastKept.IsZero():
		kept = r.lastKept
	default:
		kept = reserved
	}
	if !kept.IsZero() {
		r.lastKept = kept
		for index := len(r.kept) - 1; index >= r.head && r.kept[index].IsZero(); index-- {
			r.kept[index] = kept
		}
	}
	r.journal = append(r.journal, entry)
	r.kept = append(r.kept, kept)
}

func (r *Registry) Journal(after uint64, limit int) []Record {
	if r == nil || !r.built || limit < 1 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	live := r.journal[r.head:]
	first := sort.Search(len(live), func(i int) bool { return live[i].Sequence > after })
	rest := live[first:]
	return append([]Record{}, rest[:min(limit, len(rest))]...)
}

func (r *Registry) JournalStart() uint64 {
	if r == nil || !r.built {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.head < len(r.journal) {
		return r.journal[r.head].Sequence
	}
	return r.sequence + 1
}

func (r *Registry) drop(key operationKey, op *operation) {
	if r.operations[key] == op {
		delete(r.operations, key)
		r.stored.remove(holder{op.workspace, op.user})
	}
}

func (r *Registry) expired(op *operation, now time.Time, window time.Duration) bool {
	return !now.IsZero() && !op.created.IsZero() && !now.Before(op.created.Add(window))
}

func (r *Registry) sweepOperations(now time.Time) {
	for r.first < len(r.order) {
		entry := r.order[r.first]
		if r.operations[entry.key] == entry.op {
			if !r.expired(entry.op, now, r.limits.Retention) {
				break
			}
			r.drop(entry.key, entry.op)
		}
		r.order[r.first] = ordered{}
		r.first++
	}
	if r.first > 0 && r.first*2 >= len(r.order) {
		r.order = append([]ordered(nil), r.order[r.first:]...)
		r.first = 0
	}
}

type verdict int

const (
	reserving verdict = iota
	conflicting
	lapsed
	replaying
	full
	inflight
)

type flight struct {
	object   string
	applying string
	release  func()
	key      operationKey
	op       *operation
}

type lockKey struct {
	workspace string
	object    string
}

type lockEntry struct {
	whole  sync.RWMutex
	single sync.Mutex
	refs   int
}

func (r *Registry) entry(key lockKey) *lockEntry {
	found := r.locks[key]
	if found == nil {
		found = &lockEntry{}
		r.locks[key] = found
	}
	found.refs++
	return found
}

func (r *Registry) unref(key lockKey) {
	if r.locks[key].refs--; r.locks[key].refs == 0 {
		delete(r.locks, key)
	}
}

func (r *Registry) acquire(workspace, object string) func() {
	whole, single := lockKey{workspace, ""}, lockKey{workspace, object}
	r.mu.Lock()
	space := r.entry(whole)
	var one *lockEntry
	if object != "" {
		one = r.entry(single)
	}
	r.mu.Unlock()
	if one == nil {
		space.whole.Lock()
	} else {
		space.whole.RLock()
		one.single.Lock()
	}
	return func() {
		if one == nil {
			space.whole.Unlock()
		} else {
			one.single.Unlock()
			space.whole.RUnlock()
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		r.unref(whole)
		if one != nil {
			r.unref(single)
		}
	}
}

func (r *Registry) lookup(key operationKey, who holder, digest [32]byte, now time.Time) (verdict, Result, *operation) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sweepOperations(now)
	stored := r.operations[key]
	switch {
	case stored != nil && (stored.workspace != who.workspace || stored.user != who.user || stored.digest != digest):
		return conflicting, Result{}, nil
	case stored != nil && stored.pending:
		return inflight, Result{}, nil
	case stored != nil && r.expired(stored, now, r.limits.Replay):
		return lapsed, Result{}, nil
	case stored != nil:
		return replaying, detached(stored.result), nil
	case len(r.operations) >= r.limits.Operations || !r.stored.room(who, r.limits, r.limits.UserOperations, r.limits.WorkspaceOperations):
		return full, Result{}, nil
	}
	op := &operation{workspace: who.workspace, user: who.user, digest: digest, pending: true}
	r.operations[key] = op
	r.stored.add(who)
	return reserving, Result{}, op
}

func (r *Registry) complete(key operationKey, op *operation, result Result, now, reservedAt time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	created := now
	if created.IsZero() {
		created = reservedAt
	}
	if created.IsZero() {
		created = r.lastKept
	}
	if !created.IsZero() {
		if created.Before(r.lastCreated) {
			created = r.lastCreated
		}
		r.lastCreated = created
		for index := len(r.order) - 1; index >= r.first && r.order[index].op.created.IsZero(); index-- {
			r.order[index].op.created = created
		}
	}
	op.result, op.pending, op.created = detached(result), false, created
	r.order = append(r.order, ordered{key: key, op: op})
}

func (r *Registry) abandon(key operationKey, op *operation) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if op.pending {
		r.drop(key, op)
	}
}
