package identity

import (
	"sort"
	"time"
)

func (r *Registry) countable(found *invitation, now time.Time) bool {
	if found.state != invitationPending || !now.Before(found.validity.NotAfter) {
		return false
	}
	return r.inviterStands(found, now)
}

func (r *Registry) inviterStands(found *invitation, now time.Time) bool {
	inviter := r.users[found.inviter]
	own := r.member(found.workspace, found.inviter)
	return inviter != nil && inviter.epoch == found.inviterEpoch && own != nil && own.version == found.inviterVersion &&
		r.activeAdministrator(found.workspace, found.inviter, now)
}

func (h *Workspace) Invite(actor Verifier, workspace, invitationID string, role Role, verifier Verifier, validity Validity, invitee string) error {
	return h.InviteUntil(actor, workspace, invitationID, role, verifier, validity, invitee, time.Time{})
}

func (h *Workspace) InviteUntil(actor Verifier, workspace, invitationID string, role Role, verifier Verifier, validity Validity, invitee string, roleUntil time.Time) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	now := r.clock()
	if !validID(workspace) || !validID(invitationID) || !validID(invitee) || (role != Member && role != Administrator) || !verifier.valid() ||
		!validity.valid(r.limits.InvitationValidity) || !now.Before(validity.NotAfter) ||
		!roleUntil.IsZero() && (!roleUntil.After(validity.NotBefore) || !now.Before(roleUntil)) {
		return ErrInvalid
	}
	found, err := r.administrator(actor, workspace, now)
	if err != nil {
		return err
	}
	if r.invitations[invitationID] != nil {
		return ErrConflict
	}
	pending := 0
	for _, existing := range r.invitations {
		if existing.workspace == workspace && r.countable(existing, now) {
			pending++
		}
	}
	if pending >= r.limits.PendingInvitations {
		return ErrLimit
	}
	if !roleUntil.IsZero() && roleUntil.Before(validity.NotAfter) {
		validity.NotAfter = roleUntil
	}
	r.invitations[invitationID] = &invitation{workspace: workspace, role: role, roleUntil: roleUntil, verifier: verifier, validity: validity, inviter: found.user,
		inviterEpoch: found.epoch, inviterVersion: r.member(workspace, found.user).version, invitee: invitee}
	return nil
}

func (h *Workspace) RevokeInvitation(actor Verifier, workspace, invitationID string) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(invitationID) {
		return ErrInvalid
	}
	if _, err := r.administrator(actor, workspace, r.clock()); err != nil {
		return err
	}
	found := r.invitations[invitationID]
	if found == nil || found.workspace != workspace {
		return refused(ReasonUnknown)
	}
	if found.state != invitationPending {
		return refused(ReasonState)
	}
	found.state = invitationRevoked
	return nil
}

func (h *Workspace) Accept(workspace, invitationID string, verifier, actor Verifier) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(invitationID) || !verifier.valid() {
		return ErrInvalid
	}
	now := r.clock()
	acting, err := r.acting(actor, now)
	if err != nil {
		return err
	}
	found := r.invitations[invitationID]
	if found == nil || found.workspace != workspace {
		return refused(ReasonUnknown)
	}
	if found.state != invitationPending {
		return refused(ReasonState)
	}
	if !found.validity.at(now) {
		return refused(ReasonExpired)
	}
	if found.invitee != acting.user {
		return refused(ReasonNotInvitee)
	}
	if !r.inviterStands(found, now) {
		return refused(ReasonInviter)
	}
	if r.member(workspace, acting.user) != nil {
		return refused(ReasonMember)
	}
	if !found.verifier.matches(verifier) {
		found.attempts++
		if found.attempts >= r.limits.Attempts {
			found.state = invitationLocked
		}
		return refused(ReasonVerifier)
	}
	version, joined := uint64(0), uint64(0)
	if previous := r.memberships[workspace][acting.user]; previous != nil {
		version, joined = previous.version, previous.joined
	}
	if !bump(&version) {
		return ErrConflict
	}
	found.state = invitationAccepted
	r.putMembership(workspace, acting.user, &membership{role: found.role, state: MembershipActive, version: version, joined: joined + 1, from: now, until: found.roleUntil})
	r.generation++
	return nil
}

