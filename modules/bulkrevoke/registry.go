package bulkrevoke

import (
	"sort"
	"sync"
	"time"
)

type entry struct {
	v     View
	users map[string]bool
}

type Registry struct {
	mu      sync.Mutex
	limits  Limits
	clock   func() time.Time
	entries map[uint64]*entry
	next    uint64
	refs    map[string]map[string][]latestRef
	visited int
}

func New(limits Limits, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{limits: limits, clock: clock, entries: map[uint64]*entry{}, refs: map[string]map[string][]latestRef{}}, nil
}

func (r *Registry) now() time.Time { return r.clock().UTC().Round(0) }

func (r *Registry) expire(now time.Time) {
	for _, e := range r.entries {
		if (e.v.State == Draft || e.v.State == Prepared) && !now.Before(e.v.ExpiresAt) {
			e.v.State = Expired
		}
	}
}

func (r *Registry) Expire() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire(r.now())
}

func frozen(v View) bool { return v.Digest != "" }

func (r *Registry) find(workspace, id string) *entry {
	n, draft := Number(id, DraftPrefix)
	if !draft {
		n, _ = Number(id, ManifestPrefix)
	}
	if e := r.entries[n]; e != nil && e.v.Workspace == workspace && frozen(e.v) != draft {
		return e
	}
	return nil
}

func (r *Registry) visible(workspace, caller, id string) *entry {
	e := r.find(workspace, id)
	if e == nil || !frozen(e.v) && e.v.Owner != caller {
		return nil
	}
	return e
}

func (r *Registry) Visible(workspace, caller, id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire(r.now())
	return r.visible(workspace, caller, id) != nil
}

func (r *Registry) pending(workspace string) int {
	count := 0
	for _, e := range r.entries {
		if e.v.Workspace == workspace && (e.v.State == Draft || e.v.State == Prepared) {
			count++
		}
	}
	return count
}

func (r *Registry) Limits(workspace, caller string) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire(r.now())
	return r.pending(workspace), r.openDraft(workspace, caller) != nil
}

func (r *Registry) openDraft(workspace, caller string) *entry {
	for _, e := range r.entries {
		if e.v.Workspace == workspace && e.v.Owner == caller && e.v.State == Draft {
			return e
		}
	}
	return nil
}

func (r *Registry) Add(workspace, owner, draft string, users []string) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(owner) || len(users) == 0 {
		return View{}, ErrInvalid
	}
	for _, id := range users {
		if !validID(id) {
			return View{}, ErrInvalid
		}
	}
	now := r.now()
	r.expire(now)
	var e *entry
	if draft == "" {
		if r.openDraft(workspace, owner) != nil {
			return View{}, ErrOpenDraft
		}
		if r.pending(workspace) >= r.limits.Pending {
			return View{}, ErrPending
		}
		e = &entry{v: View{Number: r.next + 1, Workspace: workspace, State: Draft, Origin: OriginCommand, Owner: owner, Selection: Explicit}, users: map[string]bool{}}
	} else if e = r.visible(workspace, owner, draft); e == nil || e.v.State != Draft {
		return View{}, ErrNotFound
	} else if e.v.Selection != Explicit {
		return View{}, ErrState
	}
	added, repeated := map[string]bool{}, 0
	for _, id := range users {
		if e.users[id] || added[id] {
			repeated++
			continue
		}
		added[id] = true
	}
	if len(e.users)+len(added) > r.limits.MaxTargets {
		return View{}, ErrTooMany
	}
	if draft == "" {
		r.next++
		r.entries[e.v.Number] = e
	}
	for id := range added {
		e.users[id] = true
	}
	e.v.ID, e.v.Count, e.v.Duplicates, e.v.Updated, e.v.ExpiresAt = DraftID(e.v.Number), len(e.users), e.v.Duplicates+repeated, now, now.Add(r.limits.Lifetime)
	return copyView(e.v), nil
}

func (r *Registry) AddMatching(workspace, owner string, terms []FilterTerm, users []string, fixedAt time.Time, revision string) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(owner) || !validTerms(terms) || fixedAt.IsZero() || revision == "" || len(revision) > maxIDBytes {
		return View{}, ErrInvalid
	}
	if len(users) == 0 {
		return View{}, ErrEmpty
	}
	if len(users) > r.limits.MaxMatching {
		return View{}, ErrTooMany
	}
	set := make(map[string]bool, len(users))
	for _, id := range users {
		if !validID(id) || set[id] {
			return View{}, ErrInvalid
		}
		set[id] = true
	}
	now := r.now()
	r.expire(now)
	if r.openDraft(workspace, owner) != nil {
		return View{}, ErrOpenDraft
	}
	if r.pending(workspace) >= r.limits.Pending {
		return View{}, ErrPending
	}
	r.next++
	e := &entry{v: View{Number: r.next, ID: DraftID(r.next), Workspace: workspace, State: Draft, Origin: OriginCommand, Owner: owner, Selection: Filter,
		Filter: append([]FilterTerm(nil), terms...), Matched: len(users), FixedAt: fixedAt.UTC().Round(0), FixedRevision: revision, Count: len(users), Updated: now,
		ExpiresAt: now.Add(r.limits.Lifetime)}, users: set}
	r.entries[e.v.Number] = e
	return copyView(e.v), nil
}

