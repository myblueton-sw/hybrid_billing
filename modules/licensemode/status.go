package licensemode

import "time"

func (r *Registry) Status(id string) (StatusView, error) {
	local := r.enter()
	defer r.leave()
	if !validID(id) {
		return StatusView{}, ErrInvalid
	}
	inst := r.installations[id]
	if inst == nil {
		return StatusView{}, ErrNotFound
	}
	v := StatusView{Installation: id, Mode: inst.mode, Actions: []string{ActionRegister, ActionContact}}
	if n := len(inst.submissions); n > 0 {
		v.LastOutcome = inst.submissions[n-1].Outcome
	}
	switch inst.mode {
	case "":
		v.State = StateUnselected
		return v, nil
	case ModeNoncommercial:
		v.State = StateNoncommercial
		return v, nil
	}
	now, outcome := r.authoritative(local)
	switch {
	case outcome != "":
		v.State, v.Reason = StateUnverified, outcome
	case inst.mode == ModeEvaluation:
		r.latch(inst, now)
		r.evaluation(inst, &v, now)
	default:
		r.latch(inst, now)
		r.commercial(inst, &v, now)
	}
	if v.State == StateUnverified {
		v.Warning, v.LastVerified, v.LastVerifiedAt = true, inst.last, inst.lastAt
		return v, nil
	}
	inst.last, inst.lastAt = v.State, now
	v.Warning = v.State == StateEvaluationExpired || v.State == StateCommercialExpired
	return v, nil
}

func (r *Registry) evaluation(inst *installation, v *StatusView, now time.Time) {
	a := inst.activation
	switch {
	case a == nil:
		v.State = StateEvaluationPending
		return
	case r.compromised(a.view.KeyID):
		v.State, v.Reason = StateUnverified, ReasonKeyCompromised
		return
	}
	view := copyActivation(a.view)
	v.Activation = &view
	switch {
	case a.expired:
		v.State = StateEvaluationExpired
	default:
		v.State, v.Remaining = StateEvaluationActive, a.view.End.Sub(now)
	}
}

func (r *Registry) commercial(inst *installation, v *StatusView, now time.Time) {
	e, active := r.current(inst, now.Unix())
	switch {
	case e == nil && len(inst.accepted) > 0:
		v.State, v.Reason = StateUnverified, ReasonKeyCompromised
		return
	case e == nil:
		v.State = StateCommercialPending
		return
	}
	claims := copyClaims(e.claims)
	v.Entitlement = &claims
	if active {
		v.State = StateCommercialActive
		return
	}
	at := now.Unix()
	for _, x := range inst.accepted {
		if r.compromised(x.claims.KeyID) && !x.ended && x.claims.Start <= at && at < end(x.claims) {
			v.State, v.Reason, v.Entitlement = StateUnverified, ReasonKeyCompromised, nil
			return
		}
	}
	v.State = StateCommercialExpired
}
