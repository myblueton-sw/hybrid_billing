package server

import (
	"net/http"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

type planMembersPage struct {
	Revision   string       `json:"revision"`
	ServerTime string       `json:"server_time"`
	Plan       string       `json:"plan"`
	Kind       string       `json:"kind"`
	Order      string       `json:"order"`
	Members    []changedRow `json:"members"`
	Next       string       `json:"next"`
	Total      int          `json:"total"`
}

func (s *Server) changedRows(m *planMeta, kind, after string, limit int) ([]changedRow, error) {
	changes, err := s.rels.ChangedMembers(m.workspace, m.id, relationships.Kind(kind), after, limit)
	out := []changedRow{}
	for _, c := range changes {
		cr := changedRow{Kind: kind, Member: c.Member, Name: s.label(m.workspace, c.Member), Differs: []rangeRow{}, Before: append([]string{}, c.Before...), After: append([]string{}, c.After...)}
		for _, iv := range c.Differs {
			rr := rangeRow{From: iv.Start().ISO()}
			if end, bounded := iv.End(); bounded {
				rr.Until = end.ISO()
			}
			cr.Differs = append(cr.Differs, rr)
		}
		out = append(out, cr)
	}
	return out, err
}

func (s *Server) listPlanMembers(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	values := r.URL.Query()
	limit, valid := planMembersShown, true
	for name, given := range values {
		switch {
		case len(given) != 1 || given[0] == "":
			valid = false
		case name == "limit":
			n, err := strconv.Atoi(given[0])
			limit, valid = n, valid && err == nil && n >= 1 && n <= planMembersShown && strconv.Itoa(n) == given[0]
		case name == "after":
			valid = valid && validWorkspace(given[0])
		case name != "plan" && name != "kind":
			valid = false
		}
	}
	kind, known := values.Get("kind"), false
	for _, k := range relationKinds {
		known = known || string(k) == kind
	}
	var m *planMeta
	if valid && known {
		m = s.plans.get(answer.Workspace(), values.Get("plan"))
	}
	if m == nil {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	info := s.planState(m)
	page := planMembersPage{Revision: s.Revision(), ServerTime: stamp(s.clock()), Plan: m.id, Kind: kind, Order: retentionOrder, Members: []changedRow{}, Total: info.Kinds[relationships.Kind(kind)].Changed}
	if info.State == relationships.PlanSealed || info.State == relationships.PlanActivated {
		rows, err := s.changedRows(m, kind, values.Get("after"), limit+1)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		if len(rows) > limit {
			rows = rows[:limit]
			page.Next = rows[limit-1].Member
		}
		page.Members = rows
	}
	writeJSON(w, http.StatusOK, request, page)
}
