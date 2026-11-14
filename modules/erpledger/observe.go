package erpledger

import "time"

func (o Observation) valid() bool {
	if !validID(o.ID) || !validID(o.Connection) || o.Key == "" && o.External == "" || o.Key != "" && !validID(o.Key) ||
		o.External != "" && !validID(o.External) || o.Revision < 1 || !observed[o.State] || !sources[o.Source] {
		return false
	}
	if o.State == StateAbsent && o.Source == SourceWebhook {
		return false
	}
	return o.External != "" || o.State != StateDraft && o.State != StatePosted
}

func externalKey(connection, external string) string { return connection + "|" + external }

func (r *Registry) Observe(workspace string, o Observation, actor Actor) (ObservationView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !o.valid() || !actor.valid() {
		return ObservationView{}, ErrInvalid
	}
	s, c := r.connection(workspace, o.Connection)
	if c == nil {
		return ObservationView{}, ErrNotFound
	}
	if actor.Kind == KindAssistant {
		return ObservationView{}, refuse(ReasonAINotAllowed)
	}
	id := externalKey(o.Connection, o.ID)
	if seen := s.observations[id]; seen != nil {
		if seen.Observation == o {
			return *seen, nil
		}
		return ObservationView{}, ErrConflict
	}
	var byKey, byRef *submission
	if o.Key != "" {
		byKey = s.submissions[o.Key]
		if byKey != nil && byKey.conn != c {
			return ObservationView{}, ErrConflict
		}
	}
	if o.External != "" {
		byRef = s.externals[externalKey(o.Connection, o.External)]
	}
	if byKey != nil && byRef != nil && byKey != byRef {
		return ObservationView{}, ErrConflict
	}
	target := byKey
	if target == nil {
		target = byRef
	}
	if target != nil && target.view.State == StatePrepared {
		return ObservationView{}, refuse(ReasonWrongState)
	}
	if target != nil && o.External != "" && target.view.External != "" && o.External != target.view.External {
		return ObservationView{}, ErrConflict
	}
	if len(s.observations) >= r.limits.Observations {
		return ObservationView{}, ErrLimit
	}
	view := &ObservationView{Observation: o, Actor: actor.ID, At: now}
	s.observations[id] = view
	if target == nil {
		view.Outcome = "orphan"
		s.orphans = append(s.orphans, *view)
		return *view, nil
	}
	view.Outcome = s.apply(target, o, now)
	return *view, nil
}

func (s *space) apply(sub *submission, o Observation, now time.Time) string {
	if o.Revision <= sub.view.Revision {
		return "stale"
	}
	if o.State == StateAbsent && (sub.view.State == StateSent || sub.view.State == StateDraft) {
		sub.view.Revision, sub.view.LastState, sub.view.LastRef = o.Revision, StateUnknown, o.External
		sub.view.State = StateUnknown
		s.polled(sub, o, now)
		return "premature"
	}
	sub.view.Revision, sub.view.LastState, sub.view.LastRef = o.Revision, o.State, o.External
	s.polled(sub, o, now)
	current := sub.view.State
	switch {
	case o.State == current:
		return "repeated"
	case terminal[current]:
		sub.view.Diverged = true
		s.reference(sub, o.External)
		if retired[current] {
			s.spread(sub)
		}
		return "diverged"
	}
	sub.view.State = o.State
	s.reference(sub, o.External)
	if o.State == StatePosted {
		sub.view.Posted = now
	}
	return "applied"
}

func (s *space) polled(sub *submission, o Observation, now time.Time) {
	if o.Source == SourcePoll && sub.view.Watch != nil {
		w := sub.view.Watch
		w.Last, w.Next, w.Retries, w.Revision = now, now.Add(sub.conn.view.Connection.Interval), 0, o.Revision
	}
}

func (s *space) reference(sub *submission, external string) {
	if external != "" && sub.view.External == "" {
		sub.view.External = external
		s.externals[externalKey(sub.conn.view.Connection.ID, external)] = sub
	}
}

func (s *space) spread(sub *submission) {
	for _, l := range sub.view.Submission.Lines {
		for _, other := range s.bySource[sourceKey(sub.view.Submission.Purpose, l.Source)] {
			if other != sub && other.live() {
				other.view.Diverged = true
			}
		}
	}
}

func (s *space) anchored(sub *submission) bool {
	for _, l := range sub.view.Submission.Lines {
		for _, a := range s.covering[l.Source] {
			if blocks(sub.view.Submission.Purpose, a.Kind) && !(sub.view.Submission.Path == PathOrderBased && a.ID == sub.view.Submission.Anchor) {
				return true
			}
		}
	}
	return false
}

func (s *space) shares(sub *submission) bool {
	for _, l := range sub.view.Submission.Lines {
		for _, other := range s.bySource[sourceKey(sub.view.Submission.Purpose, l.Source)] {
			if other != sub && other.live() {
				return true
			}
		}
	}
	return false
}

func (r *Registry) Resolve(workspace, key string, res Resolution, actor Actor) (SubmissionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(key) || res.Outcome != ResolveAccept && res.Outcome != ResolveKeep && res.Outcome != ResolveSuperseded || !validID(res.Evidence) || !actor.valid() {
		return SubmissionView{}, ErrInvalid
	}
	s, sub, err := r.find(workspace, key)
	if err != nil {
		return SubmissionView{}, err
	}
	if seen := sub.view.Resolution; !sub.view.Diverged && seen != nil && seen.Resolution == res && actor.is(seen.Actor) {
		return sub.copy(), nil
	}
	if err := actor.refusal(); err != nil {
		return SubmissionView{}, err
	}
	if !sub.view.Diverged || res.Outcome == ResolveAccept && sub.view.LastState == "" {
		return SubmissionView{}, refuse(ReasonWrongState)
	}
	if actor.ID == sub.view.Author {
		return SubmissionView{}, refuse(ReasonSelfApproval)
	}
	if res.Outcome != ResolveSuperseded && s.anchored(sub) {
		return SubmissionView{}, refuse(ReasonAlreadyReflected)
	}
	if res.Outcome == ResolveKeep && retired[sub.view.State] && sub.view.LastState != "" && sub.view.LastState != sub.view.State {
		return SubmissionView{}, refuse(ReasonWrongState)
	}
	state := sub.view.State
	switch res.Outcome {
	case ResolveAccept:
		state = sub.view.LastState
	case ResolveSuperseded:
		state = StateSuperseded
	}
	if !retired[state] && s.shares(sub) {
		return SubmissionView{}, refuse(ReasonAlreadySubmitted)
	}
	if res.Outcome == ResolveSuperseded {
		sub.view.State = state
	}
	if res.Outcome == ResolveAccept && state != sub.view.State {
		sub.view.State = state
		s.reference(sub, sub.view.LastRef)
		if state == StatePosted {
			sub.view.Posted = now
		}
		if sub.view.Watch == nil {
			sub.view.Watch = &Watch{Next: now.Add(sub.conn.view.Connection.Interval), Owner: sub.conn.view.Connection.Owner}
		}
	}
	sub.view.Diverged = false
	sub.view.Resolution = &ResolutionView{Resolution: res, Actor: actor.ID, At: now}
	return sub.copy(), nil
}

func (r *Registry) Orphans(workspace string) ([]ObservationView, error) {
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return nil, ErrNotFound
	}
	return append([]ObservationView(nil), s.orphans...), nil
}
