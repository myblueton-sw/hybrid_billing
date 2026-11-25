package server

import (
	"net/http"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

const resultsOrder = "latest_record_first"

type evidenceResultsPage struct {
	Revision    string              `json:"revision"`
	ServerTime  string              `json:"server_time"`
	Order       string              `json:"order"`
	RowsCanMove bool                `json:"rows_can_move"`
	Q           string              `json:"q"`
	Results     []evidenceResultRow `json:"results"`
	Next        string              `json:"next"`
	Matching    int                 `json:"matching"`
	Total       int                 `json:"total"`
}

func (f *evidenceFacts) resultRow(k evidenceResultKey) (evidenceResultRow, error) {
	s := f.s
	v, err := s.traceResults(k)
	if err != nil {
		return evidenceResultRow{}, err
	}
	outcome := outcomeFail
	if v.Result.Passed {
		outcome = outcomePass
	}
	row := evidenceResultRow{Number: s.ev.resultSeq[k], Namespace: k.namespace, Test: k.test, Build: k.build, Counted: v.Counted, LatestOutcome: outcome, Evidence: v.Result.Evidence,
		Harness: v.Harness, At: stamp(v.At), Passes: v.Passes, Failures: v.Failures}
	if !v.Counted {
		reason, err := s.resultNotCounted(k)
		if err != nil {
			return evidenceResultRow{}, err
		}
		row.NotCountedReason = optional(reason)
	}
	return row, nil
}

func (s *Server) listEvidenceResults(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	q, valid := parseKeptQuery(r.URL.Query())
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	now := s.clock()
	page := evidenceResultsPage{Revision: s.Revision(), ServerTime: stamp(now), Order: resultsOrder, RowsCanMove: true, Q: q.term, Results: []evidenceResultRow{}}
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	if s.faulty("release-evidence-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	f := s.evidenceFactsWithoutRoster(answer.User())
	page.Total = len(s.ev.results)
	walk := keptWalk{q: q}
	for n := len(s.ev.results) - 1; n >= 0; n-- {
		k := s.ev.results[n]
		if !walk.take(s.ev.resultSeq[k], q.term == "" || q.matches(k.namespace, k.test, k.build)) {
			continue
		}
		row, err := f.resultRow(k)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		page.Results = append(page.Results, row)
	}
	page.Next, page.Matching = walk.Next, walk.Matching
	writeJSON(w, http.StatusOK, request, page)
}

type evidenceBuildsPage struct {
	Revision   string             `json:"revision"`
	ServerTime string             `json:"server_time"`
	Order      string             `json:"order"`
	Q          string             `json:"q"`
	Builds     []evidenceBuildRow `json:"builds"`
	Next       string             `json:"next"`
	Matching   int                `json:"matching"`
	Total      int                `json:"total"`
}

func buildRow(builds []evidenceBuildRecord, n int) evidenceBuildRow {
	return evidenceBuildRow{Build: builds[n].id, Commit: builds[n].commit, Order: n + 1}
}

func (s *Server) listEvidenceBuilds(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	q, valid := parseKeptQuery(r.URL.Query())
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	page := evidenceBuildsPage{Revision: s.Revision(), ServerTime: stamp(s.clock()), Order: closedPlansOrder, Q: q.term, Builds: []evidenceBuildRow{}}
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	if s.faulty("release-evidence-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	builds := s.ev.builds
	page.Total = len(builds)
	walk := keptWalk{q: q}
	for n := len(builds) - 1; n >= 0; n-- {
		if walk.take(uint64(n+1), q.matches(builds[n].id, builds[n].commit)) {
			page.Builds = append(page.Builds, buildRow(builds, n))
		}
	}
	page.Next, page.Matching = walk.Next, walk.Matching
	writeJSON(w, http.StatusOK, request, page)
}
