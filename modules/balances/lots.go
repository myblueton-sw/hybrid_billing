package balances

func (r *Registry) openingForm(o Opening) (Opening, bool) {
	amount, ok := r.amount(o.Amount, false)
	if !ok || !validID(o.ID) || !validID(o.Account) || !validID(o.PriorLedger) || !validDate(o.AsOf) || !validDigest(o.Evidence) || !validID(o.MigrationKey) {
		return Opening{}, false
	}
	if !validDate(o.IncludedFrom) || o.IncludedFrom > o.AsOf || !optionalDate(o.ExpiresOn) {
		return Opening{}, false
	}
	o.Amount = amount
	return o, true
}

func viewOpening(o *opening) OpeningView {
	return OpeningView{Opening: o.o, State: o.state, Author: o.author, Approver: o.approver, WithdrawnBy: o.withdrawnBy, Digest: o.digest,
		WrittenAt: o.writtenAt, ApprovedAt: o.approvedAt, WithdrawnAt: o.withdrawnAt}
}

func (r *Registry) RecordOpening(workspace string, o Opening, author string) (OpeningView, error) {
	canonical, formed := r.openingForm(o)
	if !validID(workspace) || !validID(author) || !formed {
		return OpeningView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	acct := r.lookup(workspace, canonical.Account)
	if acct == nil {
		return OpeningView{}, ErrNotFound
	}
	s := r.workspaces[workspace]
	if existing := s.openings[canonical.ID]; existing != nil {
		if existing.o != canonical {
			return OpeningView{}, ErrConflict
		}
		return viewOpening(existing), nil
	}
	if prior := s.migrations[canonical.MigrationKey]; prior != nil && prior.state != StateWithdrawn {
		return OpeningView{}, ErrConflict
	}
	if acct.a.Authority == AuthorityExternal || acct.opened() != nil || len(acct.increases) > 0 || len(acct.apps) > 0 || canonical.ExpiresOn != "" && canonical.ExpiresOn < today(now) {
		return OpeningView{}, ErrRejected
	}
	if s.importedBefore(canonical.PriorLedger, canonical.AsOf, now) {
		return OpeningView{}, ErrRejected
	}
	if len(acct.openings) >= r.limits.Openings {
		return OpeningView{}, ErrLimit
	}
	stored := &opening{o: canonical, acct: acct, state: StatePending, author: author, writtenAt: now, digest: openingDigest(workspace, canonical)}
	acct.openings = append(acct.openings, stored)
	s.openings[canonical.ID] = stored
	s.migrations[canonical.MigrationKey] = stored
	return viewOpening(stored), nil
}

func (r *Registry) ApproveOpening(workspace, id, approver string) (OpeningView, error) {
	if !validID(workspace) || !validID(id) || !validID(approver) {
		return OpeningView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.openings[id] == nil {
		return OpeningView{}, ErrNotFound
	}
	o := s.openings[id]
	if o.state == StateApproved {
		if o.approver != approver {
			return OpeningView{}, ErrConflict
		}
		return viewOpening(o), nil
	}
	if o.state != StatePending || approver == o.author || o.o.ExpiresOn != "" && o.o.ExpiresOn < today(now) || s.importedBefore(o.o.PriorLedger, o.o.AsOf, now) {
		return OpeningView{}, ErrRejected
	}
	o.state, o.approver, o.approvedAt = StateApproved, approver, now
	if amount := r.value(o.o.Amount); amount.Sign() > 0 {
		o.lot = &lot{number: len(o.acct.lots) + 1, source: ReverseOpening, record: id, amount: amount, remainder: amount, expiry: expiryOf(o.o.ExpiresOn)}
		o.acct.lots = append(o.acct.lots, o.lot)
	}
	return viewOpening(o), nil
}

func (r *Registry) WithdrawOpening(workspace, id, actor string) (OpeningView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return OpeningView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.openings[id] == nil {
		return OpeningView{}, ErrNotFound
	}
	o := s.openings[id]
	switch o.state {
	case StateWithdrawn:
		return viewOpening(o), nil
	case StateApproved:
		return OpeningView{}, ErrRejected
	}
	o.state, o.withdrawnBy, o.withdrawnAt = StateWithdrawn, actor, now
	return viewOpening(o), nil
}

func (r *Registry) Opening(workspace, id string) (OpeningView, error) {
	if !validID(workspace) || !validID(id) {
		return OpeningView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.openings[id] == nil {
		return OpeningView{}, ErrNotFound
	}
	return viewOpening(s.openings[id]), nil
}

func (r *Registry) increaseForm(i Increase) (Increase, bool) {
	amount, ok := r.amount(i.Amount, true)
	if !ok || increaseKinds[i.Kind] == "" || !validID(i.ID) || !validID(i.Account) || !validID(i.Reference) || !optionalDate(i.ExpiresOn) || !validImport(i.Import) {
		return Increase{}, false
	}
	if i.Kind == IncreasePromotion && i.ExpiresOn == "" {
		return Increase{}, false
	}
	i.Amount = amount
	return i, true
}

func viewIncrease(i *increase) IncreaseView {
	return IncreaseView{Increase: i.i, State: i.state, Author: i.author, Approver: i.approver, WithdrawnBy: i.withdrawnBy, Digest: i.digest,
		WrittenAt: i.writtenAt, ApprovedAt: i.approvedAt, WithdrawnAt: i.withdrawnAt}
}

func referenceKey(i Increase) [2]string { return [2]string{i.Kind, i.Reference} }

func (r *Registry) RecordIncrease(workspace string, i Increase, author string) (IncreaseView, error) {
	canonical, formed := r.increaseForm(i)
	if !validID(workspace) || !validID(author) || !formed {
		return IncreaseView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	acct := r.lookup(workspace, canonical.Account)
	if acct == nil {
		return IncreaseView{}, ErrNotFound
	}
	s := r.workspaces[workspace]
	if existing := s.increases[canonical.ID]; existing != nil {
		if existing.i != canonical {
			return IncreaseView{}, ErrConflict
		}
		return viewIncrease(existing), nil
	}
	if prior := s.references[referenceKey(canonical)]; prior != nil && prior.state != StateWithdrawn {
		return IncreaseView{}, ErrConflict
	}
	if r.imported(s, canonical.Import, now) {
		return IncreaseView{}, ErrConflict
	}
	if acct.a.Authority == AuthorityExternal || increaseKinds[canonical.Kind] != acct.a.Kind || acct.pending() || canonical.ExpiresOn != "" && canonical.ExpiresOn < today(now) {
		return IncreaseView{}, ErrRejected
	}
	if err := r.importRefused(s, acct, canonical.Import, now); err != nil {
		return IncreaseView{}, err
	}
	if len(acct.increases) >= r.limits.Increases {
		return IncreaseView{}, ErrLimit
	}
	stored := &increase{i: canonical, acct: acct, state: StatePending, author: author, writtenAt: now, digest: increaseDigest(workspace, canonical)}
	acct.increases = append(acct.increases, stored)
	s.increases[canonical.ID] = stored
	s.references[referenceKey(canonical)] = stored
	if canonical.Import != (Import{}) {
		s.importedIncreases[importKey(canonical.Import)] = stored
	}
	return viewIncrease(stored), nil
}

func (r *Registry) ApproveIncrease(workspace, id, approver string) (IncreaseView, error) {
	if !validID(workspace) || !validID(id) || !validID(approver) {
		return IncreaseView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.increases[id] == nil {
		return IncreaseView{}, ErrNotFound
	}
	i := s.increases[id]
	if i.state == StateApproved {
		if i.approver != approver {
			return IncreaseView{}, ErrConflict
		}
		return viewIncrease(i), nil
	}
	if i.state != StatePending || approver == i.author || i.i.ExpiresOn != "" && i.i.ExpiresOn < today(now) {
		return IncreaseView{}, ErrRejected
	}
	i.state, i.approver, i.approvedAt = StateApproved, approver, now
	amount := r.value(i.i.Amount)
	i.lot = &lot{number: len(i.acct.lots) + 1, source: ReverseIncrease, record: id, amount: amount, remainder: amount, expiry: expiryOf(i.i.ExpiresOn)}
	i.acct.lots = append(i.acct.lots, i.lot)
	return viewIncrease(i), nil
}

func (r *Registry) WithdrawIncrease(workspace, id, actor string) (IncreaseView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return IncreaseView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.increases[id] == nil {
		return IncreaseView{}, ErrNotFound
	}
	i := s.increases[id]
	switch i.state {
	case StateWithdrawn:
		return viewIncrease(i), nil
	case StateApproved:
		return IncreaseView{}, ErrRejected
	}
	i.state, i.withdrawnBy, i.withdrawnAt = StateWithdrawn, actor, now
	return viewIncrease(i), nil
}

func (r *Registry) Increase(workspace, id string) (IncreaseView, error) {
	if !validID(workspace) || !validID(id) {
		return IncreaseView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.increases[id] == nil {
		return IncreaseView{}, ErrNotFound
	}
	return viewIncrease(s.increases[id]), nil
}
