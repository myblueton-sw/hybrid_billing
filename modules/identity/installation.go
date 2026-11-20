package identity

import "math"

func (h *Installation) ArmBootstrap(verifier Verifier, validity Validity, workspace string) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !verifier.valid() || !validID(workspace) || !validity.valid(r.limits.BootstrapValidity) || !r.clock().Before(validity.NotAfter) {
		return ErrInvalid
	}
	if r.boot.armed || len(r.users) != 0 {
		return refused(ReasonBootstrap)
	}
	r.boot = bootstrap{armed: true, verifier: verifier, validity: validity, workspace: workspace}
	return nil
}

func (r *Registry) link(userID string, triple Triple, email string) {
	owner := r.users[userID]
	bump(&owner.links)
	owner.linked++
	r.identities[triple] = &identityRecord{user: userID, email: email, linked: true, order: owner.links}
}

func (h *Installation) CreateUser(operator Operator, userID string, triple Triple, email string) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !operator.valid() || !validID(userID) || !triple.valid() || len(email) > maxEmailBytes {
		return ErrInvalid
	}
	if !r.boot.consumed {
		return refused(ReasonBootstrap)
	}
	if r.users[userID] != nil || r.identities[triple] != nil {
		return ErrConflict
	}
	r.putUser(userID, &user{kind: Human, state: Active, epoch: 1})
	r.link(userID, triple, email)
	r.subjectChanged()
	return nil
}

func (r *Registry) change(operator Operator, userID string, expected uint64) (*user, error) {
	if !operator.valid() || !validID(userID) {
		return nil, ErrInvalid
	}
	found := r.users[userID]
	if found == nil {
		return nil, refused(ReasonUnknown)
	}
	if found.epoch != expected {
		return nil, ErrConflict
	}
	return found, nil
}

func (h *Installation) SuspendUser(operator Operator, userID string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	found, err := r.change(operator, userID, expected)
	if err != nil {
		return err
	}
	blocking := found.kind == Service && found.state == Suspended && !found.operatorBlock
	if (found.state != Active && !blocking) || !bump(&found.epoch) {
		return ErrConflict
	}
	found.state = Suspended
	found.operatorBlock = found.kind == Service
	r.drop(userID)
	r.subjectChanged()
	return nil
}

func (h *Installation) ReinstateUser(operator Operator, userID string, expected uint64) error {
	return h.reinstateUser(operator, userID, expected, false)
}

func (h *Installation) CheckReinstateUser(operator Operator, userID string, expected uint64) error {
	return h.reinstateUser(operator, userID, expected, true)
}

func (h *Installation) reinstateUser(operator Operator, userID string, expected uint64, dry bool) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	found, err := r.change(operator, userID, expected)
	if err != nil {
		return err
	}
	if found.state != Suspended || found.epoch == math.MaxUint64 {
		return ErrConflict
	}
	if dry {
		return nil
	}
	found.epoch++
	found.state, found.operatorBlock = Active, false
	r.subjectChanged()
	return nil
}

func (h *Installation) DepartUser(operator Operator, userID string, expected uint64) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	found, err := r.change(operator, userID, expected)
	if err != nil {
		return err
	}
	if found.state == Departed {
		return ErrConflict
	}
	now := r.clock()
	for workspace := range r.memberships {
		if r.member(workspace, userID) != nil && (r.strands(workspace, userID, nil, now) || r.onlySeat(workspace, userID, now)) {
			return ErrLastAdministrator
		}
	}
	if !bump(&found.epoch) {
		return ErrConflict
	}
	for workspace := range r.memberships {
		if own := r.member(workspace, userID); own != nil {
			own.state = MembershipRemoved
			bump(&own.version)
		}
	}
	for _, linked := range r.identities {
		if linked.user == userID {
			linked.linked = false
		}
	}
	found.state, found.linked, found.operatorBlock = Departed, 0, false
	r.drop(userID)
	r.subjectChanged()
	return nil
}

func (h *Installation) Users(after string, limit int) ([]UserInfo, error) {
	r, ok := h.open()
	if !ok {
		return nil, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !r.pageable(after, limit) {
		return nil, ErrInvalid
	}
	start := beyond(r.userIndex, after)
	list := []UserInfo{}
	for _, id := range r.userIndex[start : start+min(limit, len(r.userIndex)-start)] {
		list = append(list, userInfo(id, r.users[id]))
	}
	return list, nil
}

func (h *Installation) WithoutAdministrator() ([]string, error) {
	r, ok := h.open()
	if !ok {
		return nil, ErrUnavailable
	}
	defer r.mu.Unlock()
	return r.withoutAdministrator(r.clock()), nil
}
