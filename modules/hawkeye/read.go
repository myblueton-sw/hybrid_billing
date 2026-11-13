package hawkeye

import (
	"sort"
	"time"
)

func (held *reception) contends() bool {
	return held.state != Voided && !held.spec.Mock && held.statement.supersededBy == ""
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
	note(held.spec.Mock, IsMock)
	note(held.spec.Status != Final, IsProvisional)
	note(held.spec.Tier != Measured, NotMeasured)
	note(held.spec.Coverage < r.coverage, CoverageBelow)
	note(held.spec.Declaration != SourceHeader, DeclarationDerived)
	note(held.statement.supersededBy != "", IsSuperseded)
	if held.contends() {
		var others []string
		for _, other := range held.origin.receptions {
			if other != held && other.contends() && held.spec.PeriodStart.Before(other.spec.PeriodEnd) && other.spec.PeriodStart.Before(held.spec.PeriodEnd) {
				others = append(others, other.spec.ID)
			}
		}
		sort.Strings(others)
		out.Conflicts.Total = len(others)
		if len(others) > r.limits.Listed {
			others = others[:r.limits.Listed]
		}
		out.Conflicts.Items = others
		note(out.Conflicts.Total > 0, PeriodConflict)
	}
	if held.state == Closed {
		note(held.unmeasured, UnmeasuredLines)
		note(len(held.lines) != held.spec.Lines, IncompleteList)
		differ, outside := len(held.sums) != len(held.declared), false
		for currency, sum := range held.sums {
			if stated := held.declared[currency]; stated == nil || !stated.equals(sum) {
				differ = true
			}
			if !sum.fits(r.profile) {
				outside = true
			}
		}
		note(differ, TotalsDiffer)
		note(outside, SumOutsideProfile)
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
			return Counted{Reception: held.spec.ID, Statement: held.spec.Statement, Sequence: origin.sequence}, nil
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
	view := View{Spec: held.spec, State: held.state, WasClosed: held.wasClosed, Lines: len(held.lines), SupersededBy: held.statement.supersededBy, VoidReason: held.reason, Opened: held.opened, Changed: held.changed}
	view.Spec.Totals = map[string]string{}
	for currency, total := range held.spec.Totals {
		view.Spec.Totals[currency] = total
	}
	if held.sums != nil {
		view.Sums = map[string]string{}
		for currency, sum := range held.sums {
			view.Sums[currency] = sum.text()
		}
	}
	return view, nil
}

func (r *Registry) Lines(workspaceID, id string, after, limit int) ([]Line, error) {
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
	if after > len(held.teams) {
		after = len(held.teams)
	}
	end := len(held.teams)
	if limit < end-after {
		end = after + limit
	}
	var out []Line
	for _, team := range held.teams[after:end] {
		out = append(out, held.lines[team].stated)
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
		out = append(out, Row{ID: id, Statement: held.spec.Statement, PeriodStart: held.spec.PeriodStart, PeriodEnd: held.spec.PeriodEnd, State: held.state, Mock: held.spec.Mock, Superseded: held.statement.supersededBy != ""})
	}
	return out, nil
}
