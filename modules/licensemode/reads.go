package licensemode

func (r *Registry) Key(id string) (KeyView, error) {
	r.enter()
	defer r.leave()
	k := r.keys[id]
	if k == nil {
		return KeyView{}, ErrNotFound
	}
	return KeyView{Key: k.view, Compromised: k.compromised, Retired: k.retired}, nil
}

func (r *Registry) History(id string) ([]ModeChange, []Submission, error) {
	r.enter()
	defer r.leave()
	inst := r.installations[id]
	if inst == nil {
		return nil, nil, ErrNotFound
	}
	return append([]ModeChange{}, inst.modes...), append([]Submission{}, inst.submissions...), nil
}