func (r *Registry) DraftView(workspace, owner, draft string) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire(r.now())
	e := r.visible(workspace, owner, draft)
	if e == nil || e.v.State != Draft {
		return View{}, ErrNotFound
	}
	return copyView(e.v), nil
}

func (r *Registry) DraftUsers(workspace, owner, draft string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire(r.now())
	e := r.visible(workspace, owner, draft)
	if e == nil || e.v.State != Draft {
		return nil, ErrNotFound
	}
	out := make([]string, 0, len(e.users))
	for id := range e.users {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func (r *Registry) Discard(workspace, caller, id string) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire(r.now())
	e := r.visible(workspace, caller, id)
	if e == nil {
		return View{}, ErrNotFound
	}
	if e.v.State != Draft && e.v.State != Prepared {
		return View{}, ErrState
	}
	e.v.State = Discarded
	r.prune()
	return copyView(e.v), nil
}

func (r *Registry) Get(workspace, caller, id string) (View, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire(r.now())
	e := r.visible(workspace, caller, id)
	if e == nil {
		return View{}, false
	}
	return copyView(e.v), true
}

func rank(v View) int {
	switch v.State {
	case Draft:
		return 0
	case Prepared:
		return 1
	case Executed:
		return 2
	}
	return 3
}

func (r *Registry) List(workspace, caller string) []View {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire(r.now())
	out := []View{}
	for _, e := range r.entries {
		if e.v.Workspace == workspace && (frozen(e.v) || e.v.Owner == caller) {
			v := e.v
			v.Items, v.Targets, v.Exclusions = nil, nil, nil
			v = copyView(v)
			v.Items = nil
			out = append(out, v)
		}
	}
	sort.Slice(out, func(a, b int) bool {
		if rank(out[a]) != rank(out[b]) {
			return rank(out[a]) < rank(out[b])
		}
		return out[a].Number > out[b].Number
	})
	return out
}

func (r *Registry) Highest() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.next
}

func (r *Registry) Restore(next uint64, jobs []View) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next = max(r.next, next)
	for _, v := range jobs {
		v.State, v.Origin = Executed, OriginReplay
		v.ID = ManifestID(v.Number)
		v.Counts = count(v)
		if old := r.entries[v.Number]; old != nil && old.v.State == Executed {
			r.unindex(old)
		}
		e := &entry{v: copyView(v), users: map[string]bool{}}
		r.entries[v.Number] = e
		r.index(e)
		r.next = max(r.next, v.Number)
	}
	r.prune()
}

func unfinished(v View) bool {
	if v.ReplayMissing > 0 {
		return true
	}
	for _, item := range v.Items {
		if item.Outcome == OutcomeUnprocessed {
			return true
		}
	}
	return false
}

func (r *Registry) prune() {
	executed, ended := []uint64{}, []uint64{}
	for n, e := range r.entries {
		switch {
		case e.v.State == Executed && !unfinished(e.v):
			executed = append(executed, n)
		case e.v.State == Discarded || e.v.State == Expired:
			ended = append(ended, n)
		}
	}
	for _, list := range [][]uint64{executed, ended} {
		sort.Slice(list, func(a, b int) bool { return list[a] > list[b] })
		for _, n := range list[min(len(list), r.limits.Kept):] {
			if e := r.entries[n]; e.v.State == Executed {
				r.unindex(e)
			}
			delete(r.entries, n)
		}
	}
}

func copyView(v View) View {
	v.Filter = append([]FilterTerm(nil), v.Filter...)
	v.Targets = append([]Target(nil), v.Targets...)
	v.Exclusions = append([]Exclusion(nil), v.Exclusions...)
	v.Impact.Pending = append([]PendingStore(nil), v.Impact.Pending...)
	items := make([]Item, len(v.Items))
	for i, item := range v.Items {
		item.Cleanup = append([]Cleanup(nil), item.Cleanup...)
		items[i] = item
	}
	v.Items = items
	return v
}
