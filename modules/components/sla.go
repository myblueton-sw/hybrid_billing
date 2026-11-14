package components

import "time"

func incidentForm(i Incident) bool {
	return validID(i.ID) && validID(i.Contract) && validID(i.Component) && !i.Start.IsZero() && i.End.After(i.Start) && validDigest(i.Evidence) &&
		validDate(i.Start.UTC().Format(dateLayout)) && validDate(i.End.UTC().Format(dateLayout))
}

func (r *Registry) RecordIncident(workspace string, i Incident, actor string) (IncidentView, error) {
	if !validID(workspace) || !validID(actor) || !incidentForm(i) {
		return IncidentView{}, ErrInvalid
	}
	i.Start, i.End = i.Start.UTC().Round(0), i.End.UTC().Round(0)
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.components[i.Component] == nil || s.components[i.Component].view.Component.Contract != i.Contract {
		return IncidentView{}, ErrNotFound
	}
	if old := s.incidents[i.ID]; old != nil {
		if old.view.Incident != i || old.view.Author != actor {
			return IncidentView{}, ErrConflict
		}
		return old.view, nil
	}
	if i.End.After(now) {
		return IncidentView{}, ErrRejected
	}
	if s.counts[i.Contract] >= r.limits.Incidents {
		return IncidentView{}, ErrLimit
	}
	made := &incident{view: IncidentView{Incident: i, Author: actor, Written: now, Digest: incidentDigest(workspace, i, actor)}}
	s.incidents[i.ID] = made
	s.counts[i.Contract]++
	return made.view, nil
}

