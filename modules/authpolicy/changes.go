package authpolicy

import (
	"strconv"
	"strings"
	"time"
)

func validPlanID(id string) bool {
	cut := strings.LastIndexByte(id, '/')
	if cut < 0 || !validID(id[:cut]) || strings.HasPrefix(id[cut+1:], "0") {
		return false
	}
	n, err := strconv.ParseInt(id[cut+1:], 10, 64)
	return err == nil && n > 0 && !strings.HasPrefix(id[cut+1:], "+")
}

func clean(c Change) (Change, error) {
	switch c.Kind {
	case ChangeRevision:
		x := c.Revision
		if !validID(x.Workspace) || !ascending(x.Methods, 1, len(Methods), func(m string) bool { return oneOf(m, Methods...) }) ||
			!ascending(x.Issuers, 0, maxIssuers, validID) || !oneOf(x.MFA, MFAScopes...) || !ageOK(x.MaxAge, MaxPolicyAge) || len(x.StepUp) > maxRules {
			return Change{}, ErrInvalid
		}
		for index, rule := range x.StepUp {
			if !validID(rule.Action) || !ageOK(rule.MaxAge, x.MaxAge) || index > 0 && x.StepUp[index-1].Action >= rule.Action {
				return Change{}, ErrInvalid
			}
		}
		return copyChange(Change{Kind: c.Kind, Revision: x}), nil
	case ChangePlan:
		x := c.Plan
		if x.ID != "" || !validID(x.Workspace) || !ascending(x.Users, 1, maxPlanUsers, validID) || x.Chunk < 1 || x.Chunk > maxChunk || !validText(x.Reason) {
			return Change{}, ErrInvalid
		}
		return copyChange(Change{Kind: c.Kind, Plan: x}), nil
	case ChangeLift:
		if !validPlanID(c.Lift.Plan) || !validText(c.Lift.Reason) {
			return Change{}, ErrInvalid
		}
		return Change{Kind: c.Kind, Lift: c.Lift}, nil
	}
	return Change{}, ErrInvalid
}

func (r *Registry) check(c Change, proposer, approver string) error {
	switch c.Kind {
	case ChangeRevision:
		if r.workspaces[c.Revision.Workspace] == nil && len(r.workspaces) >= maxWorkspaces {
			return ErrLimit
		}
	case ChangePlan:
		if r.workspaces[c.Plan.Workspace].active >= maxActivePlans {
			return ErrLimit
		}
	case ChangeLift:
		p := r.plans[c.Lift.Plan]
		switch {
		case p == nil:
			return ErrNotFound
		case p.lifted:
			return refuse(ReasonLifted)
		case contains(p.view.Users, proposer) || approver != "" && contains(p.view.Users, approver):
			return refuse(ReasonInScope)
		}
	}
	return nil
}

func (r *Registry) apply(c Change, by people, now time.Time) {
	switch c.Kind {
	case ChangeRevision:
		w := r.workspaces[c.Revision.Workspace]
		if w == nil {
			w = &workspace{}
			r.workspaces[c.Revision.Workspace] = w
		}
		w.number++
		w.revisions = append(w.revisions, RevisionView{Revision: copyChange(c).Revision, Number: w.number, Proposer: by.proposer, Approver: by.approver, At: now})
		if len(w.revisions) > maxKeptRevisions {
			w.revisions = w.revisions[1:]
		}
	case ChangePlan:
		space := c.Plan.Workspace
		w := r.workspaces[space]
		w.plans++
		x := copyChange(c).Plan
		x.ID = space + "/" + strconv.FormatUint(w.plans, 10)
		r.plans[x.ID] = &plan{view: x, by: by}
		for _, user := range x.Users {
			r.blocked[[2]string{space, user}]++
		}
		w.active++
	case ChangeLift:
		p := r.plans[c.Lift.Plan]
		space := p.view.Workspace
		p.lifted, p.liftBy = true, by
		for _, user := range p.view.Users {
			key := [2]string{space, user}
			if r.blocked[key]--; r.blocked[key] == 0 {
				delete(r.blocked, key)
			}
		}
		w := r.workspaces[space]
		w.active--
		r.reserve(space, -len(p.view.Users))
		r.lifted = append(r.lifted, p.view.ID)
		if len(r.lifted) > maxLiftedKept {
			delete(r.plans, r.lifted[0])
			r.lifted = r.lifted[1:]
		}
	}
}

func (r *Registry) bump(space string) {
	r.workspaces[space].version++
	for _, number := range append([]uint64{}, r.pending...) {
		if p := r.proposals[number]; p.view.Workspace == space {
			delete(r.proposals, number)
			r.pending = drop(r.pending, number)
			r.reserve(space, -len(p.view.Change.Plan.Users))
		}
	}
}
