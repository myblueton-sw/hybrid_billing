package identity

import "time"

func (r *Registry) countable(found *invitation, now time.Time) bool {
	if found.state != invitationPending || !now.Before(found.validity.NotAfter) {
		return false
	}
	return r.inviterStands(found)
}

func (r *Registry) inviterStands(found *invitation) bool {
	inviter := r.users[found.inviter]
	own := r.member(found.workspace, found.inviter)
	return inviter != nil && inviter.epoch == found.inviterEpoch && own != nil && own.version == found.inviterVersion &&
		r.activeAdministrator(found.workspace, found.inviter)
}

func (h *Workspace) Invite(actor Verifier, workspace, invitationID string, role Role, verifier Verifier, validity Validity, invitee string) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	now := r.clock()
	if !validID(workspace) || !validID(invitationID) || !validID(invitee) || (role != Member && role != Administrator) || !verifier.valid() ||
		!validity.valid(r.limits.InvitationValidity) || !now.Before(validity.NotAfter) {
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
	r.invitations[invitationID] = &invitation{workspace: workspace, role: role, verifier: verifier, validity: validity, inviter: found.user,
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
	if !r.inviterStands(found) {
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
	r.memberships[workspace][acting.user] = &membership{role: found.role, state: MembershipActive, version: version, joined: joined + 1}
	r.generation++
	return nil
}

func (r *Registry) transition(actor Verifier, workspace, target string, expected uint64, self bool, apply func(membership) (membership, bool)) error {
	if !validID(workspace) || !validID(target) {
		return ErrInvalid
	}
	now := r.clock()
	acting, err := r.acting(actor, now)
	if err != nil {
		return err
	}
	own := r.member(workspace, acting.user)
	if !(self && acting.user == target && own != nil && own.state == MembershipActive) && !r.activeAdministrator(workspace, acting.user) {
		return refused(ReasonNotAdministrator)
	}
	found := r.member(workspace, target)
	if found == nil {
		return refused(ReasonUnknown)
	}
	if found.version != expected {
		return ErrConflict
	}
	next, allowed := apply(*found)
	if !allowed {
		return refused(ReasonState)
	}
	if r.strands(workspace, target, &next) {
		return ErrLastAdministrator
	}
	if !bump(&next.version) {
		return ErrConflict
	}
	*found = next
	r.generation++
	return nil
}

func (h *Workspace) ChangeRole(actor Verifier, workspace, target string, role Role, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if role != Member && role != Administrator {
		return ErrInvalid
	}
	return r.transition(actor, workspace, target, expected, false, func(found membership) (membership, bool) {
		changed := found.role != role
		found.role = role
		return found, changed
	})
}

func (h *Workspace) SuspendMembership(actor Verifier, workspace, target string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.transition(actor, workspace, target, expected, false, func(found membership) (membership, bool) {
		allowed := found.state == MembershipActive
		found.state = MembershipSuspended
		return found, allowed
	})
}

func (h *Workspace) ReinstateMembership(actor Verifier, workspace, target string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.transition(actor, workspace, target, expected, false, func(found membership) (membership, bool) {
		allowed := found.state == MembershipSuspended
		found.state = MembershipActive
		return found, allowed
	})
}

func (h *Workspace) RemoveMembership(actor Verifier, workspace, target string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.transition(actor, workspace, target, expected, true, func(found membership) (membership, bool) {
		found.state = MembershipRemoved
		return found, true
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
	if _, err := r.administrator(actor, workspace, r.clock()); err != nil {
		return nil, err
	}
	ids := []string{}
	for userID := range r.memberships[workspace] {
		if r.member(workspace, userID) != nil {
			ids = append(ids, userID)
		}
	}
	list := []MemberInfo{}
	for _, userID := range page(ids, after, limit) {
		found := r.member(workspace, userID)
		list = append(list, MemberInfo{User: userID, Role: found.role, State: found.state, Version: found.version, UserState: r.users[userID].state})
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
	acting, err := r.administrator(actor, workspace, r.clock())
	if err != nil {
		return err
	}
	holding := r.member(workspace, owner)
	if holding == nil || holding.state != MembershipActive || !r.activeHuman(owner) {
		return refused(ReasonOwner)
	}
	if r.users[serviceID] != nil {
		return ErrConflict
	}
	r.users[serviceID] = &user{kind: Service, state: Active, epoch: 1, workspace: workspace, creator: acting.user, owner: owner}
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

func (h *Workspace) ServiceUsers(actor Verifier, workspace, after string, limit int) ([]ServiceInfo, error) {
	r, ok := h.open()
	if !ok {
		return nil, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !r.pageable(after, limit) {
		return nil, ErrInvalid
	}
	if _, err := r.administrator(actor, workspace, r.clock()); err != nil {
		return nil, err
	}
	ids := []string{}
	for id, found := range r.users {
		if found.kind == Service && found.workspace == workspace {
			ids = append(ids, id)
		}
	}
	list := []ServiceInfo{}
	for _, id := range page(ids, after, limit) {
		found := r.users[id]
		list = append(list, ServiceInfo{ID: id, State: found.state, Epoch: found.epoch, Creator: found.creator, Owner: found.owner})
	}
	return list, nil
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
	for _, existing := range r.credentials {
		if existing.user == serviceID && !existing.revoked && now.Before(existing.validity.NotAfter) {
			held++
		}
	}
	if held >= r.limits.CredentialsPerServiceUser {
		return ErrLimit
	}
	r.credentials[credentialID] = &credential{user: serviceID, verifier: verifier, validity: validity}
	return nil
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
	return nil
}
