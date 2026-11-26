package auth

import "time"

type holding struct {
	bySubject   map[string][]string
	byClient    map[string][]string
	byAction    map[Action][]string
	byDelegator map[string][]string
	byDelegate  map[string][]string
}

func newHolding() holding {
	return holding{bySubject: map[string][]string{}, byClient: map[string][]string{}, byAction: map[Action][]string{}, byDelegator: map[string][]string{}, byDelegate: map[string][]string{}}
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
