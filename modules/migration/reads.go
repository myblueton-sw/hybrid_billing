package migration

import "sort"

func (r *Registry) owned(tenant, id string) (*source, error) {
	if !validID(tenant) || !validID(id) {
		return nil, ErrInvalid
	}
	s := r.sources[id]
	if s == nil || s.view.Tenant != tenant {
		return nil, ErrNotFound
	}
	return s, nil
}

func (r *Registry) view(s *source, b *batch) BatchView {
	v := BatchView{Source: s.view.ID, ID: b.spec.ID, State: b.state, Count: b.spec.Count, Digest: b.spec.Digest, Computed: b.computed, Classes: map[string]int{}, Series: []SeriesChange{},
		By: b.rejectedBy, Reason: b.reason}
	switch b.state {
	case BatchVerified:
		if p, err := r.classify(s, b); err == nil {
			for _, c := range p.classes {
				v.Classes[c.Class]++
			}
			v.Effect, v.Series, v.Printed, v.Keys = p.effect, seriesChanges(p.series), len(p.printed), len(p.keys)
		}
	case BatchAccepted:
		for class, n := range b.summary {
			v.Classes[class] = n
		}
		v.Effect = b.effect
	}
	return v
}

func (r *Registry) Batch(tenant, id, batchID string) (BatchView, error) {
	r.enter()
	defer r.leave()
	s, err := r.owned(tenant, id)
	if err != nil {
		return BatchView{}, err
	}
	b := s.batches[batchID]
	if b == nil {
		return BatchView{}, ErrNotFound
	}
	return r.view(s, b), nil
}

func (r *Registry) Document(tenant, id, sourceID string) (Document, error) {
	r.enter()
	defer r.leave()
	s, err := r.owned(tenant, id)
	if err != nil {
		return Document{}, err
	}
	e := s.history[sourceID]
	if e == nil || e.document == nil {
		return Document{}, ErrNotFound
	}
	return *e.document, nil
}

func (r *Registry) Key(tenant, id, sourceID string) (KeyView, error) {
	r.enter()
	defer r.leave()
	s, err := r.owned(tenant, id)
	if err != nil {
		return KeyView{}, err
	}
	e := s.history[sourceID]
	if e == nil || e.key == nil {
		return KeyView{}, ErrNotFound
	}
	k := *e.key
	k.DER = append([]byte(nil), k.DER...)
	return k, nil
}

func (r *Registry) Quarantine(tenant, id string) ([]Classified, error) {
	r.enter()
	defer r.leave()
	s, err := r.owned(tenant, id)
	if err != nil {
		return nil, err
	}
	return append([]Classified{}, s.quarantine...), nil
}

func (r *Registry) fences(tenant, workspace string) (*tenant, error) {
	if !validID(tenant) || !validID(workspace) {
		return nil, ErrInvalid
	}
	t := r.tenants[tenant]
	if t == nil {
		return nil, ErrNotFound
	}
	return t, nil
}

func (r *Registry) HighWater(tenant, workspace, series string) (int64, error) {
	r.enter()
	defer r.leave()
	if !validID(series) {
		return 0, ErrInvalid
	}
	t, err := r.fences(tenant, workspace)
	if err != nil {
		return 0, err
	}
	return t.high[[2]string{workspace, series}], nil
}

func (r *Registry) PrintedTaken(tenant, workspace, display string) (bool, error) {
	r.enter()
	defer r.leave()
	if !printable(display, 64) {
		return false, ErrInvalid
	}
	t, err := r.fences(tenant, workspace)
	if err != nil {
		return false, err
	}
	return t.printed[[2]string{workspace, fenceKey("dsp-", tenant, display)}], nil
}

func (r *Registry) DeliveryFenced(tenant, workspace, id, sourceID string) (bool, error) {
	r.enter()
	defer r.leave()
	if !validID(id) || !validID(sourceID) {
		return false, ErrInvalid
	}
	t, err := r.fences(tenant, workspace)
	if err != nil {
		return false, err
	}
	return t.delivery[[2]string{workspace, fenceKey("src-", tenant, id, sourceID)}], nil
}

func (r *Registry) ExternalFenced(tenant, workspace, key string) (bool, error) {
	r.enter()
	defer r.leave()
	if !validID(key) {
		return false, ErrInvalid
	}
	t, err := r.fences(tenant, workspace)
	if err != nil {
		return false, err
	}
	return t.delivery[[2]string{workspace, fenceKey("ext-", tenant, key)}], nil
}

func (r *Registry) Fences(tenant, workspace string) (Export, error) {
	r.enter()
	defer r.leave()
	t, err := r.fences(tenant, workspace)
	if err != nil {
		return Export{}, err
	}
	out := Export{HighWater: []SeriesChange{}, Keys: []string{}, Printed: []string{}}
	for k, n := range t.high {
		if k[0] == workspace {
			out.HighWater = append(out.HighWater, SeriesChange{Workspace: workspace, Series: k[1], Number: n})
		}
	}
	for k := range t.delivery {
		if k[0] == workspace {
			out.Keys = append(out.Keys, k[1])
		}
	}
	for k := range t.printed {
		if k[0] == workspace {
			out.Printed = append(out.Printed, k[1])
		}
	}
	sort.Slice(out.HighWater, func(a, b int) bool { return out.HighWater[a].Series < out.HighWater[b].Series })
	sort.Strings(out.Keys)
	sort.Strings(out.Printed)
	return out, nil
}
