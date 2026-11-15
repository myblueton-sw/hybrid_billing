package closing

import (
	"sort"
	"time"
)

func (r *Registry) item(workspace, id, target, stageName string) (*cycle, *item, error) {
	_, c, err := r.cycle(workspace, id)
	if err != nil {
		return nil, nil, err
	}
	st := c.byStage[stageName]
	if st == nil || st.byTarget[target] == nil {
		return nil, nil, ErrNotFound
	}
	return c, st.byTarget[target], nil
}

func itemForm(workspace, id, target, stageName string, actor Actor) bool {
	return validID(workspace) && validID(id) && validID(target) && stageOrder[stageName] > 0 && actor.valid()
}

func (it *item) record(c *cycle, e ItemEvent) {
	e.Sequence = c.seq
	it.events = append(it.events, e)
}

func late(at, due time.Time) time.Duration {
	if at.After(due) {
		return at.Sub(due)
	}
	return 0
}

func (r *Registry) Complete(workspace, id, target, stageName string, e Evidence, actor Actor) (ItemView, error) {
	now := r.enter()
	defer r.leave()
	if !itemForm(workspace, id, target, stageName, actor) || !validID(e.Kind) || !validID(e.Reference) {
		return ItemView{}, ErrInvalid
	}
	if err := actor.notAssistant(); err != nil {
		return ItemView{}, err
	}
	c, it, err := r.item(workspace, id, target, stageName)
	if err != nil {
		return ItemView{}, err
	}
	spec := it.stage.view.Spec
	if spec.Basis == BasisInternal && actor.Kind != KindHuman {
		return ItemView{}, refuse(RefusalNotHuman)
	}
	if it.state == ItemDone {
		if last := it.events[len(it.events)-1]; last.Evidence == e {
			return r.view(c, it, now), nil
		}
		return ItemView{}, ErrConflict
	}
	switch {
	case bases[spec.Basis] != e.Kind:
		return ItemView{}, refuse(RefusalBasisMismatch)
	case it.waiver != nil:
		return ItemView{}, refuse(RefusalWaiverPending)
	case it.state != ItemOpen:
		return ItemView{}, refuse(RefusalState)
	}
	for _, earlier := range c.stages[:it.stage.index] {
		if state := earlier.byTarget[target].state; state != ItemDone && state != ItemWaived {
			return ItemView{}, refuse(RefusalStageOrder)
		}
	}
	if err := r.room(c, 1); err != nil {
		return ItemView{}, err
	}
	c.seq++
	it.state = ItemDone
	it.stage.sync(it)
	it.record(c, ItemEvent{Kind: EventCompleted, Actor: actor.ID, At: now, Evidence: e, LateOriginal: late(now, it.stage.view.OriginalDue), LateCurrent: late(now, it.stage.view.CurrentDue)})
	return r.view(c, it, now), nil
}

func (r *Registry) Reopen(workspace, id, target, stageName, reason string, actor Actor) (ItemView, error) {
	now := r.enter()
	defer r.leave()
	if !itemForm(workspace, id, target, stageName, actor) || !validID(reason) {
		return ItemView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return ItemView{}, err
	}
	c, it, err := r.item(workspace, id, target, stageName)
	if err != nil {
		return ItemView{}, err
	}
	if it.state != ItemDone {
		return ItemView{}, refuse(RefusalState)
	}
	var reopened []*item
	for _, st := range c.stages[it.stage.index:] {
		if other := st.byTarget[target]; other.state == ItemDone {
			reopened = append(reopened, other)
		}
	}
	if err := r.room(c, len(reopened)); err != nil {
		return ItemView{}, err
	}
	c.seq++
	for _, other := range reopened {
		other.state = ItemOpen
		other.stage.sync(other)
		other.record(c, ItemEvent{Kind: EventReopened, Actor: actor.ID, At: now, Reason: reason})
	}
	return r.view(c, it, now), nil
}

func (r *Registry) change(workspace, id, target, stageName, reason string, actor Actor, from, to, kind string) (ItemView, error) {
	now := r.enter()
	defer r.leave()
	if !itemForm(workspace, id, target, stageName, actor) || reason != "" && !validID(reason) {
		return ItemView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return ItemView{}, err
	}
	c, it, err := r.item(workspace, id, target, stageName)
	if err != nil {
		return ItemView{}, err
	}
	switch {
	case it.waiver != nil:
		return ItemView{}, refuse(RefusalWaiverPending)
	case it.state != from:
		return ItemView{}, refuse(RefusalState)
	}
	if err := r.room(c, 1); err != nil {
		return ItemView{}, err
	}
	c.seq++
	it.state = to
	it.stage.sync(it)
	it.record(c, ItemEvent{Kind: kind, Actor: actor.ID, At: now, Reason: reason})
	return r.view(c, it, now), nil
}

