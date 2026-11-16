package organizations

import "sort"

func (r *Registry) Parties(workspaceID string) ([]Party, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.workspaces[workspaceID]; !ok {
		return nil, ErrNotFound
	}
	out := []Party{}
	for key, found := range r.parties {
		if key.workspace == workspaceID {
			out = append(out, found.value)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out, nil
}

func (r *Registry) BillingProfiles(workspaceID string) ([]BillingProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.workspaces[workspaceID]; !ok {
		return nil, ErrNotFound
	}
	out := []BillingProfile{}
	for key, found := range r.profiles {
		if key.workspace == workspaceID {
			out = append(out, found.value)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out, nil
}
