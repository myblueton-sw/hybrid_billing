package webhooks

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/events"
)

const (
	phaseDue        = "due"
	phaseLeased     = "leased"
	phaseConfirming = "confirming"
	phaseConfirmed  = "confirmed"
)

type delivery struct {
	id        string
	sub       *subscription
	event     events.Stored
	kind      string
	phase     string
	attempts  int
	step      int
	opened    time.Time
	due       time.Time
	token     string
	until     time.Time
	confirmed bool
	reason    string
	version   string
	held      bool
}

func (r *Registry) backoff(d *delivery) time.Duration {
	wait := r.limits.Backoff
	for index := 0; index < d.step && wait < r.limits.MaxBackoff; index++ {
		wait *= 2
	}
	if wait > r.limits.MaxBackoff {
		wait = r.limits.MaxBackoff
	}
	sum := sha256.Sum256([]byte(d.id + "/" + strconv.Itoa(d.step)))
	return wait - wait*time.Duration(binary.BigEndian.Uint16(sum[:2])%(jitterPercent+1))/100
}

func (r *Registry) retry(d *delivery, now time.Time) {
	d.due = now.Add(r.backoff(d))
	d.step++
	d.phase, d.token = phaseDue, ""
}

func (r *Registry) room(s *subscription) bool {
	return len(s.queue)+len(s.dead) < r.limits.Dead
}

func (r *Registry) bury(sp *space, s *subscription, d *delivery, reason string) {
	if !r.room(s) {
		d.phase, d.token, d.reason, d.held = phaseDue, "", reason, true
		if s.view.State == SubscriptionActive {
			s.generation++
			s.view.State, s.view.Reason = SubscriptionSuspended, ReasonBacklog
		}
		return
	}
	d.kind, d.phase, d.token, d.reason, d.held = KindDead, phaseDue, "", reason, false
	s.open = nil
	s.dead = append(s.dead, d)
}

func (r *Registry) cancel(sp *space, s *subscription, d *delivery, reason string) {
	d.kind, d.token, d.reason = KindCancelled, "", reason
	s.open = nil
	delete(sp.deliveries, d.id)
	s.cancelled = append(s.cancelled, d)
	if len(s.cancelled) > r.limits.Dead {
		s.cancelled = s.cancelled[1:]
	}
}

func (r *Registry) close(sp *space, s *subscription, d *delivery) {
	s.open = nil
	delete(sp.deliveries, d.id)
}

func (r *Registry) expire(sp *space, s *subscription, now time.Time) {
	d := s.open
	if d == nil || d.held {
		return
	}
	switch {
	case (d.phase == phaseLeased || d.phase == phaseConfirming) && !now.Before(d.until):
		r.retry(d, now)
	case d.phase == phaseConfirmed && !now.Before(d.until):
		d.attempts++
		r.retry(d, now)
		if d.attempts >= r.limits.Attempts {
			r.bury(sp, s, d, ReasonAttempts)
			return
		}
	}
	if s.open == d && !now.Before(d.opened.Add(r.limits.MaxAge)) {
		r.bury(sp, s, d, ReasonMaxAge)
	}
}

type read struct {
	s          *subscription
	generation int
	cursor     uint64
	page       events.Page
	err        error
}

func (r *Registry) lease(sp *space, d *delivery, now time.Time) Lease {
	d.phase, d.token, d.until = phaseLeased, r.next("lease"), now.Add(r.limits.Lease)
	return Lease{Subscription: d.sub.view.Subscription.ID, Delivery: d.id, Token: d.token, Sequence: d.event.Sequence, Source: d.event.Event.Source,
		EventID: d.event.Event.ID, Destination: d.sub.view.Subscription.Destination}
}

