package migration

import "time"

func ascending(values []string, limit int) bool {
	if len(values) == 0 || len(values) > limit {
		return false
	}
	for index, value := range values {
		if !validID(value) || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func clean(c Change) (Change, string, bool) {
	out := Change{Kind: c.Kind}
	switch c.Kind {
	case ChangeSource:
		out.Source = c.Source
		out.Source.Workspaces = append([]string(nil), c.Source.Workspaces...)
		x := c.Source
		return out, registerScope, validID(x.ID) && validID(x.Tenant) && validID(x.Importer) && ascending(x.Workspaces, maxWorkspaces)
	case ChangeImporter:
		out.Source = Source{ID: c.Source.ID, Importer: c.Source.Importer}
		return out, c.Source.ID, validID(c.Source.ID) && validID(c.Source.Importer)
	case ChangeMapping:
		out.Mapping = c.Mapping
		m := c.Mapping
		return out, m.Source, validID(m.Source) && validID(m.SourceSeries) && validID(m.Workspace) && validID(m.Series)
	case ChangeBatch:
		out.Batch = c.Batch
		b := c.Batch
		return out, b.Source, validID(b.Source) && validID(b.ID) && b.Count >= 1 && b.Count <= maxItems && validHex(b.Digest)
	case ChangeAccept:
		out.Accept = c.Accept
		a := c.Accept
		return out, a.Source, validID(a.Source) && validID(a.Batch) && validHex(a.Effect)
	}
	return out, registerScope, false
}

func encodeChange(c Change) []byte {
	var e encoder
	for _, value := range []string{c.Kind, c.Source.ID, c.Source.Tenant, c.Source.Importer, c.Mapping.Source, c.Mapping.SourceSeries, c.Mapping.Workspace, c.Mapping.Series,
		c.Batch.Source, c.Batch.ID, c.Batch.Digest, c.Accept.Source, c.Accept.Batch, c.Accept.Effect} {
		e.text(value)
	}
	e.count(len(c.Source.Workspaces))
	for _, w := range c.Source.Workspaces {
		e.text(w)
	}
	e.count(c.Batch.Count)
	return e.out.Bytes()
}

func copyChange(c Change) Change {
	c.Source.Workspaces = append([]string(nil), c.Source.Workspaces...)
	return c
}

func copyProposal(p ProposalView) ProposalView {
	p.Change = copyChange(p.Change)
	return p
}

func (r *Registry) check(c Change, _ time.Time) error {
	switch c.Kind {
	case ChangeSource:
		switch {
		case r.sources[c.Source.ID] != nil:
			return ErrConflict
		case r.identities[c.Source.Importer] != "":
			return refuse(ReasonIdentityTaken)
		case len(r.sources) >= maxSources:
			return ErrLimit
		}
		for _, w := range c.Source.Workspaces {
			if owner := r.claims[w]; owner != "" && owner != c.Source.Tenant {
				return refuse(ReasonWorkspaceClaimed)
			}
		}
		return nil
	case ChangeImporter:
		if r.identities[c.Source.Importer] != "" {
			return refuse(ReasonIdentityTaken)
		}
		return nil
	}
	s := r.sources[scopeOf(c)]
	switch c.Kind {
	case ChangeMapping:
		switch {
		case !listed(s.view.Workspaces, c.Mapping.Workspace):
			return refuse(ReasonWorkspaceNotAllowed)
		case s.mappings[[2]string{c.Mapping.SourceSeries, c.Mapping.Workspace}] != "":
			return ErrConflict
		}
	case ChangeBatch:
		switch {
		case s.batches[c.Batch.ID] != nil:
			return ErrConflict
		case s.open != "":
			return refuse(ReasonBatchOpen)
		case len(s.order) >= maxBatches:
			return ErrLimit
		}
	case ChangeAccept:
		b := s.batches[c.Accept.Batch]
		if b == nil {
			return ErrNotFound
		}
		if b.state != BatchVerified {
			return refuse(ReasonNotVerified)
		}
		plan, err := r.classify(s, b)
		switch {
		case err != nil:
			return err
		case plan.effect != c.Accept.Effect:
			return refuse(ReasonEffectChanged)
		}
		return r.room(s, plan)
	}
	return nil
}

func scopeOf(c Change) string {
	switch c.Kind {
	case ChangeMapping:
		return c.Mapping.Source
	case ChangeBatch:
		return c.Batch.Source
	}
	return c.Accept.Source
}

func listed(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func (r *Registry) apply(c Change, _ time.Time) {
	switch c.Kind {
	case ChangeSource:
		x := copyChange(c).Source
		r.sources[x.ID] = &source{view: x, mappings: map[[2]string]string{}, batches: map[string]*batch{}, history: map[string]*entry{}, seen: map[string]bool{}}
		r.identities[x.Importer] = x.ID
		for _, w := range x.Workspaces {
			if r.claims[w] == "" {
				r.claims[w] = x.Tenant
			}
		}
		r.tenant(x.Tenant)
	case ChangeImporter:
		s := r.sources[c.Source.ID]
		s.view.Importer = c.Source.Importer
		r.identities[c.Source.Importer] = c.Source.ID
	case ChangeMapping:
		r.sources[c.Mapping.Source].mappings[[2]string{c.Mapping.SourceSeries, c.Mapping.Workspace}] = c.Mapping.Series
	case ChangeBatch:
		s := r.sources[c.Batch.Source]
		s.batches[c.Batch.ID] = &batch{spec: c.Batch, state: BatchOpen, items: map[string]Item{}, codes: map[string]string{}}
		s.order = append(s.order, c.Batch.ID)
		s.open = c.Batch.ID
	case ChangeAccept:
		s := r.sources[c.Accept.Source]
		b := s.batches[c.Accept.Batch]
		plan, _ := r.classify(s, b)
		r.commit(s, b, plan)
	}
}