func (r *Registry) Hold(workspace, id, target, stageName, reason string, actor Actor) (ItemView, error) {
	if !validID(reason) {
		return ItemView{}, ErrInvalid
	}
	return r.change(workspace, id, target, stageName, reason, actor, ItemOpen, ItemHeld, EventHeld)
}

func (r *Registry) Release(workspace, id, target, stageName string, actor Actor) (ItemView, error) {
	return r.change(workspace, id, target, stageName, "", actor, ItemHeld, ItemOpen, EventReleased)
}

func (r *Registry) RequestWaiver(workspace, id, target, stageName string, w Waiver, actor Actor) (ItemView, error) {
	now := r.enter()
	defer r.leave()
	if !itemForm(workspace, id, target, stageName, actor) || !validID(w.Reason) || !validID(w.Evidence) {
		return ItemView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return ItemView{}, err
	}
	c, it, err := r.item(workspace, id, target, stageName)
	if err != nil {
		return ItemView{}, err
	}
	if it.waiver != nil {
		if actor.is(it.waiver.requester) && it.waiver.waiver == w {
			return r.view(c, it, now), nil
		}
		return ItemView{}, refuse(RefusalWaiverPending)
	}
	if it.state != ItemOpen && it.state != ItemHeld {
		return ItemView{}, refuse(RefusalState)
	}
	if err := r.room(c, 1); err != nil {
		return ItemView{}, err
	}
	c.seq++
	it.waiver = &pendingWaiver{waiver: w, requester: actor.ID}
	it.record(c, ItemEvent{Kind: EventWaiverRequested, Actor: actor.ID, At: now, Reason: w.Reason, Evidence: Evidence{Kind: "waiver", Reference: w.Evidence}})
	return r.view(c, it, now), nil
}

func (r *Registry) decideWaiver(workspace, id, target, stageName string, actor Actor, approve bool) (ItemView, error) {
	now := r.enter()
	defer r.leave()
	if !itemForm(workspace, id, target, stageName, actor) {
		return ItemView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return ItemView{}, err
	}
	c, it, err := r.item(workspace, id, target, stageName)
	if err != nil {
		return ItemView{}, err
	}
	if it.waiver == nil {
		if approve && it.state == ItemWaived && actor.is(it.approver) {
			return r.view(c, it, now), nil
		}
		return ItemView{}, refuse(RefusalNotPending)
	}
	switch {
	case approve && actor.ID == it.waiver.requester:
		return ItemView{}, refuse(RefusalSelfApproval)
	case !approve && actor.ID != it.waiver.requester:
		return ItemView{}, refuse(RefusalNotRequester)
	}
	if err := r.room(c, 1); err != nil {
		return ItemView{}, err
	}
	c.seq++
	kind := EventWaiverWithdrawn
	if approve {
		kind = EventWaiverApproved
		it.state, it.approver = ItemWaived, actor.ID
		it.stage.sync(it)
	}
	it.record(c, ItemEvent{Kind: kind, Actor: actor.ID, At: now, Reason: it.waiver.waiver.Reason})
	it.waiver = nil
	return r.view(c, it, now), nil
}

func (r *Registry) ApproveWaiver(workspace, id, target, stageName string, actor Actor) (ItemView, error) {
	return r.decideWaiver(workspace, id, target, stageName, actor, true)
}

func (r *Registry) WithdrawWaiver(workspace, id, target, stageName string, actor Actor) (ItemView, error) {
	return r.decideWaiver(workspace, id, target, stageName, actor, false)
}

func (r *Registry) view(c *cycle, it *item, at time.Time) ItemView {
	v := ItemView{Target: it.target, Stage: it.stage.view.Spec.Stage, State: it.state, History: append([]ItemEvent(nil), it.events...)}
	if it.waiver != nil {
		w := it.waiver.waiver
		v.Waiver = &w
	}
	if it.state == ItemOpen {
		v.LateOpen = late(at, it.stage.view.OriginalDue)
	}
	return v
}

