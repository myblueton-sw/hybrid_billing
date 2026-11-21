package approvals

func (r *Registry) PrepareSubject(s Subject) (Subject, error) {
	prepared, _, ok := r.subjectForm(s)
	if !ok {
		return Subject{}, ErrInvalid
	}
	return prepared, nil
}

func (r *Registry) PrepareAuthority(a Authority) (Authority, error) {
	if !r.authorityForm(a) {
		return Authority{}, ErrInvalid
	}
	_, printed, ok := r.number(a.Limit, false)
	if !ok {
		return Authority{}, ErrInvalid
	}
	a.Limit = printed
	return a, nil
}
