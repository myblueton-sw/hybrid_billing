package overview

import (
	"fmt"
	"strconv"
	"time"
)

type point struct {
	seq       int
	evaluated time.Time
	token     string
}

func (r *Registry) asOf(workspace, token string, s *space, now time.Time) (point, error) {
	if token == "" {
		seq := 0
		if s != nil {
			seq = len(s.order)
		}
		sealed, err := r.seal(asOfLabel, digest(workspace), fmt.Sprintf("%020d", seq), fmt.Sprintf("%020d", now.UnixNano()))
		if err != nil {
			return point{}, err
		}
		return point{seq: seq, evaluated: now, token: sealed}, nil
	}
	parts, ok := r.unseal(asOfLabel, token)
	if !ok || len(parts) != 3 || parts[0] != digest(workspace) {
		return point{}, ErrInvalid
	}
	seq, _ := strconv.Atoi(parts[1])
	nanos, _ := strconv.ParseInt(parts[2], 10, 64)
	evaluated := time.Unix(0, nanos).UTC()
	return point{seq: seq, evaluated: evaluated, token: token}, nil
}

func (r *Registry) resolve(workspace string, s *space, q Query, actor Actor) (Filter, Sort, error) {
	if q.FilterID == "" {
		return q.Filter, q.Sort, nil
	}
	if !empty(q.Filter) || q.Sort != (Sort{}) {
		return Filter{}, Sort{}, ErrInvalid
	}
	if s == nil {
		return Filter{}, Sort{}, ErrNotFound
	}
	saved := s.saved[[2]string{q.FilterOwner, q.FilterID}]
	if saved == nil || !(saved.Owner == actor.ID && actor.Kind == KindHuman) && !saved.Saved.Shared {
		return Filter{}, Sort{}, ErrNotFound
	}
	return saved.Saved.Filter, saved.Saved.Sort, nil
}

func (r *Registry) Query(workspace string, q Query, actor Actor) (Page, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || q.Limit < 1 || q.Limit > r.limits.Page || !filterForm(q.Filter) || !sortKeys[q.Sort.Key] ||
		(q.FilterID == "") != (q.FilterOwner == "") || q.FilterID != "" && (!validID(q.FilterID) || !validID(q.FilterOwner)) {
		return Page{}, ErrInvalid
	}
	if err := actor.reader(); err != nil {
		return Page{}, err
	}
	vis, err := r.visibility(workspace, actor)
	if err != nil {
		return Page{}, err
	}
	s := r.workspaces[workspace]
	f, o, err := r.resolve(workspace, s, q, actor)
	if err != nil {
		return Page{}, err
	}
	token, offset := q.AsOf, 0
	if q.Cursor != "" {
		parts, ok := r.unseal(cursorLabel, q.Cursor)
		if !ok || len(parts) != 6 || q.AsOf != "" && q.AsOf != parts[1] || parts[0] != digest(workspace) || parts[2] != actor.Kind+":"+actor.ID || parts[3] != visibilityDigest(vis) || parts[4] != filterDigest(f, o) {
			return Page{}, refuse(ReasonCursorMismatch)
		}
		token = parts[1]
		offset, _ = strconv.Atoi(parts[5])
	}
	at, err := r.asOf(workspace, token, s, now)
	if err != nil {
		return Page{}, err
	}
	var rows []*Row
	if s != nil {
		for _, sub := range s.subjects {
			if !visible(vis, sub.key) {
				continue
			}
			row, present := r.state(s, sub, at.seq, at.evaluated, vis.Cost)
			if present && matches(f, row) {
				rows = append(rows, &row)
			}
		}
	}
	order(rows, o)
	page := Page{AsOf: at.token, Evaluated: at.evaluated, Totals: totals(rows, vis.Cost), Count: len(rows)}
	for _, row := range rows {
		if row.Status == StatusStale {
			page.Stale++
		}
	}
	end := offset + q.Limit
	if end > len(rows) {
		end = len(rows)
	}
	if offset < len(rows) {
		for _, row := range rows[offset:end] {
			page.Rows = append(page.Rows, copyRow(*row))
		}
	}
	if end < len(rows) {
		if page.Next, err = r.seal(cursorLabel, digest(workspace), at.token, actor.Kind+":"+actor.ID, visibilityDigest(vis), filterDigest(f, o), fmt.Sprintf("%020d", end)); err != nil {
			return Page{}, err
		}
	}
	return page, nil
}

