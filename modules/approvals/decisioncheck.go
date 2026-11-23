package approvals

import "time"

func (r *Registry) CheckDecision(workspace, id string, d Decision, actor Actor) error {
	if !validID(workspace) || !validID(id) || !validActor(actor) || !decisionForm(d) {
		return ErrInvalid
	}
	d.Exclude = append([]Exclusion(nil), d.Exclude...)
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.requests[id] == nil {
		return ErrNotFound
	}
	req := s.requests[id]
	for _, x := range d.Exclude {
		if req.entries[x.Item] == nil {
			return ErrNotFound
		}
	}
	if _, exists := req.deciders[actor.ID]; exists {
		return ErrConflict
	}
	_, err := s.checkDecision(req, d, actor, now)
	return err
}

func (s *space) checkDecision(req *request, d Decision, actor Actor, now time.Time) (map[string]string, error) {
	if actor.Kind != KindHuman {
		return nil, refuse(ReasonAINotAllowed)
	}
	if req.view.State != StatePending {
		return nil, refuse(ReasonNotPending)
	}
	if actor.ID == req.view.Subject.Author || actor.ID == req.view.Requester {
		return nil, refuse(ReasonSelfApproval)
	}
	extra := map[string]string{}
	for _, x := range d.Exclude {
		extra[x.Item] = x.Reason
	}
	if d.Verdict == VerdictApprove {
		if !s.covering(actor.ID, req.view.Subject.Stage, req.view.Subject.Currency, req.covered(extra), now) {
			return nil, refuse(ReasonNoAuthority)
		}
		if len(req.entries) > 0 && req.remaining(extra) == 0 {
			return nil, refuse(ReasonNothingLeft)
		}
	}
	return extra, nil
}
