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
	found, answer, ok := s.admitted(w, r)
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
	f, err := s.evidenceFactsFor(found.verifier, answer.User(), now)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
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
