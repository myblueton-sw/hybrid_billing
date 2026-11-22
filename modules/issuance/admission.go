package issuance

import "time"

func (r *Registry) internalAdmission(workspace string, s *space, d *draft, k *kind, at time.Time) (*series, admission, error) {
	v, err := r.seriesFor(s, k.view.Kind.ID)
	if err != nil {
		return nil, admission{}, err
	}
	a, err := r.admit(workspace, s, d, k, at)
	if err != nil {
		return nil, admission{}, err
	}
	if len(s.documents) >= r.limits.Documents {
		return nil, admission{}, ErrLimit
	}
	if d.reserved == nil {
		if err := r.numberCapacity(s, v); err != nil {
			return nil, admission{}, err
		}
	}
	return v, a, nil
}

func (r *Registry) externalAdmission(workspace string, s *space, d *draft, k *kind, x External, now time.Time) (admission, string, error) {
	at := x.IssuedAt.UTC().Round(0)
	switch {
	case k.view.Kind.Class != ClassERP:
		return admission{}, "", refuse(ReasonNotERP)
	case at.After(now):
		return admission{}, "", refuse(ReasonFutureInstant)
	case d.current.Approver != "" && at.Before(d.current.At):
		return admission{}, "", refuse(ReasonBeforeApproval)
	}
	a, err := r.admit(workspace, s, d, k, at)
	if err != nil {
		return admission{}, "", err
	}
	key := k.view.Kind.ID + "\x00" + x.Number
	if s.external[key] {
		return admission{}, "", refuse(ReasonExternalInUse)
	}
	if len(s.documents) >= r.limits.Documents {
		return admission{}, "", ErrLimit
	}
	return a, key, nil
}

func (r *Registry) checkDraft(workspace, id, digest string, actor Actor) (*space, *draft, *kind, error) {
	if !validDigest(digest) {
		return nil, nil, nil, ErrInvalid
	}
	s, d, k, err := r.prepare(workspace, id, actor)
	if err != nil {
		return nil, nil, nil, err
	}
	if d.document != nil {
		return nil, nil, nil, refuse(ReasonAlreadyIssued)
	}
	if d.current.Digest != digest {
		return nil, nil, nil, refuse(ReasonDigestChanged)
	}
	return s, d, k, nil
}

func (r *Registry) CheckIssue(workspace, id, digest string, actor Actor) error {
	now := r.enter()
	defer r.leave()
	s, d, k, err := r.checkDraft(workspace, id, digest, actor)
	if err != nil {
		return err
	}
	_, _, err = r.internalAdmission(workspace, s, d, k, now)
	return err
}

func (r *Registry) CheckExternal(workspace, id, digest string, x External, actor Actor) error {
	now := r.enter()
	defer r.leave()
	if !validID(x.Number) || x.IssuedAt.IsZero() {
		return ErrInvalid
	}
	s, d, k, err := r.checkDraft(workspace, id, digest, actor)
	if err != nil {
		return err
	}
	_, _, err = r.externalAdmission(workspace, s, d, k, x, now)
	return err
}
