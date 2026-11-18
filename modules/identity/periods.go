package identity

import (
	"sort"
	"time"
)

func (r *Registry) changeRole(actor Verifier, workspace, target string, role Role, until time.Time, expected uint64, now time.Time, dry bool) error {
	if role != Member && role != Administrator {
		return ErrInvalid
	}
	return r.transition(actor, workspace, target, expected, false, now, dry, func(found membership) (membership, error) {
		if ended(found, now) {
			return found, refused(ReasonRolePeriod)
		}
		if !until.IsZero() && !now.Before(until) {
			return found, ErrInvalid
		}
		changed := found.role != role || !found.until.Equal(until)
		found.role, found.until, found.from = role, until, now
		return found, allowed(changed)
	})
}

func (h *Workspace) ChangeRoleUntil(actor Verifier, workspace, target string, role Role, until time.Time, expected uint64, at time.Time) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.changeRole(actor, workspace, target, role, until, expected, at, false)
}

func (r *Registry) setRolePeriod(actor Verifier, workspace, target string, until time.Time, expected uint64, now time.Time, dry bool) error {
	if !until.IsZero() && !now.Before(until) {
		return ErrInvalid
	}
	return r.transition(actor, workspace, target, expected, false, now, dry, func(found membership) (membership, error) {
		if found.until.Equal(until) {
			return found, refused(ReasonState)
		}
		restoring := found.suspendedFor == RolePeriodEnded || found.state == MembershipActive && !effective(&found, now)
		found.until = until
		if restoring {
			found.state, found.suspendedFor, found.from = MembershipActive, SuspendedByCommand, now
		}
		return found, nil
	})
}

func (h *Workspace) SetRolePeriod(actor Verifier, workspace, target string, until time.Time, expected uint64, at time.Time) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.setRolePeriod(actor, workspace, target, until, expected, at, false)
}

func (h *Installation) ExpireRoles(operator Operator, workspace string, now time.Time) ([]Expired, error) {
	r, ok := h.open()
	if !ok {
		return nil, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !operator.valid() || !validID(workspace) {
		return nil, ErrInvalid
	}
	ids := []string{}
	for userID, found := range r.memberships[workspace] {
		if found.state == MembershipActive && !found.until.IsZero() && !now.Before(found.until) {
			ids = append(ids, userID)
		}
	}
	sort.Strings(ids)
	out := []Expired{}
	changed := false
	for _, userID := range ids {
		found := r.memberships[workspace][userID]
		record := Expired{User: userID, Role: found.role, Until: found.until, Version: found.version}
		after := *found
		after.state, after.suspendedFor = MembershipSuspended, RolePeriodEnded
		if found.role == Administrator && r.activeHuman(userID) && r.administrators(workspace, now) == 0 || !bump(&after.version) {
			record.Blocked = true
			out = append(out, record)
			continue
		}
		*found = after
		changed = true
		out = append(out, record)
	}
	if changed {
		r.generation++
	}
	return out, nil
}

func (r *Registry) strandsAfter(workspace string, leaving map[string]bool, now time.Time) bool {
	before := r.administrators(workspace, now)
	if before == 0 {
		return false
	}
	for userID := range leaving {
		if r.activeAdministrator(workspace, userID, now) {
			before--
		}
	}
	return before == 0
}

func (h *Installation) ApplyRecorded(operator Operator, workspace string, rec Recorded) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !operator.valid() || !validID(workspace) || !validID(rec.User) {
		return ErrInvalid
	}
	found := r.users[rec.User]
	if found == nil || found.kind != Human {
		return refused(ReasonUnknown)
	}
	own := r.member(workspace, rec.User)
	if (rec.Membership != "" || rec.SetUntil) && own == nil {
		return refused(ReasonUnknown)
	}
	switch rec.UserState {
	case "":
	case Suspended, Active:
		if found.state != rec.UserState {
			if found.state == Departed || !bump(&found.epoch) {
				return ErrConflict
			}
			found.state, found.operatorBlock = rec.UserState, false
			r.drop(rec.User)
		}
	default:
		return ErrInvalid
	}
	if rec.Membership != "" || rec.SetUntil {
		next := *own
		if rec.SetUntil {
			next.until = rec.Until
		}
		switch rec.Membership {
		case "":
		case MembershipActive, MembershipSuspended:
			if rec.Membership == MembershipActive && own.state != MembershipActive {
				next.from = r.clock()
			}
			next.state, next.suspendedFor = rec.Membership, rec.Reason
		default:
			return ErrInvalid
		}
		if next != *own {
			if !bump(&next.version) {
				return ErrConflict
			}
			*own = next
		}
	}
	r.generation++
	return nil
}

func (h *Installation) Members(operator Operator, workspace string) ([]MemberInfo, error) {
	r, ok := h.open()
	if !ok {
		return nil, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !operator.valid() || !validID(workspace) {
		return nil, ErrInvalid
	}
	now := r.clock()
	ids := []string{}
	for userID := range r.memberships[workspace] {
		if r.member(workspace, userID) != nil && r.users[userID] != nil {
			ids = append(ids, userID)
		}
	}
	sort.Strings(ids)
	out := make([]MemberInfo, 0, len(ids))
	for _, userID := range ids {
		found := r.member(workspace, userID)
		out = append(out, MemberInfo{User: userID, Role: found.role, State: found.state, Version: found.version, UserState: r.users[userID].state, From: found.from,
			Until: found.until, SuspendedFor: found.suspendedFor, Effective: effective(found, now)})
	}
	return out, nil
}
