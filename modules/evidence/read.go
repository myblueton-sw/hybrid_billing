package evidence

import "sort"

func (r *Registry) live(workspaceID, id string) (*workspace, *bundle, error) {
	if !validID(workspaceID) || !validID(id) {
		return nil, nil, ErrInvalid
	}
	held := r.bundle(workspaceID, id)
	if held == nil {
		return nil, nil, ErrNotFound
	}
	if held.state == Purged {
		return nil, nil, ErrRejected
	}
	return r.workspaces[workspaceID], held, nil
}

func (r *Registry) Inspect(workspaceID, id string) (Inspection, error) {
	defer r.leave()
	if !r.enter() {
		return Inspection{}, ErrInvalid
	}
	ws, held, err := r.live(workspaceID, id)
	if err != nil {
		return Inspection{}, err
	}
	out := Inspection{Verdict: Held, Rerun: Operational, Protected: held.protected(r.now), BoundTotal: len(held.gaps)}
	for _, item := range held.entries {
		if (item.role == Engine || item.role == Module) && r.barred(item.artifact) {
			out.Rerun = IsolatedOnly
		}
		switch {
		case !item.artifact.heldUntil(r.now, r.now):
			out.Verdict = Broken
			out.BrokenTotal++
			if len(out.Broken) < r.limits.Listed {
				cause, reason := item.artifact.cause()
				out.Broken = append(out.Broken, BrokenItem{Role: item.role, Artifact: item.artifact.spec.ID, Cause: cause, Reason: reason})
			}
		case r.risky(held, item.artifact):
			out.AtRiskTotal++
			if len(out.AtRisk) < r.limits.Listed {
				out.AtRisk = append(out.AtRisk, Item{Role: item.role, Artifact: item.artifact.spec.ID})
			}
		}
	}
	for _, name := range held.gaps {
		if len(out.Bound) == r.limits.Listed {
			break
		}
		out.Bound = append(out.Bound, GapView{ID: name, State: ws.gaps[name].state, Artifact: ws.gaps[name].artifact})
	}
	out.Open = r.listed(r.open(ws, held.spec.Customer, held.spec.Period))
	return out, nil
}

func (r *Registry) Exposed(workspaceID string) (Listed, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) {
		return Listed{}, ErrInvalid
	}
	ws := r.workspaces[workspaceID]
	if ws == nil {
		return Listed{}, nil
	}
	names := make([]string, 0, len(ws.bundles))
	for name := range ws.bundles {
		names = append(names, name)
	}
	return r.uncovered(ws, names), nil
}

func (r *Registry) Download(workspaceID, id string, role Kind, artifactID string) (Download, error) {
	defer r.leave()
	if !r.enter() || role.rank() < 0 || !validID(artifactID) {
		return Download{}, ErrInvalid
	}
	_, held, err := r.live(workspaceID, id)
	if err != nil {
		return Download{}, err
	}
	for _, item := range held.entries {
		if item.role != role || item.artifact.spec.ID != artifactID {
			continue
		}
		if !item.artifact.heldUntil(r.now, r.now) {
			return Download{}, ErrRejected
		}
		out := Download{ID: artifactID, Version: item.artifact.spec.Version, Digest: item.artifact.spec.Digest, Size: item.artifact.spec.Size}
		for _, stored := range item.artifact.instances {
			if !stored.lost && (stored.custody.Kind == Product || !stored.custody.VerifiedUntil.IsZero() && !stored.custody.VerifiedUntil.Before(r.now)) {
				out.Instances = append(out.Instances, stored.custody)
			}
		}
		sort.Slice(out.Instances, func(i, j int) bool {
			if out.Instances[i].Kind != out.Instances[j].Kind {
				return out.Instances[i].Kind < out.Instances[j].Kind
			}
			return out.Instances[i].Location < out.Instances[j].Location
		})
		return out, nil
	}
	return Download{}, ErrNotFound
}

func (r *Registry) Items(workspaceID, id string, after, limit int) ([]Item, error) {
	defer r.leave()
	if !r.enter() || after < 0 || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	_, held, err := r.live(workspaceID, id)
	if err != nil {
		return nil, err
	}
	if after > len(held.spec.Items) {
		after = len(held.spec.Items)
	}
	end := len(held.spec.Items)
	if limit < end-after {
		end = after + limit
	}
	return append([]Item(nil), held.spec.Items[after:end]...), nil
}

func (r *Registry) Bundle(workspaceID, id string) (BundleView, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return BundleView{}, ErrInvalid
	}
	held := r.bundle(workspaceID, id)
	if held == nil {
		return BundleView{}, ErrNotFound
	}
	view := BundleView{Spec: held.spec, State: held.state, PurgeReason: held.reason, Retained: held.retained, Counts: map[Kind]int{}, Digest: held.digest,
		Gaps: append([]string(nil), held.gaps...), Holds: len(held.holds), Confirmed: held.confirmed, Changed: held.changed}
	view.Spec.Items = nil
	for role, count := range held.counts {
		view.Counts[role] = count
	}
	return view, nil
}
