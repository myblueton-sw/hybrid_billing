package server

import (
	"net/http"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type retentionHoldsPage struct {
	Revision   string             `json:"revision"`
	ServerTime string             `json:"server_time"`
	Order      string             `json:"order"`
	Q          string             `json:"q"`
	Holds      []retentionHoldRow `json:"holds"`
	Next       string             `json:"next"`
	Matching   int                `json:"matching"`
	Total      int                `json:"total"`
}

func holdLive(h *retentionHold) bool { return !h.released }

func holdNumber(h *retentionHold) uint64 { return uint64(h.number) }

func (f *retentionFacts) workspaceHolds() []*retentionHold {
	out := []*retentionHold{}
	for _, h := range f.s.ret.holds {
		if h.workspace == f.workspace {
			out = append(out, h)
		}
	}
	return out
}

func (s *Server) listRetentionHolds(w http.ResponseWriter, r *http.Request) {
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
	page := retentionHoldsPage{Revision: s.Revision(), ServerTime: stamp(now), Order: closedPlansOrder, Q: q.term, Holds: []retentionHoldRow{}}
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	if s.faulty("retention-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	f, err := s.retentionFactsFor(found.verifier, answer.User(), answer.Workspace(), now)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	mine := f.workspaceHolds()
	walk := keptWalk{q: q}
	for index := len(mine) - 1; index >= 0; index-- {
		h := mine[index]
		if holdLive(h) {
			continue
		}
		page.Total++
		if !walk.take(holdNumber(h), q.term == "" || q.matches(h.id, retentionHoldTarget+h.id, h.rationale)) {
			continue
		}
		row, err := f.holdRow(h)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		page.Holds = append(page.Holds, row)
	}
	page.Next, page.Matching = walk.Next, walk.Matching
	writeJSON(w, http.StatusOK, request, page)
}
