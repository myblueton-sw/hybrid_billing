package claims

type Refusal struct {
	Application string
	Reason      string
}

func (r *Registry) Confirm(workspace string, ids []string, actor string) ([]ApplicationView, Refusal, error) {
	if len(ids) > r.limits.Batch {
		return nil, Refusal{}, ErrLimit
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !validID(id) || seen[id] {
			return nil, Refusal{}, ErrInvalid
		}
		seen[id] = true
	}
	if !validID(workspace) || !validID(actor) || len(ids) == 0 {
		return nil, Refusal{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	apps := make([]*application, 0, len(ids))
	for _, id := range ids {
		if s == nil || s.applications[id] == nil {
			return nil, Refusal{}, ErrNotFound
		}
		apps = append(apps, s.applications[id])
	}
	for _, app := range apps {
		switch r.stateOf(app, now) {
		case StateReserved, StateConfirmed, StateReversed:
		default:
			return nil, Refusal{Application: app.a.ID}, ErrRejected
		}
	}
	for _, app := range apps {
		if app.state != StateReserved {
			continue
		}
		reason := eligibilityReason(s.contracts[app.a.Scope.Contract], app.a)
		if reason == "" && app.a.Mode == ModeSource {
			src := s.sources[app.a.Source]
			if src.quantity != app.a.CapQuantity || src.amount != app.a.CapAmount || src.currency != app.a.Currency {
				reason = ReasonCapacityMismatch
			}
		}
		if reason != "" {
			return nil, Refusal{Application: app.a.ID, Reason: reason}, ErrRejected
		}
	}
	out := make([]ApplicationView, 0, len(apps))
	for _, app := range apps {
		if app.state == StateReserved {
			app.state, app.confirmedBy, app.confirmedAt = StateConfirmed, actor, now
		}
		out = append(out, r.view(app, now))
	}
	return out, Refusal{}, nil
}

func (r *Registry) Release(workspace, id, actor string) (ApplicationView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return ApplicationView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.applications[id] == nil {
		return ApplicationView{}, ErrNotFound
	}
	app := s.applications[id]
	switch r.stateOf(app, now) {
	case StateReleased:
		return r.view(app, now), nil
	case StateReserved:
		app.state, app.releasedBy, app.releasedAt = StateReleased, actor, now
		return r.view(app, now), nil
	}
	return ApplicationView{}, ErrRejected
}

func (r *Registry) reversalFits(app *application, rev Reversal) bool {
	if app.a.Mode == ModeInterval {
		return rev.Quantity == app.a.Quantity && rev.Amount == app.a.Amount
	}
	quantity, amount := r.reversed(app)
	quantity, amount = quantity.Plus(r.value(rev.Quantity)), amount.Plus(r.value(rev.Amount))
	return !r.value(app.a.Quantity).Less(quantity) && !r.value(app.a.Amount).Less(amount)
}

func viewReversal(rev *reversal) ReversalView {
	return ReversalView{Reversal: rev.r, State: rev.state, Author: rev.author, Approver: rev.approver, Digest: rev.digest, WrittenAt: rev.writtenAt, ApprovedAt: rev.approvedAt}
}

func (r *Registry) Reverse(workspace string, rev Reversal, actor string) (ReversalView, error) {
	quantity, okQ := r.amount(rev.Quantity)
	amount, okA := r.amount(rev.Amount)
	if !validID(workspace) || !validID(actor) || !validID(rev.ID) || !validID(rev.Application) || !okQ || !okA {
		return ReversalView{}, ErrInvalid
	}
	rev.Quantity, rev.Amount = quantity, amount
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.applications[rev.Application] == nil {
		return ReversalView{}, ErrNotFound
	}
	if existing := s.reversals[rev.ID]; existing != nil {
		if existing.r != rev {
			return ReversalView{}, ErrConflict
		}
		return viewReversal(existing), nil
	}
	app := s.applications[rev.Application]
	if r.stateOf(app, now) != StateConfirmed || !r.reversalFits(app, rev) {
		return ReversalView{}, ErrRejected
	}
	if len(app.reversals) >= r.limits.Reversals {
		return ReversalView{}, ErrLimit
	}
	stored := &reversal{r: rev, state: ReversalPending, author: actor, writtenAt: now, digest: reversalDigest(workspace, rev)}
	s.reversals[rev.ID] = stored
	app.reversals = append(app.reversals, stored)
	return viewReversal(stored), nil
}

func (r *Registry) ApproveReversal(workspace, id, approver string) (ReversalView, error) {
	if !validID(workspace) || !validID(id) || !validID(approver) {
		return ReversalView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.reversals[id] == nil {
		return ReversalView{}, ErrNotFound
	}
	rev := s.reversals[id]
	if rev.approver != "" {
		if rev.approver != approver {
			return ReversalView{}, ErrConflict
		}
		return viewReversal(rev), nil
	}
	app := s.applications[rev.r.Application]
	if approver == rev.author || r.stateOf(app, now) != StateConfirmed || !r.reversalFits(app, rev.r) {
		return ReversalView{}, ErrRejected
	}
	rev.state, rev.approver, rev.approvedAt = ReversalEffective, approver, now
	return viewReversal(rev), nil
}