func (r *Registry) Dispatch(workspace string, limit int, actor Actor) ([]Lease, error) {
	if !validID(workspace) || limit < 1 || limit > r.limits.Batch || !actor.valid() {
		return nil, ErrInvalid
	}
	if err := actor.system(); err != nil {
		return nil, err
	}
	_, head, err := r.config.Outbox.Bounds(workspace)
	if err != nil {
		return nil, refuse(ReasonDependency)
	}
	now := r.enter()
	sp := r.workspaces[workspace]
	if sp == nil {
		r.leave()
		return []Lease{}, nil
	}
	var leases []Lease
	var reads []*read
	visited := sp.next
	for offset := 0; offset < len(sp.order) && len(leases)+len(reads) < limit; offset++ {
		index := (sp.next + offset) % len(sp.order)
		s := sp.subscriptions[sp.order[index]]
		r.expire(sp, s, now)
		if s.view.State != SubscriptionActive || now.Before(s.throttle) {
			continue
		}
		if s.open != nil && s.open.held {
			r.bury(sp, s, s.open, s.open.reason)
			visited = index + 1
			continue
		}
		if s.open == nil && len(s.queue) > 0 {
			s.open, s.queue = s.queue[0], s.queue[1:]
			sp.deliveries[s.open.id] = s.open
		}
		if s.open == nil && s.view.Cursor < head {
			if len(reads) >= r.limits.Work {
				break
			}
			reads = append(reads, &read{s: s, generation: s.generation, cursor: s.view.Cursor})
		} else if s.open != nil && s.open.phase == phaseDue && !now.Before(s.open.due) {
			leases = append(leases, r.lease(sp, s.open, now))
		}
		visited = index + 1
	}
	sp.next = visited % max(len(sp.order), 1)
	r.leave()
	for _, pending := range reads {
		pending.page, pending.err = r.config.Outbox.Read(workspace, pending.cursor, r.limits.Batch)
	}
	now = r.enter()
	defer r.leave()
	for _, pending := range reads {
		s := pending.s
		if s.generation != pending.generation || s.view.Cursor != pending.cursor || s.open != nil || s.view.State != SubscriptionActive {
			continue
		}
		var expired events.Expired
		if errors.As(pending.err, &expired) {
			s.generation++
			s.view.State, s.view.Reason = SubscriptionSuspended, ReasonGap
			continue
		}
		if pending.err != nil {
			continue
		}
		for _, e := range pending.page.Events {
			s.view.Cursor = e.Sequence
			if matches(s, e) {
				d := &delivery{id: r.next("delivery"), sub: s, event: e, kind: KindOpen, phase: phaseDue, opened: now, due: now}
				s.open = d
				sp.deliveries[d.id] = d
				leases = append(leases, r.lease(sp, d, now))
				break
			}
		}
	}
	if leases == nil {
		leases = []Lease{}
	}
	return leases, nil
}

func message(workspace string, d *delivery, attempt int, version string, at time.Time, body []byte) []byte {
	var out bytes.Buffer
	part := func(value string) {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		out.Write(size[:])
		out.WriteString(value)
	}
	digest := sha256.Sum256(body)
	for _, value := range []string{signatureTag, workspace, d.sub.view.Subscription.ID, d.id, d.event.Event.Source, d.event.Event.ID, strconv.Itoa(attempt), version,
		strconv.FormatInt(at.UnixNano(), 10), string(digest[:])} {
		part(value)
	}
	return out.Bytes()
}

func (r *Registry) held(workspace, id, token string, actor Actor) (*space, *delivery, error) {
	if !validID(workspace) || !validID(id) || token == "" || !actor.valid() {
		return nil, nil, ErrInvalid
	}
	if err := actor.system(); err != nil {
		return nil, nil, err
	}
	sp := r.workspaces[workspace]
	if sp == nil || sp.deliveries[id] == nil || sp.deliveries[id].token != token {
		return nil, nil, ErrStale
	}
	return sp, sp.deliveries[id], nil
}

func (r *Registry) Confirm(workspace, id, token string, actor Actor) (Signed, error) {
	now := r.enter()
	sp, d, err := r.held(workspace, id, token, actor)
	if err == nil && (d.phase != phaseLeased || !now.Before(d.until)) {
		err = ErrStale
	}
	if err != nil {
		r.leave()
		return Signed{}, err
	}
	s := d.sub
	d.phase, d.until = phaseConfirming, now.Add(r.limits.Lease)
	subscriber, destination, subscriptionID, event, attempt := s.view.Subscription.Subscriber, s.view.Subscription.Destination, s.view.Subscription.ID, d.event, d.attempts+1
	r.leave()
	body, at := events.Body(event), now
	verdict, version, signature, signedMessage := r.check(workspace, destination, subscriber, subscriptionID, event, d, attempt, at, &at, body)
	now = r.enter()
	defer r.leave()
	if d.token != token || d.phase != phaseConfirming || !now.Before(d.until) || s.view.State != SubscriptionActive {
		if d.token == token && d.phase == phaseConfirming {
			r.retry(d, now)
		}
		return Signed{}, ErrStale
	}
	switch verdict {
	case ReasonDestinationRevoked:
		r.end(sp, s, verdict)
		return Signed{}, refuse(verdict)
	case ReasonDestinationInactive:
		d.phase, d.token = phaseDue, ""
		s.generation++
		s.view.State, s.view.Reason = SubscriptionSuspended, verdict
		return Signed{}, refuse(verdict)
	case ReasonPermissionRevoked:
		r.cancel(sp, s, d, verdict)
		return Signed{}, refuse(verdict)
	case "":
	default:
		r.retry(d, now)
		return Signed{}, refuse(verdict)
	}
	public := published(s, version, at)
	refused := ""
	switch {
	case public == nil:
		refused = ReasonKeyUnpublished
	case !ed25519.Verify(public, signedMessage, signature):
		refused = ReasonKeyMismatch
	}
	if refused != "" {
		d.phase, d.token = phaseDue, ""
		s.generation++
		s.view.State, s.view.Reason = SubscriptionSuspended, refused
		return Signed{}, refuse(refused)
	}
	d.version = version
	d.phase, d.until, d.confirmed = phaseConfirmed, now.Add(r.limits.Lease), true
	headers := map[string]string{"webhook-delivery": d.id, "webhook-event-id": event.Event.ID, "webhook-event-source": event.Event.Source,
		"webhook-attempt": strconv.Itoa(attempt), "webhook-key-version": version, "webhook-time": strconv.FormatInt(at.UnixNano(), 10),
		"webhook-subscription": subscriptionID}
	return Signed{Delivery: d.id, Attempt: attempt, Version: version, At: at, Body: body, Signature: append([]byte(nil), signature...), Headers: headers}, nil
}

