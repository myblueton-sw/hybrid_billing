package identity

import "time"

func (h *Boundary) Bootstrap(verifier Verifier, userID string, triple Triple, email string) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !verifier.valid() || !validID(userID) || !triple.valid() || len(email) > maxEmailBytes {
		return ErrInvalid
	}
	if !r.boot.armed || r.boot.consumed || r.boot.locked {
		return refused(ReasonBootstrap)
	}
	if !r.boot.validity.at(r.clock()) {
		return refused(ReasonExpired)
	}
	if !r.boot.verifier.matches(verifier) {
		r.boot.attempts++
		r.boot.locked = r.boot.attempts >= r.limits.Attempts
		return refused(ReasonVerifier)
	}
	r.boot.consumed = true
	r.users[userID] = &user{kind: Human, state: Active, epoch: 1}
	r.link(userID, triple, email)
	r.memberships[r.boot.workspace] = map[string]*membership{userID: {role: Administrator, state: MembershipActive, version: 1}}
	return nil
}

func (h *Boundary) Resolve(triple Triple) (Resolved, error) {
	r, ok := h.open()
	if !ok {
		return Resolved{}, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !triple.valid() {
		return Resolved{}, ErrInvalid
	}
	found := r.identities[triple]
	if found == nil || !found.linked {
		return Resolved{}, refused(ReasonIdentity)
	}
	return Resolved{User: found.user, Email: found.email}, nil
}

func (h *Boundary) OpenSession(triple Triple, verifier Verifier, idle, absolute time.Duration) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !triple.valid() || !verifier.valid() || idle <= 0 || absolute <= 0 || idle > absolute || idle > r.limits.SessionIdle || absolute > r.limits.SessionAbsolute {
		return ErrInvalid
	}
	found := r.identities[triple]
	if found == nil || !found.linked {
		return refused(ReasonIdentity)
	}
	if !r.activeHuman(found.user) {
		return refused(ReasonUser)
	}
	if r.sessions[verifier] != nil {
		return ErrConflict
	}
	now := r.clock()
	live := 0
	for held := range r.byUser[found.user] {
		existing := r.sessions[held]
		if !now.Before(existing.absolute) {
			delete(r.sessions, held)
			delete(r.serials, existing.serial)
			delete(r.byUser[found.user], held)
			continue
		}
		if r.current(existing, now) {
			live++
		}
	}
	if live >= r.limits.SessionsPerUser {
		return ErrLimit
	}
	if !bump(&r.serial) {
		return ErrConflict
	}
	opened := &session{serial: r.serial, user: found.user, epoch: r.users[found.user].epoch, identity: triple, opened: now, idle: idle,
		idleExpiry: now.Add(idle), absolute: now.Add(absolute)}
	r.sessions[verifier], r.serials[opened.serial] = opened, opened
	if r.byUser[found.user] == nil {
		r.byUser[found.user] = map[Verifier]struct{}{}
	}
	r.byUser[found.user][verifier] = struct{}{}
	return nil
}

func (h *Boundary) EndSession(verifier Verifier) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	found := r.sessions[verifier]
	if !verifier.valid() || found == nil {
		return refused(ReasonSession)
	}
	delete(r.sessions, verifier)
	delete(r.serials, found.serial)
	delete(r.byUser[found.user], verifier)
	return nil
}

func (h *Boundary) Access(verifier Verifier, workspace string) (Answer, error) {
	r, ok := h.open()
	if !ok {
		return Answer{}, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) {
		return Answer{}, ErrInvalid
	}
	now := r.clock()
	found, err := r.acting(verifier, now)
	if err != nil {
		return Answer{}, err
	}
	own := r.member(workspace, found.user)
	if own == nil || own.state != MembershipActive {
		return Answer{}, refused(ReasonMembership)
	}
	if extended := now.Add(found.idle); extended.After(found.idleExpiry) {
		found.idleExpiry = extended
	}
	if found.idleExpiry.After(found.absolute) {
		found.idleExpiry = found.absolute
	}
	return Answer{issued: true, instance: r.instance, installation: r.installation, workspace: workspace, user: found.user, kind: Human,
		epoch: found.epoch, role: own.role, notBefore: found.opened, notAfter: found.absolute, version: own.version, serial: found.serial, identity: found.identity}, nil
}

