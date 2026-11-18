package bulkrevoke

import "sort"

func (r *Registry) Latest(workspace string) map[string]Latest {
	r.mu.Lock()
	defer r.mu.Unlock()
	numbers := []uint64{}
	for n, e := range r.entries {
		if e.v.Workspace == workspace && e.v.State == Executed {
			numbers = append(numbers, n)
		}
	}
	sort.Slice(numbers, func(a, b int) bool { return numbers[a] > numbers[b] })
	out := map[string]Latest{}
	for _, n := range numbers {
		e := r.entries[n]
		for _, item := range e.v.Items {
			if _, seen := out[item.Target.User]; seen || item.Outcome == OutcomeExcluded || item.Outcome == OutcomeHeld || item.Target.User == "" {
				continue
			}
			out[item.Target.User] = Latest{Bulk: e.v.ID, State: item.State}
		}
	}
	return out
}