func (r *Registry) check(workspace, destination, subscriber, subscriptionID string, event events.Stored, d *delivery, attempt int, at time.Time, when *time.Time, body []byte) (string, string, []byte, []byte) {
	state, err := r.config.Destinations.State(workspace, destination)
	switch {
	case err != nil:
		return ReasonDependency, "", nil, nil
	case state == DestinationRevoked:
		return ReasonDestinationRevoked, "", nil, nil
	case state != DestinationActive:
		return ReasonDestinationInactive, "", nil, nil
	}
	allowed, err := r.config.Permits.Allowed(workspace, subscriber, event)
	switch {
	case err != nil:
		return ReasonDependency, "", nil, nil
	case !allowed:
		return ReasonPermissionRevoked, "", nil, nil
	}
	version, err := r.config.Signer.Current(workspace, subscriptionID)
	if err != nil {
		return ReasonDependency, "", nil, nil
	}
	if fresh := r.clock().UTC().Round(0); fresh.After(at) {
		*when = fresh
		at = fresh
	}
	signed := message(workspace, d, attempt, version, at, body)
	signature, err := r.config.Signer.Sign(workspace, subscriptionID, version, signed)
	if err != nil {
		return ReasonDependency, "", nil, nil
	}
	return "", version, signature, signed
}

func (r *Registry) Redirect(workspace, id, token, current string, hop int, location string, actor Actor) (string, int, error) {
	r.enter()
	_, d, err := r.held(workspace, id, token, actor)
	if err == nil && d.phase != phaseConfirmed {
		err = ErrStale
	}
	if err != nil {
		r.leave()
		return "", 0, err
	}
	destination := d.sub.view.Subscription.Destination
	r.leave()
	target, url, next, err := r.config.Redirects.Check(workspace, destination, current, hop, location)
	if err != nil || target != destination {
		return "", 0, refuse(ReasonUnapprovedRedirect)
	}
	r.enter()
	defer r.leave()
	if d.token != token || d.phase != phaseConfirmed {
		return "", 0, ErrStale
	}
	if k := d.sub.keys[d.version]; k == nil || k.State != KeyActive {
		d.phase, d.token = phaseDue, ""
		return "", 0, refuse(ReasonKeyUnpublished)
	}
	return url, next, nil
}

func (r *Registry) Report(workspace, id, token string, outcome Outcome, actor Actor) (DeliveryView, error) {
	now := r.enter()
	defer r.leave()
	sp, d, err := r.held(workspace, id, token, actor)
	if err != nil {
		return DeliveryView{}, err
	}
	if d.phase != phaseConfirmed {
		return DeliveryView{}, ErrStale
	}
	s := d.sub
	switch outcome.Status {
	case StatusDelivered:
		r.close(sp, s, d)
		d.kind, d.phase, d.token = KindOpen, phaseConfirmed, ""
		d.reason = StatusDelivered
	case StatusRejected:
		r.bury(sp, s, d, ReasonRejected)
	case StatusThrottled:
		wait := outcome.RetryAfter
		if wait <= 0 {
			wait = r.backoff(d)
		}
		wait = min(max(wait, r.limits.Backoff), r.limits.MaxBackoff)
		s.throttle = now.Add(wait)
		d.step++
		d.phase, d.token, d.due = phaseDue, "", s.throttle
	case StatusFailed, StatusUnknown:
		d.attempts++
		r.retry(d, now)
		if d.attempts >= r.limits.Attempts {
			r.bury(sp, s, d, ReasonAttempts)
		}
	default:
		return DeliveryView{}, ErrInvalid
	}
	return view(d), nil
}

