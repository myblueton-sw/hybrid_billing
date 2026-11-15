package licensemode

import (
	"crypto/ed25519"
	"time"
)

const forever = maxTime + 1

func (r *Registry) Install(id, product string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	switch {
	case !validID(id) || !validID(product):
		return ErrInvalid
	case r.installations[id] != nil:
		return ErrConflict
	case len(r.installations) >= maxInstallations:
		return ErrLimit
	}
	r.installations[id] = &installation{id: id, product: product}
	return nil
}

func (r *Registry) SelectMode(id, mode string, actor Actor) error {
	now := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validID(id) || !oneOf(mode, ModeNoncommercial, ModeEvaluation, ModeCommercial) {
		return ErrInvalid
	}
	inst := r.installations[id]
	if inst == nil {
		return ErrNotFound
	}
	inst.mode = mode
	inst.modes = append(inst.modes, ModeChange{Mode: mode, By: actor.ID, At: now})
	if len(inst.modes) > maxHistory {
		inst.modes = inst.modes[1:]
	}
	return nil
}

func (r *Registry) verify(s Signed, kind string, inst *installation, local time.Time) (Claims, time.Time, string) {
	c, ok := Decode(s.Body)
	if !ok || len(s.Signature) != ed25519.SignatureSize {
		return Claims{}, time.Time{}, OutcomeInvalid
	}
	k := r.keys[c.KeyID]
	if k == nil {
		return Claims{}, time.Time{}, OutcomeUnverifiable
	}
	if !ed25519.Verify(k.public, signedMessage(s.Body), s.Signature) {
		return Claims{}, time.Time{}, OutcomeInvalid
	}
	now, outcome := r.authoritative(local)
	if outcome != "" {
		return Claims{}, time.Time{}, outcome
	}
	at := now.Unix()
	switch {
	case k.view.Issuer != c.Issuer || k.compromised || c.IssuedAt < k.view.NotBefore || c.IssuedAt >= k.view.NotAfter || c.IssuedAt > at ||
		at < k.view.NotBefore || at >= k.view.AcceptUntil:
		return Claims{}, now, OutcomeUnverifiable
	case c.Kind != kind:
		return Claims{}, now, OutcomeWrongKind
	case c.Installation != inst.id || c.Product != inst.product:
		return Claims{}, now, OutcomeScopeMismatch
	}
	return c, now, ""
}

func (r *Registry) compromised(keyID string) bool { return r.keys[keyID].compromised }

func (r *Registry) latch(inst *installation, now time.Time) {
	if a := inst.activation; a != nil && !now.Before(a.view.End) {
		a.expired = true
	}
	for _, e := range inst.accepted {
		if !e.claims.Perpetual && e.claims.End <= now.Unix() {
			e.ended = true
		}
	}
}

func (r *Registry) Activate(id string, s Signed, acceptance Acceptance, actor Actor) (Activation, error) {
	local := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return Activation{}, err
	}
	if !validID(id) || acceptance.Actor != actor.ID || !validText(acceptance.Terms) {
		return Activation{}, ErrInvalid
	}
	inst := r.installations[id]
	if inst == nil {
		return Activation{}, ErrNotFound
	}
	c, now, outcome := r.verify(s, DocEvaluation, inst, local)
	if outcome != "" {
		return Activation{}, refuse(outcome)
	}
	r.latch(inst, now)
	if a := r.grants[c.ID]; a != nil {
		return copyActivation(a.view), nil
	}
	switch {
	case inst.mode != ModeEvaluation:
		return Activation{}, refuse(ReasonModeMismatch)
	case acceptance.Terms != c.Terms:
		return Activation{}, refuse(ReasonTermsMismatch)
	case c.ActivateBy != 0 && now.Unix() >= c.ActivateBy:
		return Activation{}, refuse(ReasonStaleGrant)
	case inst.activation != nil:
		return Activation{}, refuse(ReasonAlreadyEvaluated)
	case len(r.grants) >= maxRecords:
		return Activation{}, ErrLimit
	}
	begin := time.Unix(now.Unix(), 0).UTC()
	a := &activation{view: Activation{Grant: c.ID, Subject: c.SubjectID, Terms: c.Terms, Scope: append([]string(nil), c.Scope...), AcceptedBy: actor.ID, Start: begin,
		End: begin.Add(evaluationLength), KeyID: c.KeyID}}
	r.grants[c.ID] = a
	inst.activation = a
	return copyActivation(a.view), nil
}

