package retention

import "time"

type Standing struct {
	Access     string
	Due        bool
	DueAt      time.Time
	Held       bool
	Referenced bool
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
	v := Standing{Access: AccessOK, Due: r.due(a, now), Held: r.held(a), Referenced: r.requiredBy[id] > 0, Damaged: a.damaged, Run: a.run, Stores: map[string]string{}}
	switch {
	case a.deleted:
		v.Access = AccessDeleted
	case a.damaged || a.run != "":
		v.Access = AccessBlocked
	}
	if p := r.current(a.view.Class); p != nil {
		v.DueAt = a.view.Anchor.Add(time.Duration(p.Policy.Period+p.Policy.Grace) * day)
	}
	for store, result := range a.stores {
		v.Stores[store] = result
	}
	return v, nil
}
