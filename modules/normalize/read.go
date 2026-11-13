package normalize

func copied(v *Verdict) *Verdict {
	if v == nil {
		return nil
	}
	return &Verdict{Result: v.Result, Reasons: append([]string(nil), v.Reasons...)}
}

func (r *Registry) Run(workspaceID, id string) (RunView, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return RunView{}, ErrInvalid
	}
	_, held := r.find(workspaceID, id)
	if held == nil {
		return RunView{}, ErrNotFound
	}
	view := RunView{Spec: held.spec, State: held.state, Checked: held.checked, Normal: held.counts[normal], Errors: held.counts[inError], Quarantined: held.counts[quarantined],
		Format: copied(held.format), Billing: copied(held.billing), FindingsTotal: held.findingsTotal, Unused: Listed{Total: held.names.Total, Items: append([]string(nil), held.names.Items...)},
		ProfileID: held.profileID, ProfileRevision: held.profileRevision, AbandonReason: held.abandon, DiscardReason: held.discard, Contradiction: held.contradiction, Opened: held.opened, Changed: held.changed}
	if held.spec.Window != nil {
		window := *held.spec.Window
		view.Spec.Window = &window
	}
	view.Spec.Expected = map[string]map[string]string{}
	for field, sums := range held.spec.Expected {
		view.Spec.Expected[field] = map[string]string{}
		for currency, sum := range sums {
			view.Spec.Expected[field][currency] = sum
		}
	}
	if held.sums != nil {
		view.Sums = map[string]map[string]string{}
		for field, sums := range held.sums {
			view.Sums[field] = map[string]string{}
			for currency, sum := range sums {
				view.Sums[field][currency] = sum
			}
		}
	}
	return view, nil
}

func (r *Registry) closed(workspaceID, id string, after, limit int) (*run, error) {
	if !validID(workspaceID) || !validID(id) || after < 0 || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	_, held := r.find(workspaceID, id)
	if held == nil {
		return nil, ErrNotFound
	}
	if held.state != Closed {
		return nil, ErrRejected
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

func (r *Registry) Rows(workspaceID, id string, after, limit int) (RowsPage, error) {
	defer r.leave()
	if !r.enter() {
		return RowsPage{}, ErrInvalid
	}
	held, err := r.closed(workspaceID, id, after, limit)
	if err != nil {
		return RowsPage{}, err
	}
	page := RowsPage{Format: *copied(held.format), Billing: *copied(held.billing)}
	fields := held.revision.target.stated.Fields
	from, to := span(len(held.rows), after, limit)
	for _, made := range held.rows[from:to] {
		row := OutRow{Chunk: made.chunk, Position: made.position, Key: made.key, Cells: map[string]OutCell{}}
		for index, cell := range made.cells {
			row.Cells[fields[index].Name] = cell
		}
		for _, index := range made.lacking {
			row.Lacking = append(row.Lacking, fields[index].Name)
		}
		page.Rows = append(page.Rows, row)
	}
	return page, nil
}

func (r *Registry) Findings(workspaceID, id string, after, limit int) ([]Finding, error) {
	defer r.leave()
	if !r.enter() {
		return nil, ErrInvalid
	}
	held, err := r.closed(workspaceID, id, after, limit)
	if err != nil {
		return nil, err
	}
	from, to := span(len(held.findings), after, limit)
	return append([]Finding(nil), held.findings[from:to]...), nil
}

func (r *Registry) Quarantined(workspaceID, id string, after, limit int) ([]QuarantinedRow, error) {
	defer r.leave()
	if !r.enter() {
		return nil, ErrInvalid
	}
	held, err := r.closed(workspaceID, id, after, limit)
	if err != nil {
		return nil, err
	}
	from, to := span(len(held.apart), after, limit)
	return append([]QuarantinedRow(nil), held.apart[from:to]...), nil
}
