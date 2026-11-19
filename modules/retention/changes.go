package retention

import (
	"sort"
	"strconv"
	"time"
)

func validClass(c string) bool { return contains(Classes, c) }

func clean(c Change) (Change, error) {
	switch c.Kind {
	case ChangePolicy:
		x := c.Policy
		if !validClass(x.Class) || x.Period < 1 || x.Period > maxPeriodDays || x.Grace < 0 || x.Grace > maxGraceDays {
			return Change{}, ErrInvalid
		}
		return Change{Kind: c.Kind, Policy: x}, nil
	case ChangePlan:
		x := c.Plan
		if !validClass(x.Class) || !ascending(x.Artifacts, 1, maxRunArtifacts, validID) || x.Chunk < 1 || x.Chunk > maxChunk ||
			x.Lease < minLease || x.Lease > maxLease || !validText(x.Reason) {
			return Change{}, ErrInvalid
		}
		return copyChange(Change{Kind: c.Kind, Plan: x}), nil
	case ChangeRelease:
		if !validID(c.Release.Hold) || !validText(c.Release.Reason) {
			return Change{}, ErrInvalid
		}
		return Change{Kind: c.Kind, Release: c.Release}, nil
	}
	return Change{}, ErrInvalid
}

func (r *Registry) current(class string) *PolicyView {
	list := r.policies[class]
	if len(list) == 0 {
		return nil
	}
	return &list[len(list)-1]
}

func dueUnder(a *artifact, days int, t time.Time) bool {
	return !a.view.Anchor.Add(time.Duration(days) * day).After(t)
}

func (r *Registry) due(a *artifact, t time.Time) bool {
	p := r.current(a.view.Class)
	return p != nil && dueUnder(a, p.Policy.Period+p.Policy.Grace, t)
}

func (r *Registry) shortening(c Change) bool {
	p := r.current(c.Policy.Class)
	return c.Kind == ChangePolicy && p != nil && c.Policy.Period+c.Policy.Grace < p.Policy.Period+p.Policy.Grace
}

func (r *Registry) ids(class string) []string {
	out := make([]string, 0, len(r.byClass[class]))
	for id := range r.byClass[class] {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (r *Registry) impact(x Policy, at time.Time) ImpactView {
	v := ImpactView{Tenants: map[string]int{}}
	cur := r.current(x.Class)
	set := []string{}
	for _, id := range r.ids(x.Class) {
		a := r.artifacts[id]
		if a.deleted || !dueUnder(a, x.Period+x.Grace, at) || cur != nil && dueUnder(a, cur.Policy.Period+cur.Policy.Grace, at) {
			continue
		}
		set = append(set, id)
		for _, tenant := range a.view.Tenants {
			v.Tenants[tenant]++
		}
		switch {
		case r.held(a):
			v.Held++
		case r.requiredBy[id] > 0:
			v.Referenced++
		}
	}
	v.Impact = Impact{AsOf: at, Count: len(set), Digest: digestIDs(set)}
	return v
}

func (r *Registry) held(a *artifact) bool {
	for _, h := range r.live {
		if h.covers(a) {
			return true
		}
	}
	return false
}

func (h *hold) covers(a *artifact) bool {
	if contains(h.view.Artifacts, a.view.ID) {
		return true
	}
	for _, tenant := range a.view.Tenants {
		if h.view.Covers(tenant, a.view.Class) {
			return true
		}
	}
	return false
}

func (r *Registry) exclusion(a *artifact, t time.Time) string {
	switch {
	case r.held(a):
		return OutcomeHeld
	case r.requiredBy[a.view.ID] > 0:
		return OutcomeReferenced
	case !r.due(a, t):
		return OutcomeNotDue
	}
	return ""
}

func (r *Registry) check(c Change, proposer, approver string, reviewed Impact, now time.Time) error {
	switch c.Kind {
	case ChangePolicy:
		if approver == "" || !r.shortening(c) {
			return nil
		}
		age := now.Sub(reviewed.AsOf)
		if reviewed.AsOf.IsZero() || age < 0 || age > impactWindow {
			return refuse(ReasonImpactRequired)
		}
		if got := r.impact(c.Policy, reviewed.AsOf).Impact; got.Count != reviewed.Count || got.Digest != reviewed.Digest {
			return refuse(ReasonImpactChanged)
		}
	case ChangePlan:
		for _, id := range c.Plan.Artifacts {
			a := r.artifacts[id]
			switch {
			case a == nil:
				return ErrNotFound
			case a.view.Class != c.Plan.Class:
				return ErrInvalid
			case a.deleted:
				return refuse(ReasonAlreadyDeleted)
			case a.run != "":
				return refuse(ReasonInPlan)
			}
		}
		if r.unfinished >= maxUnfinished {
			return ErrLimit
		}
	case ChangeRelease:
		h := r.holds[c.Release.Hold]
		switch {
		case h == nil:
			return ErrNotFound
		case h.released:
			return refuse(ReasonReleased)
		case h.placer == proposer || h.placer == approver:
			return refuse(ReasonPlacer)
		}
	}
	return nil
}

func (r *Registry) apply(c Change, by people, now time.Time) {
	switch c.Kind {
	case ChangePolicy:
		list := r.policies[c.Policy.Class]
		r.policies[c.Policy.Class] = append(list, PolicyView{Policy: c.Policy, Number: uint64(len(list) + 1), Proposer: by.proposer, Approver: by.approver, At: now})
	case ChangePlan:
		r.runSeq++
		x := &run{id: "run/" + strconv.FormatUint(r.runSeq, 10), plan: copyChange(c).Plan, outcomes: map[string]string{}, by: by}
		if p := r.current(c.Plan.Class); p != nil {
			x.revision = p.Number
		}
		x.chunks = make([]chunk, (len(x.plan.Artifacts)+x.plan.Chunk-1)/x.plan.Chunk)
		for _, id := range x.plan.Artifacts {
			a := r.artifacts[id]
			if out := r.exclusion(a, now); out != "" {
				x.outcomes[id] = out
				continue
			}
			x.outcomes[id] = OutcomePending
			a.run = x.id
			a.clearFailed()
		}
		r.runs[x.id] = x
		r.unfinished++
	case ChangeRelease:
		r.holds[c.Release.Hold].released = true
		delete(r.live, c.Release.Hold)
	}
}