func copyActivation(a Activation) Activation {
	a.Scope = append([]string(nil), a.Scope...)
	return a
}

func copyClaims(c Claims) Claims {
	c.Scope = append([]string(nil), c.Scope...)
	return c
}

func end(c Claims) int64 {
	if c.Perpetual {
		return forever
	}
	return c.End
}

func (r *Registry) live(inst *installation) []*entitlement {
	out := []*entitlement{}
	for _, e := range inst.accepted {
		if !r.compromised(e.claims.KeyID) {
			out = append(out, e)
		}
	}
	return out
}

func effectiveEnd(e *entitlement, live []*entitlement) int64 {
	out := end(e.claims)
	for _, s := range live {
		if s.order > e.order && s.claims.Supersedes == e.claims.Agreement && s.claims.SubjectID == e.claims.SubjectID {
			out = min(out, max(e.claims.Start, s.claims.Start))
		}
	}
	return out
}

func (r *Registry) current(inst *installation, at int64) (*entitlement, bool) {
	live := r.live(inst)
	var best *entitlement
	for _, e := range live {
		if !e.ended && e.claims.Start <= at && at < effectiveEnd(e, live) && (best == nil || e.claims.IssuedAt > best.claims.IssuedAt) {
			best = e
		}
	}
	if best != nil {
		return best, true
	}
	for _, e := range live {
		if best == nil || e.claims.IssuedAt > best.claims.IssuedAt {
			best = e
		}
	}
	return best, false
}

func (r *Registry) Register(id string, s Signed, actor Actor) (string, error) {
	local := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return "", err
	}
	if !validID(id) {
		return "", ErrInvalid
	}
	inst := r.installations[id]
	if inst == nil {
		return "", ErrNotFound
	}
	outcome, err := r.register(inst, s, local)
	if err != nil {
		return "", err
	}
	var e encoder
	e.text(string(s.Body))
	e.text(string(s.Signature))
	inst.submissions = append(inst.submissions, Submission{Outcome: outcome, Digest: digest(e.out.Bytes()), At: local})
	if len(inst.submissions) > maxHistory {
		inst.submissions = inst.submissions[1:]
	}
	return outcome, nil
}

func (r *Registry) register(inst *installation, s Signed, local time.Time) (string, error) {
	c, now, outcome := r.verify(s, DocCommercial, inst, local)
	if outcome != "" {
		return outcome, nil
	}
	r.latch(inst, now)
	at := now.Unix()
	if old := r.documents[c.ID]; old != nil {
		if string(Encode(old.claims)) == string(s.Body) {
			return OutcomeRepeat, nil
		}
		return "", ErrConflict
	}
	switch {
	case c.Start > at:
		return OutcomeNotYetValid, nil
	case end(c) <= at:
		return OutcomeExpired, nil
	}
	live := r.live(inst)
	if current, active := r.current(inst, at); active && c.IssuedAt <= current.claims.IssuedAt {
		return OutcomeSuperseded, nil
	}
	if c.Supersedes != "" {
		found := false
		for _, e := range live {
			found = found || e.claims.Agreement == c.Supersedes && e.claims.SubjectID == c.SubjectID
		}
		if !found {
			return OutcomeBadSupersedes, nil
		}
	}
	for _, e := range live {
		if e.claims.SubjectID == c.SubjectID && e.claims.Agreement != c.Supersedes && e.claims.Start < end(c) && c.Start < effectiveEnd(e, live) {
			return OutcomeDuplicateRight, nil
		}
	}
	if len(r.documents) >= maxRecords {
		return "", ErrLimit
	}
	r.accepted++
	e := &entitlement{claims: copyClaims(c), order: r.accepted}
	r.documents[c.ID] = e
	inst.accepted = append(inst.accepted, e)
	return OutcomeAccepted, nil
}
