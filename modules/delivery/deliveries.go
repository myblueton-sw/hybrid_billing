package delivery

import (
	"time"

	"encoding/hex"
	"io"

	"github.com/myblueton-sw/hybrid_billing/modules/disclosure"
)

func viewOf(d *delivery) View {
	v := d.view
	v.Attempts = append([]Attempt(nil), v.Attempts...)
	return v
}

func (r *Registry) validity(workspace, pkg string) (disclosure.Validity, error) {
	v, err := r.config.Packages.Validity(workspace, pkg)
	if err != nil {
		return disclosure.Validity{}, refuse(ReasonUnavailable)
	}
	return v, nil
}

func (r *Registry) Request(workspace string, d Delivery, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(d.ID) || !validID(d.Package) || !validID(d.Channel) || !actor.valid() {
		return View{}, ErrInvalid
	}
	if err := r.worker(actor); err != nil {
		return View{}, err
	}
	s, c, err := r.channel(workspace, d.Channel)
	if err != nil {
		return View{}, err
	}
	if found := s.deliveries[d.ID]; found != nil {
		if found.view.Delivery == d && found.view.Requester == actor.ID {
			return viewOf(found), nil
		}
		return View{}, ErrConflict
	}
	key := [2]string{d.Package, d.Channel}
	if found := s.byKey[key]; found != nil {
		v := viewOf(found)
		v.Duplicate = true
		return v, nil
	}
	p, err := r.validity(workspace, d.Package)
	if err != nil {
		return View{}, err
	}
	switch {
	case !p.Live:
		return View{}, refuse(ReasonPackageNotLive)
	case p.Recipient != c.view.Channel.Recipient:
		return View{}, refuse(ReasonRecipientMismatch)
	}
	return r.create(s, d, key, p, actor, now, "", "")
}

func (r *Registry) create(s *space, d Delivery, key [2]string, p disclosure.Validity, actor Actor, now time.Time, resends, reason string) (View, error) {
	if len(s.deliveries) >= r.limits.Deliveries || s.queued >= r.limits.Queue {
		return View{}, ErrLimit
	}
	made := &delivery{key: key, view: View{Delivery: d, Recipient: p.Recipient, FileDigest: p.FileDigest, Policy: p.Policy, Revision: p.Revision, Grant: p.Grant,
		State: StateWaiting, Requester: actor.ID, Requested: now, Resends: resends, ResendReason: reason, PossibleDuplicate: s.uncertain[key] || resends != ""}}
	s.deliveries[d.ID] = made
	s.byKey[key] = made
	s.queued++
	return viewOf(made), nil
}

func (r *Registry) lookup(workspace, id string) (*space, *delivery, error) {
	s := r.workspaces[workspace]
	if s == nil || s.deliveries[id] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.deliveries[id], nil
}

func (r *Registry) Approve(workspace, id string, actor Actor) (View, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return View{}, ErrInvalid
	}
	if err := r.staff(actor); err != nil {
		return View{}, err
	}
	s, d, err := r.lookup(workspace, id)
	if err != nil {
		return View{}, err
	}
	if d.view.Approver != "" {
		if actor.is(d.view.Approver) {
			return viewOf(d), nil
		}
		return View{}, refuse(ReasonAlreadyApproved)
	}
	switch {
	case d.view.State != StateWaiting:
		return View{}, refuse(ReasonNotWaiting)
	case actor.ID == d.view.Requester:
		return View{}, refuse(ReasonSelfApproval)
	}
	p, err := r.validity(workspace, d.view.Delivery.Package)
	if err != nil {
		return View{}, err
	}
	switch {
	case !p.Live:
		return View{}, refuse(ReasonPackageNotLive)
	case p.Recipient != s.channels[d.view.Delivery.Channel].view.Channel.Recipient:
		return View{}, refuse(ReasonRecipientMismatch)
	}
	d.view.Approver = actor.ID
	return viewOf(d), nil
}

func (s *space) mark(d *delivery) {
	s.uncertain[d.key] = true
	d.view.PossibleDuplicate = true
	if current := s.byKey[d.key]; current != nil {
		current.view.PossibleDuplicate = true
	}
}

func (s *space) finish(d *delivery, state string) {
	if d.view.State == StateWaiting || d.view.State == StateInFlight {
		s.queued--
	}
	d.view.State = state
}

func (r *Registry) Cancel(workspace, id string, actor Actor) (View, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return View{}, ErrInvalid
	}
	if err := r.staff(actor); err != nil {
		return View{}, err
	}
	s, d, err := r.lookup(workspace, id)
	if err != nil {
		return View{}, err
	}
	switch d.view.State {
	case StateCancelled:
		if actor.is(d.view.Closer) {
			return viewOf(d), nil
		}
		return View{}, ErrConflict
	case StateAccepted, StateConfirmed:
		return View{}, refuse(ReasonFinal)
	}
	if d.running > 0 {
		s.mark(d)
	}
	s.finish(d, StateCancelled)
	d.view.Closer, d.running = actor.ID, 0
	if s.byKey[d.key] == d {
		delete(s.byKey, d.key)
	}
	return viewOf(d), nil
}

var resendable = map[string]bool{StateAccepted: true, StateConfirmed: true, StateCancelled: true, StateStopped: true, StateExhausted: true}

func (r *Registry) Resend(workspace, id string, next Delivery, reason string, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(next.ID) || !validID(reason) || !actor.valid() {
		return View{}, ErrInvalid
	}
	if err := r.staff(actor); err != nil {
		return View{}, err
	}
	s, d, err := r.lookup(workspace, id)
	if err != nil {
		return View{}, err
	}
	next.Package, next.Channel = d.view.Delivery.Package, d.view.Delivery.Channel
	if found := s.deliveries[next.ID]; found != nil {
		if found.view.Resends == id && found.view.ResendReason == reason && actor.is(found.view.Requester) {
			return viewOf(found), nil
		}
		return View{}, ErrConflict
	}
	if !resendable[d.view.State] || s.byKey[d.key] != nil && s.byKey[d.key] != d {
		return View{}, refuse(ReasonNotResendable)
	}
	p, err := r.validity(workspace, next.Package)
	if err != nil {
		return View{}, err
	}
	switch {
	case !p.Live:
		return View{}, refuse(ReasonPackageNotLive)
	case p.Recipient != s.channels[next.Channel].view.Channel.Recipient:
		return View{}, refuse(ReasonRecipientMismatch)
	}
	return r.create(s, next, d.key, p, actor, now, id, reason)
}

func (r *Registry) token() (string, error) {
	buffer := make([]byte, 16)
	if _, err := io.ReadFull(r.config.Random, buffer); err != nil {
		return "", refuse(ReasonRandomUnavailable)
	}
	return hex.EncodeToString(buffer), nil
}
