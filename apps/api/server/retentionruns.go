package server

import (
	"net/http"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

type workspaceRun struct {
	id     string
	number int
	view   retention.RunView
}

type retentionRunsPage struct {
	Revision   string            `json:"revision"`
	ServerTime string            `json:"server_time"`
	Order      string            `json:"order"`
	Q          string            `json:"q"`
	Runs       []retentionRunRow `json:"runs"`
	Next       string            `json:"next"`
	Matching   int               `json:"matching"`
	Total      int               `json:"total"`
}

func runRunning(r workspaceRun) bool { return r.view.State == retention.RunRunning }

func runNumber(r workspaceRun) uint64 { return uint64(r.number) }

func (f *retentionFacts) workspaceRuns() ([]workspaceRun, error) {
	out := []workspaceRun{}
	for _, r := range f.s.ret.runs {
		if r.workspace != f.workspace {
			continue
		}
		v, err := f.s.retentionRunView(r.id)
		if err != nil {
			return nil, err
		}
		out = append(out, workspaceRun{id: r.id, number: len(out) + 1, view: v})
	}
	return out, nil
}

func (s *Server) listRetentionRuns(w http.ResponseWriter, r *http.Request) {
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
	page := retentionRunsPage{Revision: s.Revision(), ServerTime: stamp(now), Order: closedPlansOrder, Q: q.term, Runs: []retentionRunRow{}}
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	if s.faulty("retention-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	f := s.retentionFactsWithoutRoster(answer.User(), answer.Workspace())
	mine, err := f.workspaceRuns()
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	walk := keptWalk{q: q}
	for index := len(mine) - 1; index >= 0; index-- {
		run := mine[index]
		if runRunning(run) {
			continue
		}
		page.Total++
		if !walk.take(runNumber(run), q.term == "" || q.matches(run.id, runTarget(run.id), run.view.Plan.Class, run.view.Plan.Reason)) {
			continue
		}
		row, err := f.runRow(run)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		page.Runs = append(page.Runs, row)
	}
	page.Next, page.Matching = walk.Next, walk.Matching
	writeJSON(w, http.StatusOK, request, page)
}
