package auth

import (
	"sort"
	"time"
)

type holding struct {
	bySubject   map[string][]string
	byClient    map[string][]string
	byAction    map[Action][]string
	byDelegator map[string][]string
	byDelegate  map[string][]string
	byParent    map[string][]string
}

func newHolding() holding {
	return holding{bySubject: map[string][]string{}, byClient: map[string][]string{}, byAction: map[Action][]string{}, byDelegator: map[string][]string{}, byDelegate: map[string][]string{}, byParent: map[string][]string{}}
}

func (h holding) grant(g Grant) {
	if g.SubjectID != "" {
		h.bySubject[g.SubjectID] = insertOrdered(h.bySubject[g.SubjectID], g.ID)
	} else {
		h.byClient[g.ClientID] = insertOrdered(h.byClient[g.ClientID], g.ID)
	}
	if InstallationAction(g.Terms.Action) {
		h.byAction[g.Terms.Action] = insertOrdered(h.byAction[g.Terms.Action], g.ID)
	}
}

func (h holding) delegation(d Delegation) {
	h.byDelegator[d.DelegatorID] = insertOrdered(h.byDelegator[d.DelegatorID], d.ID)
	h.byDelegate[d.DelegateID] = insertOrdered(h.byDelegate[d.DelegateID], d.ID)
	if d.ParentID != "" {
		h.byParent[d.ParentID] = insertOrdered(h.byParent[d.ParentID], d.ID)
	}
}

func (r *Registry) grantsIn(ids []string, limit int, at time.Time) ([]GrantView, bool) {
	if limit < 1 {
		return nil, false
	}
	end := min(limit, len(ids))
	out := make([]GrantView, 0, end)
	for _, id := range ids[:end] {
		out = append(out, r.grantView(id, at))
	}
	return out, end < len(ids)
}

func (r *Registry) GrantsOfSubject(subject string, limit int, at time.Time) ([]GrantView, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.grantsIn(r.held.bySubject[subject], limit, at)
}

func (r *Registry) GrantsOfClient(client string, limit int, at time.Time) ([]GrantView, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.grantsIn(r.held.byClient[client], limit, at)
}

func (r *Registry) InstallationGrants(action Action, limit int, at time.Time) ([]GrantView, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.grantsIn(r.held.byAction[action], limit, at)
}

func (r *Registry) DelegationsOfUser(user string, limit int, at time.Time) ([]DelegationView, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit < 1 {
		return nil, false
	}
	given, received := r.held.byDelegator[user], r.held.byDelegate[user]
	out := []DelegationView{}
	for i, j := 0, 0; i < len(given) || j < len(received); {
		if len(out) == limit {
			return out, true
		}
		if j == len(received) || i < len(given) && given[i] < received[j] {
			out = append(out, r.delegationView(given[i], at))
			i++
		} else {
			out = append(out, r.delegationView(received[j], at))
			j++
		}
	}
	return out, false
}

func (r *Registry) below(seeds []string, limit int) ([]string, bool) {
	seen := map[string]bool{}
	queue := []string{}
	for _, id := range seeds {
		if !seen[id] {
			seen[id] = true
			queue = append(queue, id)
		}
	}
	for at := 0; at < len(queue); at++ {
		if len(queue) > limit {
			return nil, false
		}
		for _, child := range r.held.byParent[queue[at]] {
			if !seen[child] {
				seen[child] = true
				queue = append(queue, child)
			}
		}
	}
	return queue, len(queue) <= limit
}

func (r *Registry) GrantWouldEnd(id string, limit int, at time.Time) ([]string, bool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stored, known := r.grants[id]
	if !known || limit < 1 {
		return nil, known, false
	}
	out := []string{}
	if stored.closed || stored.value.SubjectID == "" {
		return out, true, true
	}
	seeds := append(append([]string{}, r.held.byDelegator[stored.value.SubjectID]...), r.held.byDelegate[stored.value.SubjectID]...)
	candidates, within := r.below(seeds, limit)
	if !within {
		return nil, true, false
	}
	for _, candidate := range candidates {
		if d := r.delegations[candidate]; r.effective(d, at, "") && !r.effective(d, at, id) {
			out = append(out, candidate)
		}
	}
	sort.Strings(out)
	return out, true, true
}

func (r *Registry) DelegationWouldEnd(id string, limit int, at time.Time) ([]string, bool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stored, known := r.delegations[id]
	if !known || limit < 1 {
		return nil, known, false
	}
	out := []string{}
	if stored.closed {
		return out, true, true
	}
	candidates, within := r.below(r.held.byParent[id], limit)
	if !within {
		return nil, true, false
	}
	for _, candidate := range candidates {
		if r.effective(r.delegations[candidate], at, "") {
			out = append(out, candidate)
		}
	}
	sort.Strings(out)
	return out, true, true
}