func (r *Registry) Item(workspace, id, target, stageName string, at time.Time) (ItemView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(target) || stageOrder[stageName] == 0 {
		return ItemView{}, ErrInvalid
	}
	c, it, err := r.item(workspace, id, target, stageName)
	if err != nil {
		return ItemView{}, err
	}
	if at.IsZero() {
		at = now
	}
	return r.view(c, it, at), nil
}

func stateAt(it *item, sequence int) (string, bool) {
	state, present := "", false
	for _, e := range it.events {
		if e.Sequence > sequence {
			break
		}
		switch e.Kind {
		case EventAdded, EventReopened, EventReleased:
			state, present = ItemOpen, true
		case EventCompleted:
			state = ItemDone
		case EventHeld:
			state = ItemHeld
		case EventWaiverApproved:
			state = ItemWaived
		}
	}
	return state, present
}

func lastCompletion(it *item, sequence int) ItemEvent {
	var last ItemEvent
	for _, e := range it.events {
		if e.Sequence <= sequence && e.Kind == EventCompleted {
			last = e
		}
	}
	return last
}

func dueAt(st *stage, sequence int) time.Time {
	due := st.view.OriginalDue
	for _, change := range st.view.History {
		if change.Sequence <= sequence && change.Kind == EventPostponeApproved {
			due = change.Due
		}
	}
	return due
}

func (r *Registry) Progress(workspace, id, stageName string, as AsOf) (ProgressView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || stageOrder[stageName] == 0 || as.Sequence < 0 {
		return ProgressView{}, ErrInvalid
	}
	_, c, err := r.cycle(workspace, id)
	if err != nil {
		return ProgressView{}, err
	}
	st := c.byStage[stageName]
	if st == nil || as.Sequence > c.seq {
		return ProgressView{}, ErrNotFound
	}
	sequence, at := as.Sequence, as.At
	if sequence == 0 {
		sequence = c.seq
	}
	if at.IsZero() {
		at = now
	}
	due := dueAt(st, sequence)
	v := ProgressView{Sequence: sequence, OriginalDue: st.view.OriginalDue, CurrentDue: due}
	for _, it := range st.items {
		state, present := stateAt(it, sequence)
		if !present {
			continue
		}
		v.Targets++
		switch state {
		case ItemDone:
			v.Done++
			if lastCompletion(it, sequence).LateOriginal > 0 {
				v.DoneLate++
			}
		case ItemWaived:
			v.Waived++
		case ItemHeld:
			v.Held++
			if at.After(due) {
				v.HeldPastDue++
			}
		default:
			v.Open++
			if at.After(due) {
				v.OpenPastDue++
			}
		}
	}
	v.Denominator = v.Targets - v.Waived
	return v, nil
}

func (r *Registry) RequestPostponement(workspace, id, stageName string, p Postponement, actor Actor) (CycleView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || stageOrder[stageName] == 0 || p.Due.IsZero() || !validID(p.Reason) || !actor.valid() {
		return CycleView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return CycleView{}, err
	}
	_, c, err := r.cycle(workspace, id)
	if err != nil {
		return CycleView{}, err
	}
	st := c.byStage[stageName]
	if st == nil {
		return CycleView{}, ErrNotFound
	}
	p.Due = p.Due.UTC().Round(0)
	if st.view.Pending != nil {
		if actor.is(st.requester) && *st.view.Pending == p {
			return c.snapshot(), nil
		}
		return CycleView{}, refuse(RefusalPostponePending)
	}
	if err := r.order(c, st, p.Due, now); err != nil {
		return CycleView{}, err
	}
	if err := r.room(c, 1); err != nil {
		return CycleView{}, err
	}
	c.seq++
	pending := p
	st.view.Pending, st.requester = &pending, actor.ID
	st.view.History = append(st.view.History, DueChange{Sequence: c.seq, Kind: EventPostponeRequest, Due: p.Due, Reason: p.Reason, Actor: actor.ID, At: now})
	return c.snapshot(), nil
}

func (r *Registry) order(c *cycle, st *stage, due, now time.Time) error {
	if !due.After(st.view.CurrentDue) || !due.After(now) {
		return refuse(RefusalNotLater)
	}
	if st.index+1 < len(c.stages) && due.After(c.stages[st.index+1].view.CurrentDue) {
		return refuse(RefusalStageOrder)
	}
	return nil
}

