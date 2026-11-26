package accesschange

import (
	"sort"
	"time"
)

type Counts struct {
	Pending     int
	Ended       int
	Revocations int
}

func (r *Registry) Version() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.version
}

func (r *Registry) RequestPage(workspace string, before uint64, limit int, at time.Time) ([]View, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit < 1 {
		return nil, false
	}
	numbers := r.numbers()
	end := len(numbers)
	if before != 0 {
		end = sort.Search(len(numbers), func(i int) bool { return numbers[i] >= before })
	}
	out := []View{}
	for i := end - 1; i >= 0; i-- {
		q := r.requests[numbers[i]]
		if q.view.Workspace != workspace {
			continue
		}
		if len(out) == limit {
			return out, true
		}
		out = append(out, r.shown(q, at.UTC()))
	}
	return out, false
}

func (r *Registry) RevocationPage(workspace, before string, limit int) ([]RevocationView, bool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit < 1 {
		return nil, false, false
	}
	end := len(r.revocations)
	if before != "" {
		end = -1
		for i, rec := range r.revocations {
			if rec.view.ID == before && rec.view.Workspace == workspace {
				end = i
				break
			}
		}
		if end < 0 {
			return nil, false, false
		}
	}
	out := []RevocationView{}
	for i := end - 1; i >= 0; i-- {
		v := r.revocations[i].view
		if v.Workspace != workspace {
			continue
		}
		if len(out) == limit {
			return out, true, true
		}
		out = append(out, copyRevocation(v))
	}
	return out, false, true
}

func (r *Registry) UnconfirmedRevocations(workspace string, limit int) ([]RevocationView, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit < 1 {
		return nil, false
	}
	out := []RevocationView{}
	for _, rec := range r.revocations {
		v := rec.view
		if v.Workspace != workspace || v.State != RevocationReceived && v.State != RevocationUnconfirmed {
			continue
		}
		if len(out) == limit {
			return out, true
		}
		out = append(out, copyRevocation(v))
	}
	return out, false
}

func (r *Registry) Counts(workspace string, at time.Time) Counts {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out Counts
	for _, q := range r.requests {
		if q.view.Workspace != workspace {
			continue
		}
		if q.view.State == StatePending && !r.lapsed(q, at.UTC()) {
			out.Pending++
		} else {
			out.Ended++
		}
	}
	for _, rec := range r.revocations {
		if rec.view.Workspace == workspace {
			out.Revocations++
		}
	}
	return out
}
