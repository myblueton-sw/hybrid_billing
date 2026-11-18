package bulkrevoke

import "sort"

func count(v View) Counts {
	c := Counts{Duplicates: v.Duplicates, Targets: v.ReplayHeld + v.ReplayMissing, Held: v.ReplayHeld, Unprocessed: v.ReplayMissing, Excluded: v.ReplayExcluded}
	for _, item := range v.Items {
		switch item.Outcome {
		case OutcomeExcluded:
			c.Excluded++
			continue
		case OutcomeHeld:
			c.Held++
		case OutcomeSucceeded:
			c.Succeeded++
		case OutcomeFailed:
			c.Failed++
		default:
			c.Unprocessed++
		}
		c.Targets++
	}
	return c
}

func (r *Registry) executed(workspace, id string) *entry {
	e := r.find(workspace, id)
	if e == nil || e.v.State != Executed {
		return nil
	}
	return e
}

func (r *Registry) update(workspace, id string, change func(*View) bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.executed(workspace, id)
	if e == nil {
		return ErrNotFound
	}
	if !change(&e.v) {
		return ErrState
	}
	e.v.Counts = count(e.v)
	return nil
}

func pending(item Item) bool { return item.Outcome == OutcomeUnprocessed }

func (r *Registry) Header(workspace, id string, receipt Receipt) error {
	return r.update(workspace, id, func(v *View) bool {
		if v.Headed {
			return false
		}
		v.Header, v.Headed = receipt, true
		if receipt.Reason != "" {
			for i := range v.Items {
				if pending(v.Items[i]) {
					v.Items[i].State, v.Items[i].Receipt = Unconfirmed, Receipt{Reason: receipt.Reason}
				}
			}
		}
		return true
	})
}

func (r *Registry) Record(workspace, id string, indexes []int, receipts []Receipt) error {
	return r.update(workspace, id, func(v *View) bool {
		if !v.Headed || v.Header.Reason != "" || len(indexes) != len(receipts) {
			return false
		}
		for _, i := range indexes {
			if i < 0 || i >= len(v.Items) || !pending(v.Items[i]) || v.Items[i].State != Received {
				return false
			}
		}
		for n, i := range indexes {
			v.Items[i].Receipt = receipts[n]
			v.Items[i].State = DenyConfirmed
			if receipts[n].Reason != "" || receipts[n].Sequence <= v.Header.Sequence {
				v.Items[i].State = Unconfirmed
				if receipts[n].Reason == "" {
					v.Items[i].Receipt = Receipt{Reason: "commit_failed"}
				}
			}
		}
		return true
	})
}

func (r *Registry) Cleanup(workspace, id string, created bool) error {
	return r.update(workspace, id, func(v *View) bool {
		v.CleanupCreated, v.CleanupNotReady = created, !created
		return true
	})
}

func (r *Registry) Cleaning(workspace, id string, index int) error {
	return r.update(workspace, id, func(v *View) bool {
		if index < 0 || index >= len(v.Items) || !pending(v.Items[index]) || v.Items[index].State != DenyConfirmed && v.Items[index].State != CleanupRunning {
			return false
		}
		v.Items[index].State = CleanupRunning
		return true
	})
}

func (r *Registry) Cleaned(workspace, id string, index int, cleanup []Cleanup) error {
	return r.update(workspace, id, func(v *View) bool {
		if index < 0 || index >= len(v.Items) || !pending(v.Items[index]) || v.Items[index].State != CleanupRunning {
			return false
		}
		v.Items[index].State, v.Items[index].Outcome, v.Items[index].Cleanup = Complete, OutcomeSucceeded, append([]Cleanup(nil), cleanup...)
		return true
	})
}

func (r *Registry) Failed(workspace, id string, index int, reason string) error {
	return r.update(workspace, id, func(v *View) bool {
		if index < 0 || index >= len(v.Items) || !pending(v.Items[index]) || v.Items[index].State != DenyConfirmed && v.Items[index].State != CleanupRunning {
			return false
		}
		v.Items[index].State, v.Items[index].Outcome, v.Items[index].Reason = DenyConfirmed, OutcomeFailed, reason
		return true
	})
}

func JobState(v View, cleaning Cleaning) (string, string) {
	if v.ReplayMissing > 0 {
		return JobWaiting, ReasonUnconfirmed
	}
	for _, item := range v.Items {
		if item.State == Unconfirmed {
			return JobWaiting, ReasonUnconfirmed
		}
	}
	if v.CleanupNotReady {
		return JobWaiting, ReasonNotStarted
	}
	switch cleaning.State {
	case CleaningQueued, CleaningRunning:
		return JobRunning, ""
	case CleaningWaiting:
		return JobWaiting, cleaning.Reason
	case CleaningSucceeded, CleaningFailed:
		return JobSucceeded, ""
	}
	if unfinished(v) {
		return JobRunning, ""
	}
	return JobSucceeded, ""
}

func (r *Registry) Items(workspace, caller, id string, after, limit int, outcome Outcome) ([]Item, int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.visible(workspace, caller, id)
	if e == nil || after < 0 || limit < 1 {
		return nil, 0, false
	}
	matched := []Item{}
	for _, item := range e.v.Items {
		if outcome == "" || item.Outcome == outcome {
			matched = append(matched, item)
		}
	}
	start := min(after, len(matched))
	end := min(start+limit, len(matched))
	out := make([]Item, 0, end-start)
	for _, item := range matched[start:end] {
		item.Cleanup = append([]Cleanup(nil), item.Cleanup...)
		out = append(out, item)
	}
	return out, len(matched), true
}

type Latest struct {
	Bulk  string
	State string
}

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
