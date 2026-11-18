package bulkrevoke

import "sort"

type latestRef struct {
	number uint64
	index  int
}

func indexed(item Item) bool {
	return item.Outcome != OutcomeExcluded && item.Outcome != OutcomeHeld && item.Target.User != ""
}

func at(list []latestRef, number uint64) int {
	return sort.Search(len(list), func(i int) bool { return list[i].number <= number })
}

func (r *Registry) index(e *entry) {
	people := r.refs[e.v.Workspace]
	if people == nil {
		people = map[string][]latestRef{}
		r.refs[e.v.Workspace] = people
	}
	for i, item := range e.v.Items {
		r.visited++
		if !indexed(item) {
			continue
		}
		list := people[item.Target.User]
		pos := at(list, e.v.Number)
		if pos < len(list) && list[pos].number == e.v.Number {
			continue
		}
		list = append(list, latestRef{})
		copy(list[pos+1:], list[pos:])
		list[pos] = latestRef{number: e.v.Number, index: i}
		people[item.Target.User] = list
	}
}

func (r *Registry) unindex(e *entry) {
	people := r.refs[e.v.Workspace]
	for _, item := range e.v.Items {
		r.visited++
		list := people[item.Target.User]
		pos := at(list, e.v.Number)
		if !indexed(item) || pos == len(list) || list[pos].number != e.v.Number {
			continue
		}
		list = append(list[:pos], list[pos+1:]...)
		if len(list) == 0 {
			delete(people, item.Target.User)
			continue
		}
		people[item.Target.User] = list
	}
}

func (r *Registry) LatestFor(workspace string, ids []string) map[string]Latest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]Latest{}
	people := r.refs[workspace]
	for _, id := range ids {
		if list := people[id]; len(list) > 0 {
			e := r.entries[list[0].number]
			out[id] = Latest{Bulk: e.v.ID, State: e.v.Items[list[0].index].State}
		}
	}
	return out
}
