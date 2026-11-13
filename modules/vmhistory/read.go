package vmhistory

import (
	"sort"
	"time"
)

func (r *Registry) Assets(workspaceID, name, after string, limit int) (AssetPage, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || (after != "" && !validID(after)) || limit < 1 || limit > r.limits.Page {
		return AssetPage{}, ErrInvalid
	}
	home := r.origin(workspaceID, name)
	if home == nil {
		return AssetPage{}, ErrNotFound
	}
	ids := make([]string, 0, len(home.assets))
	for id := range home.assets {
		if id > after {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := AssetPage{Sequence: home.sequence, More: len(ids) > limit}
	if out.More {
		ids = ids[:limit]
	}
	for _, id := range ids {
		out.Items = append(out.Items, home.assets[id].spec)
	}
	return out, nil
}

func (r *Registry) owner(workspaceID, name, id string, positioned bool, limit int) (*source, *asset, error) {
	if !r.enter() || !validID(workspaceID) || !validID(name) || !validID(id) || !positioned || limit < 1 || limit > r.limits.Page {
		return nil, nil, ErrInvalid
	}
	home := r.origin(workspaceID, name)
	if home == nil || home.assets[id] == nil {
		return nil, nil, ErrNotFound
	}
	return home, home.assets[id], nil
}

func (r *Registry) Spans(workspaceID, name, id string, after time.Time, limit int) (SpanPage, error) {
	defer r.leave()
	home, owner, err := r.owner(workspaceID, name, id, after.IsZero() || instant(after), limit)
	if err != nil {
		return SpanPage{}, err
	}
	first := 0
	if !after.IsZero() {
		first = sort.Search(len(owner.live), func(index int) bool { return owner.live[index].spec.From.After(after) })
	}
	out := SpanPage{Sequence: home.sequence, Change: owner.change}
	for _, held := range owner.live[first:] {
		if len(out.Items) == limit {
			out.More = true
			break
		}
		out.Items = append(out.Items, held.spec)
	}
	return out, nil
}

func (r *Registry) Counters(workspaceID, name, id, after string, limit int) (CounterPage, error) {
	defer r.leave()
	home, owner, err := r.owner(workspaceID, name, id, after == "" || validID(after), limit)
	if err != nil {
		return CounterPage{}, err
	}
	first := sort.Search(len(owner.counters), func(index int) bool { return owner.counters[index].spec.ID > after })
	out := CounterPage{Sequence: home.sequence, Change: owner.change}
	for _, meter := range owner.counters[first:] {
		if len(out.Items) == limit {
			out.More = true
			break
		}
		out.Items = append(out.Items, CounterView{Counter: meter.spec, State: meter.state})
	}
	return out, nil
}

func (r *Registry) Readings(workspaceID, name, id string, after time.Time, limit int) (ReadingPage, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !validID(id) || (!after.IsZero() && !instant(after)) || limit < 1 || limit > r.limits.Page {
		return ReadingPage{}, ErrInvalid
	}
	home := r.origin(workspaceID, name)
	if home == nil || home.counters[id] == nil {
		return ReadingPage{}, ErrNotFound
	}
	meter := home.counters[id]
	if meter.state != Live {
		return ReadingPage{}, ErrRejected
	}
	first := 0
	if !after.IsZero() {
		first = sort.Search(len(meter.live), func(index int) bool { return meter.live[index].spec.At.After(after) })
	}
	out := ReadingPage{Sequence: home.sequence, Change: meter.owner.change}
	for _, held := range meter.live[first:] {
		if len(out.Items) == limit {
			out.More = true
			break
		}
		out.Items = append(out.Items, held.spec)
	}
	return out, nil
}
