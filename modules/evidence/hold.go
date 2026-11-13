package evidence

import "sort"

type hold struct {
	spec   HoldSpec
	print  string
	state  HoldState
	reason ReleaseReason
}

type gap struct {
	spec     GapSpec
	print    string
	state    GapState
	artifact string
}

func (r *Registry) uncovered(ws *workspace, names []string) Listed {
	var exposed []string
	for _, name := range names {
		if named := ws.bundles[name]; named != nil {
			if broken, risk := r.exposure(named); broken+risk > 0 {
				exposed = append(exposed, name)
			}
		}
	}
	return r.listed(exposed)
}

func (r *Registry) PlaceHold(spec HoldSpec) (Holding, error) {
	defer r.leave()
	if !r.enter() || !validID(spec.WorkspaceID) || !validID(spec.ID) || !validID(spec.Reason) || len(spec.Bundles) == 0 {
		return Holding{}, ErrInvalid
	}
	seen := map[string]bool{}
	for _, name := range spec.Bundles {
		if !validID(name) || seen[name] {
			return Holding{}, ErrInvalid
		}
		seen[name] = true
	}
	if len(spec.Bundles) > r.limits.HoldBundles {
		return Holding{}, ErrLimit
	}
	names := append([]string(nil), spec.Bundles...)
	sort.Strings(names)
	print := join(append([]string{spec.ID, spec.Reason}, names...)...)
	ws := r.workspaces[spec.WorkspaceID]
	if ws != nil {
		if held := ws.holds[spec.ID]; held != nil {
			if held.print != print {
				return Holding{}, ErrConflict
			}
			return Holding{State: held.state, Exposed: r.uncovered(ws, names)}, nil
		}
	}
	var absent, gone []int
	for position, name := range spec.Bundles {
		switch named := r.bundle(spec.WorkspaceID, name); {
		case named == nil:
			absent = append(absent, position)
		case named.state == Purged:
			gone = append(gone, position)
		}
	}
	if len(absent) > 0 {
		return Holding{Positions: absent}, ErrNotFound
	}
	if len(gone) > 0 {
		return Holding{Positions: gone}, ErrRejected
	}
	if len(ws.holds) >= r.limits.Holds {
		return Holding{}, ErrLimit
	}
	stored := spec
	stored.Bundles = append([]string(nil), spec.Bundles...)
	ws.holds[spec.ID] = &hold{spec: stored, print: print, state: Active}
	for _, name := range names {
		ws.bundles[name].holds[spec.ID] = true
	}
	return Holding{State: Active, Exposed: r.uncovered(ws, names)}, nil
}

func (r *Registry) ReleaseHold(workspaceID, id string, reason ReleaseReason) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !reason.known() {
		return ErrInvalid
	}
	ws := r.workspaces[workspaceID]
	if ws == nil || ws.holds[id] == nil {
		return ErrNotFound
	}
	held := ws.holds[id]
	if held.state == Released {
		if held.reason != reason {
			return ErrConflict
		}
		return nil
	}
	held.state, held.reason = Released, reason
	for _, name := range held.spec.Bundles {
		delete(ws.bundles[name].holds, id)
	}
	return nil
}

func (r *Registry) RecordGap(spec GapSpec) (GapState, error) {
	defer r.leave()
	if !r.enter() || !validID(spec.WorkspaceID) || !validID(spec.ID) || !validID(spec.Customer) || !validID(spec.Period) || spec.Role.rank() < 0 || !spec.Reason.known() || !validID(spec.Owner) || !validID(spec.Path) {
		return "", ErrInvalid
	}
	print := join(spec.ID, spec.Customer, spec.Period, string(spec.Role), string(spec.Reason), spec.Owner, spec.Path)
	if ws := r.workspaces[spec.WorkspaceID]; ws != nil {
		if held := ws.gaps[spec.ID]; held != nil {
			if held.print != print {
				return "", ErrConflict
			}
			return held.state, nil
		}
		if len(ws.gaps) >= r.limits.Gaps {
			return "", ErrLimit
		}
	}
	r.space(spec.WorkspaceID).gaps[spec.ID] = &gap{spec: spec, print: print, state: GapOpen}
	return GapOpen, nil
}

func (r *Registry) CloseGap(workspaceID, id, artifactID string) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !validID(artifactID) {
		return ErrInvalid
	}
	ws := r.workspaces[workspaceID]
	if ws == nil || ws.gaps[id] == nil {
		return ErrNotFound
	}
	held := ws.gaps[id]
	if held.state == GapClosed {
		if held.artifact != artifactID {
			return ErrConflict
		}
		return nil
	}
	with := ws.artifacts[artifactID]
	if with == nil {
		return ErrNotFound
	}
	if with.spec.Kind != held.spec.Role || !with.heldUntil(r.now, r.now) {
		return ErrRejected
	}
	held.state, held.artifact = GapClosed, artifactID
	return nil
}

func (r *Registry) open(ws *workspace, customer, period string) []string {
	var ids []string
	for id, known := range ws.gaps {
		if known.state == GapOpen && known.spec.Customer == customer && known.spec.Period == period {
			ids = append(ids, id)
		}
	}
	return ids
}

func (r *Registry) Gaps(workspaceID, customer, period string) (Listed, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(customer) || !validID(period) {
		return Listed{}, ErrInvalid
	}
	ws := r.workspaces[workspaceID]
	if ws == nil {
		return Listed{}, nil
	}
	return r.listed(r.open(ws, customer, period)), nil
}
