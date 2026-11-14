package protected

import (
	"sync"
	"time"
)

type change struct {
	view     View
	approved map[string]bool
}

type key struct {
	subject string
	field   string
}

type space struct {
	revision int
	active   *RulesView
	pending  *RulesView
	dropped  *RulesView
	history  []*RulesView
	channels map[string]*ChannelView
	holders  map[string][]*ChannelView
	changes  map[string]*change
	byField  map[key][]*change
}

type Registry struct {
	mu         sync.Mutex
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
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

func validDigest(text string) bool {
	if len(text) != 64 {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (ref Ref) valid() bool { return validID(ref.ID) && validDigest(ref.Fingerprint) }

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) is(id string) bool { return a.Kind == KindHuman && a.ID == id }

func (a Actor) refusal() error {
	switch a.Kind {
	case KindAssistant:
		return refuse(ReasonAINotAllowed)
	case KindSystem:
		return refuse(ReasonNotHuman)
	}
	return nil
}

func (r *Registry) Propose(workspace string, rules Rules, actor Actor) (RulesView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !rules.valid() || !actor.valid() {
		return RulesView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s != nil && s.pending != nil && s.pending.Rules == rules && actor.is(s.pending.Proposer) {
		return *s.pending, nil
	}
	if err := actor.refusal(); err != nil {
		return RulesView{}, err
	}
	if s != nil && s.active != nil && s.active.Rules == rules {
		if actor.is(s.active.Proposer) {
			return *s.active, nil
		}
		return RulesView{}, refuse(ReasonNoChange)
	}
	if s != nil && s.pending != nil && s.pending.Proposer != actor.ID {
		return RulesView{}, refuse(ReasonProposalPending)
	}
	if s != nil && len(s.history) >= maxProposals {
		return RulesView{}, ErrLimit
	}
	if s == nil {
		if len(r.workspaces) >= r.limits.Workspaces {
			return RulesView{}, ErrLimit
		}
		s = &space{channels: map[string]*ChannelView{}, holders: map[string][]*ChannelView{}, changes: map[string]*change{}, byField: map[key][]*change{}}
		r.workspaces[workspace] = s
	}
	s.revision++
	s.pending = &RulesView{Rules: rules, Revision: s.revision, Proposer: actor.ID, Proposed: now}
	s.history = append(s.history, s.pending)
	return *s.pending, nil
}

func (r *Registry) Confirm(workspace string, revision int, actor Actor) (RulesView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || revision < 1 || !actor.valid() {
		return RulesView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil {
		return RulesView{}, ErrNotFound
	}
	if s.active != nil && s.active.Revision == revision && actor.is(s.active.Confirmer) {
		return *s.active, nil
	}
	if err := actor.refusal(); err != nil {
		return RulesView{}, err
	}
	if s.pending == nil || s.pending.Revision != revision {
		return RulesView{}, refuse(ReasonStaleProposal)
	}
	if s.pending.Proposer == actor.ID {
		return RulesView{}, refuse(ReasonSelfApproval)
	}
	s.pending.Confirmer, s.pending.Confirmed = actor.ID, now
	s.active, s.pending = s.pending, nil
	return *s.active, nil
}

func (r *Registry) Discard(workspace string, revision int, actor Actor) (RulesView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || revision < 1 || !actor.valid() {
		return RulesView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil {
		return RulesView{}, ErrNotFound
	}
	if s.dropped != nil && s.dropped.Revision == revision && actor.is(s.dropped.Discarder) {
		return *s.dropped, nil
	}
	if err := actor.refusal(); err != nil {
		return RulesView{}, err
	}
	if s.pending == nil || s.pending.Revision != revision {
		return RulesView{}, refuse(ReasonStaleProposal)
	}
	s.pending.Discarder, s.pending.Discarded = actor.ID, now
	s.dropped, s.pending = s.pending, nil
	return *s.dropped, nil
}

func (r *Registry) RulesOf(workspace string) (RulesState, error) {
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return RulesState{}, ErrNotFound
	}
	var out RulesState
	if s.active != nil {
		view := *s.active
		out.Active = &view
	}
	if s.pending != nil {
		view := *s.pending
		out.Pending = &view
	}
	for _, view := range s.history {
		out.History = append(out.History, *view)
	}
	return out, nil
}

func (c Channel) valid() bool {
	return validID(c.ID) && validID(c.Subject) && channelKinds[c.Kind] && c.Ref.valid() && validID(c.Evidence)
}

func (r *Registry) Register(workspace string, channel Channel, actor Actor) (ChannelView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !channel.valid() || !actor.valid() {
		return ChannelView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil {
		return ChannelView{}, ErrNotFound
	}
	if seen := s.channels[channel.ID]; seen != nil {
		if seen.Channel == channel && actor.is(seen.Registrar) {
			return *seen, nil
		}
		return ChannelView{}, ErrConflict
	}
	if err := actor.refusal(); err != nil {
		return ChannelView{}, err
	}
	if len(s.channels) >= r.limits.Channels {
		return ChannelView{}, ErrLimit
	}
	view := &ChannelView{Channel: channel, Registrar: actor.ID, Registered: now}
	s.channels[channel.ID] = view
	s.holders[channel.Subject] = append(s.holders[channel.Subject], view)
	return *view, nil
}

func (r *Registry) Admit(workspace, channel, evidence string, actor Actor) (ChannelView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(channel) || !validID(evidence) || !actor.valid() {
		return ChannelView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil || s.channels[channel] == nil {
		return ChannelView{}, ErrNotFound
	}
	view := s.channels[channel]
	if view.Admitter != "" {
		if actor.is(view.Admitter) && view.Admission == evidence {
			return *view, nil
		}
		return ChannelView{}, ErrConflict
	}
	if err := actor.refusal(); err != nil {
		return ChannelView{}, err
	}
	if view.RetiredBy != "" {
		return ChannelView{}, refuse(ReasonChannelRetired)
	}
	if view.Registrar == actor.ID {
		return ChannelView{}, refuse(ReasonSelfApproval)
	}
	view.Admitter, view.Admitted, view.Admission = actor.ID, now, evidence
	return *view, nil
}

func (view *ChannelView) custodian(id string) bool {
	return view.Registrar == id || view.Admitter == id
}

func (r *Registry) Retire(workspace, channel string, actor Actor) (ChannelView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(channel) || !actor.valid() {
		return ChannelView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil || s.channels[channel] == nil {
		return ChannelView{}, ErrNotFound
	}
	view := s.channels[channel]
	if view.RetiredBy != "" {
		if actor.is(view.RetiredBy) {
			return *view, nil
		}
		return ChannelView{}, ErrConflict
	}
	if err := actor.refusal(); err != nil {
		return ChannelView{}, err
	}
	view.RetiredBy, view.Retired = actor.ID, now
	return *view, nil
}

func (r *Registry) ChannelOf(workspace, channel string) (ChannelView, error) {
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.channels[channel] == nil {
		return ChannelView{}, ErrNotFound
	}
	return *s.channels[channel], nil
}
