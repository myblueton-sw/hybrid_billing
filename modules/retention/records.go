package retention

import (
	"sort"
	"time"
)

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func subset(small, big []string) bool {
	for _, s := range small {
		if !contains(big, s) {
			return false
		}
	}
	return true
}

func (x Artifact) valid(now time.Time) bool {
	return validID(x.ID) && validClass(x.Class) && ascending(x.Tenants, 1, maxTenants, validID) && !x.Anchor.IsZero() && !x.Anchor.After(now) &&
		ascending(x.Stores, 1, maxStores, validID)
}

func (r *Registry) Register(x Artifact, actor Actor) error {
	now := r.enter()
	defer r.leave()
	if err := actor.system(); err != nil {
		return err
	}
	x.Anchor = x.Anchor.UTC()
	if !x.valid(now) {
		return ErrInvalid
	}
	if a := r.artifacts[x.ID]; a != nil {
		switch {
		case a.deleted:
			return refuse(ReasonDeleted)
		case a.view.Class != x.Class || !a.view.Anchor.Equal(x.Anchor) || !sameStrings(a.view.Stores, x.Stores) || !subset(x.Tenants, a.view.Tenants):
			return ErrConflict
		}
		return nil
	}
	if len(r.artifacts) >= maxArtifacts {
		return ErrLimit
	}
	x.Tenants, x.Stores = copyStrings(x.Tenants), copyStrings(x.Stores)
	r.artifacts[x.ID] = &artifact{view: x, stores: map[string]string{}, storeAt: map[string]time.Time{}}
	if r.byClass[x.Class] == nil {
		r.byClass[x.Class] = map[string]bool{}
	}
	r.byClass[x.Class][x.ID] = true
	return nil
}

func (r *Registry) AddTenant(id, tenant string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.system(); err != nil {
		return err
	}
	a := r.artifacts[id]
	switch {
	case !validID(tenant):
		return ErrInvalid
	case a == nil:
		return ErrNotFound
	case a.deleted:
		return refuse(ReasonDeleted)
	case contains(a.view.Tenants, tenant):
		return nil
	case len(a.view.Tenants) >= maxTenants:
		return ErrLimit
	}
	list := append(copyStrings(a.view.Tenants), tenant)
	for i := len(list) - 1; i > 0 && list[i] < list[i-1]; i-- {
		list[i], list[i-1] = list[i-1], list[i]
	}
	a.view.Tenants = list
	return nil
}

func (r *Registry) busy(a *artifact) bool {
	if a.deleted || a.damaged {
		return true
	}
	if x := r.runs[a.run]; x != nil {
		out := x.outcomes[a.view.ID]
		return out == OutcomeDeleting || out == OutcomeFailed
	}
	return false
}

func (r *Registry) Refer(x Reference, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.system(); err != nil {
		return err
	}
	if !validID(x.From) || !validID(x.To) || x.From == x.To || !oneOf(x.Kind, Requires, Mentions, Guards) {
		return ErrInvalid
	}
	from, to := r.artifacts[x.From], r.artifacts[x.To]
	switch {
	case from == nil || to == nil:
		return ErrNotFound
	case x.Kind == Guards && (from.view.Class != ClassClaim || to.view.Class == ClassClaim || !shares(from.view.Tenants, to.view.Tenants)):
		return ErrInvalid
	case r.refs[[3]string{x.From, x.To, x.Kind}]:
		return nil
	case r.busy(from) || r.busy(to):
		return refuse(ReasonDeleting)
	case len(r.refs) >= maxReferences:
		return ErrLimit
	}
	r.refs[[3]string{x.From, x.To, x.Kind}] = true
	if x.Kind == Requires {
		r.requiredBy[x.To]++
		r.requires[x.From] = append(r.requires[x.From], x.To)
	}
	if x.Kind == Guards {
		r.guards[x.From] = append(r.guards[x.From], x.To)
	}
	return nil
}

func shares(a, b []string) bool {
	for _, x := range a {
		if contains(b, x) {
			return true
		}
	}
	return false
}

func (x Hold) valid() bool {
	return validID(x.ID) && validID(x.Owner) && validText(x.Reason) && ascending(x.Artifacts, 0, maxHoldArtifacts, validID) &&
		ascending(x.Tenants, 0, maxTenants, validID) && ascending(x.Classes, 0, len(Classes), validClass) &&
		len(x.Artifacts)+len(x.Tenants) > 0 && (len(x.Classes) == 0 || len(x.Tenants) > 0)
}

func (r *Registry) PlaceHold(x Hold, actor Actor) (HoldView, error) {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return HoldView{}, err
	}
	if !x.valid() {
		return HoldView{}, ErrInvalid
	}
	if h := r.holds[x.ID]; h != nil {
		return HoldView{}, ErrConflict
	}
	for _, id := range x.Artifacts {
		if r.artifacts[id] == nil {
			return HoldView{}, ErrNotFound
		}
	}
	if len(r.live) >= maxHolds {
		return HoldView{}, ErrLimit
	}
	x.Artifacts, x.Tenants, x.Classes = copyStrings(x.Artifacts), copyStrings(x.Tenants), copyStrings(x.Classes)
	h := &hold{view: x, placer: actor.ID}
	r.holds[x.ID] = h
	r.live[x.ID] = h
	return r.holdView(h), nil
}

func (r *Registry) outcome(a *artifact) string {
	switch {
	case a.deleted:
		return HoldNotPreserved
	case a.damaged:
		return HoldPartlyDeleted
	}
	if x := r.runs[a.run]; x != nil && x.outcomes[a.view.ID] == OutcomeDeleting {
		return HoldOutcomePending
	}
	return ""
}

func (r *Registry) covered(h *hold) []string {
	out := []string{}
	for _, id := range r.allIDs() {
		if a := r.artifacts[id]; h.covers(a) && r.outcome(a) != "" {
			out = append(out, id)
		}
	}
	return out
}

func (r *Registry) allIDs() []string {
	out := []string{}
	for _, class := range Classes {
		out = append(out, r.ids(class)...)
	}
	sort.Strings(out)
	return out
}

func (r *Registry) holdView(h *hold) HoldView {
	v := HoldView{Hold: h.view, Placer: h.placer, Released: h.released, Counts: map[string]int{}}
	v.Hold.Artifacts, v.Hold.Tenants, v.Hold.Classes = copyStrings(h.view.Artifacts), copyStrings(h.view.Tenants), copyStrings(h.view.Classes)
	for _, id := range r.covered(h) {
		v.Counts[r.outcome(r.artifacts[id])]++
	}
	return v
}

func (r *Registry) Hold(id string) (HoldView, error) {
	r.enter()
	defer r.leave()
	h := r.holds[id]
	if h == nil {
		return HoldView{}, ErrNotFound
	}
	return r.holdView(h), nil
}

func (r *Registry) HoldOutcomes(id, after string, limit int) ([]HoldOutcome, error) {
	r.enter()
	defer r.leave()
	h := r.holds[id]
	switch {
	case h == nil:
		return nil, ErrNotFound
	case limit < 1 || limit > maxOutcomePage:
		return nil, ErrInvalid
	}
	out := []HoldOutcome{}
	for _, id := range r.covered(h) {
		if id <= after {
			continue
		}
		if len(out) == limit {
			break
		}
		a := r.artifacts[id]
		stores := map[string]string{}
		for s, v := range a.stores {
			stores[s] = v
		}
		out = append(out, HoldOutcome{Artifact: id, Outcome: r.outcome(a), Stores: stores})
	}
	return out, nil
}
