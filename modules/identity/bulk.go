package identity

import "math"

func (h *Workspace) SuspendMemberships(actor Verifier, workspace string, targets []Target, reason SuspendReason) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || len(targets) == 0 || reason != BulkRevocation {
		return ErrInvalid
	}
	now := r.clock()
	if _, err := r.administrator(actor, workspace, now); err != nil {
		return err
	}
	leaving := map[string]bool{}
	for _, t := range targets {
		found := r.member(workspace, t.User)
		switch {
		case !validID(t.User) || leaving[t.User]:
			return ErrInvalid
		case found == nil:
			return refused(ReasonUnknown)
		case found.version != t.Version || found.version == math.MaxUint64:
			return ErrConflict
		case found.state != MembershipActive:
			return refused(ReasonState)
		}
		leaving[t.User] = true
	}
	if r.strandsAfter(workspace, leaving, now) {
		return ErrLastAdministrator
	}
	for _, t := range targets {
		found := r.member(workspace, t.User)
		found.state, found.suspendedFor = MembershipSuspended, reason
		found.version++
	}
	r.subjectChanged()
	return nil
}

func (h *Installation) SuspendUsers(operator Operator, targets []Target) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !operator.valid() || len(targets) == 0 {
		return ErrInvalid
	}
	now := r.clock()
	leaving := map[string]bool{}
	for _, t := range targets {
		found := r.users[t.User]
		switch {
		case !validID(t.User) || leaving[t.User]:
			return ErrInvalid
		case found == nil || found.kind != Human:
			return refused(ReasonUnknown)
		case found.epoch != t.Version || found.epoch == math.MaxUint64:
			return ErrConflict
		case found.state != Active:
			return refused(ReasonState)
		}
		leaving[t.User] = true
	}
	for workspace := range r.memberships {
		if r.strandsAfter(workspace, leaving, now) {
			return ErrLastAdministrator
		}
	}
	for _, t := range targets {
		found := r.users[t.User]
		found.state = Suspended
		found.epoch++
		r.drop(t.User)
	}
	r.subjectChanged()
	return nil
}
