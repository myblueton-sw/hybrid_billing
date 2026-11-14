package cases

import (
	"sort"
	"time"
)

const (
	caseAssign   = "assign"
	caseDue      = "due"
	caseClose    = "close"
	caseReopen   = "reopen"
	caseWithdraw = "withdraw"
)

func caseForm(c Case) bool {
	if !validID(c.ID) || !caseKinds[c.Kind] || !validID(c.Customer) || !topics[c.Topic] || !validDigest(c.Body) {
		return false
	}
	if c.Kind == KindDispute && len(c.Subjects) == 0 {
		return false
	}
	seen := map[Subject]bool{}
	for _, subject := range c.Subjects {
		if !validID(subject.Document) || !optionalID(subject.Line) || seen[subject] {
			return false
		}
		seen[subject] = true
	}
	if c.Amount == "" && c.Currency == "" {
		return true
	}
	return c.Kind == KindDispute && validNumber(c.Amount, false) && !zeroText(c.Amount) && validCurrency(c.Currency)
}

func zeroText(text string) bool {
	for index := 0; index < len(text); index++ {
		if text[index] != '0' && text[index] != '.' {
			return false
		}
	}
	return true
}

func sameCase(a, b Case) bool {
	if a.ID != b.ID || a.Kind != b.Kind || a.Customer != b.Customer || a.Topic != b.Topic || a.Amount != b.Amount || a.Currency != b.Currency || a.Body != b.Body || len(a.Subjects) != len(b.Subjects) {
		return false
	}
	for index := range a.Subjects {
		if a.Subjects[index] != b.Subjects[index] {
			return false
		}
	}
	return true
}

func copyCase(c Case) Case {
	c.Subjects = append([]Subject(nil), c.Subjects...)
	return c
}

func publicRemedy(m Remedy) bool {
	return m.Kind == "correction_document" || m.Kind == "credit_document"
}

func viewCase(c *billingCase, actor Actor, now time.Time) CaseView {
	v := CaseView{Case: copyCase(c.c), State: c.state, OpenedSide: c.openedSide, Opened: c.opened, Due: c.due, Outcome: c.outcome, Entries: []EntryView{}}
	v.ResponseOverdue = c.state == CaseOpen && c.awaiting && !c.due.IsZero() && !now.Before(c.due)
	operator := actor.Side == SideOperator
	for _, e := range c.entries {
		switch {
		case operator:
			v.Entries = append(v.Entries, e)
		case e.Kind == EntryNote:
		case e.Side == SideOperator:
			e.Actor, e.Digest = "", ""
			v.Entries = append(v.Entries, e)
		default:
			v.Entries = append(v.Entries, e)
		}
	}
	if operator {
		v.OpenedBy, v.Owner, v.ClosedBy, v.Closed, v.OriginalDue, v.Remedy, v.Digest = c.openedBy, c.owner, c.closedBy, c.closed, c.originalDue, c.remedy, c.digest
		v.DueHistory = append([]DueChange(nil), c.dues...)
		v.Owners = append([]OwnerChange(nil), c.owners...)
		v.Closings = append([]ClosingView(nil), c.closings...)
		return v
	}
	if c.openedSide == SideCustomer {
		v.OpenedBy = c.openedBy
	}
	if publicRemedy(c.remedy) {
		v.Remedy = c.remedy
	}
	return v
}

