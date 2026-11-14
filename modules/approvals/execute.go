package approvals

import "time"

func (r *Registry) executionForm(e Execution) (Execution, bool) {
	if !validID(e.ID) || !validID(e.Request) || !stages[e.Stage] || !validID(e.Target) || !validCurrency(e.Currency) || len(e.Digests) == 0 {
		return e, false
	}
	if e.Item == "" && e.ItemDigest != "" || e.Item != "" && (!validID(e.Item) || !validDigest(e.ItemDigest)) {
		return e, false
	}
	for k, v := range e.Digests {
		if !digestKinds[k] || !validDigest(v) {
			return e, false
		}
	}
	_, printed, ok := r.number(e.Amount, true)
	e.Amount = printed
	e.Digests = copyDigests(e.Digests)
	return e, ok
}

func sameExecution(a, b Execution) bool {
	return a.ID == b.ID && a.Request == b.Request && a.Stage == b.Stage && a.Target == b.Target && a.Item == b.Item && a.ItemDigest == b.ItemDigest &&
		a.Amount == b.Amount && a.Currency == b.Currency && sameDigests(a.Digests, b.Digests)
}

func viewExecution(x *execution) ExecutionView {
	v := x.view
	v.Execution.Digests = copyDigests(x.view.Execution.Digests)
	return v
}

func (r *Registry) check(s *space, req *request, e Execution, now time.Time) error {
	switch req.view.State {
	case StatePending, StateRejected:
		return refuse(ReasonNotApproved)
	case StateSuperseded:
		return refuse(ReasonSuperseded)
	case StateWithdrawn:
		return refuse(ReasonWithdrawn)
	}
	sub := req.view.Subject
	policy := s.policies[sub.Stage]
	if policy.Revision != req.view.Policy {
		return refuse(ReasonPolicyChanged)
	}
	if now.Sub(req.view.ApprovedAt) > policy.Validity {
		return refuse(ReasonExpired)
	}
	if e.Stage != sub.Stage || e.Target != sub.Target || e.Currency != sub.Currency || !sameDigests(e.Digests, sub.Digests) || (e.Item == "") != (len(req.entries) == 0) {
		return refuse(ReasonSubjectChanged)
	}
	if e.Item == "" {
		if e.Amount != sub.Amount {
			return refuse(ReasonSubjectChanged)
		}
	} else if en := req.entries[e.Item]; en != nil && (en.e.Amount != e.Amount || en.e.Digest != e.ItemDigest) {
		return refuse(ReasonSubjectChanged)
	}
	covered := req.covered(nil)
	for _, approver := range req.approvers {
		if !s.covering(approver, sub.Stage, sub.Currency, covered, now) {
			return refuse(ReasonAuthorityLost)
		}
	}
	if e.Item != "" && (req.entries[e.Item] == nil || req.excluded[e.Item] != "") {
		return refuse(ReasonItemIneligible)
	}
	return nil
}

func executedKey(e Execution) [2]string {
	if e.Item != "" {
		return [2]string{e.Stage, e.Item}
	}
	return [2]string{e.Stage, e.Target}
}

func (r *Registry) Execute(workspace string, e Execution, actor Actor) (ExecutionView, error) {
	if !validID(workspace) || !validActor(actor) {
		return ExecutionView{}, ErrInvalid
	}
	e, ok := r.executionForm(e)
	if !ok {
		return ExecutionView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.requests[e.Request] == nil {
		return ExecutionView{}, ErrNotFound
	}
	req := s.requests[e.Request]
	if old := s.executions[e.ID]; old != nil {
		if !sameExecution(old.view.Execution, e) || old.view.Actor != actor.ID {
			return ExecutionView{}, ErrConflict
		}
		if old.view.Outcome.Result == ResultSucceeded {
			return viewExecution(old), nil
		}
		if err := r.check(s, req, e, now); err != nil {
			return ExecutionView{}, err
		}
		return viewExecution(old), nil
	}
	if err := r.check(s, req, e, now); err != nil {
		return ExecutionView{}, err
	}
	key := executedKey(e)
	if s.keys[key] != "" {
		return ExecutionView{}, refuse(ReasonAlreadyExecuted)
	}
	if len(s.executions) >= r.limits.Executions {
		return ExecutionView{}, ErrLimit
	}
	made := &execution{req: req, view: ExecutionView{Execution: e, Actor: actor.ID, At: now}}
	made.view.Digest = executionDigest(workspace, e, actor.ID, now)
	s.executions[e.ID] = made
	s.keys[key] = e.ID
	marker := e.Target
	if e.Item != "" {
		marker = e.Item
	}
	req.executed[marker] = e.ID
	return viewExecution(made), nil
}

func (r *Registry) Report(workspace, id string, o Outcome, actor Actor) (ExecutionView, error) {
	if !validID(workspace) || !validID(id) || !validActor(actor) {
		return ExecutionView{}, ErrInvalid
	}
	switch o.Result {
	case ResultSucceeded:
		if o.Reason != "" {
			return ExecutionView{}, ErrInvalid
		}
	case ResultFailed:
		if !failReasons[o.Reason] {
			return ExecutionView{}, ErrInvalid
		}
	default:
		return ExecutionView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.executions[id] == nil {
		return ExecutionView{}, ErrNotFound
	}
	x := s.executions[id]
	if x.view.Outcome == o && x.view.Actor == actor.ID {
		return viewExecution(x), nil
	}
	if actor.ID != x.view.Actor {
		return ExecutionView{}, ErrRejected
	}
	switch {
	case x.view.Outcome == (Outcome{}):
	case x.view.Outcome.Result == ResultFailed && x.view.Earlier == (Outcome{}) && o.Result == ResultSucceeded:
		x.view.Earlier = x.view.Outcome
	default:
		return ExecutionView{}, ErrConflict
	}
	x.view.Outcome, x.view.Reported = o, now
	return viewExecution(x), nil
}

func (r *Registry) ExecutionOf(workspace, id string) (ExecutionView, error) {
	if !validID(workspace) || !validID(id) {
		return ExecutionView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.executions[id] == nil {
		return ExecutionView{}, ErrNotFound
	}
	return viewExecution(s.executions[id]), nil
}