func (r *Registry) transition(actor Verifier, workspace, target string, expected uint64, self bool, now time.Time, dry bool,
	apply func(membership) (membership, error)) error {
	if !validID(workspace) || !validID(target) {
		return ErrInvalid
	}
	acting, err := r.acting(actor, now)
	if err != nil {
		return err
	}
	own := r.member(workspace, acting.user)
	if !(self && acting.user == target && effective(own, now)) && !r.activeAdministrator(workspace, acting.user, now) {
		return refused(ReasonNotAdministrator)
	}
	found := r.member(workspace, target)
	if found == nil {
		return refused(ReasonUnknown)
	}
	if found.version != expected {
		return ErrConflict
	}
	next, err := apply(*found)
	if err != nil {
		return err
	}
	if r.strands(workspace, target, &next, now) {
		return ErrLastAdministrator
	}
	if !bump(&next.version) {
		return ErrConflict
	}
	if dry {
		return nil
	}
	*found = next
	r.generation++
	return nil
}

func allowed(ok bool) error {
	if ok {
		return nil
	}
	return refused(ReasonState)
}

func ended(found membership, now time.Time) bool {
	return found.suspendedFor == RolePeriodEnded || !found.until.IsZero() && !now.Before(found.until)
}

func (h *Workspace) ChangeRole(actor Verifier, workspace, target string, role Role, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	now := r.clock()
	found := r.member(workspace, target)
	until := time.Time{}
	if found != nil {
		until = found.until
	}
	return r.changeRole(actor, workspace, target, role, until, expected, now, false)
}

func (h *Workspace) SuspendMembership(actor Verifier, workspace, target string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.transition(actor, workspace, target, expected, false, r.clock(), false, func(found membership) (membership, error) {
		active := found.state == MembershipActive
		found.state, found.suspendedFor = MembershipSuspended, SuspendedByCommand
		return found, allowed(active)
	})
}

func (h *Workspace) ReinstateMembership(actor Verifier, workspace, target string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.reinstateMembership(actor, workspace, target, expected, r.clock(), false)
}

func (h *Workspace) ReinstateMembershipAt(actor Verifier, workspace, target string, expected uint64, at time.Time) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.reinstateMembership(actor, workspace, target, expected, at, false)
}

func (r *Registry) reinstateMembership(actor Verifier, workspace, target string, expected uint64, now time.Time, dry bool) error {
	return r.transition(actor, workspace, target, expected, false, now, dry, func(found membership) (membership, error) {
		if found.state != MembershipSuspended {
			return found, refused(ReasonState)
		}
		if ended(found, now) {
			return found, refused(ReasonRolePeriod)
		}
		found.state, found.suspendedFor = MembershipActive, SuspendedByCommand
		return found, nil
	})
}

func (h *Workspace) RemoveMembership(actor Verifier, workspace, target string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.transition(actor, workspace, target, expected, true, r.clock(), false, func(found membership) (membership, error) {
		found.state = MembershipRemoved
		return found, nil
	})
}

func (h *Workspace) Members(actor Verifier, workspace, after string, limit int) ([]MemberInfo, error) {
	r, ok := h.open()
	if !ok {
		return nil, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !r.pageable(after, limit) {
		return nil, ErrInvalid
	}
	now := r.clock()
	if _, err := r.administrator(actor, workspace, now); err != nil {
		return nil, err
	}
	index := r.memberIndex[workspace]
	list := []MemberInfo{}
	for i := beyond(index, after); i < len(index) && len(list) < limit; i++ {
		if found := r.member(workspace, index[i]); found != nil {
			list = append(list, r.memberInfo(index[i], found, now))
		}
	}
	return list, nil
}

func (h *Workspace) CreateServiceUser(actor Verifier, workspace, serviceID, owner string) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(serviceID) || !validID(owner) {
		return ErrInvalid
	}
	now := r.clock()
	acting, err := r.administrator(actor, workspace, now)
	if err != nil {
		return err
	}
	holding := r.member(workspace, owner)
	if !effective(holding, now) || !r.activeHuman(owner) {
		return refused(ReasonOwner)
	}
	if r.users[serviceID] != nil {
		return ErrConflict
	}
	r.putUser(serviceID, &user{kind: Service, state: Active, epoch: 1, workspace: workspace, creator: acting.user, owner: owner})
	r.generation++
	return nil
}

func (r *Registry) service(actor Verifier, workspace, serviceID string) (*user, error) {
	if !validID(workspace) || !validID(serviceID) {
		return nil, ErrInvalid
	}
	if _, err := r.administrator(actor, workspace, r.clock()); err != nil {
		return nil, err
	}
	found := r.users[serviceID]
	if found == nil || found.kind != Service || found.workspace != workspace {
		return nil, refused(ReasonUnknown)
	}
	return found, nil
}

func (r *Registry) serviceState(actor Verifier, workspace, serviceID string, expected uint64, from func(*user) error, to UserState) error {
	found, err := r.service(actor, workspace, serviceID)
	if err != nil {
		return err
	}
	if found.epoch != expected {
		return ErrConflict
	}
	if err := from(found); err != nil {
		return err
	}
	if !bump(&found.epoch) {
		return ErrConflict
	}
	found.state, found.operatorBlock = to, false
	r.generation++
	return nil
}

