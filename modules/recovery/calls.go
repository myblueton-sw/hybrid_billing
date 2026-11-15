package recovery

import (
	"strconv"
	"time"
)

func (r *Registry) located(id string) (*site, error) {
	if !validID(id) {
		return nil, ErrInvalid
	}
	s := r.installations[id]
	if s == nil {
		return nil, ErrNotFound
	}
	return s, nil
}

func (r *Registry) reclaim(s *site, now time.Time) {
	for id, c := range s.open {
		if !now.Before(c.view.Deadline) {
			r.close(s, id, OutcomeUnknown, now)
		}
	}
}

func (r *Registry) close(s *site, id, outcome string, now time.Time) {
	c := s.open[id]
	c.closed = now
	delete(s.open, id)
	s.writers[c.view.Writer].open--
	c.view.Outcome = outcome
	s.closed = append(s.closed, c)
	s.closedByID[id] = c
	if len(s.closed) > maxClosed {
		delete(s.closedByID, s.closed[0].view.ID)
		s.closed = s.closed[1:]
	}
}

func (r *Registry) acting(s *site, writerID string, actor Actor) (*writer, error) {
	w := s.writers[writerID]
	if actor.Kind != KindSystem || actor.ID != writerID || w == nil || w.removed {
		return nil, refuse(ReasonNotWriter)
	}
	return w, nil
}

func stateRefusal(state string) error {
	if state == StateLocked {
		return refuse(ReasonLocked)
	}
	return refuse(ReasonRecovering)
}

func (r *Registry) Permit(id, writerID string, actor Actor) (Token, error) {
	now := r.enter()
	defer r.leave()
	if !actor.valid() || !validID(writerID) {
		return Token{}, ErrInvalid
	}
	s, err := r.located(id)
	if err != nil {
		return Token{}, err
	}
	w, err := r.acting(s, writerID, actor)
	if err != nil {
		return Token{}, err
	}
	if s.state != StateServing {
		return Token{}, stateRefusal(s.state)
	}
	s.sequence++
	t := &permit{writer: writerID, generation: s.generation, expiry: now.Add(w.view.MaxCall)}
	s.tokens[s.sequence] = t
	w.live = append(w.live, s.sequence)
	if len(w.live) > maxLive {
		delete(s.tokens, w.live[0])
		w.live = w.live[1:]
	}
	return Token{Generation: t.generation, Sequence: s.sequence, Expiry: t.expiry}, nil
}

func (t Target) valid() bool {
	return validID(t.Workspace) && validID(t.Key) && (t.Series == "" && t.Number == 0 || validID(t.Series) && t.Number > 0)
}

func (r *Registry) Call(id, writerID string, tok Token, target Target, actor Actor) (CallView, error) {
	now := r.enter()
	defer r.leave()
	if !actor.valid() || !validID(writerID) || !target.valid() {
		return CallView{}, ErrInvalid
	}
	s, err := r.located(id)
	if err != nil {
		return CallView{}, err
	}
	w, err := r.acting(s, writerID, actor)
	if err != nil {
		return CallView{}, err
	}
	r.reclaim(s, now)
	t := s.tokens[tok.Sequence]
	switch {
	case tok.Generation != s.generation || t == nil || t.generation != s.generation:
		return CallView{}, refuse(ReasonFenced)
	case s.state != StateServing:
		return CallView{}, stateRefusal(s.state)
	case t.writer != writerID:
		return CallView{}, refuse(ReasonNotWriter)
	case !now.Before(t.expiry):
		return CallView{}, refuse(ReasonExpired)
	case s.fenced[[2]string{target.Workspace, target.Key}]:
		return CallView{}, refuse(ReasonFencedKey)
	case target.Series != "" && target.Number <= s.high[[2]string{target.Workspace, target.Series}]:
		return CallView{}, refuse(ReasonBelowHighWater)
	case w.open >= maxOpenPerWriter || len(s.open) >= maxOpen:
		return CallView{}, ErrLimit
	}
	s.calls++
	c := &call{view: CallView{ID: "call-" + strconv.FormatUint(s.calls, 10), Writer: writerID, Target: target, Deadline: now.Add(w.view.MaxCall)}, at: now}
	s.open[c.view.ID] = c
	w.open++
	return c.view, nil
}

func (r *Registry) Done(id, callID, outcome string, actor Actor) (CallView, error) {
	now := r.enter()
	defer r.leave()
	if !actor.valid() || !validID(callID) || outcome != OutcomeConfirmed && outcome != OutcomeFailed && outcome != OutcomeUnknown {
		return CallView{}, ErrInvalid
	}
	s, err := r.located(id)
	if err != nil {
		return CallView{}, err
	}
	r.reclaim(s, now)
	c := s.open[callID]
	if c == nil {
		c = s.closedByID[callID]
	}
	switch {
	case c == nil:
		return CallView{}, ErrNotFound
	case actor.Kind != KindSystem || actor.ID != c.view.Writer:
		return CallView{}, refuse(ReasonNotWriter)
	case c.view.Outcome != "" && c.view.Outcome != outcome:
		return CallView{}, ErrConflict
	case c.view.Outcome == "":
		r.close(s, callID, outcome, now)
	}
	return c.view, nil
}

func (r *Registry) stop(s *site, now time.Time) {
	s.stoppedAt = now
	s.inflight = time.Time{}
	s.stopCall = 0
	for _, w := range s.writers {
		if !w.removed && w.view.MaxCall > s.stopCall {
			s.stopCall = w.view.MaxCall
		}
	}
	for _, c := range s.open {
		if c.view.Deadline.After(s.inflight) {
			s.inflight = c.view.Deadline
		}
	}
}

func (r *Registry) Lock(id, reason string, actor Actor) error {
	now := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validText(reason) {
		return ErrInvalid
	}
	s, err := r.located(id)
	if err != nil {
		return err
	}
	switch s.state {
	case StateLocked:
		return nil
	case StateRecovering:
		return refuse(ReasonRecovering)
	}
	r.reclaim(s, now)
	r.bump(s)
	s.state = StateLocked
	r.stop(s, now)
	return nil
}

func (r *Registry) RemoveWriter(id, writerID string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validID(writerID) {
		return ErrInvalid
	}
	s, err := r.located(id)
	if err != nil {
		return err
	}
	w := s.writers[writerID]
	if w == nil {
		return ErrNotFound
	}
	w.removed = true
	for _, seq := range w.live {
		delete(s.tokens, seq)
	}
	w.live = nil
	return nil
}

func (r *Registry) HighWater(id, workspace, series string) (int64, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(series) {
		return 0, ErrInvalid
	}
	s, err := r.located(id)
	if err != nil {
		return 0, err
	}
	return s.high[[2]string{workspace, series}], nil
}

func (r *Registry) Fenced(id, workspace, key string) (bool, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(key) {
		return false, ErrInvalid
	}
	s, err := r.located(id)
	if err != nil {
		return false, err
	}
	return s.fenced[[2]string{workspace, key}], nil
}
