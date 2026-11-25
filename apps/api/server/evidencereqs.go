package server

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
)

const (
	requirementsPage     = 50
	requirementTargets   = 20
	requirementNumberMax = 1000000000
	requirementsOrder    = "recorded"
	requirementsRetired  = "yes"
	requirementsCurrent  = "no"
)

var requirementStates = []string{releasetrace.StateDesigned, releasetrace.StateImplemented, releasetrace.StateFailing, releasetrace.StateTested, releasetrace.StateCustomerProven}

type requirementsQuery struct {
	after   int
	limit   int
	search  keptQuery
	state   string
	retired string
	targets []int
	byName  bool
}

type requirementsAnswer struct {
	Revision     string                   `json:"revision"`
	ServerTime   string                   `json:"server_time"`
	Order        string                   `json:"order"`
	Q            string                   `json:"q"`
	State        string                   `json:"state"`
	Retired      string                   `json:"retired"`
	Requirements []evidenceRequirementRow `json:"requirements"`
	Next         string                   `json:"next"`
	Matching     int                      `json:"matching"`
	Total        int                      `json:"total"`
}

func parseRequirementTargets(raw string) ([]int, bool) {
	out, seen := []int{}, map[int]bool{}
	for _, part := range strings.Split(raw, ",") {
		number, found := strings.CutPrefix(part, traceRequirementTarget)
		n, ok := canonicalNumber(number)
		if !found || !ok || n == 0 || n > requirementNumberMax || seen[int(n)] || len(out) == requirementTargets {
			return nil, false
		}
		out, seen[int(n)] = append(out, int(n)), true
	}
	return out, true
}

func parseRequirementsQuery(values url.Values) (requirementsQuery, bool) {
	out := requirementsQuery{limit: requirementsPage}
	for name, given := range values {
		valid := len(given) == 1 && given[0] != ""
		switch {
		case !valid:
		case name == "target":
			out.targets, valid = parseRequirementTargets(given[0])
			out.byName = true
		case name == "after":
			n, ok := canonicalNumber(given[0])
			out.after, valid = int(n), ok && n > 0 && n <= requirementNumberMax
		case name == "limit":
			n, ok := canonicalNumber(given[0])
			out.limit, valid = int(n), ok && n >= 1 && n <= requirementsPage
		case name == "q":
			out.search.term, valid = parseSearch(given[0])
		case name == "state":
			out.state, valid = given[0], false
			for _, known := range requirementStates {
				valid = valid || known == given[0]
			}
		case name == "retired":
			out.retired, valid = given[0], given[0] == requirementsRetired || given[0] == requirementsCurrent
		default:
			valid = false
		}
		if !valid {
			return requirementsQuery{}, false
		}
	}
	return out, !out.byName || len(values) == 1
}

func (q requirementsQuery) filtered() bool {
	return q.search.term != "" || q.state != "" || q.retired != ""
}

func (q requirementsQuery) matches(v releasetrace.RequirementView, k releasetrace.Key) bool {
	r := v.Requirement
	switch {
	case q.state != "" && v.State != q.state:
		return false
	case q.retired != "" && v.Retired != (q.retired == requirementsRetired):
		return false
	}
	return q.search.matches(append([]string{label(k), r.Namespace, r.ID, r.Source, r.Owner, r.Design}, r.Interfaces...)...)
}

func (f *evidenceFacts) requirementsAnswer(q requirementsQuery) (requirementsAnswer, error) {
	stored := f.s.ev.requirements
	out := requirementsAnswer{Order: requirementsOrder, Q: q.search.term, State: q.state, Retired: q.retired, Requirements: []evidenceRequirementRow{}, Total: len(stored)}
	add := func(index int) error {
		row, err := f.requirementRow(index)
		out.Requirements = append(out.Requirements, row)
		return err
	}
	if q.byName {
		for _, number := range q.targets {
			if number > len(stored) {
				continue
			}
			if err := add(number - 1); err != nil {
				return requirementsAnswer{}, err
			}
		}
		out.Matching = len(out.Requirements)
		return out, nil
	}
	last := 0
	for index := range stored {
		if q.filtered() {
			v, err := f.s.traceRequirement(stored[index], f.build)
			if err != nil {
				return requirementsAnswer{}, err
			}
			if !q.matches(v, stored[index]) {
				continue
			}
		}
		out.Matching++
		switch {
		case index < q.after:
		case len(out.Requirements) == q.limit:
			out.Next = strconv.Itoa(last)
			if !q.filtered() {
				out.Matching = len(stored)
				return out, nil
			}
		default:
			if err := add(index); err != nil {
				return requirementsAnswer{}, err
			}
			last = index + 1
		}
	}
	return out, nil
}

func (s *Server) listEvidenceRequirements(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	now := s.clock()
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, installationEmpty{Revision: s.Revision(), Now: stamp(now), State: stateNotConfigured})
		return
	}
	q, valid := parseRequirementsQuery(r.URL.Query())
	if !valid || q.state != "" && s.ev.candidate == nil {
		failCode(w, commands.InvalidRequest, request)
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
	page, err := f.requirementsAnswer(q)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page.Revision, page.ServerTime = s.Revision(), stamp(now)
	writeJSON(w, http.StatusOK, request, page)
}
