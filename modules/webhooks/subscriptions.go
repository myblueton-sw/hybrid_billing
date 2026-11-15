package webhooks

import (
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/events"
)

type subscription struct {
	view       SubscriptionView
	open       *delivery
	queue      []*delivery
	dead       []*delivery
	cancelled  []*delivery
	throttle   time.Time
	step       int
	redeliver  map[string]string
	dismiss    map[string]string
	generation int
	keys       map[string]*KeyView
	names      map[string]bool
}

type space struct {
	subscriptions map[string]*subscription
	order         []string
	next          int
	deliveries    map[string]*delivery
}

type Registry struct {
	mu         sync.Mutex
	config     Config
	limits     Limits
	clock      func() time.Time
	now        time.Time
	counter    uint64
	workspaces map[string]*space
}

func NewRegistry(config Config, limits Limits, clock func() time.Time) (*Registry, error) {
	if config.Outbox == nil || config.Destinations == nil || config.Permits == nil || config.Signer == nil || config.Redirects == nil || clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{config: config, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (l Limits) valid() bool {
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Subscriptions, maxSubscription}, {l.Types, maxTypes}, {l.Attempts, maxAttempts}, {l.Dead, maxDead},
		{l.Batch, maxBatch}, {l.Work, maxWork}, {l.Keys, maxKeys}}
	for _, p := range pairs {
		if p[0] < 1 || p[0] > p[1] {
			return false
		}
	}
	return l.Lease >= minLease && l.Lease <= maxLease && l.Backoff >= minBackoff && l.Backoff <= l.MaxBackoff && l.MaxBackoff <= maxMaxBackoff &&
		l.MaxAge >= l.Backoff && l.MaxAge <= maxMaxAge && l.KeyLife >= minKeyLife && l.KeyLife <= maxKeyLife
}

func (r *Registry) enter() time.Time {
	r.mu.Lock()
	now := r.clock().UTC().Round(0)
	if now.After(r.now) {
		r.now = now
	}
	return r.now
}

func (r *Registry) leave() { r.mu.Unlock() }

func refuse(reason string) error { return Refusal{Reason: reason} }

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) human() error {
	switch a.Kind {
	case KindAssistant:
		return refuse(ReasonAINotAllowed)
	case KindSystem:
		return refuse(ReasonNotHuman)
	}
	return nil
}

func (a Actor) system() error {
	if a.Kind != KindSystem {
		return refuse(ReasonNotSystem)
	}
	return nil
}

func (r *Registry) next(label string) string {
	r.counter++
	return label + "-" + strconv.FormatUint(r.counter, 10)
}

func copySubscription(s Subscription) Subscription {
	s.Types = append([]string(nil), s.Types...)
	sort.Strings(s.Types)
	return s
}

func sameSubscription(a, b Subscription) bool {
	a, b = copySubscription(a), copySubscription(b)
	if a.Subscriber != b.Subscriber || a.Destination != b.Destination || len(a.Types) != len(b.Types) {
		return false
	}
	for index := range a.Types {
		if a.Types[index] != b.Types[index] {
			return false
		}
	}
	return true
}

func viewSubscription(s *subscription) SubscriptionView {
	v := s.view
	v.Subscription = copySubscription(v.Subscription)
	v.Gaps = append([]Gap(nil), v.Gaps...)
	return v
}

func (r *Registry) subscriptionForm(s Subscription) bool {
	if !validID(s.ID) || !validID(s.Subscriber) || !validID(s.Destination) || len(s.Types) < 1 || len(s.Types) > r.limits.Types {
		return false
	}
	seen := map[string]bool{}
	for _, t := range s.Types {
		if !validID(t) || seen[t] {
			return false
		}
		seen[t] = true
	}
	return true
}

