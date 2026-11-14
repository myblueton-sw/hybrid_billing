package receivables

import "time"

func termForm(t Term) bool {
	return validID(t.ID) && t.Revision >= 1 && t.Revision <= 1000000 && t.NetDays >= 0 && modes[t.Mode]
}

func (r *Registry) RecordTerm(workspace string, t Term, actor string) (TermView, error) {
	if !validID(workspace) || !validID(actor) || !termForm(t) {
		return TermView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	key := [2]string{t.ID, itoa(t.Revision)}
	if s := r.workspaces[workspace]; s != nil {
		if old := s.terms[key]; old != nil {
			if old.view.Term != t || old.view.Author != actor {
				return TermView{}, ErrConflict
			}
			return old.view, nil
		}
	}
	if t.NetDays > r.limits.NetDays {
		return TermView{}, ErrLimit
	}
	s, err := r.open(workspace)
	if err != nil {
		return TermView{}, err
	}
	if len(s.terms) >= r.limits.Terms {
		return TermView{}, ErrLimit
	}
	made := &term{view: TermView{Term: t, State: StatePending, Author: actor, Written: now, Digest: termDigest(workspace, t, actor)}}
	s.terms[key] = made
	return made.view, nil
}

func (r *Registry) ApproveTerm(workspace, id string, revision int, actor string) (TermView, error) {
	if !validID(workspace) || !validID(id) || revision < 1 || !validID(actor) {
		return TermView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.terms[[2]string{id, itoa(revision)}] == nil {
		return TermView{}, ErrNotFound
	}
	t := s.terms[[2]string{id, itoa(revision)}]
	if t.view.State == StateApproved && t.view.Approver == actor {
		return t.view, nil
	}
	if t.view.State != StatePending || t.view.Author == actor || t.view.Term.Mode != ModeCalendar {
		return TermView{}, ErrRejected
	}
	t.view.State, t.view.Approver, t.view.ApprovedAt = StateApproved, actor, now
	return t.view, nil
}

func (r *Registry) Term(workspace, id string, revision int) (TermView, error) {
	if !validID(workspace) || !validID(id) || revision < 1 {
		return TermView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.terms[[2]string{id, itoa(revision)}] == nil {
		return TermView{}, ErrNotFound
	}
	return s.terms[[2]string{id, itoa(revision)}].view, nil
}

func bindingForm(b Binding) bool {
	return validID(b.ID) && validID(b.Contract) && validID(b.Term) && b.Revision >= 1 && zoneForm(b.Zone) && validDate(b.From) &&
		(b.To == "" || validDate(b.To) && b.To >= b.From) && bindReasons[b.Reason]
}

func viewBinding(b *binding) BindingView {
	v := b.view
	v.Ends = append([]EndView(nil), b.view.Ends...)
	return v
}

func end(b *binding) string {
	if b.view.Binding.To == "" {
		return "9999-12-31"
	}
	return b.view.Binding.To
}

func (b *binding) covers(date string) bool { return b.view.Binding.From <= date && date <= end(b) }

func (r *Registry) Bind(workspace string, b Binding, actor string) (BindingView, error) {
	if !validID(workspace) || !validID(actor) || !bindingForm(b) {
		return BindingView{}, ErrInvalid
	}
	location, err := r.zones(b.Zone)
	if err != nil || location == nil {
		return BindingView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.terms[[2]string{b.Term, itoa(b.Revision)}] == nil {
		return BindingView{}, ErrNotFound
	}
	if old := s.bindings[b.ID]; old != nil {
		if old.original != b || old.view.Author != actor {
			return BindingView{}, ErrConflict
		}
		return viewBinding(old), nil
	}
	for _, other := range s.contracts[b.Contract] {
		if other.view.State == StateWithdrawn {
			continue
		}
		if other.view.Binding.Zone != b.Zone {
			return BindingView{}, refuse(ReasonZoneMismatch)
		}
	}
	if len(s.contracts[b.Contract]) >= r.limits.Bindings {
		return BindingView{}, ErrLimit
	}
	made := &binding{original: b, view: BindingView{Binding: b, State: StatePending, Author: actor, Written: now, Digest: bindingDigest(workspace, b, actor)}, location: location}
	s.bindings[b.ID] = made
	s.contracts[b.Contract] = append(s.contracts[b.Contract], made)
	return viewBinding(made), nil
}

func overlaps(a, b *binding) bool {
	return a.view.Binding.From <= end(b) && b.view.Binding.From <= end(a)
}

func (r *Registry) ApproveBinding(workspace, id, actor string) (BindingView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return BindingView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.bindings[id] == nil {
		return BindingView{}, ErrNotFound
	}
	b := s.bindings[id]
	if b.view.State == StateApproved && b.view.Approver == actor {
		return viewBinding(b), nil
	}
	if b.view.State != StatePending || b.view.Author == actor {
		return BindingView{}, ErrRejected
	}
	if t := s.terms[[2]string{b.view.Binding.Term, itoa(b.view.Binding.Revision)}]; t.view.State != StateApproved {
		return BindingView{}, ErrRejected
	}
	for _, other := range s.contracts[b.view.Binding.Contract] {
		if other != b && other.view.State != StateWithdrawn && overlaps(other, b) {
			return BindingView{}, ErrRejected
		}
	}
	b.view.State, b.view.Approver, b.view.ApprovedAt = StateApproved, actor, now
	return viewBinding(b), nil
}

func (r *Registry) WithdrawBinding(workspace, id, actor string) (BindingView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return BindingView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.bindings[id] == nil {
		return BindingView{}, ErrNotFound
	}
	b := s.bindings[id]
	if b.view.State == StateWithdrawn && b.view.WithdrawnBy == actor {
		return viewBinding(b), nil
	}
	if b.view.State != StatePending {
		return BindingView{}, ErrRejected
	}
	b.view.State, b.view.WithdrawnBy = StateWithdrawn, actor
	return viewBinding(b), nil
}

func (b *binding) lastIssue() string {
	last := ""
	for _, d := range b.issued {
		if d.view.Snapshot.IssueDate > last {
			last = d.view.Snapshot.IssueDate
		}
	}
	return last
}

func (r *Registry) EndBinding(workspace, id, to, actor string) (BindingView, error) {
	if !validID(workspace) || !validID(id) || !validDate(to) || !validID(actor) {
		return BindingView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.bindings[id] == nil {
		return BindingView{}, ErrNotFound
	}
	b := s.bindings[id]
	if b.view.PendingEnd == to && b.view.EndAuthor == actor {
		return viewBinding(b), nil
	}
	if b.view.State != StateApproved || b.view.PendingEnd != "" || to < b.view.Binding.From || to >= end(b) || b.lastIssue() > to {
		return BindingView{}, ErrRejected
	}
	if len(b.view.Ends) >= r.limits.Bindings {
		return BindingView{}, ErrLimit
	}
	b.view.PendingEnd, b.view.EndAuthor = to, actor
	return viewBinding(b), nil
}

func (r *Registry) ApproveEnd(workspace, id, actor string) (BindingView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return BindingView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.bindings[id] == nil {
		return BindingView{}, ErrNotFound
	}
	b := s.bindings[id]
	if n := len(b.view.Ends); b.view.PendingEnd == "" && n > 0 && b.view.Ends[n-1].Approver == actor && b.view.Ends[n-1].To == b.view.Binding.To {
		return viewBinding(b), nil
	}
	if b.view.PendingEnd == "" || b.view.EndAuthor == actor || b.lastIssue() > b.view.PendingEnd {
		return BindingView{}, ErrRejected
	}
	b.view.Ends = append(b.view.Ends, EndView{From: b.view.Binding.To, To: b.view.PendingEnd, Author: b.view.EndAuthor, Approver: actor, At: now})
	b.view.Binding.To, b.view.PendingEnd, b.view.EndAuthor = b.view.PendingEnd, "", ""
	return viewBinding(b), nil
}

func (r *Registry) BindingOf(workspace, id string) (BindingView, error) {
	if !validID(workspace) || !validID(id) {
		return BindingView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.bindings[id] == nil {
		return BindingView{}, ErrNotFound
	}
	return viewBinding(s.bindings[id]), nil
}

func (r *Registry) cover(s *space, contract, date string) (*binding, error) {
	var approved, pending []*binding
	for _, b := range s.contracts[contract] {
		if !b.covers(date) {
			continue
		}
		switch b.view.State {
		case StateApproved:
			approved = append(approved, b)
		case StatePending:
			pending = append(pending, b)
		}
	}
	switch {
	case len(approved) >= 1 && len(approved)+len(pending) >= 2:
		return nil, refuse(ReasonTermConflict)
	case len(approved) == 1:
		return approved[0], nil
	case len(pending) >= 1:
		return nil, refuse(ReasonTermPending)
	}
	return nil, refuse(ReasonTermMissing)
}

func (r *Registry) zoneOf(s *space, contract string) *time.Location {
	for _, b := range s.contracts[contract] {
		if b.view.State != StateWithdrawn {
			return b.location
		}
	}
	return nil
}

func (r *Registry) Estimate(workspace, contract, date string) (Estimate, error) {
	if !validID(workspace) || !validID(contract) || !validDate(date) {
		return Estimate{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return Estimate{}, refuse(ReasonTermMissing)
	}
	b, err := r.cover(s, contract, date)
	if err != nil {
		return Estimate{}, err
	}
	t := s.terms[[2]string{b.view.Binding.Term, itoa(b.view.Binding.Revision)}]
	return Estimate{Contract: contract, Binding: b.view.Binding.ID, Date: date, Due: addDays(date, t.view.Term.NetDays)}, nil
}
