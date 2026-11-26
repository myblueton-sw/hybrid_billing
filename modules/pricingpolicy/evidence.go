package pricingpolicy

import "time"

func (r *Registry) CheckEvidence(scope Scope, id, want string, from, to time.Time) (Record, error) {
	if !validScope(scope) || !validID(id) || !validDigest(want) || !interval(from, to) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.evidenceLocked(scope, id, want, from, to)
}

func (r *Registry) evidenceLocked(scope Scope, id, want string, from, to time.Time) (Record, error) {
	record, ok := r.records[[2]string{scope.Workspace, id}]
	if !ok {
		return Record{}, ErrNotFound
	}
	p := record.Proposal
	if record.Digest != want || p.Spec.Scope != scope {
		return Record{}, ErrConflict
	}
	if record.State != Approved || from.Before(p.Spec.From) || to.After(p.Spec.To) {
		return Record{}, ErrRejected
	}
	if err := r.current(p); err != nil {
		return Record{}, err
	}
	if _, err := r.now(); err != nil {
		return Record{}, err
	}
	return clone(record), nil
}