func (r *Registry) ProposeSubscription(workspace string, s Subscription, actor Actor) (SubscriptionView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !r.subscriptionForm(s) {
		return SubscriptionView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return SubscriptionView{}, err
	}
	sp := r.workspaces[workspace]
	if sp != nil && sp.subscriptions[s.ID] != nil {
		found := sp.subscriptions[s.ID]
		if found.view.State == SubscriptionProposed && found.view.Proposer == actor.ID && sameSubscription(found.view.Subscription, s) {
			return viewSubscription(found), nil
		}
		return SubscriptionView{}, ErrConflict
	}
	if sp == nil {
		if len(r.workspaces) >= r.limits.Workspaces {
			return SubscriptionView{}, ErrLimit
		}
		sp = &space{subscriptions: map[string]*subscription{}, deliveries: map[string]*delivery{}}
		r.workspaces[workspace] = sp
	}
	if len(sp.order) >= r.limits.Subscriptions {
		return SubscriptionView{}, ErrLimit
	}
	if len(sp.subscriptions) >= 2*r.limits.Subscriptions {
		r.purge(sp)
		if len(sp.subscriptions) >= 2*r.limits.Subscriptions {
			return SubscriptionView{}, ErrLimit
		}
	}
	made := &subscription{view: SubscriptionView{Subscription: copySubscription(s), State: SubscriptionProposed, Proposer: actor.ID}, redeliver: map[string]string{},
		dismiss: map[string]string{}, keys: map[string]*KeyView{}, names: map[string]bool{}}
	sp.subscriptions[s.ID] = made
	sp.order = append(sp.order, s.ID)
	return viewSubscription(made), nil
}

func (r *Registry) find(workspace, id string, actor Actor) (*space, *subscription, error) {
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return nil, nil, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return nil, nil, err
	}
	sp := r.workspaces[workspace]
	if sp == nil || sp.subscriptions[id] == nil {
		return nil, nil, ErrNotFound
	}
	return sp, sp.subscriptions[id], nil
}

func (r *Registry) admit(workspace string, s Subscription) error {
	state, err := r.config.Destinations.State(workspace, s.Destination)
	if err != nil {
		return refuse(ReasonDependency)
	}
	if state != DestinationActive {
		return refuse(ReasonDestinationInactive)
	}
	allowed, err := r.config.Permits.Subscribe(workspace, s.Subscriber, append([]string(nil), s.Types...))
	switch {
	case err != nil:
		return refuse(ReasonDependency)
	case !allowed:
		return refuse(ReasonSubscribeRefused)
	}
	return nil
}

func (r *Registry) ApproveSubscription(workspace, id string, actor Actor) (SubscriptionView, error) {
	r.enter()
	_, s, err := r.find(workspace, id, actor)
	if err != nil {
		r.leave()
		return SubscriptionView{}, err
	}
	switch {
	case s.view.State != SubscriptionProposed && s.view.Approver == actor.ID:
		defer r.leave()
		return viewSubscription(s), nil
	case s.view.State != SubscriptionProposed:
		r.leave()
		return SubscriptionView{}, refuse(ReasonNotPending)
	case s.view.Proposer == actor.ID:
		r.leave()
		return SubscriptionView{}, refuse(ReasonSelfApproval)
	}
	wanted, generation := copySubscription(s.view.Subscription), s.generation
	r.leave()
	if err := r.admit(workspace, wanted); err != nil {
		return SubscriptionView{}, err
	}
	_, head, err := r.config.Outbox.Bounds(workspace)
	if err != nil {
		return SubscriptionView{}, refuse(ReasonDependency)
	}
	r.enter()
	defer r.leave()
	if s.generation != generation || s.view.State != SubscriptionProposed {
		return SubscriptionView{}, ErrStale
	}
	s.generation++
	s.view.State, s.view.Approver, s.view.Cursor = SubscriptionActive, actor.ID, head
	return viewSubscription(s), nil
}

func (r *Registry) PauseSubscription(workspace, id string, actor Actor) (SubscriptionView, error) {
	r.enter()
	defer r.leave()
	_, s, err := r.find(workspace, id, actor)
	if err != nil {
		return SubscriptionView{}, err
	}
	if s.view.State != SubscriptionActive {
		return SubscriptionView{}, refuse(ReasonNotActive)
	}
	s.generation++
	s.view.State, s.view.Reason = SubscriptionPaused, ReasonPaused
	return viewSubscription(s), nil
}

