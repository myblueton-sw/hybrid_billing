package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

const (
	pathIDs     = 20
	pathIDBytes = 128
)

type evidencePathsPage struct {
	Revision   string            `json:"revision"`
	ServerTime string            `json:"server_time"`
	Order      string            `json:"order"`
	Q          string            `json:"q"`
	Paths      []evidencePathRow `json:"paths"`
	Next       string            `json:"next"`
	Matching   int               `json:"matching"`
	Total      int               `json:"total"`
}

func (f *evidenceFacts) pathAt(index int) (evidencePathRow, error) {
	row, err := f.pathRow(f.s.ev.paths[index])
	row.Number = index + 1
	return row, err
}

func pathIDsOf(raw string) (map[string]bool, bool) {
	out := map[string]bool{}
	for _, id := range strings.Split(raw, ",") {
		if id == "" || len(id) > pathIDBytes || out[id] || len(out) == pathIDs {
			return nil, false
		}
		out[id] = true
	}
	return out, true
}

func (s *Server) listEvidencePaths(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	values := r.URL.Query()
	named, byID := values["id"]
	var q keptQuery
	var wanted map[string]bool
	valid := true
	if byID {
		valid = len(values) == 1 && len(named) == 1
		if valid {
			wanted, valid = pathIDsOf(named[0])
		}
	} else if q, valid = parseKeptQuery(values); valid {
		valid = q.limit <= evidenceShown
	}
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	now := s.clock()
	page := evidencePathsPage{Revision: s.Revision(), ServerTime: stamp(now), Order: requirementsOrder, Q: q.term, Paths: []evidencePathRow{}}
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	if s.faulty("release-evidence-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	f, err := s.evidenceFactsFor(found.verifier, answer.User(), now)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page.Total = len(s.ev.paths)
	last := 0
	for index, id := range s.ev.paths {
		if byID && !wanted[id] || !byID && !q.matches(id) {
			continue
		}
		page.Matching++
		if !byID {
			if index < int(q.before) {
				continue
			}
			if len(page.Paths) == q.limit {
				page.Next = strconv.Itoa(last)
				continue
			}
		}
		row, err := f.pathAt(index)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		page.Paths, last = append(page.Paths, row), index+1
	}
	writeJSON(w, http.StatusOK, request, page)
}
