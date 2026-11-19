package webhooks

func firstFree(queue []*delivery) int {
	for index, d := range queue {
		if d.hold == "" {
			return index
		}
	}
	return -1
}

func ended(state string) bool { return state == SubscriptionRevoked }

func (r *Registry) subscription(workspace, id string) (*space, *subscription, error) {
	if !validID(workspace) || !validID(id) {
		return nil, nil, ErrInvalid
	}
	sp := r.workspaces[workspace]
	if sp == nil || sp.subscriptions[id] == nil {
		return nil, nil, ErrNotFound
	}
	return sp, sp.subscriptions[id], nil
}

func (r *Registry) WithdrawSubscription(workspace, id string, actor Actor) (SubscriptionView, error) {
	r.enter()
	defer r.leave()
	sp, s, err := r.find(workspace, id, actor)
	switch {
	case err != nil:
		return SubscriptionView{}, err
	case s.view.Proposer != actor.ID:
		return SubscriptionView{}, refuse(ReasonNotRequester)
	case s.view.State != SubscriptionProposed:
		return SubscriptionView{}, refuse(ReasonNotPending)
	}
	r.end(sp, s, ReasonWithdrawn)
	return viewSubscription(s), nil
}

func (r *Registry) CloseSubscription(workspace, id, reason string) (SubscriptionView, error) {
	r.enter()
	defer r.leave()
	sp, s, err := r.subscription(workspace, id)
	switch {
	case err != nil:
		return SubscriptionView{}, err
	case reason != ReasonRequesterLostRight:
		return SubscriptionView{}, ErrInvalid
	case s.view.State != SubscriptionProposed:
		return SubscriptionView{}, refuse(ReasonNotPending)
	}
	r.end(sp, s, reason)
	return viewSubscription(s), nil
}

func resuming(s *subscription) bool {
	return s.view.Resumer != "" && (s.view.State == SubscriptionPaused || s.view.State == SubscriptionSuspended)
}

func (r *Registry) WithdrawResume(workspace, id string, actor Actor) (SubscriptionView, error) {
	r.enter()
	defer r.leave()
	_, s, err := r.find(workspace, id, actor)
	switch {
	case err != nil:
		return SubscriptionView{}, err
	case !resuming(s):
		return SubscriptionView{}, refuse(ReasonNotPending)
	case s.view.Resumer != actor.ID:
		return SubscriptionView{}, refuse(ReasonNotRequester)
	}
	s.view.Resumer = ""
	return viewSubscription(s), nil
}

func (r *Registry) CloseResume(workspace, id, reason string) (SubscriptionView, error) {
	r.enter()
	defer r.leave()
	_, s, err := r.subscription(workspace, id)
	switch {
	case err != nil:
		return SubscriptionView{}, err
	case reason != ReasonRequesterLostRight:
		return SubscriptionView{}, ErrInvalid
	case !resuming(s):
		return SubscriptionView{}, refuse(ReasonNotPending)
	}
	s.view.Resumer = ""
	return viewSubscription(s), nil
}

func (r *Registry) request(workspace, id, letterID string, dismiss bool) (map[string]string, error) {
	_, s, err := r.subscription(workspace, id)
	if err != nil {
		return nil, err
	}
	if !validID(letterID) {
		return nil, ErrInvalid
	}
	dead := false
	for _, d := range s.dead {
		dead = dead || d.id == letterID
	}
	if !dead {
		return nil, ErrNotFound
	}
	requests := s.redeliver
	if dismiss {
		requests = s.dismiss
	}
	if requests[letterID] == "" {
		return nil, refuse(ReasonNotPending)
	}
	return requests, nil
}

func (r *Registry) WithdrawRequest(workspace, id, letterID string, dismiss bool, actor Actor) error {
	r.enter()
	defer r.leave()
	if !actor.valid() {
		return ErrInvalid
	}
	if err := actor.human(); err != nil {
		return err
	}
	requests, err := r.request(workspace, id, letterID, dismiss)
	if err != nil {
		return err
	}
	if requests[letterID] != actor.ID {
		return refuse(ReasonNotRequester)
	}
	delete(requests, letterID)
	return nil
}

func (r *Registry) CloseRequest(workspace, id, letterID string, dismiss bool, reason string) error {
	r.enter()
	defer r.leave()
	if reason != ReasonRequesterLostRight {
		return ErrInvalid
	}
	requests, err := r.request(workspace, id, letterID, dismiss)
	if err != nil {
		return err
	}
	delete(requests, letterID)
	return nil
}

func (r *Registry) redelivery(workspace, id, deliveryID string) (*space, *subscription, *delivery, error) {
	sp, s, err := r.subscription(workspace, id)
	if err != nil {
		return nil, nil, nil, err
	}
	if !validID(deliveryID) {
		return nil, nil, nil, ErrInvalid
	}
	if d := s.open; d != nil && d.id == deliveryID && d.kind == KindRedelivery && d.phase == phaseDue && d.step == 0 && d.attempts == 0 {
		return sp, s, d, nil
	}
	for _, d := range s.queue {
		if d.id == deliveryID && d.kind == KindRedelivery {
			return sp, s, d, nil
		}
	}
	return nil, nil, nil, ErrNotFound
}

func (r *Registry) unqueue(sp *space, s *subscription, d *delivery) {
	if s.open == d {
		s.open = nil
		delete(sp.deliveries, d.id)
		return
	}
	for index, queued := range s.queue {
		if queued == d {
			s.queue = append(s.queue[:index:index], s.queue[index+1:]...)
			return
		}
	}
}

func (r *Registry) HoldRedelivery(workspace, id, deliveryID, reason string) (DeliveryView, error) {
	r.enter()
	defer r.leave()
	if reason != ReasonApproverLostRight {
		return DeliveryView{}, ErrInvalid
	}
	sp, s, d, err := r.redelivery(workspace, id, deliveryID)
	if err != nil {
		return DeliveryView{}, err
	}
	if s.open == d {
		r.unqueue(sp, s, d)
		s.queue = append([]*delivery{d}, s.queue...)
	}
	d.hold = reason
	return view(d), nil
}

func (r *Registry) CloseRedelivery(workspace, id, deliveryID, reason string) (DeliveryView, error) {
	r.enter()
	defer r.leave()
	if reason != ReasonRequesterLostRight {
		return DeliveryView{}, ErrInvalid
	}
	sp, s, d, err := r.redelivery(workspace, id, deliveryID)
	if err != nil {
		return DeliveryView{}, err
	}
	r.unqueue(sp, s, d)
	d.hold, d.kind, d.token, d.reason = "", KindCancelled, "", reason
	s.cancelled = append(s.cancelled, d)
	if len(s.cancelled) > r.limits.Dead {
		s.cancelled = s.cancelled[1:]
	}
	return view(d), nil
}

func (r *Registry) ReapproveRedelivery(workspace, id, deliveryID string, actor Actor) (DeliveryView, error) {
	r.enter()
	defer r.leave()
	if !actor.valid() {
		return DeliveryView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return DeliveryView{}, err
	}
	_, s, d, err := r.redelivery(workspace, id, deliveryID)
	switch {
	case err != nil:
		return DeliveryView{}, err
	case d.hold == "":
		return DeliveryView{}, refuse(ReasonNotHeld)
	case d.requester == actor.ID || d.approver == actor.ID:
		return DeliveryView{}, refuse(ReasonSelfApproval)
	case ended(s.view.State):
		return DeliveryView{}, refuse(ReasonRevoked)
	}
	d.hold, d.approver = "", actor.ID
	return view(d), nil
}
