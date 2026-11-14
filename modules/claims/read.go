package claims

import "sort"

func (r *Registry) Application(workspace, id string) (ApplicationView, error) {
	if !validID(workspace) || !validID(id) {
		return ApplicationView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.applications[id] == nil {
		return ApplicationView{}, ErrNotFound
	}
	return r.view(s.applications[id], now), nil
}

func (r *Registry) Coverage(workspace string, scope Scope) ([]ApplicationView, error) {
	if !validID(workspace) || !validID(scope.Contract) || !validID(scope.Component) || !validID(scope.Instance) {
		return nil, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || len(s.scopes[scope]) == 0 {
		return nil, ErrNotFound
	}
	out := []ApplicationView{}
	for _, app := range s.scopes[scope] {
		if r.active(app, now) {
			out = append(out, r.view(app, now))
		}
	}
	sort.Slice(out, func(a, b int) bool {
		x, y := out[a].Application, out[b].Application
		return x.Start < y.Start || x.Start == y.Start && x.ID < y.ID
	})
	return out, nil
}

func (r *Registry) Source(workspace, id string) (SourceView, error) {
	if !validID(workspace) || !validID(id) {
		return SourceView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.sources[id] == nil {
		return SourceView{}, ErrNotFound
	}
	src := s.sources[id]
	out := SourceView{Source: id, Quantity: src.quantity, Amount: src.amount, Currency: src.currency, Remaining: []Remaining{}}
	usedQ, usedA := r.value("0"), r.value("0")
	for _, app := range src.apps {
		if !r.active(app, now) {
			continue
		}
		q, m := r.remaining(app)
		usedQ, usedA = usedQ.Plus(q), usedA.Plus(m)
		qt, _ := format(q)
		mt, _ := format(m)
		out.Remaining = append(out.Remaining, Remaining{Application: app.a.ID, Scope: app.a.Scope, Quantity: qt, Amount: mt})
	}
	sort.Slice(out.Remaining, func(a, b int) bool { return out.Remaining[a].Application < out.Remaining[b].Application })
	out.UsedQuantity, _ = format(usedQ)
	out.UsedAmount, _ = format(usedA)
	return out, nil
}

func (r *Registry) Reversals(workspace, application string) ([]ReversalView, error) {
	if !validID(workspace) || !validID(application) {
		return nil, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.applications[application] == nil {
		return nil, ErrNotFound
	}
	out := []ReversalView{}
	for _, rev := range s.applications[application].reversals {
		out = append(out, viewReversal(rev))
	}
	return out, nil
}
