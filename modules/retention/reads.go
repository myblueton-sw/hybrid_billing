package retention

func (r *Registry) Access(id string) (string, error) {
	r.enter()
	defer r.leave()
	a := r.artifacts[id]
	switch {
	case a == nil:
		return "", ErrNotFound
	case a.deleted:
		return AccessDeleted, nil
	case a.damaged || a.run != "":
		return AccessBlocked, nil
	}
	return AccessOK, nil
}

func (r *Registry) Artifact(id string) (Artifact, error) {
	r.enter()
	defer r.leave()
	a := r.artifacts[id]
	if a == nil {
		return Artifact{}, ErrNotFound
	}
	v := a.view
	v.Tenants, v.Stores = copyStrings(v.Tenants), copyStrings(v.Stores)
	return v, nil
}

func (r *Registry) Policy(class string) (PolicyView, error) {
	r.enter()
	defer r.leave()
	p := r.current(class)
	if p == nil {
		return PolicyView{}, ErrNotFound
	}
	return *p, nil
}

func (r *Registry) Impact(x Policy) (ImpactView, error) {
	now := r.enter()
	defer r.leave()
	if _, err := clean(Change{Kind: ChangePolicy, Policy: x}); err != nil {
		return ImpactView{}, err
	}
	return r.impact(x, now), nil
}

func (r *Registry) Preview(class, after string, limit int) (PreviewView, error) {
	now := r.enter()
	defer r.leave()
	if !validClass(class) || limit < 1 || limit > maxPage {
		return PreviewView{}, ErrInvalid
	}
	v := PreviewView{Eligible: []string{}}
	for _, id := range r.ids(class) {
		a := r.artifacts[id]
		if a.deleted || a.run != "" || !r.due(a, now) {
			continue
		}
		switch r.exclusion(a, now) {
		case OutcomeHeld:
			v.Held++
			continue
		case OutcomeReferenced:
			v.Referenced++
			continue
		}
		if id <= after {
			continue
		}
		if len(v.Eligible) == limit {
			v.More = true
			continue
		}
		v.Eligible = append(v.Eligible, id)
	}
	return v, nil
}

func (r *Registry) Tombstones(after uint64, limit int) ([]Tombstone, error) {
	r.enter()
	defer r.leave()
	if limit < 1 || limit > maxTombstonePage {
		return nil, ErrInvalid
	}
	if after >= uint64(len(r.tombstones)) {
		return []Tombstone{}, nil
	}
	end := min(after+uint64(limit), uint64(len(r.tombstones)))
	return append([]Tombstone{}, r.tombstones[after:end]...), nil
}

func (r *Registry) Watermark() uint64 {
	r.enter()
	defer r.leave()
	return uint64(len(r.tombstones))
}

func (r *Registry) Restored(applied uint64) bool {
	r.enter()
	defer r.leave()
	return applied == uint64(len(r.tombstones))
}