func (r *Registry) decidePostponement(workspace, id, stageName string, actor Actor, approve bool) (CycleView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || stageOrder[stageName] == 0 || !actor.valid() {
		return CycleView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return CycleView{}, err
	}
	_, c, err := r.cycle(workspace, id)
	if err != nil {
		return CycleView{}, err
	}
	st := c.byStage[stageName]
	if st == nil {
		return CycleView{}, ErrNotFound
	}
	if st.view.Pending == nil {
		if last := len(st.view.History) - 1; approve && last >= 0 && st.view.History[last].Kind == EventPostponeApproved && actor.is(st.approver) {
			return c.snapshot(), nil
		}
		return CycleView{}, refuse(RefusalNotPending)
	}
	switch {
	case approve && actor.ID == st.requester:
		return CycleView{}, refuse(RefusalSelfApproval)
	case !approve && actor.ID != st.requester:
		return CycleView{}, refuse(RefusalNotRequester)
	}
	p := *st.view.Pending
	if approve {
		if err := r.order(c, st, p.Due, now); err != nil {
			return CycleView{}, err
		}
	}
	if err := r.room(c, 1); err != nil {
		return CycleView{}, err
	}
	c.seq++
	kind := EventPostponeWithdraw
	if approve {
		kind = EventPostponeApproved
		st.view.CurrentDue, st.approver = p.Due, actor.ID
		for _, it := range st.items {
			st.sync(it)
		}
	}
	st.view.History = append(st.view.History, DueChange{Sequence: c.seq, Kind: kind, Due: p.Due, Reason: p.Reason, Actor: actor.ID, At: now})
	st.view.Pending, st.requester = nil, ""
	return c.snapshot(), nil
}

func (r *Registry) ApprovePostponement(workspace, id, stageName string, actor Actor) (CycleView, error) {
	return r.decidePostponement(workspace, id, stageName, actor, true)
}

func (r *Registry) WithdrawPostponement(workspace, id, stageName string, actor Actor) (CycleView, error) {
	return r.decidePostponement(workspace, id, stageName, actor, false)
}

var steps = []string{StepDueSoon, StepDue, StepOverdue}

type pick struct {
	due    time.Time
	at     time.Time
	cycle  string
	stage  *stage
	step   int
	target *item
}

func (r *Registry) Claim(workspace string, limit int, actor Actor) ([]Event, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || limit < 1 || limit > r.limits.Claims || !actor.valid() {
		return nil, ErrInvalid
	}
	if err := actor.notAssistant(); err != nil {
		return nil, err
	}
	if actor.Kind != KindSystem {
		return nil, refuse(RefusalNotSystem)
	}
	s := r.workspaces[workspace]
	if s == nil {
		return nil, nil
	}
	var picks []pick
	for id, c := range s.cycles {
		for _, st := range c.stages {
			if len(st.owed) == 0 {
				continue
			}
			due := st.view.CurrentDue
			times := []time.Time{due.Add(-r.limits.Lead), due, due.Add(overdueAfter)}
			step := -1
			for index, at := range times {
				if !now.Before(at) {
					step = index
				}
			}
			if step < 0 {
				continue
			}
			for it := range st.owed {
				if it.claimed[due] <= step {
					picks = append(picks, pick{due: due, at: times[step], cycle: id, stage: st, step: step, target: it})
				}
			}
		}
	}
	sort.Slice(picks, func(i, j int) bool {
		a, b := picks[i], picks[j]
		switch {
		case !a.due.Equal(b.due):
			return a.due.Before(b.due)
		case a.cycle != b.cycle:
			return a.cycle < b.cycle
		case a.stage.index != b.stage.index:
			return a.stage.index < b.stage.index
		}
		return a.target.target < b.target.target
	})
	if len(picks) > limit {
		picks = picks[:limit]
	}
	var out []Event
	for _, p := range picks {
		skipped := p.target.claimed[p.due] < p.step
		p.target.claimed[p.due] = p.step + 1
		p.stage.sync(p.target)
		last := len(out) - 1
		if last < 0 || out[last].Cycle != p.cycle || out[last].Stage != p.stage.view.Spec.Stage || out[last].Step != steps[p.step] {
			out = append(out, Event{Cycle: p.cycle, Stage: p.stage.view.Spec.Stage, Step: steps[p.step], Due: p.due, At: p.at})
			last++
		}
		out[last].Targets = append(out[last].Targets, p.target.target)
		if skipped {
			out[last].Skipped = append(out[last].Skipped, p.target.target)
		}
	}
	return out, nil
}
