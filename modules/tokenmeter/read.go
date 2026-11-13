package tokenmeter

import (
	"sort"
	"time"
)

func (r *Registry) listed(items []string) Listed {
	sort.Strings(items)
	shown := items
	if len(shown) > r.limits.Listed {
		shown = shown[:r.limits.Listed]
	}
	return Listed{Total: len(items), Items: shown}
}

func (r *Registry) judge(held *reception) Verdict {
	out := Verdict{Result: NotEligible}
	if held.state == Voided {
		out.Reasons = []string{IsVoided}
		return out
	}
	note := func(applies bool, reason string) {
		if applies {
			out.Reasons = append(out.Reasons, reason)
		}
	}
	note(held.state == Open, NotClosed)
	note(held.spec.Produced != SnapshotExport, FromListInterface)
	note(held.spec.Declaration != SourceManifest, DeclarationDerived)
	note(held.imprecise, FloatTotalImprecise)
	var overlapping, sharing []string
	for _, other := range held.origin.receptions {
		if other == held || other.state == Voided {
			continue
		}
		if held.spec.PeriodStart.Before(other.spec.PeriodEnd) && other.spec.PeriodStart.Before(held.spec.PeriodEnd) {
			overlapping = append(overlapping, other.spec.ID)
		}
		if held.shared[other] > 0 {
			sharing = append(sharing, other.spec.ID)
		}
	}
	out.Conflicts, out.Shared = r.listed(overlapping), r.listed(sharing)
	note(out.Conflicts.Total > 0, PeriodConflict)
	note(out.Shared.Total > 0, EventShared)
	if held.state == Closed {
		note(held.figures.InError > 0, EventsInError)
		note(len(held.events) != held.spec.Events, Incomplete)
		note(len(held.tokens) != len(kinds), TotalsNotStated)
		note(held.differ, TotalsDiffer)
		note(held.outside, SumOutsideProfile)
	}
	if len(out.Reasons) == 0 {
		out.Result = Eligible
	}
	return out
}

func (r *Registry) Eligibility(workspaceID, id string) (Verdict, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return Verdict{}, ErrInvalid
	}
	held := r.find(workspaceID, id)
	if held == nil {
		return Verdict{}, ErrNotFound
	}
	return r.judge(held), nil
}

func (r *Registry) origin(workspaceID, name string) *source {
	if ws := r.workspaces[workspaceID]; ws != nil {
		return ws.sources[name]
	}
	return nil
}

func (r *Registry) Counting(workspaceID, name string, from, to time.Time) (Counted, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !r.period(from, to) {
		return Counted{}, ErrInvalid
	}
	origin := r.origin(workspaceID, name)
	if origin == nil {
		return Counted{}, ErrNotFound
	}
	for _, held := range origin.receptions {
		if held.spec.PeriodStart.Equal(from) && held.spec.PeriodEnd.Equal(to) && r.judge(held).Result == Eligible {
			return Counted{Reception: held.spec.ID, Snapshot: held.spec.Snapshot, Sequence: origin.sequence}, nil
		}
	}
	return Counted{Sequence: origin.sequence}, ErrNotFound
}

func (r *Registry) Reception(workspaceID, id string) (View, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return View{}, ErrInvalid
	}
	held := r.find(workspaceID, id)
	if held == nil {
		return View{}, ErrNotFound
	}
	view := View{Spec: held.spec, State: held.state, WasClosed: held.wasClosed, Events: len(held.events), VoidReason: held.reason, Opened: held.opened, Changed: held.changed}
	view.Spec.Tokens, view.Spec.Costs = map[Kind]string{}, map[string]string{}
	for kind, total := range held.spec.Tokens {
		view.Spec.Tokens[kind] = total
	}
	for currency, total := range held.spec.Costs {
		view.Spec.Costs[currency] = total
	}
	if held.figures != nil {
		view.Figures = held.figures.copy()
	}
	return view, nil
}

func (f *Figures) copy() *Figures {
	out := &Figures{InError: f.InError, NetInput: f.NetInput, NetUnknown: f.NetUnknown, Outcomes: map[Outcome]int{}, Tokens: map[Usage]map[Kind]string{}, Unknown: map[Kind]int{}, Costs: map[string]map[string]string{}, Documents: map[string]DocumentSum{}}
	for outcome, count := range f.Outcomes {
		out.Outcomes[outcome] = count
	}
	for kind, count := range f.Unknown {
		out.Unknown[kind] = count
	}
	for usage, byKind := range f.Tokens {
		out.Tokens[usage] = map[Kind]string{}
		for kind, sum := range byKind {
			out.Tokens[usage][kind] = sum
		}
	}
	for class, byCurrency := range f.Costs {
		out.Costs[class] = map[string]string{}
		for currency, sum := range byCurrency {
			out.Costs[class][currency] = sum
		}
	}
	for document, entry := range f.Documents {
		copied := DocumentSum{Events: entry.Events, Sums: map[string]string{}}
		for currency, sum := range entry.Sums {
			copied.Sums[currency] = sum
		}
		out.Documents[document] = copied
	}
	return out
}

func (r *Registry) Events(workspaceID, id string, after, limit int) ([]EventView, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || after < 0 || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	held := r.find(workspaceID, id)
	if held == nil {
		return nil, ErrNotFound
	}
	if held.state == Open {
		return nil, ErrRejected
	}
	if after > len(held.order) {
		after = len(held.order)
	}
	end := len(held.order)
	if limit < end-after {
		end = after + limit
	}
	var out []EventView
	for _, name := range held.order[after:end] {
		made := held.events[name]
		view := EventView{Event: made.event, Findings: append([]string(nil), made.findings...), Class: made.class}
		if made.event.Cost != nil {
			cost := *made.event.Cost
			view.Event.Cost = &cost
		}
		out = append(out, view)
	}
	return out, nil
}

func (r *Registry) Receptions(workspaceID, name, after string, limit int) ([]Row, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || (after != "" && !validID(after)) || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	origin := r.origin(workspaceID, name)
	if origin == nil {
		return nil, nil
	}
	ids := make([]string, 0, len(origin.receptions))
	byID := map[string]*reception{}
	for _, held := range origin.receptions {
		if held.spec.ID > after {
			ids = append(ids, held.spec.ID)
			byID[held.spec.ID] = held
		}
	}
	sort.Strings(ids)
	if len(ids) > limit {
		ids = ids[:limit]
	}
	var out []Row
	for _, id := range ids {
		held := byID[id]
		out = append(out, Row{ID: id, Snapshot: held.spec.Snapshot, PeriodStart: held.spec.PeriodStart, PeriodEnd: held.spec.PeriodEnd, State: held.state})
	}
	return out, nil
}
