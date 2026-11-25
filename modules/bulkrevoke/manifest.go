package bulkrevoke

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"sort"
	"strconv"
	"time"
)

type printer struct{ sum hash.Hash }

func (p printer) fields(texts ...string) {
	for _, text := range texts {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(text)))
		p.sum.Write(size[:])
		p.sum.Write([]byte(text))
	}
}

func stamp(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.UTC().Format(time.RFC3339Nano)
}

func Digest(effect Effect, selection string, targets []Target, exclusions []Exclusion) string {
	return digest(effect, selection, nil, targets, exclusions)
}

func fixedFields(v View) []string {
	if v.Selection != Filter {
		return nil
	}
	return []string{CanonicalFilter(v.Filter), strconv.Itoa(v.Matched), stamp(v.FixedAt), v.FixedRevision}
}

func ListDigest(v View, effect Effect, targets []Target, exclusions []Exclusion) string {
	return digest(effect, v.Selection, fixedFields(v), targets, exclusions)
}

func digest(effect Effect, selection string, fixed []string, targets []Target, exclusions []Exclusion) string {
	p := printer{sum: sha256.New()}
	p.fields(digestLabel, string(effect), selection)
	p.fields(fixed...)
	p.fields(strconv.Itoa(len(targets)))
	for _, t := range sortedTargets(targets) {
		p.fields(t.User, t.Login, t.Role, stamp(t.Until), strconv.FormatUint(t.Version, 10), strconv.FormatUint(t.Epoch, 10), t.UserState, t.MembershipState,
			strconv.FormatBool(t.Administrator))
	}
	p.fields(strconv.Itoa(len(exclusions)))
	for _, x := range sortedExclusions(exclusions) {
		p.fields(x.User, x.Login, x.Reason)
	}
	return hex.EncodeToString(p.sum.Sum(nil))
}

func sortedTargets(in []Target) []Target {
	out := append([]Target(nil), in...)
	sort.Slice(out, func(a, b int) bool { return out[a].User < out[b].User })
	return out
}

func sortedExclusions(in []Exclusion) []Exclusion {
	out := append([]Exclusion(nil), in...)
	sort.Slice(out, func(a, b int) bool { return out[a].User < out[b].User })
	return out
}

func (r *Registry) Prepare(workspace, owner, draft string, effect Effect, revision string, targets []Target, exclusions []Exclusion, impact Impact) (View, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	r.expire(now)
	e := r.visible(workspace, owner, draft)
	if e == nil || e.v.State != Draft {
		return View{}, false, ErrNotFound
	}
	if !effect.Valid() || revision == "" {
		return View{}, false, ErrInvalid
	}
	listed := map[string]bool{}
	for _, t := range targets {
		if kept, in := e.users[t.User]; in && !kept {
			return View{}, false, ErrInvalid
		}
		listed[t.User] = true
	}
	for _, x := range exclusions {
		listed[x.User] = true
	}
	if len(listed) != len(e.users) || len(targets)+len(exclusions) != len(listed) {
		return View{}, false, ErrInvalid
	}
	for id := range e.users {
		if !listed[id] {
			return View{}, false, ErrInvalid
		}
	}
	if len(targets) == 0 {
		return View{}, false, ErrEmpty
	}
	digest := ListDigest(e.v, effect, targets, exclusions)
	for _, other := range r.entries {
		if other.v.Workspace == workspace && other.v.Owner == owner && other.v.State == Prepared && other.v.Effect == effect && other.v.Digest == digest {
			delete(r.entries, e.v.Number)
			return copyView(other.v), true, nil
		}
	}
	e.v.State, e.v.ID, e.v.Effect, e.v.Revision, e.v.PreparedAt, e.v.ExpiresAt = Prepared, ManifestID(e.v.Number), effect, revision, now, now.Add(r.limits.Lifetime)
	e.v.Targets, e.v.Exclusions, e.v.Impact, e.v.Digest = sortedTargets(targets), sortedExclusions(exclusions), impact, digest
	e.v.Impact.Pending = append([]PendingStore(nil), impact.Pending...)
	e.v.Counts = Counts{Targets: len(targets), Excluded: len(exclusions), Duplicates: e.v.Duplicates}
	return copyView(e.v), false, nil
}

func (r *Registry) Begin(workspace, id, digest string) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire(r.now())
	e := r.find(workspace, id)
	switch {
	case e == nil:
		return View{}, ErrNotFound
	case e.v.State != Prepared:
		return View{}, ErrState
	case e.v.Digest != digest:
		return View{}, ErrDigest
	}
	return copyView(e.v), nil
}

func (r *Registry) Execute(workspace, id, digest, executor string, generation uint64, held map[string]string) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	r.expire(now)
	e := r.find(workspace, id)
	switch {
	case !validID(executor):
		return View{}, ErrInvalid
	case e == nil:
		return View{}, ErrNotFound
	case e.v.State != Prepared:
		return View{}, ErrState
	case e.v.Digest != digest:
		return View{}, ErrDigest
	}
	items := make([]Item, 0, len(e.v.Targets)+len(e.v.Exclusions))
	for _, t := range e.v.Targets {
		item := Item{Index: len(items), Target: t, Outcome: OutcomeUnprocessed, State: Received}
		if reason, ok := held[t.User]; ok {
			item.Outcome, item.Reason, item.State = OutcomeHeld, reason, ""
		}
		items = append(items, item)
	}
	for _, x := range e.v.Exclusions {
		items = append(items, Item{Index: len(items), Target: Target{User: x.User, Login: x.Login}, Outcome: OutcomeExcluded, Reason: x.Reason})
	}
	e.v.State, e.v.Executor, e.v.ExecutedAt, e.v.DenyAt, e.v.DenyGeneration, e.v.Items = Executed, executor, now, now, generation, items
	e.v.Counts = count(e.v)
	r.index(e)
	r.prune()
	return copyView(e.v), nil
}

func (r *Registry) Exclusions(workspace, owner, id string, after, limit int) ([]Exclusion, int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.find(workspace, id)
	if e == nil || e.v.Owner != owner || !frozen(e.v) || after < 0 || limit < 1 {
		return nil, 0, false
	}
	start := min(after, len(e.v.Exclusions))
	end := min(start+limit, len(e.v.Exclusions))
	return append([]Exclusion(nil), e.v.Exclusions[start:end]...), len(e.v.Exclusions), true
}

func (r *Registry) Targets(workspace, owner, id string, after, limit int) ([]Target, int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire(r.now())
	e := r.find(workspace, id)
	if e == nil || e.v.Owner != owner || e.v.State == Draft || after < 0 || limit < 1 {
		return nil, 0, false
	}
	start := min(after, len(e.v.Targets))
	end := min(start+limit, len(e.v.Targets))
	return append([]Target(nil), e.v.Targets[start:end]...), len(e.v.Targets), true
}