func (r *Registry) OpenCase(workspace string, c Case, actor Actor) (CaseView, error) {
	if !validID(workspace) || !validActor(actor) || !caseForm(c) {
		return CaseView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	if actor.Side == SideCustomer && actor.Customer != c.Customer {
		return CaseView{}, ErrNotFound
	}
	key := [2]string{c.Customer, c.ID}
	s := r.workspaces[workspace]
	if s != nil {
		if old := s.cases[key]; old != nil {
			if !sameCase(old.c, c) || old.openedBy != actor.ID || old.openedSide != actor.Side {
				return CaseView{}, ErrConflict
			}
			return viewCase(old, actor, now), nil
		}
	}
	if len(c.Subjects) > r.limits.Subjects {
		return CaseView{}, ErrLimit
	}
	if s != nil {
		if len(s.cases) >= r.limits.Cases || s.customers[c.Customer] >= r.limits.CustomerCases {
			return CaseView{}, ErrLimit
		}
	} else if len(r.workspaces) >= r.limits.Workspaces {
		return CaseView{}, ErrLimit
	}
	s = r.space(workspace)
	made := &billingCase{c: copyCase(c), state: CaseOpen, openedBy: actor.ID, openedSide: actor.Side, opened: now, awaiting: true,
		public: map[string]int{}, notes: map[string]int{}, steps: map[[2]string]caseStep{}, digest: caseDigest(workspace, c, actor)}
	s.cases[key] = made
	s.customers[c.Customer]++
	return viewCase(made, actor, now), nil
}

func (r *Registry) find(workspace, customer, id string, actor Actor) *billingCase {
	if actor.Side == SideCustomer && actor.Customer != customer {
		return nil
	}
	s := r.workspaces[workspace]
	if s == nil {
		return nil
	}
	return s.cases[[2]string{customer, id}]
}

func addressForm(workspace, customer, id string, actor Actor) bool {
	return validID(workspace) && validID(customer) && validID(id) && validActor(actor)
}

func (r *Registry) Case(workspace, customer, id string, actor Actor) (CaseView, error) {
	if !addressForm(workspace, customer, id, actor) {
		return CaseView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	c := r.find(workspace, customer, id, actor)
	if c == nil {
		return CaseView{}, ErrNotFound
	}
	return viewCase(c, actor, now), nil
}

func caseStepForm(kind string, st CaseStep) bool {
	if !validID(st.Step) {
		return false
	}
	empty := Closing{}
	switch kind {
	case caseAssign:
		return validID(st.Owner) && st.Due.IsZero() && st.Reason == "" && st.Closing == empty
	case caseDue:
		return st.Owner == "" && !st.Due.IsZero() && dueReasons[st.Reason] && st.Closing == empty
	case caseClose:
		return st.Owner == "" && st.Due.IsZero() && st.Reason == "" && closingShape(st.Closing)
	}
	return st.Owner == "" && st.Due.IsZero() && st.Reason == "" && st.Closing == empty
}

func closingShape(cl Closing) bool {
	switch cl.Outcome {
	case OutcomeAnswered, OutcomeNotUpheld:
		return cl.Remedy == (Remedy{})
	case OutcomeUpheld:
		return caseRemedies[cl.Remedy.Kind] && validID(cl.Remedy.Reference)
	}
	return false
}

func (r *Registry) AssignCase(workspace, customer, id string, actor Actor, st CaseStep) (CaseView, error) {
	return r.step(workspace, customer, id, actor, caseAssign, st)
}

func (r *Registry) SetDue(workspace, customer, id string, actor Actor, st CaseStep) (CaseView, error) {
	return r.step(workspace, customer, id, actor, caseDue, st)
}

func (r *Registry) Close(workspace, customer, id string, actor Actor, st CaseStep) (CaseView, error) {
	return r.step(workspace, customer, id, actor, caseClose, st)
}

func (r *Registry) ReopenCase(workspace, customer, id string, actor Actor, st CaseStep) (CaseView, error) {
	return r.step(workspace, customer, id, actor, caseReopen, st)
}

func (r *Registry) Withdraw(workspace, customer, id string, actor Actor, st CaseStep) (CaseView, error) {
	return r.step(workspace, customer, id, actor, caseWithdraw, st)
}

func (r *Registry) step(workspace, customer, id string, actor Actor, kind string, st CaseStep) (CaseView, error) {
	if !addressForm(workspace, customer, id, actor) || !caseStepForm(kind, st) {
		return CaseView{}, ErrInvalid
	}
	st.Due = st.Due.UTC().Round(0)
	now := r.enter()
	defer r.leave()
	c := r.find(workspace, customer, id, actor)
	if c == nil {
		return CaseView{}, ErrNotFound
	}
	key := [2]string{actor.Side, st.Step}
	if old, ok := c.steps[key]; ok {
		if old.kind != kind || old.actor != actor.ID || !old.step.Due.Equal(st.Due) || old.step.Owner != st.Owner || old.step.Reason != st.Reason || old.step.Closing != st.Closing {
			return CaseView{}, ErrConflict
		}
		return viewCase(c, actor, now), nil
	}
	if !r.allowed(c, actor, kind, st, now) {
		return CaseView{}, ErrRejected
	}
	if (kind == caseAssign || kind == caseDue || kind == caseReopen) && c.counted >= r.limits.Steps {
		return CaseView{}, ErrLimit
	}
	c.steps[key] = caseStep{kind: kind, step: st, actor: actor.ID}
	switch kind {
	case caseAssign:
		c.owner, c.counted = st.Owner, c.counted+1
		c.owners = append(c.owners, OwnerChange{Step: st.Step, Owner: st.Owner, Actor: actor.ID, At: now, Digest: moveDigest(workspace, customer, id, kind, st.Step, actor.ID, now, st.Owner)})
	case caseDue:
		if c.originalDue.IsZero() {
			c.originalDue = st.Due
		}
		c.due, c.counted = st.Due, c.counted+1
		c.dues = append(c.dues, DueChange{Due: st.Due, Reason: st.Reason, Actor: actor.ID, At: now, Digest: moveDigest(workspace, customer, id, kind, st.Step, actor.ID, now, stamp(st.Due), st.Reason)})
	case caseClose:
		c.state, c.outcome, c.remedy, c.closedBy, c.closed = CaseClosed, st.Closing.Outcome, st.Closing.Remedy, actor.ID, now
		c.closings = append(c.closings, ClosingView{Outcome: st.Closing.Outcome, Remedy: st.Closing.Remedy, ClosedBy: actor.ID, Closed: now,
			Digest: moveDigest(workspace, customer, id, kind, st.Step, actor.ID, now, st.Closing.Outcome, st.Closing.Remedy.Kind, st.Closing.Remedy.Reference)})
	case caseReopen:
		last := &c.closings[len(c.closings)-1]
		last.Reopened, last.Reopener, last.Reopen = now, actor.ID, moveDigest(workspace, customer, id, kind, st.Step, actor.ID, now, actor.Side)
		c.state, c.outcome, c.remedy, c.closedBy, c.closed, c.awaiting, c.counted = CaseOpen, "", Remedy{}, "", time.Time{}, true, c.counted+1
		c.clearDue(workspace, customer, id, "cleared_on_reopen", actor.ID, now)
	case caseWithdraw:
		c.state, c.closedBy, c.closed = CaseWithdrawn, actor.ID, now
	}
	return viewCase(c, actor, now), nil
}

func (c *billingCase) clearDue(workspace, customer, id, reason, actor string, now time.Time) {
	if c.due.IsZero() {
		return
	}
	c.due = time.Time{}
	c.dues = append(c.dues, DueChange{Reason: reason, Actor: actor, At: now, Digest: moveDigest(workspace, customer, id, caseDue, "", actor, now, "", reason)})
}

func (r *Registry) allowed(c *billingCase, actor Actor, kind string, st CaseStep, now time.Time) bool {
	live := c.state == CaseOpen || c.state == CaseAnswered
	operator := actor.Side == SideOperator
	switch kind {
	case caseAssign:
		return operator && live
	case caseDue:
		return operator && live && st.Due.After(now) && st.Due.Sub(now) <= maxDueAhead && (st.Reason == "initial") == c.originalDue.IsZero()
	case caseClose:
		return operator && live && (st.Closing.Outcome == OutcomeAnswered) == (c.c.Kind == KindInquiry)
	case caseReopen:
		return c.state == CaseClosed
	}
	return !operator && live
}

func (r *Registry) Note(workspace, customer, id string, actor Actor, e Entry) (CaseView, error) {
	return r.write(workspace, customer, id, actor, EntryNote, e)
}

func (r *Registry) Answer(workspace, customer, id string, actor Actor, e Entry) (CaseView, error) {
	return r.write(workspace, customer, id, actor, EntryAnswer, e)
}

func (r *Registry) Reply(workspace, customer, id string, actor Actor, e Entry) (CaseView, error) {
	return r.write(workspace, customer, id, actor, EntryMessage, e)
}

func (r *Registry) write(workspace, customer, id string, actor Actor, kind string, e Entry) (CaseView, error) {
	if !addressForm(workspace, customer, id, actor) || !validID(e.ID) || !validDigest(e.Body) {
		return CaseView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	c := r.find(workspace, customer, id, actor)
	if c == nil {
		return CaseView{}, ErrNotFound
	}
	space := c.public
	if kind == EntryNote {
		space = c.notes
		if actor.Side != SideOperator {
			return CaseView{}, ErrRejected
		}
	}
	if index, ok := space[e.ID]; ok {
		old := c.entries[index]
		if old.Kind != kind || old.Body != e.Body || old.Actor != actor.ID || old.Side != actor.Side {
			return CaseView{}, ErrConflict
		}
		return viewCase(c, actor, now), nil
	}
	live := c.state == CaseOpen || c.state == CaseAnswered
	if (kind == EntryMessage) != (actor.Side == SideCustomer) || kind != EntryNote && !live {
		return CaseView{}, ErrRejected
	}
	if kind == EntryMessage && c.messages >= r.limits.Messages || kind != EntryMessage && c.written >= r.limits.Entries {
		return CaseView{}, ErrLimit
	}
	view := EntryView{ID: e.ID, Kind: kind, Body: e.Body, Actor: actor.ID, Side: actor.Side, At: now}
	view.Digest = entryDigest(workspace, customer, id, view)
	space[e.ID] = len(c.entries)
	c.entries = append(c.entries, view)
	switch kind {
	case EntryAnswer:
		c.state, c.awaiting, c.written = CaseAnswered, false, c.written+1
	case EntryNote:
		c.written++
	case EntryMessage:
		if c.state == CaseAnswered {
			c.clearDue(workspace, customer, id, "cleared_on_message", actor.ID, now)
		}
		c.state, c.awaiting, c.messages = CaseOpen, true, c.messages+1
	}
	return viewCase(c, actor, now), nil
}

func caseFilterForm(f CaseFilter) bool {
	return (f.State == "" || caseStates[f.State]) && optionalID(f.Customer) && (f.Kind == "" || caseKinds[f.Kind]) && optionalID(f.Document) &&
		optionalID(f.AfterCustomer) && optionalID(f.After) && (f.After == "" || f.AfterCustomer != "")
}

var caseStates = map[string]bool{CaseOpen: true, CaseAnswered: true, CaseClosed: true, CaseWithdrawn: true}

func names(c *billingCase, document string) bool {
	for _, subject := range c.c.Subjects {
		if subject.Document == document {
			return true
		}
	}
	return false
}

func after(key [2]string, customer, id string) bool {
	return key[0] > customer || key[0] == customer && key[1] > id
}

func (r *Registry) Cases(workspace string, actor Actor, filter CaseFilter) ([]CaseView, error) {
	if !validID(workspace) || !validActor(actor) || !caseFilterForm(filter) {
		return nil, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	out := []CaseView{}
	if s == nil {
		return out, nil
	}
	keys := [][2]string{}
	for key, c := range s.cases {
		if actor.Side == SideCustomer && key[0] != actor.Customer || !after(key, filter.AfterCustomer, filter.After) {
			continue
		}
		if (filter.State == "" || c.state == filter.State) && (filter.Customer == "" || key[0] == filter.Customer) && (filter.Kind == "" || c.c.Kind == filter.Kind) && (filter.Document == "" || names(c, filter.Document)) {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(a, b int) bool { return after(keys[b], keys[a][0], keys[a][1]) })
	if len(keys) > r.limits.Page {
		keys = keys[:r.limits.Page]
	}
	for _, key := range keys {
		out = append(out, viewCase(s.cases[key], actor, now))
	}
	return out, nil
}

func (r *Registry) Disputes(workspace, customer, document, after string, actor Actor) ([]CaseView, error) {
	if !validActor(actor) {
		return nil, ErrInvalid
	}
	if actor.Side != SideOperator {
		return nil, ErrRejected
	}
	if !validID(workspace) || !validID(customer) || !validID(document) || !optionalID(after) {
		return nil, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	out := []CaseView{}
	if s == nil {
		return out, nil
	}
	keys := [][2]string{}
	for key, c := range s.cases {
		if key[0] == customer && key[1] > after && c.c.Kind == KindDispute && (c.state == CaseOpen || c.state == CaseAnswered) && names(c, document) {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(a, b int) bool { return keys[a][1] < keys[b][1] })
	if len(keys) > r.limits.Page {
		keys = keys[:r.limits.Page]
	}
	for _, key := range keys {
		out = append(out, viewCase(s.cases[key], actor, now))
	}
	return out, nil
}
