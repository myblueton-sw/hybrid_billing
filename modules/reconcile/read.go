package reconcile

import "sort"

func (r *Registry) outside(held *run) bool {
	if !held.internal.fits(r.profile) || !held.explained.fits(r.profile) {
		return true
	}
	for _, sum := range held.sums {
		if !sum.fits(r.profile) {
			return true
		}
	}
	return false
}

func (r *Registry) Readiness(workspaceID, id string) (Readiness, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return Readiness{}, ErrInvalid
	}
	ws, held := r.find(workspaceID, id)
	if held == nil {
		return Readiness{}, ErrNotFound
	}
	out := Readiness{Verdict: Ready, Cost: Undetermined}
	switch {
	case held.state == Voided:
		out.Reasons = []string{IsVoided}
	case held.spec.Basis == InternalCost:
		out.Cost = Declared
	case held.spec.Basis == IndependentPrice:
		if !held.spec.QuantityComplete {
			out.Reasons = []string{QuantityIncomplete}
		}
		if held.cost != nil {
			out.Cost = Declared
		}
	default:
		if held.state == Open {
			out.Reasons = append(out.Reasons, NotClosed)
		} else if held.verdict != Explained {
			out.Reasons = append(out.Reasons, UnexplainedResidue)
		}
		if ws.superseded(held) {
			out.Reasons = append(out.Reasons, Superseded)
		}
		if held.state == Closed && r.outside(held) {
			out.Reasons = append(out.Reasons, SumOutsideProfile)
		}
		if len(out.Reasons) == 0 {
			out.Cost = Determined
		}
	}
	if len(out.Reasons) > 0 {
		out.Verdict = NotReady
	}
	return out, nil
}

func (r *Registry) Run(workspaceID, id string) (RunView, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return RunView{}, ErrInvalid
	}
	ws, held := r.find(workspaceID, id)
	if held == nil {
		return RunView{}, ErrNotFound
	}
	view := RunView{Spec: held.spec, State: held.state, WasClosed: held.wasClosed, Superseded: ws.superseded(held), Posted: len(held.parts), Lines: len(held.lines),
		Verdict: held.verdict, VoidReason: held.reason, Opened: held.opened, Changed: held.changed}
	if held.residual != nil {
		view.Internal, view.Explained, view.Residual = held.internal.text(), held.explained.text(), held.residual.text()
		for _, category := range categories {
			if held.counts[category] > 0 {
				view.Categories = append(view.Categories, CategorySum{Category: category, Lines: held.counts[category], Sum: held.sums[category].text()})
			}
		}
	}
	return view, nil
}

func (r *Registry) page(workspaceID, id string, after, limit int) (*run, error) {
	if !validID(workspaceID) || !validID(id) || after < 0 || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	_, held := r.find(workspaceID, id)
	if held == nil {
		return nil, ErrNotFound
	}
	if held.spec.Basis != ProviderDocument || held.state == Open {
		return nil, ErrRejected
	}
	if held.partIDs == nil {
		held.partIDs, held.lineIDs = make([]string, 0, len(held.parts)), make([]string, 0, len(held.lines))
		for name := range held.parts {
			held.partIDs = append(held.partIDs, name)
		}
		for name := range held.lines {
			held.lineIDs = append(held.lineIDs, name)
		}
		sort.Strings(held.partIDs)
		sort.Strings(held.lineIDs)
	}
	return held, nil
}

func span(length, after, limit int) (int, int) {
	if after > length {
		after = length
	}
	end := length
	if limit < end-after {
		end = after + limit
	}
	return after, end
}

func (r *Registry) Parts(workspaceID, id string, after, limit int) ([]Part, error) {
	defer r.leave()
	if !r.enter() {
		return nil, ErrInvalid
	}
	held, err := r.page(workspaceID, id, after, limit)
	if err != nil {
		return nil, err
	}
	var out []Part
	from, to := span(len(held.partIDs), after, limit)
	for _, name := range held.partIDs[from:to] {
		out = append(out, held.parts[name].stated)
	}
	return out, nil
}

func (r *Registry) Lines(workspaceID, id string, after, limit int) ([]Line, error) {
	defer r.leave()
	if !r.enter() {
		return nil, ErrInvalid
	}
	held, err := r.page(workspaceID, id, after, limit)
	if err != nil {
		return nil, err
	}
	var out []Line
	from, to := span(len(held.lineIDs), after, limit)
	for _, name := range held.lineIDs[from:to] {
		out = append(out, held.lines[name].stated)
	}
	return out, nil
}

func (r *Registry) Holder(workspaceID string, subject Subject) (string, error) {
	defer r.leave()
	key, _, formed := subject.key()
	if !r.enter() || !validID(workspaceID) || !formed {
		return "", ErrInvalid
	}
	ws := r.workspaces[workspaceID]
	if ws == nil || ws.subjects[key] == nil {
		return "", ErrNotFound
	}
	return ws.subjects[key].spec.ID, nil
}

func (r *Registry) Current(workspaceID, issuer, account, document string) (CurrentView, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(issuer) || !validID(account) || !validID(document) {
		return CurrentView{}, ErrInvalid
	}
	ws := r.workspaces[workspaceID]
	name := join(issuer, account, document)
	if ws == nil || ws.highest[name] == 0 {
		return CurrentView{}, ErrNotFound
	}
	view := CurrentView{Revision: ws.highest[name]}
	var earlier []*run
	for _, held := range ws.byDoc[name] {
		switch {
		case held.state == Voided:
		case held.spec.Revision == view.Revision:
			view.Runs = append(view.Runs, CurrentRun{ID: held.spec.ID, Currency: held.spec.Currency, State: held.state, Verdict: held.verdict})
		default:
			earlier = append(earlier, held)
		}
	}
	sort.Slice(view.Runs, func(i, j int) bool { return view.Runs[i].Currency < view.Runs[j].Currency })
	sort.Slice(earlier, func(i, j int) bool {
		if earlier[i].spec.Revision != earlier[j].spec.Revision {
			return earlier[i].spec.Revision < earlier[j].spec.Revision
		}
		return earlier[i].spec.ID < earlier[j].spec.ID
	})
	view.Superseded.Total = len(earlier)
	for _, held := range earlier {
		if len(view.Superseded.Items) == r.limits.Listed {
			break
		}
		view.Superseded.Items = append(view.Superseded.Items, held.spec.ID)
	}
	return view, nil
}