func state(d *delivery) string {
	switch {
	case d.kind == KindCancelled:
		return JobCancelled
	case d.kind == KindDead:
		return JobFailed
	case d.reason == StatusDelivered:
		return JobSucceeded
	case d.phase != phaseDue:
		return JobRunning
	case d.kind == KindRedelivery && d.sub.open != d:
		return JobQueued
	case d.sub.view.State != SubscriptionActive || d.step > 0:
		return JobWaiting
	}
	return JobQueued
}

func view(d *delivery) DeliveryView {
	return DeliveryView{ID: d.id, Subscription: d.sub.view.Subscription.ID, Sequence: d.event.Sequence, Source: d.event.Event.Source, EventID: d.event.Event.ID,
		Kind: d.kind, State: state(d), Reason: d.reason, Attempts: d.attempts, OutcomeUnknown: d.kind == KindCancelled && d.confirmed}
}

func (r *Registry) listed(workspace, id string, actor Actor) (*space, *subscription, error) {
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return nil, nil, ErrInvalid
	}
	if actor.Kind == KindAssistant {
		return nil, nil, refuse(ReasonAINotAllowed)
	}
	sp := r.workspaces[workspace]
	if sp == nil || sp.subscriptions[id] == nil {
		return nil, nil, ErrNotFound
	}
	return sp, sp.subscriptions[id], nil
}

func (r *Registry) Deliveries(workspace, id string, after, limit int, actor Actor) ([]DeliveryView, error) {
	r.enter()
	defer r.leave()
	if after < 0 || limit < 1 {
		return nil, ErrInvalid
	}
	_, s, err := r.listed(workspace, id, actor)
	if err != nil {
		return nil, err
	}
	var all []*delivery
	if s.open != nil {
		all = append(all, s.open)
	}
	all = append(append(append(all, s.queue...), s.dead...), s.cancelled...)
	out := []DeliveryView{}
	for index := after; index < len(all) && len(out) < limit; index++ {
		out = append(out, view(all[index]))
	}
	return out, nil
}

func (r *Registry) letter(workspace, id, letterID string, actor Actor) (*subscription, int, error) {
	_, s, err := r.find(workspace, id, actor)
	if err != nil {
		return nil, 0, err
	}
	if !validID(letterID) {
		return nil, 0, ErrInvalid
	}
	for index, d := range s.dead {
		if d.id == letterID {
			return s, index, nil
		}
	}
	return nil, 0, ErrNotFound
}

func (r *Registry) RequestRedeliver(workspace, id, letterID string, actor Actor) error {
	r.enter()
	defer r.leave()
	s, _, err := r.letter(workspace, id, letterID, actor)
	if err != nil {
		return err
	}
	s.redeliver[letterID] = actor.ID
	return nil
}

func (r *Registry) ApproveRedeliver(workspace, id, letterID string, actor Actor) (DeliveryView, error) {
	r.enter()
	defer r.leave()
	s, index, err := r.letter(workspace, id, letterID, actor)
	if err != nil {
		return DeliveryView{}, err
	}
	switch requester := s.redeliver[letterID]; {
	case requester == "":
		return DeliveryView{}, refuse(ReasonNotPending)
	case requester == actor.ID:
		return DeliveryView{}, refuse(ReasonSelfApproval)
	}
	if s.view.State == SubscriptionRevoked {
		return DeliveryView{}, refuse(ReasonRevoked)
	}
	old := s.dead[index]
	delete(r.workspaces[workspace].deliveries, old.id)
	s.dead = append(s.dead[:index:index], s.dead[index+1:]...)
	delete(s.redeliver, letterID)
	delete(s.dismiss, letterID)
	fresh := &delivery{id: r.next("delivery"), sub: s, event: old.event, kind: KindRedelivery, phase: phaseDue, opened: r.now, due: r.now}
	s.queue = append(s.queue, fresh)
	return view(fresh), nil
}

func (r *Registry) RequestDismiss(workspace, id, letterID string, actor Actor) error {
	r.enter()
	defer r.leave()
	s, _, err := r.letter(workspace, id, letterID, actor)
	if err != nil {
		return err
	}
	s.dismiss[letterID] = actor.ID
	return nil
}

func (r *Registry) ApproveDismiss(workspace, id, letterID string, actor Actor) error {
	r.enter()
	defer r.leave()
	s, index, err := r.letter(workspace, id, letterID, actor)
	if err != nil {
		return err
	}
	switch requester := s.dismiss[letterID]; {
	case requester == "":
		return refuse(ReasonNotPending)
	case requester == actor.ID:
		return refuse(ReasonSelfApproval)
	}
	delete(r.workspaces[workspace].deliveries, s.dead[index].id)
	s.dead = append(s.dead[:index:index], s.dead[index+1:]...)
	delete(s.dismiss, letterID)
	delete(s.redeliver, letterID)
	if d := s.open; d != nil && d.held {
		r.bury(r.workspaces[workspace], s, d, d.reason)
	}
	return nil
}
