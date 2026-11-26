package trustchange

import "strings"

const MaxPage = 100

type Query struct {
	Workspace string
	States    []string
	Match     string
	After     string
	Limit     int
}

type Page struct {
	Views   []View
	More    bool
	Total   int
	Version uint64
	Known   bool
}

type Summary struct {
	Unknown    string
	EarlierRun bool
	Version    uint64
}

func (q Query) admits(v *View) bool {
	if v.Workspace != q.Workspace || !strings.Contains(v.ID, q.Match) {
		return false
	}
	if len(q.States) == 0 {
		return true
	}
	for _, state := range q.States {
		if v.State == state {
			return true
		}
	}
	return false
}

func (r *Registry) Version() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.version
}

func (r *Registry) Page(q Query) Page {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := Page{Views: []View{}, Version: r.version, Known: q.After == ""}
	if q.Limit < 1 || q.Limit > MaxPage {
		return Page{Views: []View{}, Version: r.version}
	}
	if anchor := r.changes[q.After]; q.After != "" && (anchor == nil || anchor.Workspace != q.Workspace) {
		return out
	}
	passed := q.After == ""
	for i := len(r.order) - 1; i >= 0; i-- {
		id := r.order[i]
		v := r.changes[id]
		if q.admits(v) {
			out.Total++
			switch {
			case !passed:
			case len(out.Views) < q.Limit:
				out.Views = append(out.Views, copyView(*v))
			default:
				out.More = true
			}
		}
		if id == q.After {
			passed = true
		}
	}
	out.Known = true
	return out
}

func (r *Registry) Summary(workspace string) Summary {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := Summary{Version: r.version}
	for i := len(r.order) - 1; i >= 0; i-- {
		v := r.changes[r.order[i]]
		if v.Workspace != workspace {
			continue
		}
		if out.Unknown == "" && v.State == StateUnknown {
			out.Unknown = v.ID
		}
		out.EarlierRun = out.EarlierRun || v.EarlierRun
	}
	return out
}
