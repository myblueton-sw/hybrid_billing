package adjustments

import "strconv"

func (r *Registry) Validate(workspace, id string, c Check, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	final, ok := r.number(c.Final, true, false)
	tax, taxed := r.number(c.TaxAmount, true, false)
	if !validID(workspace) || !validID(id) || c.Revision < 1 || !validDigest(c.Digest) || !validDigest(c.Tax) || !validDigest(c.Allocation) || !ok || !taxed || !validActor(actor) {
		return View{}, ErrInvalid
	}
	d, err := r.find(workspace, id)
	if err != nil {
		return View{}, err
	}
	scale := d.view.Draft.Base.Scale
	if !fits(final, scale) || !fits(tax, scale) {
		return View{}, ErrInvalid
	}
	if known := d.view.Check; known != nil && known.Check.Revision == c.Revision {
		if known.Check == c && known.Actor == actor {
			return viewOf(d), nil
		}
		return View{}, ErrConflict
	}
	if actor.Kind == KindAssistant {
		return View{}, refuse(RefuseAINotAllowed)
	}
	if err := open(d); err != nil {
		return View{}, err
	}
	if c.Revision != d.view.Revision {
		return View{}, refuse(RefuseStaleRevision)
	}
	if c.Digest != d.view.Digest {
		return View{}, refuse(RefuseDigestChanged)
	}
	if err := d.blocked(); err != nil {
		return View{}, err
	}
	if !same(final, d.result.total.Plus(tax)) {
		return View{}, refuse(RefuseFinalMismatch)
	}
	d.view.Check = &CheckView{Check: c, Actor: actor, Recorded: now}
	d.checkers[actor.ID] = true
	d.view.Subject = subjectDigest(d)
	return viewOf(d), nil
}

func subjectDigest(d *draft) string {
	b, c := d.view.Draft.Base, d.view.Check.Check
	p := newPrinter(releaseLabel)
	p.fields(d.view.Digest, b.Run, b.Digest, strconv.FormatBool(b.Issued), b.Document, c.Tax, c.Allocation, c.TaxAmount, c.Final, show(d.result.gross), b.Currency)
	return p.hex()
}

func (d *draft) involved(a Actor) bool {
	if d.view.Opener.ID == a.ID {
		return true
	}
	for _, e := range d.entries {
		if e.view.Author.ID == a.ID || !e.view.Active && e.view.RemovedBy.ID == a.ID {
			return true
		}
	}
	return d.checkers[a.ID]
}

func sameRelease(a, b Release) bool {
	if a.Request != b.Request || a.Revision != b.Revision || a.Subject != b.Subject || a.Base != b.Base || a.Approval != b.Approval || len(a.Approvers) != len(b.Approvers) {
		return false
	}
	for index := range a.Approvers {
		if a.Approvers[index] != b.Approvers[index] {
			return false
		}
	}
	return true
}

func releaseForm(q Release) bool {
	if !validID(q.Request) || q.Revision < 1 || !validDigest(q.Subject) || !validID(q.Approval) || len(q.Approvers) == 0 || len(q.Approvers) > maxApprovers {
		return false
	}
	if !baseForm(q.Base) {
		return false
	}
	seen := map[string]bool{}
	for _, a := range q.Approvers {
		if !validActor(a) || seen[a.ID] {
			return false
		}
		seen[a.ID] = true
	}
	return true
}

func runKey(b Base) string { return b.Run + "\x00" + b.Digest }

func (r *Registry) gate(workspace, id string, q Release, actor Actor) (*draft, bool, error) {
	if !validID(workspace) || !validID(id) || !releaseForm(q) || !validActor(actor) {
		return nil, false, ErrInvalid
	}
	d, err := r.find(workspace, id)
	if err != nil {
		return nil, false, err
	}
	s := r.workspaces[workspace]
	if done := d.view.Outcome; done != nil && sameRelease(done.Release, q) && done.Actor == actor {
		return d, true, nil
	}
	if _, ok := s.requests[q.Request]; ok {
		return nil, false, ErrConflict
	}
	if owner, ok := s.used[q.Approval]; ok && owner != id {
		return nil, false, ErrConflict
	}
	if d.view.Outcome != nil {
		return nil, false, refuse(RefuseAlreadyReleased)
	}
	if actor.Kind == KindAssistant {
		return nil, false, refuse(RefuseAINotAllowed)
	}
	if d.view.State == StateWithdrawn {
		return nil, false, refuse(RefuseWithdrawn)
	}
	if q.Revision != d.view.Revision {
		return nil, false, refuse(RefuseStaleRevision)
	}
	if d.view.Check == nil || d.view.Check.Check.Revision != d.view.Revision {
		return nil, false, refuse(RefuseNotValidated)
	}
	if q.Subject != d.view.Subject {
		return nil, false, refuse(RefuseDigestChanged)
	}
	b := d.view.Draft.Base
	if q.Base != b {
		return nil, false, refuse(RefuseBaseChanged)
	}
	if _, ok := s.runs[runKey(b)]; ok {
		return nil, false, refuse(RefuseBaseConsumed)
	}
	for _, a := range q.Approvers {
		if a.Kind != KindHuman {
			return nil, false, refuse(RefuseAINotAllowed)
		}
	}
	for _, a := range q.Approvers {
		if d.involved(a) {
			return nil, false, refuse(RefuseSelfApproval)
		}
	}
	if err := d.blocked(); err != nil {
		return nil, false, err
	}
	return d, false, nil
}

func (r *Registry) Ready(workspace, id string, q Release, actor Actor) error {
	r.enter()
	defer r.leave()
	_, _, err := r.gate(workspace, id, q, actor)
	return err
}

func (r *Registry) Release(workspace, id string, q Release, actor Actor) (Outcome, error) {
	now := r.enter()
	defer r.leave()
	d, replay, err := r.gate(workspace, id, q, actor)
	if err != nil {
		return Outcome{}, err
	}
	if replay {
		return *viewOf(d).Outcome, nil
	}
	b := d.view.Draft.Base
	q.Approvers = append([]Actor(nil), q.Approvers...)
	c := d.view.Check.Check
	out := Outcome{Release: q, Disposition: DispositionNewRun, Run: b.Run, Currency: b.Currency, Scale: b.Scale, Differences: append([]Difference(nil), d.result.differences...),
		Source: show(d.result.source), Adjustment: show(d.result.adjustment), Total: show(d.result.total), Gross: show(d.result.gross),
		Digest: d.view.Digest, Subject: d.view.Subject, Tax: c.Tax, Allocation: c.Allocation, TaxAmount: c.TaxAmount, Final: c.Final, Actor: actor, Released: now}
	if b.Issued {
		out.Disposition, out.Document = DispositionCorrection, b.Document
	}
	d.view.Outcome = &out
	d.view.State = StateReleased
	s := r.workspaces[workspace]
	s.requests[q.Request], s.used[q.Approval], s.runs[runKey(b)] = id, id, id
	return *viewOf(d).Outcome, nil
}

func (r *Registry) Withdraw(workspace, id string, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validActor(actor) {
		return View{}, ErrInvalid
	}
	d, err := r.find(workspace, id)
	if err != nil {
		return View{}, err
	}
	if d.view.State == StateWithdrawn {
		return viewOf(d), nil
	}
	if actor.Kind == KindAssistant {
		return View{}, refuse(RefuseAINotAllowed)
	}
	if err := open(d); err != nil {
		return View{}, err
	}
	d.view.State, d.view.Withdrawer, d.view.Withdrawn = StateWithdrawn, actor, now
	return viewOf(d), nil
}