func (h *Workspace) SuspendServiceUser(actor Verifier, workspace, serviceID string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.serviceState(actor, workspace, serviceID, expected, func(found *user) error {
		if found.state != Active {
			return refused(ReasonState)
		}
		return nil
	}, Suspended)
}

func (h *Workspace) ReinstateServiceUser(actor Verifier, workspace, serviceID string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.serviceState(actor, workspace, serviceID, expected, func(found *user) error {
		if found.state != Suspended {
			return refused(ReasonState)
		}
		if found.operatorBlock {
			return refused(ReasonOperatorBlock)
		}
		return nil
	}, Active)
}

func (h *Workspace) RetireServiceUser(actor Verifier, workspace, serviceID string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.serviceState(actor, workspace, serviceID, expected, func(found *user) error {
		if found.state == Departed {
			return refused(ReasonState)
		}
		return nil
	}, Departed)
}

func (h *Workspace) AddCredential(actor Verifier, workspace, serviceID, credentialID string, verifier Verifier, validity Validity) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	now := r.clock()
	if !validID(credentialID) || !verifier.valid() || !validity.valid(r.limits.CredentialValidity) || !now.Before(validity.NotAfter) {
		return ErrInvalid
	}
	found, err := r.service(actor, workspace, serviceID)
	if err != nil {
		return err
	}
	if found.state == Departed {
		return refused(ReasonState)
	}
	if r.credentials[credentialID] != nil {
		return ErrConflict
	}
	held := 0
	for _, id := range r.credentialIndex[serviceID] {
		existing := r.credentials[id]
		if !existing.revoked && now.Before(existing.validity.NotAfter) {
			held++
		}
	}
	if held >= r.limits.CredentialsPerServiceUser {
		return ErrLimit
	}
	if !bump(&r.serial) {
		return ErrConflict
	}
	r.putCredential(credentialID, &credential{serial: r.serial, user: serviceID, verifier: verifier, validity: validity})
	r.pruneCredentials(serviceID, now)
	return nil
}

func (r *Registry) pruneCredentials(serviceID string, now time.Time) {
	ended := []string{}
	for _, id := range r.credentialIndex[serviceID] {
		found := r.credentials[id]
		if found.revoked || !now.Before(found.validity.NotAfter) {
			ended = append(ended, id)
		}
	}
	if len(ended) <= EndedCredentialsKept {
		return
	}
	sort.Slice(ended, func(a, b int) bool {
		x, y := r.credentials[ended[a]].validity.NotAfter, r.credentials[ended[b]].validity.NotAfter
		if !x.Equal(y) {
			return x.After(y)
		}
		return ended[a] < ended[b]
	})
	for _, id := range ended[EndedCredentialsKept:] {
		r.removeCredential(id)
	}
}

func (h *Workspace) RevokeCredential(actor Verifier, workspace, credentialID string) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(credentialID) {
		return ErrInvalid
	}
	if _, err := r.administrator(actor, workspace, r.clock()); err != nil {
		return err
	}
	found := r.credentials[credentialID]
	if found == nil || r.users[found.user] == nil || r.users[found.user].workspace != workspace {
		return refused(ReasonUnknown)
	}
	if found.revoked {
		return refused(ReasonState)
	}
	found.revoked = true
	r.pruneCredentials(found.user, r.clock())
	return nil
}

func (h *Workspace) Credential(actor Verifier, workspace, credentialID string) (string, CredentialInfo, error) {
	r, ok := h.open()
	if !ok {
		return "", CredentialInfo{}, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !validID(credentialID) {
		return "", CredentialInfo{}, ErrInvalid
	}
	if _, err := r.administrator(actor, workspace, r.clock()); err != nil {
		return "", CredentialInfo{}, err
	}
	found := r.credentials[credentialID]
	if found == nil || r.users[found.user] == nil || r.users[found.user].workspace != workspace {
		return "", CredentialInfo{}, refused(ReasonUnknown)
	}
	return found.user, CredentialInfo{ID: credentialID, Validity: found.validity, Revoked: found.revoked}, nil
}

func (h *Workspace) Credentials(actor Verifier, workspace, serviceID string) ([]CredentialInfo, error) {
	r, ok := h.open()
	if !ok {
		return nil, ErrUnavailable
	}
	defer r.mu.Unlock()
	if _, err := r.service(actor, workspace, serviceID); err != nil {
		return nil, err
	}
	return r.credentialInfos(r.credentialIndex[serviceID]), nil
}
