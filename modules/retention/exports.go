package retention

import (
	"sort"
	"time"
)

type Standing struct {
	Access     string
	Due        bool
	DueAt      time.Time
	Held       bool
	Referenced bool
	Guarding   bool
	Damaged    bool
	Run        string
	Stores     map[string]string
}

func ValidText(text string) bool { return validText(text) }

func (r *Registry) Check(c Change, proposer, approver string, reviewed Impact) error {
	now := r.enter()
	defer r.leave()
	c, err := clean(c)
	if err != nil {
		return err
	}
	return r.check(c, proposer, approver, reviewed, now)
}

func (r *Registry) Standing(id string) (Standing, error) {
	now := r.enter()
	defer r.leave()
	a := r.artifacts[id]
	if a == nil {
		return Standing{}, ErrNotFound
	}
	v := Standing{Access: AccessOK, Due: r.due(a, now), Held: r.held(a), Referenced: r.requiredBy[id] > 0, Guarding: r.guarding(a), Damaged: a.damaged, Run: a.run, Stores: map[string]string{}}
	switch {
	case a.deleted:
		v.Access = AccessDeleted
	case a.damaged || a.run != "":
		v.Access = AccessBlocked
	}
	if at, settled := r.anchor(a); settled {
		if p := r.current(a.view.Class); p != nil {
			v.DueAt = at.Add(time.Duration(p.Policy.Period+p.Policy.Grace) * day)
		}
	}
	for store, result := range a.stores {
		v.Stores[store] = result
	}
	return v, nil
}

func (h Hold) Covers(tenant, class string) bool {
	return contains(h.Tenants, tenant) && (len(h.Classes) == 0 || contains(h.Classes, class))
}

func (r *Registry) HoldsCovering(class string) []HoldView {
	r.enter()
	defer r.leave()
	out := []HoldView{}
	for _, h := range r.holds {
		if len(h.view.Tenants) > 0 && (len(h.view.Classes) == 0 || contains(h.view.Classes, class)) {
			snapshot := h.view
			snapshot.Artifacts, snapshot.Tenants, snapshot.Classes = copyStrings(h.view.Artifacts), copyStrings(h.view.Tenants), copyStrings(h.view.Classes)
			out = append(out, HoldView{Hold: snapshot, Placer: h.placer, Released: h.released})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Hold.ID < out[b].Hold.ID })
	return out
}
