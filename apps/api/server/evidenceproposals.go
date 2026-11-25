package server

import (
	"net/http"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type evidenceProposalsPage struct {
	Revision   string                `json:"revision"`
	ServerTime string                `json:"server_time"`
	Order      string                `json:"order"`
	Q          string                `json:"q"`
	Proposals  []evidenceProposalRow `json:"proposals"`
	Next       string                `json:"next"`
	Matching   int                   `json:"matching"`
	History    keptFacts             `json:"proposal_history"`
}

func evidencePending(e *evidenceEntry) bool { return e.state == entryPending }

func evidenceSequence(e *evidenceEntry) uint64 { return e.seq }

func (s *Server) listEvidenceProposals(w http.ResponseWriter, r *http.Request) {
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
	page := evidenceProposalsPage{Revision: s.Revision(), ServerTime: stamp(now), Order: closedPlansOrder, Q: q.term, Proposals: []evidenceProposalRow{}}
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
	page.History = keptFacts{Kept: installKept, Dropped: s.ev.dropped}
	walk := keptWalk{q: q}
	for index := len(s.ev.entries) - 1; index >= 0; index-- {
		e := s.ev.entries[index]
		if evidencePending(e) {
			continue
		}
		page.History.Total++
		number := strconv.FormatUint(e.number, 10)
		if walk.take(e.seq, q.matches(number, evidenceProposalTarget(e.module, e.number), e.module, e.kind, e.state, e.proposer, s.loginOf(e.proposer), e.rationale)) {
			row, err := f.proposalRow(e)
			if err != nil {
				failCode(w, commands.DependencyUnavailable, request)
				return
			}
			page.Proposals = append(page.Proposals, row)
		}
	}
	page.Next, page.Matching = walk.Next, walk.Matching
	writeJSON(w, http.StatusOK, request, page)
}
