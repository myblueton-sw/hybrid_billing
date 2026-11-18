package server

import (
	"errors"
	"net/http"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type targetRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MembersOn int    `json:"members_on"`
}

type targetsPage struct {
	Revision     string      `json:"revision"`
	Relationship uint64      `json:"relationship_revision"`
	Active       uint64      `json:"active_revision"`
	On           string      `json:"on"`
	Kind         string      `json:"kind"`
	Targets      []targetRow `json:"targets"`
	Next         string      `json:"next"`
	ScanStopped  bool        `json:"scan_stopped"`
}

func (s *Server) targetsQuery(r *http.Request) (orgQuery, bool) {
	q, ok := s.orgQuery(r)
	values := r.URL.Query()
	for _, name := range []string{"subject", "entity", "unattributed", "member", "target"} {
		ok = ok && !values.Has(name)
	}
	return q, ok && values.Get("kind") != ""
}

func (s *Server) targetsPage(ws string, number uint64, q orgQuery, budget int) ([]targetRow, string, bool, error) {
	size := RelationshipLimits().PageSize
	rows := []targetRow{}
	cursor, examined := q.after, 0
	for {
		ids, err := s.rels.Targets(ws, number, q.kind, cursor, size)
		if err != nil {
			return nil, "", false, err
		}
		for _, id := range ids {
			if examined == budget {
				return rows, cursor, true, nil
			}
			examined++
			cursor = id
			if !matchesSearch(id, q.filter.q) {
				continue
			}
			if len(rows) == q.limit {
				return rows, rows[len(rows)-1].ID, false, nil
			}
			members, err := s.rels.MembersOn(ws, number, q.kind, id, q.on)
			if err != nil {
				return nil, "", false, err
			}
			rows = append(rows, targetRow{ID: id, Name: orgName(id), MembersOn: len(members)})
		}
		if len(ids) < size {
			return rows, "", false, nil
		}
	}
}

func (s *Server) readTargets(ws string, q orgQuery) (targetsPage, error) {
	if s.faulty("organizations") {
		return targetsPage{}, errors.New("organization targets fault")
	}
	active, err := s.rels.Active(ws)
	if err != nil {
		return targetsPage{}, err
	}
	number := active
	if q.pinned {
		if q.revision > active {
			return targetsPage{}, errInvalidQuery
		}
		number = q.revision
	}
	rows, next, stopped, err := s.targetsPage(ws, number, q, s.cfg.Org.FilterScan)
	if err != nil {
		return targetsPage{}, err
	}
	return targetsPage{Revision: s.Revision(), Relationship: number, Active: active, On: q.on.ISO(), Kind: string(q.kind), Targets: rows, Next: next, ScanStopped: stopped}, nil
}

func (s *Server) listTargets(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	if r.URL.Query().Has("provider") {
		failModule(w, commands.UnsupportedCapability, http.StatusNotImplemented, false, commands.NextNone, request)
		return
	}
	q, valid := s.targetsQuery(r)
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	page, err := s.readTargets(answer.Workspace(), q)
	switch {
	case errors.Is(err, errInvalidQuery):
		failCode(w, commands.InvalidRequest, request)
	case err != nil:
		failCode(w, commands.DependencyUnavailable, request)
	default:
		writeJSON(w, http.StatusOK, request, page)
	}
}
