package server

import (
	"net/http"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type evidenceReplacementsPage struct {
	Revision     string           `json:"revision"`
	ServerTime   string           `json:"server_time"`
	Requirement  string           `json:"requirement"`
	Order        string           `json:"order"`
	Q            string           `json:"q"`
	Replacements []evidenceTarget `json:"replacements"`
	Matching     int              `json:"matching"`
	More         bool             `json:"more"`
}

func (s *Server) listEvidenceReplacements(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	values := r.URL.Query()
	limit, term, wanted, valid := evidenceReplacements, "", "", true
	_, byID := values["id"]
	for name, given := range values {
		switch {
		case len(given) != 1 || given[0] == "":
			valid = false
		case name == "requirement":
		case name == "id":
			wanted, valid = given[0], valid && len(values) == 2
		case name == "q":
			var fits bool
			term, fits = parseSearch(given[0])
			valid = valid && fits && !byID
		case name == "limit":
			n, err := strconv.Atoi(given[0])
			limit, valid = n, valid && !byID && err == nil && n >= 1 && n <= evidenceReplacements && strconv.Itoa(n) == given[0]
		default:
			valid = false
		}
	}
	target := values.Get("requirement")
	raw, found := cutRequirementTarget(target)
	index, numbered := canonicalNumber(raw)
	if !valid || !found || !numbered || answer.Workspace() != s.bootWorkspace || index == 0 || index > uint64(len(s.ev.requirements)) {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	if s.faulty("release-evidence-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	f := s.evidenceFactsWithoutRoster(answer.User())
	numbers, err := f.replacementNumbers(int(index) - 1)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page := evidenceReplacementsPage{Revision: s.Revision(), ServerTime: stamp(s.clock()), Requirement: target, Order: requirementsOrder, Q: term, Replacements: []evidenceTarget{}}
	q := keptQuery{term: term}
	for _, number := range numbers {
		row := f.replacementAt(number)
		if byID && row.Target != wanted || !byID && !q.matches(row.Label, row.Target) {
			continue
		}
		if page.Matching++; len(page.Replacements) < limit {
			page.Replacements = append(page.Replacements, row)
		}
	}
	page.More = page.Matching > len(page.Replacements)
	writeJSON(w, http.StatusOK, request, page)
}
