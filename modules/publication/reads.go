package publication

import "sort"

func (r *Registry) Publish(tenant, name, id string) (View, error) {
	now := r.enter()
	defer r.leave()
	d, v, err := r.version(tenant, name, id, now)
	if err != nil {
		return View{}, err
	}
	switch v.view.State {
	case StatePublished:
		return v.view, nil
	case StateRetained:
		return View{}, refuse(ReasonRetained)
	case StateStaging:
		return View{}, refuse(ReasonNotSealed)
	}
	if old := d.current; old != nil {
		old.view.State = StateRetained
		d.retained = append(d.retained, old)
		if len(d.retained) > r.limits.Retained {
			r.evict(d, d.retained[0])
		}
	}
	v.view.State, v.view.Published = StatePublished, now
	d.current, d.unpublished = v, nil
	r.open--
	return v.view, nil
}

func (r *Registry) Current(tenant, name string) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(tenant) || !validID(name) {
		return View{}, ErrInvalid
	}
	d := r.lookup(tenant, name, now)
	if d == nil || d.current == nil {
		return View{}, ErrNotFound
	}
	return d.current.view, nil
}

func (r *Registry) readable(tenant, name, id string, limit int) (*version, error) {
	now := r.enter()
	if limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	_, v, err := r.version(tenant, name, id, now)
	if err != nil {
		return nil, err
	}
	if v.view.State == StateStaging {
		return nil, refuse(ReasonNotSealed)
	}
	return v, nil
}

func (r *Registry) Rows(tenant, name, id, after string, limit int) (RowPage, error) {
	defer r.leave()
	v, err := r.readable(tenant, name, id, limit)
	if err != nil {
		return RowPage{}, err
	}
	start := sort.Search(len(v.rows), func(index int) bool { return v.rows[index].Key > after })
	end := min(start+limit, len(v.rows))
	page := RowPage{Rows: append([]Row{}, v.rows[start:end]...)}
	if end < len(v.rows) {
		page.Next = v.rows[end-1].Key
	}
	return page, nil
}

func (r *Registry) Totals(tenant, name, id string, after *TotalKey, limit int) (TotalPage, error) {
	defer r.leave()
	v, err := r.readable(tenant, name, id, limit)
	if err != nil {
		return TotalPage{}, err
	}
	start := 0
	if after != nil {
		start = sort.Search(len(v.totals), func(index int) bool { return keyLess(*after, v.totals[index].Key) })
	}
	end := min(start+limit, len(v.totals))
	page := TotalPage{Totals: append([]Total{}, v.totals[start:end]...)}
	if end < len(v.totals) {
		next := v.totals[end-1].Key
		page.Next = &next
	}
	return page, nil
}