func copyRow(row Row) Row {
	amounts := make(map[string]string, len(row.Amounts))
	for k, v := range row.Amounts {
		amounts[k] = v
	}
	row.Amounts = amounts
	return row
}

func (r *Registry) Detail(workspace string, key Subject, token string, actor Actor) (Detail, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !subjectForm(key) || !actor.valid() {
		return Detail{}, ErrInvalid
	}
	if err := actor.reader(); err != nil {
		return Detail{}, err
	}
	vis, err := r.visibility(workspace, actor)
	if err != nil {
		return Detail{}, err
	}
	s := r.workspaces[workspace]
	at, err := r.asOf(workspace, token, s, now)
	if err != nil {
		return Detail{}, err
	}
	if s == nil || s.subjects[key] == nil || !visible(vis, key) {
		return Detail{}, ErrNotFound
	}
	sub := s.subjects[key]
	row, _ := r.state(s, sub, at.seq, at.evaluated, vis.Cost)
	out := Detail{AsOf: at.token, Evaluated: at.evaluated, Row: copyRow(row)}
	for _, e := range sub.entries {
		if e.seq > at.seq {
			break
		}
		v := view(e, s.current(e.fact.Source.ID, at.seq) != e)
		if !vis.Cost {
			delete(v.Fact.Amounts, MeaningCost)
		}
		out.Timeline = append(out.Timeline, v)
	}
	if len(out.Timeline) == 0 {
		return Detail{}, ErrNotFound
	}
	return out, nil
}

func sharedForm(f Filter) bool {
	return len(f.Customers)+len(f.Contracts)+len(f.Entities)+len(f.Owners) == 0
}

func (r *Registry) Save(workspace string, v Saved, actor Actor) (SavedView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(v.ID) || !actor.valid() || !filterForm(v.Filter) || !sortKeys[v.Sort.Key] {
		return SavedView{}, ErrInvalid
	}
	switch actor.Kind {
	case KindAssistant:
		return SavedView{}, refuse(ReasonAINotAllowed)
	case KindSystem:
		return SavedView{}, refuse(ReasonNotHuman)
	}
	if v.Shared && !sharedForm(v.Filter) {
		return SavedView{}, refuse(ReasonSharedIDs)
	}
	s, err := r.open(workspace)
	if err != nil {
		return SavedView{}, err
	}
	key := [2]string{actor.ID, v.ID}
	if s.saved[key] == nil && len(s.saved) >= r.limits.Filters {
		return SavedView{}, ErrLimit
	}
	v.Filter = copyFilter(v.Filter)
	s.saved[key] = &SavedView{Saved: v, Owner: actor.ID, At: now}
	return viewSaved(s.saved[key]), nil
}

func copyFilter(f Filter) Filter {
	f.Customers = append([]string(nil), f.Customers...)
	f.Contracts = append([]string(nil), f.Contracts...)
	f.Entities = append([]string(nil), f.Entities...)
	f.Periods = append([]string(nil), f.Periods...)
	f.Currencies = append([]string(nil), f.Currencies...)
	f.Stages = append([]string(nil), f.Stages...)
	f.Owners = append([]string(nil), f.Owners...)
	return f
}

func viewSaved(v *SavedView) SavedView {
	out := *v
	out.Saved.Filter = copyFilter(v.Saved.Filter)
	return out
}

func (r *Registry) Filters(workspace string, actor Actor) ([]SavedView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() {
		return nil, ErrInvalid
	}
	if err := actor.reader(); err != nil {
		return nil, err
	}
	var out []SavedView
	if s := r.workspaces[workspace]; s != nil {
		for _, v := range s.saved {
			if v.Owner == actor.ID && actor.Kind == KindHuman || v.Saved.Shared {
				out = append(out, viewSaved(v))
			}
		}
	}
	sortSaved(out)
	return out, nil
}