func (h *Boundary) AccessByCredential(credentialID string, verifier Verifier) (Answer, error) {
	r, ok := h.open()
	if !ok {
		return Answer{}, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(credentialID) || !verifier.valid() {
		return Answer{}, ErrInvalid
	}
	found := r.credentials[credentialID]
	if found == nil || found.revoked {
		return Answer{}, refused(ReasonUnknown)
	}
	if !found.validity.at(r.clock()) {
		return Answer{}, refused(ReasonExpired)
	}
	if !found.verifier.matches(verifier) {
		return Answer{}, refused(ReasonVerifier)
	}
	owner := r.users[found.user]
	if owner == nil || owner.kind != Service || owner.state != Active {
		return Answer{}, refused(ReasonUser)
	}
	return Answer{issued: true, instance: r.instance, installation: r.installation, workspace: owner.workspace, user: found.user, kind: Service,
		epoch: owner.epoch, role: ServiceRole, notBefore: found.validity.NotBefore, notAfter: found.validity.NotAfter, credential: credentialID}, nil
}

func (h *Boundary) Recheck(answer Answer) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	now := r.clock()
	owner := r.users[answer.user]
	if !answer.issued || answer.instance != r.instance || now.Before(answer.notBefore) || !now.Before(answer.notAfter) ||
		owner == nil || owner.state != Active || owner.epoch != answer.epoch || owner.kind != answer.kind {
		return refused(ReasonAnswer)
	}
	if answer.kind == Service {
		found := r.credentials[answer.credential]
		if found == nil || found.revoked || found.user != answer.user {
			return refused(ReasonAnswer)
		}
		return nil
	}
	own := r.member(answer.workspace, answer.user)
	linked := r.identities[answer.identity]
	opened := r.serials[answer.serial]
	if own == nil || own.state != MembershipActive || own.role != answer.role || own.version != answer.version ||
		linked == nil || !linked.linked || linked.user != answer.user || opened == nil || opened.user != answer.user {
		return refused(ReasonAnswer)
	}
	return nil
}

func (r *Registry) recent(verifier Verifier) (*session, *user, error) {
	now := r.clock()
	found, err := r.acting(verifier, now)
	if err != nil {
		return nil, nil, err
	}
	if now.Sub(found.opened) > r.limits.Reauthentication {
		return nil, nil, refused(ReasonReauthentication)
	}
	return found, r.users[found.user], nil
}

func (h *Boundary) LinkIdentity(verifier Verifier, triple Triple, email, proof string) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !triple.valid() || len(email) > maxEmailBytes || proof == "" || len(proof) > maxTextBytes {
		return ErrInvalid
	}
	found, owner, err := r.recent(verifier)
	if err != nil {
		return err
	}
	if r.identities[triple] != nil || !bump(&owner.epoch) {
		return ErrConflict
	}
	userID := found.user
	r.drop(userID)
	r.link(userID, triple, email)
	return nil
}

func (h *Boundary) UnlinkIdentity(verifier Verifier, triple Triple) error {
	r, ok := h.open()
	if !ok {
		return ErrUnavailable
	}
	defer r.mu.Unlock()
	if !triple.valid() {
		return ErrInvalid
	}
	found, owner, err := r.recent(verifier)
	if err != nil {
		return err
	}
	target := r.identities[triple]
	if target == nil || !target.linked || target.user != found.user {
		return refused(ReasonIdentity)
	}
	if owner.linked <= 1 {
		return refused(ReasonLastIdentity)
	}
	if r.identities[found.identity].order > target.order {
		return refused(ReasonLaterIdentity)
	}
	if !bump(&owner.epoch) {
		return ErrConflict
	}
	target.linked = false
	owner.linked--
	r.drop(found.user)
	return nil
}
