package approvals

func (r *Registry) CheckCurrent(workspace string, expected Execution, actor Actor) error {
	if !validID(workspace) || !validActor(actor) {
		return ErrInvalid
	}
	expected, ok := r.executionForm(expected)
	if !ok {
		return ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.requests[expected.Request] == nil {
		return ErrNotFound
	}
	return r.check(s, s.requests[expected.Request], expected, now)
}
