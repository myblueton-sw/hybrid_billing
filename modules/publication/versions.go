package publication

func (s Spec) valid(l Limits) bool {
	return validID(s.Tenant) && validID(s.Dataset) && validID(s.Version) && s.Rows >= 1 && s.Rows <= l.VersionRows && validHex(s.Digest)
}

func (r *Registry) Begin(s Spec) (View, error) {
	now := r.enter()
	defer r.leave()
	if !s.valid(r.limits) {
		return View{}, ErrInvalid
	}
	key := [2]string{s.Tenant, s.Dataset}
	r.sweep(now)
	d := r.lookup(s.Tenant, s.Dataset, now)
	if d != nil {
		if v := d.ids[s.Version]; v != nil {
			if v == d.unpublished && v.view.State == StateStaging && v.view.Rows == s.Rows && v.view.Digest == s.Digest {
				out := v.view
				out.Repeat = true
				return out, nil
			}
			return View{}, ErrConflict
		}
		switch {
		case s.Version <= d.highest:
			return View{}, refuse(ReasonStale)
		case d.unpublished != nil:
			return View{}, refuse(ReasonUnpublishedOpen)
		}
	} else if r.tenantSets[s.Tenant] >= r.limits.TenantDatasets || len(r.datasets) >= r.limits.Datasets {
		return View{}, ErrLimit
	}
	if !r.reserve(s.Tenant, s.Rows) {
		return View{}, ErrLimit
	}
	if d == nil {
		d = &dataset{tenant: s.Tenant, ids: map[string]*version{}}
		r.datasets[key] = d
		r.tenantSets[s.Tenant]++
	}
	d.sequence++
	v := &version{view: View{Tenant: s.Tenant, Dataset: s.Dataset, Version: s.Version, State: StateStaging, Sequence: d.sequence, Rows: s.Rows, Digest: s.Digest,
		Begun: now, Deadline: now.Add(r.limits.StagingTTL)}}
	d.ids[s.Version] = v
	d.unpublished = v
	d.highest = s.Version
	r.enqueue(d, v)
	return v.view, nil
}

func (r *Registry) State(tenant, name, id string) (View, error) {
	now := r.enter()
	defer r.leave()
	_, v, err := r.version(tenant, name, id, now)
	if err != nil {
		return View{}, err
	}
	return v.view, nil
}

func (r *Registry) Abort(tenant, name, id string) error {
	now := r.enter()
	defer r.leave()
	d, v, err := r.version(tenant, name, id, now)
	if err != nil {
		return err
	}
	if v != d.unpublished {
		return refuse(ReasonPublished)
	}
	r.drop(d, v)
	return nil
}

func (r *Registry) Evict(tenant, name, id string) error {
	now := r.enter()
	defer r.leave()
	d, v, err := r.version(tenant, name, id, now)
	if err != nil {
		return err
	}
	switch v.view.State {
	case StateRetained:
		r.evict(d, v)
		return nil
	case StatePublished:
		return refuse(ReasonPublished)
	case StateSealed:
		return refuse(ReasonSealed)
	}
	return refuse(ReasonNotSealed)
}

func (r *Registry) evict(d *dataset, v *version) {
	r.unhold(d.tenant, v.view.Rows)
	delete(d.ids, v.view.Version)
	v.chunks, v.rows, v.totals = nil, nil, nil
	kept := d.retained[:0]
	for _, other := range d.retained {
		if other != v {
			kept = append(kept, other)
		}
	}
	d.retained = kept
}

func (r *Registry) Retire(tenant, name string) error {
	now := r.enter()
	defer r.leave()
	if !validID(tenant) || !validID(name) {
		return ErrInvalid
	}
	d := r.lookup(tenant, name, now)
	if d == nil {
		return ErrNotFound
	}
	if v := d.unpublished; v != nil {
		r.drop(d, v)
	}
	if v := d.current; v != nil {
		r.evict(d, v)
		d.current = nil
	}
	for len(d.retained) > 0 {
		r.evict(d, d.retained[0])
	}
	return nil
}