func (r *Registry) IncidentOf(workspace, id string) (IncidentView, error) {
	if !validID(workspace) || !validID(id) {
		return IncidentView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.incidents[id] == nil {
		return IncidentView{}, ErrNotFound
	}
	return s.incidents[id].view, nil
}

func clauseForm(c Clause) bool {
	return validID(c.ID) && validID(c.Contract) && validID(c.Component) && validDate(c.From) && (c.To == "" || validDate(c.To) && c.To >= c.From)
}

func (r *Registry) AddClause(workspace string, c Clause, actor string) (ClauseView, error) {
	if !validID(workspace) || !validID(actor) || !clauseForm(c) {
		return ClauseView{}, ErrInvalid
	}
	rate, printed, ok := r.number(c.Rate, true)
	if !ok || exactOne().Less(rate) {
		return ClauseView{}, ErrInvalid
	}
	c.Rate = printed
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.components[c.Component] == nil || s.components[c.Component].view.Component.Contract != c.Contract {
		return ClauseView{}, ErrNotFound
	}
	if old := s.clauses[c.ID]; old != nil {
		if old.view.Clause != c || old.view.Author != actor {
			return ClauseView{}, ErrConflict
		}
		return old.view, nil
	}
	comp := s.components[c.Component]
	k := comp.view.Component
	if !fixedKinds[k.Kind] || c.From < k.From || endOf(c.To) > endOf(k.To) {
		return ClauseView{}, refuse(ReasonClauseOutside)
	}
	if len(s.byContract[c.Contract]) >= r.limits.Clauses {
		return ClauseView{}, ErrLimit
	}
	made := &clause{rate: rate, comp: comp, view: ClauseView{Clause: c, State: StatePending, Author: actor, Written: now, Digest: clauseDigest(workspace, c, actor)}}
	s.clauses[c.ID] = made
	s.byContract[c.Contract] = append(s.byContract[c.Contract], made)
	return made.view, nil
}

func (r *Registry) ApproveClause(workspace, id, actor string) (ClauseView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return ClauseView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.clauses[id] == nil {
		return ClauseView{}, ErrNotFound
	}
	c := s.clauses[id]
	if c.view.State == StateApproved && c.view.Approver == actor {
		return c.view, nil
	}
	if c.view.State != StatePending || c.view.Author == actor {
		return ClauseView{}, ErrRejected
	}
	me := c.view.Clause
	for _, other := range s.byContract[me.Contract] {
		o := other.view.Clause
		if other != c && other.view.State == StateApproved && o.Component == me.Component && overlap(me.From, me.To, o.From, o.To) {
			return ClauseView{}, refuse(ReasonClauseOverlap)
		}
	}
	c.view.State, c.view.Approver, c.view.ApprovedAt = StateApproved, actor, now
	return c.view, nil
}

func copyCredit(c Credit) Credit {
	c.Incidents = append([]string(nil), c.Incidents...)
	return c
}

func sameCredit(a, b Credit) bool {
	if a.ID != b.ID || a.Clause != b.Clause || a.PeriodStart != b.PeriodStart || a.PeriodEnd != b.PeriodEnd || len(a.Incidents) != len(b.Incidents) {
		return false
	}
	for index := range a.Incidents {
		if a.Incidents[index] != b.Incidents[index] {
			return false
		}
	}
	return true
}

func (r *Registry) creditForm(c Credit) bool {
	if !validID(c.ID) || !validID(c.Clause) || !month(c.PeriodStart, c.PeriodEnd) || len(c.Incidents) == 0 || len(c.Incidents) > r.limits.IncidentsPerCredit {
		return false
	}
	seen := map[string]bool{}
	for _, id := range c.Incidents {
		if !validID(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func viewCredit(c *credit) CreditView {
	v := c.view
	v.Credit = copyCredit(c.view.Credit)
	return v
}

func (s *space) creditConflict(c *credit) error {
	comp := c.clause.comp
	k := comp.view.Component
	if c.clause.view.State != StateApproved || comp.view.State != StateApproved || c.view.Credit.PeriodStart < k.From || c.view.Credit.PeriodEnd > endOf(k.To) {
		return ErrRejected
	}
	for _, other := range s.credits {
		if other == c || other.view.State == StateWithdrawn || other.clause.comp != comp {
			continue
		}
		if overlap(c.view.Credit.PeriodStart, c.view.Credit.PeriodEnd, other.view.Credit.PeriodStart, other.view.Credit.PeriodEnd) {
			return refuse(ReasonCreditOverlap)
		}
	}
	for _, id := range c.view.Credit.Incidents {
		if used := s.incidents[id].view.Credit; used != "" && used != c.view.Credit.ID && s.credits[used].view.State != StateWithdrawn {
			return refuse(ReasonIncidentUsed)
		}
	}
	return nil
}

func (r *Registry) ProposeCredit(workspace string, c Credit, actor string) (CreditView, error) {
	if !validID(workspace) || !validID(actor) || !r.creditForm(c) {
		return CreditView{}, ErrInvalid
	}
	c = copyCredit(c)
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.clauses[c.Clause] == nil {
		return CreditView{}, ErrNotFound
	}
	for _, id := range c.Incidents {
		if s.incidents[id] == nil {
			return CreditView{}, ErrNotFound
		}
	}
	if old := s.credits[c.ID]; old != nil {
		if !sameCredit(old.view.Credit, c) || old.view.Author != actor {
			return CreditView{}, ErrConflict
		}
		return viewCredit(old), nil
	}
	cl := s.clauses[c.Clause]
	for _, id := range c.Incidents {
		i := s.incidents[id].view.Incident
		from, to := i.Start.Format(dateLayout), i.End.Add(-time.Nanosecond).Format(dateLayout)
		if i.Component != cl.view.Clause.Component || to < c.PeriodStart || from > c.PeriodEnd || from < cl.view.Clause.From || to > endOf(cl.view.Clause.To) {
			return CreditView{}, refuse(ReasonIncidentOutsidePeriod)
		}
	}
	if c.PeriodStart < cl.view.Clause.From || c.PeriodEnd > endOf(cl.view.Clause.To) {
		return CreditView{}, ErrRejected
	}
	amount := round(cl.comp.fee.Times(cl.rate), s.pricing[cl.view.Clause.Contract].Pricing)
	made := &credit{clause: cl, amount: amount, view: CreditView{Credit: c, Component: cl.view.Clause.Component, Amount: show(amount), State: StatePending, Author: actor, Written: now}}
	if err := s.creditConflict(made); err != nil {
		return CreditView{}, err
	}
	if amount.Sign() <= 0 {
		return CreditView{}, refuse(ReasonZeroCredit)
	}
	if s.perContract[cl.view.Clause.Contract] >= r.limits.Credits {
		return CreditView{}, ErrLimit
	}
	made.view.Digest = creditDigest(workspace, c, made.view.Amount, actor)
	s.credits[c.ID] = made
	s.perContract[cl.view.Clause.Contract]++
	for _, id := range c.Incidents {
		s.incidents[id].view.Credit = c.ID
	}
	return viewCredit(made), nil
}

func (r *Registry) ApproveCredit(workspace, id, actor string) (CreditView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return CreditView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.credits[id] == nil {
		return CreditView{}, ErrNotFound
	}
	c := s.credits[id]
	if c.view.State == StateApproved && c.view.Approver == actor {
		return viewCredit(c), nil
	}
	if c.view.State != StatePending || c.view.Author == actor {
		return CreditView{}, ErrRejected
	}
	if err := s.creditConflict(c); err != nil {
		return CreditView{}, err
	}
	c.view.State, c.view.Approver, c.view.ApprovedAt = StateApproved, actor, now
	return viewCredit(c), nil
}

func (r *Registry) WithdrawCredit(workspace, id, actor string) (CreditView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return CreditView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.credits[id] == nil {
		return CreditView{}, ErrNotFound
	}
	c := s.credits[id]
	if c.view.State == StateWithdrawn && c.view.WithdrawnBy == actor {
		return viewCredit(c), nil
	}
	if c.view.State != StatePending {
		return CreditView{}, ErrRejected
	}
	c.view.State, c.view.WithdrawnBy = StateWithdrawn, actor
	for _, incident := range c.view.Credit.Incidents {
		if s.incidents[incident].view.Credit == id {
			s.incidents[incident].view.Credit = ""
		}
	}
	return viewCredit(c), nil
}

func (r *Registry) LinkCredit(workspace, id string, l Link, actor string) (CreditView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) || !validID(l.Document) || !validID(l.Increase) {
		return CreditView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.credits[id] == nil {
		return CreditView{}, ErrNotFound
	}
	c := s.credits[id]
	if c.view.Link != (Link{}) {
		if c.view.Link != l || c.view.LinkedBy != actor {
			return CreditView{}, ErrConflict
		}
		return viewCredit(c), nil
	}
	if c.view.State != StateApproved {
		return CreditView{}, ErrRejected
	}
	if s.documents[l.Document] != "" || s.increases[l.Increase] != "" {
		return CreditView{}, refuse(ReasonLinkUsed)
	}
	c.view.Link, c.view.LinkedBy, c.view.LinkedAt = l, actor, now
	s.documents[l.Document], s.increases[l.Increase] = id, id
	return viewCredit(c), nil
}

func (r *Registry) CreditOf(workspace, id string) (CreditView, error) {
	if !validID(workspace) || !validID(id) {
		return CreditView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.credits[id] == nil {
		return CreditView{}, ErrNotFound
	}
	return viewCredit(s.credits[id]), nil
}
