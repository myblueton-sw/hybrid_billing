package events

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"strconv"
	"time"
)

type subjectKey struct {
	source  string
	subject string
}

type revision struct {
	value  uint64
	id     string
	digest [32]byte
}

type seenEvent struct {
	digest [32]byte
	at     time.Time
}

type reservation struct {
	token   string
	key     eventKey
	subject subjectKey
	scoped  bool
	value   uint64
	ranked  bool
	fresh   bool
	gone    bool
	event   Event
	digest  [32]byte
	expires time.Time
}

func Digest(e Event) [32]byte {
	var out bytes.Buffer
	part := func(value string) {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		out.Write(size[:])
		out.WriteString(value)
	}
	for _, value := range []string{e.ID, e.Source, e.Type, e.SchemaVersion, e.Subject, e.ObjectRevision} {
		part(value)
	}
	names := make([]string, 0, len(e.Data))
	for name := range e.Data {
		names = append(names, name)
	}
	sort.Strings(names)
	part(strconv.Itoa(len(names)))
	for _, name := range names {
		part(name)
		part(e.Data[name])
	}
	return sha256.Sum256(out.Bytes())
}

func (r *Registry) BindSource(workspace, source, connection string, actor Actor) (Binding, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validSource(source) || !validID(connection) || !actor.valid() {
		return Binding{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return Binding{}, err
	}
	if s := r.workspaces[workspace]; s != nil && s.bindings[source] != nil && s.bindings[source].State != StateRevoked {
		found := s.bindings[source]
		if found.State == StateProposed && found.Proposer == actor.ID && found.Connection == connection {
			return *found, nil
		}
		return Binding{}, ErrConflict
	}
	s, err := r.open(workspace)
	if err != nil {
		return Binding{}, err
	}
	if s.bindings[source] == nil && len(s.bindings) >= r.limits.Sources {
		return Binding{}, ErrLimit
	}
	made := &Binding{Source: source, Connection: connection, State: StateProposed, Proposer: actor.ID}
	s.bindings[source] = made
	return *made, nil
}

func (r *Registry) binding(workspace, source string, actor Actor) (*Binding, error) {
	if !validID(workspace) || !validSource(source) || !actor.valid() {
		return nil, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return nil, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.bindings[source] == nil {
		return nil, ErrNotFound
	}
	return s.bindings[source], nil
}

func (r *Registry) ApproveSource(workspace, source string, actor Actor) (Binding, error) {
	r.enter()
	defer r.leave()
	b, err := r.binding(workspace, source, actor)
	if err != nil {
		return Binding{}, err
	}
	switch {
	case b.State == StateActive && b.Approver == actor.ID:
		return *b, nil
	case b.State != StateProposed:
		return Binding{}, refuse(ReasonNotPending)
	case b.Proposer == actor.ID:
		return Binding{}, refuse(ReasonSelfApproval)
	}
	b.State, b.Approver = StateActive, actor.ID
	return *b, nil
}

func (r *Registry) UnbindSource(workspace, source string, actor Actor) (Binding, error) {
	r.enter()
	defer r.leave()
	b, err := r.binding(workspace, source, actor)
	if err != nil {
		return Binding{}, err
	}
	b.State = StateRevoked
	return *b, nil
}

func (r *Registry) sweepSeen(s *space, now time.Time) {
	for s.queued < len(s.queue) {
		key := s.queue[s.queued]
		if found, ok := s.seen[key]; ok {
			if now.Before(found.at.Add(r.limits.InboxRetention)) {
				break
			}
			delete(s.seen, key)
		}
		s.queue[s.queued] = eventKey{}
		s.queued++
	}
	if s.queued > 0 && s.queued*2 >= len(s.queue) {
		s.queue = append([]eventKey(nil), s.queue[s.queued:]...)
		s.queued = 0
	}
}

func (r *Registry) live(s *space, token string, now time.Time) *reservation {
	found := s.reservations[token]
	if found == nil || !now.Before(found.expires) {
		return nil
	}
	return found
}

func (r *Registry) release(s *space, res *reservation) {
	res.gone = true
	if res.fresh {
		s.fresh--
	}
	delete(s.reservations, res.token)
	if s.pending[res.key] == res.token {
		delete(s.pending, res.key)
	}
	if res.scoped && s.subjects[res.subject] == res.token {
		delete(s.subjects, res.subject)
	}
}

func (r *Registry) sweepReservations(s *space, now time.Time) {
	for s.expired < len(s.expiring) {
		res := s.expiring[s.expired]
		if !res.gone && now.Before(res.expires) {
			break
		}
		if !res.gone {
			r.release(s, res)
		}
		s.expiring[s.expired] = nil
		s.expired++
	}
	if s.expired > 0 && s.expired*2 >= len(s.expiring) {
		s.expiring = append([]*reservation(nil), s.expiring[s.expired:]...)
		s.expired = 0
	}
}

func (r *Registry) tidy(s *space) {
	for s.expired < len(s.expiring) && s.expiring[s.expired].gone {
		s.expiring[s.expired] = nil
		s.expired++
	}
	if len(s.expiring)-s.expired > 2*len(s.reservations)+1 {
		kept := make([]*reservation, 0, len(s.reservations))
		for _, res := range s.expiring[s.expired:] {
			if !res.gone {
				kept = append(kept, res)
			}
		}
		s.expiring, s.expired = kept, 0
	}
}

func (r *Registry) Receive(workspace, connection string, verified []string, e Event, actor Actor) (Reception, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(connection) || !actor.valid() || !eventForm(e) {
		return Reception{}, ErrInvalid
	}
	if len(verified) > maxVerified {
		return Reception{}, ErrInvalid
	}
	listed := false
	for _, id := range verified {
		if !validID(id) {
			return Reception{}, ErrInvalid
		}
		listed = listed || id == e.ID
	}
	if err := actor.system(); err != nil {
		return Reception{}, err
	}
	if !listed {
		return Reception{}, refuse(ReasonUnverified)
	}
	s := r.workspaces[workspace]
	if s == nil || s.bindings[e.Source] == nil || s.bindings[e.Source].State != StateActive || s.bindings[e.Source].Connection != connection {
		return Reception{}, refuse(ReasonSourceNotBound)
	}
	r.sweepSeen(s, now)
	key := eventKey{e.Source, e.ID}
	digest := Digest(e)
	if found, ok := s.seen[key]; ok {
		if found.digest == digest {
			return Reception{Decision: DecisionDuplicate}, nil
		}
		return Reception{Decision: DecisionConflict}, nil
	}
	res := &reservation{key: key, event: Event{ID: e.ID, Source: e.Source, Subject: e.Subject, ObjectRevision: e.ObjectRevision}, digest: digest, expires: now.Add(r.limits.Reservation)}
	if e.Subject != "" {
		res.subject, res.scoped = subjectKey{e.Source, e.Subject}, true
		if e.ObjectRevision != "" {
			res.value, _ = parseRevision(e.ObjectRevision)
			res.ranked = true
			if stored, ok := s.revisions[res.subject]; ok {
				switch {
				case stored.value == res.value && stored.id == e.ID && stored.digest == digest:
					return Reception{Decision: DecisionDuplicate}, nil
				case stored.value >= res.value:
					return Reception{Decision: DecisionStale}, nil
				}
			}
		}
		if r.live(s, s.subjects[res.subject], now) != nil {
			return Reception{Decision: DecisionInProgress}, nil
		}
	}
	if r.live(s, s.pending[key], now) != nil {
		return Reception{Decision: DecisionInProgress}, nil
	}
	for _, token := range []string{s.pending[key], s.subjects[res.subject]} {
		if old := s.reservations[token]; old != nil {
			r.release(s, old)
		}
	}
	if len(s.seen)+len(s.reservations) >= r.limits.Seen {
		r.sweepReservations(s, now)
		if len(s.seen)+len(s.reservations) >= r.limits.Seen {
			return Reception{}, refuse(ReasonInboxCapacity)
		}
	}
	if _, known := s.revisions[res.subject]; res.ranked && !known {
		if len(s.revisions)+s.fresh >= r.limits.Revisions {
			return Reception{}, refuse(ReasonRevisionLimit)
		}
		res.fresh = true
		s.fresh++
	}
	r.tokens++
	res.token = "reservation-" + strconv.FormatUint(r.tokens, 10)
	s.reservations[res.token] = res
	s.expiring = append(s.expiring, res)
	r.tidy(s)
	s.pending[key] = res.token
	if res.scoped {
		s.subjects[res.subject] = res.token
	}
	return Reception{Decision: DecisionAccepted, Token: res.token, Event: copyEvent(e)}, nil
}

func (r *Registry) reserved(workspace, token string, actor Actor) (*space, *reservation, error) {
	if !validID(workspace) || token == "" || !actor.valid() {
		return nil, nil, ErrInvalid
	}
	if err := actor.system(); err != nil {
		return nil, nil, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.reservations[token] == nil {
		return nil, nil, ErrStale
	}
	return s, s.reservations[token], nil
}

func (r *Registry) Commit(workspace, token string, actor Actor) (Reception, error) {
	now := r.enter()
	defer r.leave()
	s, res, err := r.reserved(workspace, token, actor)
	if err != nil {
		return Reception{}, err
	}
	r.release(s, res)
	r.tidy(s)
	if stored, ok := s.revisions[res.subject]; res.ranked && ok && stored.value >= res.value {
		return Reception{}, ErrStale
	}
	s.seen[res.key] = seenEvent{digest: res.digest, at: now}
	s.queue = append(s.queue, res.key)
	if res.ranked {
		s.revisions[res.subject] = revision{value: res.value, id: res.key.id, digest: res.digest}
	}
	return Reception{Decision: DecisionAccepted, Event: res.event}, nil
}

func (r *Registry) Abandon(workspace, token string, actor Actor) error {
	r.enter()
	defer r.leave()
	s, res, err := r.reserved(workspace, token, actor)
	if err != nil {
		return err
	}
	r.release(s, res)
	r.tidy(s)
	return nil
}