func (r *Registry) RequestResume(workspace, id string, actor Actor) (SubscriptionView, error) {
	r.enter()
	defer r.leave()
	_, s, err := r.find(workspace, id, actor)
	if err != nil {
		return SubscriptionView{}, err
	}
	if s.view.State != SubscriptionPaused && s.view.State != SubscriptionSuspended {
		return SubscriptionView{}, refuse(ReasonNotPending)
	}
	s.view.Resumer = actor.ID
	return viewSubscription(s), nil
}

func (r *Registry) ApproveResume(workspace, id string, actor Actor) (SubscriptionView, error) {
	r.enter()
	_, s, err := r.find(workspace, id, actor)
	if err != nil {
		r.leave()
		return SubscriptionView{}, err
	}
	switch {
	case s.view.Resumer == "" || s.view.State != SubscriptionPaused && s.view.State != SubscriptionSuspended:
		r.leave()
		return SubscriptionView{}, refuse(ReasonNotPending)
	case s.view.Resumer == actor.ID:
		r.leave()
		return SubscriptionView{}, refuse(ReasonSelfApproval)
	}
	wanted, generation := copySubscription(s.view.Subscription), s.generation
	r.leave()
	if err := r.admit(workspace, wanted); err != nil {
		return SubscriptionView{}, err
	}
	floor, _, err := r.config.Outbox.Bounds(workspace)
	if err != nil {
		return SubscriptionView{}, refuse(ReasonDependency)
	}
	r.enter()
	defer r.leave()
	if s.generation != generation || s.view.Resumer == "" || s.view.State != SubscriptionPaused && s.view.State != SubscriptionSuspended {
		return SubscriptionView{}, ErrStale
	}
	if s.view.Reason == ReasonGap && floor > s.view.Cursor {
		s.view.Gaps = append(s.view.Gaps, Gap{After: s.view.Cursor, To: floor})
		if len(s.view.Gaps) > r.limits.Dead {
			s.view.Gaps = s.view.Gaps[1:]
		}
		s.view.Cursor = floor
	}
	s.generation++
	s.view.State, s.view.Reason, s.view.Resumer = SubscriptionActive, "", ""
	return viewSubscription(s), nil
}

func (r *Registry) RevokeSubscription(workspace, id string, actor Actor) (SubscriptionView, error) {
	r.enter()
	defer r.leave()
	sp, s, err := r.find(workspace, id, actor)
	if err != nil {
		return SubscriptionView{}, err
	}
	r.end(sp, s, ReasonSubscriptionRevoked)
	return viewSubscription(s), nil
}

func (r *Registry) end(sp *space, s *subscription, reason string) {
	if s.view.State == SubscriptionRevoked {
		return
	}
	s.generation++
	s.view.State, s.view.Reason, s.view.Resumer = SubscriptionRevoked, reason, ""
	for index, id := range sp.order {
		if id == s.view.Subscription.ID {
			sp.order = append(sp.order[:index:index], sp.order[index+1:]...)
			break
		}
	}
	if s.open != nil {
		r.cancel(sp, s, s.open, reason)
	}
}

func (r *Registry) SubscriptionOf(workspace, id string) (SubscriptionView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return SubscriptionView{}, ErrInvalid
	}
	sp := r.workspaces[workspace]
	if sp == nil || sp.subscriptions[id] == nil {
		return SubscriptionView{}, ErrNotFound
	}
	return viewSubscription(sp.subscriptions[id]), nil
}

func matches(s *subscription, e events.Stored) bool {
	for _, t := range s.view.Subscription.Types {
		if t == e.Event.Type {
			return true
		}
	}
	return false
}

func (r *Registry) purge(sp *space) {
	for id, s := range sp.subscriptions {
		if s.view.State == SubscriptionRevoked && len(s.dead) == 0 && len(s.queue) == 0 {
			delete(sp.subscriptions, id)
		}
	}
}
