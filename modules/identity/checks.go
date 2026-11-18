package identity

import "time"

func (h *Workspace) CheckSetRolePeriod(actor Verifier, workspace, target string, until time.Time, expected uint64, at time.Time) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.setRolePeriod(actor, workspace, target, until, expected, at, true)
}

func (h *Workspace) CheckChangeRoleUntil(actor Verifier, workspace, target string, role Role, until time.Time, expected uint64, at time.Time) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.changeRole(actor, workspace, target, role, until, expected, at, true)
}

func (h *Workspace) CheckReinstateMembership(actor Verifier, workspace, target string, expected uint64, at time.Time) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.reinstateMembership(actor, workspace, target, expected, at, true)
}
