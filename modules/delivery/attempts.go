package delivery

var results = map[string]bool{ResultAccepted: true, ResultRejected: true, ResultUnknown: true}

func (r *Registry) Begin(workspace, id string, actor Actor) (Start, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return Start{}, ErrInvalid
	}
	if err := r.worker(actor); err != nil {
		return Start{}, err
	}
	s, d, err := r.lookup(workspace, id)
	if err != nil {
		return Start{}, err
	}
	if d.running > 0 {
		a := &d.view.Attempts[d.running-1]
		if now.Sub(a.Started) < r.limits.Lease {
			return Start{}, refuse(ReasonAttemptRunning)
		}
		a.Superseded = true
		d.running = 0
		d.view.State = StateWaiting
		s.mark(d)
	}
	c := s.channels[d.view.Delivery.Channel]
	switch {
	case d.view.State != StateWaiting:
		return Start{}, refuse(ReasonNotWaiting)
	case d.view.Approver == "" && (!c.view.Open || d.view.Resends != ""):
		return Start{}, refuse(ReasonNotApproved)
	}
	if len(d.view.Attempts) >= r.limits.Attempts {
		s.finish(d, StateExhausted)
		return Start{}, ErrLimit
	}
	p, err := r.validity(workspace, d.view.Delivery.Package)
	if err != nil {
		return Start{}, err
	}
	if !p.Live || p.Recipient != c.view.Channel.Recipient {
		s.finish(d, StateStopped)
		d.view.Reason = p.Reason
		if p.Live {
			d.view.Reason = ReasonRecipientMismatch
		}
		return Start{}, refuse(ReasonPackageNotLive)
	}
	token, err := r.token()
	if err != nil {
		return Start{}, err
	}
	if _, taken := s.tokens[token]; taken {
		return Start{}, refuse(ReasonRandomUnavailable)
	}
	duplicate := s.uncertain[d.key] || d.view.Resends != ""
	d.view.Attempts = append(d.view.Attempts, Attempt{Number: len(d.view.Attempts) + 1, Actor: actor.ID, Kind: actor.Kind, Started: now, PossibleDuplicate: duplicate})
	d.running = len(d.view.Attempts)
	d.view.State = StateInFlight
	s.tokens[token] = ticket{delivery: d, index: d.running - 1}
	return Start{Delivery: id, Token: token, Package: d.view.Delivery.Package, Recipient: d.view.Recipient, FileDigest: d.view.FileDigest,
		Channel: d.view.Delivery.Channel, Number: d.running, PossibleDuplicate: duplicate}, nil
}

func (r *Registry) Report(workspace, id, token string, o Outcome, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || len(token) != 32 || !results[o.Result] || !validID(o.Reference) || !actor.valid() {
		return View{}, ErrInvalid
	}
	if err := r.worker(actor); err != nil {
		return View{}, err
	}
	s, d, err := r.lookup(workspace, id)
	if err != nil {
		return View{}, err
	}
	t, ok := s.tokens[token]
	if !ok || t.delivery != d {
		return View{}, ErrNotFound
	}
	a := &d.view.Attempts[t.index]
	if a.Result != "" {
		if a.Result == o.Result && a.Reference == o.Reference && a.Reporter == actor.ID {
			return viewOf(d), nil
		}
		return View{}, ErrConflict
	}
	current := d.running == t.index+1
	a.Result, a.Reference, a.Reporter, a.Reported, a.Late = o.Result, o.Reference, actor.ID, now, !current
	if current {
		d.running = 0
		d.view.State = StateWaiting
	}
	switch o.Result {
	case ResultUnknown:
		s.mark(d)
	case ResultAccepted:
		current := s.byKey[d.key]
		if t.index+1 < len(d.view.Attempts) || current != nil && current != d {
			s.mark(d)
		}
		if current == nil {
			s.byKey[d.key] = d
		}
		if d.view.State != StateConfirmed && d.view.State != StateAccepted {
			s.finish(d, StateAccepted)
			d.running = 0
		}
	}
	return viewOf(d), nil
}

func (r *Registry) Confirm(workspace, id, receipt string, actor Actor) (View, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(receipt) || !actor.valid() {
		return View{}, ErrInvalid
	}
	if err := r.worker(actor); err != nil {
		return View{}, err
	}
	_, d, err := r.lookup(workspace, id)
	if err != nil {
		return View{}, err
	}
	switch d.view.State {
	case StateConfirmed:
		if d.view.Receipt == receipt && d.view.Confirmer == actor.ID {
			return viewOf(d), nil
		}
		return View{}, ErrConflict
	case StateAccepted:
		d.view.State, d.view.Receipt, d.view.Confirmer = StateConfirmed, receipt, actor.ID
		return viewOf(d), nil
	}
	return View{}, refuse(ReasonNotAccepted)
}

func (r *Registry) Of(workspace, id string) (View, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return View{}, ErrInvalid
	}
	_, d, err := r.lookup(workspace, id)
	if err != nil {
		return View{}, err
	}
	return viewOf(d), nil
}
