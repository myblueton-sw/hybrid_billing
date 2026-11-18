package bulkrevoke

import "sort"

type latestRef struct {
	number uint64
	index  int
}

func (r *Registry) reindex() {
	numbers := []uint64{}
	for n, e := range r.entries {
		if e.v.State == Executed {
			numbers = append(numbers, n)
		}
	}
	sort.Slice(numbers, func(a, b int) bool { return numbers[a] > numbers[b] })
	index := map[string]map[string]latestRef{}
	for _, n := range numbers {
		e := r.entries[n]
		people := index[e.v.Workspace]
		if people == nil {
			people = map[string]latestRef{}
			index[e.v.Workspace] = people
		}
		for i, item := range e.v.Items {
			if _, seen := people[item.Target.User]; seen || item.Outcome == OutcomeExcluded || item.Outcome == OutcomeHeld || item.Target.User == "" {
				continue
			}
			people[item.Target.User] = latestRef{number: n, index: i}
		}
	}
	r.latestIndex = index
}

func (r *Registry) LatestFor(workspace string, ids []string) map[string]Latest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]Latest{}
	people := r.latestIndex[workspace]
	for _, id := range ids {
		if ref, ok := people[id]; ok {
			e := r.entries[ref.number]
			out[id] = Latest{Bulk: e.v.ID, State: e.v.Items[ref.index].State}
		}
	}
	return out
}
