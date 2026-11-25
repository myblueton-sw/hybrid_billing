package server

import (
	"net/http"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

const ownersShown = 20

type retentionOwnersPage struct {
	Revision   string       `json:"revision"`
	ServerTime string       `json:"server_time"`
	Order      string       `json:"order"`
	Q          string       `json:"q"`
	Owners     []subjectRow `json:"owners"`
	Matching   int          `json:"matching"`
	More       bool         `json:"more"`
}

func (s *Server) listRetentionOwners(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	values := r.URL.Query()
	limit, term, id, valid := ownersShown, "", "", true
	for name, given := range values {
		switch {
		case len(given) != 1 || given[0] == "":
			valid = false
		case name == "id":
			id, valid = given[0], valid && len(values) == 1 && validWorkspace(given[0])
		case name == "q":
			var fits bool
			term, fits = parseSearch(given[0])
			valid = valid && fits
		case name == "limit":
			n, err := strconv.Atoi(given[0])
			limit, valid = n, valid && err == nil && n >= 1 && n <= ownersShown && strconv.Itoa(n) == given[0]
		default:
			valid = false
		}
	}
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	page := retentionOwnersPage{Revision: s.Revision(), ServerTime: stamp(s.clock()), Order: retentionOrder, Q: term, Owners: []subjectRow{}}
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	if s.faulty("retention-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	candidates, err := s.grantees(found.verifier)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	q := keptQuery{term: term}
	for _, c := range candidates {
		if id != "" && c.UserID != id || id == "" && !q.matches(c.UserID, c.Login) {
			continue
		}
		if page.Matching++; len(page.Owners) < limit {
			page.Owners = append(page.Owners, c)
		}
	}
	page.More = page.Matching > len(page.Owners)
	writeJSON(w, http.StatusOK, request, page)
}
