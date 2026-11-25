package protected

import "time"

func (c *change) observed(now time.Time) View {
	seen := change{view: copyView(c.view)}
	seen.settle(now)
	return seen.view
}

func (r *Registry) Peek(workspace, id string) (View, error) {
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.changes[id] == nil {
		return View{}, ErrNotFound
	}
	return s.changes[id].observed(now), nil
}

func (r *Registry) PeekTrail(workspace, subject, field string) ([]View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(subject) || !fields[field] {
		return nil, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil {
		return nil, ErrNotFound
	}
	list := s.byField[key{subject, field}]
	out := make([]View, 0, len(list))
	for _, c := range list {
		out = append(out, c.observed(now))
	}
	return out, nil
}
